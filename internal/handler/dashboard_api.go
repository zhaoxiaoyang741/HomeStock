package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	httpreq "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/request"
	httpresp "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/response"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
	gormrepo "github.com/zhaoxiaoyang741/HomeStock/internal/repository/gorm"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

// DashboardHandler serves dashboard statistics endpoints.
type DashboardHandler struct {
	uow *gormrepo.UnitOfWork
	cfg *config.Config
}

// NewDashboardHandler creates a DashboardHandler.
func NewDashboardHandler(uow *gormrepo.UnitOfWork, cfg *config.Config) *DashboardHandler {
	return &DashboardHandler{uow: uow, cfg: cfg}
}

// RegisterRoutes mounts dashboard endpoints.
func (h *DashboardHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/dashboard/stats", h.Stats)
}

type dashboardStatsResponse struct {
	TotalMaterials  int64 `json:"total_materials"`
	TotalLots       int64 `json:"total_lots"`
	ExpiringSoon    int64 `json:"expiring_soon"`
	TotalCategories int64 `json:"total_categories"`
}

// Stats handles GET /api/v1/dashboard/stats.
func (h *DashboardHandler) Stats(c *gin.Context) {
	tenantID := httpreq.TenantID(c)
	repos := h.uow.Repos()

	// Count materials
	materials, err := repos.Materials().List(repository.MaterialFilter{TenantID: tenantID, ShowZeroStock: true})
	totalMaterials := int64(0)
	if err != nil {
		logger.ErrorCF("handler", "dashboard: list materials failed", map[string]any{"error": err.Error()})
	} else {
		totalMaterials = int64(len(materials))
	}

	// Count lots and expiring soon
	lots, err := repos.StockLots().List(repository.StockLotFilter{TenantID: tenantID, ShowZeroStock: true})
	totalLots := int64(0)
	expiringSoon := int64(0)
	if err != nil {
		logger.ErrorCF("handler", "dashboard: list lots failed", map[string]any{"error": err.Error()})
	} else {
		totalLots = int64(len(lots))
		expiryDays := h.cfg.Cron.ExpiryCheckIntervalDays
		if expiryDays <= 0 {
			expiryDays = 7
		}
		threshold := time.Hour * 24 * time.Duration(expiryDays)
		for _, lot := range lots {
			if lot.Status == "active" && lot.ExpireAt != nil && time.Until(*lot.ExpireAt) <= threshold {
				expiringSoon++
			}
		}
	}

	// Count categories
	categories, err := repos.Categories().List(tenantID)
	totalCategories := int64(0)
	if err != nil {
		logger.ErrorCF("handler", "dashboard: list categories failed", map[string]any{"error": err.Error()})
	} else {
		totalCategories = int64(len(categories))
	}

	httpresp.OK(c, dashboardStatsResponse{
		TotalMaterials:  totalMaterials,
		TotalLots:       totalLots,
		ExpiringSoon:    expiringSoon,
		TotalCategories: totalCategories,
	})
}
