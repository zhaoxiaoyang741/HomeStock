package handler

import (
	"github.com/gin-gonic/gin"
	httpreq "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/request"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
)

func tenantIDFromRequest(c *gin.Context) string { return httpreq.TenantID(c) }

func svcActorFromRequest(c *gin.Context) service.Actor {
	a := httpreq.From(c)
	return service.Actor{UserName: a.UserName, UserID: a.UserID, Channel: a.Channel, TenantID: a.TenantID}
}
