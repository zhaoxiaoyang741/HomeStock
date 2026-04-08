# Home Inventory OpenClaw Plugin

家用物料管理插件，基于 OpenClaw 工具插件规范实现。支持通过自然语言和 slash command 管理食材、日用品库存。

## 功能

- `add_item`：添加物料，支持位置、分类、过期日期、备注。
- `remove_item`：消耗部分数量或整条删除库存。
- `query_items`：按位置、分类、关键字、临期状态查询库存。
- `update_item`：更新数量、过期时间、备注或位置。

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

示例：

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

## 使用示例

自然语言：

- “帮我添加 5 个土豆放冰箱，2026-04-30 过期”
- “冰箱里还有什么调料”
- “把土豆改成 3 个”
- “土豆吃完了”

Slash command：

```bash
/add_item 土豆 5个 冰箱
/remove_item 土豆 2
/query_items 冰箱 调料
/update_item 牛奶 2026-05-01
```

## 推荐模型

根据 OpenClaw 官方测试文档，做自然语言 tool calling 联调时，优先使用官方列为 baseline 的模型：

- `openai/gpt-5.4`
- `anthropic/claude-opus-4-6`
- `google/gemini-3-flash-preview`
- `zai/glm-4.7`
- `minimax/MiniMax-M2.7`

参考文档：

- `https://docs.openclaw.ai/help/testing`
- `https://docs.openclaw.ai/models`
- `https://docs.openclaw.ai/providers`

如果当前环境只能使用 `kimi/kimi-code` 或 `moonshot/kimi-k2.5`，建议优先验证 slash command 路径，例如 `/add_item 土豆 5个 冰箱`。在部分模型或 provider 接法下，自然语言可能会把 tool call 标记直接输出为文本，而不是由宿主真正执行工具。

## 调试

```bash
openclaw tools call add_item '{"name":"土豆","quantity":5,"location":"冰箱","expire_at":"2026-04-30"}'
openclaw tools call query_items '{"keyword":"土","expiring_soon":true}'
openclaw tools call update_item '{"name":"土豆","quantity":3}'
openclaw tools call remove_item '{"name":"土豆","quantity":1}'
```

## 构建、测试、打包

```bash
pnpm build
pnpm test
npm pack
```

远程部署脚本保留在 [scripts/deploy-plugin.ps1](/e:/pro-tmp/agent/openclaw-plugin/scripts/deploy-plugin.ps1)。

## 已知限制

- 后端默认监听端口已切到 `8888`；若未显式配置，插件默认连 `http://localhost:8888`。
- 后端 `expire_at` 仅接受 RFC3339；插件已把 `YYYY-MM-DD` 自动转换。
- 后端 `/api/v1/items` 目前只原生支持 `location` 和 `category_id` 过滤；`keyword` 与 `expiring_soon` 由插件本地过滤。
- 查询时的分类名需要插件先解析成 `category_id`。
- 当前未写入 `compat/build` 版本块，避免在未核实宿主 OpenClaw 版本前声明错误兼容范围。
- 自然语言工具调用的稳定性强依赖宿主模型与 provider；如果出现 `<|tool_call_begin|>` 一类标记被直接回显，通常说明模型没有成功走结构化 tool calling 链路。

## 后端 TODO

- 原生支持 `keyword` 查询。
- 原生支持 `expiring_soon` 查询。
- 原生支持按分类名查询，或提供分类名解析接口。
- 可选支持直接接收 `YYYY-MM-DD` 日期输入。
