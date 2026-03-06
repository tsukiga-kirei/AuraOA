package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// AIModelRepo 提供 AI 模型配置的数据访问。
type AIModelRepo struct {
	*BaseRepo
}

func NewAIModelRepo(db *gorm.DB) *AIModelRepo {
	return &AIModelRepo{BaseRepo: NewBaseRepo(db)}
}

// List 返回所有 AI 模型配置，按创建时间排序。
func (r *AIModelRepo) List() ([]model.AIModelConfig, error) {
	var items []model.AIModelConfig
	if err := r.DB.Order("created_at ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ListEnabled 返回启用的 AI 模型列表。
func (r *AIModelRepo) ListEnabled() ([]model.AIModelConfig, error) {
	var items []model.AIModelConfig
	if err := r.DB.Where("enabled = ?", true).Order("created_at ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// FindByID 通过 UUID 查找 AI 模型配置。
func (r *AIModelRepo) FindByID(id uuid.UUID) (*model.AIModelConfig, error) {
	var item model.AIModelConfig
	if err := r.DB.Where("id = ?", id).First(&item).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

// Create 创建新的 AI 模型配置记录。
func (r *AIModelRepo) Create(item *model.AIModelConfig) error {
	return r.DB.Create(item).Error
}

// Update 更新 AI 模型配置。
func (r *AIModelRepo) Update(id uuid.UUID, fields map[string]interface{}) error {
	return r.DB.Model(&model.AIModelConfig{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除 AI 模型配置。
func (r *AIModelRepo) Delete(id uuid.UUID) error {
	return r.DB.Where("id = ?", id).Delete(&model.AIModelConfig{}).Error
}
