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

// ClaimPending 原子领取待执行任务，并校验交互/后台队列与数据库优先级一致。
func (r *ProcessSummaryLogRepo) ClaimPending(c *gin.Context, id uuid.UUID, interactive bool) (bool, error) {
	query := r.WithTenant(c).
		Model(&model.ProcessSummaryLog{}).
		Where("id = ? AND status = ?", id, model.JobStatusPending)
	if interactive {
		query = query.Where("priority >= ?", model.SummaryPriorityVisible)
	} else {
		query = query.Where("priority < ?", model.SummaryPriorityVisible)
	}
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
