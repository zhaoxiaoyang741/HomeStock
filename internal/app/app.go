package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/agent"
	"github.com/zhaoxiaoyang741/HomeStock/internal/channel"
	"github.com/zhaoxiaoyang741/HomeStock/internal/channel/feishu"
	"github.com/zhaoxiaoyang741/HomeStock/internal/database"
	"github.com/zhaoxiaoyang741/HomeStock/internal/handler"
	"github.com/zhaoxiaoyang741/HomeStock/internal/httpserver"
	"github.com/zhaoxiaoyang741/HomeStock/internal/llm"
	gormrepo "github.com/zhaoxiaoyang741/HomeStock/internal/repository/gorm"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
	"github.com/zhaoxiaoyang741/HomeStock/internal/taskcenter"
	"github.com/zhaoxiaoyang741/HomeStock/internal/tasks"
	"github.com/zhaoxiaoyang741/HomeStock/internal/tool"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

type App struct {
	server        *httpserver.Server
	db            *gorm.DB
	sqlDB         *sql.DB
	taskCenterSvc *taskcenter.TaskCenterService

	bus        *agent.MessageBus
	agentLoop  *agent.AgentLoop
	channelMgr *channel.Manager

	// outbound router lifecycle
	outboundCtx    context.Context
	outboundCancel context.CancelFunc
	outboundWg     sync.WaitGroup
}

func New(cfg *config.Config) (*App, error) {
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
	systemSettingsSvc := service.NewSystemSettingsService(uow, cfg)
	notificationSvc := service.NewNotificationService(uow, systemSettingsSvc)
	taskCenterSvc := taskcenter.NewTaskCenterService(
		uow,
		tasks.NewExpiryNotificationTaskDefinition(notificationSvc),
	)

	// Repositories exposed directly for handlers that don't need a service layer
	notificationRepo := gormrepo.NewNotificationRepository(db)

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
	feishuHandler := handler.NewFeishuHandler(oauthSvc, channelMgr, fc, cfg.Channels.Feishu.FrontendURL)

	// Seed the Feishu token cache from stored OAuth credentials (non-fatal on error).
	if cfg.Channels.Feishu.Enabled {
		if err := oauthSvc.SeedTokenCache(context.Background(), fc.GetTokenCache()); err != nil {
			logger.WarnCF("app", "feishu token cache seed failed", map[string]any{"error": err.Error()})
		}
	}

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
	systemSettingsHandler := handler.NewSystemSettingsHandler(systemSettingsSvc)
	scheduledTaskHandler := handler.NewScheduledTaskHandler(taskCenterSvc)
	schedulerHandler := handler.NewSchedulerHandler(taskCenterSvc)
	notificationHandler := handler.NewNotificationHandler(notificationRepo)

	server := httpserver.New(cfg.Server,
		categoryHandler.RegisterRoutes,
		materialHandler.RegisterRoutes,
		stockLotHandler.RegisterRoutes,
		stockMovementHandler.RegisterRoutes,
		auditLogHandler.RegisterRoutes,
		systemSettingsHandler.RegisterRoutes,
		scheduledTaskHandler.RegisterRoutes,
		schedulerHandler.RegisterRoutes,
		notificationHandler.RegisterRoutes,
		feishuHandler.RegisterRoutes,
	)

	return &App{
		server:        server,
		db:            db,
		sqlDB:         sqlDB,
		taskCenterSvc: taskCenterSvc,
		bus:           bus,
		agentLoop:     agentLoop,
		channelMgr:    channelMgr,
	}, nil
}

func (a *App) Start() error {
	ctx := context.Background()

	// 1. AgentLoop starts first — ready to consume inbound messages
	a.agentLoop.Start(ctx)

	// 2. Outbound router — delivers agent responses to the correct channel
	a.outboundCtx, a.outboundCancel = context.WithCancel(ctx)
	a.outboundWg.Add(1)
	go a.routeOutbound()

	// 3. Start channel manager — begins receiving messages from channels
	if err := a.channelMgr.StartAll(ctx); err != nil {
		return fmt.Errorf("app: start channels: %w", err)
	}

	// 4. Start scheduled task center
	if err := a.taskCenterSvc.Start(ctx); err != nil {
		return err
	}

	// 5. Start HTTP server (blocking)
	return a.server.Start()
}

func (a *App) Shutdown(ctx context.Context) error {
	// Reverse order: channels → agent → HTTP

	if err := a.channelMgr.StopAll(ctx); err != nil {
		logger.ErrorCF("app", "channel manager stop error", map[string]any{"error": err.Error()})
	}

	a.agentLoop.Stop()

	if a.outboundCancel != nil {
		a.outboundCancel()
	}
	a.outboundWg.Wait()

	a.taskCenterSvc.Stop()
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
