package dto

//LoginRequest 是 POST /api/auth/login 的请求正文
type LoginRequest struct {
	Username      string `json:"username" binding:"required"`
	Password      string `json:"password" binding:"required"`
	TenantID      string `json:"tenant_id"`
	PreferredRole string `json:"preferred_role"`
}

//RoleInfo 表示登录响应中的角色分配
type RoleInfo struct {
	ID         string  `json:"id"`
	Role       string  `json:"role"`
	TenantID   *string `json:"tenant_id"`
	TenantName *string `json:"tenant_name"`
	Label      string  `json:"label"`
}

//UserInfo 表示登录响应中的用户详细信息
type UserInfo struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Locale      string `json:"locale"`
}

//LoginResponse 是 POST /api/auth/login 的响应正文
type LoginResponse struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	User         UserInfo   `json:"user"`
	Roles        []RoleInfo `json:"roles"`
	ActiveRole   RoleInfo   `json:"active_role"`
	Permissions  []string   `json:"permissions"`
}

//RefreshRequest 是 POST /api/auth/refresh 的请求正文
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

//RefreshResponse 是 POST /api/auth/refresh 的响应正文
type RefreshResponse struct {
	AccessToken string `json:"access_token"`
}

//SwitchRoleRequest 是 PUT /api/auth/switch-role 的请求正文
type SwitchRoleRequest struct {
	RoleID string `json:"role_id" binding:"required"`
}

//SwitchRoleResponse 是 PUT /api/auth/switch-role 的响应正文
type SwitchRoleResponse struct {
	AccessToken string     `json:"access_token"`
	ActiveRole  RoleInfo   `json:"active_role"`
	Permissions []string   `json:"permissions"`
	Menus       []MenuItem `json:"menus"`
}

//MenuItem 代表单个菜单项
type MenuItem struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Path  string `json:"path"`
}

//MenuResponse 是 GET /api/auth/menu 的响应正文
type MenuResponse struct {
	Menus []MenuItem `json:"menus"`
}

// ChangePasswordRequest is the request body for PUT /api/auth/change-password
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

// ---------------------------------------------------------------------------
// GET /api/auth/me
// ---------------------------------------------------------------------------

// MeOrgRole represents an org role in the /auth/me response.
type MeOrgRole struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	PagePermissions []string `json:"page_permissions"`
	IsSystem        bool     `json:"is_system"`
}

// MeResponse is the response body for GET /api/auth/me.
type MeResponse struct {
	// User basic info
	User UserInfo `json:"user"`

	// All role assignments (system-level, same as login)
	Roles      []RoleInfo `json:"roles"`
	ActiveRole RoleInfo   `json:"active_role"`

	// Org-level info (only present when active role has a tenant)
	TenantName     string      `json:"tenant_name"`
	DepartmentName string      `json:"department_name"`
	Position       string      `json:"position"`
	OrgRoles        []MeOrgRole `json:"org_roles"`
	PagePermissions []string    `json:"page_permissions"`

	// Security info
	PasswordChangedAt string           `json:"password_changed_at"`
	LoginHistory      []LoginHistoryItem `json:"login_history"`
}

// LoginHistoryItem represents a single login history entry in the /auth/me response.
type LoginHistoryItem struct {
	Time   string `json:"time"`
	IP     string `json:"ip"`
	Device string `json:"device"`
}

// UpdateProfileRequest is the request body for PUT /api/auth/profile.
type UpdateProfileRequest struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
}

// UpdateLocaleRequest is the request body for PUT /api/auth/locale.
type UpdateLocaleRequest struct {
	Locale string `json:"locale" binding:"required"`
}

