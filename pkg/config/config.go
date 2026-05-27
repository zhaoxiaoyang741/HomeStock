package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

type Config struct {
	// Version is the config schema version for migration support.
	// Current version: 1.
	Version int `json:"version"`

	// Server controls HTTP service settings.
	Server ServerConfig `json:"server"`
	// Database controls storage backend settings.
	Database DatabaseConfig `json:"database"`
	// Auth controls user authentication settings.
	Auth AuthConfig `json:"auth"`

	// Outbound controls outbound event push endpoint settings.
	Outbound OutboundConfig `json:"outbound"`

	// Cron controls background scheduled task settings.
	Cron CronConfig `json:"cron"`

	Log LogConfig `json:"log"`
}

// AuthConfig configures JWT-based user authentication and API key access.
type AuthConfig struct {
	// JWTSecret is the HMAC signing key for JWT tokens.
	// If empty (the default), a random key is generated at startup.
	// Once set, changing it invalidates all existing tokens.
	JWTSecret string `json:"jwt_secret,omitempty"`
	// TokenDurationMinutes controls JWT token lifetime. Default: 1440 (24 h).
	TokenDurationMinutes int `json:"token_duration_minutes,omitempty"`
	// APIKeys is a list of static API keys for programmatic access.
	APIKeys []string `json:"api_keys,omitempty"`
}

// EndpointConfig defines an outbound push endpoint.
type EndpointConfig struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

// OutboundConfig controls outbound event push settings.
type OutboundConfig struct {
	Endpoints []EndpointConfig `json:"endpoints,omitempty"`
}

type ServerConfig struct {
	// Port is the HTTP listen port, for example "8080".
	Port string `json:"port"`
	// HotReload enables config file polling for runtime hot-reload.
	// When true, changes to config.json are detected within ~2.5 s and
	// applied to running services without a restart.
	HotReload bool `json:"hot_reload,omitempty"`
}

type DatabaseConfig struct {
	// Driver selects the database backend, such as "sqlite" or "postgres".
	Driver string `json:"driver"`
	// DSN is the database connection string or sqlite file path.
	DSN string `json:"dsn"`
}

// CronConfig controls background scheduled task settings.
type CronConfig struct {
	// Enabled enables the cron scheduler. Default: true.
	Enabled bool `json:"enabled"`
	// ExpiryCheckIntervalDays is the number of days before expiry to flag lots.
	// Default: 7.
	ExpiryCheckIntervalDays int `json:"expiry_check_interval_days,omitempty"`
	// ExpiryCheckPollInterval is how often the expiry scanner runs.
	// Accepts Go duration strings: "30m", "1h", "6h". Default: "6h".
	ExpiryCheckPollInterval string `json:"expiry_check_poll_interval,omitempty"`
	// NotifyEnabled controls whether expiry notifications are sent.
	// Default: false.
	NotifyEnabled bool `json:"notify_enabled,omitempty"`
	// NotifyTimeStart is the start of the notification time window in HH:MM format.
	// Default: "09:00".
	NotifyTimeStart string `json:"notify_time_start,omitempty"`
	// NotifyTimeEnd is the end of the notification time window in HH:MM format.
	// Default: "21:00".
	NotifyTimeEnd string `json:"notify_time_end,omitempty"`
}

type LogConfig struct {
	Level int    `json:"level"`
	Path  string `json:"path"`
}

// IsValidAPIKey checks whether the given key is in the configured API key list.
func (c *Config) IsValidAPIKey(key string) bool {
	for _, k := range c.Auth.APIKeys {
		if k == key {
			return true
		}
	}
	return false
}

var (
	mu      sync.RWMutex
	current *Config
)

// env > config.json > default config
func Load(path string) (*Config, error) {
	cfg := defaultConfig()

	if err := loadFile(path, cfg); err != nil {
		return nil, err
	}

	// version migration
	if cfg.Version < 1 {
		cfg.Version = 1
	}

	// auto-create config.json on first run
	if path != "" {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(path), 0755); err == nil {
				data, _ := json.MarshalIndent(cfg, "", "  ")
				_ = os.WriteFile(path, data, 0600)
			}
		}
	}

	if err := applyEnvOverrides(cfg); err != nil {
		return nil, err
	}

	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	mu.Lock()
	current = cfg
	mu.Unlock()

	return cloneConfig(cfg), nil
}

