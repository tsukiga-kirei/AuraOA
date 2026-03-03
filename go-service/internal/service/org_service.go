package service

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/dto"
	"oa-smart-audit/go-service/internal/model"
	"oa-smart-audit/go-service/internal/pkg/errcode"
	"oa-smart-audit/go-service/internal/pkg/hash"
	"oa-smart-audit/go-service/internal/repository"
)

// OrgService handles department, role, and member CRUD operations with tenant isolation.
type OrgService struct {
	orgRepo  *repository.OrgRepo
	userRepo *repository.UserRepo
	db       *gorm.DB
}

// NewOrgService creates a new OrgService instance.
func NewOrgService(orgRepo *repository.OrgRepo, userRepo *repository.UserRepo, db *gorm.DB) *OrgService {
	return &OrgService{
		orgRepo:  orgRepo,
		userRepo: userRepo,
		db:       db,
	}
}

// ---------------------------------------------------------------------------
// Department CRUD
// ---------------------------------------------------------------------------

// ListDepartments returns all departments for the current tenant.
func (s *OrgService) ListDepartments(c *gin.Context) ([]dto.DepartmentResponse, error) {
	departments, err := s.orgRepo.ListDepartments(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	result := make([]dto.DepartmentResponse, len(departments))
	for i, d := range departments {
		result[i] = toDepartmentResponse(&d)
	}
	return result, nil
}

// CreateDepartment creates a new department in the current tenant.
func (s *OrgService) CreateDepartment(c *gin.Context, tenantID uuid.UUID, req *dto.CreateDepartmentRequest) (*dto.DepartmentResponse, error) {
	dept := &model.Department{
		TenantID:  tenantID,
		Name:      req.Name,
		Manager:   req.Manager,
		SortOrder: req.SortOrder,
	}
	if req.ParentID != nil {
		pid, err := uuid.Parse(*req.ParentID)
		if err != nil {
			return nil, newServiceError(errcode.ErrParamValidation, "参数校验失败")
		}
		dept.ParentID = &pid
	}
	if err := s.orgRepo.CreateDepartment(dept); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	resp := toDepartmentResponse(dept)
	return &resp, nil
}

// UpdateDepartment updates an existing department.
func (s *OrgService) UpdateDepartment(c *gin.Context, id uuid.UUID, req *dto.UpdateDepartmentRequest) (*dto.DepartmentResponse, error) {
	dept, err := s.orgRepo.FindDepartmentByID(c, id)
	if err != nil {
		return nil, newServiceError(errcode.ErrResourceNotFound, "资源不存在")
	}
	if req.Name != "" {
		dept.Name = req.Name
	}
	if req.ParentID != nil {
		pid, err := uuid.Parse(*req.ParentID)
		if err != nil {
			return nil, newServiceError(errcode.ErrParamValidation, "参数校验失败")
		}
		dept.ParentID = &pid
	}
	dept.Manager = req.Manager
	dept.SortOrder = req.SortOrder

	if err := s.orgRepo.UpdateDepartment(dept); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	resp := toDepartmentResponse(dept)
	return &resp, nil
}

// DeleteDepartment deletes a department after checking it has no members.
func (s *OrgService) DeleteDepartment(c *gin.Context, id uuid.UUID) error {
	// Verify department exists in current tenant
	_, err := s.orgRepo.FindDepartmentByID(c, id)
	if err != nil {
		return newServiceError(errcode.ErrResourceNotFound, "资源不存在")
	}
	// Check if department has members
	count, err := s.orgRepo.CountMembersByDept(id)
	if err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	if count > 0 {
		return newServiceError(errcode.ErrParamValidation, "部门下存在成员，无法删除")
	}
	if err := s.orgRepo.DeleteDepartment(id); err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Role CRUD
// ---------------------------------------------------------------------------

// ListRoles returns all org roles for the current tenant.
func (s *OrgService) ListRoles(c *gin.Context) ([]dto.RoleResponse, error) {
	roles, err := s.orgRepo.ListRoles(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	result := make([]dto.RoleResponse, len(roles))
	for i, r := range roles {
		result[i] = toRoleResponse(&r)
	}
	return result, nil
}

// CreateRole creates a new org role in the current tenant.
func (s *OrgService) CreateRole(c *gin.Context, tenantID uuid.UUID, req *dto.CreateRoleRequest) (*dto.RoleResponse, error) {
	pagePerms, err := json.Marshal(req.PagePermissions)
	if err != nil {
		pagePerms = []byte("[]")
	}
	role := &model.OrgRole{
		TenantID:        tenantID,
		Name:            req.Name,
		Description:     req.Description,
		PagePermissions: pagePerms,
	}
	if err := s.orgRepo.CreateRole(role); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	resp := toRoleResponse(role)
	return &resp, nil
}

// UpdateRole updates an existing org role.
func (s *OrgService) UpdateRole(c *gin.Context, id uuid.UUID, req *dto.UpdateRoleRequest) (*dto.RoleResponse, error) {
	role, err := s.orgRepo.FindRoleByID(c, id)
	if err != nil {
		return nil, newServiceError(errcode.ErrResourceNotFound, "资源不存在")
	}
	if req.Name != "" {
		role.Name = req.Name
	}
	if req.Description != "" {
		role.Description = req.Description
	}
	if req.PagePermissions != nil {
		pagePerms, err := json.Marshal(req.PagePermissions)
		if err != nil {
			return nil, newServiceError(errcode.ErrParamValidation, "参数校验失败")
		}
		role.PagePermissions = pagePerms
	}
	if err := s.orgRepo.UpdateRole(role); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	resp := toRoleResponse(role)
	return &resp, nil
}

// DeleteRole deletes an org role after checking it is not a system role.
func (s *OrgService) DeleteRole(c *gin.Context, id uuid.UUID) error {
	role, err := s.orgRepo.FindRoleByID(c, id)
	if err != nil {
		return newServiceError(errcode.ErrResourceNotFound, "资源不存在")
	}
	if role.IsSystem {
		return newServiceError(errcode.ErrParamValidation, "系统角色不可删除")
	}
	if err := s.orgRepo.DeleteRole(id); err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Member CRUD
// ---------------------------------------------------------------------------

// ListMembers returns all org members for the current tenant.
func (s *OrgService) ListMembers(c *gin.Context) ([]dto.MemberResponse, error) {
	members, err := s.orgRepo.ListMembers(c)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	result := make([]dto.MemberResponse, len(members))
	for i, m := range members {
		result[i] = toMemberResponse(&m)
	}
	return result, nil
}

// CreateMember creates a new org member with automatic user creation and role assignment.
func (s *OrgService) CreateMember(c *gin.Context, tenantID uuid.UUID, req *dto.CreateMemberRequest) (*dto.MemberResponse, error) {
	// 1. Check if user already exists; if so, check uniqueness within tenant
	existingUser, _ := s.userRepo.FindByUsername(req.Username)
	if existingUser != nil {
		// Check if this user already has a member record in this tenant
		existingMember, _ := s.orgRepo.FindByUserAndTenant(existingUser.ID, tenantID)
		if existingMember != nil {
			return nil, newServiceError(errcode.ErrResourceConflict, "资源冲突")
		}
	}

	// 2. Validate department_id exists in tenant
	deptID, err := uuid.Parse(req.DepartmentID)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "参数校验失败")
	}
	dept, err := s.orgRepo.FindDepartmentByID(c, deptID)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "参数校验失败")
	}

	// 3. Validate role_ids exist in tenant
	roleUUIDs := make([]uuid.UUID, len(req.RoleIDs))
	for i, rid := range req.RoleIDs {
		parsed, err := uuid.Parse(rid)
		if err != nil {
			return nil, newServiceError(errcode.ErrParamValidation, "参数校验失败")
		}
		roleUUIDs[i] = parsed
	}
	roles, err := s.orgRepo.FindRolesByIDs(roleUUIDs)
	if err != nil || len(roles) != len(roleUUIDs) {
		return nil, newServiceError(errcode.ErrParamValidation, "参数校验失败")
	}
	// Verify all roles belong to the current tenant
	for _, role := range roles {
		if role.TenantID != tenantID {
			return nil, newServiceError(errcode.ErrParamValidation, "参数校验失败")
		}
	}

	// 4. Find or create user
	var user *model.User
	if existingUser != nil {
		user = existingUser
	} else {
		passwordHash, err := hash.HashPassword(req.Password)
		if err != nil {
			return nil, newServiceError(errcode.ErrInternalServer, "服务器内部错误")
		}
		user = &model.User{
			Username:          req.Username,
			PasswordHash:      passwordHash,
			DisplayName:       req.DisplayName,
			Email:             req.Email,
			Phone:             req.Phone,
			Status:            "active",
			PasswordChangedAt: time.Now(),
		}
		if err := s.db.Create(user).Error; err != nil {
			return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
		}
	}

	// 5. Create OrgMember record
	member := &model.OrgMember{
		TenantID:     tenantID,
		UserID:       user.ID,
		DepartmentID: deptID,
		Position:     req.Position,
		Status:       "active",
	}
	if err := s.orgRepo.CreateMember(member); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	// 6. Create org_member_roles associations
	if err := s.db.Model(member).Association("Roles").Append(&roles); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	// 7. Create UserRoleAssignment with role="business" for this tenant
	businessAssignment := &model.UserRoleAssignment{
		UserID:  user.ID,
		Role:    "business",
		TenantID: &tenantID,
		Label:   "业务用户 - " + req.DisplayName,
	}
	if err := s.db.Create(businessAssignment).Error; err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	// 8. If role_ids contains the tenant admin role, also create tenant_admin assignment
	for _, role := range roles {
		if role.Name == "租户管理员" || (role.IsSystem && role.Name == "租户管理员") {
			tenantAdminAssignment := &model.UserRoleAssignment{
				UserID:  user.ID,
				Role:    "tenant_admin",
				TenantID: &tenantID,
				Label:   "租户管理员 - " + req.DisplayName,
			}
			if err := s.db.Create(tenantAdminAssignment).Error; err != nil {
				return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
			}
			break
		}
	}

	// 9. Reload member with associations for response
	member.User = *user
	member.Department = *dept
	member.Roles = roles

	resp := toMemberResponse(member)
	return &resp, nil
}

