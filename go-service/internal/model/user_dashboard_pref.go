package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// UserDashboardPref 用户仪表板偏好，按 tenant_id + user_id 唯一约束。
type UserDashboardPref struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID       uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	UserID         uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
	EnabledWidgets datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"enabled_widgets"`
	WidgetSizes    datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"widget_sizes"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

func (UserDashboardPref) TableName() string { return "user_dashboard_prefs" }
