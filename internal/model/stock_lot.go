package model

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// StockLot stores a concrete inbound batch with its own shelf-life and balance.
type StockLot struct {
	ID             string     `gorm:"primaryKey;type:varchar(36)" json:"id"`
	TenantID       string     `gorm:"index;type:varchar(36);not null;default:'default'" json:"tenant_id"`
	MaterialID     string     `gorm:"type:varchar(36);not null;index" json:"material_id"`
	Material       *Material  `gorm:"foreignKey:MaterialID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"material"`
	QuantityOnHand float64    `gorm:"not null;default:0" json:"quantity_on_hand"`
	Unit           string     `gorm:"type:varchar(20);not null;default:'件'" json:"unit"`
	Location       string     `gorm:"type:varchar(100);default:''" json:"location"`
	ExpireAt       *time.Time `gorm:"index" json:"expire_at"`
	PurchasedAt    *time.Time `gorm:"index" json:"purchased_at"`
	ReceivedAt     time.Time  `gorm:"index;not null" json:"received_at"`
	Notes          string     `gorm:"type:text;default:''" json:"notes"`
	Status         string     `gorm:"type:varchar(20);not null;default:'active'" json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (lot *StockLot) BeforeCreate(*gorm.DB) error {
	if lot.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		lot.ID = id
	}
	if lot.TenantID == "" {
		lot.TenantID = "default"
	}
	if lot.Unit == "" {
		lot.Unit = "件"
	}
	lot.Location = strings.TrimSpace(lot.Location)
	lot.Notes = strings.TrimSpace(lot.Notes)
	if lot.ReceivedAt.IsZero() {
		lot.ReceivedAt = time.Now()
	}
	if lot.Status == "" {
		if lot.QuantityOnHand <= 0 {
			lot.Status = "depleted"
		} else {
			lot.Status = "active"
		}
	}
	return nil
}
