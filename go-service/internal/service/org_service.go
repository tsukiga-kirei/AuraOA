package service

import (
	"encoding/json"
	"regexp"
	"strings"
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

// OrgService 通过租户隔离处理部门、角色和成员 CRUD 操作。
type OrgService struct {
	orgRepo  *repository.OrgRepo
	userRepo *repository.UserRepo
	db       *gorm.DB
}

// NewOrgService 创建一个新的 OrgService 实例。
func NewOrgService(orgRepo *repository.OrgRepo, userRepo *repository.UserRepo, db *gorm.DB) *OrgService {
	return &OrgService{
		orgRepo:  orgRepo,
		userRepo: userRepo,
		db:       db,
	}
}

// ---------------------------------------------------------------------------
//部门增删改查
// ---------------------------------------------------------------------------

// ListDepartments 返回当前租户的所有部门。
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

// CreateDepartment 在当前租户中创建一个新部门。
func (s *OrgService) CreateDepartment(c *gin.Context, tenantID uuid.UUID, req *dto.CreateDepartmentRequest) (*dto.DepartmentResponse, error) {
	dept := &model.Department{
		ID:        uuid.New(),
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

// UpdateDepartment 更新现有部门。
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

// DeleteDepartment 在检查部门没有成员后删除该部门。
func (s *OrgService) DeleteDepartment(c *gin.Context, id uuid.UUID) error {
	//验证当前租户中是否存在部门
	_, err := s.orgRepo.FindDepartmentByID(c, id)
	if err != nil {
		return newServiceError(errcode.ErrResourceNotFound, "资源不存在")
	}
	//检查部门是否有成员
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
//角色增删改查
// ---------------------------------------------------------------------------

// ListRoles 返回当前租户的所有组织角色。
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

// CreateRole 在当前租户中创建新的组织角色。
func (s *OrgService) CreateRole(c *gin.Context, tenantID uuid.UUID, req *dto.CreateRoleRequest) (*dto.RoleResponse, error) {
	pagePerms, err := json.Marshal(req.PagePermissions)
	if err != nil {
		pagePerms = []byte("[]")
	}
	role := &model.OrgRole{
		ID:              uuid.New(),
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

// UpdateRole 更新现有组织角色。
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

// DeleteRole 在检查组织角色不是系统角色后删除该角色。
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
//会员增删改查
// ---------------------------------------------------------------------------

// ListMembers 返回当前租户的所有组织成员。
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

// CreateMember 通过自动用户创建和角色分配创建新的组织成员。
func (s *OrgService) CreateMember(c *gin.Context, tenantID uuid.UUID, req *dto.CreateMemberRequest) (*dto.MemberResponse, error) {
	// 0. 参数格式校验
	// 用户名只能包含英文字母、数字和下划线
	usernameRegex := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)
	if !usernameRegex.MatchString(req.Username) {
		return nil, newServiceError(errcode.ErrParamValidation, "用户名只能包含英文字母、数字和下划线，且以字母开头")
	}
	// 邮箱格式校验（如果提供）
	if req.Email != "" {
		emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
		if !emailRegex.MatchString(req.Email) {
			return nil, newServiceError(errcode.ErrParamValidation, "邮箱格式不正确")
		}
	}
	// 手机号必须为11位数字（如果提供）
	if req.Phone != "" {
		phoneRegex := regexp.MustCompile(`^\d{11}$`)
		if !phoneRegex.MatchString(req.Phone) {
			return nil, newServiceError(errcode.ErrParamValidation, "手机号必须为11位数字")
		}
	}

	//1. 检查用户是否已经存在；如果是这样，请检查租户内的唯一性
	existingUser, _ := s.userRepo.FindByUsername(req.Username)
	if existingUser != nil {
		//检查该用户在此租户中是否已有会员记录
		existingMember, _ := s.orgRepo.FindByUserAndTenant(existingUser.ID, tenantID)
		if existingMember != nil {
			return nil, newServiceError(errcode.ErrResourceConflict, "该用户名已存在于当前租户中")
		}
	}

	//2. 验证tenant中存在department_id
	deptID, err := uuid.Parse(req.DepartmentID)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "参数校验失败")
	}
	dept, err := s.orgRepo.FindDepartmentByID(c, deptID)
	if err != nil {
		return nil, newServiceError(errcode.ErrParamValidation, "参数校验失败")
	}

	//3. 验证租户中存在role_ids
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
	//验证所有角色都属于当前租户
	for _, role := range roles {
		if role.TenantID != tenantID {
			return nil, newServiceError(errcode.ErrParamValidation, "参数校验失败")
		}
	}

	//4. 查找或创建用户
	var user *model.User
	if existingUser != nil {
		user = existingUser
	} else {
		// Use default password if not provided by the request
		password := req.Password
		if password == "" {
			password = "123456"
		}
		passwordHash, err := hash.HashPassword(password)
		if err != nil {
			return nil, newServiceError(errcode.ErrInternalServer, "服务器内部错误")
		}
		user = &model.User{
			ID:                uuid.New(),
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

	//5. 创建组织成员记录
	member := &model.OrgMember{
		ID:           uuid.New(),
		TenantID:     tenantID,
		UserID:       user.ID,
		DepartmentID: deptID,
		Position:     req.Position,
		Status:       "active",
	}
	if err := s.orgRepo.CreateMember(member); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	//6. 创建 org_member_roles 关联
	if err := s.db.Model(member).Association("Roles").Append(&roles); err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	//7. 根据 page_permissions 推断需要的系统角色
	//   包含前台页面（非 /admin/ 前缀）→ business
	//   包含后台页面（/admin/ 前缀）→ tenant_admin
	needBusiness := false
	needTenantAdmin := false
	for _, role := range roles {
		var paths []string
		if err := json.Unmarshal(role.PagePermissions, &paths); err != nil {
			continue
		}
		for _, p := range paths {
			if strings.HasPrefix(p, "/admin/") {
				needTenantAdmin = true
			} else {
				needBusiness = true
			}
		}
	}
	// 至少给一个 business，确保用户能登录
	if !needBusiness && !needTenantAdmin {
		needBusiness = true
	}

	if needBusiness {
		businessAssignment := &model.UserRoleAssignment{
			ID:       uuid.New(),
			UserID:   user.ID,
			Role:     "business",
			TenantID: &tenantID,
			Label:    "业务用户 - " + req.DisplayName,
		}
		if err := s.db.Create(businessAssignment).Error; err != nil {
			return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
		}
	}

	if needTenantAdmin {
		tenantAdminAssignment := &model.UserRoleAssignment{
			ID:       uuid.New(),
			UserID:   user.ID,
			Role:     "tenant_admin",
			TenantID: &tenantID,
			Label:    "租户管理员 - " + req.DisplayName,
		}
		if err := s.db.Create(tenantAdminAssignment).Error; err != nil {
			return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
		}
	}

	//8. 重新加载成员的关联以进行响应
	member.User = *user
	member.Department = *dept
	member.Roles = roles

	resp := toMemberResponse(member)
	return &resp, nil
}

// UpdateMember 更新现有组织成员的部门、职位、状态和角色。
func (s *OrgService) UpdateMember(c *gin.Context, id uuid.UUID, req *dto.UpdateMemberRequest) (*dto.MemberResponse, error) {
	member, err := s.orgRepo.FindMemberByID(c, id)
	if err != nil {
		return nil, newServiceError(errcode.ErrResourceNotFound, "资源不存在")
	}

	//更新 Department_id（如果提供）
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

	// 同步更新 users 表字段（display_name, email, phone, status）
	userUpdates := map[string]interface{}{}
	if req.DisplayName != "" {
		userUpdates["display_name"] = req.DisplayName
	}
	if req.Email != "" {
		userUpdates["email"] = req.Email
	}
	if req.Phone != "" {
		userUpdates["phone"] = req.Phone
	}
	if req.Status != "" {
		userUpdates["status"] = req.Status
	}
	if len(userUpdates) > 0 {
		if err := s.db.Model(&model.User{}).Where("id = ?", member.UserID).Updates(userUpdates).Error; err != nil {
			return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
		}
	}

	//如果提供了 role_ids，则替换角色关联
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

	//重新加载以获得完整响应
	reloaded, err := s.orgRepo.FindMemberByID(c, id)
	if err != nil {
		return nil, newServiceError(errcode.ErrDatabase, "数据库错误")
	}
	resp := toMemberResponse(reloaded)
	return &resp, nil
}

// DeleteMember 删除组织成员并级联清理角色和 user_role_assignments。
func (s *OrgService) DeleteMember(c *gin.Context, id uuid.UUID) error {
	member, err := s.orgRepo.FindMemberByID(c, id)
	if err != nil {
		return newServiceError(errcode.ErrResourceNotFound, "资源不存在")
	}

	//1. 清除 org_member_roles 关联
	if err := s.db.Model(member).Association("Roles").Clear(); err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	//2.删除org_members记录
	if err := s.orgRepo.DeleteMember(id); err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	//3.删除该用户+租户的user_role_assignments
	if err := s.db.Where("user_id = ? AND tenant_id = ?", member.UserID, member.TenantID).
		Delete(&model.UserRoleAssignment{}).Error; err != nil {
		return newServiceError(errcode.ErrDatabase, "数据库错误")
	}

	return nil
}

// ---------------------------------------------------------------------------
//辅助函数：模型 → DTO 转换
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
