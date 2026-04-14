package repository

import (
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// OptionRepo 提供各类系统选项表的只读查询（全局，无租户隔离）。
type OptionRepo struct {
	*BaseRepo
}

// NewOptionRepo 创建 OptionRepo 实例。
func NewOptionRepo(db *gorm.DB) *OptionRepo {
	return &OptionRepo{BaseRepo: NewBaseRepo(db)}
}

// ListOATypes 查询所有已启用的 OA 系统类型选项，按 sort_order 升序排列。
func (r *OptionRepo) ListOATypes() ([]model.OATypeOption, error) {
	var items []model.OATypeOption
	if err := r.DB.Where("enabled = ?", true).Order("sort_order ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ListDBDrivers 查询所有已启用的数据库驱动选项，按 sort_order 升序排列。
func (r *OptionRepo) ListDBDrivers() ([]model.DBDriverOption, error) {
	var items []model.DBDriverOption
	if err := r.DB.Where("enabled = ?", true).Order("sort_order ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ListAIDeployTypes 查询所有已启用的 AI 部署类型选项，按 sort_order 升序排列。
func (r *OptionRepo) ListAIDeployTypes() ([]model.AIDeployTypeOption, error) {
	var items []model.AIDeployTypeOption
	if err := r.DB.Where("enabled = ?", true).Order("sort_order ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ListAIProviders 查询所有已启用的 AI 服务商选项，按 sort_order 升序排列。
func (r *OptionRepo) ListAIProviders() ([]model.AIProviderOption, error) {
	var items []model.AIProviderOption
	if err := r.DB.Where("enabled = ?", true).Order("sort_order ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
