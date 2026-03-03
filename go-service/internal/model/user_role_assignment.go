package model

import (
	"time"

	"github.com/google/uuid"
)

//UserRoleAssignment 将用户映射到系统级角色 (business|tenant_admin|system_admin)。
type UserRoleAssignment struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	Role      string     `gorm:"size:30;not null"` //业务|租户管理员|系统管理员
	TenantID  *uuid.UUID `gorm:"type:uuid;index"`  //system_admin 为 NULL
	Label     string     `gorm:"size:200"`
	IsDefault bool       `gorm:"not null;default:false"`
	CreatedAt time.Time
}
