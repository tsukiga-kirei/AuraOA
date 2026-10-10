package model

import (
	"encoding/json"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"time"
)

// AuditDataQuery 是租户内复用的审核取数定义，草稿与不可变发布版本分离。
type AuditDataQuery struct {
	ID            uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID      uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	Name          string         `json:"name"`
	Category      string         `json:"category"`
	Draft         datatypes.JSON `gorm:"type:jsonb" json:"draft"`
	LatestVersion int            `json:"latest_version"`
	TestedHash    string         `json:"tested_hash"`
	TestedAt      *time.Time     `json:"tested_at"`
	Archived      bool           `json:"archived"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

// AuditDataQueryVersion 保存发布时的查询和数据说明，不随草稿修改。
type AuditDataQueryVersion struct {
	ID         uuid.UUID      `gorm:"type:uuid;primaryKey" json:"id"`
	TenantID   uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	QueryID    uuid.UUID      `gorm:"type:uuid;not null" json:"query_id"`
	VersionNo  int            `json:"version_no"`
	Definition datatypes.JSON `gorm:"type:jsonb" json:"definition"`
	CreatedAt  time.Time      `json:"created_at"`
}

// AuditDataQueryDefinition 定义查询输出、参数和允许使用的表，不保存流程特定值。
type AuditDataQueryDefinition struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	SQL         string                    `json:"sql"`
	Dialect     string                    `json:"dialect"`
	Tables      []string                  `json:"tables"`
	Parameters  []AuditDataQueryParameter `json:"parameters"`
	MaxRows     int                       `json:"max_rows"`
	// Builder 仅保存可视化配置以便再次编辑；执行始终使用经过检查的 SQL。
	Builder json.RawMessage `json:"builder,omitempty"`
}

// AuditDataQueryParameter 定义命名参数的类型和中文含义。
type AuditDataQueryParameter struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Type  string `json:"type"` // string / number / date / string_list
}

// AuditDataQueryReference 将发布版本参数绑定到当前流程字段或系统变量。
type AuditDataQueryReference struct {
	QueryID   string                           `json:"query_id"`
	VersionNo int                              `json:"version_no"`
	Enabled   bool                             `json:"enabled"`
	Bindings  map[string]AuditDataQueryBinding `json:"bindings"`
}

// AuditDataQueryBinding 的 field 使用 main:字段或明细表:字段，系统值由服务端解析。
type AuditDataQueryBinding struct {
	Source     string `json:"source"` // field / system / constant
	Field      string `json:"field,omitempty"`
	Value      string `json:"value,omitempty"`
	DaysOffset int    `json:"days_offset,omitempty"`
}

// AuditDataQueryPrompts 保存可编辑的租户级 AI SQL 辅助提示词。
type AuditDataQueryPrompts struct {
	TenantID  uuid.UUID `gorm:"type:uuid;primaryKey" json:"tenant_id"`
	Generate  string    `json:"generate"`
	Explain   string    `json:"explain"`
	Repair    string    `json:"repair"`
	UpdatedAt time.Time `json:"updated_at"`
}
