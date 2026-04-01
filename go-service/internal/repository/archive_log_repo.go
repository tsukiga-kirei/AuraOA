package repository

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// ArchiveLogFilter 归档复盘日志分页查询过滤条件。
type ArchiveLogFilter struct {
	Keyword     string
	ProcessType string
	Compliance  string
	StartDate   *time.Time
	EndDate     *time.Time
}

// ArchiveLogStats 归档复盘日志统计。
type ArchiveLogStats struct {
	Total         int64 `json:"total"`
	Compliant     int64 `json:"compliant"`
	Partial       int64 `json:"partial"`
	NonCompliant  int64 `json:"non_compliant"`
	PendingReview int64 `json:"pending_review"` // 非 completed 状态
}

// ArchiveLogRepo 提供归档复盘日志的数据访问方法。
type ArchiveLogRepo struct {
	*BaseRepo
}

func NewArchiveLogRepo(db *gorm.DB) *ArchiveLogRepo {
	return &ArchiveLogRepo{BaseRepo: NewBaseRepo(db)}
}

func (r *ArchiveLogRepo) Create(log *model.ArchiveLog) error {
	return r.DB.Create(log).Error
}

func (r *ArchiveLogRepo) GetByID(c *gin.Context, id uuid.UUID) (*model.ArchiveLog, error) {
	var log model.ArchiveLog
	err := r.WithTenant(c).Where("id = ?", id).First(&log).Error
	return &log, err
}

func (r *ArchiveLogRepo) UpdateFields(c *gin.Context, id uuid.UUID, updates map[string]interface{}) error {
	return r.WithTenant(c).Model(&model.ArchiveLog{}).Where("id = ?", id).Updates(updates).Error
}

func (r *ArchiveLogRepo) GetLatestByProcessID(c *gin.Context, processID string) (*model.ArchiveLog, error) {
	var log model.ArchiveLog
	err := r.WithTenant(c).
		Where("process_id = ?", processID).
		Order("created_at DESC").
		First(&log).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	return &log, err
}

func (r *ArchiveLogRepo) GetLatestResultMap(c *gin.Context, processIDs []string) (map[string]*model.ArchiveLog, error) {
	if len(processIDs) == 0 {
		return map[string]*model.ArchiveLog{}, nil
	}

	var logs []model.ArchiveLog
	err := r.WithTenant(c).
		Where("process_id IN ?", processIDs).
		Order("created_at DESC").
		Find(&logs).Error
	if err != nil {
		return nil, err
	}

	result := make(map[string]*model.ArchiveLog, len(processIDs))
	for i := range logs {
		if _, exists := result[logs[i].ProcessID]; !exists {
			result[logs[i].ProcessID] = &logs[i]
		}
	}
	return result, nil
}

type ArchiveLogWithUser struct {
	model.ArchiveLog
	UserName string `json:"user_name"`
}

func (r *ArchiveLogRepo) ListCompletedByProcessIDWithUser(c *gin.Context, processID string) ([]ArchiveLogWithUser, error) {
	var logs []ArchiveLogWithUser
	err := r.WithTenant(c).
		Table("archive_logs").
		Select("archive_logs.*, users.display_name as user_name").
		Joins("left join users on archive_logs.user_id = users.id").
		Where("archive_logs.process_id = ? AND archive_logs.status = ?", processID, model.AuditStatusCompleted).
		Order("archive_logs.created_at DESC").
		Find(&logs).Error
	return logs, err
}

// ListByIDsWithUserOrdered 按给定 id 顺序返回归档日志（有效复盘链）。
func (r *ArchiveLogRepo) ListByIDsWithUserOrdered(c *gin.Context, ids []uuid.UUID) ([]ArchiveLogWithUser, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var logs []ArchiveLogWithUser
	err := r.WithTenant(c).
		Table("archive_logs").
		Select("archive_logs.*, users.display_name as user_name").
		Joins("LEFT JOIN users ON archive_logs.user_id = users.id").
		Where("archive_logs.id IN ?", ids).
		Find(&logs).Error
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]ArchiveLogWithUser, len(logs))
	for _, l := range logs {
		byID[l.ID] = l
	}
	out := make([]ArchiveLogWithUser, 0, len(ids))
	for _, id := range ids {
		if row, ok := byID[id]; ok {
			out = append(out, row)
		}
	}
	return out, nil
}

// ArchiveLogWithUser2 归档日志 + 用户名（数据管理页专用）。
type ArchiveLogWithUser2 struct {
	model.ArchiveLog
	UserName string `json:"user_name"`
}

// ListPagedWithUser 数据管理页：分页查询归档复盘日志，JOIN 用户名，支持多维过滤。
func (r *ArchiveLogRepo) ListPagedWithUser(c *gin.Context, filter ArchiveLogFilter, page, pageSize int) ([]ArchiveLogWithUser2, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}

	base := r.WithTenant(c).
		Table("archive_logs").
		Select("archive_logs.*, COALESCE(users.display_name, users.username, '') as user_name").
		Joins("LEFT JOIN users ON archive_logs.user_id = users.id")

	base = applyArchiveLogFilter(base, filter)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []ArchiveLogWithUser2
	err := base.Order("archive_logs.created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error
	return items, total, err
}

