package model

import (
	"time"

	"github.com/google/uuid"
)

// UserRoleAssignment 用户系统角色绑定，支持 business / tenant_admin / system_admin 三种角色。
type UserRoleAssignment struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	Role      string     `gorm:"size:30;not null"` // business | tenant_admin | system_admin
	TenantID  *uuid.UUID `gorm:"type:uuid;index"`  // system_admin 角色时为 NULL
	Label     string     `gorm:"size:200"`
	IsDefault bool       `gorm:"not null;default:false"`
	CreatedAt time.Time
}
