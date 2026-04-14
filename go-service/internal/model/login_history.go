package model

import (
	"time"

	"github.com/google/uuid"
)

// LoginHistory 用户登录记录，用于安全审计。
type LoginHistory struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	TenantID  *uuid.UUID `gorm:"type:uuid;index"`
	IP        string     `gorm:"size:45"`
	UserAgent string     `gorm:"size:500"`
	LoginAt   time.Time  `gorm:"not null;default:now()"`
}

// TableName 指定表名，避免 GORM 自动复数化。
func (LoginHistory) TableName() string {
	return "login_history"
}
