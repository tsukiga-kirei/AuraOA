export interface UserNotificationItem {
  id: string
  category: string
  title: string
  body?: string
  link_path?: string
  read: boolean
  created_at: string
}

export interface UserNotificationListResponse {
  items: UserNotificationItem[]
  total: number
}

export interface UserNotificationUnreadResponse {
  count: number
}
