package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// 仪表盘布局存储维度（与 JWT active_role 对齐，见迁移 000021）。
const (
	DashboardPrefScopePlatform    = "platform"
	DashboardPrefScopeBusiness    = "business"
	DashboardPrefScopeTenantAdmin = "tenant_admin"
)

// UserDashboardPref 用户仪表板偏好。
// 租户内：tenant_id 非空，(tenant_id, user_id, pref_scope) 唯一，scope 为 business / tenant_admin。
// 系统管理员平台：tenant_id 为空，pref_scope=platform，按 user_id 唯一（迁移 000020 + 000021）。
type UserDashboardPref struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID       *uuid.UUID     `gorm:"type:uuid" json:"tenant_id,omitempty"`
	UserID         uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
	PrefScope      string         `gorm:"type:varchar(32);not null" json:"pref_scope"`
	EnabledWidgets datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"enabled_widgets"`
	WidgetSizes    datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"widget_sizes"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

func (UserDashboardPref) TableName() string { return "user_dashboard_prefs" }
