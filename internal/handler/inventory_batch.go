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

// BatchHandler provides batch inventory operations and material name resolution.
type BatchHandler struct {
	inventory   *service.InventoryService
	materialSvc *service.MaterialService
}

func NewBatchHandler(inventory *service.InventoryService, materialSvc *service.MaterialService) *BatchHandler {
	return &BatchHandler{inventory: inventory, materialSvc: materialSvc}
}

func (h *BatchHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.POST("/batch/inbound", h.BatchInbound)
	api.POST("/batch/consume", h.BatchConsume)
	api.POST("/materials/resolve", h.ResolveMaterial)
}

type batchInboundItem struct {
	MaterialID string  `json:"material_id"`
	Name       string  `json:"name"`
	Spec       string  `json:"spec"`
	CategoryID string  `json:"category_id"`
	Quantity   float64 `json:"quantity"`
	Unit       string  `json:"unit"`
	Location   string  `json:"location"`
	ExpireAt   string  `json:"expire_at"`
	Notes      string  `json:"notes"`
}

type batchInboundRequest struct {
	Items []batchInboundItem `json:"items"`
}

type batchItemResult struct {
	Index int         `json:"index"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

// BatchInbound processes multiple inbound operations in one request.
// Each item succeeds or fails independently (partial success model).
func (h *BatchHandler) BatchInbound(c *gin.Context) {
	var req batchInboundRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if len(req.Items) == 0 {
		httpresp.Error(c, http.StatusBadRequest, "items is required")
		return
	}

	actor := svcActorFromRequest(c)
	results := make([]batchItemResult, 0, len(req.Items))

	for i, item := range req.Items {
		expireAt, err := parseOptionalRFC3339(item.ExpireAt)
		if err != nil {
			results = append(results, batchItemResult{Index: i, Error: "invalid expire_at: " + err.Error()})
			continue
		}

		lot, err := h.inventory.Inbound(c.Request.Context(), actor, service.InboundInput{
			TenantID:   actor.TenantID,
			Name:       item.Name,
			Spec:       item.Spec,
			CategoryID: item.CategoryID,
			MaterialID: item.MaterialID,
			Quantity:   item.Quantity,
			Unit:       item.Unit,
			Location:   item.Location,
			ExpireAt:   expireAt,
			Notes:      item.Notes,
		})
		if err != nil {
			results = append(results, batchItemResult{Index: i, Error: err.Error()})
		} else {
			results = append(results, batchItemResult{Index: i, Data: lot})
		}
	}

	httpresp.OK(c, results)
}

type batchConsumeItem struct {
	MaterialID string  `json:"material_id"`
	Quantity   float64 `json:"quantity"`
	Reason     string  `json:"reason"`
}

type batchConsumeRequest struct {
	Items []batchConsumeItem `json:"items"`
}

// BatchConsume processes multiple consume operations in one request.
// Each item succeeds or fails independently.
func (h *BatchHandler) BatchConsume(c *gin.Context) {
	var req batchConsumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if len(req.Items) == 0 {
		httpresp.Error(c, http.StatusBadRequest, "items is required")
		return
	}

	actor := svcActorFromRequest(c)
	results := make([]batchItemResult, 0, len(req.Items))

	for i, item := range req.Items {
		result, err := h.inventory.Consume(c.Request.Context(), actor, item.MaterialID, actor.TenantID, item.Quantity, item.Reason)
		if err != nil {
			results = append(results, batchItemResult{Index: i, Error: err.Error()})
		} else {
			results = append(results, batchItemResult{Index: i, Data: result})
		}
	}

	httpresp.OK(c, results)
}

type resolveMaterialRequest struct {
	Name string `json:"name"`
	Spec string `json:"spec"`
}

type resolveMaterialResult struct {
	MaterialID   string  `json:"material_id"`
	Name         string  `json:"name"`
	Spec         string  `json:"spec"`
	DefaultUnit  string  `json:"default_unit"`
	Score        float64 `json:"score"`
	IsExactMatch bool    `json:"is_exact_match"`
}

// ResolveMaterial returns fuzzy-matched material candidates for a given name.
func (h *BatchHandler) ResolveMaterial(c *gin.Context) {
	var req resolveMaterialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpresp.Error(c, http.StatusBadRequest, "name is required")
		return
	}

	tenantID := httpreq.TenantID(c)
	summaries, err := h.materialSvc.List(c.Request.Context(), repository.MaterialFilter{
		TenantID: tenantID,
		Keyword:       req.Name,
		ShowZeroStock: true,
	})
	if err != nil {
		httpresp.Error(c, http.StatusInternalServerError, err.Error())
		return
	}

	results := make([]resolveMaterialResult, 0, len(summaries))
	for _, s := range summaries {
		score := repository.ComputeMatchScore(req.Name, s.Name, s.Spec)
		results = append(results, resolveMaterialResult{
			MaterialID:   s.ID,
			Name:         s.Name,
			Spec:         s.Spec,
			DefaultUnit:  s.DefaultUnit,
			Score:        score,
			IsExactMatch: score >= 0.95,
		})
	}

	httpresp.OK(c, results)
}

