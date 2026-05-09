package hotreload

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/internal/agent"
	"github.com/zhaoxiaoyang741/HomeStock/internal/channel/feishu"
	"github.com/zhaoxiaoyang741/HomeStock/internal/handler"
	"github.com/zhaoxiaoyang741/HomeStock/internal/llm"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

// Orchestrator is the single entry point for runtime config hot-reload.
//
// It is triggered either by a file watcher (config.json changed on disk) or
// by an explicit HTTP POST /reload call.  It diffs the new config against
// the last known good config and applies per-component updates.
type Orchestrator struct {
	configPath string
	agentLoop  *agent.AgentLoop
	feishuCh   *feishu.FeishuChannel
	oauthSvc   *feishu.OAuthService
	modelHnd   *handler.ModelHandler

	lastCfg        atomic.Value // stores *config.Config — the last successfully applied config
	reloading      atomic.Bool  // prevents concurrent reloads
	lastReloadTime atomic.Value // stores time.Time — when the last reload completed
}

// NewOrchestrator creates an Orchestrator and seeds it with the initial config.
func NewOrchestrator(
	configPath string,
	agentLoop *agent.AgentLoop,
	feishuCh *feishu.FeishuChannel,
	oauthSvc *feishu.OAuthService,
	modelHnd *handler.ModelHandler,
) *Orchestrator {
	o := &Orchestrator{
		configPath: configPath,
		agentLoop:  agentLoop,
		feishuCh:   feishuCh,
		oauthSvc:   oauthSvc,
		modelHnd:   modelHnd,
	}
	o.lastCfg.Store(config.Get())
	o.lastReloadTime.Store(time.Now())
	return o
}

// LastReloadTime returns the time of the last successful config reload.
func (o *Orchestrator) LastReloadTime() time.Time {
	t, ok := o.lastReloadTime.Load().(time.Time)
	if !ok {
		return time.Time{}
	}
	return t
}

// Reload reloads the config file from disk, diffs it against the last known
// config, and applies all detected changes to running services.
func (o *Orchestrator) Reload() error {
	if !o.reloading.CompareAndSwap(false, true) {
		logger.WarnCF("hotreload", "reload already in progress, skipping", nil)
		return nil
	}
	defer o.reloading.Store(false)

	newCfg, err := config.Load(o.configPath)
	if err != nil {
		return err
	}

	oldCfg := o.lastCfg.Load().(*config.Config)
	diff := CalcDiff(oldCfg, newCfg)

	// 1. Logger hot-update (stateless, safe)
	if diff.LogChanged {
		logger.SetLevel(logger.LogLevel(newCfg.Log.Level))
		if newCfg.Log.Path != "" && newCfg.Log.Path != oldCfg.Log.Path {
			if err := logger.EnableFileLogging(newCfg.Log.Path); err != nil {
				logger.WarnCF("hotreload", "failed to update log path", map[string]any{
					"path":  newCfg.Log.Path,
					"error": err.Error(),
				})
			}
		}
		// Re-update level again after EnableFileLogging which resets it
		logger.SetLevel(logger.LogLevel(newCfg.Log.Level))
		logger.InfoCF("hotreload", "log config hot-updated", map[string]any{
			"level": newCfg.Log.Level,
			"path":  newCfg.Log.Path,
		})
	}

	// 2. Model provider hot-swap
	if diff.ModelChanged {
		active := firstEnabledModel(newCfg.ModelList)
		if active != nil {
			provider, err := llm.NewProvider(*active)
			if err != nil {
				logger.ErrorCF("hotreload", "new provider creation failed, keeping old provider", map[string]any{
					"model_name": active.ModelName,
					"error":      err.Error(),
				})
			} else {
				o.agentLoop.SwapProvider(provider)
				o.modelHnd.SetActiveName(active.ModelName)

				logger.InfoCF("hotreload", "model provider hot-swapped", map[string]any{
					"model_name": active.ModelName,
					"provider":   active.Provider,
					"model":      active.Model,
				})
			}
		}
	}

	// 3. Feishu channel hot-reconfigure
	if diff.FeishuChanged {
		ctx := context.Background()
		fcCfg := newCfg.Channels.Feishu

		// The feishu channel reference may not be available in all builds
		if o.feishuCh != nil && o.oauthSvc != nil {
			if err := o.feishuCh.Reconfigure(ctx, fcCfg.AppID, fcCfg.AppSecret, fcCfg.Enabled); err != nil {
				logger.ErrorCF("hotreload", "feishu reconfigure failed", map[string]any{"error": err.Error()})
			}

			o.oauthSvc.UpdateCredentials(fcCfg.AppID, fcCfg.AppSecret)
			_ = o.oauthSvc.ClearAuth(ctx)
		}

		logger.InfoCF("hotreload", "feishu channel reconfigured", map[string]any{
			"enabled": fcCfg.Enabled,
		})
	}

	// 4. Warn about changes that require a full restart
	if diff.PortChanged {
		logger.WarnCF("hotreload", "server.port changed, restart required to take effect", map[string]any{
			"old": oldCfg.Server.Port,
			"new": newCfg.Server.Port,
		})
	}
	if diff.DatabaseChanged {
		logger.WarnCF("hotreload", "database config changed, restart required to take effect", map[string]any{
			"old_driver": oldCfg.Database.Driver,
			"new_driver": newCfg.Database.Driver,
		})
	}

	o.lastCfg.Store(newCfg)
	o.lastReloadTime.Store(time.Now())
	return nil
}
