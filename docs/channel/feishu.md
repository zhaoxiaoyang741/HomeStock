# 飞书渠道配置

## 配置结构

```json
{
  "channels": {
    "feishu": {
      "enabled": true,
      "app_id": "cli_xxxxxxxxxxxx",
      "app_secret": "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
      "redirect_uri": "http://localhost:8888/api/v1/feishu/callback",
      "frontend_url": "http://localhost:5173"
    }
  }
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `enabled` | bool | 是 | 是否启用飞书渠道 |
| `app_id` | string | 是 | 飞书应用的 App ID（以 `cli_` 开头） |
| `app_secret` | string | 是 | 飞书应用的 App Secret |
| `redirect_uri` | string | 否 | OAuth 回调地址，默认 `http://localhost:8888/api/v1/feishu/callback` |
| `frontend_url` | string | 否 | 前端地址，用于 OAuth 完成后跳转，默认 `http://localhost:5173` |

> 首次启动时程序会自动在 `config.json` 中生成默认配置，只需修改 `app_id` 和 `app_secret` 并启用即可。

## 设置流程

### 1. 创建飞书应用

1. 前往 [飞书开放平台](https://open.feishu.cn/) 创建企业自建应用
2. 在"凭据与基础信息"页面获取 **App ID**（以 `cli_` 开头）和 **App Secret**

### 2. 配置应用权限

1. 在应用左侧菜单进入"权限管理"
2. 启用以下权限：
   - `im:message` — 读写消息
   - `im:resource` — 获取图片、文件等资源
   - `contact:user.base` — 读取用户基础信息
   - `contact:user.employee_id:readonly` — 读取用户邮箱等

### 3. 配置事件订阅

1. 在应用左侧菜单进入"事件与回调"
2. 添加以下事件：
   - `接收消息（消息事件）` — `im.message.receive_v1`
3. 回调地址填写为重定向页面地址（如 `https://your-domain.com/feishu/callback`，或默认前端地址）

### 4. 配置 OAuth 回调域名

1. 进入"安全设置"
2. 在"重定向 URL"中添加 `https://your-domain.com/api/v1/feishu/callback`

### 5. 发布应用

1. 在应用左侧菜单进入"版本管理与发布"
2. 创建版本并成功发布

### 6. 启动服务并授权

1. 确保 `config.json` 中已填写 `app_id` 和 `app_secret`
2. 启动服务
3. 在浏览器中打开 Web 管理界面，进入"设置 → 飞书"页面
4. 点击"飞书授权"按钮完成 OAuth 授权流程

### 7. 验证

在飞书中搜索你的机器人应用，发送消息验证是否正常响应。

## 工作原理

- 本系统通过 **飞书开放 API（SDK 模式）** 建立长连接 WebSocket，无需公网 IP 或 Webhook 地址
- 使用 **OAuth 2.0 授权码流程** 获取 tenant_access_token，无需手动填写 token
- 授权完成后 token 持久化存储，重启服务后自动恢复

## 注意事项

- 飞书应用必须**发布**后相关配置才会生效
- OAuth 授权状态与 `app_id`/`app_secret` 绑定，修改凭据后需重新授权
- 在 Web 设置页面更新飞书配置后会触发热重载，无需重启服务
