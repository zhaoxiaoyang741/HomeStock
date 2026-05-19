package cron

import (
	"context"
	"time"
)

// Task is the interface every scheduled task must implement.
type Task interface {
	// Name returns a unique identifier for this task.
	Name() string
	// Run executes the task. The context may carry a deadline.
	Run(ctx context.Context) error
}

// ScheduleDef describes when a task should run.
type ScheduleDef struct {
	// Interval between runs. If zero, the task runs once then is not rescheduled.
	Interval time.Duration
}
