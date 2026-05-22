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
		"database": {"driver": "postgres", "dsn": "postgres://db"}
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
}

func TestLoad_envOverridesFile(t *testing.T) {
	resetCurrent()

	t.Setenv("HOMESTOCK_SERVER_PORT", "7070")
	t.Setenv("HOMESTOCK_DATABASE_DRIVER", "mysql")
	t.Setenv("HOMESTOCK_DATABASE_DSN", "mysql://dsn")

	path := writeConfigFile(t, `{
		"server": {"port": "9090"},
		"database": {"driver": "postgres", "dsn": "postgres://db"}
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
}

func TestLoad_envOnly(t *testing.T) {
	resetCurrent()

	t.Setenv("HOMESTOCK_SERVER_PORT", "6060")
	t.Setenv("HOMESTOCK_DATABASE_DRIVER", "postgres")
	t.Setenv("HOMESTOCK_DATABASE_DSN", "postgres://env-only")

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

func TestGet_returnsCopy(t *testing.T) {
	resetCurrent()

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	got := Get()
	got.Server.Port = "9999"

	if cfg.Server.Port != "80" {
		t.Fatalf("Load result was mutated: %q", cfg.Server.Port)
	}
	if Get().Server.Port != "80" {
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
	if cfg.Server.Port != "80" {
		t.Fatalf("Server.Port = %q", cfg.Server.Port)
	}
	if cfg.Database.Driver != "sqlite" {
		t.Fatalf("Database.Driver = %q", cfg.Database.Driver)
	}
	if cfg.Database.DSN != "./data/inventory.db" {
		t.Fatalf("Database.DSN = %q", cfg.Database.DSN)
	}
}
