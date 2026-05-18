package app

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	httpreq "github.com/zhaoxiaoyang741/HomeStock/internal/api/http/request"
	"github.com/zhaoxiaoyang741/HomeStock/internal/handler"
	"github.com/zhaoxiaoyang741/HomeStock/internal/integration/agent"
	"github.com/zhaoxiaoyang741/HomeStock/internal/integration/channel/feishu"
	"github.com/zhaoxiaoyang741/HomeStock/internal/integration/channel/wechat"
	"github.com/zhaoxiaoyang741/HomeStock/internal/integration/hotreload"
	"github.com/zhaoxiaoyang741/HomeStock/internal/integration/tool"
	gormrepo "github.com/zhaoxiaoyang741/HomeStock/internal/repository/gorm"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/bus"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/channel"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/database"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/llm"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/server"
)

// Server is the application container, owning all subsystems.
type Server struct {
	configPath string
	server     *server.Server
	db         *gorm.DB
	sqlDB      *sql.DB

	// Agent system
	bus       *bus.MessageBus
	agentLoop *agent.AgentLoop

	// Channel system
	channelMgr *channel.Manager
	wechatCh   *wechat.WechatChannel

	// Hot-reload
	orchestrator *hotreload.Orchestrator
	hotReloadW   *hotreload.Watcher

	// Outbound router lifecycle
	outboundCtx    context.Context
	outboundCancel context.CancelFunc
	outboundWg     sync.WaitGroup
}

// New is the composition root. It initializes all subsystems in dependency order.
func New(cfg *config.Config, configPath string) (*Server, error) {
	// 1. Database
	db, sqlDB, err := initDatabase(cfg.Database)
	if err != nil {
		return nil, err
	}

	// 2. Services
	uow, materialSvc, inventorySvc, authSvc := initServices(db, cfg.Auth)

	// Auto-create admin user if not exists (first startup)
	if err := initAdminUser(authSvc); err != nil {
		return nil, err
	}

	// Auto-generate JWT secret if none configured
	if cfg.Auth.JWTSecret == "" {
		cfg.Auth.JWTSecret = authSvc.GetSecretHex()
		logger.InfoCF("app", "auto-generated JWT secret (set HOMESTOCK_AUTH_JWT_SECRET to persist across restarts)", map[string]any{
			"secret_hex": cfg.Auth.JWTSecret,
		})
	}

	// 3. Agent system (LLM, message bus, tools)
	modelCfg, _, msgBus, disp, agentLoop, err := initAgent(cfg.ModelList, materialSvc, inventorySvc, cfg.Server.Port)
	if err != nil {
		return nil, err
	}

	// 4. Channels (Feishu, WeChat, OAuth)
	channelMgr, feishuHandler, fc, oauthSvc, wechatCh := initChannels(cfg, msgBus, uow, configPath)

	// 5. Model handler
	modelHandler := initModelHandler(configPath, modelCfg, agentLoop)

	// 6. Hot-reload orchestrator
	orch := initHotReload(configPath, agentLoop, fc, oauthSvc, modelHandler, wechatCh, channelMgr)

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
	if feishuCfg, ok := cfg.FeishuConfig(); ok && feishuCfg.Enabled {
		if err := oauthSvc.SeedTokenCache(context.Background(), fc.GetTokenCache()); err != nil {
			logger.WarnCF("app", "feishu token cache seed failed", map[string]any{"error": err.Error()})
		}
	}

	// 9. HTTP server
	wechatHandler := handler.NewWechatHandler(channelMgr, wechatCh, configPath)
	srv := initServer(cfg.Server, db, uow, authSvc, orch, materialSvc, inventorySvc, feishuHandler, modelHandler, wechatHandler)

	// Register tool definitions on dispatcher
	tool.RegisterInventoryTools(disp, &tool.InventoryTools{
		InventorySvc: inventorySvc,
		MaterialSvc:  materialSvc,
	})
	tool.RegisterHealthTool(disp, fmt.Sprintf("http://localhost:%s", cfg.Server.Port))
	defs := tool.InventoryToolDefinitions()
	defs = append(defs, tool.HealthToolDefinition())
	disp.SetDefinitions(defs)

	return &Server{
		configPath:   configPath,
		server:       srv,
		db:           db,
		sqlDB:        sqlDB,
		bus:          msgBus,
		agentLoop:    agentLoop,
		channelMgr:   channelMgr,
		wechatCh:     wechatCh,
		orchestrator: orch,
	}, nil
}

