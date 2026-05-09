package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"

	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/agent"
	"github.com/zhaoxiaoyang741/HomeStock/internal/channel"
	"github.com/zhaoxiaoyang741/HomeStock/internal/channel/feishu"
	"github.com/zhaoxiaoyang741/HomeStock/internal/database"
	"github.com/zhaoxiaoyang741/HomeStock/internal/handler"
	"github.com/zhaoxiaoyang741/HomeStock/internal/hotreload"
	"github.com/zhaoxiaoyang741/HomeStock/internal/httpserver"
	"github.com/zhaoxiaoyang741/HomeStock/internal/llm"
	gormrepo "github.com/zhaoxiaoyang741/HomeStock/internal/repository/gorm"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
	"github.com/zhaoxiaoyang741/HomeStock/internal/tool"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

type App struct {
	configPath string
	server     *httpserver.Server
	db            *gorm.DB
	sqlDB         *sql.DB

	bus        *agent.MessageBus
	agentLoop  *agent.AgentLoop
	channelMgr *channel.Manager
	orchestrator *hotreload.Orchestrator
	hotReloadW   *hotreload.Watcher

	// outbound router lifecycle
	outboundCtx    context.Context
	outboundCancel context.CancelFunc
	outboundWg     sync.WaitGroup
}

func New(cfg *config.Config, configPath string) (*App, error) {
	db, err := database.OpenAndMigrate(cfg.Database)
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	// Services
	uow := gormrepo.NewUnitOfWork(db)
	materialSvc := service.NewMaterialService(uow)
	inventorySvc := service.NewInventoryService(uow)

	// LLM Provider — use the first enabled model from model_list
	modelCfg := firstEnabledModel(cfg.ModelList)
	if modelCfg == nil {
		return nil, errors.New("app: no model configured in model_list")
	}
	llmProvider, err := llm.NewProvider(*modelCfg)
	if err != nil {
		return nil, fmt.Errorf("app: create llm provider: %w", err)
	}

	// MessageBus — decouples channels from the AgentLoop
	bus := agent.NewMessageBus(0)

	// Tool dispatcher (tools are registered in Start() after services are ready)
	disp := tool.NewDispatcher()

	// AgentLoop
	systemPrompt := `你是 HomeStock（家库）库存管理助手，可以通过飞书帮助用户管理家庭库存。
	你可以帮助用户：
	1. 查询库存情况
	2. 新增物品入库
	3. 消耗出库
	4. 更新批次信息

	每次操作前先确认用户意图，操作完成后反馈结果。
	回复简洁友好。`
	agentLoop := agent.NewAgentLoop(bus, llmProvider, disp, systemPrompt)

	// Channel manager — channels are added via factory or manually
	channelMgr := channel.NewManager()

	// Always create the FeishuChannel wrapper for OAuth lifecycle management;
	// only add it to the manager when config has it enabled.
	inboundHandler := func(ctx context.Context, msg channel.InboundMessage) {
		if err := bus.PublishInbound(ctx, agent.InboundMessage{
			Channel:    msg.Channel,
			ChatID:     msg.ChatID,
			SenderID:   msg.SenderID,
			SenderName: msg.SenderName,
			Text:       msg.Text,
			MediaType:  msg.MediaType,
			FileKey:    msg.FileKey,
		}); err != nil {
			logger.ErrorCF("app", "publish inbound failed", map[string]any{
				"channel": msg.Channel,
				"error":   err.Error(),
			})
		}
	}

	fc := feishu.NewFeishuChannel(cfg.Channels.Feishu.AppID, cfg.Channels.Feishu.AppSecret)
	fc.SetInboundHandler(inboundHandler)
	if cfg.Channels.Feishu.Enabled {
		channelMgr.AddChannel(fc)
		logger.InfoCF("app", "Feishu channel enabled", nil)
	}

	// OAuth service and handler for Feishu (always registered, channel can be started
	// dynamically after OAuth callback).
	oauthSvc := feishu.NewOAuthService(
		cfg.Channels.Feishu.AppID,
		cfg.Channels.Feishu.AppSecret,
		cfg.Channels.Feishu.RedirectURI,
		cfg.Channels.Feishu.FrontendURL,
		uow.Repos().SystemSettings(),
	)
	feishuHandler := handler.NewFeishuHandler(oauthSvc, channelMgr, fc, cfg.Channels.Feishu.FrontendURL, configPath)
	feishuHandler.SetChannelUpdateFn(func(feishuCfg config.FeishuChannelConfig) error {
		ctx := context.Background()

		if err := fc.Reconfigure(ctx, feishuCfg.AppID, feishuCfg.AppSecret, feishuCfg.Enabled); err != nil {
			return fmt.Errorf("reconfigure feishu channel: %w", err)
		}

		oauthSvc.UpdateCredentials(feishuCfg.AppID, feishuCfg.AppSecret)

		if err := oauthSvc.ClearAuth(ctx); err != nil {
			logger.WarnCF("feishu", "failed to clear stale oauth token", map[string]any{"error": err.Error()})
		}

		if feishuCfg.Enabled {
			if _, exists := channelMgr.GetChannel("feishu"); !exists {
				channelMgr.AddChannel(fc)
			}
		} else {
			channelMgr.RemoveChannel("feishu")
		}

		return nil
	})

	// Seed the Feishu token cache from stored OAuth credentials (non-fatal on error).
	if cfg.Channels.Feishu.Enabled {
		if err := oauthSvc.SeedTokenCache(context.Background(), fc.GetTokenCache()); err != nil {
			logger.WarnCF("app", "feishu token cache seed failed", map[string]any{"error": err.Error()})
		}
	}

	// Model configuration handler
	activeModelName := ""
	if modelCfg != nil {
		activeModelName = modelCfg.ModelName
	}
	modelHandler := handler.NewModelHandler(configPath)
	modelHandler.SetActiveName(activeModelName)
	modelHandler.SetSwapFn(func(name string, cfg config.ModelConfig) error {
		provider, err := llm.NewProvider(cfg)
		if err != nil {
			return fmt.Errorf("create provider for model %q: %w", name, err)
		}
		agentLoop.SwapProvider(provider)
		return nil
	})

	// Hot-reload orchestrator — single entry point for all config changes
	orch := hotreload.NewOrchestrator(configPath, agentLoop, fc, oauthSvc, modelHandler)
	modelHandler.SetPostUpdateFn(func() {
		if err := orch.Reload(); err != nil {
			logger.ErrorCF("app", "hot-reload after model update failed", map[string]any{"error": err.Error()})
		}
	})
	modelHandler.SetReloadTimeFn(func() string {
		return orch.LastReloadTime().Format("2006-01-02 15:04:05")
	})

	// Tool registration
	tool.RegisterInventoryTools(disp, &tool.InventoryTools{
		InventorySvc: inventorySvc,
		MaterialSvc:  materialSvc,
	})
	tool.RegisterHealthTool(disp, fmt.Sprintf("http://localhost:%s", cfg.Server.Port))

	defs := tool.InventoryToolDefinitions()
	defs = append(defs, tool.HealthToolDefinition())
	disp.SetDefinitions(defs)

	// Handlers
	categoryHandler := handler.NewCategoryHandler(service.NewCategoryService(uow))
	materialHandler := handler.NewMaterialHandler(materialSvc, inventorySvc)
	stockLotHandler := handler.NewStockLotHandler(inventorySvc)
	stockMovementHandler := handler.NewStockMovementHandler(gormrepo.NewStockMovementRepository(db))
	auditLogHandler := handler.NewAuditLogHandler(service.NewAuditService(uow))

	server := httpserver.New(cfg.Server,
		categoryHandler.RegisterRoutes,
		materialHandler.RegisterRoutes,
		stockLotHandler.RegisterRoutes,
		stockMovementHandler.RegisterRoutes,
		auditLogHandler.RegisterRoutes,
		feishuHandler.RegisterRoutes,
		modelHandler.RegisterRoutes,
	)

	return &App{
		configPath:   configPath,
		server:       server,
		db:           db,
		sqlDB:        sqlDB,
		bus:          bus,
		agentLoop:    agentLoop,
		channelMgr:   channelMgr,
		orchestrator: orch,
	}, nil
}

