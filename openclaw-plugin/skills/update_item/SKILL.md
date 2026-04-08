---
name: update_item
description: 更新已有库存物料。用户说“把X改成Y个”“X的过期时间改成Z”时调用。
user-invocable: true
disable-model-invocation: false
command-dispatch: tool
command-tool: update_item
---

本技能对应 `update_item` 工具。识别到修改库存数量、过期时间、备注或位置的明确意图时，直接调用工具。

## 触发场景
- “把土豆改成 3 个”
- “牛奶的过期时间改成 2026-05-01”
- “把苹果挪到厨柜”
- “给酱油加个备注：快用完了”
- “清空牛奶的过期日期”

## 参数提取
- `name`：物料名称，必填。
- `quantity`：新数量。
- `expire_at`：新过期日期，统一转换为 `YYYY-MM-DD`；若用户明确要求清空，传空字符串。
- `notes`：新备注内容；若用户明确要求清空备注，传空字符串。
- `location`：新位置；若用户明确要求清空位置，传空字符串。

## 执行规则
- 至少提取一个更新字段再调用工具。
- “月底”“下周五”这类自然语言日期，先推断成明确日期，再传 `YYYY-MM-DD`。
- 用户要清空字段时，传空字符串，不要省略字段。

## 示例
输入：
```text
把土豆改成 10 个
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
/update_item 土豆 10个
/update_item 牛奶 2026-05-01
/update_item 苹果 冰箱上层
```
