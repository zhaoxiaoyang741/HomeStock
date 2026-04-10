---
name: update_stock_lot
description: 更新单个批次的库存或元数据。用户说“把 X 改成 Y”“X 的过期时间改成 Y”时调用。
user-invocable: true
disable-model-invocation: false
command-dispatch: tool
command-tool: update_stock_lot
---

本技能对应 `update_stock_lot` 工具。识别到修改批次库存、过期时间、备注或位置的明确意图时，直接调用工具。

## 触发场景
- “把土豆改成 3 袋”
- “牛奶的过期时间改成 2026-05-01”
- “把苹果挪到厨房”
- “给酱油加个备注：快用完了”
- “清空牛奶的过期日期”

## 参数提取
- `name`：物料名称，必填。
- `quantity`：新的批次库存。
- `expire_at`：新过期日期；统一转换成 `YYYY-MM-DD`。如明确要求清空，传空字符串。
- `notes`：新的备注内容；如明确要求清空，传空字符串。
- `location`：新的位置；如明确要求清空，传空字符串。

## 执行规则
- 至少提取一个更新字段再调用工具。
- 数量修改走库存调整。
- 过期日、备注、位置修改走批次更新。
- 如果同一物料存在多个批次，工具会提示用户先到 Web 中明确批次，不自动猜测。

## 示例
输入：
```text
把土豆改成 10 袋
```

工具参数：
```json
{
  "name": "土豆",
  "quantity": 10
}
```

输入：
```text
牛奶的过期时间改成 2026-05-01
```

工具参数：
```json
{
  "name": "牛奶",
  "expire_at": "2026-05-01"
}
```

Slash command：
```text
/update_stock_lot 土豆 10袋
/update_stock_lot 牛奶 2026-05-01
/update_stock_lot 苹果 冰箱上层
```
