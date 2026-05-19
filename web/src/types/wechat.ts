export interface WechatStatus {
  connected: boolean
  enabled: boolean
  has_token: boolean
  account_id: string
}

export interface UpdateWechatConfigPayload {
  enabled?: boolean
}

export interface WechatQRFlowResponse {
  flow_id: string
  status: 'wait' | 'scaned' | 'confirmed' | 'expired' | 'error'
  qr_data_uri?: string
  account_id?: string
  error?: string
}
