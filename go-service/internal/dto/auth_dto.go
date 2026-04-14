package dto

// BootstrapStatusResponse 系统初始化状态响应（GET /api/auth/bootstrap-status）。
type BootstrapStatusResponse struct {
	NeedsSetup bool `json:"needs_setup"`
}

// BootstrapAdminRequest 初始化超级管理员请求（仅系统无任何用户时允许，POST /api/auth/bootstrap）。
type BootstrapAdminRequest struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required,min=8"`
	DisplayName string `json:"display_name" binding:"required"`
}

// LoginRequest 用户登录请求（POST /api/auth/login）。
type LoginRequest struct {
	Username      string `json:"username" binding:"required"`
	Password      string `json:"password" binding:"required"`
	TenantID      string `json:"tenant_id"`
	PreferredRole string `json:"preferred_role"`
}

// RoleInfo 登录响应中的角色分配信息。
type RoleInfo struct {
	ID         string  `json:"id"`
	Role       string  `json:"role"`
	TenantID   *string `json:"tenant_id"`
	TenantName *string `json:"tenant_name"`
	Label      string  `json:"label"`
}

// UserInfo 登录响应中的用户基本信息。
type UserInfo struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	Locale      string `json:"locale"`
}

// LoginResponse 用户登录响应（POST /api/auth/login）。
type LoginResponse struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	User         UserInfo   `json:"user"`
	Roles        []RoleInfo `json:"roles"`
	ActiveRole   RoleInfo   `json:"active_role"`
	Permissions  []string   `json:"permissions"`
}

// RefreshRequest 刷新访问令牌请求（POST /api/auth/refresh）。
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshResponse 刷新访问令牌响应（POST /api/auth/refresh）。
type RefreshResponse struct {
	AccessToken string `json:"access_token"`
}

// SwitchRoleRequest 切换当前角色请求（PUT /api/auth/switch-role）。
type SwitchRoleRequest struct {
	RoleID string `json:"role_id" binding:"required"`
}

// SwitchRoleResponse 切换角色响应，包含新令牌和菜单（PUT /api/auth/switch-role）。
type SwitchRoleResponse struct {
	AccessToken string     `json:"access_token"`
	ActiveRole  RoleInfo   `json:"active_role"`
	Permissions []string   `json:"permissions"`
	Menus       []MenuItem `json:"menus"`
}

// MenuItem 单个菜单项。
type MenuItem struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Path  string `json:"path"`
}

// MenuResponse 菜单列表响应（GET /api/auth/menu）。
type MenuResponse struct {
	Menus []MenuItem `json:"menus"`
}

// ChangePasswordRequest 修改密码请求（PUT /api/auth/change-password）。
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required"`
}

// MeOrgRole 当前用户在租户内的组织角色信息。
type MeOrgRole struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	PagePermissions []string `json:"page_permissions"`
	IsSystem        bool     `json:"is_system"`
}

// MeResponse 当前登录用户详情响应（GET /api/auth/me）。
type MeResponse struct {
	// 用户基本信息
	User UserInfo `json:"user"`

	// 所有角色分配（系统级，与登录响应一致）
	Roles      []RoleInfo `json:"roles"`
	ActiveRole RoleInfo   `json:"active_role"`

	// 租户级信息（仅当前角色归属某租户时存在）
	TenantName      string      `json:"tenant_name"`
	DepartmentName  string      `json:"department_name"`
	Position        string      `json:"position"`
	OrgRoles        []MeOrgRole `json:"org_roles"`
	PagePermissions []string    `json:"page_permissions"`

	// 安全信息
	PasswordChangedAt string             `json:"password_changed_at"`
	LoginHistory      []LoginHistoryItem `json:"login_history"`
}

// LoginHistoryItem 单条登录历史记录。
type LoginHistoryItem struct {
	Time   string `json:"time"`
	IP     string `json:"ip"`
	Device string `json:"device"`
}

// UpdateProfileRequest 更新个人资料请求（PUT /api/auth/profile）。
type UpdateProfileRequest struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
}

// UpdateLocaleRequest 更新界面语言偏好请求（PUT /api/auth/locale）。
type UpdateLocaleRequest struct {
	Locale string `json:"locale" binding:"required"`
}
