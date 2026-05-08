import { api } from '@/lib/api'
import type { Page, Result } from '@/types/api'
import type {
  ScheduledTask,
  ScheduledTaskRun,
  ScheduledTaskRunPage,
  UpdateScheduledTaskPayload,
} from '@/types/scheduledTask'

export interface ScheduledTaskRunListParams {
  task_code?: string
  status?: string
  trigger_source?: string
  page?: number
  page_size?: number
}

export const scheduledTasksApi = {
  list: async (): Promise<ScheduledTask[]> => {
    const res = await api.get<Result<ScheduledTask[]>>('/v1/scheduled-tasks')
    return res.data
  },

  get: async (code: string): Promise<ScheduledTask> => {
    const res = await api.get<Result<ScheduledTask>>(`/v1/scheduled-tasks/${code}`)
    return res.data
  },

  update: async (code: string, payload: UpdateScheduledTaskPayload): Promise<ScheduledTask> => {
    const res = await api.patch<Result<ScheduledTask>>(`/v1/scheduled-tasks/${code}`, payload)
    return res.data
  },

  trigger: async (code: string): Promise<ScheduledTaskRun> => {
    const res = await api.post<Result<ScheduledTaskRun>>(`/v1/scheduled-tasks/${code}/trigger`, {})
    return res.data
  },

  listRuns: async (params: ScheduledTaskRunListParams = {}): Promise<ScheduledTaskRunPage> => {
    const qs = new URLSearchParams()
    if (params.task_code) qs.set('task_code', params.task_code)
    if (params.status) qs.set('status', params.status)
    if (params.trigger_source) qs.set('trigger_source', params.trigger_source)
    if (params.page) qs.set('page', String(params.page))
    if (params.page_size) qs.set('page_size', String(params.page_size))
    const query = qs.toString()
    const res = await api.get<Result<Page<ScheduledTaskRun>>>(`/v1/scheduled-task-runs${query ? `?${query}` : ''}`)
    return {
      items: res.data.items,
      total: res.data.total,
      page: res.data.page ?? 1,
      page_size: res.data.page_size ?? 20,
    }
  },
}
