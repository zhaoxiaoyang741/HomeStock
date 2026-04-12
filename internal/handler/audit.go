package handler

import (
	"encoding/json"

	"github.com/gin-gonic/gin"

	httpreq "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/request"
	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

type changesPayload struct {
	Before any `json:"before,omitempty"`
	After  any `json:"after,omitempty"`
}

func marshalChanges(before, after any) string {
	b, err := json.Marshal(changesPayload{Before: before, After: after})
	if err != nil { return "" }
	return string(b)
}

func actorFromRequest(c *gin.Context) httpreq.Actor { return httpreq.From(c) }

func recordAuditLog(
	repo repository.AuditLogRepo,
	actor httpreq.Actor,
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

