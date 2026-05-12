package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
)

type Config struct {
	// Server controls HTTP service settings.
	Server ServerConfig `json:"server"`
	// Database controls storage backend settings.
	Database DatabaseConfig `json:"database"`
	// Channels controls external messaging channel configurations.
	Channels ChannelsConfig `json:"channels"`
	// ModelList is the list of LLM model configurations for multi-model support.
	ModelList []ModelConfig `json:"model_list"`
	// Auth controls user authentication settings.
	Auth AuthConfig `json:"auth"`

	Log LogConfig `json:"log"`
}

// AuthConfig configures JWT-based user authentication.
type AuthConfig struct {
	// JWTSecret is the HMAC signing key for JWT tokens.
	// If empty (the default), a random key is generated at startup.
	// Once set, changing it invalidates all existing tokens.
	JWTSecret string `json:"jwt_secret,omitempty"`
	// TokenDurationMinutes controls JWT token lifetime. Default: 1440 (24 h).
	TokenDurationMinutes int `json:"token_duration_minutes,omitempty"`
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
	// Provider specifies the LLM provider type: "openai" (default), "ollama", or "deepseek".
	Provider string `json:"provider,omitempty"`
	// Enabled enables this model configuration. If false, the entry is skipped.
	// Defaults to false for clean configs; the app falls back to the first entry if none enabled.
	Enabled bool `json:"enabled,omitempty"`
	// APIKey is the API key for the LLM provider.
	APIKey string `json:"api_key"`
	// APIBase is the base URL for the API, optional 鈥?defaults to the provider's default.
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
		Server: ServerConfig{
			Port: "8888",
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			DSN:    "./data/inventory.db",
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
				Provider:  "openai",
				APIKey:    "",
			},
		},
		Auth: AuthConfig{
			JWTSecret:            "",
			TokenDurationMinutes: 1440, // 24 h
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
	if cfg.ModelList != nil {
		cloned.ModelList = make([]ModelConfig, len(cfg.ModelList))
		copy(cloned.ModelList, cfg.ModelList)
	}
	return &cloned
}

// validateConfig checks that the config is internally consistent.
func validateConfig(cfg *Config) error {
	if len(cfg.ModelList) == 0 {
		cfg.ModelList = []ModelConfig{
			{
				ModelName: "default",
				Model:     "gpt-4o",
				Provider:  "openai",
			},
		}
	}
	for i, m := range cfg.ModelList {
		if m.ModelName == "" {
			return fmt.Errorf("model_list[%d].model_name is required", i)
		}
		if m.Model == "" {
			return fmt.Errorf("model_list[%d].model is required", i)
		}
		if m.Provider == "" {
			cfg.ModelList[i].Provider = "openai"
		}
	}
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

	return nil
}
