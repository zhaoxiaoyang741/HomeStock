package repository

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
)

var (
	// ErrInvalidCategoryID reports that a category reference is missing or belongs to another tenant.
	ErrInvalidCategoryID = errors.New("invalid category_id")
	// ErrCategoryInUse reports that a category is still referenced by items.
	ErrCategoryInUse = errors.New("category is in use")
	// ErrCategoryNameExists reports duplicate category names within a tenant.
	ErrCategoryNameExists = errors.New("category name already exists")
)

// CategoryRepository stores and fetches categories.
type CategoryRepository struct {
	db *gorm.DB
}

// NewCategoryRepository creates a category repository backed by GORM.
func NewCategoryRepository(db *gorm.DB) *CategoryRepository {
	return &CategoryRepository{db: db}
}

// Create inserts a category record.
func (r *CategoryRepository) Create(category *model.Category) error {
	category.TenantID = normalizeTenantID(category.TenantID)
	category.Name = strings.TrimSpace(category.Name)

	if err := r.ensureUniqueName(category.TenantID, category.Name, ""); err != nil {
		return err
	}

	return r.db.Create(category).Error
}

// Get returns a single category by id and tenant.
func (r *CategoryRepository) Get(id string, tenantID string) (*model.Category, error) {
	var category model.Category
	if err := r.scopedCategories(tenantID).Where("id = ?", strings.TrimSpace(id)).First(&category).Error; err != nil {
		return nil, err
	}

	return &category, nil
}

// List returns all categories for a tenant.
func (r *CategoryRepository) List(tenantID string) ([]model.Category, error) {
	var categories []model.Category
	if err := r.scopedCategories(tenantID).Order("name ASC").Find(&categories).Error; err != nil {
		return nil, err
	}

	return categories, nil
}

// Update persists changes to a category.
func (r *CategoryRepository) Update(category *model.Category) error {
	category.TenantID = normalizeTenantID(category.TenantID)
	category.Name = strings.TrimSpace(category.Name)

	if err := r.ensureUniqueName(category.TenantID, category.Name, category.ID); err != nil {
		return err
	}

	return r.db.Save(category).Error
}

// Delete removes a category when it is no longer referenced by materials.
func (r *CategoryRepository) Delete(id string, tenantID string) error {
	category, err := r.Get(id, tenantID)
	if err != nil {
		return err
	}

	var materialCount int64
	if err := r.db.Model(&model.Material{}).
		Where("tenant_id = ? AND category_id = ?", category.TenantID, category.ID).
		Count(&materialCount).Error; err != nil {
		return err
	}
	if materialCount > 0 {
		return ErrCategoryInUse
	}

	return r.db.Delete(category).Error
}

func (r *CategoryRepository) scopedCategories(tenantID string) *gorm.DB {
	return r.db.Model(&model.Category{}).Where("tenant_id = ?", normalizeTenantID(tenantID))
}

func (r *CategoryRepository) ensureUniqueName(tenantID string, name string, excludeID string) error {
	if name == "" {
		return nil
	}

	query := r.scopedCategories(tenantID).Where("name = ?", name)
	if excludeID != "" {
		query = query.Where("id <> ?", excludeID)
	}

	var count int64
	if err := query.Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return ErrCategoryNameExists
	}

	return nil
}

func normalizeTenantID(tenantID string) string {
	trimmedTenantID := strings.TrimSpace(tenantID)
	if trimmedTenantID == "" {
		return "default"
	}

	return trimmedTenantID
}
