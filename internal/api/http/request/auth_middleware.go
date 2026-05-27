package request

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

// JWTAuthMiddleware returns a Gin middleware that validates a Bearer JWT
// token and populates the Actor in the request context.
// Returns 401 on invalid or missing token.
func JWTAuthMiddleware(authSvc *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(401, gin.H{"code": 401, "message": "missing authorization header"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(401, gin.H{"code": 401, "message": "invalid authorization format"})
			return
		}

		claims, err := authSvc.ValidateToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{"code": 401, "message": service.ErrInvalidToken.Error()})
			return
		}

		// Overwrite the Actor with JWT-authenticated identity so existing
		// handlers and audit logging continue to work unchanged.
		actor := Actor{
			UserName: claims.Username,
			UserID:   fmt.Sprintf("%d", claims.UserID),
			Channel:  "web",
			TenantID: "default",
		}
		ctx := context.WithValue(c.Request.Context(), ctxKeyActor, actor)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

// APIKeyAuthMiddleware returns a Gin middleware that validates the X-API-Key
// header against the configured API keys and injects an API Actor.
func APIKeyAuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader("X-API-Key")
		if key == "" {
			c.AbortWithStatusJSON(401, gin.H{"code": 401, "message": "missing X-API-Key header"})
			return
		}

		if !cfg.IsValidAPIKey(key) {
			c.AbortWithStatusJSON(401, gin.H{"code": 401, "message": "invalid API key"})
			return
		}

		actor := Actor{
			UserName: "api",
			UserID:   "api",
			Channel:  "api",
			TenantID: "default",
		}
		ctx := context.WithValue(c.Request.Context(), ctxKeyActor, actor)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
