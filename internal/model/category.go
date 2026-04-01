package model

import (
	"time"

	"gorm.io/gorm"
)

// Category groups inventory items under a reusable label.
type Category struct {
	ID        string    `gorm:"primaryKey;type:varchar(10)" json:"id"`
	TenantID  string    `gorm:"type:varchar(36);not null;default:'default';uniqueIndex:idx_categories_tenant_name" json:"tenant_id"`
	Name      string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_categories_tenant_name" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (category *Category) BeforeCreate(*gorm.DB) error {
	if category.ID == "" {
		id, err := newCategoryID()
		if err != nil {
			return err
		}
		category.ID = id
	}

	if category.TenantID == "" {
		category.TenantID = "default"
	}

	return nil
}
