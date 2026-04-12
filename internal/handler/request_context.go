package handler

import (
	"github.com/gin-gonic/gin"
	httpreq "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/request"
)

func tenantIDFromRequest(c *gin.Context) string { return httpreq.TenantID(c) }
