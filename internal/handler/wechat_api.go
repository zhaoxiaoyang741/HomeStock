package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	httpresp "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/response"
	wx "github.com/zhaoxiaoyang741/HomeStock/internal/integration/channel/wechat"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/channel"
)

// WechatHandler serves QR code login, status, and config endpoints for the WeChat channel.
type WechatHandler struct {
	chMgr    *channel.Manager
	wxCh     *wx.WechatChannel
}

// NewWechatHandler creates a WechatHandler.
func NewWechatHandler(chMgr *channel.Manager, wxCh *wx.WechatChannel) *WechatHandler {
	return &WechatHandler{
		chMgr: chMgr,
		wxCh:  wxCh,
	}
}

// RegisterRoutes mounts WeChat endpoints under the API group.
func (h *WechatHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/wechat/qrcode", h.QrCode)
	api.GET("/wechat/status", h.Status)
	api.POST("/wechat/disconnect", h.Disconnect)
	api.POST("/wechat/reconnect", h.Reconnect)
}

// QrCode returns the current QR code URL and login status for QR scanning.
func (h *WechatHandler) QrCode(c *gin.Context) {
	if h.wxCh == nil {
		httpresp.Error(c, http.StatusInternalServerError, "wechat channel not initialized")
		return
	}

	sess := h.wxCh.GetLoginSession()
	httpresp.OK(c, gin.H{
		"qr_url":  sess.QrURL,
		"uuid":    sess.UUID,
		"status":  sess.Status.String(),
	})
}

// Status returns the WeChat channel connection status.
func (h *WechatHandler) Status(c *gin.Context) {
	isConnected := false
	if h.wxCh != nil {
		isConnected = h.wxCh.IsRunning()
	}

	isLoggedIn := false
	if h.wxCh != nil {
		isLoggedIn = h.wxCh.IsLoggedIn()
	}

	_, enabled := h.chMgr.GetChannel("wechat")

	httpresp.OK(c, gin.H{
		"connected":  isConnected,
		"logged_in":  isLoggedIn,
		"enabled":    enabled,
	})
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

// Reconnect stops and restarts the WeChat channel (generates new QR login).
func (h *WechatHandler) Reconnect(c *gin.Context) {
	if h.wxCh == nil {
		httpresp.Error(c, http.StatusBadRequest, "wechat channel not initialized")
		return
	}

	// Ensure channel is in the manager
	_, exists := h.chMgr.GetChannel("wechat")
	if !exists {
		h.chMgr.AddChannel(h.wxCh)
	}

	// Use background context for channel lifecycle — the channel must outlive
	// the HTTP request, otherwise the connection is torn down when the handler returns.
	bgCtx := context.Background()

	// Stop the channel if running
	if h.wxCh.IsRunning() {
		if err := h.wxCh.Stop(bgCtx); err != nil {
			httpresp.Error(c, http.StatusInternalServerError, "stop channel failed: "+err.Error())
			return
		}
	}

	// Start the channel (triggers QR login flow)
	if err := h.wxCh.Start(bgCtx); err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "start channel failed: "+err.Error())
		return
	}

	httpresp.OK(c, gin.H{"message": "reconnected"})
}

// Ensure WechatHandler implements the registration interface.
var _ interface{ RegisterRoutes(*gin.RouterGroup) } = (*WechatHandler)(nil)
