package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

type StockLotHandler struct {
	db           *gorm.DB
	repo         *repository.StockLotRepository
	materialRepo *repository.MaterialRepository
	moveRepo     *repository.StockMovementRepository
	auditRepo    *repository.AuditLogRepository
}

func NewStockLotHandler(
	db *gorm.DB,
	repo *repository.StockLotRepository,
	materialRepo *repository.MaterialRepository,
	moveRepo *repository.StockMovementRepository,
	auditRepo *repository.AuditLogRepository,
) *StockLotHandler {
	return &StockLotHandler{
		db:           db,
		repo:         repo,
		materialRepo: materialRepo,
		moveRepo:     moveRepo,
		auditRepo:    auditRepo,
	}
}

func (h *StockLotHandler) RegisterRoutes(api *gin.RouterGroup) {
	api.GET("/stock-lots", h.List)
	api.POST("/stock-lots/inbound", h.Inbound)
	api.PUT("/stock-lots/:id", h.Update)
	api.POST("/stock-lots/:id/adjust", h.Adjust)
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
	lots, err := h.repo.List(repository.StockLotFilter{
		TenantID:     tenantIDFromRequest(c),
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
	c.JSON(http.StatusOK, gin.H{
		"lots":  lots,
		"total": len(lots),
	})
}

func (h *StockLotHandler) Inbound(c *gin.Context) {
	var req inboundStockLotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Quantity <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "quantity must be greater than 0"})
		return
	}
	if strings.TrimSpace(req.MaterialID) == "" && strings.TrimSpace(req.Name) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "material_id or name is required"})
		return
	}

	expireAt, err := parseOptionalRFC3339(req.ExpireAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid expire_at, must be RFC3339"})
		return
	}
	purchasedAt, err := parseOptionalRFC3339(req.PurchasedAt)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid purchased_at, must be RFC3339"})
		return
	}

	tenantID := tenantIDFromRequest(c)
	actor := actorFromRequest(c)
	var createdLot *model.StockLot
	err = h.db.Transaction(func(tx *gorm.DB) error {
		materialRepo := repository.NewMaterialRepository(tx)
		lotRepo := repository.NewStockLotRepository(tx)
		moveRepo := repository.NewStockMovementRepository(tx)
		auditRepo := repository.NewAuditLogRepository(tx)

		material, err := resolveInboundMaterial(materialRepo, tenantID, &req)
		if err != nil {
			return err
		}

		unit := strings.TrimSpace(req.Unit)
		if unit == "" {
			unit = material.DefaultUnit
		}
		if material.DefaultUnit != "" && unit != material.DefaultUnit {
			return repository.ErrMaterialUnitMismatch
		}

		lot := &model.StockLot{
			TenantID:       tenantID,
			MaterialID:     material.ID,
			QuantityOnHand: req.Quantity,
			Unit:           unit,
			Location:       strings.TrimSpace(req.Location),
			ExpireAt:       expireAt,
			PurchasedAt:    purchasedAt,
			ReceivedAt:     time.Now(),
			Notes:          strings.TrimSpace(req.Notes),
			Status:         "active",
		}
		if err := lotRepo.Create(lot); err != nil {
			return err
		}
		if err := moveRepo.Create(&model.StockMovement{
			TenantID:      tenantID,
			MaterialID:    material.ID,
			LotID:         lot.ID,
			MovementType:  "inbound",
			QuantityDelta: req.Quantity,
			Unit:          unit,
			Reason:        "inbound",
			Channel:       actor.Channel,
			UserName:      actor.UserName,
			UserID:        actor.UserID,
			Remark:        strings.TrimSpace(req.Notes),
		}); err != nil {
			return err
		}

		createdLot, err = lotRepo.Get(lot.ID, tenantID)
		if err != nil {
			return err
		}

		recordAuditLog(auditRepo, actor, "create", "stock_lot", lot.ID, material.Name, marshalChanges(nil, createdLot))
		return nil
	})
	if err != nil {
		handleStockLotRepositoryError(c, err, "inbound stock lot failed")
		return
	}

	c.JSON(http.StatusCreated, createdLot)
}

