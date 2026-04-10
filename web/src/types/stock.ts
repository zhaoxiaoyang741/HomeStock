import type { Material } from '@/types/material'

export type StockMovementType = 'inbound' | 'consume' | 'adjustment' | 'void'

export interface StockLot {
  id: string
  tenant_id: string
  material_id: string
  material: Material | null
  quantity_on_hand: number
  unit: string
  location: string
  expire_at: string | null
  purchased_at: string | null
  received_at: string
  notes: string
  status: string
  created_at: string
  updated_at: string
}

export interface InboundStockLotPayload {
  material_id?: string
  name?: string
  spec?: string
  category_id?: string
  quantity: number
  unit?: string
  location?: string
  expire_at?: string
  purchased_at?: string
  notes?: string
}

export interface UpdateStockLotPayload {
  expire_at?: string
  purchased_at?: string
  location?: string
  notes?: string
}

export interface AdjustStockLotPayload {
  target_quantity: number
  reason?: string
  remark?: string
}

export interface StockLotListResponse {
  lots: StockLot[]
  total: number
}

export interface StockMovement {
  id: string
  tenant_id: string
  material_id: string
  material: Material | null
  lot_id: string
  lot: StockLot | null
  movement_type: StockMovementType
  quantity_delta: number
  unit: string
  reason: string
  channel: string
  user_name: string
  user_id: string
  remark: string
  created_at: string
}

export interface StockMovementListResponse {
  movements: StockMovement[]
  total: number
}
