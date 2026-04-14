package dto

// --- 部门 DTO ---

// CreateDepartmentRequest 创建部门请求（POST /api/tenant/org/departments）。
type CreateDepartmentRequest struct {
	Name      string  `json:"name" binding:"required"`
	ParentID  *string `json:"parent_id"`
	Manager   string  `json:"manager"`
	SortOrder int     `json:"sort_order"`
}

// UpdateDepartmentRequest 更新部门信息请求（PUT /api/tenant/org/departments/:id）。
type UpdateDepartmentRequest struct {
	Name      string  `json:"name"`
	ParentID  *string `json:"parent_id"`
	Manager   string  `json:"manager"`
	SortOrder int     `json:"sort_order"`
}

// DepartmentResponse 部门信息响应。
type DepartmentResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	ParentID    *string `json:"parent_id"`
	Manager     string  `json:"manager"`
	SortOrder   int     `json:"sort_order"`
	MemberCount int64   `json:"member_count"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// --- 组织角色 DTO ---

// CreateRoleRequest 创建组织角色请求（POST /api/tenant/org/roles）。
type CreateRoleRequest struct {
	Name            string      `json:"name" binding:"required"`
	Description     string      `json:"description"`
	PagePermissions interface{} `json:"page_permissions"` // JSON 数组，页面权限列表
}

// UpdateRoleRequest 更新组织角色请求（PUT /api/tenant/org/roles/:id）。
type UpdateRoleRequest struct {
	Name            string      `json:"name"`
	Description     string      `json:"description"`
	PagePermissions interface{} `json:"page_permissions"`
}

// RoleResponse 组织角色信息响应。
type RoleResponse struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	Description     string      `json:"description"`
	PagePermissions interface{} `json:"page_permissions"`
	IsSystem        bool        `json:"is_system"`
	CreatedAt       string      `json:"created_at"`
	UpdatedAt       string      `json:"updated_at"`
}

// --- 成员 DTO ---

// CreateMemberRequest 创建租户成员请求（POST /api/tenant/org/members）。
type CreateMemberRequest struct {
	Username     string   `json:"username" binding:"required"`
	DisplayName  string   `json:"display_name" binding:"required"`
	Password     string   `json:"password"`
	Email        string   `json:"email"`
	Phone        string   `json:"phone"`
	DepartmentID string   `json:"department_id" binding:"required"`
	RoleIDs      []string `json:"role_ids" binding:"required"`
	Position     string   `json:"position"`
}

// UpdateMemberRequest 更新租户成员信息请求（PUT /api/tenant/org/members/:id）。
type UpdateMemberRequest struct {
	DisplayName  string   `json:"display_name"`
	Email        string   `json:"email"`
	Phone        string   `json:"phone"`
	DepartmentID string   `json:"department_id"`
	RoleIDs      []string `json:"role_ids"`
	Position     string   `json:"position"`
	Status       string   `json:"status"`
}

// MemberResponse 租户成员信息响应。
type MemberResponse struct {
	ID         string             `json:"id"`
	User       MemberUserInfo     `json:"user"`
	Department DepartmentResponse `json:"department"`
	Roles      []RoleResponse     `json:"roles"`
	Position   string             `json:"position"`
	Status     string             `json:"status"`
	CreatedAt  string             `json:"created_at"`
	UpdatedAt  string             `json:"updated_at"`
}

// MemberUserInfo 成员响应中内嵌的用户基本信息。
type MemberUserInfo struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	AvatarURL   string `json:"avatar_url"`
}
