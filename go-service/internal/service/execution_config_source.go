package service

import (
	"encoding/json"
	"errors"
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

	activated, err := s.versions.SetActiveBaseVersion(c.Request.Context(), tenantID, module, sourceConfigID, versionNo)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, newServiceError(errcode.ErrConfigNotFound, "指定版本不存在")
		}
		return nil, newServiceError(errcode.ErrDatabase, "激活指定版本失败")
	}

	if err := s.applySnapshotToSource(c, tenantID, module, sourceConfigID, activated.ConfigSnapshot); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "应用版本配置到生效库失败")
	}

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

	fingerprint := stableJSONFingerprint(snapshot)
	updated, err := s.versions.UpdateBaseVersionSnapshot(
		c.Request.Context(), tenantID, module, sourceConfigID, versionNo, fingerprint, snapshot,
	)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, newServiceError(errcode.ErrConfigNotFound, "指定版本不存在")
		}
		return nil, newServiceError(errcode.ErrDatabase, "保存版本快照失败")
	}

	// 若更新的版本恰好是当前启用版本，同步更新主表生效内容
	if updated.IsActive {
		_ = s.applySnapshotToSource(c, tenantID, module, sourceConfigID, updated.ConfigSnapshot)
	}

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

		existingRules, _ := s.auditRules.ListByConfigID(c, sourceConfigID)
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
				_ = s.auditRules.Update(c, ruleModel)
			} else {
				_ = s.auditRules.Create(c, ruleModel)
			}
		}
		for idStr, id := range existingMap {
			if !snapshotRuleIDs[idStr] {
				_ = s.auditRules.Delete(c, id)
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

		existingRules, _ := s.archiveRules.ListByConfigID(c, sourceConfigID)
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
				_ = s.archiveRules.Update(c, ruleModel)
			} else {
				_ = s.archiveRules.Create(c, ruleModel)
			}
		}
		for idStr, id := range existingMap {
			if !snapshotRuleIDs[idStr] {
				_ = s.archiveRules.Delete(c, id)
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
