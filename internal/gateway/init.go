package gateway

import (
	"fmt"
	"time"

	appcron "github.com/zhaoxiaoyang741/HomeStock/internal/integration/cron"
	"github.com/zhaoxiaoyang741/HomeStock/internal/integration/agent"
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

const systemPrompt = `你是 HomeStock（变便）库存管理助手，可以通过飞书帮助用户管理家庭库存。你可以帮助用户：
1. 查询库存情况
2. 新增物品入库
3. 消耗出库
4. 更新批次信息

每次操作前先确认用户意图，操作完成后反馈结果。回复简洁友好。
== 批量输入指引 ==
用户可以一次输入多个物品，例如：「买了5斤苹果、1箱牛奶、一袋大米」。遇到这种情况，请分别调用入库/出库工具处理每个物品，每次调用一个物品。处理完所有物品后，汇总结果一次性回复用户。
== 确认为先 ==
如果用户一次提及多个物品，或者操作可能影响较大（如大量出库），先列出物品让用户自然确认（如"识别到苹果5斤、牛奶1箱，需要入库吗？"），
等用户回复确认后再调用工具执行。如果是单个物品的简单操作或查询类请求，直接执行不需要确认。`

// initAgent creates the LLM provider, message bus, dispatcher, and agent loop.
func initAgent(
	cfg *config.Config,
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
	modelCfg, err = cfg.ActiveModelConfig()
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("gateway: %w", err)
	}

	llmProvider, err = llm.NewProvider(*modelCfg)
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("gateway: create llm provider: %w", err)
	}

	msgBus = bus.NewMessageBus(0)
	disp = tool.NewDispatcher()

	nluEngine := agent.NewNluEngine(materialSvc)
	agentLoop = agent.NewAgentLoop(msgBus, llmProvider, disp, systemPrompt, nluEngine)

	return
}

// initCron creates the cron scheduler and registers the expiry stock notifier.
func initCron(uow *gormrepo.UnitOfWork, cfg config.CronConfig, channelMgr *channel.Manager) *cron.Service {
	svc := cron.New()

	if cfg.Enabled {
		interval, err := time.ParseDuration(cfg.ExpiryCheckPollInterval)
		if err != nil || interval <= 0 {
			interval = 6 * time.Hour
		}
		svc.Register(
			appcron.NewExpiringStockNotifier(uow, cfg.ExpiryCheckIntervalDays, channelMgr),
			cron.ScheduleDef{Interval: interval},
		)
		logger.InfoCF("gateway", "cron: registered expiry_notifier", map[string]any{
			"interval":    interval.String(),
			"expiry_days": cfg.ExpiryCheckIntervalDays,
		})
	}

	return svc
}
