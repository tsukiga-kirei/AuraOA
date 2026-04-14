// 租户管理处理器，负责租户的增删改查及成员统计查询。
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

// TenantHandler 处理租户管理相关的 HTTP 请求。
type TenantHandler struct {
	tenantService *service.TenantService
}

// NewTenantHandler 创建租户管理处理器实例。
func NewTenantHandler(tenantService *service.TenantService) *TenantHandler {
	return &TenantHandler{
		tenantService: tenantService,
	}
}

// ListPublicTenants 获取公开的租户列表（登录页选择租户使用，无需鉴权）。
// GET /api/tenants/list
// 返回：租户简要信息数组（code + name）。
func (h *TenantHandler) ListPublicTenants(c *gin.Context) {
	tenants, err := h.tenantService.ListPublicTenants()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, tenants)
}

// ListTenants 获取所有租户的完整列表（仅系统管理员可用）。
// GET /api/admin/tenants
// 返回：租户列表数组，包含状态、成员数等信息。
func (h *TenantHandler) ListTenants(c *gin.Context) {
	tenants, err := h.tenantService.ListTenants()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, tenants)
}

// CreateTenant 创建新租户（仅系统管理员可用）。
// POST /api/admin/tenants
// 请求体：CreateTenantRequest（租户名称、code、初始管理员信息等）
// 返回：新建的租户对象。
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

// UpdateTenant 更新指定租户信息（仅系统管理员可用）。
// PUT /api/admin/tenants/:id
// 路径参数：id（UUID 格式）
// 请求体：UpdateTenantRequest
// 返回：更新后的租户对象。
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

// DeleteTenant 删除指定租户（仅系统管理员可用，需提供管理员密码确认）。
// DELETE /api/admin/tenants/:id
// 路径参数：id（UUID 格式）
// 请求体：DeleteTenantRequest（admin_password）
// 返回：null（成功时）。
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

	// 从 JWT 中提取当前操作者的用户 ID，用于密码验证
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

// GetTenantStats 获取指定租户的统计数据（成员数、审核数等）。
// GET /api/admin/tenants/:id/stats
// 路径参数：id（UUID 格式）
// 返回：租户统计数据对象。
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

// ListTenantMembers 获取指定租户的成员列表。
// GET /api/admin/tenants/:id/members
// 路径参数：id（UUID 格式）
// 返回：成员列表数组（含用户名、部门、角色等信息）。
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