func Get() *Config {
	mu.RLock()
	defer mu.RUnlock()

	if current == nil {
		return defaultConfig()
	}

	return cloneConfig(current)
}

func defaultConfig() *Config {
	return &Config{
		Version: 1,
		Server: ServerConfig{
			Port: "80",
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			DSN:    "./data/inventory.db",
		},
		Auth: AuthConfig{
			JWTSecret:            "",
			TokenDurationMinutes: 1440, // 24 h
		},
		Outbound: OutboundConfig{},
		Cron: CronConfig{
			Enabled:                 true,
			ExpiryCheckIntervalDays: 7,
			ExpiryCheckPollInterval: "6h",
			NotifyEnabled:           false,
			NotifyTimeStart:         "09:00",
			NotifyTimeEnd:           "21:00",
		},
		Log: LogConfig{
			Level: 0, // INFO
			Path:  "logs/info.log",
		},
	}
}

// Save atomically updates config.json and the in-memory Config.
// fn receives the Config loaded from file (without env overrides).
// After fn returns, the modified config is written to disk atomically,
// then env overrides are re-applied to the in-memory copy.
func Save(path string, fn func(cfg *Config)) error {
	mu.Lock()
	defer mu.Unlock()

	cfg := defaultConfig()
	if err := loadFile(path, cfg); err != nil {
		return fmt.Errorf("save: load file config: %w", err)
	}

	fn(cfg)

	if err := validateConfig(cfg); err != nil {
		return fmt.Errorf("save: validation failed: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("save: marshal: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("save: write temp: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("save: rename: %w", err)
	}

	// Re-apply env overrides and refresh in-memory config
	if err := applyEnvOverrides(cfg); err != nil {
		return fmt.Errorf("save: re-apply env overrides: %w", err)
	}
	current = cfg
	return nil
}

func cloneConfig(cfg *Config) *Config {
	if cfg == nil {
		return nil
	}

	cloned := *cfg
	if cfg.Outbound.Endpoints != nil {
		cloned.Outbound.Endpoints = make([]EndpointConfig, len(cfg.Outbound.Endpoints))
		copy(cloned.Outbound.Endpoints, cfg.Outbound.Endpoints)
	}
	return &cloned
}

// validateConfig checks that the config is internally consistent.
func validateConfig(cfg *Config) error {
	return nil
}

func loadFile(path string, cfg *Config) error {
	if path == "" {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}

		return fmt.Errorf("read config file %q: %w", path, err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return fmt.Errorf("parse config file %q: %w", path, err)
	}

	return nil
}

func applyEnvOverrides(cfg *Config) error {
	if value, ok := os.LookupEnv("HOMESTOCK_SERVER_PORT"); ok {
		cfg.Server.Port = value
	}

	if value, ok := os.LookupEnv("HOMESTOCK_DATABASE_DRIVER"); ok {
		cfg.Database.Driver = value
	}

	if value, ok := os.LookupEnv("HOMESTOCK_DATABASE_DSN"); ok {
		cfg.Database.DSN = value
	}

	if value, ok := os.LookupEnv("HOMESTOCK_AUTH_JWT_SECRET"); ok {
		cfg.Auth.JWTSecret = value
	}

	if value, ok := os.LookupEnv("HOMESTOCK_AUTH_API_KEYS"); ok {
		cfg.Auth.APIKeys = strings.Split(value, ",")
	}

	if value, ok := os.LookupEnv("HOMESTOCK_CRON_EXPIRY_DAYS"); ok {
		if days, err := strconv.Atoi(value); err == nil {
			cfg.Cron.ExpiryCheckIntervalDays = days
		}
	}
	if value, ok := os.LookupEnv("HOMESTOCK_CRON_EXPIRY_POLL"); ok {
		cfg.Cron.ExpiryCheckPollInterval = value
	}
	if value, ok := os.LookupEnv("HOMESTOCK_CRON_NOTIFY_ENABLED"); ok {
		cfg.Cron.NotifyEnabled = value == "true"
	}
	if value, ok := os.LookupEnv("HOMESTOCK_CRON_NOTIFY_TIME_START"); ok {
		cfg.Cron.NotifyTimeStart = value
	}
	if value, ok := os.LookupEnv("HOMESTOCK_CRON_NOTIFY_TIME_END"); ok {
		cfg.Cron.NotifyTimeEnd = value
	}

	return nil
}
