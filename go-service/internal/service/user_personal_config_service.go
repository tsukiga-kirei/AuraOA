package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"auraoa/go-service/internal/dto"
	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/apptime"
	"auraoa/go-service/internal/pkg/errcode"
	"auraoa/go-service/internal/repository"
)

// UserPersonalConfigService 处理用户个人配置的业务逻辑。
type UserPersonalConfigService struct {
	userConfigRepo    *repository.UserPersonalConfigRepo
	configRepo        *repository.ProcessAuditConfigRepo
	auditRuleRepo     *repository.AuditRuleRepo
	archiveConfigRepo *repository.ProcessArchiveConfigRepo
	archiveRuleRepo   *repository.ArchiveRuleRepo
	summaryConfigRepo *repository.ProcessSummaryConfigRepo
	orgRepo           *repository.OrgRepo
	versions          *repository.ExecutionConfigVersionRepo
	tenantRepo        *repository.TenantRepo
	oaConnRepo        *repository.OAConnectionRepo
}

// NewUserPersonalConfigService 创建 UserPersonalConfigService，注入所有依赖仓储。
func NewUserPersonalConfigService(
	userConfigRepo *repository.UserPersonalConfigRepo,
	configRepo *repository.ProcessAuditConfigRepo,
	auditRuleRepo *repository.AuditRuleRepo,
	archiveConfigRepo *repository.ProcessArchiveConfigRepo,
	archiveRuleRepo *repository.ArchiveRuleRepo,
	summaryConfigRepo *repository.ProcessSummaryConfigRepo,
	orgRepo *repository.OrgRepo,
	versions *repository.ExecutionConfigVersionRepo,
	tenantRepo *repository.TenantRepo,
	oaConnRepo *repository.OAConnectionRepo,
) *UserPersonalConfigService {
	return &UserPersonalConfigService{
		userConfigRepo:    userConfigRepo,
		configRepo:        configRepo,
		auditRuleRepo:     auditRuleRepo,
		archiveConfigRepo: archiveConfigRepo,
		archiveRuleRepo:   archiveRuleRepo,
		summaryConfigRepo: summaryConfigRepo,
		orgRepo:           orgRepo,
		versions:          versions,
		tenantRepo:        tenantRepo,
		oaConnRepo:        oaConnRepo,
	}
}

func (s *UserPersonalConfigService) ensureAuditBaseVersion(
	c *gin.Context,
	tenantID, userID uuid.UUID,
	config *model.ProcessAuditConfig,
	rules []model.AuditRule,
) (*model.TenantConfigVersion, error) {
	snapshot := auditConfigSourceSnapshot(config, rules)
	version, err := s.versions.GetOrCreateLatestBaseVersion(
		c.Request.Context(), tenantID, userID, model.ExecutionConfigModuleAudit,
		config.ID, stableJSONFingerprint(snapshot), snapshot,
	)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "读取审核基础配置版本失败")
	}
	return version, nil
}

func (s *UserPersonalConfigService) ensureArchiveBaseVersion(
	c *gin.Context,
	tenantID, userID uuid.UUID,
	config *model.ProcessArchiveConfig,
	rules []model.ArchiveRule,
) (*model.TenantConfigVersion, error) {
	snapshot := archiveConfigSourceSnapshot(config, rules)
	version, err := s.versions.GetOrCreateLatestBaseVersion(
		c.Request.Context(), tenantID, userID, model.ExecutionConfigModuleArchive,
		config.ID, stableJSONFingerprint(snapshot), snapshot,
	)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "读取归档复盘基础配置版本失败")
	}
	return version, nil
}

func validatePersonalVersion(baseVersion, requestBaseVersion, personalVersion, requestPersonalVersion int) error {
	if requestBaseVersion != baseVersion {
		return newServiceError(errcode.ErrTenantConfigVersionConflict, "租户配置已更新，请刷新页面后再保存个人配置")
	}
	if requestPersonalVersion != personalVersion {
		return newServiceError(errcode.ErrPersonalConfigVersionConflict, "个人配置已在其他页面更新，请刷新后再保存")
	}
	return nil
}

func versionedCustomRules(
	requested []dto.CustomRuleDTO,
	existing []model.CustomRule,
	baseVersion, nextPersonalVersion int,
) []model.CustomRule {
	existingByID := make(map[string]model.CustomRule, len(existing))
	for _, rule := range existing {
		existingByID[rule.ID] = rule
	}
	result := make([]model.CustomRule, len(requested))
	for i, rule := range requested {
		baseAdded := baseVersion
		personalAdded := nextPersonalVersion
		if previous, ok := existingByID[rule.ID]; ok && previous.AddedInPersonalVersion > 0 {
			baseAdded = previous.BaseConfigVersion
			personalAdded = previous.AddedInPersonalVersion
		}
		result[i] = model.CustomRule{
			ID: rule.ID, Content: rule.Content, Enabled: rule.Enabled, RelatedFlow: rule.RelatedFlow,
			BaseConfigVersion: baseAdded, AddedInPersonalVersion: personalAdded,
		}
	}
	return result
}

// userCanAccess 判断用户是否命中流程配置中的成员、角色或部门授权。
func (s *UserPersonalConfigService) userCanAccess(
	_ *gin.Context,
	tenantID, userID uuid.UUID,
	accessControl datatypes.JSON,
) bool {
	member, _ := s.orgRepo.FindByUserAndTenant(userID, tenantID)
	return accessControlAllows(accessControl, member)
}

// GetProcessList 获取用户可见的审核工作台流程列表。
// 访问控制规则：用户 ID/角色/部门命中任一列表即可访问；未配置时默认拒绝。
func (s *UserPersonalConfigService) GetProcessList(c *gin.Context, userID uuid.UUID) ([]dto.ProcessListItem, error) {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}

	configs, err := s.configRepo.ListByTenant(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	if len(configs) == 0 {
		return []dto.ProcessListItem{}, nil
	}

	// 获取用户在租户内的成员信息（角色、部门）
	member, _ := s.orgRepo.FindByUserAndTenant(userID, tenantID)

	var result []dto.ProcessListItem
	for _, cfg := range configs {
		if cfg.Status != "active" {
			continue
		}
		if accessControlAllows(cfg.AccessControl, member) {
			result = append(result, dto.ProcessListItem{ProcessType: cfg.ProcessType, ProcessTypeLabel: cfg.ProcessTypeLabel, ConfigID: cfg.ID.String()})
		}
	}

	if result == nil {
		result = []dto.ProcessListItem{}
	}
	return result, nil
}

