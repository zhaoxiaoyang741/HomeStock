package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	httpresp "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/response"
	"github.com/zhaoxiaoyang741/HomeStock/internal/channel"
	"github.com/zhaoxiaoyang741/HomeStock/internal/channel/feishu"
)

// FeishuHandler serves OAuth and status endpoints for the Feishu channel.
type FeishuHandler struct {
	authSvc  *feishu.OAuthService
	chMgr    *channel.Manager
	feishuCh *feishu.FeishuChannel
	frontend string
}

// NewFeishuHandler creates a FeishuHandler.
func NewFeishuHandler(authSvc *feishu.OAuthService, chMgr *channel.Manager, feishuCh *feishu.FeishuChannel, frontendURL string) *FeishuHandler {
	return &FeishuHandler{
		authSvc:  authSvc,
		chMgr:    chMgr,
		feishuCh: feishuCh,
		frontend: frontendURL,
	}
}

// RegisterRoutes mounts Feishu OAuth endpoints under the API group.
func (h *FeishuHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/feishu/auth-url", h.AuthURL)
	api.GET("/feishu/callback", h.Callback)
	api.GET("/feishu/status", h.Status)
	api.POST("/feishu/disconnect", h.Disconnect)
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

	status, err := h.authSvc.GetStatus(c.Request.Context(), isConnected)
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
