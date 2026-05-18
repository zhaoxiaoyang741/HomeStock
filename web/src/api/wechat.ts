import { api } from '@/lib/api'
import type { WechatStatus, WechatQRFlowResponse } from '@/types/wechat'
import type { Result } from '@/types/api'

export async function getWechatStatus(): Promise<WechatStatus> {
  const res = await api.get<Result<WechatStatus>>('/v1/wechat/status')
  return res.data
}

export async function disconnectWechat(): Promise<void> {
  await api.post<Result<unknown>>('/v1/wechat/disconnect', {})
}

export async function reconnectWechat(): Promise<void> {
  await api.post<Result<unknown>>('/v1/wechat/reconnect', {})
}

export async function startWechatQRFlow(): Promise<WechatQRFlowResponse> {
  const res = await api.post<Result<WechatQRFlowResponse>>('/v1/wechat/qrcode', {})
  return res.data
}

export async function pollWechatQRFlow(flowID: string): Promise<WechatQRFlowResponse> {
  const res = await api.get<Result<WechatQRFlowResponse>>(`/v1/wechat/qrcode/${encodeURIComponent(flowID)}`)
  return res.data
}
