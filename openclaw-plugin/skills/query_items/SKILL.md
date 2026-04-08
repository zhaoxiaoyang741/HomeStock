---
name: query_items
description: 查询库存。用户说“还有什么”“冰箱里有什么”“还有什么调料”时调用。
user-invocable: true
disable-model-invocation: false
command-dispatch: tool
command-tool: query_items
---

本技能对应 `query_items` 工具。识别到查询库存、按位置查看、按分类查看、按关键字搜索或查看临期物料时，直接调用工具。

## 触发场景
- “冰箱里有什么”
- “家里还有什么调料”
- “查一下快过期的”
- “还有没有土豆”
- “厨柜里还有什么零食”

## 参数提取
- `location`：位置关键词，如“冰箱”“冷冻层”“厨柜”。
- `category`：分类名，如“调料”“蔬菜”“零食”。
- `expiring_soon`：用户提到“快过期”“临期”“即将过期”时传 `true`。
- `keyword`：按名称搜索的关键词，如“土豆”“牛奶”。

## 执行规则
- 用户是在问库存现状时，直接调用工具。
- “还有什么调料”这类话术优先把“调料”识别成 `category`。
- “还有没有土豆”这类话术优先把“土豆”识别成 `keyword`。
- 如果同时出现位置和分类，都要提取。

## 示例
输入：
```text
冰箱里还有什么调料
```

工具参数：
```json
{
  "location": "冰箱",
  "category": "调料"
}
```

输入：
```text
还有没有土豆
```

工具参数：
```json
{
  "keyword": "土豆"
}
```

Slash command：
```text
/query_items 冰箱
/query_items 冰箱 调料
/query_items expiring
```
