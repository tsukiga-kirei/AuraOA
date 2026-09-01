package service

import (
	"sort"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/errcode"
	"auraoa/go-service/internal/repository"
)

// ExecutionConfigVersionStatus 描述租户当前配置对应的基础版本。
type ExecutionConfigVersionStatus struct {
	Status            string `json:"status"`
	CurrentVersionNo  *int   `json:"current_version_no,omitempty"`
	LatestVersionNo   *int   `json:"latest_version_no,omitempty"`
	HasPendingChanges bool   `json:"has_pending_changes"`
}

// ExecutionConfigSourceService 为配置页提供可靠的租户基础版本状态。
type ExecutionConfigSourceService struct {
	versions       *repository.ExecutionConfigVersionRepo
	auditConfigs   *repository.ProcessAuditConfigRepo
	auditRules     *repository.AuditRuleRepo
	archiveConfigs *repository.ProcessArchiveConfigRepo
	archiveRules   *repository.ArchiveRuleRepo
	summaryConfigs *repository.ProcessSummaryConfigRepo
}

func NewExecutionConfigSourceService(
	versions *repository.ExecutionConfigVersionRepo,
	auditConfigs *repository.ProcessAuditConfigRepo,
	auditRules *repository.AuditRuleRepo,
	archiveConfigs *repository.ProcessArchiveConfigRepo,
	archiveRules *repository.ArchiveRuleRepo,
	summaryConfigs *repository.ProcessSummaryConfigRepo,
) *ExecutionConfigSourceService {
	return &ExecutionConfigSourceService{
		versions:       versions,
		auditConfigs:   auditConfigs,
		auditRules:     auditRules,
		archiveConfigs: archiveConfigs,
		archiveRules:   archiveRules,
		summaryConfigs: summaryConfigs,
	}
}

// GetStatus 查询租户当前配置与最新发布基础版本的对应状态。
// 本方法为纯只读查询，不会自动生成新版本。
func (s *ExecutionConfigSourceService) GetStatus(
	c *gin.Context,
	module string,
	sourceConfigID uuid.UUID,
) (*ExecutionConfigVersionStatus, error) {
	if module != model.ExecutionConfigModuleAudit &&
		module != model.ExecutionConfigModuleArchive &&
		module != model.ExecutionConfigModuleSummary {
		return nil, newServiceError(errcode.ErrParamValidation, "配置模块无效")
	}
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}

	snapshot, sourceFingerprint, err := s.currentSourceSnapshot(c, module, sourceConfigID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, newServiceError(errcode.ErrConfigNotFound, "流程配置不存在")
		}
		return nil, newServiceError(errcode.ErrDatabase, "读取配置版本状态失败")
	}

	latest, err := s.versions.GetLatestBaseVersion(c.Request.Context(), tenantID, module, sourceConfigID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 若尚未创建任何版本，自动初始化 V1
			userID, userErr := getUserUUID(c)
			if userErr != nil {
				return nil, newServiceError(errcode.ErrParamValidation, "用户ID无效")
			}
			initial, initErr := s.versions.GetOrCreateLatestBaseVersion(
				c.Request.Context(), tenantID, userID, module, sourceConfigID, sourceFingerprint, snapshot,
			)
			if initErr != nil {
				return nil, newServiceError(errcode.ErrDatabase, "初始化配置版本失败")
			}
			versionNo := initial.VersionNo
			return &ExecutionConfigVersionStatus{
				Status: "current", CurrentVersionNo: &versionNo, LatestVersionNo: &versionNo, HasPendingChanges: false,
			}, nil
		}
		return nil, newServiceError(errcode.ErrDatabase, "查询配置版本状态失败")
	}

	versionNo := latest.VersionNo
	if latest.Fingerprint == sourceFingerprint {
		return &ExecutionConfigVersionStatus{
			Status: "current", CurrentVersionNo: &versionNo, LatestVersionNo: &versionNo, HasPendingChanges: false,
		}, nil
	}

	return &ExecutionConfigVersionStatus{
		Status: "updated", CurrentVersionNo: &versionNo, LatestVersionNo: &versionNo, HasPendingChanges: true,
	}, nil
}

// Publish 显式将当前已保存的租户配置固化并发布为新版本。
func (s *ExecutionConfigSourceService) Publish(
	c *gin.Context,
	module string,
	sourceConfigID uuid.UUID,
) (*ExecutionConfigVersionStatus, error) {
	if module != model.ExecutionConfigModuleAudit &&
		module != model.ExecutionConfigModuleArchive &&
		module != model.ExecutionConfigModuleSummary {
		return nil, newServiceError(errcode.ErrParamValidation, "配置模块无效")
	}
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}

	snapshot, sourceFingerprint, err := s.currentSourceSnapshot(c, module, sourceConfigID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, newServiceError(errcode.ErrConfigNotFound, "流程配置不存在")
		}
		return nil, newServiceError(errcode.ErrDatabase, "读取配置版本状态失败")
	}

	userID, err := getUserUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "用户ID无效")
	}

	published, err := s.versions.PublishBaseVersion(
		c.Request.Context(), tenantID, userID, module, sourceConfigID, sourceFingerprint, snapshot,
	)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "发布配置版本失败")
	}

	versionNo := published.VersionNo
	return &ExecutionConfigVersionStatus{
		Status: "current", CurrentVersionNo: &versionNo, LatestVersionNo: &versionNo, HasPendingChanges: false,
	}, nil
}

