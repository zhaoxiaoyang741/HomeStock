# 飞书 → OpenClaw → HomeStock 集成方案

## 文档索引

- [后端分层重构规划](./backend-layering-refactor-plan.md)

## 概述

通过飞书向 OpenClaw AI Agent 发送自然语言指令，OpenClaw 理解意图后调用 HomeStock 插件，插件调用后端 REST API 完成物料增删查操作，结果通过飞书回复给用户。

---

## 系统架构

```
飞书用户（发送消息："帮我添加5个土豆放冰箱"）
  │
  │  WebSocket 长连接（OpenClaw 内置，无需公网 IP）
  ▼
OpenClaw AI Agent
  │  LLM 解析意图 → tool_use: add_item({name:"土豆", quantity:5, location:"冰箱"})
  ▼
openclaw-plugin（TypeScript，本目录）
  │  HTTP POST /api/v1/items
  ▼
HomeStock Go 后端（localhost:8080）
  │  写入 SQLite，返回 Item 对象
  ▼
openclaw-plugin → OpenClaw → 飞书回复："✅ 已添加 土豆 5个，存放在【冰箱】"
```

---

## 组件说明

| 组件 | 位置 | 职责 |
|------|------|------|
| HomeStock 后端 | `cmd/server/` | REST API，物料 CRUD，数据持久化 |
| openclaw-plugin | `openclaw-plugin/` | 连接 OpenClaw 与后端，实现 3 个 tool |
| OpenClaw 配置 | `openclaw.config.yaml` | 飞书凭证、插件路径、LLM 配置 |

---

## 支持的操作（Tool）

| Tool | 触发示例 | 说明 |
|------|---------|------|
| `add_item` | "帮我加5个土豆放冰箱"、"买了两瓶酱油" | 向库存添加物料 |
| `remove_item` | "刚用掉一个土豆"、"鸡蛋吃完了" | 消耗或删除物料 |
| `query_items` | "冰箱里有什么"、"还有什么快过期的" | 查询库存 |

---

## 飞书应用配置

> 详见 `.docs/07-feishu-setup.md`

1. 打开[飞书开放平台](https://open.feishu.cn/app) → 创建企业自建应用
2. **凭证与基础信息** → 复制 **App ID** 和 **App Secret**
3. **权限管理** → 申请以下权限：
   - `im:message`（获取与发送单聊、群组消息）
   - `im:message:send_as_bot`（以应用的身份发消息）
   - `im:message.p2p_msg:readonly`（读取用户发给机器人的单聊消息）
   - `im:message.group_at_msg:readonly`（接收群聊中 @机器人 消息，可选）
4. **事件与回调** → 添加事件 `im.message.receive_v1` → 订阅方式选**使用长连接**（无需公网 IP）
5. **测试企业与成员** → 添加自己 → 无需等待审批即可测试

---

## 开发环境部署

### 前提条件

- Go 1.22+（后端）
- Node.js 18+ + pnpm（插件）
- OpenClaw CLI 已安装（`npm install -g @openclaw/cli` 或参考 OpenClaw 官方文档）

### 步骤

```bash
# 1. 启动 HomeStock 后端
cd /path/to/agent
go run ./cmd/server
# 输出：server started on :8080

# 2. 构建插件
cd openclaw-plugin
pnpm install
pnpm build
# 输出：dist/index.js 生成

# 3. 填写配置
# 编辑 openclaw.config.yaml，替换：
#   channels.feishu.app_id
#   channels.feishu.app_secret
#   llm.api_key

# 4. 启动 OpenClaw
cd ..
openclaw start --config openclaw.config.yaml
# 期望输出：
# ✓ Feishu channel connected (WebSocket)
# ✓ Plugin loaded: home-inventory
```

### 环境变量（可选，覆盖插件配置）

```bash
export INVENTORY_API_URL=http://localhost:8080  # 后端地址，默认值即为此
```

---

## 测试方法

### 第一层：验证后端 API

```bash
# 添加物料
curl -X POST http://localhost:8080/api/v1/items \
  -H "Content-Type: application/json" \
  -d '{"name":"土豆","quantity":5,"unit":"个","location":"冰箱"}'
# 期望：HTTP 201 + Item JSON

# 查询物料
curl http://localhost:8080/api/v1/items
# 期望：{"items":[...],"total":1}
```

### 第二层：验证插件构建

```bash
cd openclaw-plugin
pnpm build
# 期望：无 TypeScript 错误，dist/index.js 存在
```

### 第三层：端到端飞书测试

在飞书中私聊机器人，发送以下消息验证：

| 发送消息 | 期望回复 |
|---------|---------|
| `帮我添加5个土豆放冰箱` | `✅ 已添加 土豆 5个，存放在【冰箱】` |
| `冰箱里有什么` | `📦 【冰箱】库存（共 1 项）：• 土豆 5个` |
| `刚用掉一个土豆` | `✅ 已消耗 土豆 1个，剩余 4个` |

同时用 `curl GET /api/v1/items` 确认数据库记录与回复一致。

---

## 常见问题

### Q: OpenClaw 内存占用多少？

约 200-500MB（含 LLM）。若树莓派内存有限，可将 OpenClaw 部署在 PC 上，HomeStock 后端单独运行在树莓派上，通过局域网 IP 互连（修改 `openclaw.config.yaml` 中插件的 `INVENTORY_API_URL`）。

### Q: 国内无法访问 Anthropic/OpenAI API？

修改 `openclaw.config.yaml` 使用本地 Ollama：

```yaml
llm:
  provider: ollama
  base_url: http://localhost:11434
  model: qwen2.5:7b  # 中文理解较好的本地模型
```

### Q: 机器人收不到消息？

1. 检查飞书应用权限是否已申请
2. 确认已加入测试企业或应用已正式发布
3. 查看 OpenClaw 日志：`openclaw start --log-level debug`
4. 私聊机器人无需 @，群聊需要 @机器人

### Q: 如何添加更多 Tool（如查询过期提醒）？

1. 在 `openclaw-plugin/src/tools/` 新增 tool 文件
2. 在 `openclaw.plugin.json` 的 `tools` 数组中添加声明
3. 在 `src/index.ts` 的 `handleTool` switch 中添加 case
4. 在 `skills/SKILL.md` 中补充触发描述
5. `pnpm build` 重新构建，重启 OpenClaw
