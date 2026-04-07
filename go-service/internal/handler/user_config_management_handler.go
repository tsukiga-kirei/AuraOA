package handler

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/response"
	"oa-smart-audit/go-service/internal/repository"
)

// UserConfigManagementHandler 处理租户管理端的用户配置管理 HTTP 请求。
type UserConfigManagementHandler struct {
	userConfigRepo    *repository.UserPersonalConfigRepo
	cronTaskRepo      *repository.CronTaskRepo
	orgRepo           *repository.OrgRepo
	auditRuleRepo     *repository.AuditRuleRepo
	archiveRuleRepo   *repository.ArchiveRuleRepo
	auditConfigRepo   *repository.ProcessAuditConfigRepo
	archiveConfigRepo *repository.ProcessArchiveConfigRepo
}

// NewUserConfigManagementHandler 创建一个新的 UserConfigManagementHandler 实例。
func NewUserConfigManagementHandler(
	userConfigRepo *repository.UserPersonalConfigRepo,
	cronTaskRepo *repository.CronTaskRepo,
	orgRepo *repository.OrgRepo,
	auditRuleRepo *repository.AuditRuleRepo,
	archiveRuleRepo *repository.ArchiveRuleRepo,
	auditConfigRepo *repository.ProcessAuditConfigRepo,
	archiveConfigRepo *repository.ProcessArchiveConfigRepo,
) *UserConfigManagementHandler {
	return &UserConfigManagementHandler{
		userConfigRepo:    userConfigRepo,
		cronTaskRepo:      cronTaskRepo,
		orgRepo:           orgRepo,
		auditRuleRepo:     auditRuleRepo,
		archiveRuleRepo:   archiveRuleRepo,
		auditConfigRepo:   auditConfigRepo,
		archiveConfigRepo: archiveConfigRepo,
	}
}

// ListUserConfigs 处理 GET /api/tenant/user-configs
// 返回当前租户内有个人配置记录或定时任务实例的用户，附带成员信息和配置摘要。
func (h *UserConfigManagementHandler) ListUserConfigs(c *gin.Context) {
	configs, err := h.userConfigRepo.ListByTenant(c)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, errcode.ErrDatabase, "数据库错误")
		return
	}
	cronTasks, err := h.cronTaskRepo.ListByTenant(c)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, errcode.ErrDatabase, "数据库错误")
		return
	}

	memberList, memberMap, auditRuleMap, archiveRuleMap, auditPermsMap, archivePermsMap, archiveAccessMap, auditFieldsMap, archiveFieldsMap := h.loadSharedMaps(c)
	configMap := make(map[uuid.UUID]*model.UserPersonalConfig, len(configs))
	cronTasksByUser := groupCronTasksByOwner(cronTasks)
	orderedUserIDs := make([]uuid.UUID, 0, len(configs)+len(cronTasksByUser))
	seen := make(map[uuid.UUID]struct{}, len(configs)+len(cronTasksByUser))

	for i := range configs {
		cfg := &configs[i]
		configMap[cfg.UserID] = cfg
		if _, exists := seen[cfg.UserID]; exists {
			continue
		}
		orderedUserIDs = append(orderedUserIDs, cfg.UserID)
		seen[cfg.UserID] = struct{}{}
	}

	for _, member := range memberList {
		if len(cronTasksByUser[member.UserID]) == 0 {
			continue
		}
		if _, exists := seen[member.UserID]; exists {
			continue
		}
		orderedUserIDs = append(orderedUserIDs, member.UserID)
		seen[member.UserID] = struct{}{}
	}

	extraUserIDs := make([]string, 0)
	extraUserMap := make(map[string]uuid.UUID)
	for userID := range cronTasksByUser {
		if _, exists := seen[userID]; exists {
			continue
		}
		key := userID.String()
		extraUserIDs = append(extraUserIDs, key)
		extraUserMap[key] = userID
	}
	sort.Strings(extraUserIDs)
	for _, key := range extraUserIDs {
		orderedUserIDs = append(orderedUserIDs, extraUserMap[key])
	}

	result := make([]dto.AdminUserConfigListItem, 0, len(orderedUserIDs))
	for _, userID := range orderedUserIDs {
		item := buildAdminUserConfigItem(userID, configMap[userID], cronTasksByUser[userID], memberMap, auditRuleMap, archiveRuleMap, auditPermsMap, archivePermsMap, archiveAccessMap, auditFieldsMap, archiveFieldsMap)
		result = append(result, item)
	}

	response.Success(c, result)
}

