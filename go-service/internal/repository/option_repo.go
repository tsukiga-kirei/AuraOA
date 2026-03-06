package repository

import (
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// OptionRepo 提供各类选项表的只读查询。
type OptionRepo struct {
	*BaseRepo
}

func NewOptionRepo(db *gorm.DB) *OptionRepo {
	return &OptionRepo{BaseRepo: NewBaseRepo(db)}
}

// ListOATypes 返回启用的OA系统类型选项，按排序字段排列。
func (r *OptionRepo) ListOATypes() ([]model.OATypeOption, error) {
	var items []model.OATypeOption
	if err := r.DB.Where("enabled = ?", true).Order("sort_order ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ListDBDrivers 返回启用的数据库驱动选项。
func (r *OptionRepo) ListDBDrivers() ([]model.DBDriverOption, error) {
	var items []model.DBDriverOption
	if err := r.DB.Where("enabled = ?", true).Order("sort_order ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ListAIDeployTypes 返回启用的AI部署类型选项。
func (r *OptionRepo) ListAIDeployTypes() ([]model.AIDeployTypeOption, error) {
	var items []model.AIDeployTypeOption
	if err := r.DB.Where("enabled = ?", true).Order("sort_order ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ListAIProviders 返回启用的AI服务商选项。
func (r *OptionRepo) ListAIProviders() ([]model.AIProviderOption, error) {
	var items []model.AIProviderOption
	if err := r.DB.Where("enabled = ?", true).Order("sort_order ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
