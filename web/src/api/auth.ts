import { api } from '@/lib/api'
import type { LoginRequest, AuthResponse, User } from '@/types/auth'

export const authApi = {
  login: (payload: LoginRequest) =>
    api.post<{ code: number; message: string; data: AuthResponse }>('/v1/auth/login', payload),

  me: () =>
    api.get<{ code: number; message: string; data: User }>('/v1/auth/me'),
}
