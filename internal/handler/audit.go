package handler

import (
	"encoding/json"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

type actorInfo struct {
	UserName string
	UserID   string
	Channel  string
	TenantID string
}

func actorFromRequest(c *gin.Context) actorInfo {
	channel := strings.TrimSpace(c.GetHeader("X-Channel"))
	if channel == "" {
		channel = "web"
	}
	return actorInfo{
		UserName: strings.TrimSpace(c.GetHeader("X-User-Name")),
		UserID:   strings.TrimSpace(c.GetHeader("X-User-ID")),
		Channel:  channel,
		TenantID: tenantIDFromRequest(c),
	}
}

type changesPayload struct {
	Before any `json:"before,omitempty"`
	After  any `json:"after,omitempty"`
}

func marshalChanges(before, after any) string {
	b, err := json.Marshal(changesPayload{Before: before, After: after})
	if err != nil {
		return ""
	}
	return string(b)
}

func recordAuditLog(
	repo *repository.AuditLogRepository,
	actor actorInfo,
	action, entityType, entityID, entityName, changes string,
) {
	// Errors intentionally swallowed — audit failure must not break the primary operation.
	_ = repo.Create(&model.AuditLog{
		TenantID:      actor.TenantID,
		UserName:      actor.UserName,
		UserID:        actor.UserID,
		Channel:       actor.Channel,
		Action:        action,
		EntityType:    entityType,
		EntityID:      entityID,
		EntityName:    entityName,
		ChangesDetail: changes,
	})
}
