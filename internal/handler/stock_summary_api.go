package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	httpreq "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/request"
	httpresp "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/response"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
	gormrepo "github.com/zhaoxiaoyang741/HomeStock/internal/repository/gorm"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

// StockSummaryHandler serves stock lot summary endpoints.
type StockSummaryHandler struct {
	uow *gormrepo.UnitOfWork
}

// NewStockSummaryHandler creates a StockSummaryHandler.
func NewStockSummaryHandler(uow *gormrepo.UnitOfWork) *StockSummaryHandler {
	return &StockSummaryHandler{uow: uow}
}

// RegisterRoutes mounts stock summary endpoints.
func (h *StockSummaryHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/stock-lots/summary", h.Summary)
}

type stockLotSummaryItem struct {
	MaterialID     string  `json:"material_id"`
	MaterialName   string  `json:"material_name"`
	Spec           string  `json:"spec"`
	Unit           string  `json:"unit"`
	TotalQuantity  float64 `json:"total_quantity"`
	LotCount       int     `json:"lot_count"`
	ActiveLotCount int     `json:"active_lot_count"`
	ExpiringCount  int     `json:"expiring_count"`
}

// Summary handles GET /api/v1/stock-lots/summary.
// Returns stock lot quantities grouped by material.
func (h *StockSummaryHandler) Summary(c *gin.Context) {
	tenantID := httpreq.TenantID(c)
	repos := h.uow.Repos()

	// Fetch all materials (for name/spec lookup)
	materials, err := repos.Materials().List(repository.MaterialFilter{TenantID: tenantID, ShowZeroStock: true})
	if err != nil {
		logger.ErrorCF("handler", "stock summary: list materials failed", map[string]any{"error": err.Error()})
		httpresp.Error(c, http.StatusInternalServerError, "failed to query materials")
		return
	}
	materialMap := make(map[string]repository.MaterialSummary, len(materials))
	for _, m := range materials {
		materialMap[m.ID] = m
	}

	// Fetch all lots
	lots, err := repos.StockLots().List(repository.StockLotFilter{TenantID: tenantID, ShowZeroStock: true})
	if err != nil {
		logger.ErrorCF("handler", "stock summary: list lots failed", map[string]any{"error": err.Error()})
		httpresp.Error(c, http.StatusInternalServerError, "failed to query stock lots")
		return
	}

	// Group lots by material ID
	type materialAgg struct {
		TotalQuantity  float64
		LotCount       int
		ActiveLotCount int
		ExpiringCount  int
	}
	aggregated := make(map[string]*materialAgg)

	for _, lot := range lots {
		agg, exists := aggregated[lot.MaterialID]
		if !exists {
			agg = &materialAgg{}
			aggregated[lot.MaterialID] = agg
		}
		agg.TotalQuantity += lot.QuantityOnHand
		agg.LotCount++
		if lot.Status == "active" {
			agg.ActiveLotCount++
			if lot.ExpireAt != nil && time.Until(*lot.ExpireAt).Hours() <= 7*24 {
				agg.ExpiringCount++
			}
		}
	}

	items := make([]stockLotSummaryItem, 0, len(aggregated))
	for mid, agg := range aggregated {
		info := materialMap[mid]
		items = append(items, stockLotSummaryItem{
			MaterialID:     mid,
			MaterialName:   info.Name,
			Spec:           info.Spec,
			Unit:           info.DefaultUnit,
			TotalQuantity:  agg.TotalQuantity,
			LotCount:       agg.LotCount,
			ActiveLotCount: agg.ActiveLotCount,
			ExpiringCount:  agg.ExpiringCount,
		})
	}

	httpresp.OK(c, items)
}