// GetUserConfig 处理 GET /api/tenant/user-configs/:userId
func (h *UserConfigManagementHandler) GetUserConfig(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("userId"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	cfg, err := h.userConfigRepo.GetByUserID(c, userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, errcode.ErrDatabase, "数据库错误")
		return
	}
	cronTasks, err := h.cronTaskRepo.ListByOwner(c, userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, errcode.ErrDatabase, "数据库错误")
		return
	}
	if cfg == nil && len(cronTasks) == 0 {
		response.Error(c, http.StatusNotFound, errcode.ErrResourceNotFound, "用户配置不存在")
		return
	}

	_, memberMap, auditRuleMap, archiveRuleMap, auditPermsMap, archivePermsMap, archiveAccessMap, auditFieldsMap, archiveFieldsMap := h.loadSharedMaps(c)
	item := buildAdminUserConfigItem(userID, cfg, cronTasks, memberMap, auditRuleMap, archiveRuleMap, auditPermsMap, archivePermsMap, archiveAccessMap, auditFieldsMap, archiveFieldsMap)
	response.Success(c, item)

}

// ruleInfo 保存规则的内容、管理员启用状态和作用域，用于 toggle 覆盖的展示对比。
type ruleInfo struct {
	Content      string
	AdminEnabled bool
	RuleScope    string
}

// loadSharedMaps 批量加载成员、规则、权限及访问控制映射，供各接口共用。
func (h *UserConfigManagementHandler) loadSharedMaps(c *gin.Context) (
	memberList []model.OrgMember,
	memberMap map[uuid.UUID]*model.OrgMember,
	auditRuleMap map[string]ruleInfo,
	archiveRuleMap map[string]ruleInfo,
	auditPermsMap map[string]model.UserPermissionsData,
	archivePermsMap map[string]model.ArchiveUserPermissionsData,
	archiveAccessMap map[string]model.AccessControlData,
	auditFieldsMap map[string]processConfigInfo,
	archiveFieldsMap map[string]processConfigInfo,
) {
	memberMap = make(map[uuid.UUID]*model.OrgMember)
	if members, err := h.orgRepo.ListMembers(c); err == nil {
		memberList = members
		for i := range memberList {
			memberMap[memberList[i].UserID] = &memberList[i]
		}
	}

	auditRuleMap = make(map[string]ruleInfo)
	if rules, err := h.auditRuleRepo.ListByTenant(c); err == nil {
		for _, r := range rules {
			enabled := r.Enabled != nil && *r.Enabled
			auditRuleMap[r.ID.String()] = ruleInfo{Content: r.RuleContent, AdminEnabled: enabled, RuleScope: r.RuleScope}
		}
	}

	archiveRuleMap = make(map[string]ruleInfo)
	if rules, err := h.archiveRuleRepo.ListByTenant(c); err == nil {
		for _, r := range rules {
			enabled := r.Enabled != nil && *r.Enabled
			archiveRuleMap[r.ID.String()] = ruleInfo{Content: r.RuleContent, AdminEnabled: enabled, RuleScope: r.RuleScope}
		}
	}

	// 审核工作台权限和字段映射
	auditPermsMap = make(map[string]model.UserPermissionsData)
	auditFieldsMap = make(map[string]processConfigInfo)
	if cfgs, err := h.auditConfigRepo.ListByTenant(c); err == nil {
		for _, cfg := range cfgs {
			var perms model.UserPermissionsData
			if err2 := json.Unmarshal(cfg.UserPermissions, &perms); err2 == nil {
				auditPermsMap[cfg.ProcessType] = perms
			}
			var mf []rawField
			_ = json.Unmarshal(cfg.MainFields, &mf)
			var dt []rawDetailTable
			_ = json.Unmarshal(cfg.DetailTables, &dt)
			auditFieldsMap[cfg.ProcessType] = processConfigInfo{FieldMode: cfg.FieldMode, MainFields: mf, DetailTables: dt}
		}
	}

	// 归档复盘权限、字段及访问控制映射
	archivePermsMap = make(map[string]model.ArchiveUserPermissionsData)
	archiveAccessMap = make(map[string]model.AccessControlData)
	archiveFieldsMap = make(map[string]processConfigInfo)
	if cfgs, err := h.archiveConfigRepo.ListByTenant(c); err == nil {
		for _, cfg := range cfgs {
			if cfg.Status != "active" {
				continue
			}
			var perms model.ArchiveUserPermissionsData
			if err2 := json.Unmarshal(cfg.UserPermissions, &perms); err2 == nil {
				archivePermsMap[cfg.ProcessType] = perms
			}
			var ac model.AccessControlData
			if err2 := json.Unmarshal(cfg.AccessControl, &ac); err2 == nil {
				archiveAccessMap[cfg.ProcessType] = ac
			}
			var mf []rawField
			_ = json.Unmarshal(cfg.MainFields, &mf)
			var dt []rawDetailTable
			_ = json.Unmarshal(cfg.DetailTables, &dt)
			archiveFieldsMap[cfg.ProcessType] = processConfigInfo{FieldMode: cfg.FieldMode, MainFields: mf, DetailTables: dt}
		}
	}
	return
}

