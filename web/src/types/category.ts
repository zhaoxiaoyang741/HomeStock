export interface Category {
  id: string
  tenant_id: string
  name: string
  created_at: string
  updated_at: string
}

export interface CreateCategoryPayload {
  name: string
}

export interface UpdateCategoryPayload {
  name?: string
}

export interface CategoryListResponse {
  categories: Category[]
  total: number
}
