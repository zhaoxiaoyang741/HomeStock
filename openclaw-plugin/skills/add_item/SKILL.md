---
name: add_item
description: 向库存添加物料（食材/日用品），用户说"帮我添加X"、"买了X"、"放了X个Y"时调用
user-invocable: true
command-dispatch: tool
command-tool: add_item
---

向库存添加物料。用法：

/add_item <名称> [数量+单位] [位置]

示例：
- /add_item 土豆 5个 冰箱
- /add_item 酱油 2瓶 厨柜
- /add_item 鸡蛋
