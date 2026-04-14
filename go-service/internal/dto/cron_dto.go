package dto

import (
	"time"

	"gorm.io/datatypes"
)

// ===================== Cron 任务类型预设 DTO =====================

// CronTaskTypePresetResponse 系统预设任务类型响应（前端展示用）。
type CronTaskTypePresetResponse struct {
	TaskType        string         `json:"task_type"`
	Module          string         `json:"module"`
	LabelZh         string         `json:"label_zh"`
	LabelEn         string         `json:"label_en"`
	DescriptionZh   string         `json:"description_zh"`
	DescriptionEn   string         `json:"description_en"`
	DefaultCron     string         `json:"default_cron"`
	PushFormat      string         `json:"push_format"`
	ContentTemplate datatypes.JSON `json:"content_template"`
	SortOrder       int            `json:"sort_order"`
}

// CronTaskTypeConfigResponse 合并后的 Cron 任务类型配置响应（预设 + 租户覆盖）。
type CronTaskTypeConfigResponse struct {
	TaskType      string `json:"task_type"`
	Module        string `json:"module"`
	LabelZh       string `json:"label_zh"`
	LabelEn       string `json:"label_en"`
	DescriptionZh string `json:"description_zh"`
	DescriptionEn string `json:"description_en"`
	// 预设默认值（供"恢复默认"使用）
	DefaultCron           string         `json:"default_cron"`
	PresetPushFormat      string         `json:"preset_push_format"`
	PresetContentTemplate datatypes.JSON `json:"preset_content_template"`
	SortOrder             int            `json:"sort_order"`
	// 租户当前配置（若未启用则为 null）
	IsEnabled       bool           `json:"is_enabled"`       // 租户是否已启用该任务类型
	PushFormat      string         `json:"push_format"`      // 当前生效的推送格式
	ContentTemplate datatypes.JSON `json:"content_template"` // 当前生效的内容模板
	BatchLimit      *int           `json:"batch_limit"`      // 当前批处理限制
}

// SaveCronTaskTypeConfigRequest 保存（启用/更新）定时任务类型配置请求。
type SaveCronTaskTypeConfigRequest struct {
	PushFormat      string         `json:"push_format"`
	ContentTemplate datatypes.JSON `json:"content_template"`
	BatchLimit      *int           `json:"batch_limit"`
}

// ===================== Cron 任务实例 DTO =====================

// CronTaskResponse 定时任务实例详情响应。
type CronTaskResponse struct {
	ID             string         `json:"id"`
	TenantID       string         `json:"tenant_id"`
	OwnerUserID    string         `json:"owner_user_id"`
	TaskType       string         `json:"task_type"`
	TaskLabel      string         `json:"task_label"`
	Module         string         `json:"module"`
	CronExpression string         `json:"cron_expression"`
	IsActive       bool           `json:"is_active"`
	IsBuiltin      bool           `json:"is_builtin"`
	PushEmail      string         `json:"push_email"`
	LastRunAt      *time.Time     `json:"last_run_at"`
	NextRunAt      *time.Time     `json:"next_run_at"`
	SuccessCount   int            `json:"success_count"`
	FailCount      int            `json:"fail_count"`
	WorkflowIds    datatypes.JSON `json:"workflow_ids"`
	DateRange      int            `json:"date_range"`
	CurrentLogID   *string        `json:"current_log_id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
}

// CreateCronTaskRequest 创建定时任务实例请求。
type CreateCronTaskRequest struct {
	TaskType       string         `json:"task_type" binding:"required"`
	TaskLabel      string         `json:"task_label"`
	CronExpression string         `json:"cron_expression" binding:"required"`
	PushEmail      string         `json:"push_email"`
	WorkflowIds    datatypes.JSON `json:"workflow_ids"` // 关联的流程 ID 列表
	DateRange      int            `json:"date_range"`   // 数据范围天数：30 / 90 / 365
}

// UpdateCronTaskRequest 更新定时任务实例请求。
// PushEmail 使用指针：nil 表示不修改，"" 表示清空，非空字符串表示设置新值。
type UpdateCronTaskRequest struct {
	TaskLabel      string          `json:"task_label"`
	CronExpression string          `json:"cron_expression"`
	PushEmail      *string         `json:"push_email"`
	WorkflowIds    *datatypes.JSON `json:"workflow_ids"`
	DateRange      *int            `json:"date_range"`
}
