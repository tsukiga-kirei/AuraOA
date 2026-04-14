package model

import (
	"time"

	"github.com/google/uuid"
)

// Department 租户内的组织部门。
type Department struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TenantID  uuid.UUID  `gorm:"type:uuid;not null;index"`
	Name      string     `gorm:"size:200;not null"`
	ParentID  *uuid.UUID `gorm:"type:uuid;index"`
	Manager   string     `gorm:"size:100"`
	SortOrder int        `gorm:"not null;default:0"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
