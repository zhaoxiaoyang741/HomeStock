package gateway

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

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
	"github.com/zhaoxiaoyang741/HomeStock/pkg/cron"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/llm"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

// Gateway is the Agent runtime, owning the message bus, agent loop, channel
// manager, cron scheduler, hot-reload orchestrator, and the outbound message
// router. It exposes accessors for HTTP handlers that need to interact with
// these subsystems.
type Gateway struct {
	configPath string

	// Agent system
	bus       *bus.MessageBus
	agentLoop *agent.AgentLoop

	// Channel system
	channelMgr *channel.Manager

	// Hot-reload
	orchestrator *hotreload.Orchestrator
	hotReloadW   *hotreload.Watcher

	// Cron scheduler
	cronSvc *cron.Service

	// Outbound router lifecycle
	outboundCtx    context.Context
	outboundCancel context.CancelFunc
	outboundWg     sync.WaitGroup

	// Handlers — owned by Gateway but exposed to the HTTP layer.
	feishuHandler *handler.FeishuHandler
	modelHandler  *handler.ModelHandler
	wechatHandler *handler.WechatHandler
	cronHandler   *handler.CronHandler

	// Tool dispatcher (definitions are set during New)
	dispatcher *tool.Dispatcher

	startedAt time.Time
}

// New creates a Gateway and all its subsystems. It takes the application
// config, the config file path, and the services required by the agent loop
// and cron scheduler.
func New(
	cfg *config.Config,
	configPath string,
	materialSvc *service.MaterialService,
	inventorySvc *service.InventoryService,
	uow *gormrepo.UnitOfWork,
) (*Gateway, error) {
	// -----------------------------------------------------------------------
	// Agent system
	// -----------------------------------------------------------------------
	modelCfg, _, msgBus, disp, agentLoop, err := initAgent(cfg, materialSvc, inventorySvc, cfg.Server.Port)
	if err != nil {
		return nil, err
	}

	// -----------------------------------------------------------------------
	// Channel system
	// -----------------------------------------------------------------------
	channelMgr := channel.NewManager()

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
			logger.ErrorCF("gateway", "publish inbound failed", map[string]any{
				"channel": msg.Channel,
				"error":   err.Error(),
			})
		}
	}

	// Use factory registry for all channels
	channel.RangeFactories(func(name string, factory channel.Factory) bool {
		raw, ok := cfg.Channels[name]
		if !ok {
			return true
		}
		ch, err := factory(raw)
		if err != nil {
			logger.ErrorCF("gateway", "channel factory error", map[string]any{"name": name, "error": err.Error()})
			return true
		}
		if ch == nil {
			return true
		}
		if setter, ok := ch.(channel.InboundHandlerSetter); ok {
			setter.SetInboundHandler(inboundHandler)
		}
		channelMgr.AddChannel(ch)
		logger.InfoCF("gateway", "channel created via factory", map[string]any{"name": name})
		return true
	})

	// Feishu OAuth service
	feishuCfg, _ := cfg.FeishuConfig()
	oauthSvc := feishu.NewOAuthService(
		feishuCfg.AppID,
		feishuCfg.AppSecret,
		feishuCfg.RedirectURI,
		feishuCfg.FrontendURL,
		uow.Repos().SystemSettings(),
	)

	// Feishu channel — may be nil if disabled
	var feishuCh *feishu.FeishuChannel
	if rawCh, ok := channelMgr.GetChannel("feishu"); ok {
		feishuCh, _ = rawCh.(*feishu.FeishuChannel)
	}

	feishuHandler := handler.NewFeishuHandler(oauthSvc, channelMgr, feishuCh, feishuCfg.FrontendURL, configPath)

	// Feishu channel update callback: reconfigures the Feishu channel when settings change
	feishuHandler.SetChannelUpdateFn(func(feishuCfg config.FeishuChannelConfig) error {
		ctx := context.Background()

		fc := feishuCh
		if fc == nil {
			if rawCh, ok := channelMgr.GetChannel("feishu"); ok {
				fc, _ = rawCh.(*feishu.FeishuChannel)
			}
		}
		if fc == nil && feishuCfg.AppID != "" && feishuCfg.AppSecret != "" {
			fc = feishu.NewFeishuChannel(feishuCfg.AppID, feishuCfg.AppSecret)
			fc.SetInboundHandler(inboundHandler)
			feishuHandler.SetChannel(fc)
		}

		if fc != nil {
			if err := fc.Reconfigure(ctx, feishuCfg.AppID, feishuCfg.AppSecret, feishuCfg.Enabled); err != nil {
				return err
			}
		}

		oauthSvc.UpdateCredentials(feishuCfg.AppID, feishuCfg.AppSecret)
		if err := oauthSvc.ClearAuth(ctx); err != nil {
			logger.WarnCF("gateway", "failed to clear stale oauth token", map[string]any{"error": err.Error()})
		}

		if feishuCfg.Enabled {
			if _, exists := channelMgr.GetChannel("feishu"); !exists && fc != nil {
				channelMgr.AddChannel(fc)
			}
		} else {
			channelMgr.RemoveChannel("feishu")
		}

		return nil
	})

	// WeChat channel — may be nil if disabled
	var wechatCh *wechat.WechatChannel
	if rawCh, ok := channelMgr.GetChannel("wechat"); ok {
		wechatCh, _ = rawCh.(*wechat.WechatChannel)
	}

	wechatHandler := handler.NewWechatHandler(channelMgr, wechatCh, configPath)

	// WeChat channel update callback: manages WeChat channel enable/disable
	wechatHandler.SetChannelUpdateFn(func(wechatCfg config.WechatChannelConfig) error {
		ctx := context.Background()

		if wechatCfg.Enabled {
			wc := wechatCh
			if wc == nil {
				if rawCh, ok := channelMgr.GetChannel("wechat"); ok {
					wc, _ = rawCh.(*wechat.WechatChannel)
				}
			}
			if wc == nil {
				wc = wechat.NewWechatChannel(wechatCfg)
				wc.SetInboundHandler(func(ctx context.Context, msg channel.InboundMessage) {
					if err := msgBus.PublishInbound(ctx, bus.InboundMessage{
						Channel:    msg.Channel,
						ChatID:     msg.ChatID,
						SenderID:   msg.SenderID,
						SenderName: msg.SenderName,
						Text:       msg.Text,
						MediaType:  msg.MediaType,
						FileKey:    msg.FileKey,
					}); err != nil {
						logger.ErrorCF("gateway", "publish inbound failed", map[string]any{
							"channel": msg.Channel,
							"error":   err.Error(),
						})
					}
				})
				wechatHandler.SetChannel(wc)
				channelMgr.AddChannel(wc)
			}

			if !wc.IsRunning() {
				wc.SetConfig(wechatCfg)
				if err := wc.Start(ctx); err != nil {
					return err
				}
			}
			if _, exists := channelMgr.GetChannel("wechat"); !exists {
				channelMgr.AddChannel(wc)
			}
		} else {
			if rawCh, ok := channelMgr.GetChannel("wechat"); ok {
				if wc, ok := rawCh.(*wechat.WechatChannel); ok && wc.IsRunning() {
					if err := wc.Stop(ctx); err != nil {
						return err
					}
				}
			}
			channelMgr.RemoveChannel("wechat")
		}

		return nil
	})

	// Model handler
	modelHandler := handler.NewModelHandler(configPath)
	if modelCfg != nil {
		modelHandler.SetActiveName(modelCfg.ModelName)
	}
	modelHandler.SetSwapFn(func(name string, mc config.ModelConfig) error {
		provider, err := llm.NewProvider(mc)
		if err != nil {
			return fmt.Errorf("create provider for model %q: %w", name, err)
		}
		agentLoop.SwapProvider(provider)
		return nil
	})

	// -----------------------------------------------------------------------
	// Hot-reload orchestrator
	// -----------------------------------------------------------------------
	orch := hotreload.NewOrchestrator(configPath, agentLoop, feishuCh, oauthSvc, modelHandler, wechatCh, channelMgr)

	// Wire model handler callbacks (depend on orchestrator)
	modelHandler.SetPostUpdateFn(func() {
		if err := orch.Reload(); err != nil {
			logger.ErrorCF("gateway", "hot-reload after model update failed", map[string]any{"error": err.Error()})
		}
	})
	modelHandler.SetReloadTimeFn(func() string {
		return orch.LastReloadTime().Format("2006-01-02 15:04:05")
	})

	// -----------------------------------------------------------------------
	// Cron
	// -----------------------------------------------------------------------
	cronHandler := handler.NewCronHandler(configPath)
	cronSvc := initCron(uow, cfg.Cron, channelMgr)

	// -----------------------------------------------------------------------
	// Seed Feishu token cache from stored OAuth credentials
	// -----------------------------------------------------------------------
	if feishuCfg, ok := cfg.FeishuConfig(); ok && feishuCfg.Enabled && oauthSvc != nil && feishuCh != nil {
		if err := oauthSvc.SeedTokenCache(context.Background(), feishuCh.GetTokenCache()); err != nil {
			logger.WarnCF("gateway", "feishu token cache seed failed", map[string]any{"error": err.Error()})
		}
	}

	// -----------------------------------------------------------------------
	// Register tool definitions on dispatcher
	// -----------------------------------------------------------------------
	tool.RegisterInventoryTools(disp, &tool.InventoryTools{
		InventorySvc: inventorySvc,
		MaterialSvc:  materialSvc,
	})
	tool.RegisterHealthTool(disp, fmt.Sprintf("http://localhost:%s", cfg.Server.Port))
	defs := tool.InventoryToolDefinitions()
	defs = append(defs, tool.HealthToolDefinition())
	disp.SetDefinitions(defs)

	return &Gateway{
		configPath:     configPath,
		bus:            msgBus,
		agentLoop:      agentLoop,
		channelMgr:     channelMgr,
		orchestrator:   orch,
		cronSvc:        cronSvc,
		feishuHandler:  feishuHandler,
		modelHandler:   modelHandler,
		wechatHandler:  wechatHandler,
		cronHandler:    cronHandler,
		dispatcher:     disp,
	}, nil
}

