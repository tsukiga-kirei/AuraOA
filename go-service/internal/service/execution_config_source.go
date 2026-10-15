package service

import (
	"auraoa/go-service/internal/cache"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
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
	dataQueries    *AuditDataQueryService
	invalidator    *cache.InvalidationManager
	scheduleMgr    EmbedRefreshScheduleManager
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

	_, sourceFingerprint, err := s.currentSourceSnapshot(c, module, sourceConfigID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, newServiceError(errcode.ErrConfigNotFound, "流程配置不存在")
		}
		return nil, newServiceError(errcode.ErrDatabase, "读取配置版本状态失败")
	}

	active, err := s.versions.GetActiveBaseVersion(c.Request.Context(), tenantID, module, sourceConfigID)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return &ExecutionConfigVersionStatus{
				Status: "unversioned", HasPendingChanges: true,
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
	if s.matchesActiveSnapshot(module, active.ConfigSnapshot, sourceFingerprint) {
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

	userID, err := getUserUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "用户ID无效")
	}
	err = s.inTransaction(c, func(scoped *ExecutionConfigSourceService) error {
		if err := scoped.versions.LockBaseVersion(c.Request.Context(), tenantID, module, sourceConfigID); err != nil {
			return err
		}
		snapshot, fingerprint, err := scoped.currentSourceSnapshot(c, module, sourceConfigID)
		if err != nil {
			return err
		}
		_, err = scoped.versions.PublishBaseVersion(c.Request.Context(), tenantID, userID, module, sourceConfigID, fingerprint, snapshot,
			func(raw datatypes.JSON) bool { return scoped.matchesActiveSnapshot(module, raw, fingerprint) })
		return err
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, newServiceError(errcode.ErrConfigNotFound, "流程配置不存在")
		}
		return nil, newServiceError(errcode.ErrDatabase, "发布配置版本失败")
	}

	return s.sourceChangeStatus(c, tenantID, module, sourceConfigID)
}

// SaveDraft 保存管理员当前编辑内容，但不创建发布版本。
// 首次发布前保持 unversioned 草稿状态；已有版本时仅产生待发布修改。
func (s *ExecutionConfigSourceService) SaveDraft(
	c *gin.Context,
	module string,
	sourceConfigID uuid.UUID,
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
		return nil, newServiceError(errcode.ErrParamValidation, "配置草稿格式无效")
	}
	if err := validateSourceSnapshot(module, raw); err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, err.Error())
	}
	if err := s.inTransaction(c, func(scoped *ExecutionConfigSourceService) error {
		if err := scoped.versions.LockBaseVersion(c.Request.Context(), tenantID, module, sourceConfigID); err != nil {
			return err
		}
		return scoped.applySnapshotToSource(c, tenantID, module, sourceConfigID, datatypes.JSON(raw))
	}); err != nil {
		return nil, fmt.Errorf("保存配置草稿失败: %w", err)
	}
	return s.sourceChangeStatus(c, tenantID, module, sourceConfigID)
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
	return s.sourceChangeStatus(c, tenantID, module, sourceConfigID)
}

// SaveVersion 兼容旧客户端：确认历史版本存在后另存草稿，不修改已发布版本。
func (s *ExecutionConfigSourceService) SaveVersion(c *gin.Context, module string, sourceConfigID uuid.UUID, versionNo int, snapshot interface{}) (*ExecutionConfigVersionStatus, error) {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}
	if _, err := s.versions.GetBaseVersionByNo(c.Request.Context(), tenantID, module, sourceConfigID, versionNo); err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, newServiceError(errcode.ErrConfigNotFound, "配置版本不存在")
		}
		return nil, newServiceError(errcode.ErrDatabase, "读取配置版本失败")
	}
	return s.SaveDraft(c, module, sourceConfigID, snapshot)
}

