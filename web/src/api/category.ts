import { api } from '@/lib/api'
import type { Category, CategoryListResponse, CreateCategoryPayload, UpdateCategoryPayload } from '@/types/category'
import type { Page, Result } from '@/types/api'

export const categoryApi = {
  list: async (): Promise<CategoryListResponse> => {
    const res = await api.get<Result<Page<Category>>>('/v1/categories')
    return { categories: res.data.items, total: res.data.total }
  },

  get: async (id: string): Promise<Category> => {
    const res = await api.get<Result<Category>>(`/v1/categories/${id}`)
    return res.data
  },

  create: async (payload: CreateCategoryPayload): Promise<Category> => {
    const res = await api.post<Result<Category>>('/v1/categories', payload)
    return res.data
  },

  update: async (id: string, payload: UpdateCategoryPayload): Promise<Category> => {
    const res = await api.put<Result<Category>>(`/v1/categories/${id}`, payload)
    return res.data
  },

  delete: (id: string) => api.delete<void>(`/v1/categories/${id}`),
}
