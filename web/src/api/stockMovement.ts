import { api } from '@/lib/api'
import type { StockMovementListResponse, StockMovement } from '@/types/stock'
import type { Page, Result } from '@/types/api'

export const stockMovementApi = {
  list: async (params?: {
    material_id?: string
    lot_id?: string
    movement_type?: string
    start_date?: string
    end_date?: string
  }): Promise<StockMovementListResponse> => {
    const qs = new URLSearchParams()
    if (params?.material_id) qs.set('material_id', params.material_id)
    if (params?.lot_id) qs.set('lot_id', params.lot_id)
    if (params?.movement_type) qs.set('movement_type', params.movement_type)
    if (params?.start_date) qs.set('start_date', params.start_date)
    if (params?.end_date) qs.set('end_date', params.end_date)
    const query = qs.toString()
    const res = await api.get<Result<Page<StockMovement>>>(`/v1/stock-movements${query ? `?${query}` : ''}`)
    return { movements: res.data.items, total: res.data.total }
  },
}
