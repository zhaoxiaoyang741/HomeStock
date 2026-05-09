import { api } from '@/lib/api'
import type { ModelListData, UpdateModelPayload, SwapModelPayload } from '@/types/model'
import type { Result } from '@/types/api'

export async function getModelList(): Promise<ModelListData> {
  const res = await api.get<Result<ModelListData>>('/v1/models')
  return res.data
}

export async function updateModelConfig(payload: UpdateModelPayload): Promise<void> {
  await api.patch<Result<unknown>>('/v1/models', payload)
}

export async function swapActiveModel(payload: SwapModelPayload): Promise<void> {
  await api.post<Result<unknown>>('/v1/models/swap', payload)
}
