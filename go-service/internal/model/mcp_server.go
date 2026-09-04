package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// MCPServer MCP 外部服务器配置
type MCPServer struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID         *uuid.UUID     `gorm:"type:uuid;index" json:"tenant_id,omitempty"` // NULL 表示平台模板
	ServerCode       string         `gorm:"size:64;not null" json:"server_code"`
	Name             string         `gorm:"size:128;not null" json:"name"`
	Description      string         `gorm:"type:text" json:"description"`
	TransportType    string         `gorm:"size:32;not null;default:http" json:"transport_type"` // http | sse
	EndpointURL      string         `gorm:"type:text;not null" json:"endpoint_url"`
	HeadersEncrypted string         `gorm:"type:text" json:"-"`
	Enabled          bool           `gorm:"not null;default:true" json:"enabled"`
	CachedTools      datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"cached_tools"`
	LastSyncedAt     *time.Time     `json:"last_synced_at,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

func (MCPServer) TableName() string {
	return "mcp_servers"
}
