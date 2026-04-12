package request

import (
	"context"
	"strings"

	"github.com/gin-gonic/gin"
)

// Actor carries authenticated/attributed request identity and tenancy.
type Actor struct {
	UserName string
	UserID   string
	Channel  string
	TenantID string
}

type ctxKey string

const (
	ctxKeyActor ctxKey = "actor"
)

// Middleware captures headers into a typed Actor and stores it in context.
func Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		actor := Actor{
			UserName: strings.TrimSpace(c.GetHeader("X-User-Name")),
			UserID:   strings.TrimSpace(c.GetHeader("X-User-ID")),
			Channel:  strings.TrimSpace(c.GetHeader("X-Channel")),
			TenantID: strings.TrimSpace(c.GetHeader("X-Tenant-ID")),
		}
		if actor.Channel == "" {
			actor.Channel = "web"
		}
		if actor.TenantID == "" {
			actor.TenantID = "default"
		}

		ctx := context.WithValue(c.Request.Context(), ctxKeyActor, actor)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// From returns the Actor from context; zero Actor if missing.
func From(c *gin.Context) Actor {
	if v := c.Request.Context().Value(ctxKeyActor); v != nil {
		if a, ok := v.(Actor); ok {
			return a
		}
	}
	return Actor{}
}

// TenantID returns the resolved tenant id from context (defaults already applied).
func TenantID(c *gin.Context) string { return From(c).TenantID }
