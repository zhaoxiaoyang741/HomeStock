package handler

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"rsc.io/qr"

	"github.com/gin-gonic/gin"

	httpresp "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/response"
	wx "github.com/zhaoxiaoyang741/HomeStock/internal/integration/channel/wechat"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/channel"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

// WechatHandler serves status, config, and QR login endpoints for the WeChat (iLink) channel.
type WechatHandler struct {
	chMgr      *channel.Manager
	wxCh       *wx.WechatChannel
	configPath string // path to config file, for saving token bindings

	// QR flow management
	weixinMu    sync.Mutex
	weixinFlows map[string]*weixinFlow

	// updateChan is called when the user updates wechat config from the UI.
	updateChan func(config.WechatChannelConfig) error
}

// Name returns the handler name. Implements channel.WebhookRegistrar.
func (h *WechatHandler) Name() string { return "wechat" }

// SetChannelUpdateFn registers the callback for reconfiguring the channel
// after a config update.
func (h *WechatHandler) SetChannelUpdateFn(fn func(config.WechatChannelConfig) error) {
	h.updateChan = fn
}

// SetChannel updates the WeChat channel instance (used at runtime when enabling a previously disabled channel).
func (h *WechatHandler) SetChannel(wxCh *wx.WechatChannel) {
	h.wxCh = wxCh
}

// weixinFlow represents a QR login flow.
type weixinFlow struct {
	ID        string
	Qrcode    string // qrcode token from iLink API (used for status polling)
	QRDataURI string // base64 PNG data URI for display
	AccountID string
	Status    string // wait / scaned / confirmed / expired / error
	Error     string
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt time.Time
}

type weixinFlowResponse struct {
	FlowID    string `json:"flow_id"`
	Status    string `json:"status"`
	QRDataURI string `json:"qr_data_uri,omitempty"`
	AccountID string `json:"account_id,omitempty"`
	Error     string `json:"error,omitempty"`
}

const (
	weixinFlowTTL       = 5 * time.Minute
	weixinFlowGCAge     = 30 * time.Minute
	weixinBaseURL       = "https://ilinkai.weixin.qq.com/"
	weixinBotType       = "3"
	weixinStatusWait    = "wait"
	weixinStatusScanned = "scaned"
	weixinStatusConfirm = "confirmed"
	weixinStatusExpired = "expired"
	weixinStatusError   = "error"
)

// NewWechatHandler creates a WechatHandler.
func NewWechatHandler(chMgr *channel.Manager, wxCh *wx.WechatChannel, configPath string) *WechatHandler {
	return &WechatHandler{
		chMgr:       chMgr,
		wxCh:        wxCh,
		configPath:  configPath,
		weixinFlows: make(map[string]*weixinFlow),
	}
}

// RegisterRoutes mounts WeChat endpoints under the API group.
func (h *WechatHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/wechat/status", h.Status)
	api.POST("/wechat/qrcode", h.StartQRFlow)
	api.GET("/wechat/qrcode/:id", h.PollQRFlow)
	api.POST("/wechat/disconnect", h.Disconnect)
	api.POST("/wechat/reconnect", h.Reconnect)
	api.PATCH("/wechat/config", h.UpdateConfig)
}

// Status returns the WeChat channel connection status.
func (h *WechatHandler) Status(c *gin.Context) {
	isConnected := false
	accountID := ""
	hasToken := false
	isTokenExpired := false
	if h.wxCh != nil {
		isConnected = h.wxCh.IsRunning()
		_, accountID = h.wxCh.GetSelfInfo()
		hasToken = h.wxCh.HasToken()
		isTokenExpired = h.wxCh.IsTokenExpired()
	}

	_, enabled := h.chMgr.GetChannel("wechat")

	httpresp.OK(c, gin.H{
		"connected":     isConnected,
		"enabled":       enabled,
		"has_token":     hasToken,
		"token_expired": isTokenExpired,
		"account_id":    accountID,
	})
}

// StartQRFlow starts a new WeChat QR login flow.
//
//	POST /api/v1/wechat/qrcode
func (h *WechatHandler) StartQRFlow(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 15*time.Second)
	defer cancel()

	api, err := wx.NewApiClient(weixinBaseURL, "", "")
	if err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "failed to create wechat client: "+err.Error())
		return
	}

	qrResp, err := api.GetQRCode(ctx, weixinBotType)
	if err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "failed to get QR code: "+err.Error())
		return
	}

	dataURI, err := generateQRDataURI(qrResp.QrcodeImgContent)
	if err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "failed to generate QR image: "+err.Error())
		return
	}

	now := time.Now()
	flow := &weixinFlow{
		ID:        newWeixinFlowID(),
		Qrcode:    qrResp.Qrcode,
		QRDataURI: dataURI,
		Status:    weixinStatusWait,
		CreatedAt: now,
		UpdatedAt: now,
		ExpiresAt: now.Add(weixinFlowTTL),
	}
	h.storeFlow(flow)

	logger.InfoCF("wechat", "QR flow started", map[string]any{"flow_id": flow.ID})

	httpresp.OK(c, weixinFlowResponse{
		FlowID:    flow.ID,
		Status:    flow.Status,
		QRDataURI: flow.QRDataURI,
	})
}

