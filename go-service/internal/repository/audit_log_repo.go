package repository

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// ErrNoTenantContext 上下文中缺少 tenant_id。
var ErrNoTenantContext = errors.New("missing tenant_id in context")

// AuditLogFilter 审核日志分页查询过滤条件。
type AuditLogFilter struct {
	// status_group: "pending_ai" = 未完成状态，"ai_done" = completed，"" = 全部
	StatusGroup    string
	Keyword        string
	ProcessType    string
	Recommendation string
	StartDate      *time.Time
	EndDate        *time.Time
}

// AuditLogStats 审核日志统计。
type AuditLogStats struct {
	Total        int64 `json:"total"`
	PendingAI    int64 `json:"pending_ai"`
	AIDone       int64 `json:"ai_done"`
	ApproveCount int64 `json:"approve_count"`
	ReturnCount  int64 `json:"return_count"`
	ReviewCount  int64 `json:"review_count"`
}

// AuditLogRepo 审核日志数据访问层。
type AuditLogRepo struct {
	*BaseRepo
}

func NewAuditLogRepo(db *gorm.DB) *AuditLogRepo {
	return &AuditLogRepo{BaseRepo: NewBaseRepo(db)}
}

func (r *AuditLogRepo) Create(log *model.AuditLog) error {
	return r.DB.Create(log).Error
}

func (r *AuditLogRepo) GetByID(c *gin.Context, id uuid.UUID) (*model.AuditLog, error) {
	var log model.AuditLog
	err := r.WithTenant(c).Where("id = ?", id).First(&log).Error
	return &log, err
}

// UpdateFields 更新审核日志指定字段（租户隔离）。
func (r *AuditLogRepo) UpdateFields(c *gin.Context, id uuid.UUID, updates map[string]interface{}) error {
	return r.WithTenant(c).Model(&model.AuditLog{}).Where("id = ?", id).Updates(updates).Error
}

// ListByProcessID 查询某流程的所有审核记录（审核链），按时间倒序。
func (r *AuditLogRepo) ListByProcessID(c *gin.Context, processID string) ([]model.AuditLog, error) {
	var logs []model.AuditLog
	err := r.WithTenant(c).
		Where("process_id = ?", processID).
		Order("created_at DESC").
		Find(&logs).Error
	return logs, err
}

type AuditLogWithUser struct {
	model.AuditLog
	UserName string `json:"user_name"`
}

// ListCompletedByProcessIDWithUser 审核链：仅已完成的记录，按时间倒序，包含用户名。
func (r *AuditLogRepo) ListCompletedByProcessIDWithUser(c *gin.Context, processID string) ([]AuditLogWithUser, error) {
	var logs []AuditLogWithUser
	err := r.WithTenant(c).
		Table("audit_logs").
		Select("audit_logs.*, users.display_name as user_name").
		Joins("left join users on audit_logs.user_id = users.id").
		Where("audit_logs.process_id = ? AND audit_logs.status = ?", processID, model.AuditStatusCompleted).
		Order("audit_logs.created_at DESC").
		Find(&logs).Error
	return logs, err
}

// ListCompletedByProcessID 审核链：仅已完成的记录，按时间倒序。
func (r *AuditLogRepo) ListCompletedByProcessID(c *gin.Context, processID string) ([]model.AuditLog, error) {
	var logs []model.AuditLog
	err := r.WithTenant(c).
		Where("process_id = ? AND status = ?", processID, model.AuditStatusCompleted).
		Order("created_at DESC").
		Find(&logs).Error
	return logs, err
}

// ListByProcessType 查询某流程类型的所有审核记录（租户内），按时间倒序。
func (r *AuditLogRepo) ListByProcessType(c *gin.Context, processType string) ([]model.AuditLog, error) {
	var logs []model.AuditLog
	err := r.WithTenant(c).
		Where("process_type = ?", processType).
		Order("created_at DESC").
		Find(&logs).Error
	return logs, err
}

// GetLatestByProcessID 获取某流程最新的审核记录。
func (r *AuditLogRepo) GetLatestByProcessID(c *gin.Context, processID string) (*model.AuditLog, error) {
	var log model.AuditLog
	err := r.WithTenant(c).
		Where("process_id = ?", processID).
		Order("created_at DESC").
		First(&log).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &log, err
}

// HasAuditRecord 检查某流程是否有审核记录。
func (r *AuditLogRepo) HasAuditRecord(c *gin.Context, processID string) (bool, error) {
	var count int64
	err := r.WithTenant(c).
		Model(&model.AuditLog{}).
		Where("process_id = ?", processID).
		Count(&count).Error
	return count > 0, err
}

