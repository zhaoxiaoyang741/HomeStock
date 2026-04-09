package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

// AuditLogHandler serves audit log query endpoints.
type AuditLogHandler struct {
	repo *repository.AuditLogRepository
}

// NewAuditLogHandler creates an audit log handler.
func NewAuditLogHandler(repo *repository.AuditLogRepository) *AuditLogHandler {
	return &AuditLogHandler{repo: repo}
}

// RegisterRoutes mounts audit log endpoints under /api/v1.
func (h *AuditLogHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/audit-logs", h.List)
}

// List handles GET /audit-logs.
func (h *AuditLogHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := repository.AuditLogFilter{
		TenantID: tenantIDFromRequest(c),
		Action:   strings.TrimSpace(c.Query("action")),
		Channel:  strings.TrimSpace(c.Query("channel")),
		UserName: strings.TrimSpace(c.Query("user_name")),
		Page:     page,
		PageSize: pageSize,
	}

	if startStr := strings.TrimSpace(c.Query("start_date")); startStr != "" {
		if t, err := time.Parse("2006-01-02", startStr); err == nil {
			filter.StartDate = t
		} else if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			filter.StartDate = t
		}
	}

	if endStr := strings.TrimSpace(c.Query("end_date")); endStr != "" {
		if t, err := time.Parse("2006-01-02", endStr); err == nil {
			// Include the entire end day.
			filter.EndDate = t.Add(24*time.Hour - time.Second)
		} else if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			filter.EndDate = t
		}
	}

	result, err := h.repo.List(filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list audit logs failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":      result.Logs,
		"total":     result.Total,
		"page":      result.Page,
		"page_size": result.PageSize,
	})
}
