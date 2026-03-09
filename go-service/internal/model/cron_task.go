package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// CronTask 定时任务。
type CronTask struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID       uuid.UUID  `gorm:"type:uuid;not null" json:"tenant_id"`
	TaskType       string     `gorm:"size:50;not null" json:"task_type"`
	TaskLabel      string     `gorm:"size:200;not null;default:''" json:"task_label"`
	CronExpression string     `gorm:"size:100;not null" json:"cron_expression"`
	IsActive       bool       `gorm:"not null;default:true" json:"is_active"`
	IsBuiltin      bool       `gorm:"not null;default:false" json:"is_builtin"`
	PushEmail      string     `gorm:"size:255;default:''" json:"push_email"`
	LastRunAt      *time.Time `json:"last_run_at"`
	NextRunAt      *time.Time `json:"next_run_at"`
	SuccessCount   int        `gorm:"not null;default:0" json:"success_count"`
	FailCount      int        `gorm:"not null;default:0" json:"fail_count"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func (CronTask) TableName() string { return "cron_tasks" }

// CronTaskTypeConfig 定时任务类型配置。
type CronTaskTypeConfig struct {
	ID              uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID        uuid.UUID      `gorm:"type:uuid;not null" json:"tenant_id"`
	TaskType        string         `gorm:"size:50;not null" json:"task_type"`
	Label           string         `gorm:"size:200;not null" json:"label"`
	Enabled         bool           `gorm:"not null;default:true" json:"enabled"`
	BatchLimit      *int           `json:"batch_limit"`
	PushFormat      string         `gorm:"size:20;not null;default:html" json:"push_format"`
	ContentTemplate datatypes.JSON `gorm:"type:jsonb;not null;default:'{}'" json:"content_template"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

func (CronTaskTypeConfig) TableName() string { return "cron_task_type_configs" }
