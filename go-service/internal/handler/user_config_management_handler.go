package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/response"
	"oa-smart-audit/go-service/internal/repository"
)

// UserConfigManagementHandler 处理租户管理端的用户配置管理 HTTP 请求。
type UserConfigManagementHandler struct {
	userConfigRepo *repository.UserPersonalConfigRepo
}

// NewUserConfigManagementHandler 创建一个新的 UserConfigManagementHandler 实例。
func NewUserConfigManagementHandler(userConfigRepo *repository.UserPersonalConfigRepo) *UserConfigManagementHandler {
	return &UserConfigManagementHandler{userConfigRepo: userConfigRepo}
}

// ListUserConfigs 处理 GET /api/tenant/user-configs
func (h *UserConfigManagementHandler) ListUserConfigs(c *gin.Context) {
	configs, err := h.userConfigRepo.ListByTenant(c)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, errcode.ErrDatabase, "数据库错误")
		return
	}
	response.Success(c, configs)
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
	response.Success(c, cfg)
}
