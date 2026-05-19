import { api } from '@/lib/api'
import type { Result } from '@/types/api'

export interface CronConfig {
  enabled: boolean
  expiry_check_interval_days: number
  expiry_check_poll_interval: string
  notify_enabled: boolean
  notify_time_start: string
  notify_time_end: string
}

export async function getCronConfig(): Promise<CronConfig> {
  const res = await api.get<Result<CronConfig>>('/v1/cron/config')
  return res.data
}

export async function updateCronConfig(payload: Partial<CronConfig>): Promise<void> {
  await api.put<Result<unknown>>('/v1/cron/config', payload)
}
