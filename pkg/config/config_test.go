package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_defaultsWithoutFileOrEnv(t *testing.T) {
	resetCurrent()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	assertDefaultConfig(t, cfg)
	assertDefaultConfig(t, Get())
}

func TestLoad_readsJSONFile(t *testing.T) {
	resetCurrent()

	path := writeConfigFile(t, `{
		"server": {"port": "9090"},
		"database": {"driver": "postgres", "dsn": "postgres://db"},
		"notify": {"feishu_webhook": "https://example.test/hook"},
		"scheduler": {"remind_days": 5, "check_time": "09:30"}
	}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server.Port != "9090" {
		t.Fatalf("Server.Port = %q", cfg.Server.Port)
	}
	if cfg.Database.Driver != "postgres" {
		t.Fatalf("Database.Driver = %q", cfg.Database.Driver)
	}
	if cfg.Database.DSN != "postgres://db" {
		t.Fatalf("Database.DSN = %q", cfg.Database.DSN)
	}
	if cfg.Notify.FeishuWebhook != "https://example.test/hook" {
		t.Fatalf("Notify.FeishuWebhook = %q", cfg.Notify.FeishuWebhook)
	}
	if cfg.Scheduler.RemindDays != 5 {
		t.Fatalf("Scheduler.RemindDays = %d", cfg.Scheduler.RemindDays)
	}
	if cfg.Scheduler.CheckTime != "09:30" {
		t.Fatalf("Scheduler.CheckTime = %q", cfg.Scheduler.CheckTime)
	}
}

func TestLoad_envOverridesFile(t *testing.T) {
	resetCurrent()

	t.Setenv("HOMESTOCK_SERVER_PORT", "7070")
	t.Setenv("HOMESTOCK_DATABASE_DRIVER", "mysql")
	t.Setenv("HOMESTOCK_DATABASE_DSN", "mysql://dsn")
	t.Setenv("HOMESTOCK_NOTIFY_FEISHU_WEBHOOK", "https://env.example/hook")
	t.Setenv("HOMESTOCK_SCHEDULER_REMIND_DAYS", "8")
	t.Setenv("HOMESTOCK_SCHEDULER_CHECK_TIME", "11:45")

	path := writeConfigFile(t, `{
		"server": {"port": "9090"},
		"database": {"driver": "postgres", "dsn": "postgres://db"},
		"notify": {"feishu_webhook": "https://file.example/hook"},
		"scheduler": {"remind_days": 5, "check_time": "09:30"}
	}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server.Port != "7070" {
		t.Fatalf("Server.Port = %q", cfg.Server.Port)
	}
	if cfg.Database.Driver != "mysql" {
		t.Fatalf("Database.Driver = %q", cfg.Database.Driver)
	}
	if cfg.Database.DSN != "mysql://dsn" {
		t.Fatalf("Database.DSN = %q", cfg.Database.DSN)
	}
	if cfg.Notify.FeishuWebhook != "https://env.example/hook" {
		t.Fatalf("Notify.FeishuWebhook = %q", cfg.Notify.FeishuWebhook)
	}
	if cfg.Scheduler.RemindDays != 8 {
		t.Fatalf("Scheduler.RemindDays = %d", cfg.Scheduler.RemindDays)
	}
	if cfg.Scheduler.CheckTime != "11:45" {
		t.Fatalf("Scheduler.CheckTime = %q", cfg.Scheduler.CheckTime)
	}
}

func TestLoad_envOnly(t *testing.T) {
	resetCurrent()

	t.Setenv("HOMESTOCK_SERVER_PORT", "6060")
	t.Setenv("HOMESTOCK_DATABASE_DRIVER", "postgres")
	t.Setenv("HOMESTOCK_DATABASE_DSN", "postgres://env-only")
	t.Setenv("HOMESTOCK_NOTIFY_FEISHU_WEBHOOK", "https://env-only.example/hook")
	t.Setenv("HOMESTOCK_SCHEDULER_REMIND_DAYS", "10")
	t.Setenv("HOMESTOCK_SCHEDULER_CHECK_TIME", "07:15")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if cfg.Server.Port != "6060" {
		t.Fatalf("Server.Port = %q", cfg.Server.Port)
	}
	if cfg.Database.Driver != "postgres" {
		t.Fatalf("Database.Driver = %q", cfg.Database.Driver)
	}
	if cfg.Database.DSN != "postgres://env-only" {
		t.Fatalf("Database.DSN = %q", cfg.Database.DSN)
	}
	if cfg.Notify.FeishuWebhook != "https://env-only.example/hook" {
		t.Fatalf("Notify.FeishuWebhook = %q", cfg.Notify.FeishuWebhook)
	}
	if cfg.Scheduler.RemindDays != 10 {
		t.Fatalf("Scheduler.RemindDays = %d", cfg.Scheduler.RemindDays)
	}
	if cfg.Scheduler.CheckTime != "07:15" {
		t.Fatalf("Scheduler.CheckTime = %q", cfg.Scheduler.CheckTime)
	}
}

func TestLoad_missingFileUsesDefaults(t *testing.T) {
	resetCurrent()

	path := filepath.Join(t.TempDir(), "missing.json")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	assertDefaultConfig(t, cfg)
}

func TestLoad_invalidJSONReturnsError(t *testing.T) {
	resetCurrent()

	path := writeConfigFile(t, `{"server":`)

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !strings.Contains(err.Error(), "parse config file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_invalidEnvIntReturnsError(t *testing.T) {
	resetCurrent()
	t.Setenv("HOMESTOCK_SCHEDULER_REMIND_DAYS", "abc")

	_, err := Load("")
	if err == nil {
		t.Fatal("expected error for invalid env int")
	}
	if !strings.Contains(err.Error(), "HOMESTOCK_SCHEDULER_REMIND_DAYS") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoad_failureDoesNotOverrideCurrent(t *testing.T) {
	resetCurrent()

	first, err := Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if first.Server.Port != "8080" {
		t.Fatalf("first.Server.Port = %q", first.Server.Port)
	}

	t.Setenv("HOMESTOCK_SERVER_PORT", "9000")
	t.Setenv("HOMESTOCK_SCHEDULER_REMIND_DAYS", "bad")

	_, err = Load("")
	if err == nil {
		t.Fatal("expected error for invalid env int")
	}

	got := Get()
	assertDefaultConfig(t, got)
}

func TestGet_returnsCopy(t *testing.T) {
	resetCurrent()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	got := Get()
	got.Server.Port = "9999"

	if cfg.Server.Port != "8080" {
		t.Fatalf("Load result was mutated: %q", cfg.Server.Port)
	}
	if Get().Server.Port != "8080" {
		t.Fatalf("stored config was mutated: %q", Get().Server.Port)
	}
}

func resetCurrent() {
	mu.Lock()
	defer mu.Unlock()
	current = nil
}

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile returned error: %v", err)
	}

	return path
}

func assertDefaultConfig(t *testing.T, cfg *Config) {
	t.Helper()

	if cfg == nil {
		t.Fatal("cfg is nil")
	}
	if cfg.Server.Port != "8080" {
		t.Fatalf("Server.Port = %q", cfg.Server.Port)
	}
	if cfg.Database.Driver != "sqlite" {
		t.Fatalf("Database.Driver = %q", cfg.Database.Driver)
	}
	if cfg.Database.DSN != "./data/inventory.db" {
		t.Fatalf("Database.DSN = %q", cfg.Database.DSN)
	}
	if cfg.Notify.FeishuWebhook != "" {
		t.Fatalf("Notify.FeishuWebhook = %q", cfg.Notify.FeishuWebhook)
	}
	if cfg.Scheduler.RemindDays != 3 {
		t.Fatalf("Scheduler.RemindDays = %d", cfg.Scheduler.RemindDays)
	}
	if cfg.Scheduler.CheckTime != "08:00" {
		t.Fatalf("Scheduler.CheckTime = %q", cfg.Scheduler.CheckTime)
	}
}
