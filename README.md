# 变便 · HomeStock

家用物料管理系统 — 管理食材与日用品库存，跟踪保质期，临期提醒，减少浪费。

- **AI 入口**：通过飞书发送文字、语音或小票照片，即可完成入库、查询、出库
- **Web 管理**：响应式仪表盘与库存管理界面，支持物料分类、批次跟踪、库存流水
- **轻量部署**：支持树莓派 / NAS / PC，SQLite 或 PostgreSQL 按需切换

---

## 架构概览

```
用户入口
  ├── 飞书（文字 / 语音 / 图片） → Agent 编排 → LLM → 工具调度 → HTTP
  └── 浏览器（Web UI）           → React       → HTTP
                                    │
                                    ▼
                            Go REST API
                            ├── 物料 / 批次 CRUD
                            ├── 保质期扫描 & 推送
                            ├── 飞书 Webhook 通知
                            └── 配置热重载
                                    │
                                    ▼
                          SQLite / PostgreSQL
```

---

## 技术栈

| 模块 | 技术 |
|------|------|
| 后端 | Go、Gin、GORM、robfig/cron、zerolog |
| 前端 | React 19、TypeScript、Vite 8、Tailwind CSS 4 |
| AI Agent | Go AgentLoop + LLM Provider |
| 大模型 | OpenAI（GPT-4o 等）/ Ollama（Qwen2.5 等），支持运行时切换 |
| 渠道 | 飞书（OAuth 授权 / Webhook 推送）、可扩展 Channel 接口 |
| 数据 | SQLite（嵌入式）/ PostgreSQL |
| 交付 | 交叉编译单二进制、Docker Compose、Nginx 静态托管 |

---

## 特性

### 库存管理

- **物料主数据**：名称、规格、分类、默认单位
- **批次追踪**：每批独立记录数量、存放位置、购买日期、到期日期、备注
- **库存操作**：入库、消耗出库、数量调整、批次废弃
- **多维筛选**：按分类、存放位置、关键字搜索，批次详情联动展示
- **库存仪表盘**：总量统计、分类分布、临期看板、最近流水

### 保质期与提醒

- 自动标记 `normal` / `expiring` / `expired` 状态
- 定时扫描临期批次
- 飞书机器人推送提醒

### AI 自然语言交互

通过飞书 + AgentLoop + LLM，支持以下操作：

| 能力 | 说明 |
|------|------|
| 入库 | "买了 2 斤苹果，放在冰箱" |
| 出库 | "用了 3 个鸡蛋" |
| 查询 | "还剩多少牛奶？" |
| 更新 | "把牛肉的保质期改到下周三" |
| 健康检查 | 确认后端服务状态 |

### 渠道集成

- **飞书**：OAuth 2.0 授权接入，支持事件订阅（长连接），免公网 IP
- **Webhook 通知**：自定义机器人推送临期提醒
- **可扩展**：`Channel` 接口，可接入微信等更多渠道

### 配置热重载

- `config.json` 文件变更自动检测（2s 轮询）
- LLM 模型切换、飞书凭据更新无需重启服务
- Web 管理界面直接编辑配置并持久化

### 多模型支持

- 配置多个 LLM 模型（OpenAI / Ollama）
- Web 设置界面一键切换活跃模型
- API Key 安全编辑（不回显）

---

## 快速开始

### 前置准备

- Go 1.25+
- Node.js + pnpm（前端构建）
- Docker + Docker Compose（可选）
- 飞书开放平台凭证（可选）
- LLM API Key（OpenAI 或 Ollama）

### 本地开发

```bash
# 1. 克隆仓库
git clone <repository-url>
cd agent

# 2. 配置
cp config.example.json config.json
# 编辑 config.json：填入 LLM API Key、飞书凭证等

# 3. 启动后端
go run ./cmd/server

# 4. 启动前端（新终端）
cd web
pnpm install
pnpm dev
```

前端默认运行于 `http://localhost:5173`，后端 API 位于 `http://localhost:8888`。

### 生产构建

```bash
# 全平台交叉编译（含前端构建与嵌入）
make build-all

# 或仅当前平台
make build
```

构建产物输出至 `bin/` 目录。

### 飞书接入

完整对话（事件订阅 + Agent）：

1. 飞书开放平台创建自建应用，开启"接收消息"事件
2. 配置 `config.json` 中的 `channels.feishu` 与 `model_list` 字段
3. 在 Web 设置页面完成 OAuth 授权

仅提醒推送：

1. 飞书群添加自定义机器人，获取 Webhook URL
2. 配置 `channels.feishu` 的推送相关参数

---

## 项目结构

```
├── cmd/
│   ├── server/              # 后端入口
│   └── homestock/           # CLI 工具
├── internal/
│   ├── agent/               # AgentLoop + MessageBus
│   ├── app/                 # 应用组装/启动/关闭
│   ├── channel/             # 消息渠道接口 + Manager
│   │   └── feishu/          # 飞书渠道 + OAuth
│   ├── database/            # 数据库连接与迁移
│   ├── handler/             # HTTP 处理器
│   ├── hotreload/           # 配置热重载 Orchestrator
│   ├── httpserver/          # Gin 服务器封装
│   ├── llm/                 # LLM Provider（OpenAI / Ollama）
│   ├── model/               # GORM 数据模型
│   ├── repository/          # 数据访问层
│   ├── service/             # 业务逻辑层
│   └── tool/                # LLM Tool 注册与分发
├── pkg/
│   └── config/              # 配置管理（JSON + 环境变量覆盖）
└── web/
```

---

## 配置

配置通过 `config.json` 管理，支持环境变量覆盖。

| 配置项 | 环境变量 | 说明 |
|--------|----------|------|
| `server.port` | `HOMESTOCK_SERVER_PORT` | 监听端口，默认 `8888` |
| `server.hot_reload` | — | 启用配置热重载 |
| `database.driver` | `HOMESTOCK_DATABASE_DRIVER` | `sqlite` / `postgres` |
| `database.dsn` | `HOMESTOCK_DATABASE_DSN` | 数据源名称 |
| `channels.feishu.app_id` | `HOMESTOCK_CHANNELS_FEISHU_APP_ID` | 飞书 App ID |
| `channels.feishu.app_secret` | `HOMESTOCK_CHANNELS_FEISHU_APP_SECRET` | 飞书 App Secret |
| `model_list` | — | LLM 模型列表，每个含 provider / model / api_key |

---

## 部署方案

| 场景 | 方案 |
|------|------|
| 树莓派 / 低功耗设备 | SQLite + 单二进制 + systemd |
| 家用服务器 | Docker Compose（后端 + Nginx） |
| 局域网访问 | 路由器 DHCP 固定 IP，前端 :5173 / API :8888 |
| SaaS | PostgreSQL + JWT + HTTPS |

---

## 许可证

[MIT License](LICENSE)
