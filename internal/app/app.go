package app

import (
	"context"
	"database/sql"
	"fmt"
	"sync"

	"gorm.io/gorm"

	"github.com/zhaoxiaoyang741/HomeStock/internal/agent"
	"github.com/zhaoxiaoyang741/HomeStock/internal/channel"
	"github.com/zhaoxiaoyang741/HomeStock/internal/hotreload"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/server"
)

// App is the top-level application container, owning all subsystems.
type App struct {
	configPath string
	server     *server.Server
	db         *gorm.DB
	sqlDB      *sql.DB

	// Agent system
	bus       *agent.MessageBus
	agentLoop *agent.AgentLoop

	// Channel system
	channelMgr *channel.Manager

	// Hot-reload
	orchestrator *hotreload.Orchestrator
	hotReloadW   *hotreload.Watcher

	// Outbound router lifecycle
	outboundCtx    context.Context
	outboundCancel context.CancelFunc
	outboundWg     sync.WaitGroup
}

// Start begins all subsystems: agent loop, hot-reload watcher, outbound router,
// channel manager, and the HTTP server (blocking).
func (a *App) Start() error {
	ctx := context.Background()
	a.agentLoop.Start(ctx)

	cfg := config.Get()
	if cfg.Server.HotReload {
		a.hotReloadW = hotreload.NewWatcher(a.configPath, a.orchestrator.Reload)
		a.hotReloadW.Start()
		logger.InfoCF("app", "config hot-reload enabled (polling every 2s)", nil)
	}

	a.outboundCtx, a.outboundCancel = context.WithCancel(ctx)
	a.outboundWg.Add(1)
	go a.routeOutbound()

	if err := a.channelMgr.StartAll(ctx); err != nil {
		return fmt.Errorf("app: start channels: %w", err)
	}

	return a.server.Start()
}

// Shutdown gracefully stops all subsystems in reverse dependency order.
func (a *App) Shutdown(ctx context.Context) error {
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

// Close releases the underlying database connection.
func (a *App) Close() error {
	if a == nil || a.sqlDB == nil {
		return nil
	}
	return a.sqlDB.Close()
}

// routeOutbound reads agent outbound messages from the bus and routes them
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