// GetByProcessType 获取用户对指定流程的个性化配置详情。
func (s *UserPersonalConfigService) GetByProcessType(c *gin.Context, userID uuid.UUID, processType string) (*model.AuditDetailItem, error) {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}
	processCfg, err := s.configRepo.GetByProcessType(c, processType)
	if err != nil {
		return nil, newServiceError(errcode.ErrConfigNotFound, "流程审核配置不存在")
	}
	if !s.userCanAccess(c, tenantID, userID, processCfg.AccessControl) {
		return nil, newServiceError(errcode.ErrPermissionDenied, "当前用户无权访问该审核流程")
	}

	userCfg, err := s.userConfigRepo.GetByTenantAndUser(c, tenantID, userID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	if userCfg == nil {
		return nil, nil
	}

	// 从 audit_details JSON 中查找对应流程的配置
	var auditDetails []model.AuditDetailItem
	if err := json.Unmarshal(userCfg.AuditDetails, &auditDetails); err != nil {
		return nil, nil
	}

	for _, detail := range auditDetails {
		if detail.ProcessType == processType {
			return &detail, nil
		}
	}

	return nil, nil
}

// UpdateByProcessType 更新用户对指定流程的个性化配置，校验权限锁定。
func (s *UserPersonalConfigService) UpdateByProcessType(c *gin.Context, userID uuid.UUID, processType string, req *dto.UpdateUserProcessConfigRequest) error {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}

	// 获取流程审核配置，检查权限锁定
	processCfg, err := s.configRepo.GetByProcessType(c, processType)
	if err != nil {
		return newServiceError(errcode.ErrConfigNotFound, "流程审核配置不存在")
	}
	if !s.userCanAccess(c, tenantID, userID, processCfg.AccessControl) {
		return newServiceError(errcode.ErrPermissionDenied, "当前用户无权修改该审核流程配置")
	}

	configID, _ := uuid.Parse(req.ConfigID)
	if configID == uuid.Nil {
		configID = processCfg.ID
	}

	// 解析 user_permissions
	var perms model.UserPermissionsData
	if err := json.Unmarshal(processCfg.UserPermissions, &perms); err != nil {
		perms = model.UserPermissionsData{AllowCustomFields: true, AllowCustomRules: true, AllowModifyStrictness: true}
	}

	// 校验权限锁定
	if !perms.AllowCustomFields && len(req.FieldConfig.FieldOverrides) > 0 {
		return newServiceError(errcode.ErrPermissionDenied, "字段自定义功能已被锁定")
	}
	if !perms.AllowCustomRules && len(req.RuleConfig.CustomRules) > 0 {
		return newServiceError(errcode.ErrPermissionDenied, "自定义规则功能已被锁定")
	}
	if !perms.AllowModifyStrictness && req.AIConfig.StrictnessOverride != "" {
		return newServiceError(errcode.ErrPermissionDenied, "审核尺度修改功能已被锁定")
	}
	if req.AIConfig.StrictnessOverride != "" && !validStrictness(req.AIConfig.StrictnessOverride) {
		return newServiceError(errcode.ErrParamValidation, "审核尺度无效")
	}
	tenantRules, err := s.auditRuleRepo.ListByConfigID(c, processCfg.ID)
	if err != nil {
		return newServiceError(errcode.ErrDatabase, "读取审核规则失败")
	}
	baseVersion, err := s.ensureAuditBaseVersion(c, tenantID, userID, processCfg, tenantRules)
	if err != nil {
		return err
	}

	// 获取或创建用户配置
	userCfg, err := s.userConfigRepo.GetByTenantAndUser(c, tenantID, userID)
	if err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	var auditDetails []model.AuditDetailItem
	if userCfg != nil {
		_ = json.Unmarshal(userCfg.AuditDetails, &auditDetails)
	}
	var existingDetail model.AuditDetailItem
	for _, detail := range auditDetails {
		if detail.ProcessType == processType || (detail.ConfigID != uuid.Nil && detail.ConfigID == processCfg.ID) {
			existingDetail = detail
			break
		}
	}
	if err := validatePersonalVersion(
		baseVersion.VersionNo, req.BaseConfigVersion,
		existingDetail.PersonalVersion, req.PersonalVersion,
	); err != nil {
		return err
	}
	nextPersonalVersion := existingDetail.PersonalVersion + 1

	// 构建新的 AuditDetailItem
	newDetail := model.AuditDetailItem{
		ConfigID:          configID,
		ProcessType:       processType,
		BaseConfigVersion: baseVersion.VersionNo,
		PersonalVersion:   nextPersonalVersion,
		FieldConfig: model.FieldConfig{
			FieldMode:      req.FieldConfig.FieldMode,
			FieldOverrides: req.FieldConfig.FieldOverrides,
		},
		RuleConfig: model.RuleConfig{
			CustomRules:         versionedCustomRules(req.RuleConfig.CustomRules, existingDetail.RuleConfig.CustomRules, baseVersion.VersionNo, nextPersonalVersion),
			RuleToggleOverrides: make([]model.RuleToggleOverride, len(req.RuleConfig.RuleToggleOverrides)),
		},
		AIConfig: model.UserAIConfig{
			StrictnessOverride: req.AIConfig.StrictnessOverride,
		},
	}

	for i, t := range req.RuleConfig.RuleToggleOverrides {
		newDetail.RuleConfig.RuleToggleOverrides[i] = model.RuleToggleOverride{RuleID: t.RuleID, Enabled: t.Enabled}
	}

	// 更新或追加到 auditDetails
	found := false
	for i, detail := range auditDetails {
		if detail.ProcessType == processType || (detail.ConfigID != uuid.Nil && detail.ConfigID == processCfg.ID) {
			auditDetails[i] = newDetail
			found = true
			break
		}
	}
	if !found {
		auditDetails = append(auditDetails, newDetail)
	}

	auditDetailsJSON, _ := json.Marshal(auditDetails)

	cfg := &model.UserPersonalConfig{
		ID:           uuid.New(),
		TenantID:     tenantID,
		UserID:       userID,
		AuditDetails: datatypes.JSON(auditDetailsJSON),
		UpdatedAt:    apptime.Now(),
	}

	if userCfg != nil {
		cfg.ID = userCfg.ID
		cfg.CronDetails = userCfg.CronDetails
		cfg.ArchiveDetails = userCfg.ArchiveDetails
		cfg.SummaryDetails = userCfg.SummaryDetails
	} else {
		cfg.CronDetails = datatypes.JSON([]byte("{}"))
		cfg.ArchiveDetails = datatypes.JSON([]byte("[]"))
		cfg.SummaryDetails = datatypes.JSON([]byte("[]"))
	}

	if err := s.userConfigRepo.Upsert(cfg); err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return nil
}

