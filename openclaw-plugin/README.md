# Home Inventory OpenClaw Plugin

家庭物料管理插件，基于 OpenClaw 工具插件规范实现。插件已切换到 HomeStock 当前的三表库存模型：

- `materials`：物料主数据
- `stock_lots`：真实库存批次
- `stock_movements`：库存流水

## 功能

- `inbound_stock`：新增一条入库批次，支持位置、分类、过期日期、备注。
- `consume_material`：按物料消耗库存，后端自动按批次扣减。
- `query_inventory`：按位置、分类、关键字、临期状态查询库存。
- `update_stock_lot`：更新单个批次的库存、过期时间、备注或位置。
- `check_homestock_service`：检查当前配置的 HomeStock 后端是否可用。

## 安装

```bash
pnpm install
pnpm build
npm pack
openclaw plugins install ./home-inventory-plugin-1.0.0.tgz
```

安装后可用以下命令确认：

```bash
openclaw plugins list
openclaw skills list
```

## 配置

`baseUrl` 的解析优先级如下：

1. `~/.openclaw/openclaw.json` 中 `plugins.entries.home-inventory.config.baseUrl`
2. 环境变量 `INVENTORY_API_URL`
3. 默认值 `http://localhost:8888`

### 基础配置示例

```json
{
  "plugins": {
    "entries": {
      "home-inventory": {
        "enabled": true,
        "config": {
          "baseUrl": "http://192.168.18.13:8888"
        }
      }
    }
  }
}
```

### tools.alsoAllow 配置示例

如果宿主启用了 `tools.alsoAllow` 白名单，需要把本插件的工具名一并加入，否则插件安装成功后，宿主仍可能无法调用这些工具。

```json
{
  "tools": {
    "profile": "messaging",
    "alsoAllow": [
      "inbound_stock",
      "consume_material",
      "query_inventory",
      "update_stock_lot",
      "check_homestock_service"
    ]
  }
}
```

若你本身还启用了飞书等其他工具，请把上面 5 个工具追加到原有 `alsoAllow` 数组里，不要覆盖掉已有内容。

### 推荐：专用库存 Agent

对于“冰箱里还有什么”“还有没有土豆”这类查询，推荐在 OpenClaw 中单独创建库存专用 agent，并把飞书入口绑定到它。这样可以把 skill、workspace 和系统提示全部收紧到 HomeStock 领域，明显降低被当成闲聊的概率。

如果你的 OpenClaw 实际运行配置是远端主机上的 `~/.openclaw/openclaw.json`，直接参考示例文件：

- [examples/精简版本.jsonc](/e:/pro-tmp/agent/openclaw-plugin/examples/%E7%B2%BE%E7%AE%80%E7%89%88.jsonc)
- [examples/openclaw.inventory-agent.example.jsonc](/e:/pro-tmp/agent/openclaw-plugin/examples/openclaw.inventory-agent.example.jsonc)

`精简版本.jsonc` 只保留 home-inventory 组件接入需要改的字段，适合直接按块合并。
完整示例保留给你做字段对照。两者都不是给插件源码目录直接读取的，而是给你手动合并到远端 `~/.openclaw/openclaw.json` 用的。

最小配置思路：

```yaml
agents:
  defaults:
    workspace: ./.openclaw/workspaces/main
    agentDir: ./.openclaw/agents/main/agent
    model: "anthropic/claude-haiku-4-5"
  list:
    - id: main
      default: true
    - id: inventory
      workspace: ./.openclaw/workspaces/inventory
      agentDir: ./.openclaw/agents/inventory/agent
      model: "anthropic/claude-sonnet-4-5"
      skills:
        - inbound_stock
        - consume_material
        - query_inventory
        - update_stock_lot
        - check_homestock_service
bindings:
  - agentId: inventory
    match:
      channel: feishu
      accountId: "*"
```

注意：
- auth profile 是按 agent 隔离的。若新增 `inventory` agent 后提示缺少 API key，请运行 `openclaw agents add inventory`，或把主 agent 的 `auth-profiles.json` 复制到 `inventory` 的 `agentDir`。
- 不要复用同一个 `agentDir` 给多个 agent，否则会发生会话或认证冲突。
- 对于 `OpenClaw 2026.4.2`，`agents.list[].systemPromptOverride` 可能还不被接受。若出现 `Unrecognized key: "systemPromptOverride"`，请直接删除该字段，改用 `skills` 白名单和 `workspace` 下的 `AGENTS.md` / `IDENTITY.md` 约束 agent。

## 使用示例

自然语言：

- “帮我入库 5 个土豆放冰箱，2026-04-30 过期”
- “冰箱里还有什么调料”
- “回复某人：冰箱里还有什么”
- “还有没有土豆”
- “把土豆改成 3 袋”
- “牛奶喝完了”
- “帮我确认目前 HomeStock 服务是否正常”

Slash command：

```bash
/inbound_stock 土豆 5个 冰箱
/consume_material 土豆 2
/query_inventory 冰箱 调料
/update_stock_lot 牛奶 2026-05-01
/check_homestock_service
```

## 调试

```bash
openclaw tools call inbound_stock '{"name":"土豆","quantity":5,"location":"冰箱","expire_at":"2026-04-30"}'
openclaw tools call query_inventory '{"keyword":"土豆","expiring_soon":true}'
openclaw tools call update_stock_lot '{"name":"土豆","quantity":3}'
openclaw tools call consume_material '{"name":"土豆","quantity":1}'
openclaw tools call check_homestock_service '{}'
```

## 后端接口映射

插件当前基于以下接口：

- `GET /api/v1/health`
- `GET /api/v1/materials`
- `GET /api/v1/materials/:id`
- `POST /api/v1/materials/:id/consume`
- `GET /api/v1/stock-lots`
- `POST /api/v1/stock-lots/inbound`
- `PUT /api/v1/stock-lots/:id`
- `POST /api/v1/stock-lots/:id/adjust`
- `GET /api/v1/categories`
- `POST /api/v1/categories`

## 已知限制

- 插件的日期输入仍使用 `YYYY-MM-DD`，再由插件转换成 RFC3339 发送给后端。
- `update_stock_lot` 只会修改单个批次；若同一物料存在多个批次，用户需要先在 Web 中明确批次。
- `consume_material` 是按物料消耗，最终扣减到哪些批次由后端自动决定。
- 自然语言调用稳定性仍依赖宿主模型和 provider；若出现 tool call 标记直接回显，通常是宿主没有成功走结构化工具调用链路。
- 正式兜底入口是 slash command：`/query_inventory ...`。它配合 `command-dispatch: tool` 可以直接绕过自然语言匹配。

## 构建、测试、打包

```bash
pnpm build
pnpm test
npm pack
```

远程部署脚本保留在 [scripts/deploy-plugin.ps1](/e:/pro-tmp/agent/openclaw-plugin/scripts/deploy-plugin.ps1)。