func (s *ExecutionConfigSourceService) currentSourceSnapshot(
	c *gin.Context,
	module string,
	sourceConfigID uuid.UUID,
) (interface{}, string, error) {
	switch module {
	case model.ExecutionConfigModuleAudit:
		config, err := s.auditConfigs.GetByID(c, sourceConfigID)
		if err != nil {
			return nil, "", err
		}
		rules, err := s.auditRules.ListByConfigID(c, sourceConfigID)
		if err != nil {
			return nil, "", err
		}
		snapshot := auditConfigSourceSnapshot(config, rules)
		return snapshot, stableJSONFingerprint(snapshot), nil
	case model.ExecutionConfigModuleArchive:
		config, err := s.archiveConfigs.GetByID(c, sourceConfigID)
		if err != nil {
			return nil, "", err
		}
		rules, err := s.archiveRules.ListByConfigID(c, sourceConfigID)
		if err != nil {
			return nil, "", err
		}
		snapshot := archiveConfigSourceSnapshot(config, rules)
		return snapshot, stableJSONFingerprint(snapshot), nil
	case model.ExecutionConfigModuleSummary:
		config, err := s.summaryConfigs.GetByID(c, sourceConfigID)
		if err != nil {
			return nil, "", err
		}
		snapshot := summaryConfigSourceSnapshot(config)
		return snapshot, stableJSONFingerprint(snapshot), nil
	default:
		return nil, "", newServiceError(errcode.ErrParamValidation, "配置模块无效")
	}
}

type ruleSource struct {
	ID             string      `json:"id"`
	RuleContent    string      `json:"rule_content"`
	RuleScope      string      `json:"rule_scope"`
	Enabled        bool        `json:"enabled"`
	RelatedFlow    bool        `json:"related_flow"`
	ContextEnabled bool        `json:"context_enabled"`
	ContextMounts  interface{} `json:"context_mounts"`
}

func normalizedAuditRuleSources(rules []model.AuditRule) []ruleSource {
	result := make([]ruleSource, 0, len(rules))
	for _, rule := range rules {
		result = append(result, ruleSource{
			ID: rule.ID.String(), RuleContent: rule.RuleContent, RuleScope: rule.RuleScope,
			Enabled: rule.Enabled == nil || *rule.Enabled, RelatedFlow: rule.RelatedFlow,
			ContextEnabled: rule.ContextEnabled, ContextMounts: rule.ContextMounts,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func normalizedArchiveRuleSources(rules []model.ArchiveRule) []ruleSource {
	result := make([]ruleSource, 0, len(rules))
	for _, rule := range rules {
		result = append(result, ruleSource{
			ID: rule.ID.String(), RuleContent: rule.RuleContent, RuleScope: rule.RuleScope,
			Enabled: rule.Enabled == nil || *rule.Enabled, RelatedFlow: rule.RelatedFlow,
			ContextEnabled: rule.ContextEnabled, ContextMounts: rule.ContextMounts,
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func auditConfigSourceSnapshot(config *model.ProcessAuditConfig, rules []model.AuditRule) map[string]interface{} {
	return map[string]interface{}{
		"process_type": config.ProcessType, "main_table_name": config.MainTableName,
		"main_fields": config.MainFields, "detail_tables": config.DetailTables,
		"field_mode": config.FieldMode, "kb_mode": config.KBMode,
		"ai_config": config.AIConfig, "user_permissions": config.UserPermissions, "status": config.Status,
		"rules": normalizedAuditRuleSources(rules),
	}
}

func auditConfigSourceFingerprint(config *model.ProcessAuditConfig, rules []model.AuditRule) string {
	return stableJSONFingerprint(auditConfigSourceSnapshot(config, rules))
}

func archiveConfigSourceSnapshot(config *model.ProcessArchiveConfig, rules []model.ArchiveRule) map[string]interface{} {
	return map[string]interface{}{
		"process_type": config.ProcessType, "main_table_name": config.MainTableName,
		"main_fields": config.MainFields, "detail_tables": config.DetailTables,
		"field_mode": config.FieldMode, "kb_mode": config.KBMode,
		"ai_config": config.AIConfig, "user_permissions": config.UserPermissions, "status": config.Status,
		"rules": normalizedArchiveRuleSources(rules),
	}
}

func archiveConfigSourceFingerprint(config *model.ProcessArchiveConfig, rules []model.ArchiveRule) string {
	return stableJSONFingerprint(archiveConfigSourceSnapshot(config, rules))
}

func summaryConfigSourceSnapshot(config *model.ProcessSummaryConfig) map[string]interface{} {
	return map[string]interface{}{
		"process_type": config.ProcessType, "main_table_name": config.MainTableName,
		"main_fields": config.MainFields, "detail_tables": config.DetailTables,
		"summary_blocks": config.SummaryBlocks, "status": config.Status,
	}
}

func summaryConfigSourceFingerprint(config *model.ProcessSummaryConfig) string {
	return stableJSONFingerprint(summaryConfigSourceSnapshot(config))
}