type processConfigInfo struct {
	FieldMode    string
	MainFields   []rawField
	DetailTables []rawDetailTable
}

type rawField struct {
	FieldKey  string `json:"field_key"`
	FieldName string `json:"field_name"`
	Selected  bool   `json:"selected"`
}

type rawDetailTable struct {
	TableName  string     `json:"table_name"`
	TableLabel string     `json:"table_label"`
	Fields     []rawField `json:"fields"`
}

// buildAdminUserConfigItem 将原始 UserPersonalConfig 富化为管理员视图 DTO。
// 以管理员权限配置为基准：被禁用的特性对应的用户数据视为空；已删除的规则从 toggle 列表中清除。
// 归档复盘流程额外进行访问控制检查，用户无权访问的流程直接跳过。
func buildAdminUserConfigItem(
	userID uuid.UUID,
	cfg *model.UserPersonalConfig,
	cronTasks []model.CronTask,
	memberMap map[uuid.UUID]*model.OrgMember,
	auditRuleMap map[string]ruleInfo,
	archiveRuleMap map[string]ruleInfo,
	auditPermsMap map[string]model.UserPermissionsData,
	archivePermsMap map[string]model.ArchiveUserPermissionsData,
	archiveAccessMap map[string]model.AccessControlData,
	auditFieldsMap map[string]processConfigInfo,
	archiveFieldsMap map[string]processConfigInfo,
) dto.AdminUserConfigListItem {
	item := dto.AdminUserConfigListItem{
		UserID:         userID.String(),
		RoleNames:      []string{},
		AuditDetails:   []dto.AdminProcessDetail{},
		CronTasks:      []dto.AdminCronTaskDetail{},
		ArchiveDetails: []dto.AdminProcessDetail{},
	}
	latestModified := time.Time{}
	if cfg != nil {
		latestModified = cfg.UpdatedAt
	}

	// 填充成员信息
	if m, ok := memberMap[userID]; ok {
		item.MemberID = m.ID.String()
		item.Username = m.User.Username
		item.DisplayName = m.User.DisplayName
		item.Department = m.Department.Name
		for _, r := range m.Roles {
			item.RoleNames = append(item.RoleNames, r.Name)
		}
	} else {
		item.Username = userID.String()
		item.DisplayName = userID.String()
	}

	// 解析审核工作台详情（应用权限过滤，只保留有实际内容的流程）
	if cfg != nil {
		var auditDetails []model.AuditDetailItem
		if err := json.Unmarshal(cfg.AuditDetails, &auditDetails); err == nil {
			for _, d := range auditDetails {
				perms := auditPermsMap[d.ProcessType] // 未命中时零值（全 false → 全清空，与 settings 一致）
				detail := toAdminProcessDetail(d.ProcessType,
					applyStrictnessPerm(d.AIConfig.StrictnessOverride, perms.AllowModifyStrictness),
					applyCustomRulesPerm(d.RuleConfig.CustomRules, perms.AllowCustomRules),
					applyFieldOverridesPerm(d.FieldConfig.FieldOverrides, perms.AllowCustomFields),
					filterToggleOverrides(d.RuleConfig.RuleToggleOverrides, auditRuleMap),
					auditRuleMap,
					auditFieldsMap[d.ProcessType],
				)
				if hasAdminProcessContent(detail) {
					item.AuditDetails = append(item.AuditDetails, detail)
				}
			}
			item.AuditProcessCount = len(item.AuditDetails)
		}
	}

	// 解析真实定时任务实例
	for _, task := range cronTasks {
		item.CronTasks = append(item.CronTasks, toAdminCronTaskDetail(task))
		if task.UpdatedAt.After(latestModified) {
			latestModified = task.UpdatedAt
		}
	}
	item.CronTaskCount = len(item.CronTasks)

	// 解析归档复盘详情（访问控制 + 权限过滤，只保留有实际内容的流程）
	if cfg != nil {
		var archiveDetails []model.ArchiveDetailItem
		if err := json.Unmarshal(cfg.ArchiveDetails, &archiveDetails); err == nil {
			member := memberMap[userID]
			for _, d := range archiveDetails {
				// 若归档配置已不存在（active 状态），跳过
				if _, cfgExists := archivePermsMap[d.ProcessType]; !cfgExists {
					continue
				}
				// 访问控制：检查用户是否仍有权访问该流程
				if ac, ok := archiveAccessMap[d.ProcessType]; ok {
					if !userCanAccessArchive(ac, member) {
						continue
					}
				}
				perms := archivePermsMap[d.ProcessType]
				detail := toAdminProcessDetail(d.ProcessType,
					applyStrictnessPerm(d.AIConfig.StrictnessOverride, perms.AllowModifyStrictness),
					applyCustomRulesPerm(d.RuleConfig.CustomRules, perms.AllowCustomRules),
					applyFieldOverridesPerm(d.FieldConfig.FieldOverrides, perms.AllowCustomFields),
					filterToggleOverrides(d.RuleConfig.RuleToggleOverrides, archiveRuleMap),
					archiveRuleMap,
					archiveFieldsMap[d.ProcessType],
				)
				if hasAdminProcessContent(detail) {
					item.ArchiveDetails = append(item.ArchiveDetails, detail)
				}
			}
			item.ArchiveProcessCount = len(item.ArchiveDetails)
		}
	}

	if !latestModified.IsZero() {
		item.LastModified = latestModified.Format(time.RFC3339)
	}

	return item
}

