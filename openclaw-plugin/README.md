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

## 使用示例

自然语言：

- “帮我入库 5 个土豆放冰箱，2026-04-30 过期”
- “冰箱里还有什么调料”
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

## 构建、测试、打包

```bash
pnpm build
pnpm test
npm pack
```

远程部署脚本保留在 [scripts/deploy-plugin.ps1](/e:/pro-tmp/agent/openclaw-plugin/scripts/deploy-plugin.ps1)。
