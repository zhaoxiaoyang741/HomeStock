import { api } from '@/lib/api'
import type { Result } from '@/types/api'
import type { SystemSettings, UpdateSystemSettingsPayload } from '@/types/systemSettings'

export const systemSettingsApi = {
  get: async (): Promise<SystemSettings> => {
    const res = await api.get<Result<SystemSettings>>('/v1/settings/system')
    return res.data
  },

  update: async (payload: UpdateSystemSettingsPayload): Promise<SystemSettings> => {
    const res = await api.put<Result<SystemSettings>>('/v1/settings/system', payload)
    return res.data
  },
}
