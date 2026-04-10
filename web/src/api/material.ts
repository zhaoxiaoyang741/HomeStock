import { api } from '@/lib/api'
import type {
  ConsumeMaterialPayload,
  ConsumeMaterialResponse,
  CreateMaterialPayload,
  Material,
  MaterialDetail,
  MaterialListResponse,
} from '@/types/material'

export const materialApi = {
  list: (params?: { category_id?: string; keyword?: string }) => {
    const qs = new URLSearchParams()
    if (params?.category_id) qs.set('category_id', params.category_id)
    if (params?.keyword) qs.set('keyword', params.keyword)
    const query = qs.toString()
    return api.get<MaterialListResponse>(`/v1/materials${query ? `?${query}` : ''}`)
  },

  get: (id: string) => api.get<MaterialDetail>(`/v1/materials/${id}`),

  create: (payload: CreateMaterialPayload) => api.post<Material>('/v1/materials', payload),

  consume: (id: string, payload: ConsumeMaterialPayload) =>
    api.post<ConsumeMaterialResponse>(`/v1/materials/${id}/consume`, payload),
}
