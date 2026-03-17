package handler

import (
	"encoding/json"
	"net/http"
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
	userConfigRepo       *repository.UserPersonalConfigRepo
	orgRepo              *repository.OrgRepo
	auditRuleRepo        *repository.AuditRuleRepo
	archiveRuleRepo      *repository.ArchiveRuleRepo
	auditConfigRepo      *repository.ProcessAuditConfigRepo
	archiveConfigRepo    *repository.ProcessArchiveConfigRepo
}

// NewUserConfigManagementHandler 创建一个新的 UserConfigManagementHandler 实例。
func NewUserConfigManagementHandler(
	userConfigRepo *repository.UserPersonalConfigRepo,
	orgRepo *repository.OrgRepo,
	auditRuleRepo *repository.AuditRuleRepo,
	archiveRuleRepo *repository.ArchiveRuleRepo,
	auditConfigRepo *repository.ProcessAuditConfigRepo,
	archiveConfigRepo *repository.ProcessArchiveConfigRepo,
) *UserConfigManagementHandler {
	return &UserConfigManagementHandler{
		userConfigRepo:    userConfigRepo,
		orgRepo:           orgRepo,
		auditRuleRepo:     auditRuleRepo,
		archiveRuleRepo:   archiveRuleRepo,
		auditConfigRepo:   auditConfigRepo,
		archiveConfigRepo: archiveConfigRepo,
	}
}

