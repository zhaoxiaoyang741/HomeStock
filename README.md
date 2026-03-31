# 家庫 · Jiaku

家用物料管理系统：管理食材与日用品库存，跟踪保质期，并在临期前提醒，减少浪费。支持飞书多模态录入（文字 / 语音 / 小票拍照）与 Web 管理界面，AI 层基于 **OpenClaw**。

对外可见信息以本 README 为准；详细设计文档仅内部维护，不随仓库公开。

---

## 特性概览

- **自然语言与多模态录入**：说话、发文字、拍购物小票即可更新库存。
- **保质期与临期提醒**：自动维护到期时间，定时扫描并推送通知（如飞书）。
- **轻量部署**：支持树莓派 / NAS / PC；数据库可在 SQLite 与 PostgreSQL 间按场景切换。

---

## 技术栈

| 模块 | 技术 |
|------|------|
| AI Agent | OpenClaw |
| 后端 | Go、Gin、GORM、robfig/cron |
| 前端 | React、Vite、Tailwind |
| 插件 | TypeScript（OpenClaw） |
| 数据 | SQLite / PostgreSQL |
| 交付 | Docker、Docker Compose |

---

## 设计与实现概要

以下为内部设计文档的对外摘要，便于贡献者理解全貌。

### 整体架构

- **入口**：飞书（文字 / 语音 / 图片）→ OpenClaw；浏览器 → React。
- **中间层**：OpenClaw 插件（TypeScript）负责 LLM 意图、ASR、小票 OCR，经 HTTP 调用后端。
- **核心服务**：Go REST API（物料 CRUD、保质期、按配置的 Cron 扫描临期项、飞书 Webhook 推送）。
- **数据**：SQLite（嵌入式 / 轻量），或 PostgreSQL（NAS / 云端 / 多租户）。
- **录入链路**：文字/语音经 LLM 调 Tool；小票图片经 OCR 批量入库；Web 表单直连 API。
- **部署梯度**：单机（树莓派 + SQLite）、家用 Docker Compose（SQLite 或 PostgreSQL）、SaaS（PostgreSQL + `tenant_id` + JWT）。

### 后端（Go）

- 分层：`cmd/server`、`config`（Viper + 环境变量）、`model` / `repository` / `service`、`handler`、到期 `scheduler`、飞书 `notifier`。
- 能力：REST `/api/v1/items`（增删改查、按位置/分类/临期筛选）、`/api/v1/health`；内置常见食材保质期字典，可结合 LLM 兜底未知品类。
- 配置项示例：服务端口、数据库 DSN、飞书 Webhook、提前提醒天数与每日检查时刻。

### OpenClaw 插件

- Tool：`add_item`、`remove_item`、`query_items`、`parse_receipt`；`SKILL.md` 描述触发话术；`openclaw.plugin.json` 声明插件。
- 运行时通过 `INVENTORY_API_URL` 等形式指向 Go 后端，将 LLM 解析结果转为 API 调用并格式化回复。

### 前端（React）

- 技术：Vite、TypeScript、Tailwind、TanStack Query、React Router；目标是在低端设备浏览器上可用。
- 页面能力：按存放位置分组展示、临期看板与颜色标识、手动添加/编辑、设置；`VITE_API_URL` 指向后端。

### 数据库

- 主表 `items`：名称、分类、数量单位、位置、`expire_at`（可空）、购买时间、备注、`tenant_id`（家用默认 `default`）。
- 表 `notifications`：与物料关联的通知记录（状态、渠道、时间）。
- GORM `AutoMigrate` 建表；复杂变更可用 golang-migrate；备份可拷贝 SQLite 文件或使用 `pg_dump`。

### 部署与运维

- **推荐**：Docker Compose 同时拉起后端（数据卷持久化 DB）与前端（Nginx 托管静态资源）；`.env` 存 Webhook 等敏感项。
- **裸机 / 树莓派**：交叉编译或直接 `go build`；前端 `npm run build`；可选 embed 静态资源进单二进制；systemd 保活。
- **局域网**：固定路由器 DHCP / 静态 IP，访问 `:3000`（前端）与 `:8080`（API）。
- **SaaS**：HTTPS（如 Nginx + Let’s Encrypt）、JWT、PostgreSQL；多架构镜像可用 `buildx`。
- **资源参考**：仅后端与前端空载约 20–40MB 量级内存；OpenClaw / LLM 另行占用，可拆机部署。

### 飞书

- **完整对话**：开放平台自建应用 + 权限（收消息、发消息等）+ 事件「接收消息」**长连接**（免公网 IP）；在 OpenClaw 中配置 `app_id` / `app_secret` 与 LLM。
- **仅提醒**：群「自定义机器人」Webhook URL 配进后端，后端 Cron 推送文本。
- 可先通 Webhook 验证提醒，再接入 OpenClaw 做多模态与对话。

---

## 快速开始

代码目录与一键启动命令将在各模块落地后补充。初始化时准备：Docker（可选）、数据库路径或 PostgreSQL、飞书 Webhook 或开放平台凭证、OpenClaw 与插件路径、以及前端 `VITE_API_URL`。

```bash
git clone <repository-url>
cd agent
# 后续：配置环境变量、数据库与 Docker Compose 等
```

---

## 许可证

本项目采用 [MIT License](LICENSE)。

若将项目 fork 或对外发布，可将 `LICENSE` 中的版权持有人行（`Copyright (c) 2026 …`）替换为你的姓名或组织名称。
