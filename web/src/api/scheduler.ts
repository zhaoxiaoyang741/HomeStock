import { api } from '@/lib/api'
import type { Result } from '@/types/api'
import type { SchedulerStatus } from '@/types/scheduler'

export const schedulerApi = {
  getStatus: async (): Promise<SchedulerStatus> => {
    const res = await api.get<Result<SchedulerStatus>>('/v1/scheduler/status')
    return res.data
  },

  trigger: async (): Promise<void> => {
    await api.post<Result<unknown>>('/v1/scheduler/trigger', {})
  },
}
