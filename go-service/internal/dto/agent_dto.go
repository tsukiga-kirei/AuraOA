package dto

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// AgentCatalogItemDTO 平台目录项
type AgentCatalogItemDTO struct {
	ToolCatalog  []SystemToolCatalogItem `json:"tool_catalog"`
	AgentCatalog []AgentDefinitionDTO    `json:"agent_catalog"`
	SkillCatalog []SkillItemDTO          `json:"skill_catalog"`
}

// SystemToolCatalogItem 系统工具目录定义
type SystemToolCatalogItem struct {
	ToolCode    string `json:"tool_code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	UIKind      string `json:"ui_kind"`
	OARequired  bool   `json:"oa_required"`
	Risk        string `json:"risk"` // read | assist | write
	Parameters  string `json:"parameters"`
}

// TenantChatAllocationDTO 租户配额 DTO
type TenantChatAllocationDTO struct {
	TenantID          uuid.UUID  `json:"tenant_id"`
	ChatEnabled       bool       `json:"chat_enabled"`
	ChatRetentionDays int        `json:"chat_retention_days"`
	PrimaryModelID    *uuid.UUID `json:"chat_primary_model_id"`
	FallbackModelID   *uuid.UUID `json:"chat_fallback_model_id"`
	AgentCodes        []string   `json:"agent_codes"`
	ToolCodes         []string   `json:"tool_codes"`
	SkillCodes        []string   `json:"skill_codes"`
	AllowCustomSkills bool       `json:"allow_custom_skills"`
	AllowTenantMCP    bool       `json:"allow_tenant_mcp"`
	MaxMCPServers     int        `json:"max_mcp_servers"`
	MCPTemplateIDs    []string   `json:"mcp_template_ids"`
}

// UpdateTenantChatAllocationRequest 系统管理员设置租户配额请求
type UpdateTenantChatAllocationRequest struct {
	ChatEnabled       *bool           `json:"chat_enabled"`
	ChatRetentionDays *int            `json:"chat_retention_days"`
	PrimaryModelID    json.RawMessage `json:"chat_primary_model_id"`
	FallbackModelID   json.RawMessage `json:"chat_fallback_model_id"`
	AgentCodes        []string        `json:"agent_codes"`
	ToolCodes         []string        `json:"tool_codes"`
	SkillCodes        []string        `json:"skill_codes"`
	AllowCustomSkills *bool           `json:"allow_custom_skills"`
	AllowTenantMCP    *bool           `json:"allow_tenant_mcp"`
	MaxMCPServers     *int            `json:"max_mcp_servers"`
	MCPTemplateIDs    []string        `json:"mcp_template_ids"`
}

// AgentDefinitionDTO 智能体 DTO
type AgentDefinitionDTO struct {
	ID           uuid.UUID  `json:"id"`
	TenantID     *uuid.UUID `json:"tenant_id,omitempty"`
	AgentCode    string     `json:"agent_code"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	SystemPrompt string     `json:"system_prompt"`
	Enabled      bool       `json:"enabled"`
	IsSystem     bool       `json:"is_system"`
	ToolCodes    []string   `json:"tool_codes"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// CreateAgentRequest 创建智能体请求
type CreateAgentRequest struct {
	AgentCode    string   `json:"agent_code" binding:"required"`
	Name         string   `json:"name" binding:"required"`
	Description  string   `json:"description"`
	SystemPrompt string   `json:"system_prompt"`
	Enabled      bool     `json:"enabled"`
	ToolCodes    []string `json:"tool_codes"`
}

// UpdateAgentRequest 更新智能体请求
type UpdateAgentRequest struct {
	Name         *string   `json:"name"`
	Description  *string   `json:"description"`
	SystemPrompt *string   `json:"system_prompt"`
	Enabled      *bool     `json:"enabled"`
	ToolCodes    *[]string `json:"tool_codes"`
}

// MCPServerDTO MCP 服务器 DTO
type MCPServerDTO struct {
	ID            uuid.UUID      `json:"id"`
	ServerCode    string         `json:"server_code"`
	Name          string         `json:"name"`
	Description   string         `json:"description"`
	TransportType string         `json:"transport_type"`
	EndpointURL   string         `json:"endpoint_url"`
	Enabled       bool           `json:"enabled"`
	CachedTools   datatypes.JSON `json:"cached_tools"`
	LastSyncedAt  *time.Time     `json:"last_synced_at,omitempty"`
	AgentCodes    []string       `json:"agent_codes"`
	CreatedAt     time.Time      `json:"created_at"`
}

// SaveMCPServerRequest 创建/更新 MCP 服务器请求
type SaveMCPServerRequest struct {
	ID            uuid.UUID `json:"-"`
	ServerCode    string    `json:"server_code" binding:"required"`
	Name          string    `json:"name" binding:"required"`
	Description   string    `json:"description"`
	TransportType string    `json:"transport_type"`
	EndpointURL   string    `json:"endpoint_url" binding:"required"`
	Headers       string    `json:"headers"` // 自定义 headers 字符串或 JSON，后端加密
	Enabled       bool      `json:"enabled"`
	AgentCodes    []string  `json:"agent_codes"`
}

// SkillItemDTO Skill DTO
type SkillItemDTO struct {
	ID          uuid.UUID `json:"id"`
	SkillCode   string    `json:"skill_code"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	Enabled     bool      `json:"enabled"`
	IsSystem    bool      `json:"is_system"`
	AgentCodes  []string  `json:"agent_codes"`
	CreatedAt   time.Time `json:"created_at"`
}

// SaveSkillRequest 保存 Skill 请求
type SaveSkillRequest struct {
	ID          uuid.UUID `json:"-"`
	SkillCode   string    `json:"skill_code" binding:"required"`
	Name        string    `json:"name" binding:"required"`
	Description string    `json:"description"`
	Content     string    `json:"content" binding:"required"`
	Enabled     bool      `json:"enabled"`
	AgentCodes  []string  `json:"agent_codes"`
}

// AgentUsageStatsDTO 数据管理页智能体用量
type AgentUsageStatsDTO struct {
	SessionCount   int64               `json:"session_count"`
	MessageCount   int64               `json:"message_count"`
	ToolCallCount  int64               `json:"tool_call_count"`
	MCPCallCount   int64               `json:"mcp_call_count"`
	SkillCallCount int64               `json:"skill_call_count"`
	Agents         []AgentUsageItemDTO `json:"agents"`
}

// AgentUsageItemDTO 单个智能体的会话、工具与 Token 统计
type AgentUsageItemDTO struct {
	AgentCode      string   `json:"agent_code"`
	AgentName      string   `json:"agent_name"`
	SessionCount   int64    `json:"session_count"`
	MessageCount   int64    `json:"message_count"`
	TokenCount     int64    `json:"token_count"`
	ToolCodes      []string `json:"tool_codes"`
	MCPCodes       []string `json:"mcp_codes"`
	SkillCodes     []string `json:"skill_codes"`
	ToolCallCount  int64    `json:"tool_call_count"`
	MCPCallCount   int64    `json:"mcp_call_count"`
	SkillCallCount int64    `json:"skill_call_count"`
}
