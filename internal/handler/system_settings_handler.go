package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	httpresp "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/response"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
)

type SystemSettingsHandler struct {
	svc *service.SystemSettingsService
}

func NewSystemSettingsHandler(svc *service.SystemSettingsService) *SystemSettingsHandler {
	return &SystemSettingsHandler{svc: svc}
}

func (h *SystemSettingsHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/settings/system", h.Get)
	api.PUT("/settings/system", h.Update)
}

type updateSystemSettingsRequest struct {
	Version  *int                                 `json:"version"`
	Reminder *updateSystemSettingsReminderRequest `json:"reminder"`
	Notify   *updateSystemSettingsNotifyRequest   `json:"notify"`
}

type updateSystemSettingsReminderRequest struct {
	RemindDays int    `json:"remind_days"`
	CheckTime  string `json:"check_time"`
}

type updateSystemSettingsNotifyRequest struct {
	Enabled           bool   `json:"enabled"`
	FeishuWebhookMode string `json:"feishu_webhook_mode"`
	FeishuWebhook     string `json:"feishu_webhook"`
}

func (h *SystemSettingsHandler) Get(c *gin.Context) {
	view, err := h.svc.Get(c.Request.Context())
	if err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "get system settings failed")
		return
	}
	httpresp.OK(c, view)
}

func (h *SystemSettingsHandler) Update(c *gin.Context) {
	var req updateSystemSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Version == nil || req.Reminder == nil || req.Notify == nil {
		httpresp.Error(c, http.StatusBadRequest, "version, reminder, and notify are required")
		return
	}

	view, err := h.svc.Update(c.Request.Context(), svcActorFromRequest(c), service.UpdateSystemSettingsInput{
		Version: *req.Version,
		Reminder: service.UpdateSystemSettingsReminderInput{
			RemindDays: req.Reminder.RemindDays,
			CheckTime:  strings.TrimSpace(req.Reminder.CheckTime),
		},
		Notify: service.UpdateSystemSettingsNotifyInput{
			Enabled:           req.Notify.Enabled,
			FeishuWebhookMode: strings.TrimSpace(req.Notify.FeishuWebhookMode),
			FeishuWebhook:     strings.TrimSpace(req.Notify.FeishuWebhook),
		},
	})
	if err != nil {
		handleSystemSettingsError(c, err)
		return
	}
	httpresp.OK(c, view)
}

func handleSystemSettingsError(c *gin.Context, err error) {
	switch {
	case service.IsSystemSettingsValidationError(err):
		httpresp.Error(c, http.StatusBadRequest, err.Error())
	case err == repository.ErrSystemSettingsVersionConflict:
		httpresp.Error(c, http.StatusConflict, err.Error())
	default:
		httpresp.Error(c, http.StatusInternalServerError, "update system settings failed")
	}
}
