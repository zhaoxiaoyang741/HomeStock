package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/model"
	"github.com/zhaoxiaoyang741/HomeStock/internal/repository"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
)

func TestSystemSettingsService_GetCachesAndClones(t *testing.T) {
	repo := &fakeSystemSettingsRepo{
		setting: &model.SystemSetting{
			ID:      "sys123",
			Key:     globalSystemSettingsKey,
			Version: 2,
			Payload: mustMarshalSystemSettingsPayload(t, storedSystemSettings{
				Reminder: storedSystemSettingsReminder{RemindDays: 5, CheckTime: "09:30"},
				Notify:   storedSystemSettingsNotify{Enabled: true, FeishuWebhook: "https://open.feishu.cn/hook/example1234"},
			}),
			UpdatedByUserName: "Alice",
			UpdatedByChannel:  "web",
			UpdatedAt:         time.Date(2026, 4, 13, 3, 4, 5, 0, time.UTC),
		},
	}
	svc := NewSystemSettingsService(&fakeUnitOfWork{repos: &fakeRepos{systemSettings: repo, auditLogs: &fakeAuditLogRepo{}}}, &config.Config{})
	svc.cacheTTL = time.Minute

	first, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	first.Reminder.RemindDays = 99

	second, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get() second error = %v", err)
	}
	if repo.getCalls != 1 {
		t.Fatalf("getCalls = %d, want 1", repo.getCalls)
	}
	if second.Reminder.RemindDays != 5 {
		t.Fatalf("RemindDays = %d", second.Reminder.RemindDays)
	}
	if !second.Notify.FeishuWebhookConfigured {
		t.Fatalf("FeishuWebhookConfigured = false")
	}
	if second.Notify.FeishuWebhookMasked == "" || second.Notify.FeishuWebhookMasked == "https://open.feishu.cn/hook/example1234" {
		t.Fatalf("FeishuWebhookMasked = %q", second.Notify.FeishuWebhookMasked)
	}
}

func TestSystemSettingsService_UpdateRefreshesCache(t *testing.T) {
	repo := &fakeSystemSettingsRepo{
		setting: &model.SystemSetting{
			ID:      "sys123",
			Key:     globalSystemSettingsKey,
			Version: 1,
			Payload: mustMarshalSystemSettingsPayload(t, storedSystemSettings{
				Reminder: storedSystemSettingsReminder{RemindDays: 3, CheckTime: "08:00"},
				Notify:   storedSystemSettingsNotify{Enabled: true, FeishuWebhook: "https://open.feishu.cn/hook/original1234"},
			}),
			UpdatedAt: time.Date(2026, 4, 13, 3, 4, 5, 0, time.UTC),
		},
	}
	svc := NewSystemSettingsService(&fakeUnitOfWork{repos: &fakeRepos{systemSettings: repo, auditLogs: &fakeAuditLogRepo{}}}, &config.Config{})
	svc.cacheTTL = time.Hour

	if _, err := svc.Get(context.Background()); err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	beforeGetCalls := repo.getCalls

	updated, err := svc.Update(context.Background(), Actor{TenantID: "default", UserName: "Bob", Channel: "web"}, UpdateSystemSettingsInput{
		Version: 1,
		Reminder: UpdateSystemSettingsReminderInput{
			RemindDays: 7,
			CheckTime:  "10:15",
		},
		Notify: UpdateSystemSettingsNotifyInput{
			Enabled:           true,
			FeishuWebhookMode: "replace",
			FeishuWebhook:     "https://open.feishu.cn/hook/updated1234",
		},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Version != 2 {
		t.Fatalf("Version = %d", updated.Version)
	}
	if repo.getCalls <= beforeGetCalls {
		t.Fatalf("expected update to read persisted settings, getCalls = %d before = %d", repo.getCalls, beforeGetCalls)
	}

	afterUpdateGetCalls := repo.getCalls
	view, err := svc.Get(context.Background())
	if err != nil {
		t.Fatalf("Get() after update error = %v", err)
	}
	if repo.getCalls != afterUpdateGetCalls {
		t.Fatalf("getCalls changed after cached Get: before=%d after=%d", afterUpdateGetCalls, repo.getCalls)
	}
	if view.Reminder.CheckTime != "10:15" {
		t.Fatalf("CheckTime = %q", view.Reminder.CheckTime)
	}
}

func mustMarshalSystemSettingsPayload(t *testing.T, payload storedSystemSettings) string {
	t.Helper()
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	return string(raw)
}

type fakeUnitOfWork struct {
	repos repository.Repos
}

func (u *fakeUnitOfWork) Repos() repository.Repos { return u.repos }

func (u *fakeUnitOfWork) WithTx(_ context.Context, fn func(r repository.Repos) error) error {
	return fn(u.repos)
}

type fakeRepos struct {
	systemSettings repository.SystemSettingRepo
	auditLogs      repository.AuditLogRepo
}

func (r *fakeRepos) Categories() repository.CategoryRepo          { panic("unused") }
func (r *fakeRepos) Materials() repository.MaterialRepo           { panic("unused") }
func (r *fakeRepos) StockLots() repository.StockLotRepo           { panic("unused") }
func (r *fakeRepos) StockMovements() repository.StockMovementRepo { panic("unused") }
func (r *fakeRepos) AuditLogs() repository.AuditLogRepo           { return r.auditLogs }
func (r *fakeRepos) SystemSettings() repository.SystemSettingRepo { return r.systemSettings }

type fakeSystemSettingsRepo struct {
	setting  *model.SystemSetting
	getCalls int
}

func (r *fakeSystemSettingsRepo) GetByKey(key string) (*model.SystemSetting, error) {
	r.getCalls++
	if r.setting == nil || r.setting.Key != key {
		return nil, gorm.ErrRecordNotFound
	}
	cloned := *r.setting
	return &cloned, nil
}

func (r *fakeSystemSettingsRepo) Upsert(setting *model.SystemSetting) error {
	cloned := *setting
	if cloned.ID == "" {
		cloned.ID = "generated-id"
	}
	if cloned.CreatedAt.IsZero() {
		cloned.CreatedAt = time.Date(2026, 4, 13, 0, 0, 0, 0, time.UTC)
	}
	cloned.UpdatedAt = time.Date(2026, 4, 13, 4, 5, 6, 0, time.UTC)
	r.setting = &cloned
	return nil
}

type fakeAuditLogRepo struct {
	logs []model.AuditLog
}

func (r *fakeAuditLogRepo) Create(log *model.AuditLog) error {
	if log != nil {
		r.logs = append(r.logs, *log)
	}
	return nil
}

func (r *fakeAuditLogRepo) List(f repository.AuditLogFilter) (*repository.AuditLogPage, error) {
	return &repository.AuditLogPage{Logs: r.logs, Total: int64(len(r.logs)), Page: f.Page, PageSize: f.PageSize}, nil
}
