# home-inventory OpenClaw 插件

家用物料管理系统，通过自然语言和斜杠命令管理家中食材和日用品。

## 安装

```bash
openclaw plugins install ~/path/to/home-inventory-plugin-1.0.0.tgz
openclaw gateway restart
```

## 配置后台地址

插件默认连接 `http://localhost:8080`。通过 `~/.openclaw/openclaw.json` 中的 `plugins.entries` 配置实际后台地址：

```json
{
  "plugins": {
    "entries": {
      "home-inventory": {
        "config": {
          "baseUrl": "http://192.168.18.13:8080"
        }
      }
    }
  }
}
```

修改后重启 gateway 生效：

```bash
openclaw gateway restart
```

地址解析优先级（由高到低）：
1. `openclaw.json` 中的 `config.baseUrl`
2. 环境变量 `INVENTORY_API_URL`
3. 默认值 `http://localhost:8080`

## 斜杠命令

| 命令 | 说明 | 示例 |
|------|------|------|
| `/add_item <名称> [数量+单位] [位置]` | 添加物料 | `/add_item 土豆 5个 冰箱` |
| `/remove_item <名称> [数量]` | 消耗或删除物料 | `/remove_item 土豆 2` |
| `/query_items [位置] [分类] [expiring]` | 查询库存 | `/query_items 冰箱` |

## 构建

```bash
pnpm build
pnpm pack
```
