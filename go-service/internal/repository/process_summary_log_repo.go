package repository

import (
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/apptime"
)

// ProcessSummaryLogRepo 流程总结执行日志数据访问层。
type ProcessSummaryLogRepo struct {
	*BaseRepo
}

func NewProcessSummaryLogRepo(db *gorm.DB) *ProcessSummaryLogRepo {
	return &ProcessSummaryLogRepo{BaseRepo: NewBaseRepo(db)}
}

func (r *ProcessSummaryLogRepo) Create(log *model.ProcessSummaryLog) error {
	return r.DB.Create(log).Error
}

func (r *ProcessSummaryLogRepo) GetByID(c *gin.Context, id uuid.UUID) (*model.ProcessSummaryLog, error) {
	var row model.ProcessSummaryLog
	err := r.WithTenant(c).Where("id = ?", id).First(&row).Error
	return &row, err
}

func (r *ProcessSummaryLogRepo) UpdateFields(c *gin.Context, id uuid.UUID, fields map[string]interface{}) error {
	return r.WithTenant(c).Model(&model.ProcessSummaryLog{}).Where("id = ?", id).Updates(fields).Error
}

// ClaimPending 原子领取待执行任务，并校验 Redis Stream 与数据库队列类型一致。
func (r *ProcessSummaryLogRepo) ClaimPending(c *gin.Context, id uuid.UUID, queueKind string) (bool, error) {
	queueKind = model.NormalizeSummaryJobQueueKind(queueKind)
	query := r.WithTenant(c).
		Model(&model.ProcessSummaryLog{}).
		Where(
			"id = ? AND status = ? AND queue_kind = ?",
			id,
			model.JobStatusPending,
			queueKind,
		)
	res := query.Updates(map[string]interface{}{
		"status":     model.JobStatusAssembling,
		"updated_at": apptime.Now(),
	})
	return res.RowsAffected == 1, res.Error
}

