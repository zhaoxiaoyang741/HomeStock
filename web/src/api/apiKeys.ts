import { api } from '@/lib/api'
import type { Result } from '@/types/api'

export async function listAPIKeys(): Promise<string[]> {
  const res = await api.get<Result<string[]>>('/auth/api-keys')
  return res.data
}

export async function addAPIKey(key: string): Promise<void> {
  await api.post('/auth/api-keys', { key })
}

export async function deleteAPIKey(key: string): Promise<void> {
  await api.delete(`/auth/api-keys/${encodeURIComponent(key)}`)
}
