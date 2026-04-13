package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	httpreq "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/request"
	httpresp "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/response"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
)

type StockLotHandler struct{ inventory *service.InventoryService }

func NewStockLotHandler(inventory *service.InventoryService) *StockLotHandler {
	return &StockLotHandler{inventory: inventory}
}

func (h *StockLotHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/stock-lots", h.List)
	api.POST("/stock-lots/inbound", h.Inbound)
	api.PUT("/stock-lots/:id", h.Update)
	api.POST("/stock-lots/:id/adjust", h.Adjust)
	api.POST("/stock-lots/:id/void", h.Void)
}

type inboundStockLotRequest struct {
	MaterialID  string  `json:"material_id"`
	Name        string  `json:"name"`
	Spec        string  `json:"spec"`
	CategoryID  string  `json:"category_id"`
	Quantity    float64 `json:"quantity"`
	Unit        string  `json:"unit"`
	Location    string  `json:"location"`
	ExpireAt    string  `json:"expire_at"`
	PurchasedAt string  `json:"purchased_at"`
	Notes       string  `json:"notes"`
}

type updateStockLotRequest struct {
	ExpireAt    *string `json:"expire_at"`
	PurchasedAt *string `json:"purchased_at"`
	Location    *string `json:"location"`
	Notes       *string `json:"notes"`
}

type adjustStockLotRequest struct {
	TargetQuantity float64 `json:"target_quantity"`
	Reason         string  `json:"reason"`
	Remark         string  `json:"remark"`
}

func (h *StockLotHandler) List(c *gin.Context) {
	lots, err := h.inventory.ListLots(c.Request.Context(), repository.StockLotFilter{
		TenantID:     httpreq.TenantID(c),
		MaterialID:   strings.TrimSpace(c.Query("material_id")),
		CategoryID:   strings.TrimSpace(c.Query("category_id")),
		Location:     strings.TrimSpace(c.Query("location")),
		Status:       strings.TrimSpace(c.Query("status")),
		Keyword:      strings.TrimSpace(c.Query("keyword")),
		ExpiringSoon: strings.EqualFold(strings.TrimSpace(c.Query("expiring_soon")), "true"),
	})
	if err != nil {
		handleStockLotRepositoryError(c, err, "list stock lots failed")
		return
	}
	httpresp.List(c, lots, len(lots), 0, 0)
}

func (h *StockLotHandler) Inbound(c *gin.Context) {
	var req inboundStockLotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.Quantity <= 0 {
		httpresp.Error(c, http.StatusBadRequest, "quantity must be greater than 0")
		return
	}
	if strings.TrimSpace(req.MaterialID) == "" && strings.TrimSpace(req.Name) == "" {
		httpresp.Error(c, http.StatusBadRequest, "material_id or name is required")
		return
	}
	actor := svcActorFromRequest(c)
	expireAt, err := parseOptionalRFC3339(req.ExpireAt)
	if err != nil {
		httpresp.Error(c, http.StatusBadRequest, "invalid expire_at, must be RFC3339")
		return
	}
	purchasedAt, err := parseOptionalRFC3339(req.PurchasedAt)
	if err != nil {
		httpresp.Error(c, http.StatusBadRequest, "invalid purchased_at, must be RFC3339")
		return
	}
	lot, err := h.inventory.Inbound(c.Request.Context(), actor, service.InboundInput{
		TenantID: actor.TenantID, Name: req.Name, Spec: req.Spec, CategoryID: req.CategoryID, MaterialID: req.MaterialID, Quantity: req.Quantity, Unit: req.Unit, Location: req.Location, ExpireAt: expireAt, PurchasedAt: purchasedAt, Notes: req.Notes,
	})
	if err != nil {
		handleStockLotRepositoryError(c, err, "inbound stock lot failed")
		return
	}
	httpresp.Created(c, lot)
}

func (h *StockLotHandler) Update(c *gin.Context) {
	var req updateStockLotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	actor := svcActorFromRequest(c)
	var expireAt, purchasedAt *time.Time
	var err error
	if req.ExpireAt != nil {
		expireAt, err = parseNullableRFC3339(*req.ExpireAt)
		if err != nil {
			httpresp.Error(c, http.StatusBadRequest, err.Error())
			return
		}
	}
	if req.PurchasedAt != nil {
		purchasedAt, err = parseNullableRFC3339(*req.PurchasedAt)
		if err != nil {
			httpresp.Error(c, http.StatusBadRequest, err.Error())
			return
		}
	}
	updated, err := h.inventory.UpdateLot(c.Request.Context(), actor, strings.TrimSpace(c.Param("id")), actor.TenantID, service.UpdateLotInput{ExpireAt: expireAt, PurchasedAt: purchasedAt, Location: req.Location, Notes: req.Notes})
	if err != nil {
		handleStockLotRepositoryError(c, err, "update stock lot failed")
		return
	}
	httpresp.OK(c, updated)
}

func (h *StockLotHandler) Void(c *gin.Context) {
	actor := svcActorFromRequest(c)
	updated, err := h.inventory.VoidLot(c.Request.Context(), actor, strings.TrimSpace(c.Param("id")), actor.TenantID)
	if err != nil {
		handleStockLotRepositoryError(c, err, "void stock lot failed")
		return
	}
	httpresp.OK(c, updated)
}

func (h *StockLotHandler) Adjust(c *gin.Context) {
	var req adjustStockLotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		httpresp.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	if req.TargetQuantity < 0 {
		httpresp.Error(c, http.StatusBadRequest, "target_quantity cannot be negative")
		return
	}
	actor := svcActorFromRequest(c)
	updated, err := h.inventory.Adjust(c.Request.Context(), actor, strings.TrimSpace(c.Param("id")), actor.TenantID, req.TargetQuantity, req.Reason, req.Remark)
	if err != nil {
		handleStockLotRepositoryError(c, err, "adjust stock lot failed")
		return
	}
	httpresp.OK(c, updated)
}

// keep helpers for error/status mapping and time parsing
func handleStockLotRepositoryError(c *gin.Context, err error, fallbackMessage string) {
	switch {
	case repository.IsNotFound(err):
		httpresp.Error(c, http.StatusNotFound, "stock lot not found")
	case err == repository.ErrInvalidCategoryID:
		httpresp.Error(c, http.StatusBadRequest, err.Error())
	case err == repository.ErrMaterialUnitMismatch:
		httpresp.Error(c, http.StatusBadRequest, err.Error())
	default:
		httpresp.Error(c, http.StatusInternalServerError, fallbackMessage)
	}
}

func parseOptionalRFC3339(raw string) (*time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func parseNullableRFC3339(raw string) (*time.Time, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, trimmed)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