// GetFullAuditProcessConfig 返回审核工作台指定流程的完整配置（租户字段/规则 + 用户覆盖合并）。
func (s *UserPersonalConfigService) GetFullAuditProcessConfig(c *gin.Context, userID uuid.UUID, processType string) (*dto.FullAuditProcessConfigResponse, error) {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}

	// 获取租户流程审核配置
	tenantCfg, err := s.configRepo.GetByProcessType(c, processType)
	if err != nil {
		return nil, newServiceError(errcode.ErrConfigNotFound, "流程审核配置不存在")
	}
	if !s.userCanAccess(c, tenantID, userID, tenantCfg.AccessControl) {
		return nil, newServiceError(errcode.ErrPermissionDenied, "当前用户无权访问该审核流程")
	}

	// 解析用户权限
	var perms model.UserPermissionsData
	if err := json.Unmarshal(tenantCfg.UserPermissions, &perms); err != nil {
		perms = model.UserPermissionsData{AllowCustomFields: true, AllowCustomRules: true, AllowModifyStrictness: true}
	}

	// 解析 AI 配置获取默认严格度
	var aiConfig model.AIConfigData
	_ = json.Unmarshal(tenantCfg.AIConfig, &aiConfig)
	if aiConfig.AuditStrictness == "" {
		aiConfig.AuditStrictness = "standard"
	}

	// 获取该流程的租户审核规则
	tenantRules, err := s.auditRuleRepo.ListByConfigID(c, tenantCfg.ID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "读取审核规则失败")
	}
	baseVersion, err := s.ensureAuditBaseVersion(c, tenantID, userID, tenantCfg, tenantRules)
	if err != nil {
		return nil, err
	}

	// 获取用户个人配置
	userCfg, err := s.userConfigRepo.GetByTenantAndUser(c, tenantID, userID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	var userDetail model.AuditDetailItem
	hasPersonalConfig := false
	if userCfg != nil {
		var auditDetails []model.AuditDetailItem
		if err := json.Unmarshal(userCfg.AuditDetails, &auditDetails); err == nil {
			for _, d := range auditDetails {
				if d.ProcessType == processType || (d.ConfigID != uuid.Nil && d.ConfigID == tenantCfg.ID) {
					userDetail = d
					hasPersonalConfig = true
					break
				}
			}
		}
	}

	// 规则同步逻辑：过滤掉已经不存在的租户规则覆盖
	validRuleToggles := []model.RuleToggleOverride{}
	tenantRuleMap := make(map[string]bool)
	for _, tr := range tenantRules {
		tenantRuleMap[tr.ID.String()] = true
	}
	for _, ut := range userDetail.RuleConfig.RuleToggleOverrides {
		if tenantRuleMap[ut.RuleID] {
			validRuleToggles = append(validRuleToggles, ut)
		}
	}
	userDetail.RuleConfig.RuleToggleOverrides = validRuleToggles

	// 构建规则开关 map (用于快速查找)
	toggleMap := map[string]bool{}
	for _, t := range userDetail.RuleConfig.RuleToggleOverrides {
		toggleMap[t.RuleID] = t.Enabled
	}

	// 字段合并
	fieldResult := MergeFields(FieldMergeInput{
		FieldMode:         tenantCfg.FieldMode,
		MainFieldsJSON:    tenantCfg.MainFields,
		DetailTablesJSON:  tenantCfg.DetailTables,
		UserOverrides:     userDetail.FieldConfig.FieldOverrides,
		AllowCustomFields: perms.AllowCustomFields,
	})
	mainFields := fieldResult.MainFields
	detailTables := fieldResult.DetailTables

	// 构建租户规则 DTO（应用用户开关覆盖）
	tenantRuleDTOs := make([]dto.TenantRuleDTO, len(tenantRules))
	for i, r := range tenantRules {
		effectiveEnabled := true
		if r.Enabled != nil {
			effectiveEnabled = *r.Enabled
		}

		if r.RuleScope != "mandatory" {
			if v, ok := toggleMap[r.ID.String()]; ok {
				effectiveEnabled = v
			}
		} else {
			effectiveEnabled = true // 强制开启
		}
		tenantRuleDTOs[i] = dto.TenantRuleDTO{
			ID:          r.ID.String(),
			RuleContent: r.RuleContent,
			RuleScope:   r.RuleScope,
			RelatedFlow: r.RelatedFlow,
			Enabled:     effectiveEnabled,
		}
	}

	// 有效严格度（用户覆盖优先）
	effectiveStrictness := aiConfig.AuditStrictness
	if userDetail.AIConfig.StrictnessOverride != "" && perms.AllowModifyStrictness {
		effectiveStrictness = userDetail.AIConfig.StrictnessOverride
	}

	// 构建自定义规则 DTO（仅在允许自定义规则时返回）
	var customRuleDTOs []dto.CustomRuleDTO
	if perms.AllowCustomRules {
		customRuleDTOs = make([]dto.CustomRuleDTO, len(userDetail.RuleConfig.CustomRules))
		for i, r := range userDetail.RuleConfig.CustomRules {
			customRuleDTOs[i] = dto.CustomRuleDTO{
				ID: r.ID, Content: r.Content, Enabled: r.Enabled, RelatedFlow: r.RelatedFlow,
				BaseConfigVersion: r.BaseConfigVersion, AddedInPersonalVersion: r.AddedInPersonalVersion,
			}
		}
	} else {
		customRuleDTOs = []dto.CustomRuleDTO{}
	}

	personalBaseVersion := userDetail.BaseConfigVersion
	if personalBaseVersion == 0 {
		personalBaseVersion = baseVersion.VersionNo
	}
	return &dto.FullAuditProcessConfigResponse{
		ProcessType:              tenantCfg.ProcessType,
		ProcessTypeLabel:         tenantCfg.ProcessTypeLabel,
		ConfigID:                 tenantCfg.ID.String(),
		BaseConfigVersion:        personalBaseVersion,
		CurrentBaseConfigVersion: baseVersion.VersionNo,
		PersonalVersion:          userDetail.PersonalVersion,
		HasPersonalConfig:        hasPersonalConfig,
		FieldMode:                tenantCfg.FieldMode,
		KBMode:                   tenantCfg.KBMode,
		AuditStrictness:          effectiveStrictness,
		UserPermissions:          dto.UserPermissionsDTO{AllowCustomFields: perms.AllowCustomFields, AllowCustomRules: perms.AllowCustomRules, AllowModifyStrictness: perms.AllowModifyStrictness},
		MainFields:               mainFields,
		DetailTables:             detailTables,
		TenantRules:              tenantRuleDTOs,
		CustomRules:              customRuleDTOs,
	}, nil
}

