// 仪表盘概览处理器，负责聚合并返回仪表盘所需的统计数据。
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/pkg/errcode"
	jwtpkg "oa-smart-audit/go-service/internal/pkg/jwt"
	"oa-smart-audit/go-service/internal/pkg/response"
	"oa-smart-audit/go-service/internal/service"
)

// DashboardOverviewHandler 处理仪表盘概览相关的 HTTP 请求。
type DashboardOverviewHandler struct {
	svc *service.DashboardOverviewService
}

// NewDashboardOverviewHandler 创建仪表盘概览处理器实例。
func NewDashboardOverviewHandler(svc *service.DashboardOverviewService) *DashboardOverviewHandler {
	return &DashboardOverviewHandler{svc: svc}
}

// GetOverview 获取当前租户用户的仪表盘聚合数据。
// GET /api/tenant/settings/dashboard-overview
// 从 JWT claims 中提取用户 ID 和角色，按角色维度聚合统计数据。
// 返回：仪表盘各组件所需的统计数据对象。
func (h *DashboardOverviewHandler) GetOverview(c *gin.Context) {
	claimsVal, ok := c.Get("jwt_claims")
	if !ok {
		response.Error(c, http.StatusUnauthorized, 40100, "未提供认证令牌")
		return
	}
	claims, ok := claimsVal.(*jwtpkg.JWTClaims)
	if !ok {
		response.Error(c, http.StatusInternalServerError, 50000, "服务器内部错误")
		return
	}

	viewerID, err := uuid.Parse(claims.Sub)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "用户标识无效")
		return
	}
	data, err := h.svc.BuildOverview(c, claims.ActiveRole.Role, viewerID, claims.Username)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, data)
}

// GetPlatformOverview 获取全平台聚合的仪表盘数据（仅系统管理员可用）。
// GET /api/admin/dashboard-overview
// 不依赖租户上下文，聚合所有租户的统计数据。
// 返回：平台级仪表盘统计数据对象。
func (h *DashboardOverviewHandler) GetPlatformOverview(c *gin.Context) {
	data, err := h.svc.BuildPlatformOverview()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, data)
}