func groupCronTasksByOwner(tasks []model.CronTask) map[uuid.UUID][]model.CronTask {
	grouped := make(map[uuid.UUID][]model.CronTask)
	for _, task := range tasks {
		grouped[task.OwnerUserID] = append(grouped[task.OwnerUserID], task)
	}
	return grouped
}

func toAdminCronTaskDetail(task model.CronTask) dto.AdminCronTaskDetail {
	workflowIDs := make([]string, 0)
	if len(task.WorkflowIds) > 0 {
		_ = json.Unmarshal(task.WorkflowIds, &workflowIDs)
	}
	dateRange := task.DateRange
	if dateRange <= 0 {
		dateRange = 30
	}
	return dto.AdminCronTaskDetail{
		ID:             task.ID.String(),
		TaskType:       task.TaskType,
		TaskLabel:      task.TaskLabel,
		Module:         cronTaskModule(task.TaskType),
		CronExpression: task.CronExpression,
		IsActive:       task.IsActive,
		IsBuiltin:      task.IsBuiltin,
		PushEmail:      task.PushEmail,
		WorkflowIDs:    workflowIDs,
		DateRange:      dateRange,
	}
}

func cronTaskModule(taskType string) string {
	if strings.HasPrefix(taskType, "audit_") {
		return "audit"
	}
	if strings.HasPrefix(taskType, "archive_") {
		return "archive"
	}
	return ""
}