// DiscardDraft 丢弃未发布配置，恢复当前可用发布版本，不改变历史版本和流程绑定。
func (s *ExecutionConfigSourceService) DiscardDraft(c *gin.Context, module string, sourceConfigID uuid.UUID) (*ExecutionConfigVersionStatus, error) {
	if module != model.ExecutionConfigModuleAudit && module != model.ExecutionConfigModuleArchive && module != model.ExecutionConfigModuleSummary {
		return nil, newServiceError(errcode.ErrParamValidation, "配置模块无效")
	}
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}
	err = s.inTransaction(c, func(scoped *ExecutionConfigSourceService) error {
		if err := scoped.versions.LockBaseVersion(c.Request.Context(), tenantID, module, sourceConfigID); err != nil {
			return err
		}
		// 同时校验当前租户的源配置仍存在。
		if _, _, err := scoped.currentSourceSnapshot(c, module, sourceConfigID); err != nil {
			return err
		}
		active, err := scoped.versions.GetActiveBaseVersion(c.Request.Context(), tenantID, module, sourceConfigID)
		if err != nil {
			return err
		}
		if err := validateSourceSnapshot(module, active.ConfigSnapshot); err != nil {
			return err
		}
		return scoped.applySnapshotToSource(c, tenantID, module, sourceConfigID, active.ConfigSnapshot)
	})
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, newServiceError(errcode.ErrConfigNotFound, "配置不存在或尚无可恢复的发布版本")
		}
		return nil, fmt.Errorf("丢弃配置草稿失败: %w", err)
	}
	return s.sourceChangeStatus(c, tenantID, module, sourceConfigID)
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
		if err := decodeContentJSON(rawSnapshot, &snapshot); err != nil {
			return err
		}
		fields := map[string]interface{}{
			"process_type":     snapshot.ProcessType,
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
		existingSources := make(map[string]string)
		existingQueries := make(map[string]datatypes.JSON)
		for _, r := range existingRules {
			existingMap[r.ID.String()] = r.ID
			existingSources[r.ID.String()] = r.Source
			existingQueries[r.ID.String()] = r.DataQueries
		}
		snapshotRuleIDs := make(map[string]bool)
		for _, rs := range snapshot.Rules {
			queryJSON := datatypes.JSON([]byte("[]"))
			if rs.DataQueries != nil {
				raw, marshalErr := json.Marshal(rs.DataQueries)
				if marshalErr != nil {
					return marshalErr
				}
				queryJSON = datatypes.JSON(raw)
			}
			if s.dataQueries != nil {
				if err := s.dataQueries.ValidateReferenceChange(c, existingQueries[rs.ID], queryJSON); err != nil {
					return err
				}
			}
			ruleID, parseErr := uuid.Parse(rs.ID)
			if parseErr != nil {
				ruleID = uuid.New()
			}
			snapshotRuleIDs[ruleID.String()] = true
			enabled := rs.Enabled
			source := defaultStr(rs.Source, defaultStr(existingSources[ruleID.String()], "manual"))
			ruleModel := &model.AuditRule{
				ID:                     ruleID,
				TenantID:               tenantID,
				ConfigID:               &sourceConfigID,
				ProcessType:            snapshot.ProcessType,
				RuleContent:            rs.RuleContent,
				ApplicabilityCondition: strings.TrimSpace(rs.ApplicabilityCondition),
				RuleScope:              rs.RuleScope,
				Enabled:                &enabled,
				Source:                 source,
				RelatedFlow:            rs.RelatedFlow,
				ContextEnabled:         rs.ContextEnabled,
				DataQueries:            queryJSON,
				ContextMounts:          datatypes.JSON(`[]`),
			}
			if rs.ContextMounts != nil {
				if mountJSON, err := json.Marshal(rs.ContextMounts); err == nil {
					ruleModel.ContextMounts = datatypes.JSON(mountJSON)
				}
			}
			if _, exists := existingMap[ruleID.String()]; exists {
				if err := s.auditRules.WithTenant(c).Model(ruleModel).Where("id = ?", ruleModel.ID).Select("rule_content", "applicability_condition", "rule_scope", "enabled", "source", "related_flow", "context_enabled", "context_mounts", "data_queries").Updates(ruleModel).Error; err != nil {
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
		if err := decodeContentJSON(rawSnapshot, &snapshot); err != nil {
			return err
		}
		fields := map[string]interface{}{
			"process_type":     snapshot.ProcessType,
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
		existingSources := make(map[string]string)
		existingQueries := make(map[string]datatypes.JSON)
		for _, r := range existingRules {
			existingMap[r.ID.String()] = r.ID
			existingSources[r.ID.String()] = r.Source
			existingQueries[r.ID.String()] = r.DataQueries
		}
		snapshotRuleIDs := make(map[string]bool)
		for _, rs := range snapshot.Rules {
			queryJSON := datatypes.JSON([]byte("[]"))
			if rs.DataQueries != nil {
				raw, marshalErr := json.Marshal(rs.DataQueries)
				if marshalErr != nil {
					return marshalErr
				}
				queryJSON = datatypes.JSON(raw)
			}
			if s.dataQueries != nil {
				if err := s.dataQueries.ValidateReferenceChange(c, existingQueries[rs.ID], queryJSON); err != nil {
					return err
				}
			}
			ruleID, parseErr := uuid.Parse(rs.ID)
			if parseErr != nil {
				ruleID = uuid.New()
			}
			snapshotRuleIDs[ruleID.String()] = true
			enabled := rs.Enabled
			source := defaultStr(rs.Source, defaultStr(existingSources[ruleID.String()], "manual"))
			ruleModel := &model.ArchiveRule{
				ID:                     ruleID,
				TenantID:               tenantID,
				ConfigID:               &sourceConfigID,
				ProcessType:            snapshot.ProcessType,
				RuleContent:            rs.RuleContent,
				ApplicabilityCondition: strings.TrimSpace(rs.ApplicabilityCondition),
				RuleScope:              rs.RuleScope,
				Enabled:                &enabled,
				Source:                 source,
				RelatedFlow:            rs.RelatedFlow,
				ContextEnabled:         rs.ContextEnabled,
				DataQueries:            queryJSON,
				ContextMounts:          datatypes.JSON(`[]`),
			}
			if rs.ContextMounts != nil {
				if mountJSON, err := json.Marshal(rs.ContextMounts); err == nil {
					ruleModel.ContextMounts = datatypes.JSON(mountJSON)
				}
			}
			if _, exists := existingMap[ruleID.String()]; exists {
				if err := s.archiveRules.WithTenant(c).Model(ruleModel).Where("id = ?", ruleModel.ID).Select("rule_content", "applicability_condition", "rule_scope", "enabled", "source", "related_flow", "context_enabled", "context_mounts", "data_queries").Updates(ruleModel).Error; err != nil {
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
			ProcessType   string         `json:"process_type"`
			MainTableName string         `json:"main_table_name"`
			MainFields    datatypes.JSON `json:"main_fields"`
			DetailTables  datatypes.JSON `json:"detail_tables"`
			SummaryBlocks datatypes.JSON `json:"summary_blocks"`
			Status        string         `json:"status"`
		}
		if err := decodeContentJSON(rawSnapshot, &snapshot); err != nil {
			return err
		}
		fields := map[string]interface{}{
			"process_type":    snapshot.ProcessType,
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
		return snapshot, sourceContentFingerprint(snapshot), nil
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
		return snapshot, sourceContentFingerprint(snapshot), nil
	case model.ExecutionConfigModuleSummary:
		config, err := s.summaryConfigs.GetByID(c, sourceConfigID)
		if err != nil {
			return nil, "", err
		}
		snapshot := summaryConfigSourceSnapshot(config)
		return snapshot, sourceContentFingerprint(snapshot), nil
	default:
		return nil, "", newServiceError(errcode.ErrParamValidation, "配置模块无效")
	}
}

type ruleSource struct {
	ApplicabilityCondition string      `json:"applicability_condition,omitempty"`
	ID                     string      `json:"id"`
	RuleContent            string      `json:"rule_content"`
	RuleScope              string      `json:"rule_scope"`
	Enabled                bool        `json:"enabled"`
	Source                 string      `json:"source,omitempty"`
	RelatedFlow            bool        `json:"related_flow"`
	ContextEnabled         bool        `json:"context_enabled"`
	DataQueries            interface{} `json:"data_queries"`
	ContextMounts          interface{} `json:"context_mounts"`
}

func normalizedAuditRuleSources(rules []model.AuditRule) []ruleSource {
	result := make([]ruleSource, 0, len(rules))
	for _, rule := range rules {
		result = append(result, ruleSource{
			ID: rule.ID.String(), RuleContent: rule.RuleContent, RuleScope: rule.RuleScope,
			ApplicabilityCondition: strings.TrimSpace(rule.ApplicabilityCondition),
			Enabled:                rule.Enabled == nil || *rule.Enabled, RelatedFlow: rule.RelatedFlow,
			ContextEnabled: rule.ContextEnabled, ContextMounts: normalizeFieldArrayJSON(rule.ContextMounts), DataQueries: normalizeFieldArrayJSON(rule.DataQueries),
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
			ApplicabilityCondition: strings.TrimSpace(rule.ApplicabilityCondition),
			Enabled:                rule.Enabled == nil || *rule.Enabled, RelatedFlow: rule.RelatedFlow,
			ContextEnabled: rule.ContextEnabled, ContextMounts: normalizeFieldArrayJSON(rule.ContextMounts), DataQueries: normalizeFieldArrayJSON(rule.DataQueries),
		})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result
}

func auditConfigSourceSnapshot(config *model.ProcessAuditConfig, rules []model.AuditRule) map[string]interface{} {
	return map[string]interface{}{
		"process_type": config.ProcessType, "main_table_name": config.MainTableName,
		"main_fields": normalizeFieldArrayJSON(config.MainFields), "detail_tables": normalizeFieldArrayJSON(config.DetailTables),
		"field_mode": config.FieldMode, "kb_mode": config.KBMode,
		"ai_config": normalizedAIConfigContent(config.AIConfig), "user_permissions": normalizedPermissionContent(config.UserPermissions), "status": config.Status,
		"rules": normalizedAuditRuleSources(rules),
	}
}

func auditConfigSourceFingerprint(config *model.ProcessAuditConfig, rules []model.AuditRule) string {
	return sourceContentFingerprint(auditConfigSourceSnapshot(config, rules))
}

func archiveConfigSourceSnapshot(config *model.ProcessArchiveConfig, rules []model.ArchiveRule) map[string]interface{} {
	return map[string]interface{}{
		"process_type": config.ProcessType, "main_table_name": config.MainTableName,
		"main_fields": normalizeFieldArrayJSON(config.MainFields), "detail_tables": normalizeFieldArrayJSON(config.DetailTables),
		"field_mode": config.FieldMode, "kb_mode": config.KBMode,
		"ai_config": normalizedAIConfigContent(config.AIConfig), "user_permissions": normalizedPermissionContent(config.UserPermissions), "status": config.Status,
		"rules": normalizedArchiveRuleSources(rules),
	}
}

func archiveConfigSourceFingerprint(config *model.ProcessArchiveConfig, rules []model.ArchiveRule) string {
	return sourceContentFingerprint(archiveConfigSourceSnapshot(config, rules))
}

type summaryBlockSource struct {
	ID                   string      `json:"id"`
	Title                string      `json:"title"`
	UserPrompt           string      `json:"user_prompt"`
	EnableThinking       bool        `json:"enable_thinking"`
	IncludeMeta          bool        `json:"include_meta"`
	EnabledDataVariables []string    `json:"enabled_data_variables"`
	FieldMode            string      `json:"field_mode"`
	SelectedFields       []string    `json:"selected_fields"`
	ContextMounts        interface{} `json:"context_mounts"`
	Enabled              bool        `json:"enabled"`
	SortOrder            int         `json:"sort_order"`
}

func normalizedSummaryBlockSources(raw datatypes.JSON) []summaryBlockSource {
	if len(raw) == 0 || string(raw) == "null" {
		return []summaryBlockSource{}
	}
	var blocks []model.SummaryBlockConfig
	if err := json.Unmarshal(raw, &blocks); err != nil {
		return []summaryBlockSource{}
	}
	result := make([]summaryBlockSource, 0, len(blocks))
	for i, b := range blocks {
		enabledVars := b.EnabledDataVariables
		if enabledVars == nil {
			enabledVars = []string{}
		} else {
			enabledVars = append([]string{}, enabledVars...)
			sort.Strings(enabledVars)
		}

		selectedFields := b.SelectedFields
		if selectedFields == nil {
			selectedFields = []string{}
		} else {
			selectedFields = append([]string{}, selectedFields...)
			sort.Strings(selectedFields)
		}

		var mounts interface{} = []interface{}{}
		if b.ContextMounts != nil {
			mounts = b.ContextMounts
		}

		includeMeta := true
		if b.IncludeMeta != nil {
			includeMeta = *b.IncludeMeta
		}

		fieldMode := strings.TrimSpace(b.FieldMode)
		if fieldMode == "" {
			fieldMode = "all"
		}

		sortOrder := b.SortOrder
		if sortOrder == 0 {
			sortOrder = i + 1
		}

		result = append(result, summaryBlockSource{
			ID:                   strings.TrimSpace(b.ID),
			Title:                strings.TrimSpace(b.Title),
			UserPrompt:           strings.TrimSpace(b.UserPrompt),
			EnableThinking:       b.EnableThinking,
			IncludeMeta:          includeMeta,
			EnabledDataVariables: enabledVars,
			FieldMode:            fieldMode,
			SelectedFields:       selectedFields,
			ContextMounts:        mounts,
			Enabled:              b.Enabled,
			SortOrder:            sortOrder,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].SortOrder != result[j].SortOrder {
			return result[i].SortOrder < result[j].SortOrder
		}
		return result[i].ID < result[j].ID
	})
	return result
}

func normalizeFieldArrayJSON(raw datatypes.JSON) []interface{} {
	if len(raw) == 0 || string(raw) == "null" {
		return []interface{}{}
	}
	var arr []interface{}
	if err := decodeContentJSON(raw, &arr); err != nil || arr == nil {
		return []interface{}{}
	}
	return arr
}

func summaryConfigSourceSnapshot(config *model.ProcessSummaryConfig) map[string]interface{} {
	return map[string]interface{}{
		"process_type":    config.ProcessType,
		"main_table_name": config.MainTableName,
		"main_fields":     normalizeFieldArrayJSON(config.MainFields),
		"detail_tables":   normalizeFieldArrayJSON(config.DetailTables),
		"summary_blocks":  normalizedSummaryBlockSources(config.SummaryBlocks),
		"status":          config.Status,
	}
}

func summaryConfigSourceFingerprint(config *model.ProcessSummaryConfig) string {
	return sourceContentFingerprint(summaryConfigSourceSnapshot(config))
}

func (s *ExecutionConfigSourceService) matchesActiveSnapshot(
	module string,
	snapshotRaw datatypes.JSON,
	sourceFingerprint string,
) bool {
	if len(snapshotRaw) == 0 || string(snapshotRaw) == "null" {
		return false
	}
	switch module {
	case model.ExecutionConfigModuleAudit:
		var cfg model.ProcessAuditConfig
		var snapshot struct {
			Rules []model.AuditRule `json:"rules"`
		}
		if json.Unmarshal(snapshotRaw, &cfg) != nil || json.Unmarshal(snapshotRaw, &snapshot) != nil {
			return false
		}
		return auditConfigSourceFingerprint(&cfg, snapshot.Rules) == sourceFingerprint
	case model.ExecutionConfigModuleArchive:
		var cfg model.ProcessArchiveConfig
		var snapshot struct {
			Rules []model.ArchiveRule `json:"rules"`
		}
		if json.Unmarshal(snapshotRaw, &cfg) != nil || json.Unmarshal(snapshotRaw, &snapshot) != nil {
			return false
		}
		return archiveConfigSourceFingerprint(&cfg, snapshot.Rules) == sourceFingerprint
	case model.ExecutionConfigModuleSummary:
		var cfg model.ProcessSummaryConfig
		if json.Unmarshal(snapshotRaw, &cfg) != nil {
			return false
		}
		return summaryConfigSourceFingerprint(&cfg) == sourceFingerprint
	}
	return false
}

// SetEmbedRefreshScheduleManager 注入配置恢复后的定时检查同步器。
func (s *ExecutionConfigSourceService) SetEmbedRefreshScheduleManager(manager EmbedRefreshScheduleManager) {
	s.scheduleMgr = manager
}

// sourceChangeStatus 在配置事务提交后同步缓存与调度，避免恢复启停状态后仍沿用旧 Cron。
func (s *ExecutionConfigSourceService) sourceChangeStatus(c *gin.Context, tenantID uuid.UUID, module string, sourceConfigID uuid.UUID) (*ExecutionConfigVersionStatus, error) {
	s.invalidate(c, tenantID, module)
	if s.scheduleMgr != nil {
		var err error
		switch module {
		case model.ExecutionConfigModuleAudit:
			var config *model.ProcessAuditConfig
			config, err = s.auditConfigs.GetByID(c, sourceConfigID)
			if err == nil {
				err = s.scheduleMgr.SyncAuditConfig(c.Request.Context(), config)
			}
		case model.ExecutionConfigModuleSummary:
			var config *model.ProcessSummaryConfig
			config, err = s.summaryConfigs.GetByID(c, sourceConfigID)
			if err == nil {
				err = s.scheduleMgr.SyncSummaryConfig(c.Request.Context(), config)
			}
		}
		if err != nil {
			return nil, newServiceError(errcode.ErrDatabase, "配置已保存，但定时检查调度同步失败")
		}
	}
	return s.GetStatus(c, module, sourceConfigID)
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
		if s.dataQueries != nil {
			queries := *s.dataQueries
			queries.repo = repository.NewAuditDataQueryRepo(tx)
			scoped.dataQueries = &queries
		}
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
	if rules, ok := fields["rules"]; ok {
		var values []ruleSource
		if err := json.Unmarshal(rules, &values); err != nil {
			return fmt.Errorf("规则快照格式无效")
		}
		for _, rule := range values {
			if len([]rune(strings.TrimSpace(rule.ApplicabilityCondition))) > maxImportedRuleSize {
				return fmt.Errorf("单条规则的适用前提不能超过 5000 个字符")
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
