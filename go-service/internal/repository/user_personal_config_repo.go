package repository

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"oa-smart-audit/go-service/internal/model"
)

// UserPersonalConfigRepo 提供用户个人配置的数据访问方法，按 tenant_id + user_id 唯一约束。
type UserPersonalConfigRepo struct {
	*BaseRepo
}

// NewUserPersonalConfigRepo 创建一个新的 UserPersonalConfigRepo 实例。
func NewUserPersonalConfigRepo(db *gorm.DB) *UserPersonalConfigRepo {
	return &UserPersonalConfigRepo{BaseRepo: NewBaseRepo(db)}
}

// GetByTenantAndUser 查询指定租户和用户的个人配置，不存在时返回 nil。
func (r *UserPersonalConfigRepo) GetByTenantAndUser(c *gin.Context, tenantID, userID uuid.UUID) (*model.UserPersonalConfig, error) {
	var cfg model.UserPersonalConfig
	err := r.DB.Where("tenant_id = ? AND user_id = ?", tenantID, userID).First(&cfg).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Upsert 创建或更新用户个人配置（基于 tenant_id + user_id 唯一约束）。
func (r *UserPersonalConfigRepo) Upsert(cfg *model.UserPersonalConfig) error {
	return r.DB.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"audit_details", "cron_details", "archive_details", "updated_at"}),
	}).Create(cfg).Error
}

// UpdateFields 通过 map 更新指定字段。
func (r *UserPersonalConfigRepo) UpdateFields(tenantID, userID uuid.UUID, fields map[string]interface{}) error {
	return r.DB.Model(&model.UserPersonalConfig{}).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Updates(fields).Error
}

// ListByTenant 查询当前租户所有用户的个人配置摘要。
func (r *UserPersonalConfigRepo) ListByTenant(c *gin.Context) ([]model.UserPersonalConfig, error) {
	var configs []model.UserPersonalConfig
	if err := r.WithTenant(c).Order("created_at ASC").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// GetByUserID 查询当前租户下指定用户的个人配置。
func (r *UserPersonalConfigRepo) GetByUserID(c *gin.Context, userID uuid.UUID) (*model.UserPersonalConfig, error) {
	var cfg model.UserPersonalConfig
	err := r.WithTenant(c).Where("user_id = ?", userID).First(&cfg).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
