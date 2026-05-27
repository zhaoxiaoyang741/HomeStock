import { api } from '@/lib/api'
import type { Result } from '@/types/api'

export interface EndpointConfig {
  name: string
  url: string
  enabled: boolean
}

export async function listEndpoints(): Promise<EndpointConfig[]> {
  const res = await api.get<Result<EndpointConfig[]>>('/outbound/endpoints')
  return res.data
}

export async function createEndpoint(data: { name: string; url: string; enabled?: boolean }): Promise<void> {
  await api.post('/outbound/endpoints', data)
}

export async function updateEndpoint(name: string, data: { url?: string; enabled?: boolean }): Promise<void> {
  await api.put(`/outbound/endpoints/${encodeURIComponent(name)}`, data)
}

export async function deleteEndpoint(name: string): Promise<void> {
  await api.delete(`/outbound/endpoints/${encodeURIComponent(name)}`)
}

export async function testEndpoint(data: { url: string; type?: string; body?: string }): Promise<void> {
  await api.post('/outbound/test', data)
}