// PollQRFlow polls the WeChat API for QR code status and updates the flow.
//
//	GET /api/v1/wechat/qrcode/:id
func (h *WechatHandler) PollQRFlow(c *gin.Context) {
	flowID := strings.TrimSpace(c.Param("id"))
	if flowID == "" {
		httpresp.Error(c, http.StatusBadRequest, "missing flow id")
		return
	}

	flow, ok := h.getFlow(flowID)
	if !ok {
		httpresp.Error(c, http.StatusNotFound, "flow not found")
		return
	}

	// Return terminal states directly without polling WeChat again
	if flow.Status == weixinStatusConfirm ||
		flow.Status == weixinStatusExpired ||
		flow.Status == weixinStatusError {
		httpresp.OK(c, weixinFlowResponse{
			FlowID: flow.ID,
			Status: flow.Status,
			Error:  flow.Error,
		})
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()

	api, err := wx.NewApiClient(weixinBaseURL, "", "")
	if err != nil {
		h.setFlowError(flowID, "client error: "+err.Error())
		flow, _ = h.getFlow(flowID)
		httpresp.OK(c, weixinFlowResponse{FlowID: flow.ID, Status: flow.Status, Error: flow.Error})
		return
	}

	statusResp, err := api.GetQRCodeStatus(ctx, flow.Qrcode)
	if err != nil {
		// Transient error - keep current status
		httpresp.OK(c, weixinFlowResponse{
			FlowID:    flow.ID,
			Status:    flow.Status,
			QRDataURI: flow.QRDataURI,
		})
		return
	}

	switch statusResp.Status {
	case weixinStatusWait:
		// no change

	case weixinStatusScanned:
		h.updateFlowStatus(flowID, weixinStatusScanned)

	case weixinStatusConfirm:
		if statusResp.BotToken == "" {
			h.setFlowError(flowID, "login confirmed but missing bot_token")
			break
		}
		if saveErr := h.saveWechatBinding(statusResp.BotToken, statusResp.IlinkBotID); saveErr != nil {
			h.setFlowError(flowID, "failed to save token: "+saveErr.Error())
			logger.ErrorCF("wechat", "failed to save token", map[string]any{"error": saveErr.Error()})
			break
		}
		h.setFlowConfirmed(flowID, statusResp.IlinkBotID)
		logger.InfoCF("wechat", "QR login confirmed, token saved", map[string]any{
			"flow_id":    flowID,
			"account_id": statusResp.IlinkBotID,
		})

	case weixinStatusExpired:
		h.updateFlowStatus(flowID, weixinStatusExpired)

	default:
		// unknown status, keep as-is
	}

	flow, _ = h.getFlow(flowID)
	resp := weixinFlowResponse{
		FlowID:    flow.ID,
		Status:    flow.Status,
		AccountID: flow.AccountID,
		Error:     flow.Error,
	}
	if flow.Status == weixinStatusWait || flow.Status == weixinStatusScanned {
		resp.QRDataURI = flow.QRDataURI
	}
	httpresp.OK(c, resp)
}

// saveWechatBinding writes the token to config and restarts the channel with the new token.
func (h *WechatHandler) saveWechatBinding(token, accountID string) error {
	baseURL := weixinBaseURL
	cdnBaseURL := "https://novac2c.cdn.weixin.qq.com/c2c"

	var savedCfg config.WechatChannelConfig
	if err := config.Save(h.configPath, func(cfg *config.Config) {
		wechatCfg, _ := cfg.WechatConfig()
		wechatCfg.Token = token
		if accountID != "" {
			wechatCfg.AccountID = accountID
		}
		wechatCfg.Enabled = true
		wechatCfg.BaseURL = baseURL
		wechatCfg.CDNBaseURL = cdnBaseURL
		savedCfg = wechatCfg
		_ = cfg.SetChannelConfig("wechat", wechatCfg)
	}); err != nil {
		return fmt.Errorf("save config: %w", err)
	}

	// Restart or create the channel with the new token.
	if h.wxCh != nil {
		bgCtx := context.Background()

		if h.wxCh.IsRunning() {
			_ = h.wxCh.Stop(bgCtx)
		}

		h.wxCh.SetConfig(savedCfg)

		if err := h.wxCh.Start(bgCtx); err != nil {
			return fmt.Errorf("restart channel after binding: %w", err)
		}

		// Ensure the channel is registered with the manager
		if _, exists := h.chMgr.GetChannel("wechat"); !exists {
			h.chMgr.AddChannel(h.wxCh)
		}

		logger.InfoCF("wechat", "channel restarted with new token", map[string]any{
			"account_id": accountID,
		})
	} else if h.updateChan != nil {
		// Channel was disabled at startup (h.wxCh nil).
		// Delegate to updateChan which creates a new channel, sets the inbound handler, etc.
		if err := h.updateChan(savedCfg); err != nil {
			return fmt.Errorf("create channel after binding: %w", err)
		}
	}

	return nil
}

// Disconnect stops the WeChat channel.
func (h *WechatHandler) Disconnect(c *gin.Context) {
	if h.wxCh != nil && h.wxCh.IsRunning() {
		if err := h.wxCh.Stop(c.Request.Context()); err != nil {
			httpresp.Error(c, http.StatusInternalServerError, "stop channel failed: "+err.Error())
			return
		}
	}
	httpresp.OK(c, gin.H{"message": "disconnected"})
}

// Reconnect stops and restarts the WeChat channel.
func (h *WechatHandler) Reconnect(c *gin.Context) {
	if h.wxCh == nil {
		httpresp.Error(c, http.StatusBadRequest, "wechat channel not initialized")
		return
	}

	if h.wxCh.IsTokenExpired() {
		httpresp.Error(c, http.StatusBadRequest, "token expired, please re-bind via QR code")
		return
	}

	// Ensure channel is in the manager
	_, exists := h.chMgr.GetChannel("wechat")
	if !exists {
		h.chMgr.AddChannel(h.wxCh)
	}

	bgCtx := context.Background()

	if h.wxCh.IsRunning() {
		if err := h.wxCh.Stop(bgCtx); err != nil {
			httpresp.Error(c, http.StatusInternalServerError, "stop channel failed: "+err.Error())
			return
		}
	}

	if err := h.wxCh.Start(bgCtx); err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "start channel failed: "+err.Error())
		return
	}

	httpresp.OK(c, gin.H{"message": "reconnected"})
}

