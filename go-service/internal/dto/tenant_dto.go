package dto

// CreateTenantRequest 创建租户请求（POST /api/admin/tenants）。
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

// UpdateTenantRequest 更新租户信息请求（PUT /api/admin/tenants/:id）。
type UpdateTenantRequest struct {
	Name                   string   `json:"name"`
	Status                 string   `json:"status"`
	Description            string   `json:"description"`
	EmbedEnabled           *bool    `json:"embed_enabled"`
	SSOBasicEnabled        *bool    `json:"sso_basic_enabled"`
	SSOBasicPassword       *string  `json:"sso_basic_password"`
	SSOBasicAllowedIPs     *string  `json:"sso_basic_allowed_ips"`
	SSOBasicAllowedDomains *string  `json:"sso_basic_allowed_domains"`
	OADBConnectionID       *string  `json:"oa_db_connection_id"`
	TokenQuota             int      `json:"token_quota"`
	MaxConcurrency         int      `json:"max_concurrency"`
	PrimaryModelID         *string  `json:"primary_model_id"`
	FallbackModelID        *string  `json:"fallback_model_id"`
	MaxTokensPerRequest    int      `json:"max_tokens_per_request"`
	Temperature            *float64 `json:"temperature"`
	TimeoutSeconds         int      `json:"timeout_seconds"`
	RetryCount             int      `json:"retry_count"`
	LogRetentionDays       int      `json:"log_retention_days"`
	DataRetentionDays      int      `json:"data_retention_days"`
	ContactName            string   `json:"contact_name"`
	ContactEmail           string   `json:"contact_email"`
	ContactPhone           string   `json:"contact_phone"`
}

// TenantResponse 租户详情响应。
type TenantResponse struct {
	ID                     string  `json:"id"`
	Name                   string  `json:"name"`
	Code                   string  `json:"code"`
	Description            string  `json:"description"`
	Status                 string  `json:"status"`
	EmbedEnabled           bool    `json:"embed_enabled"`
	EmbedTokenConfigured   bool    `json:"embed_token_configured"`
	EmbedTokenHint         string  `json:"embed_token_hint"`
	EmbedTokenRotatedAt    string  `json:"embed_token_rotated_at"`
	SSOBasicEnabled        bool    `json:"sso_basic_enabled"`
	SSOBasicPasswordSet    bool    `json:"sso_basic_password_set"`
	SSOBasicAllowedIPs     string  `json:"sso_basic_allowed_ips"`
	SSOBasicAllowedDomains string  `json:"sso_basic_allowed_domains"`
	OADBConnectionID       string  `json:"oa_db_connection_id"`
	TokenQuota             int     `json:"token_quota"`
	TokenUsed              int     `json:"token_used"`
	MaxConcurrency         int     `json:"max_concurrency"`
	PrimaryModelID         string  `json:"primary_model_id"`
	FallbackModelID        string  `json:"fallback_model_id"`
	MaxTokensPerRequest    int     `json:"max_tokens_per_request"`
	Temperature            float64 `json:"temperature"`
	TimeoutSeconds         int     `json:"timeout_seconds"`
	RetryCount             int     `json:"retry_count"`
	LogRetentionDays       int     `json:"log_retention_days"`
	DataRetentionDays      int     `json:"data_retention_days"`
	ContactName            string  `json:"contact_name"`
	ContactEmail           string  `json:"contact_email"`
	ContactPhone           string  `json:"contact_phone"`
	AdminUserID            string  `json:"admin_user_id"`
	CreatedAt              string  `json:"created_at"`
	UpdatedAt              string  `json:"updated_at"`
}

// DeleteTenantRequest 删除租户请求，需管理员密码确认以防误操作（DELETE /api/admin/tenants/:id）。
type DeleteTenantRequest struct {
	AdminPassword string `json:"admin_password" binding:"required"`
}

// TenantStatsResponse 租户统计数据响应（GET /api/admin/tenants/:id/stats）。
type TenantStatsResponse struct {
	TenantID        string `json:"tenant_id"`
	MemberCount     int64  `json:"member_count"`
	DepartmentCount int64  `json:"department_count"`
	RoleCount       int64  `json:"role_count"`
}

// PublicTenantItem 公共登录页面展示的轻量级租户条目。
type PublicTenantItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}

// TenantMemberItem 系统管理员查看租户成员列表的响应条目。
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

// RotateEmbedTokenResponse 生成或重置租户嵌入密钥后的返回。
// access_token 仅在本次接口调用时返回一次，后续不会再次展示明文。
type RotateEmbedTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenHint   string `json:"token_hint"`
	RotatedAt   string `json:"rotated_at"`
}

// OAJumpConfigResponse 租户 OA 流程跳转配置响应。
type OAJumpConfigResponse struct {
	Enabled            bool   `json:"enabled"`
	OABaseURL          string `json:"oa_base_url"`
	ProcessURLTemplate string `json:"process_url_template"`
	ResolvedTemplate   string `json:"resolved_template"`
}

