package taskcenter

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
)

type TaskCenterService struct {
	uow    repository.UnitOfWork
	parser cron.Parser

	mu       sync.RWMutex
	started  bool
	baseCtx  context.Context
	cancel   context.CancelFunc
	cron     *cron.Cron
	defs     map[string]TaskDefinition
	entryIDs map[string]cron.EntryID
	running  map[string]bool
}

func NewTaskCenterService(uow repository.UnitOfWork, defs ...TaskDefinition) *TaskCenterService {
	indexed := make(map[string]TaskDefinition, len(defs))
	for _, def := range defs {
		clean := sanitizeTaskDefinition(def)
		if clean.Code == "" || clean.Run == nil {
			continue
		}
		indexed[clean.Code] = clean
	}

	return &TaskCenterService{
		uow:    uow,
		parser: cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow),
		defs:   indexed,
	}
}

func (s *TaskCenterService) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.started {
		s.mu.Unlock()
		return nil
	}

	baseCtx, cancel := context.WithCancel(ctx)
	engine := cron.New(cron.WithParser(s.parser), cron.WithLocation(time.Local))

	s.started = true
	s.baseCtx = baseCtx
	s.cancel = cancel
	s.cron = engine
	s.entryIDs = make(map[string]cron.EntryID, len(s.defs))
	s.running = make(map[string]bool, len(s.defs))
	s.mu.Unlock()

	if err := s.SyncDefinitions(baseCtx); err != nil {
		s.Stop()
		return err
	}

	tasks, err := s.uow.Repos().ScheduledTasks().List()
	if err != nil {
		s.Stop()
		return err
	}
	for _, task := range tasks {
		if err := s.scheduleTask(baseCtx, task); err != nil {
			s.Stop()
			return err
		}
	}

	engine.Start()
	return nil
}

func (s *TaskCenterService) Stop() {
	s.mu.Lock()
	if !s.started {
		s.mu.Unlock()
		return
	}

	cancel := s.cancel
	engine := s.cron
	s.started = false
	s.cancel = nil
	s.baseCtx = nil
	s.cron = nil
	s.entryIDs = nil
	s.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	if engine != nil {
		stopCtx := engine.Stop()
		select {
		case <-stopCtx.Done():
		case <-time.After(5 * time.Second):
		}
	}
}

func (s *TaskCenterService) SyncDefinitions(ctx context.Context) error {
	if err := s.syncDefinitions(ctx); err != nil {
		return err
	}

	tasks, err := s.uow.Repos().ScheduledTasks().List()
	if err != nil {
		return err
	}
	for _, task := range tasks {
		if err := s.rescheduleTask(ctx, task); err != nil {
			return err
		}
	}
	return nil
}

func (s *TaskCenterService) ListTasks(ctx context.Context) ([]ScheduledTaskView, error) {
	items, err := s.uow.Repos().ScheduledTasks().List()
	if err != nil {
		return nil, err
	}
	result := make([]ScheduledTaskView, 0, len(items))
	for _, item := range items {
		result = append(result, s.buildTaskView(item))
	}
	return result, nil
}

func (s *TaskCenterService) GetTask(ctx context.Context, code string) (*ScheduledTaskView, error) {
	task, err := s.uow.Repos().ScheduledTasks().GetByCode(strings.TrimSpace(code))
	if err != nil {
		return nil, err
	}
	view := s.buildTaskView(task)
	return &view, nil
}

