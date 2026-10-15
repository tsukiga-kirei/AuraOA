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

// AuditDataQueryDefinition 定义查询输出、参数默认取值和允许使用的表，不保存具体流程实例值。
type AuditDataQueryDefinition struct {
	Mode        string                    `json:"mode,omitempty"` // config / sql；旧版本按 builder 推断。
	AIContext   *AuditDataQueryAIContext  `json:"ai_context,omitempty"`
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	SQL         string                    `json:"sql"`
	Dialect     string                    `json:"dialect"`
	Tables      []string                  `json:"tables"`
	Parameters  []AuditDataQueryParameter `json:"parameters"`
	MaxRows     int                       `json:"max_rows"`            // 与 MaxChars 互斥；均为 0 时不限制结果大小。
	MaxChars    int                       `json:"max_chars,omitempty"` // 数据 JSON 的字符上限，达到上限时保留完整末行。
	// Builder 仅在配置模式保存；自定义 SQL 模式清除配置，执行始终使用经过检查的 SQL。
	Builder json.RawMessage `json:"builder,omitempty"`
}

// AuditDataQueryAIContext 保存业务模块、补充口径及字段说明，不保存连接凭据和预览业务行。
type AuditDataQueryAIContext struct {
	OAModule   string            `json:"oa_module"` // general / workflow / model
	Notes      string            `json:"notes"`
	FieldNotes map[string]string `json:"field_notes,omitempty"` // 表名.字段名 → 业务含义
}

// AuditDataQueryParameter 定义命名参数的类型和中文含义。
type AuditDataQueryParameter struct {
	Name  string `json:"name"`
	Label string `json:"label"`
	Type  string `json:"type"` // string / number / date / string_list
	// DefaultBinding 保存固定值或系统值；流程字段允许留待规则引用时选择。
	DefaultBinding *AuditDataQueryBinding `json:"default_binding,omitempty"`
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
	SystemConstraints string    `gorm:"-" json:"system_constraints"` // 服务端固定生成约束，只读展示，不持久化客户端输入
	TenantID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"tenant_id"`
	Generate          string    `json:"generate"`
	Explain           string    `json:"explain"`
	Repair            string    `json:"repair"`
	UpdatedAt         time.Time `json:"updated_at"`
}
