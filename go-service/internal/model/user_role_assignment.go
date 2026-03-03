package model

import (
	"time"

	"github.com/google/uuid"
)

// UserRoleAssignment maps a user to a system-level role (business|tenant_admin|system_admin).
type UserRoleAssignment struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index"`
	Role      string     `gorm:"size:30;not null"` // business|tenant_admin|system_admin
	TenantID  *uuid.UUID `gorm:"type:uuid;index"`  // NULL for system_admin
	Label     string     `gorm:"size:200"`
	IsDefault bool       `gorm:"not null;default:false"`
	CreatedAt time.Time
}
