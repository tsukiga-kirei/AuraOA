package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// CronLogFilter 定时任务日志分页查询过滤条件。
type CronLogFilter struct {
	Keyword     string // 任务名称模糊搜索
	Status      string
	TaskType    string
	TriggerType string // manual / scheduled
	CreatedBy   string // 创建人模糊搜索
	StartDate   *time.Time
	EndDate     *time.Time
}

// CronLogStats 定时任务日志统计。
type CronLogStats struct {
	Total   int64 `json:"total"`
	Success int64 `json:"success"`
	Failed  int64 `json:"failed"`
	Running int64 `json:"running"`
}

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

// ListPagedByTenant 数据管理页：分页查询租户所有任务日志，支持多维过滤。
func (r *CronLogRepo) ListPagedByTenant(tenantID uuid.UUID, filter CronLogFilter, page, pageSize int) ([]model.CronLog, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}

	base := r.db.Model(&model.CronLog{}).Where("tenant_id = ?", tenantID)
	base = applyCronLogFilter(base, filter)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []model.CronLog
	err := base.Order("started_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error
	return items, total, err
}

// CountStatsByTenant 统计租户所有任务日志各状态数量。
func (r *CronLogRepo) CountStatsByTenant(tenantID uuid.UUID) (*CronLogStats, error) {
	type row struct {
		Status string
		Cnt    int64
	}
	var rows []row
	err := r.db.Model(&model.CronLog{}).
		Select("status, COUNT(*) as cnt").
		Where("tenant_id = ?", tenantID).
		Group("status").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	stats := &CronLogStats{}
	for _, r := range rows {
		stats.Total += r.Cnt
		switch r.Status {
		case "success":
			stats.Success += r.Cnt
		case "failed":
			stats.Failed += r.Cnt
		case "running":
			stats.Running += r.Cnt
		}
	}
	return stats, nil
}

func applyCronLogFilter(db *gorm.DB, f CronLogFilter) *gorm.DB {
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		db = db.Where("task_label ILIKE ?", like)
	}
	if f.Status != "" {
		db = db.Where("status = ?", f.Status)
	}
	if f.TaskType != "" {
		db = db.Where("task_type = ?", f.TaskType)
	}
	if f.TriggerType != "" {
		db = db.Where("trigger_type = ?", f.TriggerType)
	}
	if f.CreatedBy != "" {
		db = db.Where("created_by ILIKE ?", "%"+f.CreatedBy+"%")
	}
	if f.StartDate != nil {
		db = db.Where("started_at >= ?", f.StartDate)
	}
	if f.EndDate != nil {
		db = db.Where("started_at <= ?", f.EndDate)
	}
	return db
}