// BatchCheckHasAudit 批量检查多个流程是否有审核记录，返回已有记录的 processID 集合。
func (r *AuditLogRepo) BatchCheckHasAudit(c *gin.Context, processIDs []string) (map[string]bool, error) {
	if len(processIDs) == 0 {
		return map[string]bool{}, nil
	}
	var records []struct {
		ProcessID string
	}
	err := r.WithTenant(c).
		Model(&model.AuditLog{}).
		Select("DISTINCT process_id").
		Where("process_id IN ?", processIDs).
		Find(&records).Error
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool)
	for _, rec := range records {
		result[rec.ProcessID] = true
	}
	return result, nil
}

// GetLatestResultMap 获取多个流程的最新审核结果，返回 processID -> AuditLog 映射。
func (r *AuditLogRepo) GetLatestResultMap(c *gin.Context, processIDs []string) (map[string]*model.AuditLog, error) {
	if len(processIDs) == 0 {
		return map[string]*model.AuditLog{}, nil
	}

	var logs []model.AuditLog
	err := r.WithTenant(c).
		Where("process_id IN ?", processIDs).
		Order("created_at DESC").
		Find(&logs).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]*model.AuditLog)
	for i := range logs {
		if _, exists := result[logs[i].ProcessID]; !exists {
			result[logs[i].ProcessID] = &logs[i]
		}
	}
	return result, nil
}

// AuditLogWithUser 审核日志 + 用户显示名（用于数据管理页）。
type AuditLogWithUser2 struct {
	model.AuditLog
	UserName string `json:"user_name"`
}

// ListPagedWithUser 数据管理页：分页查询审核日志，JOIN 用户名，支持多维过滤。
func (r *AuditLogRepo) ListPagedWithUser(c *gin.Context, filter AuditLogFilter, page, pageSize int) ([]AuditLogWithUser2, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}

	base := r.WithTenant(c).
		Table("audit_logs").
		Select("audit_logs.*, COALESCE(users.display_name, users.username, '') as user_name").
		Joins("LEFT JOIN users ON audit_logs.user_id = users.id")

	base = applyAuditLogFilter(base, filter)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []AuditLogWithUser2
	err := base.Order("audit_logs.created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error
	return items, total, err
}

// CountStats 数据管理页：统计各分组数量。
func (r *AuditLogRepo) CountStats(c *gin.Context) (*AuditLogStats, error) {
	type row struct {
		Status         string
		Recommendation string
		Cnt            int64
	}
	var rows []row
	err := r.WithTenant(c).
		Table("audit_logs").
		Select("status, recommendation, COUNT(*) as cnt").
		Group("status, recommendation").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	stats := &AuditLogStats{}
	completedStatuses := map[string]bool{model.AuditStatusCompleted: true}
	pendingStatuses := map[string]bool{
		model.AuditStatusPending:    true,
		model.AuditStatusAssembling: true,
		model.AuditStatusReasoning:  true,
		model.AuditStatusExtracting: true,
		model.AuditStatusFailed:     true,
	}
	for _, r := range rows {
		stats.Total += r.Cnt
		if completedStatuses[r.Status] {
			stats.AIDone += r.Cnt
			switch r.Recommendation {
			case "approve":
				stats.ApproveCount += r.Cnt
			case "return":
				stats.ReturnCount += r.Cnt
			case "review":
				stats.ReviewCount += r.Cnt
			}
		} else if pendingStatuses[r.Status] {
			stats.PendingAI += r.Cnt
		}
	}
	return stats, nil
}

// CountStatsGlobal 全库审核日志统计（system_admin 平台仪表盘，无租户过滤）。
func (r *AuditLogRepo) CountStatsGlobal() (*AuditLogStats, error) {
	type row struct {
		Status           string
		Recommendation   string
		Cnt              int64
	}
	var rows []row
	err := r.DB.
		Table("audit_logs").
		Select("status, recommendation, COUNT(*) as cnt").
		Group("status, recommendation").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	stats := &AuditLogStats{}
	completedStatuses := map[string]bool{model.AuditStatusCompleted: true}
	pendingStatuses := map[string]bool{
		model.AuditStatusPending:    true,
		model.AuditStatusAssembling: true,
		model.AuditStatusReasoning:  true,
		model.AuditStatusExtracting: true,
		model.AuditStatusFailed:     true,
	}
	for _, row := range rows {
		stats.Total += row.Cnt
		if completedStatuses[row.Status] {
			stats.AIDone += row.Cnt
			switch row.Recommendation {
			case "approve":
				stats.ApproveCount += row.Cnt
			case "return":
				stats.ReturnCount += row.Cnt
			case "review":
				stats.ReviewCount += row.Cnt
			}
		} else if pendingStatuses[row.Status] {
			stats.PendingAI += row.Cnt
		}
	}
	return stats, nil
}

