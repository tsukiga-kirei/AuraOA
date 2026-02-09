package cron

import (
	"context"
	"time"
)

type CronTaskInput struct {
	UserID         string `json:"user_id" binding:"required"`
	CronExpression string `json:"cron_expression" binding:"required"`
	TaskType       string `json:"task_type" binding:"required"` // batch_audit | daily_report | weekly_report
}

type CronTask struct {
	ID             string     `json:"id"`
	UserID         string     `json:"user_id"`
	CronExpression string     `json:"cron_expression"`
	TaskType       string     `json:"task_type"`
	IsActive       bool       `json:"is_active"`
	LastRunAt      *time.Time `json:"last_run_at,omitempty"`
	NextRunAt      *time.Time `json:"next_run_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type TaskResult struct {
	TaskID    string `json:"task_id"`
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	ItemCount int    `json:"item_count"`
}

// CronScheduler manages user-defined scheduled tasks.
type CronScheduler interface {
	CreateTask(ctx context.Context, task CronTaskInput) (CronTask, error)
	DeleteTask(ctx context.Context, taskID string) error
	ListTasks(ctx context.Context, userID string) ([]CronTask, error)
	ExecuteTask(ctx context.Context, taskID string) (TaskResult, error)
}
