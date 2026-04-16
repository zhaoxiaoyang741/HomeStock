package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

const globalSystemSettingsKey = "global"

var checkTimePattern = regexp.MustCompile(`^\d{2}:\d{2}$`)

type SystemSettingsView struct {
	Version   int                        `json:"version"`
	Reminder  SystemSettingsReminderView `json:"reminder"`
	Notify    SystemSettingsNotifyView   `json:"notify"`
	UpdatedAt string                     `json:"updated_at"`
	UpdatedBy SystemSettingsUpdatedBy    `json:"updated_by"`
}

type SystemSettingsReminderView struct {
	RemindDays int    `json:"remind_days"`
	CheckTime  string `json:"check_time"`
}

type SystemSettingsNotifyView struct {
	Enabled                 bool   `json:"enabled"`
	FeishuWebhookConfigured bool   `json:"feishu_webhook_configured"`
	FeishuWebhookMasked     string `json:"feishu_webhook_masked"`
}

type SystemSettingsUpdatedBy struct {
	UserName string `json:"user_name"`
	UserID   string `json:"user_id"`
	Channel  string `json:"channel"`
}

type UpdateSystemSettingsInput struct {
	Version  int
	Reminder UpdateSystemSettingsReminderInput
	Notify   UpdateSystemSettingsNotifyInput
}

type UpdateSystemSettingsReminderInput struct {
	RemindDays int
	CheckTime  string
}

type UpdateSystemSettingsNotifyInput struct {
	Enabled           bool
	FeishuWebhookMode string
	FeishuWebhook     string
}

type storedSystemSettings struct {
	Reminder storedSystemSettingsReminder `json:"reminder"`
	Notify   storedSystemSettingsNotify   `json:"notify"`
}

type storedSystemSettingsReminder struct {
	RemindDays int    `json:"remind_days"`
	CheckTime  string `json:"check_time"`
}

type storedSystemSettingsNotify struct {
	Enabled       bool   `json:"enabled"`
	FeishuWebhook string `json:"feishu_webhook"`
}

type SystemSettingsValidationError struct {
	Message string
}

func (e *SystemSettingsValidationError) Error() string { return e.Message }

type SystemSettingsService struct {
	uow      repository.UnitOfWork
	defaults *config.Config

	mu        sync.RWMutex
	cached    *SystemSettingsView
	expiresAt time.Time
	cacheTTL  time.Duration
}

func NewSystemSettingsService(uow repository.UnitOfWork, defaults *config.Config) *SystemSettingsService {
	return &SystemSettingsService{
		uow:      uow,
		defaults: defaults,
		cacheTTL: 30 * time.Second,
	}
}

func (s *SystemSettingsService) Get(ctx context.Context) (*SystemSettingsView, error) {
	if cached := s.cachedView(); cached != nil {
		return cached, nil
	}

	setting, err := s.uow.Repos().SystemSettings().GetByKey(globalSystemSettingsKey)
	if err != nil {
		if repository.IsNotFound(err) {
			view := s.buildView(nil, nil)
			s.storeCache(view)
			return cloneSystemSettingsView(view), nil
		}
		return nil, err
	}

	stored, err := decodeStoredSystemSettings(setting.Payload)
	if err != nil {
		return nil, err
	}
	view := s.buildView(setting, stored)
	s.storeCache(view)
	return cloneSystemSettingsView(view), nil
}