// GetCronPrefs 获取用户定时任务个人偏好（默认推送邮箱）。
func (s *UserPersonalConfigService) GetCronPrefs(c *gin.Context, userID uuid.UUID) (*dto.CronPrefsResponse, error) {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}
	userCfg, err := s.userConfigRepo.GetByTenantAndUser(c, tenantID, userID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	if userCfg == nil {
		return &dto.CronPrefsResponse{DefaultEmail: ""}, nil
	}
	var cronDetail model.CronDetailItem
	// cron_details 可能是对象或数组，兼容两种格式
	_ = json.Unmarshal(userCfg.CronDetails, &cronDetail)
	return &dto.CronPrefsResponse{DefaultEmail: cronDetail.DefaultEmail}, nil
}

// UpdateCronPrefs 更新用户定时任务个人偏好（默认推送邮箱）。
func (s *UserPersonalConfigService) UpdateCronPrefs(c *gin.Context, userID uuid.UUID, req *dto.UpdateCronPrefsRequest) error {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}
	userCfg, err := s.userConfigRepo.GetByTenantAndUser(c, tenantID, userID)
	if err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	cronDetail := model.CronDetailItem{DefaultEmail: req.DefaultEmail}
	cronJSON, _ := json.Marshal(cronDetail)

	cfg := &model.UserPersonalConfig{
		ID:             uuid.New(),
		TenantID:       tenantID,
		UserID:         userID,
		AuditDetails:   datatypes.JSON([]byte("[]")),
		CronDetails:    datatypes.JSON(cronJSON),
		ArchiveDetails: datatypes.JSON([]byte("[]")),
		SummaryDetails: datatypes.JSON([]byte("[]")),
		UpdatedAt:      apptime.Now(),
	}
	if userCfg != nil {
		cfg.ID = userCfg.ID
		cfg.AuditDetails = userCfg.AuditDetails
		cfg.ArchiveDetails = userCfg.ArchiveDetails
		cfg.SummaryDetails = userCfg.SummaryDetails
	}
	return s.userConfigRepo.Upsert(cfg)
}

// GetAccessibleArchiveConfigs 获取当前用户在租户内有权访问的归档复盘配置列表。
// 访问控制规则：用户 ID/角色/部门命中任一列表即可访问；未配置时默认拒绝。
func (s *UserPersonalConfigService) GetAccessibleArchiveConfigs(c *gin.Context, userID uuid.UUID) ([]dto.AccessibleArchiveConfigItem, error) {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}

	// 查询租户内全部归档配置
	allCfgs, err := s.archiveConfigRepo.ListByTenant(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	if len(allCfgs) == 0 {
		return []dto.AccessibleArchiveConfigItem{}, nil
	}

	// 获取用户在租户内的成员信息（角色、部门）
	member, _ := s.orgRepo.FindByUserAndTenant(userID, tenantID)

	var result []dto.AccessibleArchiveConfigItem
	for _, cfg := range allCfgs {
		if cfg.Status != "active" {
			continue
		}
		if accessControlAllows(cfg.AccessControl, member) {
			result = append(result, dto.AccessibleArchiveConfigItem{ProcessType: cfg.ProcessType, ProcessTypeLabel: cfg.ProcessTypeLabel, ConfigID: cfg.ID.String()})
		}
	}
	if result == nil {
		result = []dto.AccessibleArchiveConfigItem{}
	}
	return result, nil
}

