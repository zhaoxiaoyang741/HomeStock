import { api } from '@/lib/api'
import type {
  ConsumeMaterialPayload,
  ConsumeMaterialResponse,
  CreateMaterialPayload,
  Material,
  MaterialDetail,
  MaterialListResponse,
} from '@/types/material'
import type { Page, Result } from '@/types/api'

export const materialApi = {
  list: async (params?: { category_id?: string; keyword?: string; show_zero_stock?: boolean }): Promise<MaterialListResponse> => {
    const qs = new URLSearchParams()
    if (params?.category_id) qs.set('category_id', params.category_id)
    if (params?.keyword) qs.set('keyword', params.keyword)
    if (params?.show_zero_stock) qs.set('show_zero_stock', 'true')
    const query = qs.toString()
    const res = await api.get<Result<Page<MaterialDetail>>>(`/v1/materials${query ? `?${query}` : ''}`)
    return { materials: res.data.items, total: res.data.total }
  },

  get: async (id: string): Promise<MaterialDetail> => {
    const res = await api.get<Result<MaterialDetail>>(`/v1/materials/${id}`)
    return res.data
  },

  create: async (payload: CreateMaterialPayload): Promise<Material> => {
    const res = await api.post<Result<Material>>('/v1/materials', payload)
    return res.data
  },

  consume: async (id: string, payload: ConsumeMaterialPayload): Promise<ConsumeMaterialResponse> => {
    const res = await api.post<Result<ConsumeMaterialResponse>>(`/v1/materials/${id}/consume`, payload)
    return res.data
  },
}