// UpdateMember updates an existing org member's department, position, status, and roles.
func (s *OrgService) UpdateMember(c *gin.Context, id uuid.UUID, req *dto.UpdateMemberRequest) (*dto.MemberResponse, error) {
	member, err := s.orgRepo.FindMemberByID(c, id)
	if err != nil {
		return nil, newServiceError(errcode.ErrResourceNotFound, "资源不存在")
	}

	// Update department_id if provided
	if req.DepartmentID != "" {
		deptID, err := uuid.Parse(req.DepartmentID)
		if err != nil {
			return nil, newServiceError(errcode.ErrParamValidation, "参数校验失败")
		}
		_, err = s.orgRepo.FindDepartmentByID(c, deptID)
		if err != nil {
			return nil, newServiceError(errcode.ErrParamValidation, "参数校验失败")
		}
		member.DepartmentID = deptID
	}

	if req.Position != "" {
		member.Position = req.Position
	}
	if req.Status != "" {
		member.Status = req.Status
	}

	if err := s.orgRepo.UpdateMember(member); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	// Replace role associations if role_ids provided
	if len(req.RoleIDs) > 0 {
		roleUUIDs := make([]uuid.UUID, len(req.RoleIDs))
		for i, rid := range req.RoleIDs {
			parsed, err := uuid.Parse(rid)
			if err != nil {
				return nil, newServiceError(errcode.ErrParamValidation, "参数校验失败")
			}
			roleUUIDs[i] = parsed
		}
		roles, err := s.orgRepo.FindRolesByIDs(roleUUIDs)
		if err != nil || len(roles) != len(roleUUIDs) {
			return nil, newServiceError(errcode.ErrParamValidation, "参数校验失败")
		}
		if err := s.db.Model(member).Association("Roles").Replace(&roles); err != nil {
			return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
		}
		member.Roles = roles
	}

	// Reload for full response
	reloaded, err := s.orgRepo.FindMemberByID(c, id)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	resp := toMemberResponse(reloaded)
	return &resp, nil
}

