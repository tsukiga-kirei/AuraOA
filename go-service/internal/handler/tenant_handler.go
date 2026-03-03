package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/response"
	"oa-smart-audit/go-service/internal/service"
)

//TenantHandler 处理 system_admin 的租户管理 HTTP 请求。
type TenantHandler struct {
	tenantService *service.TenantService
}

//NewTenantHandler 创建一个新的 TenantHandler 实例。
func NewTenantHandler(tenantService *service.TenantService) *TenantHandler {
	return &TenantHandler{tenantService: tenantService}
}

//ListTenants 处理 GET /api/admin/tenants
func (h *TenantHandler) ListTenants(c *gin.Context) {
	tenants, err := h.tenantService.ListTenants()
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, tenants)
}

//CreateTenant 处理 POST /api/admin/tenants
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

//UpdateTenant 处理 PUT /api/admin/tenants/:id
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

//DeleteTenant 处理 DELETE /api/admin/tenants/:id
func (h *TenantHandler) DeleteTenant(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	if err := h.tenantService.DeleteTenant(id); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, nil)
}

//GetTenantStats 处理 GET /api/admin/tenants/:id/stats
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
