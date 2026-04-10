import type { Category } from '@/types/category'

export type InventoryStatus = 'normal' | 'expiring' | 'expired'

export interface Material {
  id: string
  tenant_id: string
  name: string
  spec: string
  category_id: string
  category: Category | null
  default_unit: string
  status: string
  created_at: string
  updated_at: string
}

export interface MaterialSummary extends Material {
  total_quantity: number
  lot_count: number
  nearest_expire_at: string | null
  locations: string[]
}

export interface MaterialDetail extends MaterialSummary {}

export interface CreateMaterialPayload {
  name: string
  spec?: string
  category_id?: string
  default_unit?: string
}

export interface ConsumeMaterialPayload {
  quantity: number
  reason?: string
}

export interface ConsumedLotResult {
  lot_id: string
  consumed_quantity: number
  remaining_quantity: number
  location: string
  expire_at: string | null
}

export interface ConsumeMaterialResponse {
  material: MaterialDetail
  requested_quantity: number
  unit: string
  consumed_lots: ConsumedLotResult[]
}

export interface MaterialListResponse {
  materials: MaterialSummary[]
  total: number
}

export function getInventoryStatus(dateLike: string | null | undefined): InventoryStatus {
  if (!dateLike) return 'normal'
  const now = new Date()
  const target = new Date(dateLike)
  if (Number.isNaN(target.getTime())) return 'normal'
  if (target < now) return 'expired'
  const days = (target.getTime() - now.getTime()) / (1000 * 60 * 60 * 24)
  if (days <= 30) return 'expiring'
  return 'normal'
}
