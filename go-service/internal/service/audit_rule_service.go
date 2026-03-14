package service

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/repository"
)

// AuditRuleService 处理审核规则的业务逻辑。
type AuditRuleService struct {
	ruleRepo *repository.AuditRuleRepo
}

// NewAuditRuleService 创建一个新的 AuditRuleService 实例。
func NewAuditRuleService(ruleRepo *repository.AuditRuleRepo) *AuditRuleService {
	return &AuditRuleService{ruleRepo: ruleRepo}
}

// Create 创建审核规则。
func (s *AuditRuleService) Create(c *gin.Context, req *dto.CreateAuditRuleRequest) (*model.AuditRule, error) {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}

	// 默认开启
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	rule := &model.AuditRule{
		ID:          uuid.New(),
		TenantID:    tenantID,
		ProcessType: req.ProcessType,
		RuleContent: req.RuleContent,
		RuleScope:   defaultStr(req.RuleScope, "default_on"),
		Enabled:     &enabled,
		Source:      defaultStr(req.Source, "manual"),
		RelatedFlow: req.RelatedFlow,
	}

	// 设置 config_id
	if req.ConfigID != "" {
		configID, err := uuid.Parse(req.ConfigID)
		if err == nil {
			rule.ConfigID = &configID
		}
	}

	if err := s.ruleRepo.Create(c, rule); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return rule, nil
}

// Update 更新审核规则。
func (s *AuditRuleService) Update(c *gin.Context, id uuid.UUID, req *dto.UpdateAuditRuleRequest) (*model.AuditRule, error) {
	_, err := s.ruleRepo.GetByID(c, id)
	if err != nil {
		return nil, newServiceError(errcode.ErrRuleNotFound, "审核规则不存在")
	}

	fields := make(map[string]interface{})
	if req.RuleContent != "" {
		fields["rule_content"] = req.RuleContent
	}
	if req.RuleScope != "" {
		fields["rule_scope"] = req.RuleScope
	}
	if req.Enabled != nil {
		fields["enabled"] = *req.Enabled
	}
	if req.RelatedFlow != nil {
		fields["related_flow"] = *req.RelatedFlow
	}

	if len(fields) > 0 {
		if err := s.ruleRepo.UpdateFields(c, id, fields); err != nil {
			return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
		}
	}

	rule, err := s.ruleRepo.GetByID(c, id)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return rule, nil
}

// Delete 删除审核规则。
func (s *AuditRuleService) Delete(c *gin.Context, id uuid.UUID) error {
	_, err := s.ruleRepo.GetByID(c, id)
	if err != nil {
		return newServiceError(errcode.ErrRuleNotFound, "审核规则不存在")
	}

	// 统一执行硬删除，确保前端点击删除后内容真实消失
	if err := s.ruleRepo.Delete(c, id); err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return nil
}

// ListByConfigIDFilter 按配置 ID 查询关联的审核规则，支持筛选。
func (s *AuditRuleService) ListByConfigIDFilter(c *gin.Context, configID uuid.UUID, ruleScope *string, enabled *bool) ([]model.AuditRule, error) {
	rules, err := s.ruleRepo.ListByConfigIDFilter(c, configID, ruleScope, enabled)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return rules, nil
}
