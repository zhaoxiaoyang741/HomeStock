import { useEffect, useMemo } from 'react'
import { useThemeStore, type Theme } from '@/store/themeStore'

export type { Theme }

export function useTheme() {
  const { theme, setTheme } = useThemeStore()

  const resolvedTheme: 'light' | 'dark' = useMemo(() => {
    if (theme === 'system') {
      return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
    }
    return theme
  }, [theme])

  // 主题变化时同步 DOM + 持久化
  useEffect(() => {
    document.documentElement.classList.toggle('dark', resolvedTheme === 'dark')
    try { localStorage.setItem('theme', theme) } catch {}
  }, [theme, resolvedTheme])

  // system 模式下监听系统偏好变化
  useEffect(() => {
    if (theme !== 'system') return
    const mq = window.matchMedia('(prefers-color-scheme: dark)')
    const handler = () => {
      document.documentElement.classList.toggle('dark', mq.matches)
    }
    mq.addEventListener('change', handler)
    return () => mq.removeEventListener('change', handler)
  }, [theme])

  return { theme, resolvedTheme, setTheme }
}
