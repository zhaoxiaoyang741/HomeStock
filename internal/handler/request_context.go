package handler

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func tenantIDFromRequest(c *gin.Context) string {
	tenantID := strings.TrimSpace(c.GetHeader("X-Tenant-ID"))
	if tenantID == "" {
		return "default"
	}

	return tenantID
}
