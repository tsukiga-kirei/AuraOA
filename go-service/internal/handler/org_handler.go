// 组织架构处理器，负责部门、角色和成员的增删改查。
package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/response"
	"oa-smart-audit/go-service/internal/service"
)

// OrgHandler 处理部门、角色和成员相关的 HTTP 请求。
type OrgHandler struct {
	orgService *service.OrgService
}

// NewOrgHandler 创建组织架构处理器实例。
func NewOrgHandler(orgService *service.OrgService) *OrgHandler {
	return &OrgHandler{orgService: orgService}
}

// getTenantID 从 gin.Context（由租户中间件注入）中提取并解析 tenant_id。
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

// errTenantIDMissing 表示请求上下文中缺少租户 ID 的错误。
var errTenantIDMissing = &service.ServiceError{Code: errcode.ErrParamValidation, Message: "tenant ID missing"}

// ── 部门管理 ──────────────────────────────────────────────────────────────

// ListDepartments 获取当前租户的所有部门列表。
// GET /api/tenant/org/departments
// 返回：部门列表数组（含层级结构）。
func (h *OrgHandler) ListDepartments(c *gin.Context) {
	departments, err := h.orgService.ListDepartments(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, departments)
}

// CreateDepartment 创建新部门。
// POST /api/tenant/org/departments
// 请求体：CreateDepartmentRequest（部门名称、上级部门 ID 等）
// 返回：新建的部门对象。
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

// UpdateDepartment 更新指定部门信息。
// PUT /api/tenant/org/departments/:id
// 路径参数：id（UUID 格式）
// 请求体：UpdateDepartmentRequest
// 返回：更新后的部门对象。
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

// DeleteDepartment 删除指定部门。
// DELETE /api/tenant/org/departments/:id
// 路径参数：id（UUID 格式）
// 返回：null（成功时）。
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

// ── 角色管理 ──────────────────────────────────────────────────────────────

// ListRoles 获取当前租户的所有角色列表。
// GET /api/tenant/org/roles
// 返回：角色列表数组。
func (h *OrgHandler) ListRoles(c *gin.Context) {
	roles, err := h.orgService.ListRoles(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, roles)
}

// CreateRole 创建新角色。
// POST /api/tenant/org/roles
// 请求体：CreateRoleRequest（角色名称、描述等）
// 返回：新建的角色对象。
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

// UpdateRole 更新指定角色信息。
// PUT /api/tenant/org/roles/:id
// 路径参数：id（UUID 格式）
// 请求体：UpdateRoleRequest
// 返回：更新后的角色对象。
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

// DeleteRole 删除指定角色。
// DELETE /api/tenant/org/roles/:id
// 路径参数：id（UUID 格式）
// 返回：null（成功时）。
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

// ── 成员管理 ──────────────────────────────────────────────────────────────

// ListMembers 获取当前租户的所有成员列表。
// GET /api/tenant/org/members
// 返回：成员列表数组（含部门和角色信息）。
func (h *OrgHandler) ListMembers(c *gin.Context) {
	members, err := h.orgService.ListMembers(c)
	if err != nil {
		handleServiceError(c, err)
		return
	}
	response.Success(c, members)
}

// CreateMember 创建新成员（将用户加入当前租户）。
// POST /api/tenant/org/members
// 请求体：CreateMemberRequest（用户名、部门 ID、角色 ID 等）
// 返回：新建的成员对象。
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

// UpdateMember 更新指定成员信息。
// PUT /api/tenant/org/members/:id
// 路径参数：id（UUID 格式）
// 请求体：UpdateMemberRequest
// 返回：更新后的成员对象。
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

// DeleteMember 从当前租户中移除指定成员。
// DELETE /api/tenant/org/members/:id
// 路径参数：id（UUID 格式）
// 返回：null（成功时）。
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

// ── 公共辅助函数 ──────────────────────────────────────────────────────────

// handleServiceError 将业务层错误映射为对应的 HTTP 响应。
func handleServiceError(c *gin.Context, err error) {
	httpStatus := mapServiceErrorToHTTP(err)
	if svcErr, ok := err.(*service.ServiceError); ok {
		response.Error(c, httpStatus, svcErr.Code, svcErr.Message)
		return
	}
	response.Error(c, http.StatusInternalServerError, errcode.ErrInternalServer, "internal server error")
}

// parseIntQuery 从查询参数中解析整数，解析失败时返回 defaultVal。
func parseIntQuery(c *gin.Context, key string, defaultVal int) int {
	s := c.Query(key)
	if s == "" {
		return defaultVal
	}
	var n int
	if _, err := fmt.Sscanf(s, "%d", &n); err != nil {
		return defaultVal
	}
	return n
}
