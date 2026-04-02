export type ItemStatus = 'normal' | 'expiring' | 'expired'

export interface Category {
  id: string
  tenant_id: string
  name: string
  created_at: string
  updated_at: string
}

export interface Item {
  id: string
  tenant_id: string
  name: string
  category_id: string
  category: Category | null
  quantity: number
  unit: string
  location: string
  expire_at: string | null
  purchased_at: string
  notes: string
  created_at: string
  updated_at: string
}

export interface CreateItemPayload {
  name: string
  category_id?: string
  quantity?: number
  unit?: string
  location?: string
  expire_at?: string
  purchased_at?: string
  notes?: string
}

export interface UpdateItemPayload {
  name?: string
  category_id?: string
  quantity?: number
  unit?: string
  location?: string
  expire_at?: string
  purchased_at?: string
  notes?: string
}

export interface ItemListResponse {
  items: Item[]
  total: number
}

export function getItemStatus(item: Item): ItemStatus {
  if (!item.expire_at) return 'normal'
  const now = new Date()
  const expireDate = new Date(item.expire_at)
  if (expireDate < now) return 'expired'
  const days = (expireDate.getTime() - now.getTime()) / (1000 * 60 * 60 * 24)
  if (days <= 30) return 'expiring'
  return 'normal'
}