// applyStrictnessPerm 若权限关闭则将 strictness 清空。
func applyStrictnessPerm(strictness string, allowed bool) string {
	if !allowed {
		return ""
	}
	return strictness
}

// applyCustomRulesPerm 若权限关闭则将自定义规则清空。
func applyCustomRulesPerm(rules []model.CustomRule, allowed bool) []model.CustomRule {
	if !allowed {
		return nil
	}
	return rules
}

// applyFieldOverridesPerm 若权限关闭则将字段覆盖清空。
func applyFieldOverridesPerm(fields []string, allowed bool) []string {
	if !allowed {
		return nil
	}
	return fields
}

// filterToggleOverrides 过滤规则开关覆盖列表：
// 1. 过滤掉已被管理员删除的规则（rule_id 不在 ruleMap 中）
// 2. 只保留用户实际修改过的（enabled 与管理员默认值不同）
// 注：rule_toggle_overrides 没有独立权限门控，用户始终可以切换租户通用规则的开关。
func filterToggleOverrides(toggles []model.RuleToggleOverride, ruleMap map[string]ruleInfo) []model.RuleToggleOverride {
	result := make([]model.RuleToggleOverride, 0, len(toggles))
	for _, t := range toggles {
		info, exists := ruleMap[t.RuleID]
		if !exists {
			continue // 规则已被管理员删除，跳过
		}
		if t.Enabled == info.AdminEnabled {
			continue // 与管理员默认值相同，未修改，跳过
		}
		result = append(result, t)
	}
	return result
}

// userCanAccessArchive 判断用户（OrgMember）是否有权访问指定归档配置。
// 逻辑与 user_personal_config_service.GetAccessibleArchiveConfigs 保持一致：
// 三列表均为空 → 对所有成员公开；member 为 nil（非租户成员）→ 无权访问。
func userCanAccessArchive(ac model.AccessControlData, member *model.OrgMember) bool {
	if len(ac.AllowedRoles) == 0 && len(ac.AllowedMembers) == 0 && len(ac.AllowedDepartments) == 0 {
		return true
	}
	if member == nil {
		return false
	}
	if sliceContainsLocal(ac.AllowedMembers, member.ID.String()) {
		return true
	}
	if sliceContainsLocal(ac.AllowedDepartments, member.DepartmentID.String()) {
		return true
	}
	for _, r := range member.Roles {
		if sliceContainsLocal(ac.AllowedRoles, r.ID.String()) {
			return true
		}
	}
	return false
}

