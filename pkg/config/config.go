package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
)

type Config struct {
	// Server controls HTTP service settings.
	Server ServerConfig `json:"server"`
	// Database controls storage backend settings.
	Database DatabaseConfig `json:"database"`
	// Notify controls outbound notification settings.
	Notify NotifyConfig `json:"notify"`
	// Scheduler controls reminder scheduling settings.
	Scheduler SchedulerConfig `json:"scheduler"`
	// Channels controls external messaging channel configurations.
	Channels ChannelsConfig `json:"channels"`
	// ModelList is the list of LLM model configurations for multi-model support.
	ModelList []ModelConfig `json:"model_list"`

	Log LogConfig `json:"log"`
}

type ServerConfig struct {
	// Port is the HTTP listen port, for example "8080".
	Port string `json:"port"`
}

type DatabaseConfig struct {
	// Driver selects the database backend, such as "sqlite" or "postgres".
	Driver string `json:"driver"`
	// DSN is the database connection string or sqlite file path.
	DSN string `json:"dsn"`
}

type NotifyConfig struct {
	// FeishuWebhook is the Feishu bot webhook URL used for reminder delivery.
	FeishuWebhook string `json:"feishu_webhook"`
}

type SchedulerConfig struct {
	// RemindDays is how many days before expiration to send reminders.
	RemindDays int `json:"remind_days"`
	// CheckTime is the daily reminder check time in HH:MM format.
	CheckTime string `json:"check_time"`
}

type ChannelsConfig struct {
	Feishu FeishuChannelConfig `json:"feishu"`
}

type FeishuChannelConfig struct {
	Enabled      bool   `json:"enabled"`
	AppID        string `json:"app_id"`
	AppSecret    string `json:"app_secret"`
	RedirectURI  string `json:"redirect_uri,omitempty"`
	FrontendURL  string `json:"frontend_url,omitempty"`
}

type ModelConfig struct {
	// ModelName is a human-readable label used to reference this model config.
	ModelName string `json:"model_name"`
	// Model is the model identifier passed to the API, e.g. "openai/gpt-4o" or "gpt-4o".
	Model string `json:"model"`
	// APIKey is the API key for the LLM provider.
	APIKey string `json:"api_key"`
	// APIBase is the base URL for the API, optional — defaults to the provider's default.
	APIBase string `json:"api_base,omitempty"`
}

type LogConfig struct {
	Level int    `json:"level"`
	Path  string `json:"path"`
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

	if err := applyEnvOverrides(cfg); err != nil {
		return nil, err
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
		Server: ServerConfig{
			Port: "8888",
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			DSN:    "./data/inventory.db",
		},
		Notify: NotifyConfig{
			FeishuWebhook: "",
		},
		Scheduler: SchedulerConfig{
			RemindDays: 3,
			CheckTime:  "08:00",
		},
		Channels: ChannelsConfig{
			Feishu: FeishuChannelConfig{
				Enabled:     false,
				AppID:       "",
				AppSecret:   "",
				RedirectURI: "http://localhost:8888/api/v1/feishu/callback",
				FrontendURL: "http://localhost:5173",
			},
		},
		ModelList: []ModelConfig{
			{
				ModelName: "default",
				Model:     "openai/gpt-4o",
				APIKey:    "",
			},
		},
		Log: LogConfig{
			Level: 0, // INFO
			Path:  "logs/info.log",
		},
	}
}

func cloneConfig(cfg *Config) *Config {
	if cfg == nil {
		return nil
	}

	cloned := *cfg
	return &cloned
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

	if value, ok := os.LookupEnv("HOMESTOCK_NOTIFY_FEISHU_WEBHOOK"); ok {
		cfg.Notify.FeishuWebhook = value
	}

	if value, ok := os.LookupEnv("HOMESTOCK_SCHEDULER_REMIND_DAYS"); ok {
		remindDays, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("parse HOMESTOCK_SCHEDULER_REMIND_DAYS: %w", err)
		}
		cfg.Scheduler.RemindDays = remindDays
	}

	if value, ok := os.LookupEnv("HOMESTOCK_SCHEDULER_CHECK_TIME"); ok {
		cfg.Scheduler.CheckTime = value
	}

	if value, ok := os.LookupEnv("HOMESTOCK_CHANNELS_FEISHU_APP_ID"); ok {
		cfg.Channels.Feishu.AppID = value
		cfg.Channels.Feishu.Enabled = true
	}

	if value, ok := os.LookupEnv("HOMESTOCK_CHANNELS_FEISHU_APP_SECRET"); ok {
		cfg.Channels.Feishu.AppSecret = value
		cfg.Channels.Feishu.Enabled = true
	}

	if value, ok := os.LookupEnv("HOMESTOCK_CHANNELS_FEISHU_REDIRECT_URI"); ok {
		cfg.Channels.Feishu.RedirectURI = value
	}

	if value, ok := os.LookupEnv("HOMESTOCK_CHANNELS_FEISHU_FRONTEND_URL"); ok {
		cfg.Channels.Feishu.FrontendURL = value
	}

	return nil
}
