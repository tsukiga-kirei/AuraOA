package dto

// LoginRequest is the request body for POST /api/auth/login
type LoginRequest struct {
	Username      string `json:"username" binding:"required"`
	Password      string `json:"password" binding:"required"`
	TenantID      string `json:"tenant_id"`
	PreferredRole string `json:"preferred_role"`
}

// RoleInfo represents a role assignment in the login response
type RoleInfo struct {
	ID       string  `json:"id"`
	Role     string  `json:"role"`
	TenantID *string `json:"tenant_id"`
	Label    string  `json:"label"`
}

// UserInfo represents user details in the login response
type UserInfo struct {
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Phone       string `json:"phone"`
	AvatarURL   string `json:"avatar_url"`
	Locale      string `json:"locale"`
}

// LoginResponse is the response body for POST /api/auth/login
type LoginResponse struct {
	AccessToken  string     `json:"access_token"`
	RefreshToken string     `json:"refresh_token"`
	User         UserInfo   `json:"user"`
	Roles        []RoleInfo `json:"roles"`
	ActiveRole   RoleInfo   `json:"active_role"`
	Permissions  []string   `json:"permissions"`
}

// RefreshRequest is the request body for POST /api/auth/refresh
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshResponse is the response body for POST /api/auth/refresh
type RefreshResponse struct {
	AccessToken string `json:"access_token"`
}

// SwitchRoleRequest is the request body for PUT /api/auth/switch-role
type SwitchRoleRequest struct {
	RoleID string `json:"role_id" binding:"required"`
}

// SwitchRoleResponse is the response body for PUT /api/auth/switch-role
type SwitchRoleResponse struct {
	AccessToken string   `json:"access_token"`
	ActiveRole  RoleInfo `json:"active_role"`
	Permissions []string `json:"permissions"`
}

// MenuItem represents a single menu entry
type MenuItem struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Path  string `json:"path"`
}

// MenuResponse is the response body for GET /api/auth/menu
type MenuResponse struct {
	Menus []MenuItem `json:"menus"`
}
