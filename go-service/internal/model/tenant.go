package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

//租户代表多租户平台中的租户。
type Tenant struct {
	ID                uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	Name              string         `gorm:"size:255;not null"`
	Code              string         `gorm:"uniqueIndex;size:100;not null"`
	Description       string         `gorm:"type:text"`
	Status            string         `gorm:"size:20;not null;default:active"` //活跃|不活跃
	OAType            string         `gorm:"size:50;not null;default:weaver_e9"`
	OADBConnectionID  *uuid.UUID     `gorm:"type:uuid"`
	TokenQuota        int            `gorm:"not null;default:10000"`
	TokenUsed         int            `gorm:"not null;default:0"`
	MaxConcurrency    int            `gorm:"not null;default:10"`
	AIConfig          datatypes.JSON `gorm:"type:jsonb;not null"`
	SSOEnabled        bool           `gorm:"not null;default:false"`
	SSOEndpoint       string         `gorm:"size:500"`
	LogRetentionDays  int            `gorm:"not null;default:365"`
	DataRetentionDays int            `gorm:"not null;default:1095"`
	AllowCustomModel  bool           `gorm:"not null;default:false"`
	ContactName       string         `gorm:"size:100"`
	ContactEmail      string         `gorm:"size:255"`
	ContactPhone      string         `gorm:"size:50"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