// DashboardWeeklyCompletedTrend 最近 n 个自然日（含当日）内，按 UTC 日聚合的已完成审核次数。
func (r *AuditLogRepo) DashboardWeeklyCompletedTrend(c *gin.Context, days int) ([]struct {
	Date  string `gorm:"column:date"`
	Count int64  `gorm:"column:count"`
}, error) {
	if days < 1 {
		days = 7
	}
	tid, ok := c.Get("tenant_id")
	if !ok || tid == nil || tid == "" {
		return nil, ErrNoTenantContext
	}
	tenantUUID, err := uuid.Parse(tid.(string))
	if err != nil {
		return nil, err
	}

	q := `
WITH days AS (
  SELECT generate_series(
    (CURRENT_DATE AT TIME ZONE 'UTC')::date - ($2::int - 1),
    (CURRENT_DATE AT TIME ZONE 'UTC')::date,
    INTERVAL '1 day'
  )::date AS d
)
SELECT TO_CHAR(days.d, 'MM-DD') AS date, COALESCE(b.cnt, 0)::bigint AS count
FROM days
LEFT JOIN (
  SELECT DATE(created_at AT TIME ZONE 'UTC') AS d, COUNT(*)::bigint AS cnt
  FROM audit_logs
  WHERE tenant_id = $1 AND status = $3
  GROUP BY 1
) b ON b.d = days.d
ORDER BY days.d
`
	var rows []struct {
		Date  string `gorm:"column:date"`
		Count int64  `gorm:"column:count"`
	}
	err = r.DB.Raw(q, tenantUUID, days, model.AuditStatusCompleted).Scan(&rows).Error
	return rows, err
}

// DashboardWeeklyCompletedTrendGlobal 全库：最近 n 个 UTC 自然日已完成审核次数。
func (r *AuditLogRepo) DashboardWeeklyCompletedTrendGlobal(days int) ([]struct {
	Date  string `gorm:"column:date"`
	Count int64  `gorm:"column:count"`
}, error) {
	if days < 1 {
		days = 7
	}
	q := `
WITH days AS (
  SELECT generate_series(
    (CURRENT_DATE AT TIME ZONE 'UTC')::date - ($1::int - 1),
    (CURRENT_DATE AT TIME ZONE 'UTC')::date,
    INTERVAL '1 day'
  )::date AS d
)
SELECT TO_CHAR(days.d, 'MM-DD') AS date, COALESCE(b.cnt, 0)::bigint AS count
FROM days
LEFT JOIN (
  SELECT DATE(created_at AT TIME ZONE 'UTC') AS d, COUNT(*)::bigint AS cnt
  FROM audit_logs
  WHERE status = $2
  GROUP BY 1
) b ON b.d = days.d
ORDER BY days.d
`
	var rows []struct {
		Date  string `gorm:"column:date"`
		Count int64  `gorm:"column:count"`
	}
	err := r.DB.Raw(q, days, model.AuditStatusCompleted).Scan(&rows).Error
	return rows, err
}

