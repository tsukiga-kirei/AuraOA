package repository

import (
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// SystemPromptTemplateRepo 提供系统提示词模板的数据访问方法（全局，无租户隔离）。
type SystemPromptTemplateRepo struct {
	db *gorm.DB
}

// NewSystemPromptTemplateRepo 创建一个新的 SystemPromptTemplateRepo 实例。
func NewSystemPromptTemplateRepo(db *gorm.DB) *SystemPromptTemplateRepo {
	return &SystemPromptTemplateRepo{db: db}
}

// ListAll 查询所有系统提示词模板。
func (r *SystemPromptTemplateRepo) ListAll() ([]model.SystemPromptTemplate, error) {
	var templates []model.SystemPromptTemplate
	if err := r.db.Order("prompt_key ASC").Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}

// GetByKey 按 prompt_key 查询单条模板。
func (r *SystemPromptTemplateRepo) GetByKey(key string) (*model.SystemPromptTemplate, error) {
	var tpl model.SystemPromptTemplate
	if err := r.db.Where("prompt_key = ?", key).First(&tpl).Error; err != nil {
		return nil, err
	}
	return &tpl, nil
}

// GetByStrictness 查询指定尺度的所有模板（系统提示词和用户提示词均按尺度区分）。
func (r *SystemPromptTemplateRepo) GetByStrictness(strictness string) ([]model.SystemPromptTemplate, error) {
	var templates []model.SystemPromptTemplate
	if err := r.db.Where("strictness = ?", strictness).
		Order("prompt_type ASC, phase ASC").Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}
