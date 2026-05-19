package cron

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// Service manages a collection of scheduled tasks.
type Service struct {
	mu       sync.RWMutex
	jobs     []*Job
	running  bool
	stopChan chan struct{}
	wakeChan chan struct{}
	wg       sync.WaitGroup
}

// New creates a new scheduler service.
func New() *Service {
	return &Service{
		wakeChan: make(chan struct{}, 1),
	}
}

// Register adds a task with its schedule to the service.
// Must be called before Start(). Panics on duplicate name.
func (s *Service) Register(task Task, schedule ScheduleDef) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, j := range s.jobs {
		if j.Name() == task.Name() {
			panic(fmt.Sprintf("cron: duplicate task %q", task.Name()))
		}
	}
	s.jobs = append(s.jobs, newJob(task, schedule))
}

// Start launches the scheduler goroutine.
func (s *Service) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return
	}
	s.running = true
	s.stopChan = make(chan struct{})
	s.wg.Add(1)
	go s.runLoop()
}

// Stop gracefully stops the scheduler goroutine and waits for it to exit.
func (s *Service) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	close(s.stopChan)
	s.mu.Unlock()
	s.wg.Wait()
}

// JobStatus is a snapshot of a job for observability.
type JobStatus struct {
	Name     string    `json:"name"`
	Enabled  bool      `json:"enabled"`
	NextRun  time.Time `json:"next_run"`
	LastRun  time.Time `json:"last_run"`
	LastErr  string    `json:"last_error,omitempty"`
	Interval string    `json:"interval"`
}

// Jobs returns a snapshot of all registered jobs.
func (s *Service) Jobs() []JobStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()
	res := make([]JobStatus, 0, len(s.jobs))
	for _, j := range s.jobs {
		st := JobStatus{
			Name:    j.Name(),
			Enabled: j.Enabled(),
			NextRun: j.NextRun(),
			LastRun: j.LastRun(),
			Interval: func() string {
				if j.schedule.Interval <= 0 {
					return "once"
				}
				return j.schedule.Interval.String()
			}(),
		}
		if err := j.LastError(); err != nil {
			st.LastErr = err.Error()
		}
		res = append(res, st)
	}
	return res
}

// --- internal ---

func (s *Service) runLoop() {
	defer s.wg.Done()

	// Initialize nextRun for all enabled jobs
	s.mu.Lock()
	now := time.Now()
	for _, j := range s.jobs {
		if j.Enabled() {
			j.advanceNextRun(now)
		}
	}
	s.mu.Unlock()

	timer := time.NewTimer(0)
	if !timer.Stop() {
		<-timer.C
	}
	defer timer.Stop()

	for {
		if !s.waitForWake(timer) {
			return
		}
		s.tick()
	}
}

// waitForWake sleeps until the next job is due, or until woken externally.
// Returns false if the service is stopped.
func (s *Service) waitForWake(timer *time.Timer) bool {
	s.mu.RLock()
	nextWake := s.nextWakeTime()
	s.mu.RUnlock()

	now := time.Now()
	var delay time.Duration
	if nextWake.IsZero() {
		delay = time.Hour // no jobs scheduled, sleep long
	} else if nextWake.Before(now) || nextWake.Equal(now) {
		delay = 0 // already due
	} else {
		delay = nextWake.Sub(now)
	}

	timer.Reset(delay)

	select {
	case <-s.stopChan:
		return false
	case <-s.wakeChan:
		if !timer.Stop() {
			select {
			case <-timer.C:
			default:
			}
		}
		return true
	case <-timer.C:
		return true
	}
}

// tick finds due jobs and executes them.
func (s *Service) tick() {
	s.mu.Lock()
	now := time.Now()
	var due []*Job
	for _, j := range s.jobs {
		if j.Enabled() {
			nt := j.NextRun()
			if !nt.IsZero() && (nt.Before(now) || nt.Equal(now)) {
				due = append(due, j)
				j.advanceNextRun(now)
			}
		}
	}
	s.mu.Unlock()

	for _, j := range due {
		s.executeJob(j)
	}
}

func (s *Service) executeJob(j *Job) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	start := time.Now()

	err := s.runWithRecover(ctx, j)

	j.markExecuted(start, err)

	dur := time.Since(start)
	if err != nil {
		log.Error().Err(err).Str("task", j.Name()).Dur("duration", dur).Msg("cron: task failed")
	} else {
		log.Info().Str("task", j.Name()).Dur("duration", dur).Msg("cron: task completed")
	}
}

func (s *Service) runWithRecover(ctx context.Context, j *Job) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("cron: task %q panicked: %v\n%s", j.Name(), r, debug.Stack())
		}
	}()
	return j.task.Run(ctx)
}

// notify wakes the runLoop to re-evaluate immediately.
// It is safe to call from any goroutine.
func (s *Service) notify() {
	select {
	case s.wakeChan <- struct{}{}:
	default:
	}
}

// nextWakeTime returns the earliest nextRun across all enabled jobs.
func (s *Service) nextWakeTime() time.Time {
	var earliest time.Time
	for _, j := range s.jobs {
		if j.Enabled() {
			nt := j.NextRun()
			if !nt.IsZero() && (earliest.IsZero() || nt.Before(earliest)) {
				earliest = nt
			}
		}
	}
	return earliest
}
