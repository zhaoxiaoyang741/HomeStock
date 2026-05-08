package taskcenter

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/internal/database"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
	gormrepo "github.com/zhaoxiaoyang741/HomeStock/internal/repository/gorm"
	appconfig "github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

func TestTaskCenterService_SyncDefinitionsPreservesOverridesAndRetiresRemovedTasks(t *testing.T) {
	uow, cleanup := newTaskCenterTestUnitOfWork(t)
	defer cleanup()

	svc := NewTaskCenterService(uow,
		TaskDefinition{
			Code:              "alpha_task",
			Name:              "Alpha Task",
			Description:       "first",
			DefaultCronSpec:   "0 8 * * *",
			DefaultEnabled:    true,
			RunTimeoutSeconds: 120,
			Run: func(ctx context.Context, actor Actor) (TaskResult, error) {
				return TaskResult{Summary: "alpha"}, nil
			},
		},
		TaskDefinition{
			Code:              "retired_task",
			Name:              "Retired Task",
			Description:       "to be retired",
			DefaultCronSpec:   "15 9 * * *",
			DefaultEnabled:    true,
			RunTimeoutSeconds: 90,
			Run: func(ctx context.Context, actor Actor) (TaskResult, error) {
				return TaskResult{Summary: "retired"}, nil
			},
		},
	)
	if err := svc.SyncDefinitions(context.Background()); err != nil {
		t.Fatalf("SyncDefinitions() error = %v", err)
	}

	taskRepo := uow.Repos().ScheduledTasks()
	alpha, err := taskRepo.GetByCode("alpha_task")
	if err != nil {
		t.Fatalf("GetByCode(alpha_task) error = %v", err)
	}
	alpha.CronSpec = "45 6 * * *"
	alpha.Enabled = false
	alpha.RunTimeoutSeconds = 321
	if err := taskRepo.Save(alpha); err != nil {
		t.Fatalf("Save(alpha) error = %v", err)
	}

	svc = NewTaskCenterService(uow, TaskDefinition{
		Code:              "alpha_task",
		Name:              "Alpha Task Renamed",
		Description:       "updated",
		DefaultCronSpec:   "0 1 * * *",
		DefaultEnabled:    true,
		RunTimeoutSeconds: 60,
		Run: func(ctx context.Context, actor Actor) (TaskResult, error) {
			return TaskResult{Summary: "alpha"}, nil
		},
	})
	if err := svc.SyncDefinitions(context.Background()); err != nil {
		t.Fatalf("second SyncDefinitions() error = %v", err)
	}

	alpha, err = taskRepo.GetByCode("alpha_task")
	if err != nil {
		t.Fatalf("GetByCode(alpha_task) second error = %v", err)
	}
	if alpha.Name != "Alpha Task Renamed" {
		t.Fatalf("alpha.Name = %q", alpha.Name)
	}
	if alpha.Description != "updated" {
		t.Fatalf("alpha.Description = %q", alpha.Description)
	}
	if alpha.CronSpec != "45 6 * * *" {
		t.Fatalf("alpha.CronSpec = %q", alpha.CronSpec)
	}
	if alpha.Enabled {
		t.Fatal("alpha.Enabled = true, want false")
	}
	if alpha.RunTimeoutSeconds != 321 {
		t.Fatalf("alpha.RunTimeoutSeconds = %d", alpha.RunTimeoutSeconds)
	}
	if !alpha.Registered {
		t.Fatal("alpha.Registered = false")
	}

	retired, err := taskRepo.GetByCode("retired_task")
	if err != nil {
		t.Fatalf("GetByCode(retired_task) error = %v", err)
	}
	if retired.Registered {
		t.Fatal("retired.Registered = true, want false")
	}
	if retired.Enabled {
		t.Fatal("retired.Enabled = true, want false")
	}
	if retired.NextRunAt != nil {
		t.Fatalf("retired.NextRunAt = %v, want nil", retired.NextRunAt)
	}
}

