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

// getUserID 从 JWT claims 中提取用户 ID。
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

// GetProcessList 处理 GET /api/tenant/settings/processes
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

// GetByProcessType 处理 GET /api/tenant/settings/processes/:processType
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

// UpdateByProcessType 处理 PUT /api/tenant/settings/processes/:processType
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

// GetDashboardPrefs 处理 GET /api/tenant/settings/dashboard-prefs
func (h *UserPersonalConfigHandler) GetDashboardPrefs(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
		return
	}
	tenantID, err := getTenantID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "租户ID无效")
		return
	}
	pref, err := h.dashPrefRepo.GetByTenantAndUser(c, tenantID, userID)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, errcode.ErrDatabase, "数据库错误")
		return
	}
	if pref == nil {
		// 返回默认空偏好
		pref = &model.UserDashboardPref{
			EnabledWidgets: datatypes.JSON([]byte("[]")),
			WidgetSizes:    datatypes.JSON([]byte("{}")),
		}
	}
	response.Success(c, pref)
}

// UpdateDashboardPrefs 处理 PUT /api/tenant/settings/dashboard-prefs
func (h *UserPersonalConfigHandler) UpdateDashboardPrefs(c *gin.Context) {
	userID, err := getUserID(c)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
		return
	}
	tenantID, err := getTenantID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "租户ID无效")
		return
	}
	var req dto.UpdateDashboardPrefRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}

	pref := &model.UserDashboardPref{
		ID:             uuid.New(),
		TenantID:       tenantID,
		UserID:         userID,
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

// GetFullProcessConfig 处理 GET /api/tenant/settings/processes/:processType/full
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

// GetCronPrefs 处理 GET /api/tenant/settings/cron-prefs
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

// UpdateCronPrefs 处理 PUT /api/tenant/settings/cron-prefs
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

// GetArchiveConfigList 处理 GET /api/tenant/settings/archive-configs
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

// GetFullArchiveConfig 处理 GET /api/tenant/settings/archive-configs/:processType/full
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

// UpdateArchiveConfig 处理 PUT /api/tenant/settings/archive-configs/:processType
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

// defaultDashJSON 返回 JSON 值，如果为 nil 则返回默认值。
func defaultDashJSON(val datatypes.JSON, defaultVal string) datatypes.JSON {
	if val == nil {
		return datatypes.JSON([]byte(defaultVal))
	}
	return val
}
