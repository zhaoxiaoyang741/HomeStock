import { create } from 'zustand'
import { authApi } from '@/api/auth'
import { setTokenProvider } from '@/lib/api'
import type { User } from '@/types/auth'

const TOKEN_KEY = 'auth_token'

function readToken(): string | null {
  try {
    return localStorage.getItem(TOKEN_KEY)
  } catch {
    return null
  }
}

function saveToken(token: string | null) {
  try {
    if (token) {
      localStorage.setItem(TOKEN_KEY, token)
    } else {
      localStorage.removeItem(TOKEN_KEY)
    }
  } catch {
    // localStorage may be unavailable
  }
}

// Inject token getter into the API client on module load.
setTokenProvider(() => readToken())

interface AuthState {
  token: string | null
  user: User | null
  isAuthenticated: boolean
  isInitialized: boolean
  login: (username: string, password: string) => Promise<void>
  logout: () => void
  initialize: () => Promise<void>
}

export const useAuthStore = create<AuthState>((set) => ({
  token: null,
  user: null,
  isAuthenticated: false,
  isInitialized: false,

  login: async (username, password) => {
    const res = await authApi.login({ username, password })
    const { token, user } = res.data
    saveToken(token)
    set({ token, user, isAuthenticated: true })
  },

  logout: () => {
    saveToken(null)
    set({ token: null, user: null, isAuthenticated: false })
  },

  initialize: async () => {
    const token = readToken()
    if (!token) {
      set({ isInitialized: true })
      return
    }
    try {
      const res = await authApi.me()
      set({ token, user: res.data, isAuthenticated: true, isInitialized: true })
    } catch {
      saveToken(null)
      set({ isInitialized: true })
    }
  },
}))

// Listen for 401 events dispatched by api.ts to auto-logout
window.addEventListener('auth:unauthorized', () => {
  saveToken(null)
  useAuthStore.getState().logout()
})
