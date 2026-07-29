package model

import (
	"time"

	"github.com/google/uuid"
)

// EmbedRefreshSchedule OA 嵌入审核/总结的流程级持久化调度记录。
type EmbedRefreshSchedule struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	TenantID        uuid.UUID  `gorm:"type:uuid;not null" json:"tenant_id"`
	Module          string     `gorm:"size:20;not null" json:"module"`
	ConfigID        uuid.UUID  `gorm:"type:uuid;not null" json:"config_id"`
	ProcessType     string     `gorm:"size:200;not null" json:"process_type"`
	IsActive        bool       `gorm:"not null;default:true" json:"is_active"`
	LookbackDays    int        `gorm:"not null;default:3" json:"lookback_days"`
	IntervalMinutes int        `gorm:"not null;default:5" json:"interval_minutes"`
	CronExpression  string     `gorm:"size:100;not null" json:"cron_expression"`
	LastRunAt       *time.Time `json:"last_run_at"`
	NextRunAt       *time.Time `json:"next_run_at"`
	LastStatus      string     `gorm:"size:20;not null;default:''" json:"last_status"`
	LastError       string     `gorm:"type:text;not null;default:''" json:"last_error"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

func (EmbedRefreshSchedule) TableName() string { return "embed_refresh_schedules" }
