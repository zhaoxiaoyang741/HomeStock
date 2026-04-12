package gormrepo

import (
	"strings"
	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

type CategoryRepository struct{ db *gorm.DB }

func NewCategoryRepository(db *gorm.DB) *CategoryRepository { return &CategoryRepository{db: db} }

func (r *CategoryRepository) Create(category *model.Category) error {
	category.TenantID = normalizeTenantID(category.TenantID)
	category.Name = strings.TrimSpace(category.Name)
	if err := r.ensureUniqueName(category.TenantID, category.Name, ""); err != nil { return err }
	return r.db.Create(category).Error
}

func (r *CategoryRepository) Get(id string, tenantID string) (*model.Category, error) {
	var category model.Category
	if err := r.scopedCategories(tenantID).Where("id = ?", strings.TrimSpace(id)).First(&category).Error; err != nil { return nil, err }
	return &category, nil
}

func (r *CategoryRepository) List(tenantID string) ([]model.Category, error) {
	var categories []model.Category
	if err := r.scopedCategories(tenantID).Order("name ASC").Find(&categories).Error; err != nil { return nil, err }
	return categories, nil
}

func (r *CategoryRepository) Update(category *model.Category) error {
	category.TenantID = normalizeTenantID(category.TenantID)
	category.Name = strings.TrimSpace(category.Name)
	if err := r.ensureUniqueName(category.TenantID, category.Name, category.ID); err != nil { return err }
	return r.db.Save(category).Error
}

func (r *CategoryRepository) Delete(id string, tenantID string) error {
	category, err := r.Get(id, tenantID)
	if err != nil { return err }
	var materialCount int64
	if err := r.db.Model(&model.Material{}).Where("tenant_id = ? AND category_id = ?", category.TenantID, category.ID).Count(&materialCount).Error; err != nil { return err }
	if materialCount > 0 { return repository.ErrCategoryInUse }
	return r.db.Delete(category).Error
}

func (r *CategoryRepository) scopedCategories(tenantID string) *gorm.DB {
	return r.db.Model(&model.Category{}).Where("tenant_id = ?", normalizeTenantID(tenantID))
}

func (r *CategoryRepository) ensureUniqueName(tenantID, name, excludeID string) error {
	if name == "" { return nil }
	query := r.scopedCategories(tenantID).Where("name = ?", name)
	if excludeID != "" { query = query.Where("id <> ?", excludeID) }
	var count int64
	if err := query.Count(&count).Error; err != nil { return err }
	if count > 0 { return repository.ErrCategoryNameExists }
	return nil
}
