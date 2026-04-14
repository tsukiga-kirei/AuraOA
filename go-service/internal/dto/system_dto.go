package dto

// ============================================================
// 通用选项 DTO
// ============================================================

// OptionItem 通用键值选项响应。
type OptionItem struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

// DBDriverOptionItem 数据库驱动选项响应，含默认端口。
type DBDriverOptionItem struct {
	Code        string `json:"code"`
	Label       string `json:"label"`
	DefaultPort int    `json:"default_port"`
}

// AIProviderOptionItem AI 服务商选项响应，含部署类型。
type AIProviderOptionItem struct {
	Code       string `json:"code"`
	Label      string `json:"label"`
	DeployType string `json:"deploy_type"`
}

// ============================================================
// OA 数据库连接 DTO
// ============================================================

// CreateOAConnectionRequest 创建 OA 数据库连接请求（POST /api/admin/system/oa-connections）。
type CreateOAConnectionRequest struct {
	Name              string `json:"name" binding:"required"`
	OAType            string `json:"oa_type" binding:"required"`
	OATypeLabel       string `json:"oa_type_label"`
	Driver            string `json:"driver" binding:"required"`
	Host              string `json:"host" binding:"required"`
	Port              int    `json:"port" binding:"required,min=1"`
	DatabaseName      string `json:"database_name" binding:"required"`
	Username          string `json:"username" binding:"required"`
	Password          string `json:"password" binding:"required"`
	PoolSize          int    `json:"pool_size"`
	ConnectionTimeout int    `json:"connection_timeout"`
	TestOnBorrow      bool   `json:"test_on_borrow"`
	SyncInterval      int    `json:"sync_interval"`
	Enabled           bool   `json:"enabled"`
	Description       string `json:"description"`
}

// UpdateOAConnectionRequest 更新 OA 数据库连接请求（PUT /api/admin/system/oa-connections/:id）。
type UpdateOAConnectionRequest struct {
	Name              string `json:"name"`
	OAType            string `json:"oa_type"`
	OATypeLabel       string `json:"oa_type_label"`
	Driver            string `json:"driver"`
	Host              string `json:"host"`
	Port              int    `json:"port"`
	DatabaseName      string `json:"database_name"`
	Username          string `json:"username"`
	Password          string `json:"password"`
	PoolSize          int    `json:"pool_size"`
	ConnectionTimeout int    `json:"connection_timeout"`
	TestOnBorrow      *bool  `json:"test_on_borrow"`
	SyncInterval      int    `json:"sync_interval"`
	Enabled           *bool  `json:"enabled"`
	Description       string `json:"description"`
}

// OAConnectionResponse OA 数据库连接详情响应。
type OAConnectionResponse struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	OAType            string `json:"oa_type"`
	OATypeLabel       string `json:"oa_type_label"`
	Driver            string `json:"driver"`
	Host              string `json:"host"`
	Port              int    `json:"port"`
	DatabaseName      string `json:"database_name"`
	Username          string `json:"username"`
	PoolSize          int    `json:"pool_size"`
	ConnectionTimeout int    `json:"connection_timeout"`
	TestOnBorrow      bool   `json:"test_on_borrow"`
	Status            string `json:"status"`
	SyncInterval      int    `json:"sync_interval"`
	Enabled           bool   `json:"enabled"`
	Description       string `json:"description"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

// ============================================================
// AI 模型配置 DTO
// ============================================================

// CreateAIModelRequest 创建 AI 模型配置请求（POST /api/admin/system/ai-models）。
type CreateAIModelRequest struct {
	Provider        string   `json:"provider" binding:"required"`
	ProviderLabel   string   `json:"provider_label"`
	ModelName       string   `json:"model_name" binding:"required"`
	DisplayName     string   `json:"display_name" binding:"required"`
	DeployType      string   `json:"deploy_type" binding:"required"`
	Endpoint        string   `json:"endpoint" binding:"required"`
	APIKey          string   `json:"api_key"`
	MaxTokens       int      `json:"max_tokens"`
	ContextWindow   int      `json:"context_window"`
	CostPer1kTokens float64  `json:"cost_per_1k_tokens"`
	Enabled         bool     `json:"enabled"`
	Description     string   `json:"description"`
	Capabilities    []string `json:"capabilities"`
}

// UpdateAIModelRequest 更新 AI 模型配置请求（PUT /api/admin/system/ai-models/:id）。
type UpdateAIModelRequest struct {
	Provider        string   `json:"provider"`
	ProviderLabel   string   `json:"provider_label"`
	ModelName       string   `json:"model_name"`
	DisplayName     string   `json:"display_name"`
	DeployType      string   `json:"deploy_type"`
	Endpoint        string   `json:"endpoint"`
	APIKey          string   `json:"api_key"`
	MaxTokens       int      `json:"max_tokens"`
	ContextWindow   int      `json:"context_window"`
	CostPer1kTokens *float64 `json:"cost_per_1k_tokens"`
	Enabled         *bool    `json:"enabled"`
	Status          string   `json:"status"`
	Description     string   `json:"description"`
	Capabilities    []string `json:"capabilities"`
}

// AIModelResponse AI 模型配置详情响应。
type AIModelResponse struct {
	ID               string   `json:"id"`
	Provider         string   `json:"provider"`
	ProviderLabel    string   `json:"provider_label"`
	ModelName        string   `json:"model_name"`
	DisplayName      string   `json:"display_name"`
	DeployType       string   `json:"deploy_type"`
	Endpoint         string   `json:"endpoint"`
	APIKeyConfigured bool     `json:"api_key_configured"`
	MaxTokens        int      `json:"max_tokens"`
	ContextWindow    int      `json:"context_window"`
	CostPer1kTokens  float64  `json:"cost_per_1k_tokens"`
	Status           string   `json:"status"`
	Enabled          bool     `json:"enabled"`
	Description      string   `json:"description"`
	Capabilities     []string `json:"capabilities"`
	CreatedAt        string   `json:"created_at"`
	UpdatedAt        string   `json:"updated_at"`
}
