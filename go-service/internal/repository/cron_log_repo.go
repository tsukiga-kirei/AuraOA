package repository

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// CronLogFilter 定时任务日志分页查询过滤条件。
type CronLogFilter struct {
	Keyword     string // 任务名称模糊搜索
	Status      string
	TaskType    string
	TriggerType string // manual / scheduled
	CreatedBy   string // 触发人（created_by）模糊搜索
	Department  string // 部门精确匹配
	StartDate   *time.Time
	EndDate     *time.Time
	DateRange   *int   // 数据范围（天）
}

// CronLogStats 定时任务日志统计。
type CronLogStats struct {
	Total   int64 `json:"total"`
	Success int64 `json:"success"`
	Failed  int64 `json:"failed"`
	Running int64 `json:"running"`
}

// CronLogListRow 分页列表：日志 + 任务归属用户展示名 + 部门（LEFT JOIN users + org_members + departments）。
type CronLogListRow struct {
	model.CronLog
	TaskOwnerDisplayName string         `json:"task_owner_display_name" gorm:"column:task_owner_display_name"`
	Department           string         `json:"department" gorm:"column:department"`
	TaskTypeLabel        string         `json:"task_type_label" gorm:"column:task_type_label"`
	PushEmail            string         `json:"push_email" gorm:"column:push_email"`
	WorkflowIds          datatypes.JSON `json:"workflow_ids" gorm:"column:workflow_ids"`
	DateRange            int            `json:"date_range" gorm:"column:date_range"`
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

// ListByTenantForDashboardMember 业务用户仪表盘：仅归属当前用户的任务执行日志，或手动触发且 created_by 为本人登录名。
func (r *CronLogRepo) ListByTenantForDashboardMember(tenantID, memberUserID uuid.UUID, username string, limit int) ([]model.CronLog, error) {
	if limit <= 0 {
		limit = 50
	}
	var logs []model.CronLog
	err := r.db.Where("tenant_id = ?", tenantID).
		Where("(task_owner_user_id = ? OR (task_owner_user_id IS NULL AND created_by = ?))", memberUserID, username).
		Order("started_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// ListRecentGlobal 全库最近 N 条定时任务执行日志（按 started_at 倒序）。
func (r *CronLogRepo) ListRecentGlobal(limit int) ([]model.CronLog, error) {
	if limit <= 0 {
		limit = 50
	}
	var logs []model.CronLog
	err := r.db.
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

// ListPagedByTenant 数据管理页：分页查询租户所有任务日志，支持多维过滤（JOIN 归属用户展示名 + 部门）。
func (r *CronLogRepo) ListPagedByTenant(tenantID uuid.UUID, filter CronLogFilter, page, pageSize int) ([]CronLogListRow, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}

	base := r.db.Table("cron_logs").
		Select("cron_logs.*, "+
			"COALESCE(u.display_name, u.username, '') AS task_owner_display_name, "+
			"COALESCE(d.name, '') AS department, "+
			"COALESCE(p.label_zh, cron_logs.task_type) AS task_type_label, "+
			"COALESCE(ct.push_email, '') AS push_email, "+
			"COALESCE(ct.workflow_ids, '[]'::jsonb) AS workflow_ids, "+
			"COALESCE(ct.date_range, 0) AS date_range").
		Joins("LEFT JOIN users u ON u.id = cron_logs.task_owner_user_id").
		Joins("LEFT JOIN org_members om ON om.user_id = cron_logs.task_owner_user_id AND om.tenant_id = cron_logs.tenant_id AND om.status = 'active'").
		Joins("LEFT JOIN departments d ON d.id = om.department_id AND d.tenant_id = cron_logs.tenant_id").
		Joins("LEFT JOIN cron_task_type_presets p ON p.task_type = cron_logs.task_type").
		Joins("LEFT JOIN cron_tasks ct ON ct.id = cron_logs.task_id").
		Where("cron_logs.tenant_id = ?", tenantID)
	base = applyCronLogFilterJoined(base, filter)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []CronLogListRow
	err := base.Order("cron_logs.started_at DESC").
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

// applyCronLogFilterJoined 在 JOIN users 查询上使用，列名带 cron_logs. 前缀避免歧义。
func applyCronLogFilterJoined(db *gorm.DB, f CronLogFilter) *gorm.DB {
	const t = "cron_logs."
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		db = db.Where(t+"task_label ILIKE ?", like)
	}
	if f.Status != "" {
		db = db.Where(t+"status = ?", f.Status)
	}
	if f.TaskType != "" {
		db = db.Where(t+"task_type = ?", f.TaskType)
	}
	if f.TriggerType != "" {
		db = db.Where(t+"trigger_type = ?", f.TriggerType)
	}
	if f.CreatedBy != "" {
		db = db.Where(t+"created_by ILIKE ?", "%"+f.CreatedBy+"%")
	}
	if f.Department != "" {
		db = db.Where("d.name = ?", f.Department)
	}
	if f.StartDate != nil {
		db = db.Where(t+"started_at >= ?", f.StartDate)
	}
	if f.EndDate != nil {
		db = db.Where(t+"started_at <= ?", f.EndDate)
	}
	return db
}

// ── 仪表盘查询辅助类型 ──────────────────────────────────────────────────────

// CronLogEnrichedRow 带状态和任务标签的定时任务日志行（用于最近动态）。
type CronLogEnrichedRow struct {
	ID        uuid.UUID `gorm:"column:id"`
	TaskLabel string    `gorm:"column:task_label"`
	TaskType  string    `gorm:"column:task_type"`
	Status    string    `gorm:"column:status"`
	UserName  string    `gorm:"column:user_name"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

// TenantCronCount 按租户统计定时任务执行数。
type TenantCronCount struct {
	TenantID uuid.UUID `gorm:"column:tenant_id"`
	Count    int64     `gorm:"column:count"`
}

// ── 仪表盘查询方法 ──────────────────────────────────────────────────────────

// CountThisWeek 本周（周一 00:00 UTC 至今）定时任务执行次数。
// userID 非 nil 时按 task_owner_user_id 或 created_by（从 gin context 取 username）过滤。
func (r *CronLogRepo) CountThisWeek(c *gin.Context, userID *uuid.UUID) (int64, error) {
	tenantID, _ := c.Get("tenant_id")

	args := []interface{}{tenantID}
	userFilter := ""
	if userID != nil {
		username, _ := c.Get("username")
		userFilter = "AND (cl.task_owner_user_id = ? OR cl.created_by = ?)"
		args = append(args, *userID, username)
	}

	sql := `
SELECT COUNT(*)::bigint
FROM cron_logs cl
WHERE cl.tenant_id = ?
  AND cl.started_at >= date_trunc('week', CURRENT_TIMESTAMP AT TIME ZONE 'UTC')
  ` + userFilter

	var count int64
	err := r.db.Raw(sql, args...).Scan(&count).Error
	return count, err
}

// WeeklyTrendByDay 本周每天的定时任务执行次数（generate_series 填充无数据日期）。
func (r *CronLogRepo) WeeklyTrendByDay(c *gin.Context, userID *uuid.UUID) ([]DayCount, error) {
	tenantID, _ := c.Get("tenant_id")

	userFilter := ""
	args := []interface{}{tenantID}
	if userID != nil {
		username, _ := c.Get("username")
		userFilter = "AND (cl.task_owner_user_id = ? OR cl.created_by = ?)"
		args = append(args, *userID, username)
	}

	sql := `
WITH days AS (
  SELECT generate_series(
    date_trunc('week', CURRENT_DATE AT TIME ZONE 'UTC')::date,
    (CURRENT_DATE AT TIME ZONE 'UTC')::date,
    INTERVAL '1 day'
  )::date AS d
)
SELECT TO_CHAR(days.d, 'MM-DD') AS date,
       COALESCE(b.cnt, 0)::bigint AS count
FROM days
LEFT JOIN (
  SELECT DATE(cl.started_at AT TIME ZONE 'UTC') AS d,
         COUNT(*)::bigint AS cnt
  FROM cron_logs cl
  WHERE cl.tenant_id = ?
    AND cl.started_at >= date_trunc('week', CURRENT_TIMESTAMP AT TIME ZONE 'UTC')
    ` + userFilter + `
  GROUP BY 1
) b ON b.d = days.d
ORDER BY days.d`

	var rows []DayCount
	err := r.db.Raw(sql, args...).Scan(&rows).Error
	return rows, err
}

// RecentEnriched 最近 N 条定时任务日志（带 status + task_label + 操作人信息）。
// userID 非 nil 时按 task_owner_user_id 过滤。
func (r *CronLogRepo) RecentEnriched(tenantID uuid.UUID, limit int, userID *uuid.UUID) ([]CronLogEnrichedRow, error) {
	args := []interface{}{tenantID}
	userFilter := ""
	if userID != nil {
		userFilter = "AND cl.task_owner_user_id = ?"
		args = append(args, *userID)
	}
	args = append(args, limit)

	sql := `
SELECT cl.id,
       cl.task_label,
       cl.task_type,
       cl.status,
       COALESCE(u.display_name, u.username, cl.created_by) AS user_name,
       cl.started_at AS created_at
FROM cron_logs cl
LEFT JOIN users u ON u.id = cl.task_owner_user_id
WHERE cl.tenant_id = ?
  ` + userFilter + `
ORDER BY cl.started_at DESC
LIMIT ?`

	var rows []CronLogEnrichedRow
	err := r.db.Raw(sql, args...).Scan(&rows).Error
	return rows, err
}

// CountByDepartment 按部门统计定时任务执行数（通过 task_owner_user_id → org_members → departments）。
func (r *CronLogRepo) CountByDepartment(c *gin.Context) ([]DeptCount, error) {
	tenantID, _ := c.Get("tenant_id")

	sql := `
SELECT COALESCE(d.name, '未分配') AS department,
       COUNT(*)::bigint AS count
FROM cron_logs cl
LEFT JOIN org_members om ON om.user_id = cl.task_owner_user_id AND om.tenant_id = cl.tenant_id AND om.status = 'active'
LEFT JOIN departments d ON d.id = om.department_id AND d.tenant_id = cl.tenant_id
WHERE cl.tenant_id = ?
GROUP BY d.name
ORDER BY count DESC`

	var rows []DeptCount
	err := r.db.Raw(sql, tenantID).Scan(&rows).Error
	return rows, err
}

// CountByTenantGlobal 全平台按租户统计定时任务执行数（system_admin 用，无 tenant_id 过滤）。
func (r *CronLogRepo) CountByTenantGlobal() ([]TenantCronCount, error) {
	sql := `
SELECT tenant_id,
       COUNT(*)::bigint AS count
FROM cron_logs
GROUP BY tenant_id
ORDER BY count DESC`

	var rows []TenantCronCount
	err := r.db.Raw(sql).Scan(&rows).Error
	return rows, err
}
