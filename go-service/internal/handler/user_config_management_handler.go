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
	userConfigRepo  *repository.UserPersonalConfigRepo
	orgRepo         *repository.OrgRepo
	auditRuleRepo   *repository.AuditRuleRepo
	archiveRuleRepo *repository.ArchiveRuleRepo
}

// NewUserConfigManagementHandler 创建一个新的 UserConfigManagementHandler 实例。
func NewUserConfigManagementHandler(
	userConfigRepo *repository.UserPersonalConfigRepo,
	orgRepo *repository.OrgRepo,
	auditRuleRepo *repository.AuditRuleRepo,
	archiveRuleRepo *repository.ArchiveRuleRepo,
) *UserConfigManagementHandler {
	return &UserConfigManagementHandler{
		userConfigRepo:  userConfigRepo,
		orgRepo:         orgRepo,
		auditRuleRepo:   auditRuleRepo,
		archiveRuleRepo: archiveRuleRepo,
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

	memberMap, auditRuleMap, archiveRuleMap := h.loadSharedMaps(c)

	result := make([]dto.AdminUserConfigListItem, 0, len(configs))
	for _, cfg := range configs {
		item := buildAdminUserConfigItem(cfg, memberMap, auditRuleMap, archiveRuleMap)
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

	memberMap, auditRuleMap, archiveRuleMap := h.loadSharedMaps(c)
	item := buildAdminUserConfigItem(*cfg, memberMap, auditRuleMap, archiveRuleMap)
	response.Success(c, item)
}

// loadSharedMaps 批量加载成员映射和规则内容映射，供各接口共用。
func (h *UserConfigManagementHandler) loadSharedMaps(c *gin.Context) (
	memberMap map[uuid.UUID]*model.OrgMember,
	auditRuleMap map[string]string,
	archiveRuleMap map[string]string,
) {
	memberMap = make(map[uuid.UUID]*model.OrgMember)
	if members, err := h.orgRepo.ListMembers(c); err == nil {
		for i := range members {
			memberMap[members[i].UserID] = &members[i]
		}
	}

	auditRuleMap = make(map[string]string)
	if rules, err := h.auditRuleRepo.ListByTenant(c); err == nil {
		for _, r := range rules {
			auditRuleMap[r.ID.String()] = r.RuleContent
		}
	}

	archiveRuleMap = make(map[string]string)
	if rules, err := h.archiveRuleRepo.ListByTenant(c); err == nil {
		for _, r := range rules {
			archiveRuleMap[r.ID.String()] = r.RuleContent
		}
	}
	return
}

// buildAdminUserConfigItem 将原始 UserPersonalConfig 富化为管理员视图 DTO。
func buildAdminUserConfigItem(
	cfg model.UserPersonalConfig,
	memberMap map[uuid.UUID]*model.OrgMember,
	auditRuleMap map[string]string,
	archiveRuleMap map[string]string,
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

	// 解析审核工作台详情
	var auditDetails []model.AuditDetailItem
	if err := json.Unmarshal(cfg.AuditDetails, &auditDetails); err == nil {
		item.AuditProcessCount = len(auditDetails)
		for _, d := range auditDetails {
			detail := toAdminProcessDetail(d.ProcessType, d.AIConfig.StrictnessOverride,
				d.RuleConfig.CustomRules, d.FieldConfig.FieldOverrides, d.RuleConfig.RuleToggleOverrides, auditRuleMap)
			item.AuditDetails = append(item.AuditDetails, detail)
		}
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

	// 解析归档复盘详情
	var archiveDetails []model.ArchiveDetailItem
	if err := json.Unmarshal(cfg.ArchiveDetails, &archiveDetails); err == nil {
		item.ArchiveProcessCount = len(archiveDetails)
		for _, d := range archiveDetails {
			detail := toAdminProcessDetail(d.ProcessType, d.AIConfig.StrictnessOverride,
				d.RuleConfig.CustomRules, d.FieldConfig.FieldOverrides, d.RuleConfig.RuleToggleOverrides, archiveRuleMap)
			item.ArchiveDetails = append(item.ArchiveDetails, detail)
		}
	}

	return item
}

// toAdminProcessDetail 将流程内部模型转换为管理员视图 DTO，ruleContentMap 用于填充规则内容。
func toAdminProcessDetail(
	processType, strictness string,
	customRules []model.CustomRule,
	fieldOverrides []string,
	toggles []model.RuleToggleOverride,
	ruleContentMap map[string]string,
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
		content := ruleContentMap[t.RuleID]
		detail.RuleToggleOverrides[i] = dto.AdminRuleToggleItem{RuleID: t.RuleID, RuleContent: content, Enabled: t.Enabled}
	}
	return detail
}
