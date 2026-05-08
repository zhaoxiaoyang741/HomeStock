package model

import (
	"time"

	"gorm.io/gorm"
)

// ScheduledTask stores the persisted configuration and runtime snapshot of a registered task.
type ScheduledTask struct {
	ID                string     `gorm:"primaryKey;type:varchar(36)" json:"id"`
	Code              string     `gorm:"type:varchar(100);not null;uniqueIndex" json:"code"`
	Name              string     `gorm:"type:varchar(255);not null" json:"name"`
	Description       string     `gorm:"type:text;default:''" json:"description"`
	CronSpec          string     `gorm:"type:varchar(100);not null" json:"cron_spec"`
	Enabled           bool       `gorm:"not null;default:false" json:"enabled"`
	Registered        bool       `gorm:"not null;default:true" json:"registered"`
	RunTimeoutSeconds int        `gorm:"not null;default:300" json:"run_timeout_seconds"`
	NextRunAt         *time.Time `gorm:"index" json:"next_run_at"`
	LastRunStartedAt  *time.Time `json:"last_run_started_at"`
	LastRunFinishedAt *time.Time `json:"last_run_finished_at"`
	LastResult        string     `gorm:"type:varchar(20);default:''" json:"last_result"`
	LastError         string     `gorm:"type:text;default:''" json:"last_error"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func (task *ScheduledTask) BeforeCreate(_ *gorm.DB) error {
	if task.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		task.ID = id
	}
	return nil
}
