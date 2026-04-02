import { create } from 'zustand'

export type Theme = 'light' | 'dark' | 'system'

function readStored(): Theme {
  try { return (localStorage.getItem('theme') as Theme) ?? 'system' } catch { return 'system' }
}

// 模块级同步执行：在首次 render 前应用主题，防止刷新闪烁
const initial = readStored()
const systemDark = window.matchMedia('(prefers-color-scheme: dark)').matches
document.documentElement.classList.toggle(
  'dark',
  initial === 'dark' || (initial === 'system' && systemDark)
)

interface ThemeState {
  theme: Theme
  setTheme: (theme: Theme) => void
}

export const useThemeStore = create<ThemeState>()((set) => ({
  theme: initial,
  setTheme: (theme) => set({ theme }),
}))
