package repository

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

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

// ListByProcessID 查询某流程的所有审核记录（审核链），按时间倒序。
func (r *AuditLogRepo) ListByProcessID(c *gin.Context, processID string) ([]model.AuditLog, error) {
	var logs []model.AuditLog
	err := r.WithTenant(c).
		Where("process_id = ?", processID).
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