// UpdateConfig handles runtime updates to the WeChat channel configuration.
// Request: { enabled?: bool }
func (h *WechatHandler) UpdateConfig(c *gin.Context) {
	var req struct {
		Enabled *bool `json:"enabled,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.Error(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	if req.Enabled == nil {
		httpresp.OK(c, gin.H{"message": "no changes"})
		return
	}

	var savedCfg config.WechatChannelConfig
	if err := config.Save(h.configPath, func(cfg *config.Config) {
		wc, _ := cfg.WechatConfig()
		wc.Enabled = *req.Enabled
		savedCfg = wc
		_ = cfg.SetChannelConfig("wechat", wc)
	}); err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "save config failed: "+err.Error())
		return
	}

	if h.updateChan != nil {
		if err := h.updateChan(savedCfg); err != nil {
			httpresp.Error(c, http.StatusInternalServerError, "reconfigure channel failed: "+err.Error())
			return
		}
	}

	httpresp.OK(c, gin.H{"message": "config updated"})
}

// ---------------------------------------------------------------------------
// QR flow management helpers
// ---------------------------------------------------------------------------

func (h *WechatHandler) storeFlow(flow *weixinFlow) {
	h.weixinMu.Lock()
	defer h.weixinMu.Unlock()
	h.gcFlows(time.Now())
	h.weixinFlows[flow.ID] = flow
}

func (h *WechatHandler) getFlow(flowID string) (*weixinFlow, bool) {
	h.weixinMu.Lock()
	defer h.weixinMu.Unlock()
	h.gcFlows(time.Now())
	flow, ok := h.weixinFlows[flowID]
	if !ok {
		return nil, false
	}
	cp := *flow
	return &cp, true
}

func (h *WechatHandler) updateFlowStatus(flowID, status string) {
	h.weixinMu.Lock()
	defer h.weixinMu.Unlock()
	if flow, ok := h.weixinFlows[flowID]; ok {
		flow.Status = status
		flow.UpdatedAt = time.Now()
	}
}

func (h *WechatHandler) setFlowConfirmed(flowID, accountID string) {
	h.weixinMu.Lock()
	defer h.weixinMu.Unlock()
	if flow, ok := h.weixinFlows[flowID]; ok {
		flow.Status = weixinStatusConfirm
		flow.AccountID = accountID
		flow.UpdatedAt = time.Now()
	}
}

func (h *WechatHandler) setFlowError(flowID, errMsg string) {
	h.weixinMu.Lock()
	defer h.weixinMu.Unlock()
	if flow, ok := h.weixinFlows[flowID]; ok {
		flow.Status = weixinStatusError
		flow.Error = errMsg
		flow.UpdatedAt = time.Now()
	}
}

func (h *WechatHandler) gcFlows(now time.Time) {
	for id, flow := range h.weixinFlows {
		if flow.Status == weixinStatusWait || flow.Status == weixinStatusScanned {
			if !flow.ExpiresAt.IsZero() && now.After(flow.ExpiresAt) {
				flow.Status = weixinStatusExpired
				flow.UpdatedAt = now
			}
		}
		if flow.Status != weixinStatusWait &&
			flow.Status != weixinStatusScanned &&
			now.Sub(flow.UpdatedAt) > weixinFlowGCAge {
			delete(h.weixinFlows, id)
		}
	}
}

// generateQRDataURI encodes content as a QR code PNG and returns a data URI.
func generateQRDataURI(content string) (string, error) {
	code, err := qr.Encode(content, qr.L)
	if err != nil {
		return "", fmt.Errorf("qr encode: %w", err)
	}
	pngBytes := code.PNG()
	encoded := base64.StdEncoding.EncodeToString(pngBytes)
	return "data:image/png;base64," + encoded, nil
}

func newWeixinFlowID() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("wx_%d", time.Now().UnixNano())
	}
	return "wx_" + hex.EncodeToString(buf)
}

// ---------------------------------------------------------------------------
// Handler factory registration
// ---------------------------------------------------------------------------

func init() {
	channel.RegisterHandlerFactory("wechat", func(deps *channel.HandlerDeps) (channel.WebhookRegistrar, channel.ConfigChangeHandler, error) {
		_, ok := deps.Config.WechatConfig()
		if !ok {
			return nil, nil, nil
		}

		var wxCh *wx.WechatChannel
		if rawCh, ok := deps.ChannelMgr.GetChannel("wechat"); ok {
			wxCh, _ = rawCh.(*wx.WechatChannel)
		}

		handler := NewWechatHandler(deps.ChannelMgr, wxCh, deps.ConfigPath)

		handler.SetChannelUpdateFn(func(savedCfg config.WechatChannelConfig) error {
			ctx := context.Background()

			if savedCfg.Enabled {
				wc := wxCh
				if wc == nil {
					if rawCh, ok := deps.ChannelMgr.GetChannel("wechat"); ok {
						wc, _ = rawCh.(*wx.WechatChannel)
					}
				}
				if wc == nil {
					wc = wx.NewWechatChannel(savedCfg)
					handler.SetChannel(wc)
					deps.ChannelMgr.AddChannel(wc)
				}

				if !wc.IsRunning() {
					wc.SetConfig(savedCfg)
					if err := wc.Start(ctx); err != nil {
						return err
					}
				}
				if _, exists := deps.ChannelMgr.GetChannel("wechat"); !exists {
					deps.ChannelMgr.AddChannel(wc)
				}
			} else {
				if rawCh, ok := deps.ChannelMgr.GetChannel("wechat"); ok {
					if wc, ok := rawCh.(*wx.WechatChannel); ok && wc.IsRunning() {
						if err := wc.Stop(ctx); err != nil {
							return err
						}
					}
				}
				deps.ChannelMgr.RemoveChannel("wechat")
			}

			return nil
		})

		cch := &wechatConfigChangeHandler{
			updateFn: handler.updateChan,
		}
		return handler, cch, nil
	})
}

// wechatConfigChangeHandler adapts the wechat update callback to the
// ConfigChangeHandler interface for hot-reload support.
type wechatConfigChangeHandler struct {
	updateFn func(config.WechatChannelConfig) error
}

func (h *wechatConfigChangeHandler) Name() string { return "wechat" }

func (h *wechatConfigChangeHandler) HandleConfigChange(ctx context.Context, oldCfg, newCfg *config.Config) error {
	wechatCfg, ok := newCfg.WechatConfig()
	if !ok {
		return nil
	}
	return h.updateFn(wechatCfg)
}

// Compile-time interface checks.
var _ interface{ RegisterRoutes(*gin.RouterGroup) } = (*FeishuHandler)(nil)
var _ interface{ RegisterRoutes(*gin.RouterGroup) } = (*WechatHandler)(nil)
