package cron

import (
	"sync"
	"time"
)

// Job wraps a Task with its schedule and runtime state.
type Job struct {
	mu       sync.RWMutex
	task     Task
	schedule ScheduleDef
	nextRun  time.Time
	lastRun  time.Time
	lastErr  error
	enabled  bool
}

func newJob(task Task, schedule ScheduleDef) *Job {
	return &Job{
		task:     task,
		schedule: schedule,
		enabled:  true,
	}
}

func (j *Job) Name() string           { return j.task.Name() }
func (j *Job) NextRun() time.Time     { j.mu.RLock(); defer j.mu.RUnlock(); return j.nextRun }
func (j *Job) LastRun() time.Time     { j.mu.RLock(); defer j.mu.RUnlock(); return j.lastRun }
func (j *Job) LastError() error       { j.mu.RLock(); defer j.mu.RUnlock(); return j.lastErr }
func (j *Job) Enabled() bool          { j.mu.RLock(); defer j.mu.RUnlock(); return j.enabled }

func (j *Job) Enable(en bool) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.enabled = en
	if !en {
		j.nextRun = time.Time{}
	}
}

func (j *Job) advanceNextRun(now time.Time) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.schedule.Interval <= 0 {
		j.enabled = false
		j.nextRun = time.Time{}
	} else {
		j.nextRun = now.Add(j.schedule.Interval)
	}
}

func (j *Job) markExecuted(start time.Time, err error) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.lastRun = start
	j.lastErr = err
}