// Start begins all subsystems: agent loop, hot-reload watcher, outbound router,
// channel manager, and the HTTP server (blocking).
func (s *Server) Start() error {
	ctx := context.Background()
	s.agentLoop.Start(ctx)

	cfg := config.Get()
	if cfg.Server.HotReload {
		s.hotReloadW = hotreload.NewWatcher(s.configPath, s.orchestrator.Reload)
		s.hotReloadW.Start()
		logger.InfoCF("app", "config hot-reload enabled (polling every 2s)", nil)
	}

	s.outboundCtx, s.outboundCancel = context.WithCancel(ctx)
	s.outboundWg.Add(1)
	go s.routeOutbound()

	if err := s.channelMgr.StartAll(ctx); err != nil {
		return fmt.Errorf("app: start channels: %w", err)
	}

	return s.server.Start()
}

// Shutdown gracefully stops all subsystems in reverse dependency order.
// Order: HTTP → hot-reload → outbound/drain bus → agent → channels → DB
func (s *Server) Shutdown(ctx context.Context) error {
	// 1. Stop HTTP first — no new requests
	if err := s.server.Shutdown(ctx); err != nil {
		logger.ErrorCF("app", "http server shutdown error", map[string]any{"error": err.Error()})
	}

	// 2. Stop hot-reload watcher (if any)
	if s.hotReloadW != nil {
		s.hotReloadW.Stop()
	}

	// 3. Cancel outbound router (drains bus)
	if s.outboundCancel != nil {
		s.outboundCancel()
	}
	s.outboundWg.Wait()
	s.bus.Close()

	// 4. Stop agent loop
	s.agentLoop.Stop()

	// 5. Stop channels
	if err := s.channelMgr.StopAll(ctx); err != nil {
		logger.ErrorCF("app", "channel manager stop error", map[string]any{"error": err.Error()})
	}

	// 6. Close DB
	return s.sqlDB.Close()
}

// routeOutbound reads agent outbound messages from the bus and routes them
// to the appropriate channel via the Manager's per-channel worker queue.
func (s *Server) routeOutbound() {
	defer s.outboundWg.Done()
	for {
		select {
		case msg, ok := <-s.bus.OutboundChan():
			if !ok {
				return
			}
			if err := s.channelMgr.RouteOutbound(s.outboundCtx, channel.OutboundMessage{
				Channel: msg.Channel,
				ChatID:  msg.ChatID,
				Text:    msg.Text,
			}); err != nil {
				if err == channel.ErrNotRunning {
					logger.WarnCF("app", "no active worker for outbound message", map[string]any{
						"channel": msg.Channel,
						"chat_id": msg.ChatID,
					})
				} else {
					logger.ErrorCF("app", "route outbound failed", map[string]any{
						"channel": msg.Channel,
						"chat_id": msg.ChatID,
						"error":   err.Error(),
					})
				}
			}
		case <-s.outboundCtx.Done():
			return
		}
	}
}

// ---------------------------------------------------------------------------
// Initialization helpers
// ---------------------------------------------------------------------------

func initDatabase(cfg config.DatabaseConfig) (*gorm.DB, *sql.DB, error) {
	db, err := database.OpenAndMigrate(cfg)
	if err != nil {
		return nil, nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, err
	}
	return db, sqlDB, nil
}

func initServices(db *gorm.DB, authCfg config.AuthConfig) (
	uow *gormrepo.UnitOfWork,
	materialSvc *service.MaterialService,
	inventorySvc *service.InventoryService,
	authSvc *service.AuthService,
) {
	uow = gormrepo.NewUnitOfWork(db)
	materialSvc = service.NewMaterialService(uow)
	inventorySvc = service.NewInventoryService(uow)
	authSvc = service.NewAuthService(db, authCfg.JWTSecret, authCfg.TokenDurationMinutes)
	return
}

func initAdminUser(authSvc *service.AuthService) error {
	ctx := context.Background()

	key := make([]byte, 12)
	if _, err := rand.Read(key); err != nil {
		return fmt.Errorf("generate admin password: %w", err)
	}
	password := hex.EncodeToString(key)

	_, err := authSvc.Register(ctx, "admin", password, "Admin")
	if err != nil {
		if errors.Is(err, service.ErrUserExists) {
			return nil
		}
		return fmt.Errorf("create admin user: %w", err)
	}

	logger.InfoCF("app", "========================================", nil)
	logger.InfoCF("app", "  Admin user created on first startup!", nil)
	logger.InfoCF("app", "  Username: admin", nil)
	logger.InfoCF("app", fmt.Sprintf("  Password: %s", password), nil)
	logger.InfoCF("app", "  Please change the password after login.", nil)
	logger.InfoCF("app", "========================================", nil)
	return nil
}

