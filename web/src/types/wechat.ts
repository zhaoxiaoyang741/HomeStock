export interface WechatQrCode {
  qr_url: string
  uuid: string
  status: string
}

export interface WechatStatus {
  connected: boolean
  logged_in: boolean
  enabled: boolean
}
