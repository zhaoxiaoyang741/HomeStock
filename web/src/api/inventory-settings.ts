import { api } from '@/lib/api'
import type { Result } from '@/types/api'

export interface InventoryConfig {
  default_location: string
  default_expiry_days: number
  default_quantity: number
  due_soon_days: number
  track_price: boolean
  track_opened: boolean
  auto_add_shopping_list: boolean
  nlu_default_quantity: number
  nlu_auto_select_threshold: number
  nlu_auto_select_lead: number
}

export async function getInventoryConfig(): Promise<InventoryConfig> {
  const res = await api.get<Result<InventoryConfig>>('/v1/settings/inventory')
  return res.data
}

export async function updateInventoryConfig(payload: Partial<InventoryConfig>): Promise<InventoryConfig> {
  const res = await api.put<Result<InventoryConfig>>('/v1/settings/inventory', payload)
  return res.data
}
