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
	AvatarURL   string `json:"avatar_url"`
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
