package gateway

import (
	"context"
	"sync"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/internal/integration/hotreload"
	"github.com/zhaoxiaoyang741/HomeStock/internal/outbound"
	gormrepo "github.com/zhaoxiaoyang741/HomeStock/internal/repository/gorm"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/cron"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

// Gateway manages the core subsystems: cron scheduler, outbound event dispatcher,
// and optional config hot-reload.
type Gateway struct {
	configPath string

	// Outbound event dispatcher
	outboundMgr *outbound.Manager

	// Hot-reload
	orchestrator *hotreload.Orchestrator
	hotReloadW   *hotreload.Watcher

	// Cron scheduler
	cronSvc *cron.Service

	startedAt time.Time
	mu        sync.Mutex
}

// New creates a Gateway and its subsystems.
func New(
	cfg *config.Config,
	configPath string,
	uow *gormrepo.UnitOfWork,
) (*Gateway, error) {
	// -----------------------------------------------------------------------
	// Outbound manager
	// -----------------------------------------------------------------------
	outboundMgr := outbound.NewManager(&cfg.Outbound)

	// -----------------------------------------------------------------------
	// Hot-reload orchestrator
	// -----------------------------------------------------------------------
	orch := hotreload.NewOrchestrator(configPath)

	// -----------------------------------------------------------------------
	// Cron
	// -----------------------------------------------------------------------
	cronSvc := initCron(uow, cfg.Cron, outboundMgr)

	return &Gateway{
		configPath:   configPath,
		outboundMgr:  outboundMgr,
		orchestrator: orch,
		cronSvc:      cronSvc,
	}, nil
}

// Start begins Gateway subsystems: cron scheduler and hot-reload watcher.
func (g *Gateway) Start() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.startedAt = time.Now()

	// Start cron scheduler
	g.cronSvc.Start()
	logger.InfoCF("gateway", "cron scheduler started", map[string]any{"jobs": len(g.cronSvc.Jobs())})

	cfg := config.Get()
	if cfg.Server.HotReload {
		g.hotReloadW = hotreload.NewWatcher(g.configPath, g.orchestrator.Reload)
		g.hotReloadW.Start()
		logger.InfoCF("gateway", "config hot-reload enabled (polling every 2s)", nil)
	}

	return nil
}

// Stop gracefully stops all Gateway subsystems.
func (g *Gateway) Stop(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// 1. Stop hot-reload watcher
	if g.hotReloadW != nil {
		g.hotReloadW.Stop()
	}

	// 2. Stop cron scheduler
	g.cronSvc.Stop()
	logger.InfoCF("gateway", "cron scheduler stopped", nil)

	return nil
}

// Reload triggers a hot-reload of the configuration via the orchestrator.
func (g *Gateway) Reload() error {
	return g.orchestrator.Reload()
}

// Status returns a snapshot of the Gateway's runtime state.
func (g *Gateway) Status() GatewayStatus {
	g.mu.Lock()
	uptime := time.Since(g.startedAt).Round(time.Second).String()
	cronJobs := len(g.cronSvc.Jobs())
	g.mu.Unlock()

	return GatewayStatus{
		Uptime:   uptime,
		CronJobs: cronJobs,
	}
}

// OutboundManager returns the outbound event manager.
func (g *Gateway) OutboundManager() *outbound.Manager { return g.outboundMgr }