// DashboardRecentCompletedRows 最近完成的审核记录（用于动态与归档以外的展示）。
type DashboardRecentAuditRow struct {
	ID        uuid.UUID `json:"id" gorm:"column:id"`
	Title     string    `json:"title" gorm:"column:title"`
	UserName  string    `json:"user_name" gorm:"column:user_name"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
	Status    string    `json:"status" gorm:"column:status"`
}

// DashboardRecentAudits 最近审核记录（含完成/失败），按时间倒序。
func (r *AuditLogRepo) DashboardRecentAudits(c *gin.Context, limit int) ([]DashboardRecentAuditRow, error) {
	if limit < 1 {
		limit = 8
	}
	var rows []DashboardRecentAuditRow
	err := r.WithTenant(c).
		Table("audit_logs").
		Select("audit_logs.id, audit_logs.title, COALESCE(users.display_name, users.username, '') as user_name, audit_logs.created_at, audit_logs.status").
		Joins("LEFT JOIN users ON audit_logs.user_id = users.id").
		Where("audit_logs.status IN ?", []string{model.AuditStatusCompleted, model.AuditStatusFailed}).
		Order("audit_logs.created_at DESC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

// DashboardRecentAuditsGlobal 全库最近审核记录（完成/失败），按时间倒序。
func (r *AuditLogRepo) DashboardRecentAuditsGlobal(limit int) ([]DashboardRecentAuditRow, error) {
	if limit < 1 {
		limit = 8
	}
	var rows []DashboardRecentAuditRow
	err := r.DB.
		Table("audit_logs").
		Select("audit_logs.id, audit_logs.title, COALESCE(users.display_name, users.username, '') as user_name, audit_logs.created_at, audit_logs.status").
		Joins("LEFT JOIN users ON audit_logs.user_id = users.id").
		Where("audit_logs.status IN ?", []string{model.AuditStatusCompleted, model.AuditStatusFailed}).
		Order("audit_logs.created_at DESC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

// DashboardAuditOutcomeForAIStats 用于计算 AI 成功率：已完成条数与失败条数。
func (r *AuditLogRepo) DashboardAuditOutcomeForAIStats(c *gin.Context) (completed, failed int64, err error) {
	type row struct {
		Status string
		Cnt    int64
	}
	var rows []row
	err = r.WithTenant(c).
		Model(&model.AuditLog{}).
		Select("status, COUNT(*) as cnt").
		Where("status IN ?", []string{model.AuditStatusCompleted, model.AuditStatusFailed}).
		Group("status").
		Scan(&rows).Error
	if err != nil {
		return 0, 0, err
	}
	for _, x := range rows {
		switch x.Status {
		case model.AuditStatusCompleted:
			completed += x.Cnt
		case model.AuditStatusFailed:
			failed += x.Cnt
		}
	}
	return completed, failed, nil
}

// DashboardAuditOutcomeForAIStatsGlobal 全库已完成/失败条数（AI 成功率分母）。
func (r *AuditLogRepo) DashboardAuditOutcomeForAIStatsGlobal() (completed, failed int64, err error) {
	type row struct {
		Status string
		Cnt    int64
	}
	var rows []row
	err = r.DB.
		Model(&model.AuditLog{}).
		Select("status, COUNT(*) as cnt").
		Where("status IN ?", []string{model.AuditStatusCompleted, model.AuditStatusFailed}).
		Group("status").
		Scan(&rows).Error
	if err != nil {
		return 0, 0, err
	}
	for _, x := range rows {
		switch x.Status {
		case model.AuditStatusCompleted:
			completed += x.Cnt
		case model.AuditStatusFailed:
			failed += x.Cnt
		}
	}
	return completed, failed, nil
}

// DashboardPlatformTenantRankRow 全平台按租户已完成审核数排名。
type DashboardPlatformTenantRankRow struct {
	TenantID   uuid.UUID `gorm:"column:tenant_id"`
	TenantName string    `gorm:"column:tenant_name"`
	TenantCode string    `gorm:"column:tenant_code"`
	AuditCount int64     `gorm:"column:audit_count"`
}

// DashboardTenantAuditRankingGlobal 全库各租户已完成审核 Top N。
func (r *AuditLogRepo) DashboardTenantAuditRankingGlobal(limit int) ([]DashboardPlatformTenantRankRow, error) {
	if limit < 1 {
		limit = 10
	}
	q := `
SELECT t.id AS tenant_id,
       t.name AS tenant_name,
       t.code AS tenant_code,
       COUNT(*)::bigint AS audit_count
FROM audit_logs al
JOIN tenants t ON t.id = al.tenant_id
WHERE al.status = ?
GROUP BY t.id, t.name, t.code
ORDER BY audit_count DESC
LIMIT ?
`
	var rows []DashboardPlatformTenantRankRow
	err := r.DB.Raw(q, model.AuditStatusCompleted, limit).Scan(&rows).Error
	return rows, err
}

// DashboardUserAuditRankRow 用户审核排行行。
type DashboardUserAuditRankRow struct {
	Username    string    `gorm:"column:username"`
	DisplayName string    `gorm:"column:display_name"`
	Department  string    `gorm:"column:department"`
	AuditCount  int64     `gorm:"column:audit_count"`
	LastActive  time.Time `gorm:"column:last_active"`
}

// DashboardUserAuditRanking 租户内已完成审核次数 Top N。
func (r *AuditLogRepo) DashboardUserAuditRanking(c *gin.Context, limit int) ([]DashboardUserAuditRankRow, error) {
	if limit < 1 {
		limit = 10
	}
	tid, ok := c.Get("tenant_id")
	if !ok || tid == nil || tid == "" {
		return nil, ErrNoTenantContext
	}
	tenantUUID, err := uuid.Parse(tid.(string))
	if err != nil {
		return nil, err
	}

	q := `
SELECT u.username AS username,
       u.display_name AS display_name,
       COALESCE(MAX(d.name), '') AS department,
       COUNT(*)::bigint AS audit_count,
       MAX(al.created_at) AS last_active
FROM audit_logs al
JOIN users u ON u.id = al.user_id
LEFT JOIN org_members om ON om.user_id = al.user_id AND om.tenant_id = al.tenant_id AND om.status = 'active'
LEFT JOIN departments d ON d.id = om.department_id AND d.tenant_id = al.tenant_id
WHERE al.tenant_id = ? AND al.status = ?
GROUP BY u.id, u.username, u.display_name
ORDER BY audit_count DESC
LIMIT ?
`
	var rows []DashboardUserAuditRankRow
	err = r.DB.Raw(q, tenantUUID, model.AuditStatusCompleted, limit).Scan(&rows).Error
	return rows, err
}

// DashboardDeptAuditDistribution 已完成审核按部门分布（未关联组织的记入 __unassigned__）。
func (r *AuditLogRepo) DashboardDeptAuditDistribution(c *gin.Context, limit int) ([]struct {
	Department string `gorm:"column:department"`
	Count      int64  `gorm:"column:count"`
}, error) {
	if limit < 1 {
		limit = 12
	}
	tid, ok := c.Get("tenant_id")
	if !ok || tid == nil || tid == "" {
		return nil, ErrNoTenantContext
	}
	tenantUUID, err := uuid.Parse(tid.(string))
	if err != nil {
		return nil, err
	}

	q := `
SELECT dept_key AS department, COUNT(*)::bigint AS count
FROM (
  SELECT COALESCE(d.name, '__unassigned__') AS dept_key
  FROM audit_logs al
  LEFT JOIN org_members om ON om.user_id = al.user_id AND om.tenant_id = al.tenant_id AND om.status = 'active'
  LEFT JOIN departments d ON d.id = om.department_id AND d.tenant_id = al.tenant_id
  WHERE al.tenant_id = ? AND al.status = ?
) t
GROUP BY dept_key
ORDER BY count DESC
LIMIT ?
`
	var rows []struct {
		Department string `gorm:"column:department"`
		Count      int64  `gorm:"column:count"`
	}
	err = r.DB.Raw(q, tenantUUID, model.AuditStatusCompleted, limit).Scan(&rows).Error
	return rows, err
}

// DashboardDistinctUserCountSince 指定时间以来有审核行为的去重用户数。
func (r *AuditLogRepo) DashboardDistinctUserCountSince(c *gin.Context, since time.Time) (int64, error) {
	tid, ok := c.Get("tenant_id")
	if !ok || tid == nil || tid == "" {
		return 0, ErrNoTenantContext
	}
	tenantUUID, err := uuid.Parse(tid.(string))
	if err != nil {
		return 0, err
	}
	type cntRow struct {
		N int64 `gorm:"column:n"`
	}
	var row cntRow
	err = r.DB.Raw(
		`SELECT COUNT(DISTINCT user_id)::bigint AS n FROM audit_logs WHERE tenant_id = ? AND created_at >= ?`,
		tenantUUID, since,
	).Scan(&row).Error
	return row.N, err
}

func applyAuditLogFilter(db *gorm.DB, f AuditLogFilter) *gorm.DB {
	switch f.StatusGroup {
	case "pending_ai":
		db = db.Where("audit_logs.status != ?", model.AuditStatusCompleted)
	case "ai_done":
		db = db.Where("audit_logs.status = ?", model.AuditStatusCompleted)
	}
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		db = db.Where("(audit_logs.title ILIKE ? OR audit_logs.process_id ILIKE ?)", like, like)
	}
	if f.ProcessType != "" {
		db = db.Where("audit_logs.process_type = ?", f.ProcessType)
	}
	if f.Recommendation != "" {
		db = db.Where("audit_logs.recommendation = ?", f.Recommendation)
	}
	if f.StartDate != nil {
		db = db.Where("audit_logs.created_at >= ?", f.StartDate)
	}
	if f.EndDate != nil {
		db = db.Where("audit_logs.created_at <= ?", f.EndDate)
	}
	return db
}
