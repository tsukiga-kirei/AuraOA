package model

import (
	"time"
)

// CronTaskTypePreset 定时任务类型系统预设（全局，不绑定租户）。
// 共 6 条预置数据，涵盖审核工作台和归档复盘两个模块。
type CronTaskTypePreset struct {
	TaskType        string    `gorm:"primaryKey;size:50" json:"task_type"`                    // 任务类型唯一键（如 audit_batch / archive_daily）
	Module          string    `gorm:"size:20;not null;default:audit" json:"module"`            // 所属模块：audit=审核工作台，archive=归档复盘
	LabelZh         string    `gorm:"size:200;not null" json:"label_zh"`                       // 中文显示名称
	LabelEn         string    `gorm:"size:200;not null;default:''" json:"label_en"`            // 英文显示名称
	DescriptionZh   string    `gorm:"size:500;not null;default:''" json:"description_zh"`      // 中文描述
	DescriptionEn   string    `gorm:"size:500;not null;default:''" json:"description_en"`      // 英文描述
	DefaultCron     string    `gorm:"size:100;not null;default:''" json:"default_cron"`        // 建议的默认 Cron 表达式
	PushFormat      string    `gorm:"size:20;not null;default:html" json:"push_format"`        // 默认推送格式：html/markdown/plain
	ContentTemplate []byte    `gorm:"type:jsonb;not null;default:'{}'" json:"content_template"` // 默认内容模板（JSON）
	SortOrder       int       `gorm:"not null;default:0" json:"sort_order"`                    // 页面展示排序序号
	CreatedAt       time.Time `json:"created_at"`                                              // 创建时间
	UpdatedAt       time.Time `json:"updated_at"`                                              // 最后更新时间
}

func (CronTaskTypePreset) TableName() string { return "cron_task_type_presets" }