func (a *App) Start() error {
	ctx := context.Background()

	// 1. AgentLoop starts first — ready to consume inbound messages
	a.agentLoop.Start(ctx)

	// 2. Config file watcher (hot-reload), if enabled
	cfg := config.Get()
	if cfg.Server.HotReload {
		a.hotReloadW = hotreload.NewWatcher(a.configPath, a.orchestrator.Reload)
		a.hotReloadW.Start()
		logger.InfoCF("app", "config hot-reload enabled (polling every 2s)", nil)
	}

	// 4. Outbound router — delivers agent responses to the correct channel
	a.outboundCtx, a.outboundCancel = context.WithCancel(ctx)
	a.outboundWg.Add(1)
	go a.routeOutbound()

	// 5. Start channel manager — begins receiving messages from channels
	if err := a.channelMgr.StartAll(ctx); err != nil {
		return fmt.Errorf("app: start channels: %w", err)
	}

	// 6. Register ops endpoints
	a.server.Engine().POST("/reload", func(c *gin.Context) {
		if err := a.orchestrator.Reload(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "reload failed: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "reload completed"})
	})

	// 7. Start HTTP server (blocking)
	return a.server.Start()
}

func (a *App) Shutdown(ctx context.Context) error {
	// Reverse order: channels → agent → HTTP

	if a.hotReloadW != nil {
		a.hotReloadW.Stop()
	}

	if err := a.channelMgr.StopAll(ctx); err != nil {
		logger.ErrorCF("app", "channel manager stop error", map[string]any{"error": err.Error()})
	}

	a.agentLoop.Stop()

	if a.outboundCancel != nil {
		a.outboundCancel()
	}
	a.outboundWg.Wait()

	a.bus.Close()

	return a.server.Shutdown(ctx)
}

func (a *App) Close() error {
	if a == nil || a.sqlDB == nil {
		return nil
	}
	return a.sqlDB.Close()
}

// firstEnabledModel returns the first enabled model config, or falls back
// to the first entry if none have enabled explicitly set.
func firstEnabledModel(list []config.ModelConfig) *config.ModelConfig {
	if len(list) == 0 {
		return nil
	}
	for i := range list {
		if list[i].Enabled {
			return &list[i]
		}
	}
	// Backward compatibility: if no entry has enabled:true, use the first one.
	return &list[0]
}

// routeOutbound reads agent.OutboundMessage from the bus and sends them
// to the appropriate channel via ChannelManager.
func (a *App) routeOutbound() {
	defer a.outboundWg.Done()
	for {
		select {
		case msg, ok := <-a.bus.OutboundChan():
			if !ok {
				return
			}
			ch, exists := a.channelMgr.GetChannel(msg.Channel)
			if !exists {
				logger.WarnCF("app", "no channel for outbound message", map[string]any{
					"channel": msg.Channel,
					"chat_id": msg.ChatID,
				})
				continue
			}
			if err := ch.Send(a.outboundCtx, channel.OutboundMessage{
				Channel: msg.Channel,
				ChatID:  msg.ChatID,
				Text:    msg.Text,
			}); err != nil {
				logger.ErrorCF("app", "channel send failed", map[string]any{
					"channel": msg.Channel,
					"chat_id": msg.ChatID,
					"error":   err.Error(),
				})
			}
		case <-a.outboundCtx.Done():
			return
		}
	}
}
