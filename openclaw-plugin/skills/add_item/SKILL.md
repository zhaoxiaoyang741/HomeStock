---
name: add_item
description: 添加库存物料。用户说“帮我添加X”“买了X”“记录一下X”时调用。
user-invocable: true
disable-model-invocation: false
command-dispatch: tool
command-tool: add_item
---

本技能对应 `add_item` 工具。识别到添加库存的明确意图时，直接调用工具，不要追问确认，不要执行 shell 命令，不要检查文件。

## 触发场景
- “帮我添加 5 个土豆放冰箱”
- “买了两瓶酱油，分类记成调料”
- “记录一下牛奶，2026-04-30 过期”
- “家里新添了一袋洗衣粉”

## 参数提取
- `name`：物料名称，必填。
- `quantity`：数量；未提及时留空，由工具默认成 1。
- `unit`：单位，如“个”“瓶”“袋”“盒”“克”。
- `location`：位置，如“冰箱”“冷冻层”“厨柜”“储物间”。
- `category`：分类名，如“蔬菜”“调料”“日用品”；工具会自动匹配或创建分类。
- `expire_at`：若用户给出明确日期，转换为 `YYYY-MM-DD`。
- `notes`：备注文本；若用户显式补充备注，原样传入。

## 执行规则
- 用户只要表达的是“新增一条库存记录”，就直接调用工具。
- 日期先转成 `YYYY-MM-DD` 再传，不要传自然语言日期。
- 不确定的字段可以省略，但 `name` 不能省略。

## 示例
输入：
```text
帮我添加 5 个土豆放冰箱，2026-04-30 过期
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
/add_item 土豆 5个 冰箱
/add_item 酱油 2瓶 厨柜
```
