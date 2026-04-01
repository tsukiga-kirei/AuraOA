package repository

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
)

// AuditProcessSnapshotRepo 审核有效结论快照。
type AuditProcessSnapshotRepo struct {
	*BaseRepo
}

func NewAuditProcessSnapshotRepo(db *gorm.DB) *AuditProcessSnapshotRepo {
	return &AuditProcessSnapshotRepo{BaseRepo: NewBaseRepo(db)}
}

// UpsertAppendValid 成功解析后追加日志 id 并更新最新有效结论。
func (r *AuditProcessSnapshotRepo) UpsertAppendValid(c *gin.Context, tenantID uuid.UUID, processID string, logID uuid.UUID, title, processType, recommendation string, score, confidence int) error {
	var existing model.AuditProcessSnapshot
	err := r.WithTenant(c).Where("process_id = ?", processID).First(&existing).Error
	now := time.Now()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ids := []string{logID.String()}
		b, _ := json.Marshal(ids)
		row := &model.AuditProcessSnapshot{
			TenantID:         tenantID,
			ProcessID:        processID,
			ValidLogIDs:      datatypes.JSON(b),
			LatestValidLogID: logID,
			Title:            title,
			ProcessType:      processType,
			Recommendation:   recommendation,
			Score:            score,
			Confidence:       confidence,
			UpdatedAt:        now,
		}
		return r.DB.Create(row).Error
	}
	if err != nil {
		return err
	}

	var uuidStrs []string
	_ = json.Unmarshal(existing.ValidLogIDs, &uuidStrs)
	found := false
	for _, id := range uuidStrs {
		if id == logID.String() {
			found = true
			break
		}
	}
	if !found {
		uuidStrs = append(uuidStrs, logID.String())
	}
	b, _ := json.Marshal(uuidStrs)
	return r.WithTenant(c).Model(&model.AuditProcessSnapshot{}).Where("id = ?", existing.ID).Updates(map[string]interface{}{
		"valid_log_ids":       datatypes.JSON(b),
		"latest_valid_log_id": logID,
		"title":               title,
		"process_type":        processType,
		"recommendation":      recommendation,
		"score":               score,
		"confidence":          confidence,
		"updated_at":          now,
	}).Error
}

// GetByProcessID 单流程快照。
func (r *AuditProcessSnapshotRepo) GetByProcessID(c *gin.Context, processID string) (*model.AuditProcessSnapshot, error) {
	var row model.AuditProcessSnapshot
	err := r.WithTenant(c).Where("process_id = ?", processID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

// GetMapByProcessIDs 批量查询。
func (r *AuditProcessSnapshotRepo) GetMapByProcessIDs(c *gin.Context, processIDs []string) (map[string]*model.AuditProcessSnapshot, error) {
	if len(processIDs) == 0 {
		return map[string]*model.AuditProcessSnapshot{}, nil
	}
	var rows []model.AuditProcessSnapshot
	if err := r.WithTenant(c).Where("process_id IN ?", processIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]*model.AuditProcessSnapshot, len(rows))
	for i := range rows {
		out[rows[i].ProcessID] = &rows[i]
	}
	return out, nil
}
