import { api } from '@/lib/api'
import type { AuditLogFilter, AuditLogListResponse } from '@/types/audit'
import type { Page, Result } from '@/types/api'
import type { AuditLog } from '@/types/audit'

export const auditLogApi = {
  list: async (filter?: AuditLogFilter): Promise<AuditLogListResponse> => {
    const qs = new URLSearchParams()
    if (filter?.action) qs.set('action', filter.action)
    if (filter?.channel) qs.set('channel', filter.channel)
    if (filter?.user_name) qs.set('user_name', filter.user_name)
    if (filter?.start_date) qs.set('start_date', filter.start_date)
    if (filter?.end_date) qs.set('end_date', filter.end_date)
    if (filter?.page) qs.set('page', String(filter.page))
    if (filter?.page_size) qs.set('page_size', String(filter.page_size))
    const query = qs.toString()
    const res = await api.get<Result<Page<AuditLog>>>(`/v1/audit-logs${query ? `?${query}` : ''}`)
    return { logs: res.data.items, total: res.data.total, page: res.data.page ?? 1, page_size: res.data.page_size ?? res.data.items.length }
  },
}
