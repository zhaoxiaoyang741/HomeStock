---
name: query_items
description: 查询家中库存物料，用户说"还有什么"、"冰箱里有什么"、"快过期的"时调用
user-invocable: true
command-dispatch: tool
command-tool: query_items
---

查询库存。用法：

/query_items [位置] [分类] [expiring]

示例：
- /query_items
- /query_items 冰箱
- /query_items 冰箱 蔬菜
- /query_items expiring
