package hotreload

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/internal/handler"
	"github.com/zhaoxiaoyang741/HomeStock/internal/integration/agent"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/channel"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/llm"
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
	modelHnd   *handler.ModelHandler
	channelMgr *channel.Manager

	// configChangeHandlers maps channel name to its config change handler,
	// registered by handler factories at startup.
	configChangeHandlers map[string]channel.ConfigChangeHandler

	lastCfg        atomic.Value // stores *config.Config — the last successfully applied config
	reloading      atomic.Bool  // prevents concurrent reloads
	lastReloadTime atomic.Value // stores time.Time — when the last reload completed
}

// NewOrchestrator creates an Orchestrator and seeds it with the initial config.
// The feishuCh, oauthSvc, and wechatCh parameters have been removed — channel
// lifecycle is now managed via ConfigChangeHandler registrations.
func NewOrchestrator(
	configPath string,
	agentLoop *agent.AgentLoop,
	modelHnd *handler.ModelHandler,
	channelMgr *channel.Manager,
) *Orchestrator {
	o := &Orchestrator{
		configPath:           configPath,
		agentLoop:            agentLoop,
		modelHnd:             modelHnd,
		channelMgr:           channelMgr,
		configChangeHandlers: make(map[string]channel.ConfigChangeHandler),
	}
	o.lastCfg.Store(config.Get())
	o.lastReloadTime.Store(time.Now())
	return o
}

// RegisterConfigChangeHandler registers a handler for runtime config changes.
// The handler is called from Reload() when its channel's config changes.
func (o *Orchestrator) RegisterConfigChangeHandler(hdl channel.ConfigChangeHandler) {
	if hdl != nil {
		o.configChangeHandlers[hdl.Name()] = hdl
	}
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
		active, err := newCfg.ActiveModelConfig()
		if err != nil {
			logger.ErrorCF("hotreload", "no active model after config change, keeping old provider", map[string]any{
				"error": err.Error(),
			})
		} else {
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

	// 3. Channel hot-reconfigure via registered ConfigChangeHandlers
	for name, changed := range diff.ChannelsChanged {
		if !changed {
			continue
		}

		if hdl, ok := o.configChangeHandlers[name]; ok {
			// Channel-specific handler knows how to reconfigure its channel
			if err := hdl.HandleConfigChange(context.Background(), oldCfg, newCfg); err != nil {
				logger.ErrorCF("hotreload", "config change handler failed", map[string]any{
					"name":  name,
					"error": err.Error(),
				})
			} else {
				logger.InfoCF("hotreload", "channel reconfigured via handler", map[string]any{
					"name": name,
				})
			}
			continue
		}

		// Generic channel (no registered handler): stop old → recreate from factory → start new
		raw, hasNew := newCfg.Channels[name]

		if o.channelMgr != nil {
			ctx := context.Background()

			// Stop old instance
			if ch, ok := o.channelMgr.GetChannel(name); ok {
				_ = ch.Stop(ctx)
				o.channelMgr.RemoveChannel(name)
			}

			if !hasNew {
				continue // channel was removed from config
			}

			// Re-create from factory
			factory, ok := channel.GetFactory(name)
			if !ok {
				continue
			}
			ch, err := factory(raw)
			if err != nil || ch == nil {
				continue
			}
			o.channelMgr.AddChannel(ch)
			if err := ch.Start(ctx); err != nil {
				logger.ErrorCF("hotreload", "channel start failed after reload", map[string]any{
					"name":  name,
					"error": err.Error(),
				})
			}
		}
	}

	// 4. Warn about changes that require a full restart
	if diff.PortChanged {
		logger.WarnCF("hotreload", "server.port changed, restart required to take effect", map[string]any{
			"old": oldCfg.Server.Port,
			"new": newCfg.Server.Port,
		})
	}
	if diff.DBChanged {
		logger.WarnCF("hotreload", "database config changed, restart required to take effect", map[string]any{
			"old_driver": oldCfg.Database.Driver,
			"new_driver": newCfg.Database.Driver,
		})
	}

	o.lastCfg.Store(newCfg)
	o.lastReloadTime.Store(time.Now())
	return nil
}