// GetFullArchiveConfig 返回归档复盘指定流程的完整配置（租户字段/规则 + 用户覆盖合并）。
func (s *UserPersonalConfigService) GetFullArchiveConfig(c *gin.Context, userID uuid.UUID, processType string) (*dto.FullArchiveConfigResponse, error) {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}

	// 查找归档配置
	allCfgs, err := s.archiveConfigRepo.ListByTenant(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	var tenantCfg *model.ProcessArchiveConfig
	for i := range allCfgs {
		if allCfgs[i].ProcessType == processType {
			tenantCfg = &allCfgs[i]
			break
		}
	}
	if tenantCfg == nil {
		return nil, newServiceError(errcode.ErrConfigNotFound, "归档复盘配置不存在")
	}
	if !s.userCanAccess(c, tenantID, userID, tenantCfg.AccessControl) {
		return nil, newServiceError(errcode.ErrPermissionDenied, "当前用户无权访问该归档复盘")
	}

	// 解析用户权限
	var perms model.ArchiveUserPermissionsData
	if err := json.Unmarshal(tenantCfg.UserPermissions, &perms); err != nil {
		perms = model.ArchiveUserPermissionsData{AllowCustomFields: true, AllowCustomRules: true, AllowModifyStrictness: true}
	}

	// 解析 AI 配置
	var aiConfig model.AIConfigData
	_ = json.Unmarshal(tenantCfg.AIConfig, &aiConfig)
	if aiConfig.AuditStrictness == "" {
		aiConfig.AuditStrictness = "standard"
	}

	// 获取归档规则
	archiveRules, err := s.archiveRuleRepo.ListByConfigIDFilter(c, tenantCfg.ID, nil, nil)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "读取归档复盘规则失败")
	}
	baseVersion, err := s.ensureArchiveBaseVersion(c, tenantID, userID, tenantCfg, archiveRules)
	if err != nil {
		return nil, err
	}

	// 获取用户个人归档配置
	userCfg, err := s.userConfigRepo.GetByTenantAndUser(c, tenantID, userID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	var userDetail model.ArchiveDetailItem
	hasPersonalConfig := false
	if userCfg != nil {
		var archiveDetails []model.ArchiveDetailItem
		if err := json.Unmarshal(userCfg.ArchiveDetails, &archiveDetails); err == nil {
			for _, d := range archiveDetails {
				if d.ProcessType == processType || (d.ConfigID != uuid.Nil && d.ConfigID == tenantCfg.ID) {
					userDetail = d
					hasPersonalConfig = true
					break
				}
			}
		}
	}

	// 规则同步逻辑
	validRuleToggles := []model.RuleToggleOverride{}
	tenantRuleMap := make(map[string]bool)
	for _, tr := range archiveRules {
		tenantRuleMap[tr.ID.String()] = true
	}
	for _, ut := range userDetail.RuleConfig.RuleToggleOverrides {
		if tenantRuleMap[ut.RuleID] {
			validRuleToggles = append(validRuleToggles, ut)
		}
	}
	userDetail.RuleConfig.RuleToggleOverrides = validRuleToggles

	// 构建规则开关 map
	toggleMap := map[string]bool{}
	for _, t := range userDetail.RuleConfig.RuleToggleOverrides {
		toggleMap[t.RuleID] = t.Enabled
	}

	// 字段合并
	fieldResult := MergeFields(FieldMergeInput{
		FieldMode:         tenantCfg.FieldMode,
		MainFieldsJSON:    tenantCfg.MainFields,
		DetailTablesJSON:  tenantCfg.DetailTables,
		UserOverrides:     userDetail.FieldConfig.FieldOverrides,
		AllowCustomFields: perms.AllowCustomFields,
	})
	mainFields := fieldResult.MainFields
	detailTables := fieldResult.DetailTables

	// 构建归档规则 DTO
	ruleDTOs := make([]dto.TenantRuleDTO, len(archiveRules))
	for i, r := range archiveRules {
		effectiveEnabled := true
		if r.Enabled != nil {
			effectiveEnabled = *r.Enabled
		}

		if r.RuleScope != "mandatory" {
			if v, ok := toggleMap[r.ID.String()]; ok {
				effectiveEnabled = v
			}
		} else {
			effectiveEnabled = true
		}
		ruleDTOs[i] = dto.TenantRuleDTO{
			ID:          r.ID.String(),
			RuleContent: r.RuleContent,
			RuleScope:   r.RuleScope,
			RelatedFlow: r.RelatedFlow,
			Enabled:     effectiveEnabled,
		}
	}

	// 有效严格度
	effectiveStrictness := aiConfig.AuditStrictness
	if userDetail.AIConfig.StrictnessOverride != "" && perms.AllowModifyStrictness {
		effectiveStrictness = userDetail.AIConfig.StrictnessOverride
	}

	// 构建自定义规则 DTO（仅在允许自定义规则时返回）
	var customRuleDTOs []dto.CustomRuleDTO
	if perms.AllowCustomRules {
		customRuleDTOs = make([]dto.CustomRuleDTO, len(userDetail.RuleConfig.CustomRules))
		for i, r := range userDetail.RuleConfig.CustomRules {
			customRuleDTOs[i] = dto.CustomRuleDTO{
				ID: r.ID, Content: r.Content, Enabled: r.Enabled, RelatedFlow: r.RelatedFlow,
				BaseConfigVersion: r.BaseConfigVersion, AddedInPersonalVersion: r.AddedInPersonalVersion,
			}
		}
	} else {
		customRuleDTOs = []dto.CustomRuleDTO{}
	}

	personalBaseVersion := userDetail.BaseConfigVersion
	if personalBaseVersion == 0 {
		personalBaseVersion = baseVersion.VersionNo
	}
	return &dto.FullArchiveConfigResponse{
		ProcessType:              tenantCfg.ProcessType,
		ProcessTypeLabel:         tenantCfg.ProcessTypeLabel,
		ConfigID:                 tenantCfg.ID.String(),
		BaseConfigVersion:        personalBaseVersion,
		CurrentBaseConfigVersion: baseVersion.VersionNo,
		PersonalVersion:          userDetail.PersonalVersion,
		HasPersonalConfig:        hasPersonalConfig,
		FieldMode:                tenantCfg.FieldMode,
		KBMode:                   tenantCfg.KBMode,
		AuditStrictness:          effectiveStrictness,
		UserPermissions:          dto.ArchiveUserPermissionsDTO{AllowCustomFields: perms.AllowCustomFields, AllowCustomRules: perms.AllowCustomRules, AllowModifyStrictness: perms.AllowModifyStrictness},
		MainFields:               mainFields,
		DetailTables:             detailTables,
		TenantRules:              ruleDTOs,
		CustomRules:              customRuleDTOs,
	}, nil
}

