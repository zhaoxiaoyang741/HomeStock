import { api } from '@/lib/api'
import type {
  AdjustStockLotPayload,
  InboundStockLotPayload,
  StockLot,
  StockLotListResponse,
  UpdateStockLotPayload,
} from '@/types/stock'

export const stockLotApi = {
  list: (params?: {
    material_id?: string
    category_id?: string
    location?: string
    status?: string
    keyword?: string
    expiring_soon?: boolean
  }) => {
    const qs = new URLSearchParams()
    if (params?.material_id) qs.set('material_id', params.material_id)
    if (params?.category_id) qs.set('category_id', params.category_id)
    if (params?.location) qs.set('location', params.location)
    if (params?.status) qs.set('status', params.status)
    if (params?.keyword) qs.set('keyword', params.keyword)
    if (params?.expiring_soon) qs.set('expiring_soon', 'true')
    const query = qs.toString()
    return api.get<StockLotListResponse>(`/v1/stock-lots${query ? `?${query}` : ''}`)
  },

  inbound: (payload: InboundStockLotPayload) => api.post<StockLot>('/v1/stock-lots/inbound', payload),

  update: (id: string, payload: UpdateStockLotPayload) => api.put<StockLot>(`/v1/stock-lots/${id}`, payload),

  adjust: (id: string, payload: AdjustStockLotPayload) =>
    api.post<StockLot>(`/v1/stock-lots/${id}/adjust`, payload),
}
