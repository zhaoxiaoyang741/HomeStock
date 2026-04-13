import { api } from '@/lib/api'
import type {
  AdjustStockLotPayload,
  InboundStockLotPayload,
  StockLot,
  StockLotListResponse,
  UpdateStockLotPayload,
} from '@/types/stock'
import type { Page, Result } from '@/types/api'

export const stockLotApi = {
  list: async (params?: {
    material_id?: string
    category_id?: string
    location?: string
    status?: string
    keyword?: string
    expiring_soon?: boolean
  }): Promise<StockLotListResponse> => {
    const qs = new URLSearchParams()
    if (params?.material_id) qs.set('material_id', params.material_id)
    if (params?.category_id) qs.set('category_id', params.category_id)
    if (params?.location) qs.set('location', params.location)
    if (params?.status) qs.set('status', params.status)
    if (params?.keyword) qs.set('keyword', params.keyword)
    if (params?.expiring_soon) qs.set('expiring_soon', 'true')
    const query = qs.toString()
    const res = await api.get<Result<Page<StockLot>>>(`/v1/stock-lots${query ? `?${query}` : ''}`)
    return { lots: res.data.items, total: res.data.total }
  },

  inbound: async (payload: InboundStockLotPayload): Promise<StockLot> => {
    const res = await api.post<Result<StockLot>>('/v1/stock-lots/inbound', payload)
    return res.data
  },

  update: async (id: string, payload: UpdateStockLotPayload): Promise<StockLot> => {
    const res = await api.put<Result<StockLot>>(`/v1/stock-lots/${id}`, payload)
    return res.data
  },

  adjust: async (id: string, payload: AdjustStockLotPayload): Promise<StockLot> => {
    const res = await api.post<Result<StockLot>>(`/v1/stock-lots/${id}/adjust`, payload)
    return res.data
  },

  void: async (id: string): Promise<StockLot> => {
    const res = await api.post<Result<StockLot>>(`/v1/stock-lots/${id}/void`, {})
    return res.data
  },
}
