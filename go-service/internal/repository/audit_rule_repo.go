package repository

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// AuditRuleRepo 提供审核规则的数据访问方法，按租户隔离。
type AuditRuleRepo struct {
	*BaseRepo
}

// NewAuditRuleRepo 创建一个新的 AuditRuleRepo 实例。
func NewAuditRuleRepo(db *gorm.DB) *AuditRuleRepo {
	return &AuditRuleRepo{BaseRepo: NewBaseRepo(db)}
}

// Create 创建审核规则记录。
func (r *AuditRuleRepo) Create(c *gin.Context, rule *model.AuditRule) error {
	return r.WithTenant(c).Create(rule).Error
}

// Update 更新审核规则。
func (r *AuditRuleRepo) Update(c *gin.Context, rule *model.AuditRule) error {
	return r.WithTenant(c).Model(rule).Where("id = ?", rule.ID).Updates(rule).Error
}

// UpdateFields 通过 map 更新指定字段。
func (r *AuditRuleRepo) UpdateFields(c *gin.Context, id uuid.UUID, fields map[string]interface{}) error {
	return r.WithTenant(c).Model(&model.AuditRule{}).Where("id = ?", id).Updates(fields).Error
}

// Delete 硬删除审核规则。
func (r *AuditRuleRepo) Delete(c *gin.Context, id uuid.UUID) error {
	return r.WithTenant(c).Where("id = ?", id).Delete(&model.AuditRule{}).Error
}

// GetByID 通过 ID 查询单条审核规则。
func (r *AuditRuleRepo) GetByID(c *gin.Context, id uuid.UUID) (*model.AuditRule, error) {
	var rule model.AuditRule
	if err := r.WithTenant(c).Where("id = ?", id).First(&rule).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

// ListByConfigIDFilter 按配置 ID 查询审核规则列表，支持按 rule_scope 和 enabled 筛选。
func (r *AuditRuleRepo) ListByConfigIDFilter(c *gin.Context, configID uuid.UUID, ruleScope *string, enabled *bool) ([]model.AuditRule, error) {
	query := r.WithTenant(c).Where("config_id = ?", configID)
	if ruleScope != nil {
		query = query.Where("rule_scope = ?", *ruleScope)
	}
	if enabled != nil {
		query = query.Where("enabled = ?", *enabled)
	}
	var rules []model.AuditRule
	if err := query.Order("created_at ASC").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

// ListByConfigID 按配置 ID 查询关联的审核规则。
func (r *AuditRuleRepo) ListByConfigID(c *gin.Context, configID uuid.UUID) ([]model.AuditRule, error) {
	var rules []model.AuditRule
	if err := r.WithTenant(c).Where("config_id = ?", configID).Order("created_at ASC").Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}
