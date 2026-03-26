package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// 审核日志异步状态（与 DB 迁移 000013 一致）
const (
	AuditStatusPending    = "pending"
	AuditStatusAssembling = "assembling"
	AuditStatusReasoning  = "reasoning"
	AuditStatusExtracting = "extracting"
	AuditStatusCompleted  = "completed"
	AuditStatusFailed     = "failed"
)

// AuditLog 审核日志，记录每次审核执行的结果。
type AuditLog struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID       uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	UserID         uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
	ProcessID      string         `gorm:"size:100;not null" json:"process_id"`
	Title          string         `gorm:"size:500;not null" json:"title"`
	ProcessType    string         `gorm:"size:200;not null" json:"process_type"`
	Status         string         `gorm:"size:20;not null;default:completed" json:"status"`
	Recommendation string         `gorm:"size:20;not null" json:"recommendation"`
	Score          int            `gorm:"not null;default:0" json:"score"`
	AuditResult    datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"audit_result"`
	DurationMs     int            `gorm:"not null;default:0" json:"duration_ms"`
	AIReasoning    string         `gorm:"type:text;default:''" json:"ai_reasoning"`
	Confidence     int            `gorm:"not null;default:0" json:"confidence"`
	RawContent     string         `gorm:"type:text;default:''" json:"raw_content"`
	ParseError     string         `gorm:"type:text;default:''" json:"parse_error"`
	ErrorMessage   string         `gorm:"type:text;default:''" json:"error_message"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `gorm:"not null;default:now()" json:"updated_at"`
}

// AuditResultJSON 提取阶段 JSON Schema 的 Go 映射（固定结构，前后端共用）。
// 解析时若模型仅输出 overall_compliance（见 extractionPayload），会映射为 recommendation，不单独落库。
type AuditResultJSON struct {
	Recommendation string           `json:"recommendation"`
	OverallScore   int              `json:"overall_score"`
	RuleResults    []RuleResultJSON `json:"rule_results"`
	RiskPoints     []string         `json:"risk_points"`
	Suggestions    []string         `json:"suggestions"`
	Confidence     int              `json:"confidence"`
}

// RuleResultJSON 单条规则校验结果
type RuleResultJSON struct {
	RuleContent string `json:"rule_content"`
	Passed      bool   `json:"passed"`
	Reason      string `json:"reason"`
}

func (AuditLog) TableName() string { return "audit_logs" }

// CronLog 定时任务日志。
type CronLog struct {
	ID         uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID   uuid.UUID  `gorm:"type:uuid;not null" json:"tenant_id"`
	TaskID     uuid.UUID  `gorm:"type:uuid;not null" json:"task_id"`
	TaskType   string     `gorm:"size:50;not null" json:"task_type"`
	TaskLabel  string     `gorm:"size:200;not null;default:''" json:"task_label"`
	Status     string     `gorm:"size:20;not null;default:running" json:"status"`
	Message    string     `gorm:"type:text;default:''" json:"message"`
	StartedAt  time.Time  `gorm:"not null;default:now()" json:"started_at"`
	FinishedAt *time.Time `json:"finished_at"`
}

func (CronLog) TableName() string { return "cron_logs" }

// ArchiveLog 归档复盘日志。
type ArchiveLog struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID        uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	UserID          uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
	ProcessID       string         `gorm:"size:100;not null" json:"process_id"`
	Title           string         `gorm:"size:500;not null" json:"title"`
	ProcessType     string         `gorm:"size:200;not null" json:"process_type"`
	Status          string         `gorm:"size:20;not null;default:completed" json:"status"`
	Compliance      string         `gorm:"size:30;not null" json:"compliance"`
	ComplianceScore int            `gorm:"not null;default:0" json:"compliance_score"`
	ArchiveResult   datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"archive_result"`
	ProcessSnapshot datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"process_snapshot"`
	DurationMs      int            `gorm:"not null;default:0" json:"duration_ms"`
	AIReasoning     string         `gorm:"type:text;default:''" json:"ai_reasoning"`
	Confidence      int            `gorm:"not null;default:0" json:"confidence"`
	RawContent      string         `gorm:"type:text;default:''" json:"raw_content"`
	ParseError      string         `gorm:"type:text;default:''" json:"parse_error"`
	ErrorMessage    string         `gorm:"type:text;default:''" json:"error_message"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `gorm:"not null;default:now()" json:"updated_at"`
}

func (ArchiveLog) TableName() string { return "archive_logs" }

// ArchiveResultJSON 归档复盘提取结果。
type ArchiveResultJSON struct {
	OverallCompliance string                  `json:"overall_compliance"`
	OverallScore      int                     `json:"overall_score"`
	Confidence        int                     `json:"confidence"`
	FlowAudit         ArchiveFlowAuditJSON    `json:"flow_audit"`
	FieldAudit        []ArchiveFieldAuditJSON `json:"field_audit"`
	RuleAudit         []ArchiveRuleAuditJSON  `json:"rule_audit"`
	RiskPoints        []string                `json:"risk_points"`
	Suggestions       []string                `json:"suggestions"`
	AISummary         string                  `json:"ai_summary"`
}

type ArchiveFlowAuditJSON struct {
	IsComplete   bool                        `json:"is_complete"`
	MissingNodes []string                    `json:"missing_nodes"`
	NodeResults  []ArchiveFlowNodeResultJSON `json:"node_results"`
}

type ArchiveFlowNodeResultJSON struct {
	NodeID    string `json:"node_id"`
	NodeName  string `json:"node_name"`
	Compliant bool   `json:"compliant"`
	Reasoning string `json:"reasoning"`
}

type ArchiveFieldAuditJSON struct {
	FieldKey  string `json:"field_key"`
	FieldName string `json:"field_name"`
	Passed    bool   `json:"passed"`
	Reasoning string `json:"reasoning"`
}

type ArchiveRuleAuditJSON struct {
	RuleID    string `json:"rule_id"`
	RuleName  string `json:"rule_name"`
	Passed    bool   `json:"passed"`
	Reasoning string `json:"reasoning"`
}