const systemPrompt = `你是 HomeStock（变便）库存管理助手，可以通过飞书帮助用户管理家庭库存。
你可以帮助用户：
1. 查询库存情况
2. 新增物品入库
3. 消耗出库
4. 更新批次信息

每次操作前先确认用户意图，操作完成后反馈结果。
回复简洁友好。

== 批量输入指引 ==
用户可以一次输入多个物品，例如："买了5斤苹果、2箱牛奶、一袋大米"。
遇到这种情况，请分别调用入库/出库工具处理每个物品，每次调用一个物品。
处理完所有物品后，汇总结果一次性回复用户。

== 确认为先 ==
如果用户一次提及多个物品，或者操作可能影响较大（如大量出库），
先列出物品让用户自然确认（如"识别到苹果5斤、牛奶2箱，需要入库吗？"），
等用户回复确认后再调用工具执行。
如果是单个物品的简单操作或查询类请求，直接执行不需要确认。`

func initAgent(
	modelList []config.ModelConfig,
	materialSvc *service.MaterialService,
	inventorySvc *service.InventoryService,
	port string,
) (
	modelCfg *config.ModelConfig,
	llmProvider llm.LLMProvider,
	msgBus *bus.MessageBus,
	disp *tool.Dispatcher,
	agentLoop *agent.AgentLoop,
	err error,
) {
	modelCfg = firstEnabledModel(modelList)
	if modelCfg == nil {
		return nil, nil, nil, nil, nil, errors.New("app: no model configured in model_list")
	}

	llmProvider, err = llm.NewProvider(*modelCfg)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("app: create llm provider: %w", err)
	}

	msgBus = bus.NewMessageBus(0)
	disp = tool.NewDispatcher()

	nluEngine := agent.NewNluEngine(materialSvc)
	agentLoop = agent.NewAgentLoop(msgBus, llmProvider, disp, systemPrompt, nluEngine)

	return
}

func firstEnabledModel(models []config.ModelConfig) *config.ModelConfig {
	for i := range models {
		if models[i].Enabled {
			return &models[i]
		}
	}
	return nil
}

