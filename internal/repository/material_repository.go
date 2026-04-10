package repository

import (
	"strings"
	"time"

	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
)

// MaterialFilter constrains material list queries.
type MaterialFilter struct {
	TenantID   string
	CategoryID string
	Keyword    string
}

// MaterialSummary is the aggregate inventory view returned by material list APIs.
type MaterialSummary struct {
	ID              string          `json:"id"`
	TenantID        string          `json:"tenant_id"`
	Name            string          `json:"name"`
	Spec            string          `json:"spec"`
	CategoryID      string          `json:"category_id"`
	Category        *model.Category `json:"category"`
	DefaultUnit     string          `json:"default_unit"`
	Status          string          `json:"status"`
	TotalQuantity   float64         `json:"total_quantity"`
	LotCount        int64           `json:"lot_count"`
	NearestExpireAt *time.Time      `json:"nearest_expire_at"`
	Locations       []string        `json:"locations"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// MaterialDetail represents a single material with its aggregate stock info.
type MaterialDetail struct {
	MaterialSummary
}

// MaterialRepository stores and queries material master data.
type MaterialRepository struct {
	db *gorm.DB
}

func NewMaterialRepository(db *gorm.DB) *MaterialRepository {
	return &MaterialRepository{db: db}
}

func (r *MaterialRepository) Create(material *model.Material) error {
	material.TenantID = normalizeTenantID(material.TenantID)
	material.Name = strings.TrimSpace(material.Name)
	material.Spec = strings.TrimSpace(material.Spec)
	material.DefaultUnit = strings.TrimSpace(material.DefaultUnit)
	if material.DefaultUnit == "" {
		material.DefaultUnit = "件"
	}
	if strings.TrimSpace(material.CategoryID) == "" {
		if err := r.assignDefaultCategory(material); err != nil {
			return err
		}
	}
	if err := r.validateCategoryID(material.TenantID, material.CategoryID); err != nil {
		return err
	}
	return r.db.Create(material).Error
}

func (r *MaterialRepository) Get(id string, tenantID string) (*model.Material, error) {
	var material model.Material
	if err := r.scopedMaterials(tenantID).
		Where("id = ?", strings.TrimSpace(id)).
		Preload("Category").
		First(&material).Error; err != nil {
		return nil, err
	}
	return &material, nil
}

func (r *MaterialRepository) FindByNaturalKey(tenantID, name, spec string) (*model.Material, error) {
	var material model.Material
	err := r.scopedMaterials(tenantID).
		Where(
			"LOWER(TRIM(name)) = LOWER(TRIM(?)) AND LOWER(TRIM(spec)) = LOWER(TRIM(?))",
			strings.TrimSpace(name),
			strings.TrimSpace(spec),
		).
		Preload("Category").
		First(&material).Error
	if err != nil {
		return nil, err
	}
	return &material, nil
}

func (r *MaterialRepository) Update(material *model.Material) error {
	material.TenantID = normalizeTenantID(material.TenantID)
	material.Name = strings.TrimSpace(material.Name)
	material.Spec = strings.TrimSpace(material.Spec)
	material.DefaultUnit = strings.TrimSpace(material.DefaultUnit)
	if material.DefaultUnit == "" {
		material.DefaultUnit = "件"
	}
	if err := r.validateCategoryID(material.TenantID, material.CategoryID); err != nil {
		return err
	}
	return r.db.Save(material).Error
}

func (r *MaterialRepository) List(filter MaterialFilter) ([]MaterialSummary, error) {
	var materials []model.Material
	query := r.scopedMaterials(filter.TenantID)
	if filter.CategoryID != "" {
		query = query.Where("category_id = ?", strings.TrimSpace(filter.CategoryID))
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		query = query.Where("LOWER(name) LIKE ? OR LOWER(spec) LIKE ?", "%"+strings.ToLower(keyword)+"%", "%"+strings.ToLower(keyword)+"%")
	}
	if err := query.Preload("Category").Order("name ASC, spec ASC").Find(&materials).Error; err != nil {
		return nil, err
	}

	summaries := make([]MaterialSummary, 0, len(materials))
	for _, material := range materials {
		summary, err := r.buildSummary(material)
		if err != nil {
			return nil, err
		}
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

func (r *MaterialRepository) GetDetail(id string, tenantID string) (*MaterialDetail, error) {
	material, err := r.Get(id, tenantID)
	if err != nil {
		return nil, err
	}
	summary, err := r.buildSummary(*material)
	if err != nil {
		return nil, err
	}
	return &MaterialDetail{MaterialSummary: summary}, nil
}

func (r *MaterialRepository) scopedMaterials(tenantID string) *gorm.DB {
	return r.db.Model(&model.Material{}).Where("tenant_id = ?", normalizeTenantID(tenantID))
}

func (r *MaterialRepository) buildSummary(material model.Material) (MaterialSummary, error) {
	var lots []model.StockLot
	if err := r.db.Model(&model.StockLot{}).
		Where("tenant_id = ? AND material_id = ? AND quantity_on_hand > 0", material.TenantID, material.ID).
		Order("created_at DESC").
		Find(&lots).Error; err != nil {
		return MaterialSummary{}, err
	}

	summary := MaterialSummary{
		ID:          material.ID,
		TenantID:    material.TenantID,
		Name:        material.Name,
		Spec:        material.Spec,
		CategoryID:  material.CategoryID,
		Category:    material.Category,
		DefaultUnit: material.DefaultUnit,
		Status:      material.Status,
		CreatedAt:   material.CreatedAt,
		UpdatedAt:   material.UpdatedAt,
	}
	locationSet := map[string]struct{}{}
	for _, lot := range lots {
		summary.TotalQuantity += lot.QuantityOnHand
		summary.LotCount++
		if summary.NearestExpireAt == nil && lot.ExpireAt != nil {
			t := *lot.ExpireAt
			summary.NearestExpireAt = &t
		}
		if lot.ExpireAt != nil && summary.NearestExpireAt != nil && lot.ExpireAt.Before(*summary.NearestExpireAt) {
			t := *lot.ExpireAt
			summary.NearestExpireAt = &t
		}
		if location := strings.TrimSpace(lot.Location); location != "" {
			locationSet[location] = struct{}{}
		}
	}
	for location := range locationSet {
		summary.Locations = append(summary.Locations, location)
	}
	return summary, nil
}

func (r *MaterialRepository) assignDefaultCategory(material *model.Material) error {
	var cat model.Category
	if err := r.db.Where("tenant_id = ?", material.TenantID).Order("created_at ASC").First(&cat).Error; err != nil {
		return err
	}
	material.CategoryID = cat.ID
	return nil
}

func (r *MaterialRepository) validateCategoryID(tenantID string, categoryID string) error {
	trimmedCategoryID := strings.TrimSpace(categoryID)
	if trimmedCategoryID == "" {
		return nil
	}

	var count int64
	if err := r.db.Model(&model.Category{}).
		Where("id = ? AND tenant_id = ?", trimmedCategoryID, normalizeTenantID(tenantID)).
		Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return ErrInvalidCategoryID
	}
	return nil
}
