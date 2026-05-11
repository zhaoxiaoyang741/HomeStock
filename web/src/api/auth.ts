import { api } from '@/lib/api'
import type { LoginRequest, RegisterRequest, AuthResponse, User } from '@/types/auth'

export const authApi = {
  login: (payload: LoginRequest) =>
    api.post<{ code: number; message: string; data: AuthResponse }>('/v1/auth/login', payload),

  register: (payload: RegisterRequest) =>
    api.post<{ code: number; message: string; data: { user: User } }>('/v1/auth/register', payload),

  me: () =>
    api.get<{ code: number; message: string; data: User }>('/v1/auth/me'),
}