func (r *ProcessSummaryLogRepo) GetRunningByProcessID(c *gin.Context, processID string) (*model.ProcessSummaryLog, error) {
	var row model.ProcessSummaryLog
	err := r.WithTenant(c).
		Where("process_id = ? AND status IN ?", processID, []string{
			model.JobStatusPending,
			model.JobStatusAssembling,
			model.JobStatusReasoning,
			model.JobStatusExtracting,
		}).
		Order("created_at DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &row, nil
}

// GetRunningWorkbenchByProcessID 查询当前用户在流程总结工作台发起的运行中任务。
func (r *ProcessSummaryLogRepo) GetRunningWorkbenchByProcessID(c *gin.Context, processID string, userID uuid.UUID) (*model.ProcessSummaryLog, error) {
	var row model.ProcessSummaryLog
	err := r.WithTenant(c).
		Where("process_id = ? AND user_id = ? AND trigger_source = ? AND status IN ?", processID, userID, model.SummaryTriggerWorkbench, []string{
			model.JobStatusPending,
			model.JobStatusAssembling,
			model.JobStatusReasoning,
			model.JobStatusExtracting,
		}).
		Order("created_at DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

// GetLatestByProcessID 查询流程最近一次总结尝试，包含失败和取消记录。
func (r *ProcessSummaryLogRepo) GetLatestByProcessID(c *gin.Context, processID string) (*model.ProcessSummaryLog, error) {
	var row model.ProcessSummaryLog
	err := r.WithTenant(c).
		Where("process_id = ?", processID).
		Order("created_at DESC").
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

// GetLatestWorkbenchMapByProcessIDs 批量查询当前用户在工作台对每个流程的最近一次总结尝试。
func (r *ProcessSummaryLogRepo) GetLatestWorkbenchMapByProcessIDs(c *gin.Context, processIDs []string, userID uuid.UUID) (map[string]*model.ProcessSummaryLog, error) {
	result := make(map[string]*model.ProcessSummaryLog)
	if len(processIDs) == 0 {
		return result, nil
	}
	var rows []model.ProcessSummaryLog
	err := r.WithTenant(c).
		Where("process_id IN ? AND user_id = ? AND trigger_source = ?", processIDs, userID, model.SummaryTriggerWorkbench).
		Order("process_id ASC, created_at DESC").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for i := range rows {
		if result[rows[i].ProcessID] == nil {
			row := rows[i]
			result[row.ProcessID] = &row
		}
	}
	return result, nil
}

// CountThisWeek 本周流程总结工作台完成记录数；userID 非空时只统计个人记录。
func (r *ProcessSummaryLogRepo) CountThisWeek(c *gin.Context, userID *uuid.UUID) (int64, error) {
	var count int64
	tenantID, _ := c.Get("tenant_id")
	q := r.DB.Table("process_summary_logs AS psl").
		Where("psl.tenant_id = ? AND psl.trigger_source = ? AND psl.status = ?", tenantID, model.SummaryTriggerWorkbench, model.JobStatusCompleted)
	if userID != nil {
		q = q.Where("psl.user_id = ?", *userID)
	}
	err := q.Where("psl.created_at >= date_trunc('week', CURRENT_TIMESTAMP AT TIME ZONE ?)", apptime.Name()).Count(&count).Error
	return count, err
}

// WeeklyTrendByDay 本周每天的流程总结工作台完成记录数。
func (r *ProcessSummaryLogRepo) WeeklyTrendByDay(c *gin.Context, userID *uuid.UUID) ([]DayCount, error) {
	tenantID, _ := c.Get("tenant_id")
	userFilter := ""
	args := []interface{}{apptime.Name(), apptime.Name(), apptime.Name(), tenantID, model.SummaryTriggerWorkbench, apptime.Name()}
	if userID != nil {
		userFilter = "AND psl.user_id = ?"
		args = append(args, *userID)
	}
	sql := `
WITH days AS (
  SELECT generate_series(
    date_trunc('week', CURRENT_TIMESTAMP AT TIME ZONE ?)::date,
    (CURRENT_TIMESTAMP AT TIME ZONE ?)::date,
    INTERVAL '1 day'
  )::date AS d
)
SELECT TO_CHAR(days.d, 'MM-DD') AS date,
       COALESCE(b.cnt, 0)::bigint AS count
FROM days
LEFT JOIN (
  SELECT DATE(psl.created_at AT TIME ZONE ?) AS d, COUNT(*)::bigint AS cnt
  FROM process_summary_logs psl
  WHERE psl.tenant_id = ?
    AND psl.trigger_source = ?
    AND psl.status = 'completed'
    AND psl.created_at >= date_trunc('week', CURRENT_TIMESTAMP AT TIME ZONE ?)
    ` + userFilter + `
  GROUP BY 1
) b ON b.d = days.d
ORDER BY days.d`
	var rows []DayCount
	err := r.DB.Raw(sql, args...).Scan(&rows).Error
	return rows, err
}

// ProcessSummaryLogEnrichedRow 带操作人信息的流程总结完成记录。
type ProcessSummaryLogEnrichedRow struct {
	ID         uuid.UUID `gorm:"column:id"`
	Title      string    `gorm:"column:title"`
	BlockCount int       `gorm:"column:block_count"`
	UserName   string    `gorm:"column:user_name"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

// RecentEnriched 最近 N 条流程总结工作台完成记录。
func (r *ProcessSummaryLogRepo) RecentEnriched(c *gin.Context, limit int, userID *uuid.UUID) ([]ProcessSummaryLogEnrichedRow, error) {
	tenantID, _ := c.Get("tenant_id")
	userFilter := ""
	args := []interface{}{tenantID, model.SummaryTriggerWorkbench}
	if userID != nil {
		userFilter = "AND psl.user_id = ?"
		args = append(args, *userID)
	}
	args = append(args, limit)
	sql := `
SELECT psl.id, psl.title,
       COALESCE(jsonb_array_length(psl.summary_result -> 'blocks'), 0) AS block_count,
       COALESCE(u.display_name, u.username, '') AS user_name,
       psl.created_at
FROM process_summary_logs psl
LEFT JOIN users u ON u.id = psl.user_id
WHERE psl.tenant_id = ? AND psl.trigger_source = ? AND psl.status = 'completed'
  ` + userFilter + `
ORDER BY psl.created_at DESC
LIMIT ?`
	var rows []ProcessSummaryLogEnrichedRow
	err := r.DB.Raw(sql, args...).Scan(&rows).Error
	return rows, err
}

// CancelPendingScheduled 取消指定总结配置尚未领取的定时任务，保留日志用于追溯。
func (r *ProcessSummaryLogRepo) CancelPendingScheduled(
	tenantID, configID uuid.UUID,
	message string,
) (int64, error) {
	res := r.DB.Model(&model.ProcessSummaryLog{}).
		Where(
			"tenant_id = ? AND schedule_config_id = ? AND status = ?",
			tenantID,
			configID,
			model.JobStatusPending,
		).
		Updates(map[string]interface{}{
			"status":        model.JobStatusCancelled,
			"error_message": message,
			"updated_at":    apptime.Now(),
		})
	return res.RowsAffected, res.Error
}

type ProcessSummaryLogWithUser struct {
	model.ProcessSummaryLog
	UserName string `json:"user_name"`
}

func (r *ProcessSummaryLogRepo) ListByIDsWithUserOrdered(c *gin.Context, ids []uuid.UUID) ([]ProcessSummaryLogWithUser, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var rows []ProcessSummaryLogWithUser
	err := r.WithTenant(c).
		Table("process_summary_logs").
		Select("process_summary_logs.*, users.display_name as user_name").
		Joins("LEFT JOIN users ON process_summary_logs.user_id = users.id").
		Where("process_summary_logs.id IN ?", ids).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]ProcessSummaryLogWithUser, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	out := make([]ProcessSummaryLogWithUser, 0, len(ids))
	for _, id := range ids {
		if row, ok := byID[id]; ok {
			out = append(out, row)
		}
	}
	return out, nil
}

func (r *ProcessSummaryLogRepo) FailStale(ctxTenantID uuid.UUID, cutoff time.Time) (int64, error) {
	res := r.DB.Model(&model.ProcessSummaryLog{}).
		Where("tenant_id = ? AND status IN ? AND updated_at < ?", ctxTenantID, []string{
			model.JobStatusPending,
			model.JobStatusAssembling,
			model.JobStatusReasoning,
			model.JobStatusExtracting,
		}, cutoff).
		Updates(map[string]interface{}{
			"status":        model.JobStatusFailed,
			"error_message": "总结任务超时（请重新发起）",
			"updated_at":    apptime.Now(),
		})
	return res.RowsAffected, res.Error
}
