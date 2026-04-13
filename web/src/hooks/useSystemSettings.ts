import { useContext } from 'react'
import { SystemSettingsContext } from '@/providers/SystemSettingsProvider'

export function useSystemSettings() {
  const context = useContext(SystemSettingsContext)
  if (context === null) {
    throw new Error('useSystemSettings must be used within SystemSettingsProvider')
  }
  return context
}
