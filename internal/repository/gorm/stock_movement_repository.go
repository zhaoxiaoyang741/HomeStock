package gormrepo

import (
		"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

type StockMovementRepository struct{ db *gorm.DB }

func NewStockMovementRepository(db *gorm.DB) *StockMovementRepository { return &StockMovementRepository{db: db} }

func (r *StockMovementRepository) Create(m *model.StockMovement) error {
	m.TenantID = normalizeTenantID(m.TenantID)
	return r.db.Create(m).Error
}

func (r *StockMovementRepository) List(f repository.StockMovementFilter) ([]model.StockMovement, error) {
	q := r.db.Model(&model.StockMovement{}).Where("tenant_id = ?", normalizeTenantID(f.TenantID))
	if f.MaterialID != "" { q = q.Where("material_id = ?", f.MaterialID) }
	if f.LotID != "" { q = q.Where("lot_id = ?", f.LotID) }
	if f.MovementType != "" { q = q.Where("movement_type = ?", f.MovementType) }
	if !f.StartDate.IsZero() { q = q.Where("created_at >= ?", f.StartDate) }
	if !f.EndDate.IsZero() { q = q.Where("created_at <= ?", f.EndDate) }
	var ms []model.StockMovement
	if err := q.Preload("Material.Category").Preload("Lot").Order("created_at DESC").Find(&ms).Error; err != nil { return nil, err }
	return ms, nil
}

