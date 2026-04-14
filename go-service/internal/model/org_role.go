package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// OrgRole 租户内的组织角色，控制页面级访问权限。
type OrgRole struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID        uuid.UUID      `gorm:"type:uuid;not null;index"`
	Name            string         `gorm:"size:100;not null"`
	Description     string         `gorm:"type:text"`
	PagePermissions datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'"`
	IsSystem        bool           `gorm:"not null;default:false"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
