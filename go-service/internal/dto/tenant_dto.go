package dto

// CreateTenantRequest 是 POST /api/admin/tenants 的请求正文。
// Code 由后端自动生成，无需前端提供。
type CreateTenantRequest struct {
	Name                string  `json:"name" binding:"required"`
	Code                string  `json:"code"`
	Description         string  `json:"description"`
	OADBConnectionID    string  `json:"oa_db_connection_id"`
	TokenQuota          int     `json:"token_quota"`
	MaxConcurrency      int     `json:"max_concurrency"`
	PrimaryModelID      string  `json:"primary_model_id"`
	FallbackModelID     string  `json:"fallback_model_id"`
	MaxTokensPerRequest int     `json:"max_tokens_per_request"`
	Temperature         float64 `json:"temperature"`
	TimeoutSeconds      int     `json:"timeout_seconds"`
	RetryCount          int     `json:"retry_count"`
	LogRetentionDays    int     `json:"log_retention_days"`
	DataRetentionDays   int     `json:"data_retention_days"`
	ContactName         string  `json:"contact_name"`
	ContactEmail        string  `json:"contact_email"`
	ContactPhone        string  `json:"contact_phone"`

	// 租户管理员信息（创建租户时同步创建管理员账号）
	AdminUsername    string `json:"admin_username" binding:"required"`
	AdminDisplayName string `json:"admin_display_name" binding:"required"`
	AdminPassword    string `json:"admin_password"`
	AdminEmail       string `json:"admin_email"`
	AdminPhone       string `json:"admin_phone"`
	AdminDeptName    string `json:"admin_dept_name" binding:"required"` // 默认部门名称
}

// UpdateTenantRequest 是 PUT /api/admin/tenants/:id 的请求正文。
type UpdateTenantRequest struct {
	Name                string   `json:"name"`
	Status              string   `json:"status"`
	Description         string   `json:"description"`
	OADBConnectionID    *string  `json:"oa_db_connection_id"`
	TokenQuota          int      `json:"token_quota"`
	MaxConcurrency      int      `json:"max_concurrency"`
	PrimaryModelID      *string  `json:"primary_model_id"`
	FallbackModelID     *string  `json:"fallback_model_id"`
	MaxTokensPerRequest int      `json:"max_tokens_per_request"`
	Temperature         *float64 `json:"temperature"`
	TimeoutSeconds      int      `json:"timeout_seconds"`
	RetryCount          int      `json:"retry_count"`
	SSOEnabled          *bool    `json:"sso_enabled"`
	SSOEndpoint         string   `json:"sso_endpoint"`
	LogRetentionDays    int      `json:"log_retention_days"`
	DataRetentionDays   int      `json:"data_retention_days"`
	ContactName         string   `json:"contact_name"`
	ContactEmail        string   `json:"contact_email"`
	ContactPhone        string   `json:"contact_phone"`
}

// TenantResponse 是租户端点的响应正文。
type TenantResponse struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	Code                string  `json:"code"`
	Description         string  `json:"description"`
	Status              string  `json:"status"`
	OADBConnectionID    string  `json:"oa_db_connection_id"`
	TokenQuota          int     `json:"token_quota"`
	TokenUsed           int     `json:"token_used"`
	MaxConcurrency      int     `json:"max_concurrency"`
	PrimaryModelID      string  `json:"primary_model_id"`
	FallbackModelID     string  `json:"fallback_model_id"`
	MaxTokensPerRequest int     `json:"max_tokens_per_request"`
	Temperature         float64 `json:"temperature"`
	TimeoutSeconds      int     `json:"timeout_seconds"`
	RetryCount          int     `json:"retry_count"`
	SSOEnabled          bool    `json:"sso_enabled"`
	SSOEndpoint         string  `json:"sso_endpoint"`
	LogRetentionDays    int     `json:"log_retention_days"`
	DataRetentionDays   int     `json:"data_retention_days"`
	ContactName         string  `json:"contact_name"`
	ContactEmail        string  `json:"contact_email"`
	ContactPhone        string  `json:"contact_phone"`
	CreatedAt           string  `json:"created_at"`
	UpdatedAt           string  `json:"updated_at"`
}

// TenantStatsResponse 是 GET /api/admin/tenants/:id/stats 的响应正文。
type TenantStatsResponse struct {
	TenantID        string `json:"tenant_id"`
	MemberCount     int64  `json:"member_count"`
	DepartmentCount int64  `json:"department_count"`
	RoleCount       int64  `json:"role_count"`
}

// PublicTenantItem 是公共登录页面的轻量级租户条目。
type PublicTenantItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

// TenantMemberItem 是系统管理员查看租户成员的响应条目。
type TenantMemberItem struct {
	ID             string   `json:"id"`
	Username       string   `json:"username"`
	DisplayName    string   `json:"display_name"`
	Email          string   `json:"email"`
	Phone          string   `json:"phone"`
	DepartmentName string   `json:"department_name"`
	RoleNames      []string `json:"role_names"`
	Position       string   `json:"position"`
	Status         string   `json:"status"`
	CreatedAt      string   `json:"created_at"`
}
