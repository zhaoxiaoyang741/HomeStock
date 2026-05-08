package taskcenter

import (
	"context"
	"strings"
	"time"
)

const (
	TaskStateIdle    = "idle"
	TaskStateRunning = "running"

	TaskTriggerSourceManual    = "manual"
	TaskTriggerSourceScheduled = "scheduled"

	TaskRunStatusRunning = "running"
	TaskRunStatusSuccess = "success"
	TaskRunStatusFailed  = "failed"
	TaskRunStatusSkipped = "skipped"

	LegacySchedulerTaskCode = "expiry_notification"
)

type TaskResult struct {
	Summary string `json:"summary"`
	Payload any    `json:"payload,omitempty"`
}

type TaskDefinition struct {
	Code              string
	Name              string
	Description       string
	DefaultCronSpec   string
	DefaultEnabled    bool
	RunTimeoutSeconds int
	Run               func(ctx context.Context, actor Actor) (TaskResult, error)
}

type UpdateScheduledTaskInput struct {
	CronSpec          *string
	Enabled           *bool
	RunTimeoutSeconds *int
}

type ScheduledTaskView struct {
	ID                string     `json:"id"`
	Code              string     `json:"code"`
	Name              string     `json:"name"`
	Description       string     `json:"description"`
	CronSpec          string     `json:"cron_spec"`
	Enabled           bool       `json:"enabled"`
	Registered        bool       `json:"registered"`
	RunTimeoutSeconds int        `json:"run_timeout_seconds"`
	State             string     `json:"state"`
	NextRunAt         *time.Time `json:"next_run_at"`
	LastRunStartedAt  *time.Time `json:"last_run_started_at"`
	LastRunFinishedAt *time.Time `json:"last_run_finished_at"`
	LastResult        string     `json:"last_result"`
	LastError         string     `json:"last_error"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type SchedulerState string

const (
	SchedulerStateIdle    SchedulerState = "idle"
	SchedulerStateRunning SchedulerState = "running"
)

type SchedulerStatus struct {
	State      SchedulerState `json:"state"`
	LastRunAt  *time.Time     `json:"last_run_at"`
	NextRunAt  *time.Time     `json:"next_run_at"`
	LastResult string         `json:"last_result"`
	LastError  string         `json:"last_error,omitempty"`
}

type Actor struct {
	UserName string
	UserID   string
	Channel  string
	TenantID string
}

func sanitizeTaskDefinition(def TaskDefinition) TaskDefinition {
	def.Code = strings.TrimSpace(def.Code)
	def.Name = strings.TrimSpace(def.Name)
	def.Description = strings.TrimSpace(def.Description)
	def.DefaultCronSpec = strings.TrimSpace(def.DefaultCronSpec)
	if def.RunTimeoutSeconds <= 0 {
		def.RunTimeoutSeconds = 300
	}
	return def
}