// Start begins all Gateway subsystems: agent loop, cron, hot-reload watcher,
// outbound router, and all channels. It does NOT start the HTTP server.
func (g *Gateway) Start() error {
	ctx := context.Background()
	g.startedAt = time.Now()

	g.agentLoop.Start(ctx)

	// Start cron scheduler
	g.cronSvc.Start()
	logger.InfoCF("gateway", "cron scheduler started", map[string]any{"jobs": len(g.cronSvc.Jobs())})

	cfg := config.Get()
	if cfg.Server.HotReload {
		g.hotReloadW = hotreload.NewWatcher(g.configPath, g.orchestrator.Reload)
		g.hotReloadW.Start()
		logger.InfoCF("gateway", "config hot-reload enabled (polling every 2s)", nil)
	}

	g.outboundCtx, g.outboundCancel = context.WithCancel(ctx)
	g.outboundWg.Add(1)
	go g.routeOutbound()

	if err := g.channelMgr.StartAll(ctx); err != nil {
		return fmt.Errorf("gateway: start channels: %w", err)
	}

	return nil
}

// Stop gracefully stops all Gateway subsystems in reverse dependency order.
func (g *Gateway) Stop(ctx context.Context) error {
	// 1. Stop hot-reload watcher (if any)
	if g.hotReloadW != nil {
		g.hotReloadW.Stop()
	}

	// 2. Stop cron scheduler
	g.cronSvc.Stop()
	logger.InfoCF("gateway", "cron scheduler stopped", nil)

	// 3. Cancel outbound router (drains bus)
	if g.outboundCancel != nil {
		g.outboundCancel()
	}
	g.outboundWg.Wait()
	g.bus.Close()

	// 4. Stop agent loop
	g.agentLoop.Stop()

	// 5. Stop channels
	if err := g.channelMgr.StopAll(ctx); err != nil {
		logger.ErrorCF("gateway", "channel manager stop error", map[string]any{"error": err.Error()})
	}

	return nil
}

