package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	httpresp "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/response"
	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

type NotificationHandler struct {
	repo repository.NotificationRepo
}

func NewNotificationHandler(repo repository.NotificationRepo) *NotificationHandler {
	return &NotificationHandler{repo: repo}
}

func (h *NotificationHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/notifications", h.List)
}

func (h *NotificationHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := repository.NotificationFilter{
		LotID:    strings.TrimSpace(c.Query("lot_id")),
		Status:   strings.TrimSpace(c.Query("status")),
		Page:     page,
		PageSize: pageSize,
	}

	result, err := h.repo.List(filter)
	if err != nil {
		httpresp.Error(c, http.StatusInternalServerError, "list notifications failed")
		return
	}

	httpresp.OK(c, httpresp.Page[*model.Notification]{
		Items:    result.Items,
		Total:    int(result.Total),
		Page:     result.Page,
		PageSize: result.PageSize,
	})
}
