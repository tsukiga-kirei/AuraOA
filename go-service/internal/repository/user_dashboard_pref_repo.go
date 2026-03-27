package repository

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// UserDashboardPrefRepo 提供用户仪表板偏好的数据访问方法。
type UserDashboardPrefRepo struct {
	*BaseRepo
}

// NewUserDashboardPrefRepo 创建一个新的 UserDashboardPrefRepo 实例。
func NewUserDashboardPrefRepo(db *gorm.DB) *UserDashboardPrefRepo {
	return &UserDashboardPrefRepo{BaseRepo: NewBaseRepo(db)}
}

// GetPref 查询仪表板偏好。tenantID=nil 且 scope=platform 为系统管理员平台布局。
func (r *UserDashboardPrefRepo) GetPref(tenantID *uuid.UUID, userID uuid.UUID, prefScope string) (*model.UserDashboardPref, error) {
	var pref model.UserDashboardPref
	q := r.DB.Where("user_id = ? AND pref_scope = ?", userID, prefScope)
	if tenantID == nil {
		q = q.Where("tenant_id IS NULL")
	} else {
		q = q.Where("tenant_id = ?", *tenantID)
	}
	err := q.First(&pref).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &pref, nil
}

// Upsert 创建或更新用户仪表板偏好（含 pref_scope）。
func (r *UserDashboardPrefRepo) Upsert(pref *model.UserDashboardPref) error {
	var existing model.UserDashboardPref
	q := r.DB.Where("user_id = ? AND pref_scope = ?", pref.UserID, pref.PrefScope)
	if pref.TenantID == nil {
		q = q.Where("tenant_id IS NULL")
	} else {
		q = q.Where("tenant_id = ?", *pref.TenantID)
	}
	err := q.First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if pref.CreatedAt.IsZero() {
			pref.CreatedAt = time.Now()
		}
		if pref.UpdatedAt.IsZero() {
			pref.UpdatedAt = time.Now()
		}
		return r.DB.Create(pref).Error
	}
	if err != nil {
		return err
	}
	return r.DB.Model(&existing).Updates(map[string]interface{}{
		"enabled_widgets": pref.EnabledWidgets,
		"widget_sizes":    pref.WidgetSizes,
		"updated_at":      pref.UpdatedAt,
	}).Error
}
