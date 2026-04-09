export type AuditAction = 'create' | 'update' | 'delete'
export type AuditEntityType = 'item' | 'category'

export interface AuditLog {
  id: string
  tenant_id: string
  user_name: string
  user_id: string
  channel: string
  action: AuditAction
  entity_type: AuditEntityType
  entity_id: string
  entity_name: string
  changes_detail: string // JSON string: { before?: object, after?: object }
  created_at: string
}

export interface AuditLogListResponse {
  logs: AuditLog[]
  total: number
  page: number
  page_size: number
}

export interface AuditLogFilter {
  action?: string
  channel?: string
  user_name?: string
  start_date?: string
  end_date?: string
  page?: number
  page_size?: number
}

export interface ChangesDetail {
  before?: Record<string, unknown>
  after?: Record<string, unknown>
}

export function parseChangesDetail(raw: string): ChangesDetail {
  try {
    return JSON.parse(raw) as ChangesDetail
  } catch {
    return {}
  }
}