func (s *SystemSettingsService) Update(ctx context.Context, actor Actor, in UpdateSystemSettingsInput) (*SystemSettingsView, error) {
	if err := validateUpdateSystemSettingsInput(in); err != nil {
		return nil, err
	}

	var updatedView *SystemSettingsView
	err := s.uow.WithTx(ctx, func(r repository.Repos) error {
		repo := r.SystemSettings()
		setting, err := repo.GetByKey(globalSystemSettingsKey)
		if err != nil && !repository.IsNotFound(err) {
			return err
		}

		var (
			currentStored *storedSystemSettings
			currentView   *SystemSettingsView
		)

		if repository.IsNotFound(err) {
			if in.Version != 0 {
				return repository.ErrSystemSettingsVersionConflict
			}
			currentStored = buildDefaultStoredSystemSettings(s.defaults)
			currentView = s.buildView(nil, currentStored)
		} else {
			currentStored, err = decodeStoredSystemSettings(setting.Payload)
			if err != nil {
				return err
			}
			if in.Version != setting.Version {
				return repository.ErrSystemSettingsVersionConflict
			}
			currentView = s.buildView(setting, currentStored)
		}

		nextStored := applySystemSettingsUpdate(currentStored, in)
		if err := validateEffectiveStoredSystemSettings(nextStored); err != nil {
			return err
		}

		payloadBytes, err := json.Marshal(nextStored)
		if err != nil {
			return fmt.Errorf("marshal system settings: %w", err)
		}

		nextVersion := 1
		record := &model.SystemSetting{
			Key:               globalSystemSettingsKey,
			Payload:           string(payloadBytes),
			Version:           nextVersion,
			UpdatedByUserName: actor.UserName,
			UpdatedByUserID:   actor.UserID,
			UpdatedByChannel:  actor.Channel,
		}
		if setting != nil {
			record.ID = setting.ID
			record.CreatedAt = setting.CreatedAt
			nextVersion = setting.Version + 1
			record.Version = nextVersion
		}
		if err := repo.Upsert(record); err != nil {
			return err
		}

		saved, err := repo.GetByKey(globalSystemSettingsKey)
		if err != nil {
			return err
		}
		savedStored, err := decodeStoredSystemSettings(saved.Payload)
		if err != nil {
			return err
		}
		updatedView = s.buildView(saved, savedStored)

		_ = r.AuditLogs().Create(&model.AuditLog{
			TenantID:      actor.TenantID,
			UserName:      actor.UserName,
			UserID:        actor.UserID,
			Channel:       actor.Channel,
			Action:        "update",
			EntityType:    "system_setting",
			EntityID:      globalSystemSettingsKey,
			EntityName:    "system settings",
			ChangesDetail: mustJSON(changes{Before: currentView, After: updatedView}),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.storeCache(updatedView)
	return cloneSystemSettingsView(updatedView), nil
}

func (s *SystemSettingsService) cachedView() *SystemSettingsView {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.cached == nil || time.Now().After(s.expiresAt) {
		return nil
	}
	return cloneSystemSettingsView(s.cached)
}

func (s *SystemSettingsService) storeCache(view *SystemSettingsView) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cached = cloneSystemSettingsView(view)
	s.expiresAt = time.Now().Add(s.cacheTTL)
}

func (s *SystemSettingsService) buildView(setting *model.SystemSetting, stored *storedSystemSettings) *SystemSettingsView {
	effective := buildDefaultStoredSystemSettings(s.defaults)
	if stored != nil {
		effective.Reminder = stored.Reminder
		effective.Notify = stored.Notify
	}

	view := &SystemSettingsView{
		Version: 0,
		Reminder: SystemSettingsReminderView{
			RemindDays: effective.Reminder.RemindDays,
			CheckTime:  effective.Reminder.CheckTime,
		},
		Notify: SystemSettingsNotifyView{
			Enabled:                 effective.Notify.Enabled,
			FeishuWebhookConfigured: strings.TrimSpace(effective.Notify.FeishuWebhook) != "",
			FeishuWebhookMasked:     maskWebhook(effective.Notify.FeishuWebhook),
		},
		UpdatedBy: SystemSettingsUpdatedBy{},
	}
	if setting != nil {
		view.Version = setting.Version
		view.UpdatedAt = setting.UpdatedAt.Format(time.RFC3339)
		view.UpdatedBy = SystemSettingsUpdatedBy{
			UserName: setting.UpdatedByUserName,
			UserID:   setting.UpdatedByUserID,
			Channel:  setting.UpdatedByChannel,
		}
	}
	return view
}

func buildDefaultStoredSystemSettings(defaults *config.Config) *storedSystemSettings {
	cfg := defaults
	if cfg == nil {
		cfg = config.Get()
	}
	webhook := strings.TrimSpace(cfg.Notify.FeishuWebhook)
	return &storedSystemSettings{
		Reminder: storedSystemSettingsReminder{
			RemindDays: cfg.Scheduler.RemindDays,
			CheckTime:  strings.TrimSpace(cfg.Scheduler.CheckTime),
		},
		Notify: storedSystemSettingsNotify{
			Enabled:       webhook != "",
			FeishuWebhook: webhook,
		},
	}
}

func decodeStoredSystemSettings(payload string) (*storedSystemSettings, error) {
	stored := &storedSystemSettings{}
	if strings.TrimSpace(payload) == "" {
		return stored, nil
	}
	if err := json.Unmarshal([]byte(payload), stored); err != nil {
		return nil, fmt.Errorf("parse system settings payload: %w", err)
	}
	return stored, nil
}

func applySystemSettingsUpdate(current *storedSystemSettings, in UpdateSystemSettingsInput) *storedSystemSettings {
	next := *current
	next.Reminder.RemindDays = in.Reminder.RemindDays
	next.Reminder.CheckTime = strings.TrimSpace(in.Reminder.CheckTime)
	next.Notify.Enabled = in.Notify.Enabled
	switch strings.TrimSpace(in.Notify.FeishuWebhookMode) {
	case "replace":
		next.Notify.FeishuWebhook = strings.TrimSpace(in.Notify.FeishuWebhook)
	case "clear":
		next.Notify.FeishuWebhook = ""
	case "keep":
	default:
	}
	return &next
}

func validateUpdateSystemSettingsInput(in UpdateSystemSettingsInput) error {
	if in.Reminder.RemindDays < 0 || in.Reminder.RemindDays > 30 {
		return &SystemSettingsValidationError{Message: "remind_days must be between 0 and 30"}
	}
	if !checkTimePattern.MatchString(strings.TrimSpace(in.Reminder.CheckTime)) {
		return &SystemSettingsValidationError{Message: "check_time must be in HH:MM format"}
	}
	if !isValidCheckTime(strings.TrimSpace(in.Reminder.CheckTime)) {
		return &SystemSettingsValidationError{Message: "check_time must be in HH:MM format"}
	}
	mode := strings.TrimSpace(in.Notify.FeishuWebhookMode)
	switch mode {
	case "keep", "replace", "clear":
	default:
		return &SystemSettingsValidationError{Message: "feishu_webhook_mode must be one of keep, replace, or clear"}
	}
	if mode == "replace" {
		webhook := strings.TrimSpace(in.Notify.FeishuWebhook)
		if webhook == "" {
			return &SystemSettingsValidationError{Message: "feishu_webhook is required when feishu_webhook_mode is replace"}
		}
		parsed, err := url.ParseRequestURI(webhook)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return &SystemSettingsValidationError{Message: "feishu_webhook must be a valid URL"}
		}
	}
	return nil
}

func validateEffectiveStoredSystemSettings(stored *storedSystemSettings) error {
	if stored.Notify.Enabled && strings.TrimSpace(stored.Notify.FeishuWebhook) == "" {
		return &SystemSettingsValidationError{Message: "feishu_webhook is required when notifications are enabled"}
	}
	return nil
}

func isValidCheckTime(v string) bool {
	if len(v) != 5 {
		return false
	}
	hour := (int(v[0]-'0') * 10) + int(v[1]-'0')
	minute := (int(v[3]-'0') * 10) + int(v[4]-'0')
	return hour >= 0 && hour <= 23 && minute >= 0 && minute <= 59
}

func cloneSystemSettingsView(view *SystemSettingsView) *SystemSettingsView {
	if view == nil {
		return nil
	}
	cloned := *view
	return &cloned
}

func maskWebhook(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}
	if len(trimmed) <= 12 {
		return trimmed[:1] + "***" + trimmed[len(trimmed)-1:]
	}
	return trimmed[:8] + "***" + trimmed[len(trimmed)-4:]
}

// GetFeishuWebhook returns the raw (unmasked) Feishu webhook URL, or empty string if not set.
func (s *SystemSettingsService) GetFeishuWebhook(ctx context.Context) (string, error) {
	setting, err := s.uow.Repos().SystemSettings().GetByKey(globalSystemSettingsKey)
	if err != nil {
		if repository.IsNotFound(err) {
			return strings.TrimSpace(s.defaults.Notify.FeishuWebhook), nil
		}
		return "", err
	}
	stored, err := decodeStoredSystemSettings(setting.Payload)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(stored.Notify.FeishuWebhook), nil
}

func IsSystemSettingsValidationError(err error) bool {
	var target *SystemSettingsValidationError
	return errors.As(err, &target)
}
