package repository

import "errors"

var (
	ErrScheduledTaskConflict    = errors.New("scheduled task conflict")
	ErrScheduledTaskRunning     = errors.New("scheduled task is already running")
	ErrScheduledTaskInvalidCron = errors.New("scheduled task cron spec is invalid")
)
