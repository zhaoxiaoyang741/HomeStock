import { api } from '@/lib/api'
import type { WechatQrCode, WechatStatus } from '@/types/wechat'
import type { Result } from '@/types/api'

export async function getWechatQrCode(): Promise<WechatQrCode> {
  const res = await api.get<Result<WechatQrCode>>('/v1/wechat/qrcode')
  return res.data
}

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
