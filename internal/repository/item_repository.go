package repository

import (
	"strings"

	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
)

// ItemFilter constrains list queries for inventory items.
type ItemFilter struct {
	TenantID string
	Location string
	Category string
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
	return r.db.Create(item).Error
}

// List returns items matching the provided filter.
func (r *ItemRepository) List(filter ItemFilter) ([]model.Item, error) {
	query := r.db.Model(&model.Item{})

	tenantID := strings.TrimSpace(filter.TenantID)
	if tenantID == "" {
		tenantID = "default"
	}

	query = query.Where("tenant_id = ?", tenantID)

	if filter.Location != "" {
		query = query.Where("location = ?", filter.Location)
	}

	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}

	var items []model.Item
	if err := query.Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, err
	}

	return items, nil
}