func TestTaskCenterService_TriggerRejectsManualReentryAndRecordsScheduledSkip(t *testing.T) {
	uow, cleanup := newTaskCenterTestUnitOfWork(t)
	defer cleanup()

	startedCh := make(chan struct{}, 1)
	releaseCh := make(chan struct{})

	svc := NewTaskCenterService(uow, TaskDefinition{
		Code:              "slow_task",
		Name:              "Slow Task",
		Description:       "blocks until released",
		DefaultCronSpec:   "*/5 * * * *",
		DefaultEnabled:    true,
		RunTimeoutSeconds: 30,
		Run: func(ctx context.Context, actor Actor) (TaskResult, error) {
			select {
			case startedCh <- struct{}{}:
			default:
			}
			select {
			case <-releaseCh:
				return TaskResult{Summary: "slow finished", Payload: map[string]int{"finished": 1}}, nil
			case <-ctx.Done():
				return TaskResult{Summary: "slow canceled"}, ctx.Err()
			}
		},
	})
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer svc.Stop()

	if _, err := svc.TriggerTask(context.Background(), Actor{TenantID: "default", Channel: "web"}, "slow_task"); err != nil {
		t.Fatalf("TriggerTask() error = %v", err)
	}

	select {
	case <-startedCh:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for slow task to start")
	}

	if _, err := svc.TriggerTask(context.Background(), Actor{TenantID: "default", Channel: "web"}, "slow_task"); !errors.Is(err, repository.ErrScheduledTaskRunning) {
		t.Fatalf("TriggerTask() second error = %v, want ErrScheduledTaskRunning", err)
	}

	svc.handleScheduledTrigger("slow_task")
	if err := waitForTaskRunCount(t, svc, "slow_task", 2, 3*time.Second); err != nil {
		t.Fatal(err)
	}

	close(releaseCh)
	if err := waitForTaskStatus(t, svc, "slow_task", TaskRunStatusSuccess, 3*time.Second); err != nil {
		t.Fatal(err)
	}

	runs, err := svc.ListRuns(context.Background(), repository.ScheduledTaskRunFilter{
		TaskCode: "slow_task",
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("ListRuns() error = %v", err)
	}
	if len(runs.Items) < 2 {
		t.Fatalf("len(runs.Items) = %d, want >= 2", len(runs.Items))
	}

	var foundSkipped bool
	for _, item := range runs.Items {
		if item.Status == TaskRunStatusSkipped && item.TriggerSource == TaskTriggerSourceScheduled {
			foundSkipped = true
		}
	}
	if !foundSkipped {
		t.Fatalf("expected a skipped scheduled run, got %+v", runs.Items)
	}

	task, err := svc.GetTask(context.Background(), "slow_task")
	if err != nil {
		t.Fatalf("GetTask() error = %v", err)
	}
	if task.LastResult != TaskRunStatusSuccess {
		t.Fatalf("task.LastResult = %q", task.LastResult)
	}
}

func TestTaskCenterService_UpdateTaskRecalculatesNextRun(t *testing.T) {
	uow, cleanup := newTaskCenterTestUnitOfWork(t)
	defer cleanup()

	svc := NewTaskCenterService(uow, TaskDefinition{
		Code:              "editable_task",
		Name:              "Editable Task",
		Description:       "task to patch",
		DefaultCronSpec:   "0 8 * * *",
		DefaultEnabled:    true,
		RunTimeoutSeconds: 60,
		Run: func(ctx context.Context, actor Actor) (TaskResult, error) {
			return TaskResult{Summary: "done"}, nil
		},
	})
	if err := svc.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer svc.Stop()

	newCron := "30 23 * * *"
	newTimeout := 180
	enabled := false
	view, err := svc.UpdateTask(context.Background(), Actor{TenantID: "default", Channel: "web"}, "editable_task", UpdateScheduledTaskInput{
		CronSpec:          &newCron,
		Enabled:           &enabled,
		RunTimeoutSeconds: &newTimeout,
	})
	if err != nil {
		t.Fatalf("UpdateTask() error = %v", err)
	}
	if view.CronSpec != newCron {
		t.Fatalf("view.CronSpec = %q", view.CronSpec)
	}
	if view.Enabled {
		t.Fatal("view.Enabled = true, want false")
	}
	if view.RunTimeoutSeconds != newTimeout {
		t.Fatalf("view.RunTimeoutSeconds = %d", view.RunTimeoutSeconds)
	}
	if view.NextRunAt != nil {
		t.Fatalf("view.NextRunAt = %v, want nil for disabled task", view.NextRunAt)
	}

	enabled = true
	view, err = svc.UpdateTask(context.Background(), Actor{TenantID: "default", Channel: "web"}, "editable_task", UpdateScheduledTaskInput{
		Enabled: &enabled,
	})
	if err != nil {
		t.Fatalf("UpdateTask() enable error = %v", err)
	}
	if view.NextRunAt == nil {
		t.Fatal("view.NextRunAt = nil, want recalculated value")
	}
}

func newTaskCenterTestUnitOfWork(t *testing.T) (*gormrepo.UnitOfWork, func()) {
	t.Helper()

	db, err := database.OpenAndMigrate(appconfig.DatabaseConfig{
		Driver: "sqlite",
		DSN:    filepath.Join(t.TempDir(), "task-center.db"),
	})
	if err != nil {
		t.Fatalf("OpenAndMigrate() error = %v", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("db.DB() error = %v", err)
	}

	return gormrepo.NewUnitOfWork(db), func() { _ = sqlDB.Close() }
}

func waitForTaskRunCount(t *testing.T, svc *TaskCenterService, code string, want int, timeout time.Duration) error {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		page, err := svc.ListRuns(context.Background(), repository.ScheduledTaskRunFilter{
			TaskCode: code,
			Page:     1,
			PageSize: 10,
		})
		if err == nil && len(page.Items) >= want {
			return nil
		}
		time.Sleep(25 * time.Millisecond)
	}
	return errors.New("timed out waiting for task run count")
}

func waitForTaskStatus(t *testing.T, svc *TaskCenterService, code, status string, timeout time.Duration) error {
	t.Helper()

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		page, err := svc.ListRuns(context.Background(), repository.ScheduledTaskRunFilter{
			TaskCode: code,
			Page:     1,
			PageSize: 10,
		})
		if err == nil {
			for _, item := range page.Items {
				if item.Status == status {
					return nil
				}
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	return errors.New("timed out waiting for task status")
}
