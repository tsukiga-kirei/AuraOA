package model

import (
	"time"

	"github.com/google/uuid"
)

//User代表平台用户账号。
type User struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Username          string     `gorm:"uniqueIndex;size:100;not null"`
	PasswordHash      string     `gorm:"size:255;not null"`
	DisplayName       string     `gorm:"size:100;not null"`
	Email             string     `gorm:"size:255"`
	Phone             string     `gorm:"size:50"`
	AvatarURL         string     `gorm:"size:500"`
	Status            string     `gorm:"size:20;not null;default:active"` //活动|禁用|锁定
	PasswordChangedAt time.Time  `gorm:"default:now()"`
	LoginFailCount    int        `gorm:"not null;default:0"`
	LockedUntil       *time.Time
	Locale            string     `gorm:"size:10;default:zh-CN"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
