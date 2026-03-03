package model

import (
	"time"

	"github.com/google/uuid"
)

//LoginHistory 记录每个用户登录尝试以用于审核目的。
type LoginHistory struct {
	ID       uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID   uuid.UUID  `gorm:"type:uuid;not null;index"`
	TenantID *uuid.UUID `gorm:"type:uuid;index"`
	IP       string     `gorm:"size:45"`
	UserAgent string    `gorm:"size:500"`
	LoginAt  time.Time  `gorm:"not null;default:now()"`
}
