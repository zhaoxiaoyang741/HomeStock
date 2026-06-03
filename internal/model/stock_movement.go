package model

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// StockMovement is the immutable stock ledger for inbound, consume, and adjustments.
type StockMovement struct {
	ID            string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	TenantID      string    `gorm:"index;type:varchar(36);not null;default:'default'" json:"tenant_id"`
	MaterialID    string    `gorm:"type:varchar(36);not null;index" json:"material_id"`
	Material      *Material `gorm:"foreignKey:MaterialID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"material"`
	LotID         string    `gorm:"type:varchar(36);not null;index" json:"lot_id"`
	Lot           *StockLot `gorm:"foreignKey:LotID;references:ID;constraint:OnUpdate:CASCADE,OnDelete:RESTRICT;" json:"lot"`
	MovementType  string    `gorm:"type:varchar(20);not null;index" json:"movement_type"`
	QuantityDelta float64   `gorm:"not null" json:"quantity_delta"`
	Unit          string    `gorm:"type:varchar(20);not null;default:'件'" json:"unit"`
	Price         float64   `gorm:"not null;default:0" json:"price"`     // 单价
	Reason        string    `gorm:"type:varchar(50);not null;default:''" json:"reason"`
	Channel       string    `gorm:"type:varchar(50);not null;default:'web'" json:"channel"`
	UserName      string    `gorm:"type:varchar(255);default:''" json:"user_name"`
	UserID        string    `gorm:"type:varchar(255);default:''" json:"user_id"`
	Remark        string    `gorm:"type:text;default:''" json:"remark"`
	CreatedAt     time.Time `gorm:"index" json:"created_at"`
}

func (movement *StockMovement) BeforeCreate(*gorm.DB) error {
	if movement.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		movement.ID = id
	}
	if movement.TenantID == "" {
		movement.TenantID = "default"
	}
	if movement.Unit == "" {
		movement.Unit = "件"
	}
	movement.MovementType = strings.TrimSpace(movement.MovementType)
	movement.Reason = strings.TrimSpace(movement.Reason)
	movement.Channel = strings.TrimSpace(movement.Channel)
	if movement.Channel == "" {
		movement.Channel = "web"
	}
	movement.UserName = strings.TrimSpace(movement.UserName)
	movement.UserID = strings.TrimSpace(movement.UserID)
	movement.Remark = strings.TrimSpace(movement.Remark)
	return nil
}
