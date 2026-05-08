package model

import (
	"time"

	"gorm.io/gorm"
)

// ScheduledTaskRun stores the execution history of all scheduled tasks in one table.
type ScheduledTaskRun struct {
	ID                  string     `gorm:"primaryKey;type:varchar(36)" json:"id"`
	TaskCode            string     `gorm:"type:varchar(100);not null;index:idx_task_runs_task_code_created" json:"task_code"`
	TaskName            string     `gorm:"type:varchar(255);not null" json:"task_name"`
	TriggerSource       string     `gorm:"type:varchar(20);not null;index" json:"trigger_source"`
	Status              string     `gorm:"type:varchar(20);not null;index" json:"status"`
	Summary             string     `gorm:"type:text;default:''" json:"summary"`
	ResultPayload       string     `gorm:"type:text;default:''" json:"result_payload"`
	ErrorMessage        string     `gorm:"type:text;default:''" json:"error_message"`
	StartedAt           time.Time  `gorm:"not null;index" json:"started_at"`
	FinishedAt          *time.Time `json:"finished_at"`
	DurationMs          int64      `gorm:"not null;default:0" json:"duration_ms"`
	TriggeredByUserName string     `gorm:"type:varchar(255);default:''" json:"triggered_by_user_name"`
	TriggeredByUserID   string     `gorm:"type:varchar(255);default:''" json:"triggered_by_user_id"`
	TriggeredByChannel  string     `gorm:"type:varchar(50);not null;default:'system';index" json:"triggered_by_channel"`
	CreatedAt           time.Time  `gorm:"index:idx_task_runs_task_code_created" json:"created_at"`
}

func (run *ScheduledTaskRun) BeforeCreate(_ *gorm.DB) error {
	if run.ID == "" {
		id, err := newID()
		if err != nil {
			return err
		}
		run.ID = id
	}
	if run.TriggeredByChannel == "" {
		run.TriggeredByChannel = "system"
	}
	return nil
}