func (h *StockLotHandler) Update(c *gin.Context) {
	var req updateStockLotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	lot, err := h.repo.Get(strings.TrimSpace(c.Param("id")), tenantIDFromRequest(c))
	if err != nil {
		handleStockLotRepositoryError(c, err, "get stock lot failed")
		return
	}

	before := *lot
	if lot.ExpireAt != nil {
		t := *lot.ExpireAt
		before.ExpireAt = &t
	}
	if lot.PurchasedAt != nil {
		t := *lot.PurchasedAt
		before.PurchasedAt = &t
	}

	if req.ExpireAt != nil {
		expireAt, err := parseNullableRFC3339(*req.ExpireAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		lot.ExpireAt = expireAt
	}
	if req.PurchasedAt != nil {
		purchasedAt, err := parseNullableRFC3339(*req.PurchasedAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		lot.PurchasedAt = purchasedAt
	}
	if req.Location != nil {
		lot.Location = strings.TrimSpace(*req.Location)
	}
	if req.Notes != nil {
		lot.Notes = strings.TrimSpace(*req.Notes)
	}

	if err := h.repo.Update(lot); err != nil {
		handleStockLotRepositoryError(c, err, "update stock lot failed")
		return
	}

	updated, err := h.repo.Get(lot.ID, lot.TenantID)
	if err != nil {
		handleStockLotRepositoryError(c, err, "get stock lot failed")
		return
	}
	recordAuditLog(h.auditRepo, actorFromRequest(c), "update", "stock_lot", updated.ID, updated.Material.Name, marshalChanges(before, updated))
	c.JSON(http.StatusOK, updated)
}

func (h *StockLotHandler) Adjust(c *gin.Context) {
	var req adjustStockLotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.TargetQuantity < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_quantity cannot be negative"})
		return
	}

	tenantID := tenantIDFromRequest(c)
	actor := actorFromRequest(c)
	lot, err := h.repo.Get(strings.TrimSpace(c.Param("id")), tenantID)
	if err != nil {
		handleStockLotRepositoryError(c, err, "get stock lot failed")
		return
	}
	before := *lot
	delta := req.TargetQuantity - lot.QuantityOnHand

	err = h.db.Transaction(func(tx *gorm.DB) error {
		lotRepo := repository.NewStockLotRepository(tx)
		moveRepo := repository.NewStockMovementRepository(tx)
		auditRepo := repository.NewAuditLogRepository(tx)

		lot.QuantityOnHand = req.TargetQuantity
		if req.TargetQuantity == 0 {
			lot.Status = "depleted"
		} else {
			lot.Status = "active"
		}
		if err := lotRepo.Update(lot); err != nil {
			return err
		}
		if delta != 0 {
			if err := moveRepo.Create(&model.StockMovement{
				TenantID:      tenantID,
				MaterialID:    lot.MaterialID,
				LotID:         lot.ID,
				MovementType:  "adjustment",
				QuantityDelta: delta,
				Unit:          lot.Unit,
				Reason:        strings.TrimSpace(req.Reason),
				Channel:       actor.Channel,
				UserName:      actor.UserName,
				UserID:        actor.UserID,
				Remark:        strings.TrimSpace(req.Remark),
			}); err != nil {
				return err
			}
		}
		recordAuditLog(auditRepo, actor, "update", "stock_lot", lot.ID, lot.Material.Name, marshalChanges(before, lot))
		return nil
	})
	if err != nil {
		handleStockLotRepositoryError(c, err, "adjust stock lot failed")
		return
	}

	updated, err := h.repo.Get(lot.ID, tenantID)
	if err != nil {
		handleStockLotRepositoryError(c, err, "get stock lot failed")
		return
	}
	c.JSON(http.StatusOK, updated)
}

func resolveInboundMaterial(repo *repository.MaterialRepository, tenantID string, req *inboundStockLotRequest) (*model.Material, error) {
	if materialID := strings.TrimSpace(req.MaterialID); materialID != "" {
		return repo.Get(materialID, tenantID)
	}

	name := strings.TrimSpace(req.Name)
	spec := strings.TrimSpace(req.Spec)
	material, err := repo.FindByNaturalKey(tenantID, name, spec)
	if err == nil {
		return material, nil
	}
	if !repository.IsNotFound(err) {
		return nil, err
	}

	material = &model.Material{
		TenantID:    tenantID,
		Name:        name,
		Spec:        spec,
		CategoryID:  strings.TrimSpace(req.CategoryID),
		DefaultUnit: strings.TrimSpace(req.Unit),
	}
	if err := repo.Create(material); err != nil {
		return nil, err
	}
	return repo.Get(material.ID, tenantID)
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

func handleStockLotRepositoryError(c *gin.Context, err error, fallbackMessage string) {
	switch {
	case repository.IsNotFound(err):
		c.JSON(http.StatusNotFound, gin.H{"error": "stock lot not found"})
	case err == repository.ErrInvalidCategoryID:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case err == repository.ErrMaterialUnitMismatch:
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"error": fallbackMessage})
	}
}
