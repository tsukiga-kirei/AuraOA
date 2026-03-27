package dto

import "time"

// UserNotificationItem 单条通知（API 输出）。
type UserNotificationItem struct {
	ID        string    `json:"id"`
	Category  string    `json:"category"`
	Title     string    `json:"title"`
	Body      string    `json:"body,omitempty"`
	LinkPath  string    `json:"link_path,omitempty"`
	Read      bool      `json:"read"`
	CreatedAt time.Time `json:"created_at"`
}

// UserNotificationListResponse GET /api/auth/notifications
type UserNotificationListResponse struct {
	Items []UserNotificationItem `json:"items"`
	Total int64                  `json:"total"`
}

// UserNotificationUnreadResponse GET /api/auth/notifications/unread-count
type UserNotificationUnreadResponse struct {
	Count int64 `json:"count"`
}