// ListUserConfigs 处理 GET /api/tenant/user-configs
// 返回当前租户内所有有个人配置记录的用户，附带成员信息和配置摘要。
func (h *UserConfigManagementHandler) ListUserConfigs(c *gin.Context) {
	configs, err := h.userConfigRepo.ListByTenant(c)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, errcode.ErrDatabase, "数据库错误")
		return
	}

	memberMap, auditRuleMap, archiveRuleMap, auditPermsMap, archivePermsMap, archiveAccessMap := h.loadSharedMaps(c)

	result := make([]dto.AdminUserConfigListItem, 0, len(configs))
	for _, cfg := range configs {
		item := buildAdminUserConfigItem(cfg, memberMap, auditRuleMap, archiveRuleMap, auditPermsMap, archivePermsMap, archiveAccessMap)
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
	if cfg == nil {
		response.Error(c, http.StatusNotFound, errcode.ErrResourceNotFound, "用户配置不存在")
		return
	}

	memberMap, auditRuleMap, archiveRuleMap, auditPermsMap, archivePermsMap, archiveAccessMap := h.loadSharedMaps(c)
	item := buildAdminUserConfigItem(*cfg, memberMap, auditRuleMap, archiveRuleMap, auditPermsMap, archivePermsMap, archiveAccessMap)
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
	memberMap map[uuid.UUID]*model.OrgMember,
	auditRuleMap map[string]ruleInfo,
	archiveRuleMap map[string]ruleInfo,
	auditPermsMap map[string]model.UserPermissionsData,
	archivePermsMap map[string]model.ArchiveUserPermissionsData,
	archiveAccessMap map[string]model.AccessControlData,
) {
	memberMap = make(map[uuid.UUID]*model.OrgMember)
	if members, err := h.orgRepo.ListMembers(c); err == nil {
		for i := range members {
			memberMap[members[i].UserID] = &members[i]
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

	// 审核工作台权限映射：processType → UserPermissionsData
	auditPermsMap = make(map[string]model.UserPermissionsData)
	if cfgs, err := h.auditConfigRepo.ListByTenant(c); err == nil {
		for _, cfg := range cfgs {
			var perms model.UserPermissionsData
			if err2 := json.Unmarshal(cfg.UserPermissions, &perms); err2 == nil {
				auditPermsMap[cfg.ProcessType] = perms
			}
		}
	}

	// 归档复盘权限映射 + 访问控制映射：一次查询同时构建两个 map
	archivePermsMap = make(map[string]model.ArchiveUserPermissionsData)
	archiveAccessMap = make(map[string]model.AccessControlData)
	if cfgs, err := h.archiveConfigRepo.ListByTenant(c); err == nil {
		for _, cfg := range cfgs {
			if cfg.Status != "active" {
				continue // 已下线的配置视为不存在
			}
			var perms model.ArchiveUserPermissionsData
			if err2 := json.Unmarshal(cfg.UserPermissions, &perms); err2 == nil {
				archivePermsMap[cfg.ProcessType] = perms
			}
			var ac model.AccessControlData
			if err2 := json.Unmarshal(cfg.AccessControl, &ac); err2 == nil {
				archiveAccessMap[cfg.ProcessType] = ac
			}
		}
	}
	return
}

// buildAdminUserConfigItem 将原始 UserPersonalConfig 富化为管理员视图 DTO。
// 以管理员权限配置为基准：被禁用的特性对应的用户数据视为空；已删除的规则从 toggle 列表中清除。
// 归档复盘流程额外进行访问控制检查，用户无权访问的流程直接跳过。
func buildAdminUserConfigItem(
	cfg model.UserPersonalConfig,
	memberMap map[uuid.UUID]*model.OrgMember,
	auditRuleMap map[string]ruleInfo,
	archiveRuleMap map[string]ruleInfo,
	auditPermsMap map[string]model.UserPermissionsData,
	archivePermsMap map[string]model.ArchiveUserPermissionsData,
	archiveAccessMap map[string]model.AccessControlData,
) dto.AdminUserConfigListItem {
	item := dto.AdminUserConfigListItem{
		UserID:         cfg.UserID.String(),
		LastModified:   cfg.UpdatedAt.Format(time.RFC3339),
		RoleNames:      []string{},
		AuditDetails:   []dto.AdminProcessDetail{},
		ArchiveDetails: []dto.AdminProcessDetail{},
	}

	// 填充成员信息
	if m, ok := memberMap[cfg.UserID]; ok {
		item.MemberID = m.ID.String()
		item.Username = m.User.Username
		item.DisplayName = m.User.DisplayName
		item.Department = m.Department.Name
		for _, r := range m.Roles {
			item.RoleNames = append(item.RoleNames, r.Name)
		}
	} else {
		item.DisplayName = cfg.UserID.String()
	}

	// 解析审核工作台详情（应用权限过滤，只保留有实际内容的流程）
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
			)
			if hasAdminProcessContent(detail) {
				item.AuditDetails = append(item.AuditDetails, detail)
			}
		}
		item.AuditProcessCount = len(item.AuditDetails)
	}

	// 解析定时任务偏好（邮箱数量）
	var cronDetail model.CronDetailItem
	if err := json.Unmarshal(cfg.CronDetails, &cronDetail); err == nil && cronDetail.DefaultEmail != "" {
		emails := strings.Split(cronDetail.DefaultEmail, ",")
		count := 0
		for _, e := range emails {
			if strings.TrimSpace(e) != "" {
				count++
			}
		}
		item.CronDetails = dto.AdminCronDetail{DefaultEmail: cronDetail.DefaultEmail, EmailCount: count}
		item.CronEmailCount = count
	}

	// 解析归档复盘详情（访问控制 + 权限过滤，只保留有实际内容的流程）
	var archiveDetails []model.ArchiveDetailItem
	if err := json.Unmarshal(cfg.ArchiveDetails, &archiveDetails); err == nil {
		member := memberMap[cfg.UserID]
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
			)
			if hasAdminProcessContent(detail) {
				item.ArchiveDetails = append(item.ArchiveDetails, detail)
			}
		}
		item.ArchiveProcessCount = len(item.ArchiveDetails)
	}

	return item
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
) dto.AdminProcessDetail {
	detail := dto.AdminProcessDetail{
		ProcessType:         processType,
		StrictnessOverride:  strictness,
		FieldOverrides:      fieldOverrides,
		CustomRules:         make([]dto.AdminCustomRule, len(customRules)),
		RuleToggleOverrides: make([]dto.AdminRuleToggleItem, len(toggles)),
	}
	if detail.FieldOverrides == nil {
		detail.FieldOverrides = []string{}
	}
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
	return detail
}