// Reload triggers a hot-reload of the configuration via the orchestrator.
func (g *Gateway) Reload() error {
	return g.orchestrator.Reload()
}

// Status returns a snapshot of the Gateway's runtime state.
func (g *Gateway) Status() GatewayStatus {
	st := GatewayStatus{
		Uptime:   time.Since(g.startedAt).Round(time.Second).String(),
		CronJobs: len(g.cronSvc.Jobs()),
	}

	cfg := config.Get()
	if active, err := cfg.ActiveModelConfig(); err == nil {
		st.ActiveModel = active.ModelName
	}

	// Collect channel statuses
	// channel.Manager doesn't expose iteration, so use known names
	known := []string{"feishu", "wechat"}
	for _, name := range known {
		if rawCh, ok := g.channelMgr.GetChannel(name); ok {
			st.Channels = append(st.Channels, ChannelStatus{
				Name:    name,
				Running: rawCh.IsRunning(),
			})
		}
	}

	return st
}

// routeOutbound reads agent outbound messages from the bus and routes them
// to the appropriate channel via the Manager's per-channel worker queue.
func (g *Gateway) routeOutbound() {
	defer g.outboundWg.Done()
	for {
		select {
		case msg, ok := <-g.bus.OutboundChan():
			if !ok {
				return
			}
			if err := g.channelMgr.RouteOutbound(g.outboundCtx, channel.OutboundMessage{
				Channel: msg.Channel,
				ChatID:  msg.ChatID,
				Text:    msg.Text,
			}); err != nil {
				if err == channel.ErrNotRunning {
					logger.WarnCF("gateway", "no active worker for outbound message", map[string]any{
						"channel": msg.Channel,
						"chat_id": msg.ChatID,
					})
				} else {
					logger.ErrorCF("gateway", "route outbound failed", map[string]any{
						"channel": msg.Channel,
						"chat_id": msg.ChatID,
						"error":   err.Error(),
					})
				}
			}
		case <-g.outboundCtx.Done():
			return
		}
	}
}

// ---------------------------------------------------------------------------
// Accessor methods — expose Gateway internals to the HTTP layer.
// ---------------------------------------------------------------------------

func (g *Gateway) FeishuHandler() *handler.FeishuHandler     { return g.feishuHandler }
func (g *Gateway) ModelHandler() *handler.ModelHandler        { return g.modelHandler }
func (g *Gateway) WechatHandler() *handler.WechatHandler      { return g.wechatHandler }
func (g *Gateway) CronHandler() *handler.CronHandler          { return g.cronHandler }
func (g *Gateway) Orchestrator() *hotreload.Orchestrator      { return g.orchestrator }
func (g *Gateway) Dispatcher() *tool.Dispatcher               { return g.dispatcher }

// GetWebhookHandlers returns handler RegisterRoutesFunc values that need
// Gateway-internal state (feishu, wechat, model, cron). The HTTP layer
// mounts these alongside app-level handlers.
func (g *Gateway) GetWebhookHandlers() []func(api *gin.RouterGroup) {
	return []func(api *gin.RouterGroup){
		g.feishuHandler.RegisterRoutes,
		g.modelHandler.RegisterRoutes,
		g.wechatHandler.RegisterRoutes,
		g.cronHandler.RegisterRoutes,
	}
}