func initChannels(
	cfg *config.Config,
	msgBus *bus.MessageBus,
	uow *gormrepo.UnitOfWork,
	configPath string,
) (
	channelMgr *channel.Manager,
	feishuH *handler.FeishuHandler,
	feishuCh *feishu.FeishuChannel,
	oauthSvc *feishu.OAuthService,
	wechatCh *wechat.WechatChannel,
) {
	channelMgr = channel.NewManager()

	// Inbound handler: routes channel messages into the agent MessageBus
	inboundHandler := func(ctx context.Context, msg channel.InboundMessage) {
		if err := msgBus.PublishInbound(ctx, bus.InboundMessage{
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

	// Use factory registry for all channels
	channel.RangeFactories(func(name string, factory channel.Factory) bool {
		raw, ok := cfg.Channels[name]
		if !ok {
			return true // no config for this channel, skip
		}
		ch, err := factory(raw)
		if err != nil {
			logger.ErrorCF("app", "channel factory error", map[string]any{"name": name, "error": err.Error()})
			return true
		}
		if ch == nil {
			return true // factory returned nil (channel disabled)
		}
		if setter, ok := ch.(channel.InboundHandlerSetter); ok {
			setter.SetInboundHandler(inboundHandler)
		}
		channelMgr.AddChannel(ch)
		logger.InfoCF("app", "channel created via factory", map[string]any{"name": name})
		return true
	})

	// Feishu OAuth service (always created, may be nil if not configured)
	feishuCfg, _ := cfg.FeishuConfig()
	oauthSvc = feishu.NewOAuthService(
		feishuCfg.AppID,
		feishuCfg.AppSecret,
		feishuCfg.RedirectURI,
		feishuCfg.FrontendURL,
		uow.Repos().SystemSettings(),
	)

	// Feishu channel — may be nil if disabled
	if rawCh, ok := channelMgr.GetChannel("feishu"); ok {
		feishuCh, _ = rawCh.(*feishu.FeishuChannel)
	}
	feishuH = handler.NewFeishuHandler(oauthSvc, channelMgr, feishuCh, feishuCfg.FrontendURL, configPath)

	// Channel update callback: reconfigures Feishu channel when settings change
	feishuH.SetChannelUpdateFn(func(feishuCfg config.FeishuChannelConfig) error {
		ctx := context.Background()

		if feishuCh != nil {
			if err := feishuCh.Reconfigure(ctx, feishuCfg.AppID, feishuCfg.AppSecret, feishuCfg.Enabled); err != nil {
				return err
			}
		}

		oauthSvc.UpdateCredentials(feishuCfg.AppID, feishuCfg.AppSecret)

		if err := oauthSvc.ClearAuth(ctx); err != nil {
			logger.WarnCF("feishu", "failed to clear stale oauth token", map[string]any{"error": err.Error()})
		}

		if feishuCfg.Enabled {
			if _, exists := channelMgr.GetChannel("feishu"); !exists && feishuCh != nil {
				channelMgr.AddChannel(feishuCh)
			}
		} else {
			channelMgr.RemoveChannel("feishu")
		}

		return nil
	})

	// WeChat channel — may be nil if disabled
	if rawCh, ok := channelMgr.GetChannel("wechat"); ok {
		wechatCh, _ = rawCh.(*wechat.WechatChannel)
	}

	return
}

func initModelHandler(configPath string, modelCfg *config.ModelConfig, agentLoop *agent.AgentLoop) *handler.ModelHandler {
	modelHandler := handler.NewModelHandler(configPath)
	if modelCfg != nil {
		modelHandler.SetActiveName(modelCfg.ModelName)
	}
	modelHandler.SetSwapFn(func(name string, cfg config.ModelConfig) error {
		provider, err := llm.NewProvider(cfg)
		if err != nil {
			return fmt.Errorf("create provider for model %q: %w", name, err)
		}
		agentLoop.SwapProvider(provider)
		return nil
	})
	return modelHandler
}

func initHotReload(
	configPath string,
	agentLoop *agent.AgentLoop,
	feishuCh *feishu.FeishuChannel,
	oauthSvc *feishu.OAuthService,
	modelHnd *handler.ModelHandler,
	wechatCh *wechat.WechatChannel,
	channelMgr *channel.Manager,
) *hotreload.Orchestrator {
	return hotreload.NewOrchestrator(configPath, agentLoop, feishuCh, oauthSvc, modelHnd, wechatCh, channelMgr)
}

func initServer(
	cfg config.ServerConfig,
	db *gorm.DB,
	uow *gormrepo.UnitOfWork,
	authSvc *service.AuthService,
	orch *hotreload.Orchestrator,
	materialSvc *service.MaterialService,
	inventorySvc *service.InventoryService,
	feishuHandler *handler.FeishuHandler,
	modelHandler *handler.ModelHandler,
	wechatHandler *handler.WechatHandler,
) *server.Server {
	authHandler := handler.NewAuthHandler(authSvc)
	categoryService := service.NewCategoryService(uow)
	categoryHandler := handler.NewCategoryHandler(categoryService)
	materialHandler := handler.NewMaterialHandler(materialSvc, inventorySvc)
	stockLotHandler := handler.NewStockLotHandler(inventorySvc)
	stockMovementHandler := handler.NewStockMovementHandler(gormrepo.NewStockMovementRepository(db))
	auditService := service.NewAuditService(uow)
	auditLogHandler := handler.NewAuditLogHandler(auditService)

	authMw := httpreq.JWTAuthMiddleware(authSvc)

	srv := server.New(cfg,
		[]server.RegisterRoutesFunc{
			authHandler.RegisterRoutes,
		},
		[]server.RegisterRoutesFunc{
			categoryHandler.RegisterRoutes,
			materialHandler.RegisterRoutes,
			stockLotHandler.RegisterRoutes,
			stockMovementHandler.RegisterRoutes,
			auditLogHandler.RegisterRoutes,
			feishuHandler.RegisterRoutes,
			modelHandler.RegisterRoutes,
			handler.NewBatchHandler(inventorySvc, materialSvc).RegisterRoutes,
			wechatHandler.RegisterRoutes,
			authHandler.RegisterProtectedRoutes,
		},
		server.AuthMiddleware(authMw),
	)

	// Register ops endpoint for manual config reload
	srv.Engine().POST("/reload", func(c *gin.Context) {
		if err := orch.Reload(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "reload failed: " + err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "reload completed"})
	})

	return srv
}
