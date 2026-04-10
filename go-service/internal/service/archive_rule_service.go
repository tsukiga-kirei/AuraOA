package service

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/repository"
)

// ArchiveRuleService 处理归档规则的业务逻辑。
type ArchiveRuleService struct {
	ruleRepo *repository.ArchiveRuleRepo
}

// NewArchiveRuleService 创建一个新的 ArchiveRuleService 实例。
func NewArchiveRuleService(ruleRepo *repository.ArchiveRuleRepo) *ArchiveRuleService {
	return &ArchiveRuleService{ruleRepo: ruleRepo}
}

// Create 创建归档规则。
func (s *ArchiveRuleService) Create(c *gin.Context, req *dto.CreateArchiveRuleRequest) (*model.ArchiveRule, error) {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}

	// 默认开启
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	rule := &model.ArchiveRule{
		ID:          uuid.New(),
		TenantID:    tenantID,
		ProcessType: req.ProcessType,
		RuleContent: req.RuleContent,
		RuleScope:   defaultStr(req.RuleScope, "default_on"),
		Enabled:     &enabled,
		Source:      defaultStr(req.Source, "manual"),
		RelatedFlow: req.RelatedFlow,
	}

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

// Update 更新归档规则。
func (s *ArchiveRuleService) Update(c *gin.Context, id uuid.UUID, req *dto.UpdateArchiveRuleRequest) (*model.ArchiveRule, error) {
	_, err := s.ruleRepo.GetByID(c, id)
	if err != nil {
		return nil, newServiceError(errcode.ErrRuleNotFound, "归档规则不存在")
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

// Delete 删除归档规则：manual 来源硬删除，file_import 来源标记禁用。
func (s *ArchiveRuleService) Delete(c *gin.Context, id uuid.UUID) error {
	rule, err := s.ruleRepo.GetByID(c, id)
	if err != nil {
		return newServiceError(errcode.ErrRuleNotFound, "归档规则不存在")
	}

	if rule.Source == "manual" {
		if err := s.ruleRepo.Delete(c, id); err != nil {
			return newServiceError(errcode.ErrDatabase, "数据库错误")
		}
	} else {
		if err := s.ruleRepo.UpdateFields(c, id, map[string]interface{}{"enabled": false}); err != nil {
			return newServiceError(errcode.ErrDatabase, "数据库错误")
		}
	}
	return nil
}

// ListByConfigIDFilter 按配置 ID 查询关联的归档规则。
func (s *ArchiveRuleService) ListByConfigIDFilter(c *gin.Context, configID uuid.UUID, ruleScope *string, enabled *bool) ([]model.ArchiveRule, error) {
	rules, err := s.ruleRepo.ListByConfigIDFilter(c, configID, ruleScope, enabled)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return rules, nil
}
