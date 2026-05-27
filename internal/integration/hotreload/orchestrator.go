package hotreload

import (
	"sync/atomic"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/logger"
)

// Orchestrator handles runtime config hot-reload by diffing the new config
// against the last known good config and applying per-component updates.
type Orchestrator struct {
	configPath string

	lastCfg        atomic.Value // stores *config.Config
	reloading      atomic.Bool
	lastReloadTime atomic.Value // stores time.Time
}

// NewOrchestrator creates an Orchestrator and seeds it with the initial config.
func NewOrchestrator(configPath string) *Orchestrator {
	o := &Orchestrator{
		configPath: configPath,
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
// config, and applies all detected changes.
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
		logger.SetLevel(logger.LogLevel(newCfg.Log.Level))
		logger.InfoCF("hotreload", "log config hot-updated", map[string]any{
			"level": newCfg.Log.Level,
			"path":  newCfg.Log.Path,
		})
	}

	// 2. Outbound endpoints changed — log warning (outbound Manager reloads independently)
	if diff.OutboundChanged {
		logger.InfoCF("hotreload", "outbound endpoints changed, restart required to take effect", nil)
	}

	// 3. Auth changed — log warning (API keys are read at request time)
	if diff.AuthChanged {
		logger.InfoCF("hotreload", "auth config changed, restart required to take effect", nil)
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
