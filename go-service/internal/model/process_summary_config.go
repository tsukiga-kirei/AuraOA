package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// ProcessSummaryConfig 流程总结配置，租户级别定义。
type ProcessSummaryConfig struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID         uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	ProcessType      string         `gorm:"size:200;not null" json:"process_type"`
	ProcessTypeLabel string         `gorm:"size:200;default:''" json:"process_type_label"`
	MainTableName    string         `gorm:"size:200;default:''" json:"main_table_name"`
	MainFields       datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"main_fields"`
	DetailTables     datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"detail_tables"`
	SummaryBlocks    datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"summary_blocks"`
	EmbedEnabled     bool           `gorm:"not null;default:false" json:"embed_enabled"`
	EmbedConfig      datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"embed_config"`
	Status           string         `gorm:"size:20;not null;default:active" json:"status"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

func (ProcessSummaryConfig) TableName() string { return "process_summary_configs" }

// SummaryBlockConfig 单个总结块配置。
type SummaryBlockConfig struct {
	ID                   string                 `json:"id"`
	Title                string                 `json:"title"`
	UserPrompt           string                 `json:"user_prompt"`
	IncludeMeta          *bool                  `json:"include_meta"`           // true：传入全部数据（固定模板）；false：仅传入 enabled_data_variables
	EnabledDataVariables []string               `json:"enabled_data_variables"` // include_meta=false 时生效
	FieldMode            string                 `json:"field_mode"`             // all | selected
	SelectedFields       []string               `json:"selected_fields"`
	ContextMounts        []ExternalContextMount `json:"context_mounts,omitempty"`
	Enabled              bool                   `json:"enabled"`
	SortOrder            int                    `json:"sort_order"`
}

// SummaryEmbedConfigData OA 嵌入总结页行为配置。
type SummaryEmbedConfigData struct {
	AutoSummaryOnOpen           bool `json:"auto_summary_on_open"`
	AutoSummaryOnStale          bool `json:"auto_summary_on_stale,omitempty"` // 兼容旧配置，不再直接用于决策
	AutoSummaryOnDataChange     bool `json:"auto_summary_on_data_change"`
	AutoSummaryOnReturnResubmit bool `json:"auto_summary_on_return_resubmit"`
	AutoSummaryOnFlowChange     bool `json:"auto_summary_on_flow_change"`
	ScheduledRefreshEnabled     bool `json:"scheduled_refresh_enabled"`
	ScheduledLookbackDays       int  `json:"scheduled_refresh_lookback_days"`
	ScheduledIntervalMinutes    int  `json:"scheduled_refresh_interval_minutes"`
}

// ProcessSummaryLog 总结执行日志。
type ProcessSummaryLog struct {
	ID                 uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID           uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	UserID             uuid.UUID      `gorm:"type:uuid;not null" json:"user_id"`
	ProcessID          string         `gorm:"size:100;not null" json:"process_id"`
	Title              string         `gorm:"size:500;not null;default:''" json:"title"`
	ProcessType        string         `gorm:"size:200;not null;default:''" json:"process_type"`
	Status             string         `gorm:"size:20;not null;default:completed" json:"status"`
	SummaryResult      datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"summary_result"`
	ProcessSnapshot    datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"process_snapshot"`
	DurationMs         int            `gorm:"not null;default:0" json:"duration_ms"`
	RawContent         string         `gorm:"type:text;default:''" json:"raw_content"`
	ParseError         string         `gorm:"type:text;default:''" json:"parse_error"`
	ErrorMessage       string         `gorm:"type:text;default:''" json:"error_message"`
	TriggerSource      string         `gorm:"size:30;not null;default:summary_embed_manual" json:"trigger_source"`
	TriggerDetail      string         `gorm:"size:30;not null;default:''" json:"trigger_detail"`
	QueueKind          string         `gorm:"size:20;not null;default:background" json:"queue_kind"`
	AttemptFingerprint string         `gorm:"size:80;not null;default:''" json:"attempt_fingerprint"`
	ScheduleConfigID   *uuid.UUID     `gorm:"type:uuid" json:"schedule_config_id,omitempty"`
	OAContextAnchor    datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"oa_context_anchor"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `gorm:"not null;default:now()" json:"updated_at"`
}

func (ProcessSummaryLog) TableName() string { return "process_summary_logs" }

const (
	SummaryTriggerEmbedAuto   = "summary_embed_auto"
	SummaryTriggerEmbedManual = "summary_embed_manual"

	SummaryTriggerDetailManual      = "manual"
	SummaryTriggerDetailVisibleOpen = "visible_open"
	SummaryTriggerDetailSaveSubmit  = "save_or_submit"
	SummaryTriggerDetailScheduled   = "scheduled_scan"
)

// ProcessSummarySnapshot 流程级有效总结快照。
type ProcessSummarySnapshot struct {
	ID               uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID         uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	ProcessID        string         `gorm:"size:100;not null" json:"process_id"`
	ValidLogIDs      datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"valid_log_ids"`
	LatestValidLogID uuid.UUID      `gorm:"type:uuid;not null" json:"latest_valid_log_id"`
	Title            string         `gorm:"size:500;not null;default:''" json:"title"`
	ProcessType      string         `gorm:"size:200;not null;default:''" json:"process_type"`
	BlockCount       int            `gorm:"not null;default:0" json:"block_count"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `gorm:"not null;default:now()" json:"updated_at"`
}

func (ProcessSummarySnapshot) TableName() string { return "process_summary_snapshots" }

// ProcessSummaryResultJSON 是前后端共用的总结结果结构。
type ProcessSummaryResultJSON struct {
	Blocks []ProcessSummaryBlockResult `json:"blocks"`
}

type ProcessSummaryBlockResult struct {
	BlockID    string   `json:"block_id"`
	Title      string   `json:"title"`
	Content    string   `json:"content"`
	Points     []string `json:"points"`
	DurationMs int      `json:"duration_ms,omitempty"`
}
