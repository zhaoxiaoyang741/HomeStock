import { api } from '@/lib/api'
import type { Item, ItemListResponse, CreateItemPayload, UpdateItemPayload } from '@/types/item'

export const itemApi = {
  list: (params?: { location?: string; category_id?: string }) => {
    const qs = new URLSearchParams()
    if (params?.location) qs.set('location', params.location)
    if (params?.category_id) qs.set('category_id', params.category_id)
    const query = qs.toString()
    return api.get<ItemListResponse>(`/v1/items${query ? `?${query}` : ''}`)
  },

  get: (id: string) => api.get<Item>(`/v1/items/${id}`),

  create: (payload: CreateItemPayload) => api.post<Item>('/v1/items', payload),

  update: (id: string, payload: UpdateItemPayload) => api.put<Item>(`/v1/items/${id}`, payload),

  delete: (id: string) => api.delete<void>(`/v1/items/${id}`),

  findSimilar: (name: string, spec?: string) => {
    const qs = new URLSearchParams({ name })
    if (spec) qs.set('spec', spec)
    return api.get<ItemListResponse>(`/v1/items/similar?${qs}`)
  },
}
