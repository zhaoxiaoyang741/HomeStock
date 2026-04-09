import { api } from '@/lib/api'
import type { AuditLogFilter, AuditLogListResponse } from '@/types/audit'

export const auditLogApi = {
  list: (filter?: AuditLogFilter) => {
    const qs = new URLSearchParams()
    if (filter?.action) qs.set('action', filter.action)
    if (filter?.channel) qs.set('channel', filter.channel)
    if (filter?.user_name) qs.set('user_name', filter.user_name)
    if (filter?.start_date) qs.set('start_date', filter.start_date)
    if (filter?.end_date) qs.set('end_date', filter.end_date)
    if (filter?.page) qs.set('page', String(filter.page))
    if (filter?.page_size) qs.set('page_size', String(filter.page_size))
    const query = qs.toString()
    return api.get<AuditLogListResponse>(`/v1/audit-logs${query ? `?${query}` : ''}`)
  },
}
