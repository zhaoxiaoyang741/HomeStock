# 微信渠道配置

## 配置结构

```json
{
  "channels": {
    "wechat": {
      "enabled": true,
      "token": "your_wechat_token",
      "account_id": "your_wechat_account_id",
      "base_url": "https://ilinkai.weixin.qq.com/",
      "cdn_base_url": "https://novac2c.cdn.weixin.qq.com/c2c",
      "proxy": ""
    }
  }
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `enabled` | bool | 是 | 是否启用微信渠道 |
| `token` | string | 是 | 微信客服 API Token（登录后获取） |
| `account_id` | string | 是 | 微信客服账号 ID |
| `base_url` | string | 否 | API 基础地址，默认 `https://ilinkai.weixin.qq.com/` |
| `cdn_base_url` | string | 否 | CDN 基础地址，默认 `https://novac2c.cdn.weixin.qq.com/c2c` |
| `proxy` | string | 否 | HTTP 代理地址，留空不使用代理 |

> 首次启动时程序会自动在 `config.json` 中生成默认配置，只需修改 `token` 和 `account_id` 并启用即可。

## 设置流程

### 1. 开通微信客服能力

1. 前往 [微信客服开放平台](https://open.work.weixin.qq.com/) 注册或登录
2. 开通微信客服能力，创建客服账号
3. 获取客服账号的 **账号 ID**

### 2. 获取 Token

Token 为个人微信登录后的六位数字验证码，启动服务后在 Web 管理界面中通过"微信"设置页面获取登录二维码，扫码即可完成登录并自动获取 Token。

### 3. 启动服务并绑定

1. 在 `config.json` 中填写 `account_id`，将 `enabled` 设为 `true`
2. 启动服务
3. 打开 Web 管理界面，进入"设置 → 微信"页面
4. 页面将显示微信登录二维码，使用个人微信扫码完成绑定
5. 绑定后系统会自动获取并管理 Token

### 4. 验证

在个人微信中向该客服账号发送消息，验证是否正常响应。

## 工作原理

- 本系统通过 **腾讯 iLink REST API** 连接个人微信，与 picoclaw 采用相同方案
- **非 WebHook/公众号模式**，无需公网 IP 或域名
- 使用**长轮询（long-polling）** 方式接收消息
- 登录状态持久化，重启服务后自动恢复 session

## 注意事项

- 微信 session 有时效性，过期后系统会自动暂停 5 分钟并等待重新登录
- 若 session 连续多次过期，需要重新扫码登录
- 在 Web 设置页面更新微信配置后会触发热重载，无需重启服务
- 微信消息的图片、文件等资源通过 CDN 地址获取
