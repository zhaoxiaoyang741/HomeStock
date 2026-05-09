export interface FeishuStatus {
  configured: boolean
  connected: boolean
  bot_name: string
  app_id: string
}

export interface FeishuAuthUrlResponse {
  auth_url: string
}
