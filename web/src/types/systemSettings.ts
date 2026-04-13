export interface SystemSettingsReminder {
  remind_days: number
  check_time: string
}

export interface SystemSettingsNotify {
  enabled: boolean
  feishu_webhook_configured: boolean
  feishu_webhook_masked: string
}

export interface SystemSettingsUpdatedBy {
  user_name: string
  user_id: string
  channel: string
}

export interface SystemSettings {
  version: number
  reminder: SystemSettingsReminder
  notify: SystemSettingsNotify
  updated_at: string
  updated_by: SystemSettingsUpdatedBy
}

export interface UpdateSystemSettingsPayload {
  version: number
  reminder: SystemSettingsReminder
  notify: {
    enabled: boolean
    feishu_webhook_mode: 'keep' | 'replace' | 'clear'
    feishu_webhook?: string
  }
}
