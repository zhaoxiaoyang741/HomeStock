package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	httpresp "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/response"
	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
	"github.com/zhaoxiaoyang741/HomeStock/internal/taskcenter"
)

type ScheduledTaskHandler struct {
	svc *taskcenter.TaskCenterService
}

func NewScheduledTaskHandler(svc *taskcenter.TaskCenterService) *ScheduledTaskHandler {
	return &ScheduledTaskHandler{svc: svc}
}

func (h *ScheduledTaskHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/scheduled-tasks", h.ListTasks)
	api.GET("/scheduled-tasks/:code", h.GetTask)
	api.PATCH("/scheduled-tasks/:code", h.UpdateTask)
	api.POST("/scheduled-tasks/:code/trigger", h.TriggerTask)
	api.GET("/scheduled-task-runs", h.ListRuns)
}

type updateScheduledTaskRequest struct {
	CronSpec          *string `json:"cron_spec"`
	Enabled           *bool   `json:"enabled"`
	RunTimeoutSeconds *int    `json:"run_timeout_seconds"`
}

func (h *ScheduledTaskHandler) ListTasks(c *gin.Context) {
	items, err := h.svc.ListTasks(c.Request.Context())
	if err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "list scheduled tasks failed")
		return
	}
	httpresp.OK(c, items)
}

func (h *ScheduledTaskHandler) GetTask(c *gin.Context) {
	view, err := h.svc.GetTask(c.Request.Context(), c.Param("code"))
	if err != nil {
		handleScheduledTaskError(c, err, "get scheduled task failed")
		return
	}
	httpresp.OK(c, view)
}

func (h *ScheduledTaskHandler) UpdateTask(c *gin.Context) {
	var req updateScheduledTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.CronSpec == nil && req.Enabled == nil && req.RunTimeoutSeconds == nil {
		httpresp.Error(c, http.StatusBadRequest, "at least one field must be provided")
		return
	}

	view, err := h.svc.UpdateTask(c.Request.Context(), taskActorFromRequest(c), c.Param("code"), taskcenter.UpdateScheduledTaskInput{
		CronSpec:          trimOptionalString(req.CronSpec),
		Enabled:           req.Enabled,
		RunTimeoutSeconds: req.RunTimeoutSeconds,
	})
	if err != nil {
		handleScheduledTaskError(c, err, "update scheduled task failed")
		return
	}
	httpresp.OK(c, view)
}

func (h *ScheduledTaskHandler) TriggerTask(c *gin.Context) {
	run, err := h.svc.TriggerTask(c.Request.Context(), taskActorFromRequest(c), c.Param("code"))
	if err != nil {
		handleScheduledTaskError(c, err, "trigger scheduled task failed")
		return
	}
	c.JSON(http.StatusAccepted, httpresp.Result[*model.ScheduledTaskRun]{Code: 0, Message: "accepted", Data: run})
}

func (h *ScheduledTaskHandler) ListRuns(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	result, err := h.svc.ListRuns(c.Request.Context(), repository.ScheduledTaskRunFilter{
		TaskCode:      strings.TrimSpace(c.Query("task_code")),
		Status:        strings.TrimSpace(c.Query("status")),
		TriggerSource: strings.TrimSpace(c.Query("trigger_source")),
		Page:          page,
		PageSize:      pageSize,
	})
	if err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "list scheduled task runs failed")
		return
	}

	httpresp.OK(c, httpresp.Page[*model.ScheduledTaskRun]{
		Items:    result.Items,
		Total:    int(result.Total),
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}

func handleScheduledTaskError(c *gin.Context, err error, fallback string) {
	switch {
	case repository.IsNotFound(err):
		httpresp.Error(c, http.StatusNotFound, "scheduled task not found")
	case err == repository.ErrScheduledTaskRunning:
		httpresp.Error(c, http.StatusConflict, err.Error())
	case strings.Contains(err.Error(), repository.ErrScheduledTaskInvalidCron.Error()):
		httpresp.Error(c, http.StatusBadRequest, err.Error())
	case strings.Contains(err.Error(), "run_timeout_seconds"):
		httpresp.Error(c, http.StatusBadRequest, err.Error())
	default:
		httpresp.Error(c, http.StatusInternalServerError, fallback)
	}
}

func trimOptionalString(v *string) *string {
	if v == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*v)
	return &trimmed
}
