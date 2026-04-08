package model

import (
	"time"

	"gorm.io/gorm"
)

// Item represents a household inventory record.
type Item struct {
	ID          string     `gorm:"primaryKey;type:varchar(36)" json:"id"`
	TenantID    string     `gorm:"index;type:varchar(36);not null;default:'default'" json:"tenant_id"`
	Name        string     `gorm:"type:varchar(255);not null" json:"name"`
	CategoryID  string     `gorm:"type:varchar(10);index" json:"category_id"`
	Category    *Category  `gorm:"foreignKey:CategoryID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" json:"category"`
	Quantity    float64    `gorm:"default:1" json:"quantity"`
	Unit        string     `gorm:"type:varchar(20);not null;default:'个'" json:"unit"`
	Spec        string     `gorm:"type:varchar(100);default:''" json:"spec"`
	Location    string     `gorm:"type:varchar(100);default:''" json:"location"`
	ExpireAt    *time.Time `gorm:"index" json:"expire_at"`
	PurchasedAt time.Time  `gorm:"not null" json:"purchased_at"`
	Notes       string     `gorm:"type:text;default:''" json:"notes"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

func (item *Item) BeforeCreate(*gorm.DB) error {
	if item.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		item.ID = id
	}

	if item.TenantID == "" {
		item.TenantID = "default"
	}

	if item.Quantity == 0 {
		item.Quantity = 1
	}

	if item.Unit == "" {
		item.Unit = "个"
	}

	if item.PurchasedAt.IsZero() {
		item.PurchasedAt = time.Now()
	}

	return nil
}
