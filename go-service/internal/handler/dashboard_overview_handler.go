package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	jwtpkg "oa-smart-audit/go-service/internal/pkg/jwt"
	"oa-smart-audit/go-service/internal/pkg/response"
	"oa-smart-audit/go-service/internal/service"
)

// DashboardOverviewHandler GET /api/tenant/settings/dashboard-overview
type DashboardOverviewHandler struct {
	svc *service.DashboardOverviewService
}

// NewDashboardOverviewHandler 创建处理器。
func NewDashboardOverviewHandler(svc *service.DashboardOverviewService) *DashboardOverviewHandler {
	return &DashboardOverviewHandler{svc: svc}
}

// GetOverview 聚合仪表盘数据。
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

	data, err := h.svc.BuildOverview(c, claims.ActiveRole.Role)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, data)
}

// GetPlatformOverview GET /api/admin/dashboard-overview（system_admin，全平台聚合，不依赖 tenant_id）。
func (h *DashboardOverviewHandler) GetPlatformOverview(c *gin.Context) {
	data, err := h.svc.BuildPlatformOverview()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, data)
}
