package app

import (
	"errors"
	"fmt"

	"github.com/zhaoxiaoyang741/HomeStock/internal/agent"
	"github.com/zhaoxiaoyang741/HomeStock/internal/service"
	"github.com/zhaoxiaoyang741/HomeStock/internal/tool"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/bus"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/config"
	"github.com/zhaoxiaoyang741/HomeStock/pkg/llm"
)

const systemPrompt = `你是 HomeStock（变便）库存管理助手，可以通过飞书帮助用户管理家庭库存。
你可以帮助用户：
1. 查询库存情况
2. 新增物品入库
3. 消耗出库
4. 更新批次信息

每次操作前先确认用户意图，操作完成后反馈结果。
回复简洁友好。`

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

	agentLoop = agent.NewAgentLoop(msgBus, llmProvider, disp, systemPrompt)

	// Tool registration
	tool.RegisterInventoryTools(disp, &tool.InventoryTools{
		InventorySvc: inventorySvc,
		MaterialSvc:  materialSvc,
	})
	tool.RegisterHealthTool(disp, fmt.Sprintf("http://localhost:%s", port))

	defs := tool.InventoryToolDefinitions()
	defs = append(defs, tool.HealthToolDefinition())
	disp.SetDefinitions(defs)

	return
}