// UpdateArchiveConfig 更新用户归档复盘个人配置。
func (s *UserPersonalConfigService) UpdateArchiveConfig(c *gin.Context, userID uuid.UUID, processType string, req *dto.UpdateArchiveConfigRequest) error {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}

	// 检查归档配置权限
	allCfgs, err := s.archiveConfigRepo.ListByTenant(c)
	if err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	var tenantCfg *model.ProcessArchiveConfig
	for i := range allCfgs {
		if allCfgs[i].ProcessType == processType {
			tenantCfg = &allCfgs[i]
			break
		}
	}
	if tenantCfg == nil {
		return newServiceError(errcode.ErrConfigNotFound, "归档复盘配置不存在")
	}
	if !s.userCanAccess(c, tenantID, userID, tenantCfg.AccessControl) {
		return newServiceError(errcode.ErrPermissionDenied, "当前用户无权修改该归档复盘配置")
	}

	configID, _ := uuid.Parse(req.ConfigID)
	if configID == uuid.Nil {
		configID = tenantCfg.ID
	}

	var perms model.ArchiveUserPermissionsData
	if err := json.Unmarshal(tenantCfg.UserPermissions, &perms); err != nil {
		perms = model.ArchiveUserPermissionsData{AllowCustomFields: true, AllowCustomRules: true, AllowModifyStrictness: true}
	}

	if !perms.AllowCustomFields && len(req.FieldConfig.FieldOverrides) > 0 {
		return newServiceError(errcode.ErrPermissionDenied, "字段自定义功能已被锁定")
	}
	if !perms.AllowCustomRules && len(req.RuleConfig.CustomRules) > 0 {
		return newServiceError(errcode.ErrPermissionDenied, "自定义规则功能已被锁定")
	}
	if !perms.AllowModifyStrictness && req.AIConfig.StrictnessOverride != "" {
		return newServiceError(errcode.ErrPermissionDenied, "复核尺度修改功能已被锁定")
	}
	if req.AIConfig.StrictnessOverride != "" && !validStrictness(req.AIConfig.StrictnessOverride) {
		return newServiceError(errcode.ErrParamValidation, "复核尺度无效")
	}
	archiveRules, err := s.archiveRuleRepo.ListByConfigIDFilter(c, tenantCfg.ID, nil, nil)
	if err != nil {
		return newServiceError(errcode.ErrDatabase, "读取归档复盘规则失败")
	}
	baseVersion, err := s.ensureArchiveBaseVersion(c, tenantID, userID, tenantCfg, archiveRules)
	if err != nil {
		return err
	}

	userCfg, err := s.userConfigRepo.GetByTenantAndUser(c, tenantID, userID)
	if err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	var archiveDetails []model.ArchiveDetailItem
	if userCfg != nil {
		_ = json.Unmarshal(userCfg.ArchiveDetails, &archiveDetails)
	}
	var existingDetail model.ArchiveDetailItem
	for _, detail := range archiveDetails {
		if detail.ProcessType == processType || (detail.ConfigID != uuid.Nil && detail.ConfigID == tenantCfg.ID) {
			existingDetail = detail
			break
		}
	}
	if err := validatePersonalVersion(
		baseVersion.VersionNo, req.BaseConfigVersion,
		existingDetail.PersonalVersion, req.PersonalVersion,
	); err != nil {
		return err
	}
	nextPersonalVersion := existingDetail.PersonalVersion + 1

	newDetail := model.ArchiveDetailItem{
		ConfigID:          configID,
		ProcessType:       processType,
		BaseConfigVersion: baseVersion.VersionNo,
		PersonalVersion:   nextPersonalVersion,
		FieldConfig: model.FieldConfig{
			FieldMode:      req.FieldConfig.FieldMode,
			FieldOverrides: req.FieldConfig.FieldOverrides,
		},
		RuleConfig: model.RuleConfig{
			CustomRules:         versionedCustomRules(req.RuleConfig.CustomRules, existingDetail.RuleConfig.CustomRules, baseVersion.VersionNo, nextPersonalVersion),
			RuleToggleOverrides: make([]model.RuleToggleOverride, len(req.RuleConfig.RuleToggleOverrides)),
		},
		AIConfig: model.UserAIConfig{
			StrictnessOverride: req.AIConfig.StrictnessOverride,
		},
	}
	for i, t := range req.RuleConfig.RuleToggleOverrides {
		newDetail.RuleConfig.RuleToggleOverrides[i] = model.RuleToggleOverride{RuleID: t.RuleID, Enabled: t.Enabled}
	}

	found := false
	for i, d := range archiveDetails {
		if d.ProcessType == processType || (d.ConfigID != uuid.Nil && d.ConfigID == tenantCfg.ID) {
			archiveDetails[i] = newDetail
			found = true
			break
		}
	}
	if !found {
		archiveDetails = append(archiveDetails, newDetail)
	}

	archiveJSON, _ := json.Marshal(archiveDetails)

	cfg := &model.UserPersonalConfig{
		ID:             uuid.New(),
		TenantID:       tenantID,
		UserID:         userID,
		ArchiveDetails: datatypes.JSON(archiveJSON),
		UpdatedAt:      apptime.Now(),
	}
	if userCfg != nil {
		cfg.ID = userCfg.ID
		cfg.AuditDetails = userCfg.AuditDetails
		cfg.CronDetails = userCfg.CronDetails
		cfg.SummaryDetails = userCfg.SummaryDetails
	} else {
		cfg.AuditDetails = datatypes.JSON([]byte("[]"))
		cfg.CronDetails = datatypes.JSON([]byte("{}"))
		cfg.SummaryDetails = datatypes.JSON([]byte("[]"))
	}
	return s.userConfigRepo.Upsert(cfg)
}

// GetAccessibleSummaryConfigs 获取当前用户可用的流程总结配置列表。
// 流程数据的实际可见性仍由工作台调用 OA 用户待办与归档列表决定。
func (s *UserPersonalConfigService) GetAccessibleSummaryConfigs(c *gin.Context) ([]dto.ProcessListItem, error) {
	configs, err := s.summaryConfigRepo.ListByTenant(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "读取流程总结配置失败")
	}
	result := make([]dto.ProcessListItem, 0, len(configs))
	for _, cfg := range configs {
		if cfg.Status != "active" {
			continue
		}
		result = append(result, dto.ProcessListItem{
			ProcessType: cfg.ProcessType, ProcessTypeLabel: cfg.ProcessTypeLabel, ConfigID: cfg.ID.String(),
		})
	}
	return result, nil
}

// GetFullSummaryPreference 返回租户定义的总结块与当前用户展示偏好的合并视图。
func (s *UserPersonalConfigService) GetFullSummaryPreference(c *gin.Context, userID uuid.UUID, processType string) (*dto.FullSummaryPreferenceResponse, error) {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}
	config, err := s.summaryConfigRepo.GetByProcessType(c, processType)
	if err != nil || config.Status != "active" {
		return nil, newServiceError(errcode.ErrConfigNotFound, "流程总结配置不存在或已停用")
	}

	visibleIDs := map[string]bool{}
	hasPreference := false
	userCfg, err := s.userConfigRepo.GetByTenantAndUser(c, tenantID, userID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "读取个人流程总结偏好失败")
	}
	if userCfg != nil {
		var details []model.SummaryDetailItem
		if json.Unmarshal(userCfg.SummaryDetails, &details) == nil {
			for _, detail := range details {
				if detail.ProcessType == processType || (detail.ConfigID != uuid.Nil && detail.ConfigID == config.ID) {
					hasPreference = true
					for _, id := range detail.VisibleBlockIDs {
						visibleIDs[id] = true
					}
					break
				}
			}
		}
	}

	blocks := parseSummaryBlocks(config.SummaryBlocks)
	resultBlocks := make([]dto.SummaryBlockPreferenceDTO, 0, len(blocks))
	for _, block := range blocks {
		if !block.Enabled {
			continue
		}
		resultBlocks = append(resultBlocks, dto.SummaryBlockPreferenceDTO{
			ID: block.ID, Title: block.Title, Visible: !hasPreference || visibleIDs[block.ID], EnableThinking: block.EnableThinking,
		})
	}
	return &dto.FullSummaryPreferenceResponse{
		ProcessType: config.ProcessType, ProcessTypeLabel: config.ProcessTypeLabel,
		ConfigID: config.ID.String(), Blocks: resultBlocks,
	}, nil
}

