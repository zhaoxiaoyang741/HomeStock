---
name: remove_item
description: 从库存消耗或删除物料，用户说"用了X"、"吃了X"、"X吃完了"时调用
user-invocable: true
command-dispatch: tool
command-tool: remove_item
---

从库存消耗或删除物料。用法：

/remove_item <名称> [数量]

示例：
- /remove_item 土豆 2
- /remove_item 鸡蛋
