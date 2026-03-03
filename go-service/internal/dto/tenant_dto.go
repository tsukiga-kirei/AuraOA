package dto

//CreateTenantRequest 是 POST /api/admin/tenants 的请求正文
type CreateTenantRequest struct {
	Name           string      `json:"name" binding:"required"`
	Code           string      `json:"code" binding:"required"`
	Description    string      `json:"description"`
	OAType         string      `json:"oa_type"`
	TokenQuota     int         `json:"token_quota"`
	MaxConcurrency int         `json:"max_concurrency"`
	AIConfig       interface{} `json:"ai_config"`
	ContactName    string      `json:"contact_name"`
	ContactEmail   string      `json:"contact_email"`
	ContactPhone   string      `json:"contact_phone"`
}

//UpdateTenantRequest 是 PUT /api/admin/tenants/:id 的请求正文
type UpdateTenantRequest struct {
	Name              string      `json:"name"`
	Status            string      `json:"status"`
	Description       string      `json:"description"`
	OAType            string      `json:"oa_type"`
	TokenQuota        int         `json:"token_quota"`
	MaxConcurrency    int         `json:"max_concurrency"`
	AIConfig          interface{} `json:"ai_config"`
	SSOEnabled        *bool       `json:"sso_enabled"`
	SSOEndpoint       string      `json:"sso_endpoint"`
	LogRetentionDays  int         `json:"log_retention_days"`
	DataRetentionDays int         `json:"data_retention_days"`
	AllowCustomModel  *bool       `json:"allow_custom_model"`
	ContactName       string      `json:"contact_name"`
	ContactEmail      string      `json:"contact_email"`
	ContactPhone      string      `json:"contact_phone"`
}

//TenantResponse 是租户端点的响应正文
type TenantResponse struct {
	ID                string      `json:"id"`
	Name              string      `json:"name"`
	Code              string      `json:"code"`
	Description       string      `json:"description"`
	Status            string      `json:"status"`
	OAType            string      `json:"oa_type"`
	TokenQuota        int         `json:"token_quota"`
	TokenUsed         int         `json:"token_used"`
	MaxConcurrency    int         `json:"max_concurrency"`
	AIConfig          interface{} `json:"ai_config"`
	SSOEnabled        bool        `json:"sso_enabled"`
	SSOEndpoint       string      `json:"sso_endpoint"`
	LogRetentionDays  int         `json:"log_retention_days"`
	DataRetentionDays int         `json:"data_retention_days"`
	AllowCustomModel  bool        `json:"allow_custom_model"`
	ContactName       string      `json:"contact_name"`
	ContactEmail      string      `json:"contact_email"`
	ContactPhone      string      `json:"contact_phone"`
	CreatedAt         string      `json:"created_at"`
	UpdatedAt         string      `json:"updated_at"`
}

//TenantStatsResponse 是 GET /api/admin/tenants/:id/stats 的响应正文
type TenantStatsResponse struct {
	TenantID        string `json:"tenant_id"`
	MemberCount     int64  `json:"member_count"`
	DepartmentCount int64  `json:"department_count"`
	RoleCount       int64  `json:"role_count"`
}

//PublicTenantItem 是公共登录页面的轻量级租户条目。
type PublicTenantItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}