// CountStats 数据管理页：统计各合规分组数量。
func (r *ArchiveLogRepo) CountStats(c *gin.Context) (*ArchiveLogStats, error) {
	type row struct {
		Status     string
		Compliance string
		Cnt        int64
	}
	var rows []row
	err := r.WithTenant(c).
		Table("archive_logs").
		Select("status, compliance, COUNT(*) as cnt").
		Group("status, compliance").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	stats := &ArchiveLogStats{}
	for _, r := range rows {
		stats.Total += r.Cnt
		if r.Status == model.AuditStatusCompleted {
			switch r.Compliance {
			case "compliant":
				stats.Compliant += r.Cnt
			case "partially_compliant":
				stats.Partial += r.Cnt
			case "non_compliant":
				stats.NonCompliant += r.Cnt
			}
		} else {
			stats.PendingReview += r.Cnt
		}
	}
	return stats, nil
}

// CountStatsByTimeRange 获取指定时间范围内的统计数据（租户隔离）。
func (r *ArchiveLogRepo) CountStatsByTimeRange(c *gin.Context, start, end time.Time) (*ArchiveLogStats, error) {
	type row struct {
		Status     string
		Compliance string
		Cnt        int64
	}
	var rows []row
	err := r.WithTenant(c).
		Table("archive_logs").
		Select("status, compliance, COUNT(*) as cnt").
		Where("archive_logs.created_at >= ? AND archive_logs.created_at <= ?", start, end).
		Group("status, compliance").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	stats := &ArchiveLogStats{}
	for _, r := range rows {
		stats.Total += r.Cnt
		if r.Status == model.AuditStatusCompleted {
			switch r.Compliance {
			case "compliant":
				stats.Compliant += r.Cnt
			case "partially_compliant":
				stats.Partial += r.Cnt
			case "non_compliant":
				stats.NonCompliant += r.Cnt
			}
		} else {
			stats.PendingReview += r.Cnt
		}
	}
	return stats, nil
}

// DashboardArchiveRecentRow 仪表盘归档复盘列表行。
type DashboardArchiveRecentRow struct {
	ID         uuid.UUID `json:"id" gorm:"column:id"`
	Title      string    `json:"title" gorm:"column:title"`
	Compliance string    `json:"compliance" gorm:"column:compliance"`
	UserName   string    `json:"user_name" gorm:"column:user_name"`
	CreatedAt  time.Time `json:"created_at" gorm:"column:created_at"`
}

// DashboardRecentArchiveLogs 最近归档复盘记录（已完成或失败均展示，便于感知活动）。forUserID 非空时仅该操作人。
func (r *ArchiveLogRepo) DashboardRecentArchiveLogs(c *gin.Context, limit int, forUserID *uuid.UUID) ([]DashboardArchiveRecentRow, error) {
	if limit < 1 {
		limit = 8
	}
	q := r.WithTenant(c).
		Table("archive_logs").
		Select("archive_logs.id, archive_logs.title, archive_logs.compliance, COALESCE(users.display_name, users.username, '') as user_name, archive_logs.created_at").
		Joins("LEFT JOIN users ON archive_logs.user_id = users.id").
		Where("archive_logs.status IN ?", []string{model.AuditStatusCompleted, model.AuditStatusFailed})
	if forUserID != nil {
		q = q.Where("archive_logs.user_id = ?", *forUserID)
	}
	var rows []DashboardArchiveRecentRow
	err := q.Order("archive_logs.created_at DESC").Limit(limit).Scan(&rows).Error
	return rows, err
}

// DashboardRecentArchiveLogsGlobal 全库最近归档复盘记录。
func (r *ArchiveLogRepo) DashboardRecentArchiveLogsGlobal(limit int) ([]DashboardArchiveRecentRow, error) {
	if limit < 1 {
		limit = 8
	}
	var rows []DashboardArchiveRecentRow
	err := r.DB.
		Table("archive_logs").
		Select("archive_logs.id, archive_logs.title, archive_logs.compliance, COALESCE(users.display_name, users.username, '') as user_name, archive_logs.created_at").
		Joins("LEFT JOIN users ON archive_logs.user_id = users.id").
		Where("archive_logs.status IN ?", []string{model.AuditStatusCompleted, model.AuditStatusFailed}).
		Order("archive_logs.created_at DESC").
		Limit(limit).
		Scan(&rows).Error
	return rows, err
}

// CountCompletedArchiveLogs 已完成归档复盘条数（用于概览「归档」计数）。forUserID 非空时仅该操作人。
func (r *ArchiveLogRepo) CountCompletedArchiveLogs(c *gin.Context, forUserID *uuid.UUID) (int64, error) {
	var n int64
	q := r.WithTenant(c).
		Model(&model.ArchiveLog{}).
		Where("status = ?", model.AuditStatusCompleted)
	if forUserID != nil {
		q = q.Where("user_id = ?", *forUserID)
	}
	err := q.Count(&n).Error
	return n, err
}

// CountCompletedArchiveLogsGlobal 全库已完成归档条数。
func (r *ArchiveLogRepo) CountCompletedArchiveLogsGlobal() (int64, error) {
	var n int64
	err := r.DB.
		Model(&model.ArchiveLog{}).
		Where("status = ?", model.AuditStatusCompleted).
		Count(&n).Error
	return n, err
}

func applyArchiveLogFilter(db *gorm.DB, f ArchiveLogFilter) *gorm.DB {
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		db = db.Where("(archive_logs.title ILIKE ? OR archive_logs.process_id ILIKE ?)", like, like)
	}
	if f.ProcessType != "" {
		db = db.Where("archive_logs.process_type = ?", f.ProcessType)
	}
	if f.Compliance != "" {
		db = db.Where("archive_logs.compliance = ?", f.Compliance)
	}
	if f.StartDate != nil {
		db = db.Where("archive_logs.created_at >= ?", f.StartDate)
	}
	if f.EndDate != nil {
		db = db.Where("archive_logs.created_at <= ?", f.EndDate)
	}
	return db
}
