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

// OrgHandler handles department, role, and member CRUD HTTP requests.
type OrgHandler struct {
	orgService *service.OrgService
}

// NewOrgHandler creates a new OrgHandler instance.
func NewOrgHandler(orgService *service.OrgService) *OrgHandler {
	return &OrgHandler{orgService: orgService}
}

// getTenantID extracts and parses tenant_id from the gin.Context (set by TenantMiddleware).
func getTenantID(c *gin.Context) (uuid.UUID, error) {
	tidVal, exists := c.Get("tenant_id")
	if !exists {
		return uuid.Nil, errTenantIDMissing
	}
	tidStr, ok := tidVal.(string)
	if !ok {
		return uuid.Nil, errTenantIDMissing
	}
	return uuid.Parse(tidStr)
}

var errTenantIDMissing = &service.ServiceError{Code: errcode.ErrParamValidation, Message: "租户ID缺失"}

// ---------------------------------------------------------------------------
// Department handlers
// ---------------------------------------------------------------------------

// ListDepartments handles GET /api/tenant/org/departments
func (h *OrgHandler) ListDepartments(c *gin.Context) {
	departments, err := h.orgService.ListDepartments(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, departments)
}

// CreateDepartment handles POST /api/tenant/org/departments
func (h *OrgHandler) CreateDepartment(c *gin.Context) {
	var req dto.CreateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	tenantID, err := getTenantID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "租户ID无效")
		return
	}
	dept, err := h.orgService.CreateDepartment(c, tenantID, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, dept)
}

// UpdateDepartment handles PUT /api/tenant/org/departments/:id
func (h *OrgHandler) UpdateDepartment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	var req dto.UpdateDepartmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	dept, err := h.orgService.UpdateDepartment(c, id, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, dept)
}

// DeleteDepartment handles DELETE /api/tenant/org/departments/:id
func (h *OrgHandler) DeleteDepartment(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	if err := h.orgService.DeleteDepartment(c, id); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, nil)
}

// ---------------------------------------------------------------------------
// Role handlers
// ---------------------------------------------------------------------------

// ListRoles handles GET /api/tenant/org/roles
func (h *OrgHandler) ListRoles(c *gin.Context) {
	roles, err := h.orgService.ListRoles(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, roles)
}

// CreateRole handles POST /api/tenant/org/roles
func (h *OrgHandler) CreateRole(c *gin.Context) {
	var req dto.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	tenantID, err := getTenantID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "租户ID无效")
		return
	}
	role, err := h.orgService.CreateRole(c, tenantID, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, role)
}

// UpdateRole handles PUT /api/tenant/org/roles/:id
func (h *OrgHandler) UpdateRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	var req dto.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	role, err := h.orgService.UpdateRole(c, id, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, role)
}

// DeleteRole handles DELETE /api/tenant/org/roles/:id
func (h *OrgHandler) DeleteRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	if err := h.orgService.DeleteRole(c, id); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, nil)
}

// ---------------------------------------------------------------------------
// Member handlers
// ---------------------------------------------------------------------------

// ListMembers handles GET /api/tenant/org/members
func (h *OrgHandler) ListMembers(c *gin.Context) {
	members, err := h.orgService.ListMembers(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, members)
}

// CreateMember handles POST /api/tenant/org/members
func (h *OrgHandler) CreateMember(c *gin.Context) {
	var req dto.CreateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	tenantID, err := getTenantID(c)
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "租户ID无效")
		return
	}
	member, err := h.orgService.CreateMember(c, tenantID, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, member)
}

// UpdateMember handles PUT /api/tenant/org/members/:id
func (h *OrgHandler) UpdateMember(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	var req dto.UpdateMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	member, err := h.orgService.UpdateMember(c, id, &req)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, member)
}

// DeleteMember handles DELETE /api/tenant/org/members/:id
func (h *OrgHandler) DeleteMember(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.Error(c, http.StatusBadRequest, errcode.ErrParamValidation, "参数校验失败")
		return
	}
	if err := h.orgService.DeleteMember(c, id); err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, nil)
}

// ---------------------------------------------------------------------------
// Helper: map ServiceError to HTTP response
// ---------------------------------------------------------------------------

func handleServiceError(c *gin.Context, err error) {
	httpStatus := mapServiceErrorToHTTP(err)
	if svcErr, ok := err.(*service.ServiceError); ok {
		response.Error(c, httpStatus, svcErr.Code, svcErr.Message)
		return
	}
	response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "服务器内部错误")
}
