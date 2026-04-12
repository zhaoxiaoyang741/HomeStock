package gormrepo

import (
	"strings"
		"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

// MaterialRepository implements repository.MaterialRepo via GORM.
type MaterialRepository struct{ db *gorm.DB }

func NewMaterialRepository(db *gorm.DB) *MaterialRepository { return &MaterialRepository{db: db} }

func (r *MaterialRepository) Create(material *model.Material) error {
	material.TenantID = normalizeTenantID(material.TenantID)
	material.Name = strings.TrimSpace(material.Name)
	material.Spec = strings.TrimSpace(material.Spec)
	material.DefaultUnit = strings.TrimSpace(material.DefaultUnit)
	if material.DefaultUnit == "" { material.DefaultUnit = "个" }
	if strings.TrimSpace(material.CategoryID) == "" {
		if err := r.assignDefaultCategory(material); err != nil { return err }
	}
	if err := r.validateCategoryID(material.TenantID, material.CategoryID); err != nil { return err }
	return r.db.Create(material).Error
}

func (r *MaterialRepository) Get(id, tenantID string) (*model.Material, error) {
	var m model.Material
	if err := r.scopedMaterials(tenantID).Where("id = ?", strings.TrimSpace(id)).Preload("Category").First(&m).Error; err != nil { return nil, err }
	return &m, nil
}

func (r *MaterialRepository) FindByNaturalKey(tenantID, name, spec string) (*model.Material, error) {
	var m model.Material
	err := r.scopedMaterials(tenantID).Where("LOWER(TRIM(name)) = LOWER(TRIM(?)) AND LOWER(TRIM(spec)) = LOWER(TRIM(?))", strings.TrimSpace(name), strings.TrimSpace(spec)).Preload("Category").First(&m).Error
	if err != nil { return nil, err }
	return &m, nil
}

func (r *MaterialRepository) Update(material *model.Material) error {
	material.TenantID = normalizeTenantID(material.TenantID)
	material.Name = strings.TrimSpace(material.Name)
	material.Spec = strings.TrimSpace(material.Spec)
	material.DefaultUnit = strings.TrimSpace(material.DefaultUnit)
	if material.DefaultUnit == "" { material.DefaultUnit = "个" }
	if err := r.validateCategoryID(material.TenantID, material.CategoryID); err != nil { return err }
	return r.db.Save(material).Error
}

func (r *MaterialRepository) List(filter repository.MaterialFilter) ([]repository.MaterialSummary, error) {
	var materials []model.Material
	q := r.scopedMaterials(filter.TenantID)
	if filter.CategoryID != "" { q = q.Where("category_id = ?", strings.TrimSpace(filter.CategoryID)) }
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		q = q.Where("LOWER(name) LIKE ? OR LOWER(spec) LIKE ?", "%"+strings.ToLower(keyword)+"%", "%"+strings.ToLower(keyword)+"%")
	}
	if err := q.Preload("Category").Order("name ASC, spec ASC").Find(&materials).Error; err != nil { return nil, err }

	summaries := make([]repository.MaterialSummary, 0, len(materials))
	for _, m := range materials {
		summary, err := r.buildSummary(m)
		if err != nil { return nil, err }
		summaries = append(summaries, summary)
	}
	return summaries, nil
}

func (r *MaterialRepository) GetDetail(id, tenantID string) (*repository.MaterialDetail, error) {
	m, err := r.Get(id, tenantID)
	if err != nil { return nil, err }
	summary, err := r.buildSummary(*m)
	if err != nil { return nil, err }
	return &repository.MaterialDetail{MaterialSummary: summary}, nil
}

func (r *MaterialRepository) scopedMaterials(tenantID string) *gorm.DB { return r.db.Model(&model.Material{}).Where("tenant_id = ?", normalizeTenantID(tenantID)) }

func (r *MaterialRepository) buildSummary(material model.Material) (repository.MaterialSummary, error) {
	var lots []model.StockLot
	if err := r.db.Model(&model.StockLot{}).Where("tenant_id = ? AND material_id = ? AND quantity_on_hand > 0", material.TenantID, material.ID).Order("created_at DESC").Find(&lots).Error; err != nil { return repository.MaterialSummary{}, err }
	summary := repository.MaterialSummary{
		ID: material.ID, TenantID: material.TenantID, Name: material.Name, Spec: material.Spec,
		CategoryID: material.CategoryID, Category: material.Category, DefaultUnit: material.DefaultUnit,
		Status: material.Status, CreatedAt: material.CreatedAt, UpdatedAt: material.UpdatedAt,
	}
	locationSet := map[string]struct{}{}
	for _, lot := range lots {
		summary.TotalQuantity += lot.QuantityOnHand
		summary.LotCount++
		if summary.NearestExpireAt == nil && lot.ExpireAt != nil { t := *lot.ExpireAt; summary.NearestExpireAt = &t }
		if lot.ExpireAt != nil && summary.NearestExpireAt != nil && lot.ExpireAt.Before(*summary.NearestExpireAt) { t := *lot.ExpireAt; summary.NearestExpireAt = &t }
		if loc := strings.TrimSpace(lot.Location); loc != "" { locationSet[loc] = struct{}{} }
	}
	for loc := range locationSet { summary.Locations = append(summary.Locations, loc) }
	return summary, nil
}

func (r *MaterialRepository) assignDefaultCategory(material *model.Material) error {
	var cat model.Category
	if err := r.db.Where("tenant_id = ?", material.TenantID).Order("created_at ASC").First(&cat).Error; err != nil { return err }
	material.CategoryID = cat.ID
	return nil
}

func (r *MaterialRepository) validateCategoryID(tenantID, categoryID string) error {
	tid := normalizeTenantID(tenantID)
	trimmed := strings.TrimSpace(categoryID)
	if trimmed == "" { return nil }
	var count int64
	if err := r.db.Model(&model.Category{}).Where("id = ? AND tenant_id = ?", trimmed, tid).Count(&count).Error; err != nil { return err }
	if count == 0 { return repository.ErrInvalidCategoryID }
	return nil
}

