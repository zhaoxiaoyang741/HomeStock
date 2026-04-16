import { api } from '@/lib/api'
import type { Result } from '@/types/api'
import type { NotificationPage } from '@/types/notification'

export interface NotificationListParams {
  lot_id?: string
  status?: string
  page?: number
  page_size?: number
}

export const notificationsApi = {
  list: async (params: NotificationListParams = {}): Promise<NotificationPage> => {
    const qs = new URLSearchParams()
    if (params.lot_id) qs.set('lot_id', params.lot_id)
    if (params.status) qs.set('status', params.status)
    if (params.page) qs.set('page', String(params.page))
    if (params.page_size) qs.set('page_size', String(params.page_size))
    const query = qs.toString()
    const res = await api.get<Result<NotificationPage>>(`/v1/notifications${query ? `?${query}` : ''}`)
    return res.data
  },
}
