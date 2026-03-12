package repository

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// ProcessArchiveConfigRepo 提供归档复盘配置的数据访问方法，按租户隔离。
type ProcessArchiveConfigRepo struct {
	*BaseRepo
}

// NewProcessArchiveConfigRepo 创建一个新的 ProcessArchiveConfigRepo 实例。
func NewProcessArchiveConfigRepo(db *gorm.DB) *ProcessArchiveConfigRepo {
	return &ProcessArchiveConfigRepo{BaseRepo: NewBaseRepo(db)}
}

// Create 创建归档复盘配置。
func (r *ProcessArchiveConfigRepo) Create(c *gin.Context, cfg *model.ProcessArchiveConfig) error {
	return r.WithTenant(c).Create(cfg).Error
}

// GetByID 通过 ID 查询单个归档复盘配置（含租户隔离）。
func (r *ProcessArchiveConfigRepo) GetByID(c *gin.Context, id uuid.UUID) (*model.ProcessArchiveConfig, error) {
	var cfg model.ProcessArchiveConfig
	if err := r.WithTenant(c).Where("id = ?", id).First(&cfg).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

// ListByTenant 查询当前租户的所有归档复盘配置。
func (r *ProcessArchiveConfigRepo) ListByTenant(c *gin.Context) ([]model.ProcessArchiveConfig, error) {
	var cfgs []model.ProcessArchiveConfig
	if err := r.WithTenant(c).Order("created_at ASC").Find(&cfgs).Error; err != nil {
		return nil, err
	}
	return cfgs, nil
}

// ExistsByProcessType 检查租户内是否已存在相同流程类型的配置。
func (r *ProcessArchiveConfigRepo) ExistsByProcessType(c *gin.Context, processType string) (bool, error) {
	var count int64
	if err := r.WithTenant(c).Model(&model.ProcessArchiveConfig{}).
		Where("process_type = ?", processType).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// UpdateFields 通过字段 map 更新归档复盘配置。
func (r *ProcessArchiveConfigRepo) UpdateFields(c *gin.Context, id uuid.UUID, fields map[string]interface{}) error {
	return r.WithTenant(c).Model(&model.ProcessArchiveConfig{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 删除归档复盘配置。
func (r *ProcessArchiveConfigRepo) Delete(c *gin.Context, id uuid.UUID) error {
	return r.WithTenant(c).Where("id = ?", id).Delete(&model.ProcessArchiveConfig{}).Error
}

// ArchiveRuleRepo 提供归档规则的数据访问方法，按租户隔离。
type ArchiveRuleRepo struct {
	*BaseRepo
}

// NewArchiveRuleRepo 创建一个新的 ArchiveRuleRepo 实例。
func NewArchiveRuleRepo(db *gorm.DB) *ArchiveRuleRepo {
	return &ArchiveRuleRepo{BaseRepo: NewBaseRepo(db)}
}

// Create 创建归档规则记录。
func (r *ArchiveRuleRepo) Create(c *gin.Context, rule *model.ArchiveRule) error {
	return r.WithTenant(c).Create(rule).Error
}

// GetByID 通过 ID 查询单条归档规则。
func (r *ArchiveRuleRepo) GetByID(c *gin.Context, id uuid.UUID) (*model.ArchiveRule, error) {
	var rule model.ArchiveRule
	if err := r.WithTenant(c).Where("id = ?", id).First(&rule).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// UpdateFields 通过 map 更新指定字段。
func (r *ArchiveRuleRepo) UpdateFields(c *gin.Context, id uuid.UUID, fields map[string]interface{}) error {
	return r.WithTenant(c).Model(&model.ArchiveRule{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 硬删除归档规则。
func (r *ArchiveRuleRepo) Delete(c *gin.Context, id uuid.UUID) error {
	return r.WithTenant(c).Where("id = ?", id).Delete(&model.ArchiveRule{}).Error
}

// ListByConfigIDFilter 按配置 ID 查询归档规则列表，支持按 rule_scope 和 enabled 筛选。
func (r *ArchiveRuleRepo) ListByConfigIDFilter(c *gin.Context, configID uuid.UUID, ruleScope *string, enabled *bool) ([]model.ArchiveRule, error) {
	query := r.WithTenant(c).Where("config_id = ?", configID)
	if ruleScope != nil {
		query = query.Where("rule_scope = ?", *ruleScope)
	}
	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}
	var rules []model.ArchiveRule
	if err := query.Order("created_at ASC").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}
