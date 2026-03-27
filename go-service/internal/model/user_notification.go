package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// UserNotification 用户通知，与 JWT active_role.id（user_role_assignments.id）绑定。
type UserNotification struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID           uuid.UUID      `gorm:"type:uuid;not null;index"`
	RoleAssignmentID uuid.UUID      `gorm:"type:uuid;not null;index"`
	Category         string         `gorm:"size:64;not null;default:general"`
	Title            string         `gorm:"type:text;not null"`
	Body             string         `gorm:"type:text"`
	LinkPath         string         `gorm:"type:text"`
	ReadAt           *time.Time
	Metadata         datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'"`
	CreatedAt        time.Time
}

func (UserNotification) TableName() string { return "user_notifications" }
