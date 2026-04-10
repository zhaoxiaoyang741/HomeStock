package model

import (
	"time"

	"gorm.io/gorm"
)

// Notification records reminder delivery attempts for stock lots.
type Notification struct {
	ID        string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	LotID     string    `gorm:"type:varchar(36);not null;index" json:"lot_id"`
	Lot       StockLot  `gorm:"foreignKey:LotID;references:ID;constraint:OnDelete:CASCADE;" json:"lot"`
	NotifyAt  time.Time `gorm:"not null;index" json:"notify_at"`
	Status    string    `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	Channel   string    `gorm:"type:varchar(50);not null;default:'feishu'" json:"channel"`
	Message   string    `gorm:"type:text;default:''" json:"message"`
	CreatedAt time.Time `json:"created_at"`
}

func (notification *Notification) BeforeCreate(*gorm.DB) error {
	if notification.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		notification.ID = id
	}

	if notification.Status == "" {
		notification.Status = "pending"
	}

	if notification.Channel == "" {
		notification.Channel = "feishu"
	}

	return nil
}
