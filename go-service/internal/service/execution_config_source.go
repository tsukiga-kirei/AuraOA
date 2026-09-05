package service

import (
	"auraoa/go-service/internal/cache"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/errcode"
	"auraoa/go-service/internal/repository"
)

// ExecutionConfigVersionStatus 描述租户当前配置对应的基础版本。
type ExecutionConfigVersionStatus struct {
	Status            string `json:"status"`
	ActiveVersionNo   *int   `json:"active_version_no,omitempty"`
	CurrentVersionNo  *int   `json:"current_version_no,omitempty"`
	LatestVersionNo   *int   `json:"latest_version_no,omitempty"`
	HasPendingChanges bool   `json:"has_pending_changes"`
}

// ExecutionConfigSourceService 为配置页提供可靠的租户基础版本状态。
type ExecutionConfigSourceService struct {
	invalidator    *cache.InvalidationManager
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

// GetStatus 比对当前已保存的租户配置指纹与当前激活版本指纹。
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

	active, err := s.versions.GetActiveBaseVersion(c.Request.Context(), tenantID, module, sourceConfigID)
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
				Status: "current", ActiveVersionNo: &versionNo, CurrentVersionNo: &versionNo, LatestVersionNo: &versionNo, HasPendingChanges: false,
			}, nil
		}
		return nil, newServiceError(errcode.ErrDatabase, "查询配置版本状态失败")
	}

	latest, _ := s.versions.GetLatestBaseVersion(c.Request.Context(), tenantID, module, sourceConfigID)
	var latestVersionNo int
	if latest != nil {
		latestVersionNo = latest.VersionNo
	} else {
		latestVersionNo = active.VersionNo
	}

	activeVersionNo := active.VersionNo
	if active.Fingerprint == sourceFingerprint {
		return &ExecutionConfigVersionStatus{
			Status: "current", ActiveVersionNo: &activeVersionNo, CurrentVersionNo: &activeVersionNo, LatestVersionNo: &latestVersionNo, HasPendingChanges: false,
		}, nil
	}

	return &ExecutionConfigVersionStatus{
		Status: "updated", ActiveVersionNo: &activeVersionNo, CurrentVersionNo: &activeVersionNo, LatestVersionNo: &latestVersionNo, HasPendingChanges: true,
	}, nil
}

// Publish 显式将当前已保存的租户配置固化并发布为新版本，同时设为当前可用版本。
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

	s.invalidate(c, tenantID, module)
	versionNo := published.VersionNo
	return &ExecutionConfigVersionStatus{
		Status: "current", ActiveVersionNo: &versionNo, CurrentVersionNo: &versionNo, LatestVersionNo: &versionNo, HasPendingChanges: false,
	}, nil
}

