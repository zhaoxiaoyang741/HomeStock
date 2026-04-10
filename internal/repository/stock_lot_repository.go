package repository

import (
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
)

type StockLotFilter struct {
	TenantID     string
	MaterialID   string
	CategoryID   string
	Location     string
	Status       string
	Keyword      string
	ExpiringSoon bool
}

type StockLotRepository struct {
	db *gorm.DB
}

func NewStockLotRepository(db *gorm.DB) *StockLotRepository {
	return &StockLotRepository{db: db}
}

func (r *StockLotRepository) Create(lot *model.StockLot) error {
	lot.TenantID = normalizeTenantID(lot.TenantID)
	lot.Location = strings.TrimSpace(lot.Location)
	lot.Unit = strings.TrimSpace(lot.Unit)
	lot.Notes = strings.TrimSpace(lot.Notes)
	if lot.Unit == "" {
		lot.Unit = "件"
	}
	if lot.QuantityOnHand <= 0 {
		lot.Status = "depleted"
	} else if lot.Status == "" {
		lot.Status = "active"
	}
	return r.db.Create(lot).Error
}

func (r *StockLotRepository) Get(id string, tenantID string) (*model.StockLot, error) {
	var lot model.StockLot
	if err := r.scopedLots(tenantID).
		Where("id = ?", strings.TrimSpace(id)).
		Preload("Material.Category").
		First(&lot).Error; err != nil {
		return nil, err
	}
	return &lot, nil
}

func (r *StockLotRepository) Update(lot *model.StockLot) error {
	lot.Location = strings.TrimSpace(lot.Location)
	lot.Unit = strings.TrimSpace(lot.Unit)
	lot.Notes = strings.TrimSpace(lot.Notes)
	if lot.QuantityOnHand <= 0 {
		lot.Status = "depleted"
	} else if strings.TrimSpace(lot.Status) == "" {
		lot.Status = "active"
	}
	return r.db.Save(lot).Error
}

func (r *StockLotRepository) List(filter StockLotFilter) ([]model.StockLot, error) {
	query := r.scopedLots(filter.TenantID).Joins("JOIN materials ON materials.id = stock_lots.material_id")
	if filter.MaterialID != "" {
		query = query.Where("stock_lots.material_id = ?", strings.TrimSpace(filter.MaterialID))
	}
	if filter.CategoryID != "" {
		query = query.Where("materials.category_id = ?", strings.TrimSpace(filter.CategoryID))
	}
	if filter.Location != "" {
		query = query.Where("stock_lots.location = ?", strings.TrimSpace(filter.Location))
	}
	if filter.Status != "" {
		query = query.Where("stock_lots.status = ?", strings.TrimSpace(filter.Status))
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		query = query.Where("LOWER(materials.name) LIKE ? OR LOWER(materials.spec) LIKE ?", "%"+strings.ToLower(keyword)+"%", "%"+strings.ToLower(keyword)+"%")
	}
	if filter.ExpiringSoon {
		now := time.Now()
		cutoff := now.Add(7 * 24 * time.Hour)
		query = query.Where("stock_lots.expire_at IS NOT NULL AND stock_lots.expire_at <= ?", cutoff)
	}

	var lots []model.StockLot
	if err := query.
		Preload("Material.Category").
		Order("CASE WHEN stock_lots.expire_at IS NULL THEN 1 ELSE 0 END ASC").
		Order("stock_lots.expire_at ASC").
		Order("stock_lots.purchased_at ASC").
		Order("stock_lots.created_at ASC").
		Find(&lots).Error; err != nil {
		return nil, err
	}
	return lots, nil
}

func (r *StockLotRepository) ListConsumableByMaterial(materialID, tenantID string) ([]model.StockLot, error) {
	var lots []model.StockLot
	if err := r.scopedLots(tenantID).
		Where("material_id = ? AND quantity_on_hand > 0", strings.TrimSpace(materialID)).
		Order("CASE WHEN expire_at IS NULL THEN 1 ELSE 0 END ASC").
		Order("expire_at ASC").
		Order("CASE WHEN purchased_at IS NULL THEN 1 ELSE 0 END ASC").
		Order("purchased_at ASC").
		Order("created_at ASC").
		Find(&lots).Error; err != nil {
		return nil, err
	}
	return lots, nil
}

func (r *StockLotRepository) scopedLots(tenantID string) *gorm.DB {
	return r.db.Model(&model.StockLot{}).Where("stock_lots.tenant_id = ?", normalizeTenantID(tenantID))
}
