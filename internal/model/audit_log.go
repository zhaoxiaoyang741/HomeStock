package model

import (
	"time"

	"gorm.io/gorm"
)

// AuditLog records every create/update/delete operation on items and categories.
type AuditLog struct {
	ID            string    `gorm:"primaryKey;type:varchar(36)"                         json:"id"`
	TenantID      string    `gorm:"index;type:varchar(36);not null;default:'default'"   json:"tenant_id"`
	UserName      string    `gorm:"type:varchar(255);default:''"                        json:"user_name"`
	UserID        string    `gorm:"type:varchar(255);default:''"                        json:"user_id"`
	Channel       string    `gorm:"type:varchar(50);not null;default:'web'"             json:"channel"`
	Action        string    `gorm:"type:varchar(20);not null"                           json:"action"`
	EntityType    string    `gorm:"type:varchar(20);not null"                           json:"entity_type"`
	EntityID      string    `gorm:"type:varchar(36);not null"                           json:"entity_id"`
	EntityName    string    `gorm:"type:varchar(255);default:''"                        json:"entity_name"`
	ChangesDetail string    `gorm:"type:text;default:''"                                json:"changes_detail"`
	CreatedAt     time.Time `gorm:"index"                                               json:"created_at"`
}

func (log *AuditLog) BeforeCreate(_ *gorm.DB) error {
	if log.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		log.ID = id
	}
	if log.TenantID == "" {
		log.TenantID = "default"
	}
	if log.Channel == "" {
		log.Channel = "web"
	}
	return nil
}