// TenantConfigVersionHistoryItem 描述历史版本的一条记录。
type TenantConfigVersionHistoryItem struct {
	ID             uuid.UUID      `json:"id"`
	VersionNo      int            `json:"version_no"`
	Fingerprint    string         `json:"fingerprint"`
	ConfigSnapshot datatypes.JSON `json:"config_snapshot"`
	IsActive       bool           `json:"is_active"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// ListHistory 查询指定配置发布的所有历史基础版本。
func (s *ExecutionConfigSourceService) ListHistory(
	c *gin.Context,
	module string,
	sourceConfigID uuid.UUID,
) ([]TenantConfigVersionHistoryItem, error) {
	if module != model.ExecutionConfigModuleAudit &&
		module != model.ExecutionConfigModuleArchive &&
		module != model.ExecutionConfigModuleSummary {
		return nil, newServiceError(errcode.ErrParamValidation, "配置模块无效")
	}
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}

	versions, err := s.versions.ListBaseVersions(c.Request.Context(), tenantID, module, sourceConfigID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "查询版本历史失败")
	}
	result := make([]TenantConfigVersionHistoryItem, 0, len(versions))
	for _, v := range versions {
		result = append(result, TenantConfigVersionHistoryItem{
			ID:             v.ID,
			VersionNo:      v.VersionNo,
			Fingerprint:    v.Fingerprint,
			ConfigSnapshot: v.ConfigSnapshot,
			IsActive:       v.IsActive,
			CreatedAt:      v.CreatedAt,
			UpdatedAt:      v.UpdatedAt,
		})
	}
	return result, nil
}

// ActivateVersion 将指定版本设为当前可用版本（Active Version），并将配置应用到当前生效库。
func (s *ExecutionConfigSourceService) ActivateVersion(
	c *gin.Context,
	module string,
	sourceConfigID uuid.UUID,
	versionNo int,
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

	err = s.inTransaction(c, func(scoped *ExecutionConfigSourceService) error {
		activated, err := scoped.versions.SetActiveBaseVersion(c.Request.Context(), tenantID, module, sourceConfigID, versionNo)
		if err != nil {
			return err
		}
		if err := validateSourceSnapshot(module, activated.ConfigSnapshot); err != nil {
			return err
		}
		return scoped.applySnapshotToSource(c, tenantID, module, sourceConfigID, activated.ConfigSnapshot)
	})
	if err != nil {
		return nil, fmt.Errorf("激活配置版本失败: %w", err)
	}
	s.invalidate(c, tenantID, module)

	return s.GetStatus(c, module, sourceConfigID)
}

// SaveVersion 直接修改并保存指定历史版本的快照内容。
func (s *ExecutionConfigSourceService) SaveVersion(
	c *gin.Context,
	module string,
	sourceConfigID uuid.UUID,
	versionNo int,
	snapshot interface{},
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

	raw, err := json.Marshal(snapshot)
	if err != nil {
		return nil, err
	}
	if err := validateSourceSnapshot(module, raw); err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, err.Error())
	}
	fingerprint := stableJSONFingerprint(snapshot)
	err = s.inTransaction(c, func(scoped *ExecutionConfigSourceService) error {
		updated, err := scoped.versions.UpdateBaseVersionSnapshot(c.Request.Context(), tenantID, module, sourceConfigID, versionNo, fingerprint, snapshot)
		if err != nil {
			return err
		}
		if updated.IsActive {
			return scoped.applySnapshotToSource(c, tenantID, module, sourceConfigID, updated.ConfigSnapshot)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("保存版本快照失败: %w", err)
	}
	s.invalidate(c, tenantID, module)

	return s.GetStatus(c, module, sourceConfigID)
}

func (s *ExecutionConfigSourceService) applySnapshotToSource(
	c *gin.Context,
	tenantID uuid.UUID,
	module string,
	sourceConfigID uuid.UUID,
	rawSnapshot datatypes.JSON,
) error {
	switch module {
	case model.ExecutionConfigModuleAudit:
		var snapshot struct {
			ProcessType     string         `json:"process_type"`
			MainTableName   string         `json:"main_table_name"`
			MainFields      datatypes.JSON `json:"main_fields"`
			DetailTables    datatypes.JSON `json:"detail_tables"`
			FieldMode       string         `json:"field_mode"`
			KBMode          string         `json:"kb_mode"`
			AIConfig        datatypes.JSON `json:"ai_config"`
			UserPermissions datatypes.JSON `json:"user_permissions"`
			Status          string         `json:"status"`
			Rules           []ruleSource   `json:"rules"`
		}
		if err := json.Unmarshal(rawSnapshot, &snapshot); err != nil {
			return err
		}
		fields := map[string]interface{}{
			"main_table_name":  snapshot.MainTableName,
			"main_fields":      snapshot.MainFields,
			"detail_tables":    snapshot.DetailTables,
			"field_mode":       snapshot.FieldMode,
			"kb_mode":          snapshot.KBMode,
			"ai_config":        snapshot.AIConfig,
			"user_permissions": snapshot.UserPermissions,
			"status":           snapshot.Status,
		}
		if err := s.auditConfigs.UpdateFields(c, sourceConfigID, fields); err != nil {
			return err
		}

		existingRules, err := s.auditRules.ListByConfigID(c, sourceConfigID)
		if err != nil {
			return err
		}
		existingMap := make(map[string]uuid.UUID)
		for _, r := range existingRules {
			existingMap[r.ID.String()] = r.ID
		}
		snapshotRuleIDs := make(map[string]bool)
		for _, rs := range snapshot.Rules {
			ruleID, parseErr := uuid.Parse(rs.ID)
			if parseErr != nil {
				ruleID = uuid.New()
			}
			snapshotRuleIDs[ruleID.String()] = true
			enabled := rs.Enabled
			ruleModel := &model.AuditRule{
				ID:             ruleID,
				TenantID:       tenantID,
				ConfigID:       &sourceConfigID,
				ProcessType:    snapshot.ProcessType,
				RuleContent:    rs.RuleContent,
				RuleScope:      rs.RuleScope,
				Enabled:        &enabled,
				RelatedFlow:    rs.RelatedFlow,
				ContextEnabled: rs.ContextEnabled,
			}
			if rs.ContextMounts != nil {
				if mountJSON, err := json.Marshal(rs.ContextMounts); err == nil {
					ruleModel.ContextMounts = datatypes.JSON(mountJSON)
				}
			}
			if _, exists := existingMap[ruleID.String()]; exists {
				if err := s.auditRules.WithTenant(c).Model(ruleModel).Where("id = ?", ruleModel.ID).Select("rule_content", "rule_scope", "enabled", "related_flow", "context_enabled", "context_mounts").Updates(ruleModel).Error; err != nil {
					return err
				}
			} else {
				if err := s.auditRules.Create(c, ruleModel); err != nil {
					return err
				}
			}
		}
		for idStr, id := range existingMap {
			if !snapshotRuleIDs[idStr] {
				if err := s.auditRules.Delete(c, id); err != nil {
					return err
				}
			}
		}

	case model.ExecutionConfigModuleArchive:
		var snapshot struct {
			ProcessType     string         `json:"process_type"`
			MainTableName   string         `json:"main_table_name"`
			MainFields      datatypes.JSON `json:"main_fields"`
			DetailTables    datatypes.JSON `json:"detail_tables"`
			FieldMode       string         `json:"field_mode"`
			KBMode          string         `json:"kb_mode"`
			AIConfig        datatypes.JSON `json:"ai_config"`
			UserPermissions datatypes.JSON `json:"user_permissions"`
			Status          string         `json:"status"`
			Rules           []ruleSource   `json:"rules"`
		}
		if err := json.Unmarshal(rawSnapshot, &snapshot); err != nil {
			return err
		}
		fields := map[string]interface{}{
			"main_table_name":  snapshot.MainTableName,
			"main_fields":      snapshot.MainFields,
			"detail_tables":    snapshot.DetailTables,
			"field_mode":       snapshot.FieldMode,
			"kb_mode":          snapshot.KBMode,
			"ai_config":        snapshot.AIConfig,
			"user_permissions": snapshot.UserPermissions,
			"status":           snapshot.Status,
		}
		if err := s.archiveConfigs.UpdateFields(c, sourceConfigID, fields); err != nil {
			return err
		}

		existingRules, err := s.archiveRules.ListByConfigID(c, sourceConfigID)
		if err != nil {
			return err
		}
		existingMap := make(map[string]uuid.UUID)
		for _, r := range existingRules {
			existingMap[r.ID.String()] = r.ID
		}
		snapshotRuleIDs := make(map[string]bool)
		for _, rs := range snapshot.Rules {
			ruleID, parseErr := uuid.Parse(rs.ID)
			if parseErr != nil {
				ruleID = uuid.New()
			}
			snapshotRuleIDs[ruleID.String()] = true
			enabled := rs.Enabled
			ruleModel := &model.ArchiveRule{
				ID:             ruleID,
				TenantID:       tenantID,
				ConfigID:       &sourceConfigID,
				ProcessType:    snapshot.ProcessType,
				RuleContent:    rs.RuleContent,
				RuleScope:      rs.RuleScope,
				Enabled:        &enabled,
				RelatedFlow:    rs.RelatedFlow,
				ContextEnabled: rs.ContextEnabled,
			}
			if rs.ContextMounts != nil {
				if mountJSON, err := json.Marshal(rs.ContextMounts); err == nil {
					ruleModel.ContextMounts = datatypes.JSON(mountJSON)
				}
			}
			if _, exists := existingMap[ruleID.String()]; exists {
				if err := s.archiveRules.WithTenant(c).Model(ruleModel).Where("id = ?", ruleModel.ID).Select("rule_content", "rule_scope", "enabled", "related_flow", "context_enabled", "context_mounts").Updates(ruleModel).Error; err != nil {
					return err
				}
			} else {
				if err := s.archiveRules.Create(c, ruleModel); err != nil {
					return err
				}
			}
		}
		for idStr, id := range existingMap {
			if !snapshotRuleIDs[idStr] {
				if err := s.archiveRules.Delete(c, id); err != nil {
					return err
				}
			}
		}

	case model.ExecutionConfigModuleSummary:
		var snapshot struct {
			MainTableName string         `json:"main_table_name"`
			MainFields    datatypes.JSON `json:"main_fields"`
			DetailTables  datatypes.JSON `json:"detail_tables"`
			SummaryBlocks datatypes.JSON `json:"summary_blocks"`
			Status        string         `json:"status"`
		}
		if err := json.Unmarshal(rawSnapshot, &snapshot); err != nil {
			return err
		}
		fields := map[string]interface{}{
			"main_table_name": snapshot.MainTableName,
			"main_fields":     snapshot.MainFields,
			"detail_tables":   snapshot.DetailTables,
			"summary_blocks":  snapshot.SummaryBlocks,
			"status":          snapshot.Status,
		}
		if err := s.summaryConfigs.UpdateFields(c, sourceConfigID, fields); err != nil {
			return err
		}
	}
	return nil
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

// SetInvalidator 接入共享配置缓存失效器。
func (s *ExecutionConfigSourceService) SetInvalidator(invalidator *cache.InvalidationManager) {
	s.invalidator = invalidator
}
func (s *ExecutionConfigSourceService) invalidate(c *gin.Context, tenantID uuid.UUID, module string) {
	if s.invalidator != nil {
		_ = s.invalidator.InvalidateConfigCache(context.WithoutCancel(c.Request.Context()), tenantID, module)
	}
}
func (s *ExecutionConfigSourceService) inTransaction(c *gin.Context, operation func(*ExecutionConfigSourceService) error) error {
	return s.auditConfigs.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		scoped := NewExecutionConfigSourceService(repository.NewExecutionConfigVersionRepo(tx), repository.NewProcessAuditConfigRepo(tx), repository.NewAuditRuleRepo(tx), repository.NewProcessArchiveConfigRepo(tx), repository.NewArchiveRuleRepo(tx), repository.NewProcessSummaryConfigRepo(tx))
		return operation(scoped)
	})
}

// validateSourceSnapshot 拒绝缺字段和 JSON null，避免将损坏的历史快照激活到主表。
func validateSourceSnapshot(module string, raw []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return fmt.Errorf("配置快照格式无效")
	}
	required := []string{"process_type", "main_table_name", "main_fields", "detail_tables", "status"}
	if module == model.ExecutionConfigModuleSummary {
		required = append(required, "summary_blocks")
	} else {
		required = append(required, "field_mode", "kb_mode", "ai_config", "user_permissions", "rules")
	}
	for _, key := range required {
		if len(fields[key]) == 0 || string(fields[key]) == "null" {
			return fmt.Errorf("配置快照缺少有效字段 %s", key)
		}
	}
	for _, key := range []string{"main_fields", "detail_tables", "rules", "summary_blocks"} {
		if raw, ok := fields[key]; ok {
			var values []json.RawMessage
			if err := json.Unmarshal(raw, &values); err != nil {
				return fmt.Errorf("配置字段 %s 必须为数组", key)
			}
		}
	}
	for _, key := range []string{"ai_config", "user_permissions"} {
		if raw, ok := fields[key]; ok {
			var values map[string]json.RawMessage
			if err := json.Unmarshal(raw, &values); err != nil {
				return fmt.Errorf("配置字段 %s 必须为对象", key)
			}
		}
	}
	return nil
}
