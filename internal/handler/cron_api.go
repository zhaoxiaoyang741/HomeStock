package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	httpresp "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/response"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

// CronHandler provides API endpoints for reading and updating cron configuration.
type CronHandler struct {
	configPath string
}

// NewCronHandler creates a CronHandler.
func NewCronHandler(configPath string) *CronHandler {
	return &CronHandler{configPath: configPath}
}

// RegisterRoutes registers the cron config endpoints on the given router group.
//   GET  /cron/config  – returns current CronConfig
//   PUT  /cron/config  – updates CronConfig fields
func (h *CronHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/cron/config", h.getConfig)
	rg.PUT("/cron/config", h.updateConfig)
}

type cronConfigResponse struct {
	Enabled                 bool   `json:"enabled"`
	ExpiryCheckIntervalDays int    `json:"expiry_check_interval_days"`
	ExpiryCheckPollInterval string `json:"expiry_check_poll_interval"`
	NotifyEnabled           bool   `json:"notify_enabled"`
	NotifyTimeStart         string `json:"notify_time_start"`
	NotifyTimeEnd           string `json:"notify_time_end"`
}

func (h *CronHandler) getConfig(c *gin.Context) {
	cfg := config.Get().Cron
	httpresp.OK(c, cronConfigResponse{
		Enabled:                 cfg.Enabled,
		ExpiryCheckIntervalDays: cfg.ExpiryCheckIntervalDays,
		ExpiryCheckPollInterval: cfg.ExpiryCheckPollInterval,
		NotifyEnabled:           cfg.NotifyEnabled,
		NotifyTimeStart:         cfg.NotifyTimeStart,
		NotifyTimeEnd:           cfg.NotifyTimeEnd,
	})
}

type updateCronConfigRequest struct {
	Enabled                 *bool   `json:"enabled,omitempty"`
	ExpiryCheckIntervalDays *int    `json:"expiry_check_interval_days,omitempty"`
	ExpiryCheckPollInterval *string `json:"expiry_check_poll_interval,omitempty"`
	NotifyEnabled           *bool   `json:"notify_enabled,omitempty"`
	NotifyTimeStart         *string `json:"notify_time_start,omitempty"`
	NotifyTimeEnd           *string `json:"notify_time_end,omitempty"`
}

func (h *CronHandler) updateConfig(c *gin.Context) {
	var req updateCronConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.Error(c, http.StatusBadRequest, "invalid request body: "+err.Error())
		return
	}

	if err := config.Save(h.configPath, func(cfg *config.Config) {
		if req.Enabled != nil {
			cfg.Cron.Enabled = *req.Enabled
		}
		if req.ExpiryCheckIntervalDays != nil {
			cfg.Cron.ExpiryCheckIntervalDays = *req.ExpiryCheckIntervalDays
		}
		if req.ExpiryCheckPollInterval != nil {
			cfg.Cron.ExpiryCheckPollInterval = *req.ExpiryCheckPollInterval
		}
		if req.NotifyEnabled != nil {
			cfg.Cron.NotifyEnabled = *req.NotifyEnabled
		}
		if req.NotifyTimeStart != nil {
			cfg.Cron.NotifyTimeStart = *req.NotifyTimeStart
		}
		if req.NotifyTimeEnd != nil {
			cfg.Cron.NotifyTimeEnd = *req.NotifyTimeEnd
		}
	}); err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "save config failed: "+err.Error())
		return
	}

	httpresp.OK(c, gin.H{"message": "cron config updated"})
}
