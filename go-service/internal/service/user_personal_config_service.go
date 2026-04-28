package service

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/google/uuid"
	"gorm.io/datatypes"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/repository"
)

// UserPersonalConfigService 处理用户个人配置的业务逻辑。
type UserPersonalConfigService struct {
	userConfigRepo    *repository.UserPersonalConfigRepo
	configRepo        *repository.ProcessAuditConfigRepo
	auditRuleRepo     *repository.AuditRuleRepo
	archiveConfigRepo *repository.ProcessArchiveConfigRepo
	archiveRuleRepo   *repository.ArchiveRuleRepo
	orgRepo           *repository.OrgRepo
}

// NewUserPersonalConfigService 创建 UserPersonalConfigService，注入所有依赖仓储。
func NewUserPersonalConfigService(
	userConfigRepo *repository.UserPersonalConfigRepo,
	configRepo *repository.ProcessAuditConfigRepo,
	auditRuleRepo *repository.AuditRuleRepo,
	archiveConfigRepo *repository.ProcessArchiveConfigRepo,
	archiveRuleRepo *repository.ArchiveRuleRepo,
	orgRepo *repository.OrgRepo,
) *UserPersonalConfigService {
	return &UserPersonalConfigService{
		userConfigRepo:    userConfigRepo,
		configRepo:        configRepo,
		auditRuleRepo:     auditRuleRepo,
		archiveConfigRepo: archiveConfigRepo,
		archiveRuleRepo:   archiveRuleRepo,
		orgRepo:           orgRepo,
	}
}

