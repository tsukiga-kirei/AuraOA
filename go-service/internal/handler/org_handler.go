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

//OrgHandler 处理部门、角色和成员 CRUD HTTP 请求。
type OrgHandler struct {
	orgService *service.OrgService
}

//NewOrgHandler 创建一个新的 OrgHandler 实例。
func NewOrgHandler(orgService *service.OrgService) *OrgHandler {
	return &OrgHandler{orgService: orgService}
}

//getTenantID 从 gin.Context（由 TenantMiddleware 设置）中提取并解析tenant_id。
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
//部门经理
// ---------------------------------------------------------------------------

//ListDepartments 处理 GET /api/tenant/org/departments
func (h *OrgHandler) ListDepartments(c *gin.Context) {
	departments, err := h.orgService.ListDepartments(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, departments)
}

//CreateDepartment 处理 POST /api/tenant/org/departments
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

//UpdateDepartment 处理 PUT /api/tenant/org/departments/:id
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

//DeleteDepartment 处理 DELETE /api/tenant/org/departments/:id
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
//角色处理程序
// ---------------------------------------------------------------------------

//ListRoles 处理 GET /api/tenant/org/roles
func (h *OrgHandler) ListRoles(c *gin.Context) {
	roles, err := h.orgService.ListRoles(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, roles)
}

//CreateRole 处理 POST /api/tenant/org/roles
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

//UpdateRole 处理 PUT /api/tenant/org/roles/:id
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

//DeleteRole 处理 DELETE /api/tenant/org/roles/:id
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
//会员管理员
// ---------------------------------------------------------------------------

//ListMembers 处理 GET /api/tenant/org/members
func (h *OrgHandler) ListMembers(c *gin.Context) {
	members, err := h.orgService.ListMembers(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, members)
}

//CreateMember 处理 POST /api/tenant/org/members
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

//UpdateMember 处理 PUT /api/tenant/org/members/:id
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

//DeleteMember 处理 DELETE /api/tenant/org/members/:id
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
//Helper：将 ServiceError 映射到 HTTP 响应
// ---------------------------------------------------------------------------

func handleServiceError(c *gin.Context, err error) {
	httpStatus := mapServiceErrorToHTTP(err)
	if svcErr, ok := err.(*service.ServiceError); ok {
		response.Error(c, httpStatus, svcErr.Code, svcErr.Message)
		return
	}
	response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "服务器内部错误")
}
