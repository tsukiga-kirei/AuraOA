package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	jwtpkg "oa-smart-audit/go-service/internal/pkg/jwt"
	"oa-smart-audit/go-service/internal/pkg/response"
	"oa-smart-audit/go-service/internal/service"
)

// TenantHandler handles tenant management HTTP requests.
type TenantHandler struct {
	tenantService *service.TenantService
}

// NewTenantHandler creates a new TenantHandler instance.
func NewTenantHandler(tenantService *service.TenantService) *TenantHandler {
	return &TenantHandler{
		tenantService: tenantService,
	}
}

// ListPublicTenants handles GET /api/tenants/list (public, no auth)
func (h *TenantHandler) ListPublicTenants(c *gin.Context) {
	tenants, err := h.tenantService.ListPublicTenants()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, tenants)
}

// ListTenants handles GET /api/admin/tenants
func (h *TenantHandler) ListTenants(c *gin.Context) {
	tenants, err := h.tenantService.ListTenants()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, tenants)
}

// CreateTenant handles POST /api/admin/tenants
func (h *TenantHandler) CreateTenant(c *gin.Context) {
	var req dto.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	tenant, err := h.tenantService.CreateTenant(&req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, tenant)
}

// UpdateTenant handles PUT /api/admin/tenants/:id
func (h *TenantHandler) UpdateTenant(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	var req dto.UpdateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	tenant, err := h.tenantService.UpdateTenant(id, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, tenant)
}

// DeleteTenant handles DELETE /api/admin/tenants/:id
// 需要请求体中提供管理员密码确认。
func (h *TenantHandler) DeleteTenant(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}

	var req dto.DeleteTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "请提供管理员密码")
		return
	}

	// 从 JWT 中提取当前操作者的用户 ID
	claimsVal, exists := c.Get("jwt_claims")
	if !exists {
		response.Error(c, http.StatusUnauthorized, errcode.ErrNoAuthToken, "未提供认证令牌")
		return
	}
	claims, ok := claimsVal.(*jwtpkg.JWTClaims)
	if !ok {
		response.Error(c, http.StatusUnauthorized, errcode.ErrTokenInvalid, "令牌解析失败")
		return
	}
	operatorID, err := uuid.Parse(claims.Sub)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, errcode.ErrTokenInvalid, "令牌用户ID无效")
		return
	}

	if err := h.tenantService.DeleteTenant(id, operatorID, req.AdminPassword); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, nil)
}

// GetTenantStats handles GET /api/admin/tenants/:id/stats
func (h *TenantHandler) GetTenantStats(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	stats, err := h.tenantService.GetTenantStats(id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, stats)
}

// ListTenantMembers handles GET /api/admin/tenants/:id/members
func (h *TenantHandler) ListTenantMembers(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	members, err := h.tenantService.ListTenantMembers(id)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, members)
}
