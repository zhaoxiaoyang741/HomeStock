package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	httpresp "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/response"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
)

type SchedulerHandler struct {
	svc *service.SchedulerService
}

func NewSchedulerHandler(svc *service.SchedulerService) *SchedulerHandler {
	return &SchedulerHandler{svc: svc}
}

func (h *SchedulerHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/scheduler/status", h.GetStatus)
	api.POST("/scheduler/trigger", h.Trigger)
}

func (h *SchedulerHandler) GetStatus(c *gin.Context) {
	httpresp.OK(c, h.svc.GetStatus())
}

func (h *SchedulerHandler) Trigger(c *gin.Context) {
	if err := h.svc.TriggerNow(c.Request.Context()); err != nil {
		httpresp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}
	httpresp.OK(c, gin.H{"message": "check completed"})
}
