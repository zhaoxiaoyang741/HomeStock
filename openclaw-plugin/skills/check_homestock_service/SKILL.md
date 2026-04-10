---
name: check_homestock_service
description: 检查当前 HomeStock 后端服务是否可用。用户说“确认 HomeStock 服务是否正常”“后端在线吗”“库存服务通不通”时调用。
user-invocable: true
disable-model-invocation: false
command-dispatch: tool
command-tool: check_homestock_service
---

本技能对应 `check_homestock_service` 工具。识别到确认 HomeStock 服务状态、检查后端地址是否可用、确认库存服务是否在线的明确意图时，直接调用工具。

## 触发场景
- “帮我确认目前 HomeStock 服务是否正常”
- “后端在线吗”
- “库存服务通不通”
- “现在能连上 HomeStock 吗”

## 执行规则
- 直接调用工具，不要自行猜测 `localhost:8888` 或其他地址。
- 工具会使用插件当前实际配置的 `baseUrl` 做健康检查。
- 返回内容应以工具检查结果为准，不要自行扩展为“没搭服务”“没配置地址”等主观推断。