func (s *TaskCenterService) UpdateTask(ctx context.Context, actor Actor, code string, in UpdateScheduledTaskInput) (*ScheduledTaskView, error) {
	code = strings.TrimSpace(code)

	var (
		updated *model.ScheduledTask
		before  *model.ScheduledTask
	)
	err := s.uow.WithTx(ctx, func(r repository.Repos) error {
		task, err := r.ScheduledTasks().GetByCode(code)
		if err != nil {
			return err
		}
		cloned := *task
		before = &cloned

		if in.CronSpec != nil {
			task.CronSpec = strings.TrimSpace(*in.CronSpec)
		}
		if in.Enabled != nil {
			task.Enabled = *in.Enabled
		}
		if in.RunTimeoutSeconds != nil {
			task.RunTimeoutSeconds = *in.RunTimeoutSeconds
		}
		if err := s.validateTaskModel(task); err != nil {
			return err
		}

		task.NextRunAt = s.computeNextRun(task)
		if err := r.ScheduledTasks().Save(task); err != nil {
			return err
		}
		clonedUpdated := *task
		updated = &clonedUpdated

		_ = r.AuditLogs().Create(&model.AuditLog{
			TenantID:      actor.TenantID,
			UserName:      actor.UserName,
			UserID:        actor.UserID,
			Channel:       actor.Channel,
			Action:        "update",
			EntityType:    "scheduled_task",
			EntityID:      task.Code,
			EntityName:    task.Name,
			ChangesDetail: mustJSON(changes{Before: before, After: updated}),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	if err := s.rescheduleTask(ctx, updated); err != nil {
		return nil, err
	}

	view := s.buildTaskView(updated)
	return &view, nil
}

func (s *TaskCenterService) TriggerTask(ctx context.Context, actor Actor, code string) (*model.ScheduledTaskRun, error) {
	run, err := s.startRun(ctx, strings.TrimSpace(code), TaskTriggerSourceManual, actor)
	if err != nil {
		return nil, err
	}

	_ = s.uow.WithTx(ctx, func(r repository.Repos) error {
		task, err := r.ScheduledTasks().GetByCode(code)
		if err != nil {
			return nil
		}
		_ = r.AuditLogs().Create(&model.AuditLog{
			TenantID:      actor.TenantID,
			UserName:      actor.UserName,
			UserID:        actor.UserID,
			Channel:       actor.Channel,
			Action:        "trigger",
			EntityType:    "scheduled_task",
			EntityID:      task.Code,
			EntityName:    task.Name,
			ChangesDetail: mustJSON(changes{After: map[string]any{"run_id": run.ID, "trigger_source": TaskTriggerSourceManual}}),
		})
		return nil
	})

	return run, nil
}

func (s *TaskCenterService) ListRuns(ctx context.Context, filter repository.ScheduledTaskRunFilter) (*repository.ScheduledTaskRunPage, error) {
	return s.uow.Repos().ScheduledTaskRuns().List(filter)
}

func (s *TaskCenterService) GetLegacySchedulerStatus(ctx context.Context) (SchedulerStatus, error) {
	task, err := s.uow.Repos().ScheduledTasks().GetByCode(LegacySchedulerTaskCode)
	if err != nil {
		return SchedulerStatus{}, err
	}

	state := SchedulerStateIdle
	if s.isTaskRunning(task.Code) {
		state = SchedulerStateRunning
	}

	status := SchedulerStatus{
		State:      state,
		LastRunAt:  task.LastRunFinishedAt,
		NextRunAt:  task.NextRunAt,
		LastError:  task.LastError,
		LastResult: "",
	}
	if task.LastResult == TaskRunStatusSuccess || task.LastResult == TaskRunStatusFailed {
		status.LastResult = task.LastResult
	}
	return status, nil
}

func (s *TaskCenterService) TriggerLegacySchedulerTask(ctx context.Context, actor Actor) (*model.ScheduledTaskRun, error) {
	return s.TriggerTask(ctx, actor, LegacySchedulerTaskCode)
}

func (s *TaskCenterService) syncDefinitions(ctx context.Context) error {
	return s.uow.WithTx(ctx, func(r repository.Repos) error {
		repo := r.ScheduledTasks()
		items, err := repo.List()
		if err != nil {
			return err
		}

		byCode := make(map[string]*model.ScheduledTask, len(items))
		for _, item := range items {
			byCode[item.Code] = item
		}

		for code, def := range s.defs {
			task, ok := byCode[code]
			if !ok {
				task = &model.ScheduledTask{
					Code:              def.Code,
					Name:              def.Name,
					Description:       def.Description,
					CronSpec:          def.DefaultCronSpec,
					Enabled:           def.DefaultEnabled,
					Registered:        true,
					RunTimeoutSeconds: def.RunTimeoutSeconds,
				}
				task.NextRunAt = s.computeNextRun(task)
				if err := repo.Create(task); err != nil {
					return err
				}
				continue
			}

			task.Name = def.Name
			task.Description = def.Description
			task.Registered = true
			if strings.TrimSpace(task.CronSpec) == "" {
				task.CronSpec = def.DefaultCronSpec
			}
			if task.RunTimeoutSeconds <= 0 {
				task.RunTimeoutSeconds = def.RunTimeoutSeconds
			}
			task.NextRunAt = s.computeNextRun(task)
			if err := s.validateTaskModel(task); err != nil {
				return err
			}
			if err := repo.Save(task); err != nil {
				return err
			}
		}

		for _, item := range items {
			if _, ok := s.defs[item.Code]; ok {
				continue
			}
			item.Registered = false
			item.Enabled = false
			item.NextRunAt = nil
			if err := repo.Save(item); err != nil {
				return err
			}
		}

		return nil
	})
}

func (s *TaskCenterService) scheduleTask(ctx context.Context, task *model.ScheduledTask) error {
	if task == nil || !task.Registered || !task.Enabled {
		return s.persistTaskNextRun(ctx, task)
	}
	if _, ok := s.defs[task.Code]; !ok {
		return nil
	}
	engine := s.currentCron()
	if engine == nil {
		return s.persistTaskNextRun(ctx, task)
	}

	schedule, err := s.parser.Parse(strings.TrimSpace(task.CronSpec))
	if err != nil {
		return fmt.Errorf("%w: %s", repository.ErrScheduledTaskInvalidCron, err.Error())
	}

	entryID := engine.Schedule(schedule, cron.FuncJob(func() {
		s.handleScheduledTrigger(task.Code)
	}))

	s.mu.Lock()
	if s.entryIDs != nil {
		s.entryIDs[task.Code] = entryID
	}
	s.mu.Unlock()

	return s.persistTaskNextRun(ctx, task)
}

func (s *TaskCenterService) unscheduleTask(code string) {
	engine := s.currentCron()
	if engine == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.entryIDs == nil {
		return
	}
	if entryID, ok := s.entryIDs[code]; ok {
		engine.Remove(entryID)
		delete(s.entryIDs, code)
	}
}

func (s *TaskCenterService) rescheduleTask(ctx context.Context, task *model.ScheduledTask) error {
	if task == nil {
		return nil
	}
	s.unscheduleTask(task.Code)
	if err := s.persistTaskNextRun(ctx, task); err != nil {
		return err
	}
	if task.Registered && task.Enabled {
		return s.scheduleTask(ctx, task)
	}
	return nil
}

func (s *TaskCenterService) persistTaskNextRun(ctx context.Context, task *model.ScheduledTask) error {
	if task == nil {
		return nil
	}
	cloned := *task
	cloned.NextRunAt = s.computeNextRun(&cloned)
	return s.uow.WithTx(ctx, func(r repository.Repos) error {
		current, err := r.ScheduledTasks().GetByCode(cloned.Code)
		if err != nil {
			return err
		}
		current.NextRunAt = cloned.NextRunAt
		return r.ScheduledTasks().Save(current)
	})
}

func (s *TaskCenterService) computeNextRun(task *model.ScheduledTask) *time.Time {
	if task == nil || !task.Registered || !task.Enabled {
		return nil
	}
	spec := strings.TrimSpace(task.CronSpec)
	if spec == "" {
		return nil
	}
	schedule, err := s.parser.Parse(spec)
	if err != nil {
		return nil
	}
	next := schedule.Next(time.Now().In(time.Local))
	return &next
}

func (s *TaskCenterService) validateTaskModel(task *model.ScheduledTask) error {
	if task == nil {
		return gorm.ErrRecordNotFound
	}
	if strings.TrimSpace(task.CronSpec) == "" {
		return fmt.Errorf("%w: cron_spec is required", repository.ErrScheduledTaskInvalidCron)
	}
	if _, err := s.parser.Parse(strings.TrimSpace(task.CronSpec)); err != nil {
		return fmt.Errorf("%w: %s", repository.ErrScheduledTaskInvalidCron, err.Error())
	}
	if task.RunTimeoutSeconds <= 0 {
		return errors.New("run_timeout_seconds must be greater than 0")
	}
	return nil
}

func (s *TaskCenterService) buildTaskView(task *model.ScheduledTask) ScheduledTaskView {
	state := TaskStateIdle
	if task != nil && s.isTaskRunning(task.Code) {
		state = TaskStateRunning
	}
	if task == nil {
		return ScheduledTaskView{State: state}
	}
	return ScheduledTaskView{
		ID:                task.ID,
		Code:              task.Code,
		Name:              task.Name,
		Description:       task.Description,
		CronSpec:          task.CronSpec,
		Enabled:           task.Enabled,
		Registered:        task.Registered,
		RunTimeoutSeconds: task.RunTimeoutSeconds,
		State:             state,
		NextRunAt:         task.NextRunAt,
		LastRunStartedAt:  task.LastRunStartedAt,
		LastRunFinishedAt: task.LastRunFinishedAt,
		LastResult:        task.LastResult,
		LastError:         task.LastError,
		CreatedAt:         task.CreatedAt,
		UpdatedAt:         task.UpdatedAt,
	}
}

func (s *TaskCenterService) startRun(ctx context.Context, code, triggerSource string, actor Actor) (*model.ScheduledTaskRun, error) {
	task, err := s.uow.Repos().ScheduledTasks().GetByCode(code)
	if err != nil {
		return nil, err
	}
	def, ok := s.defs[task.Code]
	if !ok || !task.Registered {
		return nil, gorm.ErrRecordNotFound
	}

	startedAt := time.Now()
	if !s.markTaskRunning(task.Code) {
		if triggerSource == TaskTriggerSourceManual {
			return nil, repository.ErrScheduledTaskRunning
		}
		return s.recordSkippedRun(ctx, task, triggerSource, actor, startedAt)
	}

	run := &model.ScheduledTaskRun{
		TaskCode:            task.Code,
		TaskName:            task.Name,
		TriggerSource:       triggerSource,
		Status:              TaskRunStatusRunning,
		StartedAt:           startedAt,
		TriggeredByUserName: actor.UserName,
		TriggeredByUserID:   actor.UserID,
		TriggeredByChannel:  defaultTaskActorChannel(actor.Channel),
	}

	err = s.uow.WithTx(ctx, func(r repository.Repos) error {
		current, err := r.ScheduledTasks().GetByCode(task.Code)
		if err != nil {
			return err
		}
		current.LastRunStartedAt = &startedAt
		if err := r.ScheduledTaskRuns().Create(run); err != nil {
			return err
		}
		return r.ScheduledTasks().Save(current)
	})
	if err != nil {
		s.markTaskStopped(task.Code)
		return nil, err
	}

	go s.executeRun(task, def, run, actor)
	return run, nil
}

func (s *TaskCenterService) executeRun(task *model.ScheduledTask, def TaskDefinition, run *model.ScheduledTaskRun, actor Actor) {
	defer s.markTaskStopped(task.Code)

	baseCtx := s.currentBaseContext()
	if baseCtx == nil {
		baseCtx = context.Background()
	}

	timeout := time.Duration(task.RunTimeoutSeconds) * time.Second
	runCtx, cancel := context.WithTimeout(baseCtx, timeout)
	defer cancel()

	result, execErr := def.Run(runCtx, actor)
	finishedAt := time.Now()
	run.FinishedAt = &finishedAt
	run.DurationMs = finishedAt.Sub(run.StartedAt).Milliseconds()
	run.Summary = strings.TrimSpace(result.Summary)
	run.ResultPayload = mustJSON(result.Payload)
	run.ErrorMessage = ""
	run.Status = TaskRunStatusSuccess
	if execErr != nil {
		run.Status = TaskRunStatusFailed
		run.ErrorMessage = execErr.Error()
	}

	_ = s.uow.WithTx(context.Background(), func(r repository.Repos) error {
		current, err := r.ScheduledTasks().GetByCode(task.Code)
		if err != nil {
			return err
		}
		current.LastRunFinishedAt = &finishedAt
		current.LastResult = run.Status
		current.LastError = run.ErrorMessage
		if err := r.ScheduledTaskRuns().Save(run); err != nil {
			return err
		}
		return r.ScheduledTasks().Save(current)
	})
}

func (s *TaskCenterService) handleScheduledTrigger(code string) {
	ctx := s.currentBaseContext()
	if ctx == nil {
		ctx = context.Background()
	}

	task, err := s.uow.Repos().ScheduledTasks().GetByCode(code)
	if err != nil {
		return
	}
	_ = s.persistTaskNextRun(ctx, task)

	_, _ = s.startRun(ctx, code, TaskTriggerSourceScheduled, Actor{
		TenantID: "default",
		Channel:  "system",
	})
}

func (s *TaskCenterService) recordSkippedRun(ctx context.Context, task *model.ScheduledTask, triggerSource string, actor Actor, startedAt time.Time) (*model.ScheduledTaskRun, error) {
	finishedAt := startedAt
	run := &model.ScheduledTaskRun{
		TaskCode:            task.Code,
		TaskName:            task.Name,
		TriggerSource:       triggerSource,
		Status:              TaskRunStatusSkipped,
		Summary:             "task skipped because a previous run is still in progress",
		ErrorMessage:        repository.ErrScheduledTaskRunning.Error(),
		StartedAt:           startedAt,
		FinishedAt:          &finishedAt,
		TriggeredByUserName: actor.UserName,
		TriggeredByUserID:   actor.UserID,
		TriggeredByChannel:  defaultTaskActorChannel(actor.Channel),
	}
	if err := s.uow.Repos().ScheduledTaskRuns().Create(run); err != nil {
		return nil, err
	}
	return run, nil
}

func (s *TaskCenterService) markTaskRunning(code string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running == nil {
		s.running = map[string]bool{}
	}
	if s.running[code] {
		return false
	}
	s.running[code] = true
	return true
}

func (s *TaskCenterService) markTaskStopped(code string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running != nil {
		delete(s.running, code)
	}
}

func (s *TaskCenterService) isTaskRunning(code string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running != nil && s.running[code]
}

func (s *TaskCenterService) currentCron() *cron.Cron {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cron
}

func (s *TaskCenterService) currentBaseContext() context.Context {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.baseCtx
}

func defaultTaskActorChannel(channel string) string {
	channel = strings.TrimSpace(channel)
	if channel == "" {
		return "system"
	}
	return channel
}
