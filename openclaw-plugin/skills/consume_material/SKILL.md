---
name: consume_material
description: 按物料消耗库存。用户说“用了 X”“X 吃完了”“消耗 X”时调用。
user-invocable: true
disable-model-invocation: false
command-dispatch: tool
command-tool: consume_material
---

本技能对应 `consume_material` 工具。识别到消耗、取用、吃完库存的明确意图时，直接调用工具。

## 触发场景
- “用了 2 个土豆”
- “鸡蛋吃完了”
- “消耗 1 瓶牛奶”
- “刚刚用掉一袋洗衣粉”

## 参数提取
- `name`：物料名称，必填。
- `quantity`：消耗数量；如用户表达“吃完了”“没有了”，可以不传，由工具按当前总库存处理。

## 执行规则
- 用户表达的是减少库存时，直接调用，不需要确认。
- 插件按物料发起消耗，后端会自动按批次扣减。
- 不要使用“删除 item”“整条删除记录”之类旧表述。

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
/consume_material 土豆 2
/consume_material 鸡蛋
```
