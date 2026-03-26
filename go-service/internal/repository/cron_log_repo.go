package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// CronLogRepo 提供 cron_logs 表的数据访问方法。
type CronLogRepo struct {
	db *gorm.DB
}

// NewCronLogRepo 创建一个新的 CronLogRepo 实例。
func NewCronLogRepo(db *gorm.DB) *CronLogRepo {
	return &CronLogRepo{db: db}
}

// Create 写入一条新的执行日志。
func (r *CronLogRepo) Create(log *model.CronLog) error {
	return r.db.Create(log).Error
}

// ListByTask 查询指定任务最近 N 条日志（按 started_at DESC）。
func (r *CronLogRepo) ListByTask(taskID uuid.UUID, limit int) ([]model.CronLog, error) {
	if limit <= 0 {
		limit = 20
	}
	var logs []model.CronLog
	err := r.db.Where("task_id = ?", taskID).
		Order("started_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// ListByTenant 查询租户最近 N 条日志（按 started_at DESC）。
func (r *CronLogRepo) ListByTenant(tenantID uuid.UUID, limit int) ([]model.CronLog, error) {
	if limit <= 0 {
		limit = 50
	}
	var logs []model.CronLog
	err := r.db.Where("tenant_id = ?", tenantID).
		Order("started_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// Finish 更新指定日志的状态和结束时间。
func (r *CronLogRepo) Finish(id uuid.UUID, status, message string) error {
	now := time.Now()
	return r.db.Model(&model.CronLog{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      status,
			"message":     message,
			"finished_at": &now,
		}).Error
}
