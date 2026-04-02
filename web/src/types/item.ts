export type ItemStatus = 'normal' | 'low' | 'expiring' | 'expired'

export interface Item {
  id: number
  name: string
  category_id: number
  quantity: number
  unit: string
  min_quantity: number
  expiry_date?: string
  status: ItemStatus
  notes?: string
  created_at: string
  updated_at: string
}

export interface CreateItemPayload {
  name: string
  category_id: number
  quantity: number
  unit: string
  min_quantity: number
  expiry_date?: string
  notes?: string
}

export interface UpdateItemPayload extends Partial<CreateItemPayload> {
  id: number
}
