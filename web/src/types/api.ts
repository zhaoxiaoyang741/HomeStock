export interface PaginatedResponse<T> {
  data: T[]
  total: number
  page: number
  page_size: number
}

export interface ApiError {
  message: string
  code?: string
}

export interface Result<T> { code: number; message: string; data: T }

export interface Page<T> { items: T[]; total: number; page?: number; page_size?: number }
