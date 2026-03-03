package dto

// --- Department DTOs ---

// CreateDepartmentRequest is the request body for POST /api/tenant/org/departments
type CreateDepartmentRequest struct {
	Name      string  `json:"name" binding:"required"`
	ParentID  *string `json:"parent_id"`
	Manager   string  `json:"manager"`
	SortOrder int     `json:"sort_order"`
}

// UpdateDepartmentRequest is the request body for PUT /api/tenant/org/departments/:id
type UpdateDepartmentRequest struct {
	Name      string  `json:"name"`
	ParentID  *string `json:"parent_id"`
	Manager   string  `json:"manager"`
	SortOrder int     `json:"sort_order"`
}

// DepartmentResponse is the response body for department endpoints
type DepartmentResponse struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	ParentID  *string `json:"parent_id"`
	Manager   string  `json:"manager"`
	SortOrder int     `json:"sort_order"`
	CreatedAt string  `json:"created_at"`
	UpdatedAt string  `json:"updated_at"`
}

// --- OrgRole DTOs ---

// CreateRoleRequest is the request body for POST /api/tenant/org/roles
type CreateRoleRequest struct {
	Name            string      `json:"name" binding:"required"`
	Description     string      `json:"description"`
	PagePermissions interface{} `json:"page_permissions"` // JSON array
}

// UpdateRoleRequest is the request body for PUT /api/tenant/org/roles/:id
type UpdateRoleRequest struct {
	Name            string      `json:"name"`
	Description     string      `json:"description"`
	PagePermissions interface{} `json:"page_permissions"`
}

// RoleResponse is the response body for org role endpoints
type RoleResponse struct {
	ID              string      `json:"id"`
	Name            string      `json:"name"`
	Description     string      `json:"description"`
	PagePermissions interface{} `json:"page_permissions"`
	IsSystem        bool        `json:"is_system"`
	CreatedAt       string      `json:"created_at"`
	UpdatedAt       string      `json:"updated_at"`
}

// --- Member DTOs ---

// CreateMemberRequest is the request body for POST /api/tenant/org/members
type CreateMemberRequest struct {
	Username     string   `json:"username" binding:"required"`
	DisplayName  string   `json:"display_name" binding:"required"`
	Password     string   `json:"password" binding:"required,min=6"`
	Email        string   `json:"email"`
	Phone        string   `json:"phone"`
	DepartmentID string   `json:"department_id" binding:"required"`
	RoleIDs      []string `json:"role_ids" binding:"required"`
	Position     string   `json:"position"`
}

// UpdateMemberRequest is the request body for PUT /api/tenant/org/members/:id
type UpdateMemberRequest struct {
	DepartmentID string   `json:"department_id"`
	RoleIDs      []string `json:"role_ids"`
	Position     string   `json:"position"`
	Status       string   `json:"status"`
}

// MemberResponse is the response body for member endpoints
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

// MemberUserInfo contains user details embedded in a member response
type MemberUserInfo struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	AvatarURL   string `json:"avatar_url"`
}
