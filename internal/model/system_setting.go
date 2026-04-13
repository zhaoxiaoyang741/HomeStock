package model

import (
	"time"

	"gorm.io/gorm"
)

// SystemSetting stores runtime-editable system settings as a JSON document.
type SystemSetting struct {
	ID                string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Key               string    `gorm:"type:varchar(50);not null;uniqueIndex" json:"key"`
	Payload           string    `gorm:"type:text;not null;default:''" json:"payload"`
	Version           int       `gorm:"not null;default:1" json:"version"`
	UpdatedByUserName string    `gorm:"type:varchar(255);default:''" json:"updated_by_user_name"`
	UpdatedByUserID   string    `gorm:"type:varchar(255);default:''" json:"updated_by_user_id"`
	UpdatedByChannel  string    `gorm:"type:varchar(50);not null;default:'web'" json:"updated_by_channel"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

func (setting *SystemSetting) BeforeCreate(_ *gorm.DB) error {
	if setting.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		setting.ID = id
	}
	if setting.Key == "" {
		setting.Key = "global"
	}
	if setting.Version <= 0 {
		setting.Version = 1
	}
	if setting.UpdatedByChannel == "" {
		setting.UpdatedByChannel = "web"
	}
	return nil
}
