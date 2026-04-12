package handler

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	httpreq "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/request"
	httpresp "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/response"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
)

type MaterialHandler struct {
	materials *service.MaterialService
	inventory *service.InventoryService
}

func NewMaterialHandler(materials *service.MaterialService, inventory *service.InventoryService) *MaterialHandler {
	return &MaterialHandler{materials: materials, inventory: inventory}
}

func (h *MaterialHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/materials", h.List)
	api.POST("/materials", h.Create)
	api.GET("/materials/:id", h.Get)
	api.POST("/materials/:id/consume", h.Consume)
}

type createMaterialRequest struct {
	Name        string `json:"name" binding:"required"`
	Spec        string `json:"spec"`
	CategoryID  string `json:"category_id"`
	DefaultUnit string `json:"default_unit"`
}

type consumeMaterialRequest struct {
	Quantity float64 `json:"quantity"`
	Reason   string  `json:"reason"`
}

func toSvcActor(a httpreq.Actor) service.Actor {
	return service.Actor{UserName: a.UserName, UserID: a.UserID, Channel: a.Channel, TenantID: a.TenantID}
}

func (h *MaterialHandler) Create(c *gin.Context) {
	var req createMaterialRequest
	if err := c.ShouldBindJSON(&req); err != nil { httpresp.Error(c, http.StatusBadRequest, err.Error()); return }
	if strings.TrimSpace(req.Name) == "" { httpresp.Error(c, http.StatusBadRequest, "name cannot be empty"); return }
	created, err := h.materials.Create(c.Request.Context(), toSvcActor(httpreq.From(c)), req.Name, req.Spec, req.CategoryID, req.DefaultUnit)
	if err != nil { handleMaterialRepositoryError(c, err, "create material failed"); return }
	httpresp.Created(c, created)
}

func (h *MaterialHandler) List(c *gin.Context) {
	summaries, err := h.materials.List(c.Request.Context(), repository.MaterialFilter{
		TenantID:   httpreq.TenantID(c),
		CategoryID: strings.TrimSpace(c.Query("category_id")),
		Keyword:    strings.TrimSpace(c.Query("keyword")),
	})
	if err != nil { handleMaterialRepositoryError(c, err, "list materials failed"); return }
	httpresp.List(c, summaries, len(summaries), 0, 0)
}

func (h *MaterialHandler) Get(c *gin.Context) {
	detail, err := h.materials.GetDetail(c.Request.Context(), strings.TrimSpace(c.Param("id")), httpreq.TenantID(c))
	if err != nil { handleMaterialRepositoryError(c, err, "get material failed"); return }
	httpresp.OK(c, detail)
}

func (h *MaterialHandler) Consume(c *gin.Context) {
	var req consumeMaterialRequest
	if err := c.ShouldBindJSON(&req); err != nil { httpresp.Error(c, http.StatusBadRequest, err.Error()); return }
	if req.Quantity <= 0 { httpresp.Error(c, http.StatusBadRequest, "quantity must be greater than 0"); return }
	actor := toSvcActor(httpreq.From(c))
	materialID := strings.TrimSpace(c.Param("id"))
	results, err := h.inventory.Consume(c.Request.Context(), actor, materialID, actor.TenantID, req.Quantity, req.Reason)
	if err != nil { handleMaterialRepositoryError(c, err, "consume material failed"); return }
	detail, err := h.materials.GetDetail(c.Request.Context(), materialID, actor.TenantID)
	if err != nil { handleMaterialRepositoryError(c, err, "get material failed"); return }
	httpresp.OK(c, gin.H{"material": detail, "requested_quantity": req.Quantity, "unit": detail.DefaultUnit, "consumed_lots": results})
}

func handleMaterialRepositoryError(c *gin.Context, err error, fallbackMessage string) {
	switch {
	case repository.IsNotFound(err):
		httpresp.Error(c, http.StatusNotFound, "material not found")
	case err == repository.ErrInvalidCategoryID:
		httpresp.Error(c, http.StatusBadRequest, err.Error())
	case err == repository.ErrMaterialUnitMismatch:
		httpresp.Error(c, http.StatusBadRequest, err.Error())
	case err == repository.ErrInsufficientStock:
		httpresp.Error(c, http.StatusBadRequest, err.Error())
	default:
		httpresp.Error(c, http.StatusInternalServerError, fallbackMessage)
	}
}


