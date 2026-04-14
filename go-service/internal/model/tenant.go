package model

import (
	"time"

	"github.com/google/uuid"
)

// Tenant 代表多租户平台中的租户。
type Tenant struct {
	ID                  uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name                string     `gorm:"size:255;not null"`
	Code                string     `gorm:"uniqueIndex;size:100;not null"`
	Description         string     `gorm:"type:text"`
	Status              string     `gorm:"size:20;not null;default:active"` // active | inactive（启用 | 停用）
	OADBConnectionID    *uuid.UUID `gorm:"type:uuid;column:oa_db_connection_id"`
	TokenQuota          int        `gorm:"not null;default:10000"`
	TokenUsed           int        `gorm:"not null;default:0"`
	MaxConcurrency      int        `gorm:"not null;default:10"`
	PrimaryModelID      *uuid.UUID `gorm:"type:uuid"`
	FallbackModelID     *uuid.UUID `gorm:"type:uuid"`
	MaxTokensPerRequest int        `gorm:"not null;default:8192"`
	Temperature         float64    `gorm:"type:decimal(3,2);not null;default:0.30"`
	TimeoutSeconds      int        `gorm:"not null;default:60"`
	RetryCount          int        `gorm:"not null;default:3"`
	SSOEnabled          bool       `gorm:"not null;default:false"`
	SSOEndpoint         string     `gorm:"size:500"`
	LogRetentionDays    int        `gorm:"not null;default:365"`
	DataRetentionDays   int        `gorm:"not null;default:1095"`
	ContactName         string     `gorm:"size:100"`
	ContactEmail        string     `gorm:"size:255"`
	ContactPhone        string     `gorm:"size:50"`
	AdminUserID         *uuid.UUID `gorm:"type:uuid"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}
