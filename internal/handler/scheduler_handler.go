package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	httpresp "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/response"
	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/taskcenter"
)

type SchedulerHandler struct {
	svc *taskcenter.TaskCenterService
}

func NewSchedulerHandler(svc *taskcenter.TaskCenterService) *SchedulerHandler {
	return &SchedulerHandler{svc: svc}
}

func (h *SchedulerHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/scheduler/status", h.GetStatus)
	api.POST("/scheduler/trigger", h.Trigger)
}

func (h *SchedulerHandler) GetStatus(c *gin.Context) {
	status, err := h.svc.GetLegacySchedulerStatus(c.Request.Context())
	if err != nil {
		handleScheduledTaskError(c, err, "get scheduler status failed")
		return
	}
	httpresp.OK(c, status)
}

func (h *SchedulerHandler) Trigger(c *gin.Context) {
	run, err := h.svc.TriggerLegacySchedulerTask(c.Request.Context(), taskActorFromRequest(c))
	if err != nil {
		handleScheduledTaskError(c, err, "trigger scheduler task failed")
		return
	}
	c.JSON(http.StatusAccepted, httpresp.Result[*model.ScheduledTaskRun]{Code: 0, Message: "accepted", Data: run})
}
