package dto

// CreateTenantRequest is the request body for POST /api/admin/tenants
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

// UpdateTenantRequest is the request body for PUT /api/admin/tenants/:id
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

// TenantResponse is the response body for tenant endpoints
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

// TenantStatsResponse is the response body for GET /api/admin/tenants/:id/stats
type TenantStatsResponse struct {
	TenantID        string `json:"tenant_id"`
	MemberCount     int64  `json:"member_count"`
	DepartmentCount int64  `json:"department_count"`
	RoleCount       int64  `json:"role_count"`
}

// PublicTenantItem is a lightweight tenant entry for the public login page.
type PublicTenantItem struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Code string `json:"code"`
}
