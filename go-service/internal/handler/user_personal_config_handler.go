package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	jwtpkg "oa-smart-audit/go-service/internal/pkg/jwt"
	"oa-smart-audit/go-service/internal/pkg/response"
	"oa-smart-audit/go-service/internal/repository"
	"oa-smart-audit/go-service/internal/service"
)

// UserPersonalConfigHandler 处理用户个人配置相关的 HTTP 请求。
type UserPersonalConfigHandler struct {
	userConfigService *service.UserPersonalConfigService
	dashPrefRepo      *repository.UserDashboardPrefRepo
}

// NewUserPersonalConfigHandler 创建一个新的 UserPersonalConfigHandler 实例。
func NewUserPersonalConfigHandler(
	userConfigService *service.UserPersonalConfigService,
	dashPrefRepo *repository.UserDashboardPrefRepo,
) *UserPersonalConfigHandler {
	return &UserPersonalConfigHandler{
		userConfigService: userConfigService,
		dashPrefRepo:      dashPrefRepo,
	}
}

// getUserID 从 gin.Context 的 JWT claims 中提取当前登录用户的 UUID。
func getUserID(c *gin.Context) (uuid.UUID, error) {
	claimsVal, exists := c.Get("jwt_claims")
	if !exists {
		return uuid.Nil, &service.ServiceError{Code: errcode.ErrNoAuthToken, Message: "未提供认证令牌"}
	}
	claims, ok := claimsVal.(*jwtpkg.JWTClaims)
	if !ok {
		return uuid.Nil, &service.ServiceError{Code: errcode.ErrInternalServer, Message: "服务器内部错误"}
	}
	return uuid.Parse(claims.Sub)
}

// dashboardPrefsTenantScope 确定仪表盘偏好的租户维度：
// system_admin 使用平台维度（数据库 tenant_id 为 NULL），其他角色必须携带租户上下文。
func dashboardPrefsTenantScope(c *gin.Context) (*uuid.UUID, error) {
	claimsVal, ok := c.Get("jwt_claims")
	if !ok {
		return nil, errTenantIDMissing
	}
	claims, ok := claimsVal.(*jwtpkg.JWTClaims)
	if !ok {
		return nil, errTenantIDMissing
	}
	if claims.ActiveRole.Role == "system_admin" {
		return nil, nil
	}
	tid, err := getTenantID(c)
	if err != nil {
		return nil, err
	}
	return &tid, nil
}

// dashboardPrefScope 根据当前 JWT active_role 返回仪表盘偏好的作用域标识，
// 用于分角色存储布局，避免 business 和 tenant_admin 互相覆盖。
func dashboardPrefScope(c *gin.Context) (string, error) {
	claimsVal, ok := c.Get("jwt_claims")
	if !ok {
		return "", errTenantIDMissing
	}
	claims, ok := claimsVal.(*jwtpkg.JWTClaims)
	if !ok {
		return "", errTenantIDMissing
	}
	switch claims.ActiveRole.Role {
	case "system_admin":
		return model.DashboardPrefScopePlatform, nil
	case "tenant_admin":
		return model.DashboardPrefScopeTenantAdmin, nil
	case "business":
		return model.DashboardPrefScopeBusiness, nil
	default:
		return model.DashboardPrefScopeBusiness, nil
	}
}

// GetProcessList 获取当前用户可访问的审核流程配置列表。
// GET /api/tenant/settings/processes
// 返回：流程配置摘要数组（流程类型、名称、启用状态等）。
func (h *UserPersonalConfigHandler) GetProcessList(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
		return
	}
	list, err := h.userConfigService.GetProcessList(c, userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, list)
}

// GetByProcessType 获取当前用户指定流程类型的个人配置详情。
// GET /api/tenant/settings/processes/:processType
// 路径参数：processType（流程类型标识）
// 返回：该流程的个人配置对象（AI 严格度、规则开关、字段覆盖等）。
func (h *UserPersonalConfigHandler) GetByProcessType(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
		return
	}
	processType := c.Param("processType")
	if processType == "" {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	detail, err := h.userConfigService.GetByProcessType(c, userID, processType)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, detail)
}

// UpdateByProcessType 更新当前用户指定流程类型的个人配置。
// PUT /api/tenant/settings/processes/:processType
// 路径参数：processType（流程类型标识）
// 请求体：UpdateUserProcessConfigRequest
// 返回：null（成功时）。
func (h *UserPersonalConfigHandler) UpdateByProcessType(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
		return
	}
	processType := c.Param("processType")
	if processType == "" {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	var req dto.UpdateUserProcessConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	if err := h.userConfigService.UpdateByProcessType(c, userID, processType, &req); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, nil)
}

