package repository

import (
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// SystemConfigRepo provides data access for system_configs table.
type SystemConfigRepo struct {
	*BaseRepo
}

func NewSystemConfigRepo(db *gorm.DB) *SystemConfigRepo {
	return &SystemConfigRepo{BaseRepo: NewBaseRepo(db)}
}

// ListAll returns all system config entries.
func (r *SystemConfigRepo) ListAll() ([]model.SystemConfig, error) {
	var configs []model.SystemConfig
	if err := r.DB.Order("key ASC").Find(&configs).Error; err != nil {
		return nil, err
	}
	return configs, nil
}

// UpdateByKey updates the value of a config entry by key. Creates if not exists.
func (r *SystemConfigRepo) UpdateByKey(key, value string) error {
	return r.DB.Model(&model.SystemConfig{}).
		Where("key = ?", key).
		Update("value", value).Error
}
