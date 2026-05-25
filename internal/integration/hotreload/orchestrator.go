package hotreload

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/zhaoxiaoyang741/HomeStock/internal/integration/agent"
	"github.com/zhaoxiaoyang741/HomeStock/internal/integration/channel/feishu"
	"github.com/zhaoxiaoyang741/HomeStock/internal/integration/channel/wechat"
	"github.com/zhaoxiaoyang741/HomeStock/internal/handler"
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
	feishuCh   *feishu.FeishuChannel
	oauthSvc   *feishu.OAuthService
	modelHnd   *handler.ModelHandler
	wechatCh   *wechat.WechatChannel
	channelMgr *channel.Manager

	lastCfg        atomic.Value // stores *config.Config 鈥?the last successfully applied config
	reloading      atomic.Bool  // prevents concurrent reloads
	lastReloadTime atomic.Value // stores time.Time 鈥?when the last reload completed
}

// NewOrchestrator creates an Orchestrator and seeds it with the initial config.
func NewOrchestrator(
	configPath string,
	agentLoop *agent.AgentLoop,
	feishuCh *feishu.FeishuChannel,
	oauthSvc *feishu.OAuthService,
	modelHnd *handler.ModelHandler,
	wechatCh *wechat.WechatChannel,
	channelMgr *channel.Manager,
) *Orchestrator {
	o := &Orchestrator{
		configPath: configPath,
		agentLoop:  agentLoop,
		feishuCh:   feishuCh,
		oauthSvc:   oauthSvc,
		modelHnd:   modelHnd,
		wechatCh:   wechatCh,
		channelMgr: channelMgr,
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

	// 3. Channel hot-reconfigure via factory-based generic loop
	feishuCfg, _ := newCfg.FeishuConfig()
	wechatCfg, _ := newCfg.WechatConfig()

	for name, changed := range diff.ChannelsChanged {
		if !changed {
			continue
		}

		raw, hasNew := newCfg.Channels[name]

		switch name {
		case "feishu":
			// Feishu has OAuth dependencies 鈥?use the existing path
			if o.feishuCh != nil && o.oauthSvc != nil {
				ctx := context.Background()

				if err := o.feishuCh.Reconfigure(ctx, feishuCfg.AppID, feishuCfg.AppSecret, feishuCfg.Enabled); err != nil {
					logger.ErrorCF("hotreload", "feishu reconfigure failed", map[string]any{"error": err.Error()})
				}

				o.oauthSvc.UpdateCredentials(feishuCfg.AppID, feishuCfg.AppSecret)
				_ = o.oauthSvc.ClearAuth(ctx)

				if feishuCfg.Enabled {
					if _, exists := o.channelMgr.GetChannel("feishu"); !exists {
						o.channelMgr.AddChannel(o.feishuCh)
					}
				} else {
					o.channelMgr.RemoveChannel("feishu")
				}
			}
			logger.InfoCF("hotreload", "feishu channel reconfigured", map[string]any{
				"enabled": feishuCfg.Enabled,
			})

		case "wechat":
			// WeChat has login state -- restart to pick up config changes (token etc.)
			if wechatCfg.Enabled {
				ctx := context.Background()
				if o.wechatCh != nil {
					if o.wechatCh.IsRunning() {
						_ = o.wechatCh.Stop(ctx)
					}
					o.wechatCh.SetConfig(wechatCfg)
					if err := o.wechatCh.Start(ctx); err != nil {
						logger.ErrorCF("hotreload", "wechat start failed", map[string]any{"error": err.Error()})
					}
				}
				if _, exists := o.channelMgr.GetChannel("wechat"); !exists {
					o.channelMgr.AddChannel(o.wechatCh)
				}
			} else {
				if o.wechatCh != nil && o.wechatCh.IsRunning() {
					ctx := context.Background()
					if err := o.wechatCh.Stop(ctx); err != nil {
						logger.ErrorCF("hotreload", "wechat stop failed", map[string]any{"error": err.Error()})
					}
				}
				o.channelMgr.RemoveChannel("wechat")
			}
			logger.InfoCF("hotreload", "wechat channel reconfigured", map[string]any{
				"enabled": wechatCfg.Enabled,
			})
		default:
			// Generic channel: stop old 鈫?recreate from factory 鈫?start new
			if o.channelMgr != nil {
				ctx := context.Background()

				// Stop old instance
				if ch, ok := o.channelMgr.GetChannel(name); ok {
					_ = ch.Stop(ctx)
					o.channelMgr.RemoveChannel(name)
				}

				if !hasNew {
					continue // channel was removed
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
				if setter, ok := ch.(channel.InboundHandlerSetter); ok {
					setter.SetInboundHandler(func(ctx context.Context, msg channel.InboundMessage) {
						// inbound routing placeholder 鈥?wired via manager in normal flow
					})
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

