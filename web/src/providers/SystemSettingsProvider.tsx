import { createContext, useCallback, useEffect, useMemo, useState, type ReactNode } from 'react'
import { systemSettingsApi } from '@/api/systemSettings'
import type { SystemSettings, UpdateSystemSettingsPayload } from '@/types/systemSettings'
import { DEFAULT_REMIND_DAYS } from '@/types/systemSettings'

type SystemSettingsContextValue = {
  settings: SystemSettings | null
  loading: boolean
  error: string
  remindDays: number
  refreshSettings: () => Promise<SystemSettings | null>
  updateSettings: (payload: UpdateSystemSettingsPayload) => Promise<SystemSettings>
}

export const SystemSettingsContext = createContext<SystemSettingsContextValue | null>(null)

export function SystemSettingsProvider({ children }: { children: ReactNode }) {
  const [settings, setSettings] = useState<SystemSettings | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const refreshSettings = useCallback(async (): Promise<SystemSettings | null> => {
    setLoading(true)
    setError('')
    try {
      const next = await systemSettingsApi.get()
      setSettings(next)
      return next
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load system settings')
      return null
    } finally {
      setLoading(false)
    }
  }, [])

  const updateSettings = useCallback(async (payload: UpdateSystemSettingsPayload): Promise<SystemSettings> => {
    const saved = await systemSettingsApi.update(payload)
    setSettings(saved)
    setError('')
    return saved
  }, [])

  useEffect(() => {
    void refreshSettings()
  }, [refreshSettings])

  const value = useMemo<SystemSettingsContextValue>(() => ({
    settings,
    loading,
    error,
    remindDays: settings?.reminder.remind_days ?? DEFAULT_REMIND_DAYS,
    refreshSettings,
    updateSettings,
  }), [settings, loading, error, refreshSettings, updateSettings])

  return (
    <SystemSettingsContext.Provider value={value}>
      {children}
    </SystemSettingsContext.Provider>
  )
}
