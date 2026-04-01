package repository

import (
	"errors"
	"strings"

	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
)

// ItemFilter constrains list queries for inventory items.
type ItemFilter struct {
	TenantID   string
	Location   string
	CategoryID string
}

// ItemRepository stores and fetches inventory items.
type ItemRepository struct {
	db *gorm.DB
}

// NewItemRepository creates a repository backed by GORM.
func NewItemRepository(db *gorm.DB) *ItemRepository {
	return &ItemRepository{db: db}
}

// Create inserts a new item record.
func (r *ItemRepository) Create(item *model.Item) error {
	item.TenantID = normalizeTenantID(item.TenantID)
	if err := r.validateCategoryID(item.TenantID, item.CategoryID); err != nil {
		return err
	}

	return r.db.Create(item).Error
}

// Get returns a single item by id and tenant.
func (r *ItemRepository) Get(id string, tenantID string) (*model.Item, error) {
	var item model.Item
	if err := r.scopedItems(tenantID).Where("id = ?", id).Preload("Category").First(&item).Error; err != nil {
		return nil, err
	}

	return &item, nil
}

// List returns items matching the provided filter.
func (r *ItemRepository) List(filter ItemFilter) ([]model.Item, error) {
	query := r.scopedItems(filter.TenantID)

	if filter.Location != "" {
		query = query.Where("location = ?", filter.Location)
	}

	if filter.CategoryID != "" {
		query = query.Where("category_id = ?", filter.CategoryID)
	}

	var items []model.Item
	if err := query.Preload("Category").Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, err
	}

	return items, nil
}

// Update persists changes to an item.
func (r *ItemRepository) Update(item *model.Item) error {
	item.TenantID = normalizeTenantID(item.TenantID)
	if err := r.validateCategoryID(item.TenantID, item.CategoryID); err != nil {
		return err
	}

	return r.db.Save(item).Error
}

// Delete removes an item by id and tenant.
func (r *ItemRepository) Delete(id string, tenantID string) error {
	result := r.scopedItems(tenantID).Where("id = ?", id).Delete(&model.Item{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

func (r *ItemRepository) scopedItems(tenantID string) *gorm.DB {
	return r.db.Model(&model.Item{}).Where("tenant_id = ?", normalizeTenantID(tenantID))
}

// IsNotFound reports whether the error means the record was not found.
func IsNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}

func (r *ItemRepository) validateCategoryID(tenantID string, categoryID string) error {
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
