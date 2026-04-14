package repository

import (
	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// AIModelRepo 提供 AI 模型配置的数据访问方法（全局，无租户隔离）。
type AIModelRepo struct {
	*BaseRepo
}

// NewAIModelRepo 创建 AIModelRepo 实例。
func NewAIModelRepo(db *gorm.DB) *AIModelRepo {
	return &AIModelRepo{BaseRepo: NewBaseRepo(db)}
}

// List 查询所有 AI 模型配置，按创建时间升序排列。
func (r *AIModelRepo) List() ([]model.AIModelConfig, error) {
	var items []model.AIModelConfig
	if err := r.DB.Order("created_at ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// ListEnabled 查询所有已启用的 AI 模型配置，按创建时间升序排列。
func (r *AIModelRepo) ListEnabled() ([]model.AIModelConfig, error) {
	var items []model.AIModelConfig
	if err := r.DB.Where("enabled = ?", true).Order("created_at ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

// FindByID 按 UUID 查询单个 AI 模型配置，不存在时返回 gorm.ErrRecordNotFound。
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

// Update 按 ID 更新 AI 模型配置的指定字段。
func (r *AIModelRepo) Update(id uuid.UUID, fields map[string]interface{}) error {
	return r.DB.Model(&model.AIModelConfig{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 按 ID 删除 AI 模型配置记录。
func (r *AIModelRepo) Delete(id uuid.UUID) error {
	return r.DB.Where("id = ?", id).Delete(&model.AIModelConfig{}).Error
}
