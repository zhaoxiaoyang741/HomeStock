---
name: inbound_stock
description: 新增一个入库批次。用户说“买了 X”“新增入库”“记录一批 X”时调用。
user-invocable: true
disable-model-invocation: false
command-dispatch: tool
command-tool: inbound_stock
---

本技能对应 `inbound_stock` 工具。识别到新增库存或记录入库批次的明确意图时，直接调用工具，不要追问确认，不要执行 shell 命令，不要检查文件。

## 触发场景
- “帮我入库 5 个土豆放冰箱”
- “买了两瓶酱油，分类记成调料”
- “记录一批牛奶，2026-04-30 过期”
- “新买了一袋洗衣粉”

## 参数提取
- `name`：物料名称，必填。
- `quantity`：数量；未提及时留空，由工具默认成 `1`。
- `unit`：单位，如“个”“瓶”“袋”“克”。
- `spec`：规格，如“500ml”“1kg”“大瓶装”。
- `location`：位置，如“冰箱”“厨房”“储物间”。
- `category`：分类名，如“蔬菜”“调料”“日用品”。
- `expire_at`：明确日期时转成 `YYYY-MM-DD`。
- `notes`：补充备注，原样传入。

## 执行规则
- 用户表达的是新增一条库存记录时，直接调用工具。
- 插件会按 `name + spec` 匹配物料，但每次入库都新增独立批次。
- 不要生成任何旧合并流程相关参数或说明。
- 日期先整理成 `YYYY-MM-DD` 再传入。

## 示例
输入：
```text
帮我入库 5 个土豆放冰箱，2026-04-30 过期
```

工具参数：
```json
{
  "name": "土豆",
  "quantity": 5,
  "unit": "个",
  "location": "冰箱",
  "expire_at": "2026-04-30"
}
```

Slash command：
```text
/inbound_stock 土豆 5个 冰箱
/inbound_stock 酱油 2瓶 厨柜
```
