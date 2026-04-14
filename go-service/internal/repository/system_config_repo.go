package repository

import (
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// SystemConfigRepo 提供 system_configs 表的数据访问方法（全局，无租户隔离）。
type SystemConfigRepo struct {
	*BaseRepo
}

// NewSystemConfigRepo 创建 SystemConfigRepo 实例。
func NewSystemConfigRepo(db *gorm.DB) *SystemConfigRepo {
	return &SystemConfigRepo{BaseRepo: NewBaseRepo(db)}
}

// ListAll 查询所有系统配置项，按键名字母序排列。
// 返回完整的配置列表，供系统设置页面展示。
func (r *SystemConfigRepo) ListAll() ([]model.SystemConfig, error) {
	var configs []model.SystemConfig
	if err := r.DB.Order("key ASC").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// UpdateByKey 按键名更新配置值；若键不存在则不做任何操作（不自动创建）。
func (r *SystemConfigRepo) UpdateByKey(key, value string) error {
	return r.DB.Model(&model.SystemConfig{}).
		Where("key = ?", key).
		Update("value", value).Error
}

// FindByKey 按键名查询单条配置，返回其 value 字段。
// 若键不存在则返回空字符串和 gorm.ErrRecordNotFound 错误。
func (r *SystemConfigRepo) FindByKey(key string) (string, error) {
	var config model.SystemConfig
	if err := r.DB.Where("key = ?", key).First(&config).Error; err != nil {
		return "", err
	}
	return config.Value, nil
}
