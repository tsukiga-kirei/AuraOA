package repository

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

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
