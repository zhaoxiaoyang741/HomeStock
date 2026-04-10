package repository

import (
	"time"

	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
)

type StockMovementFilter struct {
	TenantID     string
	MaterialID   string
	LotID        string
	MovementType string
	StartDate    time.Time
	EndDate      time.Time
}

type StockMovementRepository struct {
	db *gorm.DB
}

func NewStockMovementRepository(db *gorm.DB) *StockMovementRepository {
	return &StockMovementRepository{db: db}
}

func (r *StockMovementRepository) Create(movement *model.StockMovement) error {
	movement.TenantID = normalizeTenantID(movement.TenantID)
	return r.db.Create(movement).Error
}

func (r *StockMovementRepository) List(filter StockMovementFilter) ([]model.StockMovement, error) {
	query := r.db.Model(&model.StockMovement{}).Where("tenant_id = ?", normalizeTenantID(filter.TenantID))
	if filter.MaterialID != "" {
		query = query.Where("material_id = ?", filter.MaterialID)
	}
	if filter.LotID != "" {
		query = query.Where("lot_id = ?", filter.LotID)
	}
	if filter.MovementType != "" {
		query = query.Where("movement_type = ?", filter.MovementType)
	}
	if !filter.StartDate.IsZero() {
		query = query.Where("created_at >= ?", filter.StartDate)
	}
	if !filter.EndDate.IsZero() {
		query = query.Where("created_at <= ?", filter.EndDate)
	}

	var movements []model.StockMovement
	if err := query.Preload("Material.Category").Preload("Lot").Order("created_at DESC").Find(&movements).Error; err != nil {
		return nil, err
	}
	return movements, nil
}
