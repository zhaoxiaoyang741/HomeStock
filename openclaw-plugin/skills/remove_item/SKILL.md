---
name: remove_item
description: 消耗或删除库存物料。用户说“用了X”“X吃完了”“删掉X”时调用。
user-invocable: true
disable-model-invocation: false
command-dispatch: tool
command-tool: remove_item
---

本技能对应 `remove_item` 工具。识别到消耗、取出、吃完、删除库存的明确意图时，直接调用工具。

## 触发场景
- “用了 2 个土豆”
- “鸡蛋吃完了”
- “把酱油删掉”
- “刚刚用掉一瓶牛奶”

## 参数提取
- `name`：物料名称，必填。
- `quantity`：消耗数量；如果用户表达“吃完了”“没了”“删掉”，可以不传，由工具整条删除。

## 执行规则
- 用户表达的是减少库存时，直接调用，不需要确认。
- 如果数量明确，就提取为数字。
- 如果没有数量但意图是“删除/吃完/没了”，只传 `name`。

## 示例
输入：
```text
用了 2 个土豆
```

工具参数：
```json
{
  "name": "土豆",
  "quantity": 2
}
```

Slash command：
```text
/remove_item 土豆 2
/remove_item 鸡蛋
```