// GetProcessList 获取用户可见的审核工作台流程列表。
// 访问控制规则：access_control 所有列表均为空 → 对所有租户成员开放；
// 否则用户 ID/角色/部门命中任一列表即可访问。
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
		var ac model.AccessControlData
		if err := json.Unmarshal(cfg.AccessControl, &ac); err != nil {
			// 解析失败视为公开
			result = append(result, dto.ProcessListItem{
				ProcessType:      cfg.ProcessType,
				ProcessTypeLabel: cfg.ProcessTypeLabel,
				ConfigID:         cfg.ID.String(),
			})
			continue
		}
		// 三列表均为空 → 公开
		if len(ac.AllowedRoles) == 0 && len(ac.AllowedMembers) == 0 && len(ac.AllowedDepartments) == 0 {
			result = append(result, dto.ProcessListItem{
				ProcessType:      cfg.ProcessType,
				ProcessTypeLabel: cfg.ProcessTypeLabel,
				ConfigID:         cfg.ID.String(),
			})
			continue
		}
		if member == nil {
			continue
		}
		// 检查成员 ID
		if sliceContains(ac.AllowedMembers, member.ID.String()) {
			result = append(result, dto.ProcessListItem{ProcessType: cfg.ProcessType, ProcessTypeLabel: cfg.ProcessTypeLabel, ConfigID: cfg.ID.String()})
			continue
		}
		// 检查部门
		if sliceContains(ac.AllowedDepartments, member.DepartmentID.String()) {
			result = append(result, dto.ProcessListItem{ProcessType: cfg.ProcessType, ProcessTypeLabel: cfg.ProcessTypeLabel, ConfigID: cfg.ID.String()})
			continue
		}
		// 检查角色
		found := false
		for _, r := range member.Roles {
			if sliceContains(ac.AllowedRoles, r.ID.String()) {
				found = true
				break
			}
		}
		if found {
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

	// 获取或创建用户配置
	userCfg, err := s.userConfigRepo.GetByTenantAndUser(c, tenantID, userID)
	if err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	var auditDetails []model.AuditDetailItem
	if userCfg != nil {
		_ = json.Unmarshal(userCfg.AuditDetails, &auditDetails)
	}

	// 构建新的 AuditDetailItem
	newDetail := model.AuditDetailItem{
		ConfigID:    configID,
		ProcessType: processType,
		FieldConfig: model.FieldConfig{
			FieldMode:      req.FieldConfig.FieldMode,
			FieldOverrides: req.FieldConfig.FieldOverrides,
		},
		RuleConfig: model.RuleConfig{
			CustomRules:         make([]model.CustomRule, len(req.RuleConfig.CustomRules)),
			RuleToggleOverrides: make([]model.RuleToggleOverride, len(req.RuleConfig.RuleToggleOverrides)),
		},
		AIConfig: model.UserAIConfig{
			StrictnessOverride: req.AIConfig.StrictnessOverride,
		},
	}

	for i, r := range req.RuleConfig.CustomRules {
		newDetail.RuleConfig.CustomRules[i] = model.CustomRule{ID: r.ID, Content: r.Content, Enabled: r.Enabled, RelatedFlow: r.RelatedFlow}
	}
	for i, t := range req.RuleConfig.RuleToggleOverrides {
		newDetail.RuleConfig.RuleToggleOverrides[i] = model.RuleToggleOverride{RuleID: t.RuleID, Enabled: t.Enabled}
	}

	// 更新或追加到 auditDetails
	found := false
	for i, detail := range auditDetails {
		if detail.ProcessType == processType {
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
		UpdatedAt:    time.Now(),
	}

	if userCfg != nil {
		cfg.ID = userCfg.ID
		cfg.CronDetails = userCfg.CronDetails
		cfg.ArchiveDetails = userCfg.ArchiveDetails
	} else {
		cfg.CronDetails = datatypes.JSON([]byte("{}"))
		cfg.ArchiveDetails = datatypes.JSON([]byte("[]"))
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
		tenantRules = []model.AuditRule{}
	}

	// 获取用户个人配置
	userCfg, err := s.userConfigRepo.GetByTenantAndUser(c, tenantID, userID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	var userDetail model.AuditDetailItem
	if userCfg != nil {
		var auditDetails []model.AuditDetailItem
		if err := json.Unmarshal(userCfg.AuditDetails, &auditDetails); err == nil {
			for _, d := range auditDetails {
				if d.ProcessType == processType || (d.ConfigID != uuid.Nil && d.ConfigID == tenantCfg.ID) {
					userDetail = d
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
			customRuleDTOs[i] = dto.CustomRuleDTO{ID: r.ID, Content: r.Content, Enabled: r.Enabled, RelatedFlow: r.RelatedFlow}
		}
	} else {
		customRuleDTOs = []dto.CustomRuleDTO{}
	}

	return &dto.FullAuditProcessConfigResponse{
		ProcessType:      tenantCfg.ProcessType,
		ProcessTypeLabel: tenantCfg.ProcessTypeLabel,
		ConfigID:         tenantCfg.ID.String(),
		FieldMode:        tenantCfg.FieldMode,
		KBMode:           tenantCfg.KBMode,
		AuditStrictness:  effectiveStrictness,
		UserPermissions:  dto.UserPermissionsDTO{AllowCustomFields: perms.AllowCustomFields, AllowCustomRules: perms.AllowCustomRules, AllowModifyStrictness: perms.AllowModifyStrictness},
		MainFields:       mainFields,
		DetailTables:     detailTables,
		TenantRules:      tenantRuleDTOs,
		CustomRules:      customRuleDTOs,
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
		UpdatedAt:      time.Now(),
	}
	if userCfg != nil {
		cfg.ID = userCfg.ID
		cfg.AuditDetails = userCfg.AuditDetails
		cfg.ArchiveDetails = userCfg.ArchiveDetails
	}
	return s.userConfigRepo.Upsert(cfg)
}

// GetAccessibleArchiveConfigs 获取当前用户在租户内有权访问的归档复盘配置列表。
// 访问控制规则：access_control 所有列表均为空 → 对所有租户成员开放；
// 否则用户 ID/角色/部门命中任一列表即可访问。
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
		var ac model.AccessControlData
		if err := json.Unmarshal(cfg.AccessControl, &ac); err != nil {
			// 解析失败视为公开
			result = append(result, dto.AccessibleArchiveConfigItem{
				ProcessType:      cfg.ProcessType,
				ProcessTypeLabel: cfg.ProcessTypeLabel,
				ConfigID:         cfg.ID.String(),
			})
			continue
		}
		// 三列表均为空 → 公开
		if len(ac.AllowedRoles) == 0 && len(ac.AllowedMembers) == 0 && len(ac.AllowedDepartments) == 0 {
			result = append(result, dto.AccessibleArchiveConfigItem{
				ProcessType:      cfg.ProcessType,
				ProcessTypeLabel: cfg.ProcessTypeLabel,
				ConfigID:         cfg.ID.String(),
			})
			continue
		}
		if member == nil {
			continue
		}
		// 检查成员 ID（OrgMember ID，与前端 member.id 一致）
		if sliceContains(ac.AllowedMembers, member.ID.String()) {
			result = append(result, dto.AccessibleArchiveConfigItem{ProcessType: cfg.ProcessType, ProcessTypeLabel: cfg.ProcessTypeLabel, ConfigID: cfg.ID.String()})
			continue
		}
		// 检查部门
		if sliceContains(ac.AllowedDepartments, member.DepartmentID.String()) {
			result = append(result, dto.AccessibleArchiveConfigItem{ProcessType: cfg.ProcessType, ProcessTypeLabel: cfg.ProcessTypeLabel, ConfigID: cfg.ID.String()})
			continue
		}
		// 检查角色
		found := false
		for _, r := range member.Roles {
			if sliceContains(ac.AllowedRoles, r.ID.String()) {
				found = true
				break
			}
		}
		if found {
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
		archiveRules = []model.ArchiveRule{}
	}

	// 获取用户个人归档配置
	userCfg, err := s.userConfigRepo.GetByTenantAndUser(c, tenantID, userID)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	var userDetail model.ArchiveDetailItem
	if userCfg != nil {
		var archiveDetails []model.ArchiveDetailItem
		if err := json.Unmarshal(userCfg.ArchiveDetails, &archiveDetails); err == nil {
			for _, d := range archiveDetails {
				if d.ProcessType == processType || (d.ConfigID != uuid.Nil && d.ConfigID == tenantCfg.ID) {
					userDetail = d
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
			customRuleDTOs[i] = dto.CustomRuleDTO{ID: r.ID, Content: r.Content, Enabled: r.Enabled, RelatedFlow: r.RelatedFlow}
		}
	} else {
		customRuleDTOs = []dto.CustomRuleDTO{}
	}

	return &dto.FullArchiveConfigResponse{
		ProcessType:      tenantCfg.ProcessType,
		ProcessTypeLabel: tenantCfg.ProcessTypeLabel,
		ConfigID:         tenantCfg.ID.String(),
		FieldMode:        tenantCfg.FieldMode,
		KBMode:           tenantCfg.KBMode,
		AuditStrictness:  effectiveStrictness,
		UserPermissions:  dto.ArchiveUserPermissionsDTO{AllowCustomFields: perms.AllowCustomFields, AllowCustomRules: perms.AllowCustomRules, AllowModifyStrictness: perms.AllowModifyStrictness},
		MainFields:       mainFields,
		DetailTables:     detailTables,
		TenantRules:      ruleDTOs,
		CustomRules:      customRuleDTOs,
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

	userCfg, err := s.userConfigRepo.GetByTenantAndUser(c, tenantID, userID)
	if err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	var archiveDetails []model.ArchiveDetailItem
	if userCfg != nil {
		_ = json.Unmarshal(userCfg.ArchiveDetails, &archiveDetails)
	}

	newDetail := model.ArchiveDetailItem{
		ConfigID:    configID,
		ProcessType: processType,
		FieldConfig: model.FieldConfig{
			FieldMode:      req.FieldConfig.FieldMode,
			FieldOverrides: req.FieldConfig.FieldOverrides,
		},
		RuleConfig: model.RuleConfig{
			CustomRules:         make([]model.CustomRule, len(req.RuleConfig.CustomRules)),
			RuleToggleOverrides: make([]model.RuleToggleOverride, len(req.RuleConfig.RuleToggleOverrides)),
		},
		AIConfig: model.UserAIConfig{
			StrictnessOverride: req.AIConfig.StrictnessOverride,
		},
	}
	for i, r := range req.RuleConfig.CustomRules {
		newDetail.RuleConfig.CustomRules[i] = model.CustomRule{ID: r.ID, Content: r.Content, Enabled: r.Enabled, RelatedFlow: r.RelatedFlow}
	}
	for i, t := range req.RuleConfig.RuleToggleOverrides {
		newDetail.RuleConfig.RuleToggleOverrides[i] = model.RuleToggleOverride{RuleID: t.RuleID, Enabled: t.Enabled}
	}

	found := false
	for i, d := range archiveDetails {
		if d.ProcessType == processType {
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
		UpdatedAt:      time.Now(),
	}
	if userCfg != nil {
		cfg.ID = userCfg.ID
		cfg.AuditDetails = userCfg.AuditDetails
		cfg.CronDetails = userCfg.CronDetails
	} else {
		cfg.AuditDetails = datatypes.JSON([]byte("[]"))
		cfg.CronDetails = datatypes.JSON([]byte("{}"))
	}
	return s.userConfigRepo.Upsert(cfg)
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
