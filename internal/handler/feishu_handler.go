package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	httpresp "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/response"
	"github.com/zhaoxiaoyang741/HomeStock/internal/channel"
	"github.com/zhaoxiaoyang741/HomeStock/internal/channel/feishu"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

// FeishuHandler serves OAuth, status and config endpoints for the Feishu channel.
type FeishuHandler struct {
	authSvc    *feishu.OAuthService
	chMgr      *channel.Manager
	feishuCh   *feishu.FeishuChannel
	frontend   string
	configPath string

	// updateChan is called when the user updates feishu config from the UI.
	// Set via SetChannelUpdateFn by the app layer.
	updateChan func(config.FeishuChannelConfig) error
}

// NewFeishuHandler creates a FeishuHandler.
func NewFeishuHandler(authSvc *feishu.OAuthService, chMgr *channel.Manager, feishuCh *feishu.FeishuChannel, frontendURL string, configPath string) *FeishuHandler {
	return &FeishuHandler{
		authSvc:    authSvc,
		chMgr:      chMgr,
		feishuCh:   feishuCh,
		frontend:   frontendURL,
		configPath: configPath,
	}
}

// SetChannelUpdateFn registers the callback for hot-reloading the channel
// after a config update.
func (h *FeishuHandler) SetChannelUpdateFn(fn func(config.FeishuChannelConfig) error) {
	h.updateChan = fn
}

// RegisterRoutes mounts Feishu OAuth and config endpoints under the API group.
func (h *FeishuHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/feishu/auth-url", h.AuthURL)
	api.GET("/feishu/callback", h.Callback)
	api.GET("/feishu/status", h.Status)
	api.POST("/feishu/disconnect", h.Disconnect)
	api.PATCH("/feishu/config", h.UpdateConfig)
}

// AuthURL generates a state nonce and returns the Feishu OAuth authorization URL.
func (h *FeishuHandler) AuthURL(c *gin.Context) {
	authURL, err := h.authSvc.GetAuthURL(c.Request.Context())
	if err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "generate auth url failed: "+err.Error())
		return
	}
	httpresp.OK(c, gin.H{"auth_url": authURL})
}

// Callback handles the OAuth redirect from Feishu.
func (h *FeishuHandler) Callback(c *gin.Context) {
	code := c.Query("code")
	state := c.Query("state")

	if code == "" {
		http.Redirect(c.Writer, c.Request, h.frontend+"/settings?section=channel&error=missing_code", http.StatusFound)
		return
	}

	if err := h.authSvc.HandleCallback(c.Request.Context(), code, state); err != nil {
		http.Redirect(c.Writer, c.Request, h.frontend+"/settings?section=channel&error="+err.Error(), http.StatusFound)
		return
	}

	// Start or restart the Feishu channel if we have a reference
	if h.feishuCh != nil {
		_, exists := h.chMgr.GetChannel("feishu")
		if !exists {
			h.chMgr.AddChannel(h.feishuCh)
		}
		if !h.feishuCh.IsRunning() {
			_ = h.feishuCh.Start(context.Background())
		}
	}

	http.Redirect(c.Writer, c.Request, h.frontend+"/settings?section=channel&authorized=1", http.StatusFound)
}

// Status returns the Feishu channel configuration and connection status.
func (h *FeishuHandler) Status(c *gin.Context) {
	isConnected := false
	if h.feishuCh != nil {
		isConnected = h.feishuCh.IsRunning()
	}

	isEnabled := false
	if h.feishuCh != nil {
		_, exists := h.chMgr.GetChannel("feishu")
		isEnabled = exists
	}

	status, err := h.authSvc.GetStatus(c.Request.Context(), isConnected, isEnabled)
	if err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "get status failed")
		return
	}
	httpresp.OK(c, status)
}

// Disconnect stops the Feishu channel and clears the stored OAuth token.
func (h *FeishuHandler) Disconnect(c *gin.Context) {
	if h.feishuCh != nil && h.feishuCh.IsRunning() {
		if err := h.feishuCh.Stop(c.Request.Context()); err != nil {
			httpresp.Error(c, http.StatusInternalServerError, "stop channel failed")
			return
		}
	}

	if err := h.authSvc.ClearAuth(c.Request.Context()); err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "clear auth failed")
		return
	}

	httpresp.OK(c, gin.H{"message": "disconnected"})
}

// UpdateConfig handles runtime updates to the Feishu channel configuration.
// Request: { enabled?: bool, app_id?: string, app_secret?: string }
// Empty/omitted fields preserve existing values.
func (h *FeishuHandler) UpdateConfig(c *gin.Context) {
	var req struct {
		Enabled   *bool   `json:"enabled,omitempty"`
		AppID     *string `json:"app_id,omitempty"`
		AppSecret *string `json:"app_secret,omitempty"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.Error(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	var savedCfg config.FeishuChannelConfig
	if err := config.Save(h.configPath, func(cfg *config.Config) {
		fc := &cfg.Channels.Feishu
		if req.Enabled != nil {
			fc.Enabled = *req.Enabled
		}
		if req.AppID != nil && *req.AppID != "" {
			fc.AppID = *req.AppID
		}
		if req.AppSecret != nil && *req.AppSecret != "" {
			fc.AppSecret = *req.AppSecret
		}
		savedCfg = *fc
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