// DeleteMember deletes an org member and cascades cleanup of roles and user_role_assignments.
func (s *OrgService) DeleteMember(c *gin.Context, id uuid.UUID) error {
	member, err := s.orgRepo.FindMemberByID(c, id)
	if err != nil {
		return newServiceError(errcode.ErrResourceNotFound, "资源不存在")
	}

	// 1. Clear org_member_roles associations
	if err := s.db.Model(member).Association("Roles").Clear(); err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	// 2. Delete org_members record
	if err := s.orgRepo.DeleteMember(id); err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	// 3. Delete user_role_assignments for this user + tenant
	if err := s.db.Where("user_id = ? AND tenant_id = ?", member.UserID, member.TenantID).
		Delete(&model.UserRoleAssignment{}).Error; err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	return nil
}

// ---------------------------------------------------------------------------
// Helper functions: model → DTO conversion
// ---------------------------------------------------------------------------

func toDepartmentResponse(d *model.Department) dto.DepartmentResponse {
	resp := dto.DepartmentResponse{
		ID:        d.ID.String(),
		Name:      d.Name,
		Manager:   d.Manager,
		SortOrder: d.SortOrder,
		CreatedAt: d.CreatedAt.Format(time.RFC3339),
		UpdatedAt: d.UpdatedAt.Format(time.RFC3339),
	}
	if d.ParentID != nil {
		pid := d.ParentID.String()
		resp.ParentID = &pid
	}
	return resp
}

func toRoleResponse(r *model.OrgRole) dto.RoleResponse {
	var pagePerms interface{}
	if err := json.Unmarshal(r.PagePermissions, &pagePerms); err != nil {
		pagePerms = []interface{}{}
	}
	return dto.RoleResponse{
		ID:              r.ID.String(),
		Name:            r.Name,
		Description:     r.Description,
		PagePermissions: pagePerms,
		IsSystem:        r.IsSystem,
		CreatedAt:       r.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       r.UpdatedAt.Format(time.RFC3339),
	}
}

func toMemberResponse(m *model.OrgMember) dto.MemberResponse {
	roles := make([]dto.RoleResponse, len(m.Roles))
	for i, r := range m.Roles {
		roles[i] = toRoleResponse(&r)
	}
	return dto.MemberResponse{
		ID: m.ID.String(),
		User: dto.MemberUserInfo{
			ID:          m.User.ID.String(),
			Username:    m.User.Username,
			DisplayName: m.User.DisplayName,
			Email:       m.User.Email,
			Phone:       m.User.Phone,
			AvatarURL:   m.User.AvatarURL,
		},
		Department: toDepartmentResponse(&m.Department),
		Roles:      roles,
		Position:   m.Position,
		Status:     m.Status,
		CreatedAt:  m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  m.UpdatedAt.Format(time.RFC3339),
	}
}
