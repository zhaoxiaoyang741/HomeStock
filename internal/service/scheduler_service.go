package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
)

// SchedulerState represents the current state of the scheduler.
type SchedulerState string

const (
	SchedulerStateIdle    SchedulerState = "idle"
	SchedulerStateRunning SchedulerState = "running"
)

// SchedulerStatus is returned by GetStatus.
type SchedulerStatus struct {
	State      SchedulerState `json:"state"`
	LastRunAt  *time.Time     `json:"last_run_at"`
	NextRunAt  *time.Time     `json:"next_run_at"`
	LastResult string         `json:"last_result"` // "", "success", "failed"
	LastError  string         `json:"last_error,omitempty"`
}

// SchedulerService drives periodic expiry checks using a background goroutine.
// It reads CheckTime from SystemSettings on each iteration, so runtime
// config changes take effect automatically on the next cycle.
type SchedulerService struct {
	notifySvc   *NotificationService
	settingsSvc *SystemSettingsService

	mu     sync.Mutex
	status SchedulerStatus

	stopCh chan struct{}
	trigCh chan chan error // manual trigger: send a reply channel, receive the error
	doneCh chan struct{}  // closed when background goroutine exits
}

func NewSchedulerService(notifySvc *NotificationService, settingsSvc *SystemSettingsService) *SchedulerService {
	return &SchedulerService{
		notifySvc:   notifySvc,
		settingsSvc: settingsSvc,
	}
}

// Start launches the background scheduling goroutine.
// It is idempotent; a second call is a no-op if already running.
func (s *SchedulerService) Start(ctx context.Context) {
	s.mu.Lock()
	if s.stopCh != nil {
		s.mu.Unlock()
		return
	}
	s.stopCh = make(chan struct{})
	s.trigCh = make(chan chan error, 1)
	s.doneCh = make(chan struct{})
	s.mu.Unlock()

	go s.run(ctx)
}

// Stop signals the background goroutine to exit and waits for it to finish.
func (s *SchedulerService) Stop() {
	s.mu.Lock()
	stopCh := s.stopCh
	doneCh := s.doneCh
	s.mu.Unlock()

	if stopCh == nil {
		return
	}
	close(stopCh)
	if doneCh != nil {
		<-doneCh
	}
}

// TriggerNow manually runs the check and waits for it to complete.
func (s *SchedulerService) TriggerNow(ctx context.Context) error {
	s.mu.Lock()
	trigCh := s.trigCh
	s.mu.Unlock()

	if trigCh == nil {
		return fmt.Errorf("scheduler is not running")
	}

	replyCh := make(chan error, 1)
	select {
	case trigCh <- replyCh:
	case <-ctx.Done():
		return ctx.Err()
	}

	select {
	case err := <-replyCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// GetStatus returns a snapshot of the scheduler state.
func (s *SchedulerService) GetStatus() SchedulerStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

// run is the background goroutine body.
func (s *SchedulerService) run(ctx context.Context) {
	defer close(s.doneCh)

	for {
		nextRun := s.calcNextRun(ctx)
		s.setNextRunAt(nextRun)

		timer := time.NewTimer(time.Until(nextRun))

		select {
		case <-timer.C:
			s.execCheck(ctx, nil)

		case replyCh := <-s.trigCh:
			timer.Stop()
			err := s.execCheck(ctx, nil)
			replyCh <- err

		case <-s.stopCh:
			timer.Stop()
			return

		case <-ctx.Done():
			timer.Stop()
			return
		}
	}
}

func (s *SchedulerService) execCheck(ctx context.Context, _ interface{}) error {
	s.mu.Lock()
	s.status.State = SchedulerStateRunning
	s.mu.Unlock()

	err := s.notifySvc.CheckAndNotify(ctx)

	now := time.Now()
	s.mu.Lock()
	s.status.State = SchedulerStateIdle
	s.status.LastRunAt = &now
	if err != nil {
		s.status.LastResult = "failed"
		s.status.LastError = err.Error()
	} else {
		s.status.LastResult = "success"
		s.status.LastError = ""
	}
	s.mu.Unlock()

	return err
}

func (s *SchedulerService) setNextRunAt(t time.Time) {
	s.mu.Lock()
	s.status.NextRunAt = &t
	s.mu.Unlock()
}

// calcNextRun reads CheckTime from SystemSettings and computes the next wall-clock
// firing time. If settings cannot be read, defaults to 08:00.
func (s *SchedulerService) calcNextRun(ctx context.Context) time.Time {
	checkTime := "08:00"
	if settings, err := s.settingsSvc.Get(ctx); err == nil {
		if t := strings.TrimSpace(settings.Reminder.CheckTime); len(t) == 5 {
			checkTime = t
		}
	}

	hour, minute := 8, 0
	fmt.Sscanf(checkTime, "%d:%d", &hour, &minute)

	now := time.Now()
	candidate := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())
	if !candidate.After(now) {
		candidate = candidate.Add(24 * time.Hour)
	}
	return candidate
}
