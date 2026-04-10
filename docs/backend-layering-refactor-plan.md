# HomeStock 后端分层重规划

## Summary

- 你的主线判断基本正确：应拆成 `router -> api -> service -> repository -> model`。需要修正的一点是：`request/response` 不应放进数据库 `model`，它们应属于 HTTP 传输层 DTO。
- 当前主要问题不是“缺一层 service”，而是 HTTP 层已经承担了业务编排、事务和审计，例如 [cmd/server/main.go](/e:/pro-tmp/agent/cmd/server/main.go)、[internal/handler/material_handler.go](/e:/pro-tmp/agent/internal/handler/material_handler.go)、[internal/handler/stock_lot_handler.go](/e:/pro-tmp/agent/internal/handler/stock_lot_handler.go)。
- 参考 `gin-vue-admin` 的分层边界与目录思想即可，不直接照搬它的重样板聚合。参考来源：
  - https://www.gin-vue-admin.com/guide/server/
  - https://gin-vue-admin.com/guide/generator/server.html
  - https://raw.githubusercontent.com/flipped-aurora/gin-vue-admin/main/server/router/enter.go
  - https://raw.githubusercontent.com/flipped-aurora/gin-vue-admin/main/server/service/enter.go
  - https://raw.githubusercontent.com/flipped-aurora/gin-vue-admin/main/server/api/v1/system/enter.go

## Key Changes

- 新目录按职责重组为轻量版本：
  - `internal/app`：应用装配入口，持有启动期单例 `*gorm.DB`，统一构造 repository、service、api、router。
  - `internal/api/http`：Gin handler，只做参数绑定、调用 service、错误映射、响应输出。
  - `internal/api/http/request`：HTTP 入参 DTO。
  - `internal/api/http/response`：HTTP 出参 DTO 与统一响应包裹。
  - `internal/router`：仅负责路由分组和注册，不写业务逻辑。
  - `internal/service`：业务编排、事务边界、库存规则、审计调用。
  - `internal/repository`：定义仓储接口。
  - `internal/repository/gorm`：GORM 实现。
  - `internal/model`：纯数据库实体与 GORM hook，保留表映射、默认值、ID 生成。
  - `internal/database`：数据库打开、迁移、事务管理，不暴露“裸全局变量”给业务层直接读写。
- `cmd/server/main.go` 只保留启动流程：`load config -> init logger -> app.New() -> server.Start/Shutdown`，不再手工 new 每个 repository/handler。
- 明确层职责：
  - `router`：注册 `/api/v1`、中间件、资源路由。
  - `api`：收敛 header/query/body/path 解析，构造 service command/query，返回统一响应。
  - `service`：唯一允许跨多个 repository 编排；`Inbound / Consume / Adjust` 的事务、库存扣减顺序、审计记录都放这里。
  - `repository`：只做持久化和查询，不做事务开启，不做 HTTP 返回结构组装，不依赖 Gin。
  - `model`：只保留实体、GORM tag、`BeforeCreate` ID 生成、默认字段填充。
- 服务拆分按业务而不是按当前 handler 文件：
  - `CategoryService`：分类 CRUD。
  - `MaterialService`：物料创建、详情、列表。
  - `InventoryService`：入库、消耗、库存批次调整、批次列表、流水列表。
  - `AuditService`：审计查询；写入可由 service 内部统一调用 recorder。
- 事务统一收口：
  - handler 不再持有 `*gorm.DB`。
  - service 通过 `TxManager` 或 `database.WithTx` 进入事务。
  - 事务内由 `repository/gorm` 基于 `tx` 构造事务仓储。
- 请求上下文统一化：
  - 增加 HTTP middleware，解析 `X-Tenant-ID`、`X-User-Name`、`X-User-ID`、`X-Channel` 到 typed context。
  - service 统一接收 `context.Context` + `Actor/Tenant`，不再在每个 handler 手动读 header。
- 错误与响应规范统一：
  - 所有接口统一为 `{ code, message, data }`。
  - 列表统一 `data.items`、`data.total`，分页接口再带 `page`、`page_size`。
  - 领域错误如 `not found / invalid category / insufficient stock / category in use` 由 API 层集中映射为 HTTP 状态码和 message。

## Public APIs / Types

- 保留 `/api/v1` 前缀，资源 URL 尽量延续现有语义；这次重点迁移内部分层与响应规范，不强制重命名全部路径。
- 新增 HTTP DTO，禁止直接返回 `internal/model` 实体：
  - `request.CreateCategoryRequest`
  - `request.CreateMaterialRequest`
  - `request.InboundStockRequest`
  - `request.ConsumeMaterialRequest`
  - `request.AdjustLotRequest`
  - `response.Category`
  - `response.MaterialDetail`
  - `response.StockLot`
  - `response.StockMovement`
  - `response.Page[T]`
  - `response.Result[T]`
- repository 接口定义按实体拆开；service 只依赖接口，不依赖 GORM 实现。
- `MaterialSummary`、`MaterialDetail` 这类当前放在 repository 的返回结构，迁为 service/view 或 response mapper 输入，去掉仓储层对 HTTP 结构的暗耦合。

## Test Plan

- repository 集成测试：
  - 分类唯一性、租户隔离、默认分类校验、库存批次查询顺序、审计分页查询。
- service 单元测试：
  - 入库时自动创建物料或复用自然键物料。
  - 消耗遵循当前 FEFO/FIFO 顺序。
  - 调整库存时 movement 与 lot 状态同步。
  - 事务内任一步失败时整体回滚。
  - 审计写入不影响主流程成功返回。
- api/router 测试：
  - 参数绑定与校验错误返回统一响应。
  - 领域错误映射正确。
  - 响应包裹结构稳定。
  - `tenant/user/channel` 中间件上下文传递正确。
- bootstrap 测试：
  - `app.New()` 能正确装配 DB、service、api、router。
  - `main` 不再直接依赖具体 repository/handler。

## Assumptions

- 本次按“整体迁移”规划，不以保持现有响应格式兼容为前提。
- 数据库继续使用 SQLite + GORM，ID 继续在 `internal/model` 的 hook 中生成。
- 最终采用“启动期单例 DB + 装配注入”，不采用业务层随处可读的真全局 DB 变量。
- 参考 gin-vue-admin 的是分层思想与模块边界，不采用它那种偏大型后台的全量聚合样板。