// GetDashboardPrefs 获取当前用户在当前角色下的仪表盘布局偏好。
// GET /api/tenant/settings/dashboard-prefs
// 返回：仪表盘偏好对象（启用的组件列表、组件尺寸配置等）。
func (h *UserPersonalConfigHandler) GetDashboardPrefs(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
		return
	}
	tenantScope, err := dashboardPrefsTenantScope(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "租户ID无效")
		return
	}
	scope, err := dashboardPrefScope(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	pref, err := h.dashPrefRepo.GetPref(tenantScope, userID, scope)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, errcode.ErrDatabase, "数据库错误")
		return
	}
	if pref == nil {
		pref = &model.UserDashboardPref{
			PrefScope:      scope,
			EnabledWidgets: datatypes.JSON([]byte("[]")),
			WidgetSizes:    datatypes.JSON([]byte("{}")),
		}
	}
	response.Success(c, pref)
}

// UpdateDashboardPrefs 更新当前用户在当前角色下的仪表盘布局偏好。
// PUT /api/tenant/settings/dashboard-prefs
// 请求体：UpdateDashboardPrefRequest（enabled_widgets、widget_sizes）
// 返回：null（成功时）。
func (h *UserPersonalConfigHandler) UpdateDashboardPrefs(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
		return
	}
	tenantScope, err := dashboardPrefsTenantScope(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "租户ID无效")
		return
	}
	scope, err := dashboardPrefScope(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	var req dto.UpdateDashboardPrefRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}

	pref := &model.UserDashboardPref{
		ID:             uuid.New(),
		TenantID:       tenantScope,
		UserID:         userID,
		PrefScope:      scope,
		EnabledWidgets: defaultDashJSON(req.EnabledWidgets, "[]"),
		WidgetSizes:    defaultDashJSON(req.WidgetSizes, "{}"),
		UpdatedAt:      time.Now(),
	}

	if err := h.dashPrefRepo.Upsert(pref); err != nil {
		response.Error(c, http.StatusInternalServerError, errcode.ErrDatabase, "数据库错误")
		return
	}
	response.Success(c, nil)
}

// GetFullProcessConfig 获取当前用户指定流程类型的完整审核配置（含租户规则、字段等合并结果）。
// GET /api/tenant/settings/processes/:processType/full
// 路径参数：processType（流程类型标识）
// 返回：合并后的完整流程配置对象，供审核工作台使用。
func (h *UserPersonalConfigHandler) GetFullProcessConfig(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
		return
	}
	processType := c.Param("processType")
	if processType == "" {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	result, err := h.userConfigService.GetFullAuditProcessConfig(c, userID, processType)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, result)
}

// GetCronPrefs 获取当前用户的定时任务偏好配置。
// GET /api/tenant/settings/cron-prefs
// 返回：定时任务偏好对象（邮件推送设置等）。
func (h *UserPersonalConfigHandler) GetCronPrefs(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
		return
	}
	prefs, err := h.userConfigService.GetCronPrefs(c, userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, prefs)
}

// UpdateCronPrefs 更新当前用户的定时任务偏好配置。
// PUT /api/tenant/settings/cron-prefs
// 请求体：UpdateCronPrefsRequest
// 返回：null（成功时）。
func (h *UserPersonalConfigHandler) UpdateCronPrefs(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
		return
	}
	var req dto.UpdateCronPrefsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	if err := h.userConfigService.UpdateCronPrefs(c, userID, &req); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, nil)
}

// GetArchiveConfigList 获取当前用户有权访问的归档复盘配置列表（经访问控制过滤）。
// GET /api/tenant/settings/archive-configs
// 返回：可访问的归档配置摘要数组。
func (h *UserPersonalConfigHandler) GetArchiveConfigList(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
		return
	}
	list, err := h.userConfigService.GetAccessibleArchiveConfigs(c, userID)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, list)
}

// GetFullArchiveConfig 获取当前用户指定归档流程类型的完整配置（含规则、字段等合并结果）。
// GET /api/tenant/settings/archive-configs/:processType/full
// 路径参数：processType（归档流程类型标识）
// 返回：合并后的完整归档配置对象，供归档复盘工作台使用。
func (h *UserPersonalConfigHandler) GetFullArchiveConfig(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
		return
	}
	processType := c.Param("processType")
	if processType == "" {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	result, err := h.userConfigService.GetFullArchiveConfig(c, userID, processType)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, result)
}

// UpdateArchiveConfig 更新当前用户指定归档流程类型的个人配置。
// PUT /api/tenant/settings/archive-configs/:processType
// 路径参数：processType（归档流程类型标识）
// 请求体：UpdateArchiveConfigRequest
// 返回：null（成功时）。
func (h *UserPersonalConfigHandler) UpdateArchiveConfig(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
		return
	}
	processType := c.Param("processType")
	if processType == "" {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	var req dto.UpdateArchiveConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	if err := h.userConfigService.UpdateArchiveConfig(c, userID, processType, &req); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, nil)
}

// defaultDashJSON 若 val 为 nil 则返回 defaultVal 对应的 JSON，否则返回原值。
func defaultDashJSON(val datatypes.JSON, defaultVal string) datatypes.JSON {
	if val == nil {
		return datatypes.JSON([]byte(defaultVal))
	}
	return val
}
