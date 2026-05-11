package app

import (
	"context"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

// New is the application composition root. It initializes all subsystems
// in dependency order and returns a ready-to-run App.
func New(cfg *config.Config, configPath string) (*App, error) {
	// 1. Database
	db, sqlDB, err := initDatabase(cfg.Database)
	if err != nil {
		return nil, err
	}

	// 2. Services
	uow, materialSvc, inventorySvc, authSvc := initServices(db, cfg.Auth)

	// Auto-generate JWT secret if none configured
	if cfg.Auth.JWTSecret == "" {
		cfg.Auth.JWTSecret = authSvc.GetSecretHex()
		logger.InfoCF("app", "auto-generated JWT secret (set HOMESTOCK_AUTH_JWT_SECRET to persist across restarts)", map[string]any{
			"secret_hex": cfg.Auth.JWTSecret,
		})
	}

	// 3. Agent system (LLM, message bus, tools)
	modelCfg, _, bus, _, agentLoop, err := initAgent(cfg.ModelList, materialSvc, inventorySvc, cfg.Server.Port)
	if err != nil {
		return nil, err
	}

	// 4. Channels (Feishu, OAuth)
	channelMgr, feishuHandler, fc, oauthSvc := initChannels(cfg.Channels, bus, uow, configPath)

	// 5. Model handler
	modelHandler := initModelHandler(configPath, modelCfg, agentLoop)

	// 6. Hot-reload orchestrator
	orch := initHotReload(configPath, agentLoop, fc, oauthSvc, modelHandler)

	// 7. Wire remaining model handler callbacks (depend on orch)
	modelHandler.SetPostUpdateFn(func() {
		if err := orch.Reload(); err != nil {
			logger.ErrorCF("app", "hot-reload after model update failed", map[string]any{"error": err.Error()})
		}
	})
	modelHandler.SetReloadTimeFn(func() string {
		return orch.LastReloadTime().Format("2006-01-02 15:04:05")
	})

	// 8. Seed Feishu token cache from stored OAuth credentials
	if cfg.Channels.Feishu.Enabled {
		if err := oauthSvc.SeedTokenCache(context.Background(), fc.GetTokenCache()); err != nil {
			logger.WarnCF("app", "feishu token cache seed failed", map[string]any{"error": err.Error()})
		}
	}

	// 9. HTTP server
	server := initServer(cfg.Server, db, uow, authSvc, orch, materialSvc, inventorySvc, feishuHandler, modelHandler)

	return &App{
		configPath: configPath,
		server:     server,
		db:         db,
		sqlDB:      sqlDB,
		bus:        bus,
		agentLoop:  agentLoop,
		channelMgr: channelMgr,
		orchestrator: orch,
	}, nil
}