func sliceContainsLocal(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

// hasAdminProcessContent 判断一个流程详情是否包含任何实际的用户自定义内容。
// 用于过滤保存时未做任何修改的流程条目。
func hasAdminProcessContent(d dto.AdminProcessDetail) bool {
	return d.StrictnessOverride != "" ||
		len(d.CustomRules) > 0 ||
		len(d.FieldOverrides) > 0 ||
		len(d.RuleToggleOverrides) > 0
}

// toAdminProcessDetail 将流程内部模型转换为管理员视图 DTO，ruleMap 用于填充规则内容和管理员默认状态。
func toAdminProcessDetail(
	processType, strictness string,
	customRules []model.CustomRule,
	fieldOverrides []string,
	toggles []model.RuleToggleOverride,
	ruleMap map[string]ruleInfo,
	fieldInfo processConfigInfo,
) dto.AdminProcessDetail {
	detail := dto.AdminProcessDetail{
		ProcessType:         processType,
		StrictnessOverride:  strictness,
		FieldOverrides:      []dto.AdminFieldOverrideItem{},
		CustomRules:         make([]dto.AdminCustomRule, len(customRules)),
		RuleToggleOverrides: make([]dto.AdminRuleToggleItem, len(toggles)),
	}

	// 规则对比
	for i, r := range customRules {
		detail.CustomRules[i] = dto.AdminCustomRule{ID: r.ID, Content: r.Content, Enabled: r.Enabled}
	}
	for i, t := range toggles {
		info := ruleMap[t.RuleID]
		detail.RuleToggleOverrides[i] = dto.AdminRuleToggleItem{
			RuleID:       t.RuleID,
			RuleContent:  info.Content,
			RuleScope:    info.RuleScope,
			AdminEnabled: info.AdminEnabled,
			Enabled:      t.Enabled,
		}
	}

	// 字段对比
	// 如果租户管理员已经选择了所有字段，则默认没有修改（不需要标记任何东西，除非用户有冗余字段）
	// 但通常 'all' 模式下，fieldOverrides 应该是空的。
	if fieldInfo.FieldMode == "all" && len(fieldOverrides) == 0 {
		return detail
	}

	// 解析租户可用字段，同时记录租户的选中状态
	tenantFieldMap := make(map[string]struct {
		Name     string
		Table    string
		Label    string
		Selected bool
	})
	for _, f := range fieldInfo.MainFields {
		tenantFieldMap["main:"+f.FieldKey] = struct {
			Name     string
			Table    string
			Label    string
			Selected bool
		}{Name: f.FieldName, Table: "main", Label: "主表", Selected: f.Selected}
	}
	for _, dt := range fieldInfo.DetailTables {
		for _, f := range dt.Fields {
			tenantFieldMap[dt.TableName+":"+f.FieldKey] = struct {
				Name     string
				Table    string
				Label    string
				Selected bool
			}{Name: f.FieldName, Table: dt.TableName, Label: dt.TableLabel, Selected: f.Selected}
		}
	}

	// 遍历用户覆盖字段，计算偏差
	for _, fo := range fieldOverrides {
		table, key := parseFieldOverride(fo)
		fullKey := table + ":" + key

		info, exists := tenantFieldMap[fullKey]
		if !exists {
			// 情况1: 租户元数据里没有这个字段了 -> 系统已废弃
			detail.FieldOverrides = append(detail.FieldOverrides, dto.AdminFieldOverrideItem{
				TableName:  table,
				TableLabel: table,
				FieldKey:   key,
				FieldName:  key,
				Status:     "abandoned",
			})
		} else {
			// 情况2: 租户元数据里有，但租户管理员没配置（未选中）
			// 如果租户是 'all' 模式，或者该字段本身就被租户选中了，则视为正常同步，不显示。
			isTenantSelected := fieldInfo.FieldMode == "all" || info.Selected
			if !isTenantSelected {
				detail.FieldOverrides = append(detail.FieldOverrides, dto.AdminFieldOverrideItem{
					TableName:  info.Table,
					TableLabel: info.Label,
					FieldKey:   key,
					FieldName:  info.Name,
					Status:     "user_added",
				})
			}
		}
	}

	return detail
}

func parseFieldOverride(fo string) (string, string) {
	if strings.Contains(fo, ":") {
		parts := strings.SplitN(fo, ":", 2)
		return parts[0], parts[1]
	}
	return "main", fo
}
