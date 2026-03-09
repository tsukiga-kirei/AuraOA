package repository

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

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

// GetByTenantAndUser 查询指定租户和用户的仪表板偏好。
func (r *UserDashboardPrefRepo) GetByTenantAndUser(c *gin.Context, tenantID, userID uuid.UUID) (*model.UserDashboardPref, error) {
	var pref model.UserDashboardPref
	err := r.DB.Where("tenant_id = ? AND user_id = ?", tenantID, userID).First(&pref).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &pref, nil
}

// Upsert 创建或更新用户仪表板偏好（基于 tenant_id + user_id 唯一约束）。
func (r *UserDashboardPrefRepo) Upsert(pref *model.UserDashboardPref) error {
	return r.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"enabled_widgets", "widget_sizes", "updated_at"}),
	}).Create(pref).Error
}
