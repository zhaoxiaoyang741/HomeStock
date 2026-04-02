import { api } from '@/lib/api'
import type { Category, CategoryListResponse, CreateCategoryPayload, UpdateCategoryPayload } from '@/types/category'

export const categoryApi = {
  list: () => api.get<CategoryListResponse>('/v1/categories'),

  get: (id: string) => api.get<Category>(`/v1/categories/${id}`),

  create: (payload: CreateCategoryPayload) => api.post<Category>('/v1/categories', payload),

  update: (id: string, payload: UpdateCategoryPayload) => api.put<Category>(`/v1/categories/${id}`, payload),

  delete: (id: string) => api.delete<void>(`/v1/categories/${id}`),
}
