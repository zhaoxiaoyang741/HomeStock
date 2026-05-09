import { api } from '@/lib/api'
import type { FeishuStatus, FeishuAuthUrlResponse } from '@/types/feishu'
import type { Result } from '@/types/api'

export async function getFeishuAuthUrl(): Promise<FeishuAuthUrlResponse> {
  const res = await api.get<Result<FeishuAuthUrlResponse>>('/v1/feishu/auth-url')
  return res.data
}

export async function getFeishuStatus(): Promise<FeishuStatus> {
  const res = await api.get<Result<FeishuStatus>>('/v1/feishu/status')
  return res.data
}

export async function disconnectFeishu(): Promise<void> {
  await api.post<Result<unknown>>('/v1/feishu/disconnect', {})
}