// UpdateSummaryPreference 更新用户在流程总结工作台中的分块展示偏好。
func (s *UserPersonalConfigService) UpdateSummaryPreference(c *gin.Context, userID uuid.UUID, processType string, req *dto.UpdateSummaryPreferenceRequest) error {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}
	config, err := s.summaryConfigRepo.GetByProcessType(c, processType)
	if err != nil || config.Status != "active" {
		return newServiceError(errcode.ErrConfigNotFound, "流程总结配置不存在或已停用")
	}
	configID, err := uuid.Parse(req.ConfigID)
	if err != nil || configID != config.ID {
		return newServiceError(errcode.ErrParamValidation, "流程总结配置与流程类型不匹配")
	}
	if len(req.VisibleBlockIDs) == 0 {
		return newServiceError(errcode.ErrParamValidation, "至少保留一个可见总结块")
	}
	validIDs := make(map[string]bool)
	for _, block := range parseSummaryBlocks(config.SummaryBlocks) {
		if block.Enabled {
			validIDs[block.ID] = true
		}
	}
	seen := make(map[string]bool)
	visibleIDs := make([]string, 0, len(req.VisibleBlockIDs))
	for _, id := range req.VisibleBlockIDs {
		id = strings.TrimSpace(id)
		if id == "" || !validIDs[id] {
			return newServiceError(errcode.ErrParamValidation, "包含无效或已停用的总结块")
		}
		if !seen[id] {
			seen[id] = true
			visibleIDs = append(visibleIDs, id)
		}
	}

	userCfg, err := s.userConfigRepo.GetByTenantAndUser(c, tenantID, userID)
	if err != nil {
		return newServiceError(errcode.ErrDatabase, "读取个人流程总结偏好失败")
	}
	details := []model.SummaryDetailItem{}
	if userCfg != nil {
		_ = json.Unmarshal(userCfg.SummaryDetails, &details)
	}
	newDetail := model.SummaryDetailItem{ConfigID: config.ID, ProcessType: processType, VisibleBlockIDs: visibleIDs}
	found := false
	for i, detail := range details {
		if detail.ProcessType == processType || (detail.ConfigID != uuid.Nil && detail.ConfigID == config.ID) {
			details[i] = newDetail
			found = true
			break
		}
	}
	if !found {
		details = append(details, newDetail)
	}
	raw, _ := json.Marshal(details)
	cfg := &model.UserPersonalConfig{
		ID: uuid.New(), TenantID: tenantID, UserID: userID,
		AuditDetails: datatypes.JSON([]byte("[]")), CronDetails: datatypes.JSON([]byte("{}")),
		ArchiveDetails: datatypes.JSON([]byte("[]")), SummaryDetails: datatypes.JSON(raw), UpdatedAt: apptime.Now(),
	}
	if userCfg != nil {
		cfg.ID = userCfg.ID
		cfg.AuditDetails = userCfg.AuditDetails
		cfg.CronDetails = userCfg.CronDetails
		cfg.ArchiveDetails = userCfg.ArchiveDetails
	}
	if err := s.userConfigRepo.Upsert(cfg); err != nil {
		return newServiceError(errcode.ErrDatabase, "保存个人流程总结偏好失败")
	}
	return nil
}

// sliceContains 检查字符串切片是否包含指定值。

func sliceContains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

// GetBaselineVersionDiff 对比指定流程在两个租户基础版本之间的配置差异（规则变动、字段变动、尺度变动）。
func (s *UserPersonalConfigService) GetBaselineVersionDiff(
	c *gin.Context,
	module, processType string,
	fromVersionNo, toVersionNo int,
) (*dto.BaselineVersionDiffResponse, error) {
	tenantID, err := getTenantUUID(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "租户ID无效")
	}

	var sourceConfigID uuid.UUID
	switch module {
	case model.ExecutionConfigModuleAudit:
		cfg, err := s.configRepo.GetByProcessType(c, processType)
		if err != nil {
			return nil, newServiceError(errcode.ErrConfigNotFound, "流程审核配置不存在")
		}
		sourceConfigID = cfg.ID
	case model.ExecutionConfigModuleArchive:
		cfg, err := s.archiveConfigRepo.GetByProcessType(c, processType)
		if err != nil {
			return nil, newServiceError(errcode.ErrConfigNotFound, "流程归档复盘配置不存在")
		}
		sourceConfigID = cfg.ID
	default:
		return nil, newServiceError(errcode.ErrParamValidation, "不支持的配置模块")
	}

	// 自动校正版本号
	if toVersionNo <= 0 {
		latest, err := s.versions.GetActiveBaseVersion(c.Request.Context(), tenantID, module, sourceConfigID)
		if err != nil || latest == nil {
			return nil, newServiceError(errcode.ErrResourceNotFound, "未找到目标基线版本")
		}
		toVersionNo = latest.VersionNo
	}
	if fromVersionNo <= 0 {
		fromVersionNo = toVersionNo - 1
		if fromVersionNo < 1 {
			fromVersionNo = 1
		}
	}

	fromVersion, err := s.versions.GetBaseVersionByNo(c.Request.Context(), tenantID, module, sourceConfigID, fromVersionNo)
	if err != nil {
		return nil, newServiceError(errcode.ErrResourceNotFound, fmt.Sprintf("未找到起始版本 v%d", fromVersionNo))
	}
	toVersion, err := s.versions.GetBaseVersionByNo(c.Request.Context(), tenantID, module, sourceConfigID, toVersionNo)
	if err != nil {
		return nil, newServiceError(errcode.ErrResourceNotFound, fmt.Sprintf("未找到目标版本 v%d", toVersionNo))
	}

	return computeBaselineDiff(processType, fromVersion, toVersion)
}

