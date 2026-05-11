export interface User {
  id: number
  username: string
  display_name: string
  created_at: string
  updated_at: string
}

export interface LoginRequest {
  username: string
  password: string
}

export interface RegisterRequest {
  username: string
  password: string
  display_name?: string
}

export interface AuthResponse {
  token: string
  user: User
}
