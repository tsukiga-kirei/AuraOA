package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// CronTask 定时任务实例。
type CronTask struct {
	ID             uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID       uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	OwnerUserID    uuid.UUID      `gorm:"type:uuid;not null" json:"owner_user_id"`
	TaskType       string         `gorm:"size:50;not null" json:"task_type"`
	TaskLabel      string         `gorm:"size:200;not null;default:''" json:"task_label"`
	CronExpression string         `gorm:"size:100;not null" json:"cron_expression"`
	IsActive       bool           `gorm:"not null;default:true" json:"is_active"`
	IsBuiltin      bool           `gorm:"not null;default:false" json:"is_builtin"`
	PushEmail      string         `gorm:"size:255;default:''" json:"push_email"`
	LastRunAt      *time.Time     `json:"last_run_at"`
	NextRunAt      *time.Time     `json:"next_run_at"`
	SuccessCount   int            `gorm:"not null;default:0" json:"success_count"`
	FailCount      int            `gorm:"not null;default:0" json:"fail_count"`
	WorkflowIds    datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'" json:"workflow_ids"` // 流程多选
	DateRange      int            `gorm:"not null;default:30" json:"date_range"`                // 日期范围（天）
	CurrentLogID   *uuid.UUID     `gorm:"type:uuid" json:"current_log_id"`                      // 当前运行中的日志 ID
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

func (CronTask) TableName() string { return "cron_tasks" }

// CronTaskTypeConfig 定时任务类型配置（租户覆盖层）。
// 有记录即表示该租户已启用该任务类型配置；删除记录即表示关闭/恢复默认。
type CronTaskTypeConfig struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"` // 主键UUID
	TenantID        uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`                      // 所属租户ID
	TaskType        string         `gorm:"size:50;not null" json:"task_type"`                        // 关联预设类型编码（关联 cron_task_type_presets.task_type）
	BatchLimit      *int           `json:"batch_limit"`                                              // 单次批处理数量上限（NULL 表示使用预设默认值）
	PushFormat      string         `gorm:"size:20;not null;default:html" json:"push_format"`         // 租户自定义推送格式：html/markdown/plain
	ContentTemplate datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"content_template"` // 租户自定义内容模板
	CreatedAt       time.Time      `json:"created_at"`                                               // 创建时间
	UpdatedAt       time.Time      `json:"updated_at"`                                               // 最后更新时间
}

func (CronTaskTypeConfig) TableName() string { return "cron_task_type_configs" }