func computeBaselineDiff(processType string, fromVersion, toVersion *model.TenantConfigVersion) (*dto.BaselineVersionDiffResponse, error) {
	type ruleItem struct {
		ID          string `json:"id"`
		RuleContent string `json:"rule_content"`
		RuleScope   string `json:"rule_scope"`
	}
	type fieldItem struct {
		FieldKey  string `json:"field_key"`
		FieldName string `json:"field_name"`
		Selected  bool   `json:"selected"`
	}
	type detailTableItem struct {
		TableName  string      `json:"table_name"`
		TableLabel string      `json:"table_label"`
		Fields     []fieldItem `json:"fields"`
	}
	type snapshotObj struct {
		FieldMode    string            `json:"field_mode"`
		MainFields   datatypes.JSON    `json:"main_fields"`
		DetailTables datatypes.JSON    `json:"detail_tables"`
		AIConfig     datatypes.JSON    `json:"ai_config"`
		Rules        []ruleItem        `json:"rules"`
	}

	var fromSnap, toSnap snapshotObj
	_ = json.Unmarshal(fromVersion.ConfigSnapshot, &fromSnap)
	_ = json.Unmarshal(toVersion.ConfigSnapshot, &toSnap)

	resp := &dto.BaselineVersionDiffResponse{
		ProcessType:   processType,
		FromVersionNo: fromVersion.VersionNo,
		ToVersionNo:   toVersion.VersionNo,
		AddedRules:    []dto.RuleDiffItem{},
		RemovedRules:  []dto.RuleDiffItem{},
		ModifiedRules: []dto.RuleDiffItem{},
		AddedFields:   []dto.FieldDiffItem{},
		RemovedFields: []dto.FieldDiffItem{},
	}

	// 1. 比对规则
	fromRulesMap := make(map[string]ruleItem, len(fromSnap.Rules))
	for _, r := range fromSnap.Rules {
		fromRulesMap[r.ID] = r
	}
	toRulesMap := make(map[string]ruleItem, len(toSnap.Rules))
	for _, r := range toSnap.Rules {
		toRulesMap[r.ID] = r
		if old, exists := fromRulesMap[r.ID]; !exists {
			resp.AddedRules = append(resp.AddedRules, dto.RuleDiffItem{
				ID: r.ID, RuleContent: r.RuleContent, RuleScope: r.RuleScope,
			})
		} else if old.RuleContent != r.RuleContent || old.RuleScope != r.RuleScope {
			desc := "内容修改"
			if old.RuleScope != r.RuleScope {
				desc = fmt.Sprintf("范围变动: %s → %s", old.RuleScope, r.RuleScope)
			}
			resp.ModifiedRules = append(resp.ModifiedRules, dto.RuleDiffItem{
				ID: r.ID, RuleContent: r.RuleContent, RuleScope: r.RuleScope, ChangeDesc: desc,
			})
		}
	}
	for _, r := range fromSnap.Rules {
		if _, exists := toRulesMap[r.ID]; !exists {
			resp.RemovedRules = append(resp.RemovedRules, dto.RuleDiffItem{
				ID: r.ID, RuleContent: r.RuleContent, RuleScope: r.RuleScope,
			})
		}
	}

	// 2. 比对字段
	parseSelectedFields := func(mainJSON, detailJSON datatypes.JSON) map[string]string {
		fieldsMap := make(map[string]string)
		var main []fieldItem
		_ = json.Unmarshal(mainJSON, &main)
		for _, f := range main {
			if f.Selected {
				name := f.FieldName
				if name == "" {
					name = f.FieldKey
				}
				fieldsMap["main:"+f.FieldKey] = name
			}
		}
		var details []detailTableItem
		_ = json.Unmarshal(detailJSON, &details)
		for _, dt := range details {
			for _, f := range dt.Fields {
				if f.Selected {
					name := f.FieldName
					if name == "" {
						name = f.FieldKey
					}
					fieldsMap[dt.TableName+":"+f.FieldKey] = name
				}
			}
		}
		return fieldsMap
	}

	fromFields := parseSelectedFields(fromSnap.MainFields, fromSnap.DetailTables)
	toFields := parseSelectedFields(toSnap.MainFields, toSnap.DetailTables)

	for k, name := range toFields {
		if _, exists := fromFields[k]; !exists {
			parts := strings.SplitN(k, ":", 2)
			table := parts[0]
			fieldKey := parts[1]
			resp.AddedFields = append(resp.AddedFields, dto.FieldDiffItem{
				Table: table, FieldKey: fieldKey, FieldName: name,
			})
		}
	}
	for k, name := range fromFields {
		if _, exists := toFields[k]; !exists {
			parts := strings.SplitN(k, ":", 2)
			table := parts[0]
			fieldKey := parts[1]
			resp.RemovedFields = append(resp.RemovedFields, dto.FieldDiffItem{
				Table: table, FieldKey: fieldKey, FieldName: name,
			})
		}
	}

	// 3. 严格度与模式变动
	var fromAI, toAI struct {
		AuditStrictness string `json:"audit_strictness"`
	}
	_ = json.Unmarshal(fromSnap.AIConfig, &fromAI)
	_ = json.Unmarshal(toSnap.AIConfig, &toAI)
	if fromAI.AuditStrictness != toAI.AuditStrictness {
		resp.StrictnessFrom = fromAI.AuditStrictness
		resp.StrictnessTo = toAI.AuditStrictness
	}
	if fromSnap.FieldMode != toSnap.FieldMode {
		resp.FieldModeFrom = fromSnap.FieldMode
		resp.FieldModeTo = toSnap.FieldMode
	}

	changes := len(resp.AddedRules) + len(resp.RemovedRules) + len(resp.ModifiedRules) +
		len(resp.AddedFields) + len(resp.RemovedFields)
	if resp.StrictnessFrom != "" {
		changes++
	}
	if resp.FieldModeFrom != "" {
		changes++
	}
	resp.TotalChanges = changes

	return resp, nil
}

// GetOAJumpConfig 获取当前租户关联的 OA 系统流程跳转配置。
func (s *UserPersonalConfigService) GetOAJumpConfig(tenantID uuid.UUID) (*dto.OAJumpConfigResponse, error) {
	resp := &dto.OAJumpConfigResponse{
		Enabled: false,
	}

	if s.tenantRepo == nil || s.oaConnRepo == nil {
		return resp, nil
	}

	tenant, err := s.tenantRepo.FindByID(tenantID)
	if err != nil || tenant == nil || tenant.OADBConnectionID == nil {
		return resp, nil
	}

	oaConn, err := s.oaConnRepo.FindByID(*tenant.OADBConnectionID)
	if err != nil || oaConn == nil || !oaConn.Enabled {
		return resp, nil
	}

	baseURL := strings.TrimSpace(oaConn.OABaseURL)
	template := strings.TrimSpace(oaConn.ProcessURLTemplate)

	// 若未配置 baseURL 也未配置 template，则说明未开启跳转
	if baseURL == "" && template == "" {
		return resp, nil
	}

	resp.Enabled = true
	resp.OABaseURL = baseURL
	resp.ProcessURLTemplate = template
	resp.ResolvedTemplate = BuildOAProcessURL(baseURL, template, oaConn.OAType, "{process_id}")

	return resp, nil
}

