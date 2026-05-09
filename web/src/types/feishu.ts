export interface FeishuStatus {
  configured: boolean
  connected: boolean
  enabled: boolean
  bot_name: string
  app_id: string
}

export interface FeishuAuthUrlResponse {
  auth_url: string
}

export interface UpdateFeishuConfigPayload {
  enabled?: boolean
  app_id?: string
  app_secret?: string
}
