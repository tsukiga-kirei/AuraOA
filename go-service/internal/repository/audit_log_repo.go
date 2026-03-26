package repository

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

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
