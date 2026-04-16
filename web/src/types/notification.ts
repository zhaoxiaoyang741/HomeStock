export interface NotificationRecord {
  id: string
  lot_id: string
  notify_at: string
  status: 'pending' | 'sent' | 'failed'
  channel: string
  message: string
  created_at: string
  lot?: {
    id: string
    material?: {
      name: string
      spec?: string
    }
    expire_at?: string
    quantity_on_hand: number
    unit: string
  }
}

export interface NotificationPage {
  items: NotificationRecord[]
  total: number
  page: number
  page_size: number
}
