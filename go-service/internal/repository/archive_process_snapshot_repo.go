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

// ArchiveProcessSnapshotRepo 归档复盘有效结论快照。
type ArchiveProcessSnapshotRepo struct {
	*BaseRepo
}

func NewArchiveProcessSnapshotRepo(db *gorm.DB) *ArchiveProcessSnapshotRepo {
	return &ArchiveProcessSnapshotRepo{BaseRepo: NewBaseRepo(db)}
}

// UpsertAppendValid 成功解析后追加 archive_logs id。
func (r *ArchiveProcessSnapshotRepo) UpsertAppendValid(c *gin.Context, tenantID uuid.UUID, processID string, archiveLogID uuid.UUID, title, processType, compliance string, complianceScore, confidence int) error {
	var existing model.ArchiveProcessSnapshot
	err := r.WithTenant(c).Where("process_id = ?", processID).First(&existing).Error
	now := time.Now()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ids := []string{archiveLogID.String()}
		b, _ := json.Marshal(ids)
		row := &model.ArchiveProcessSnapshot{
			TenantID:                tenantID,
			ProcessID:               processID,
			ValidArchiveLogIDs:      datatypes.JSON(b),
			LatestValidArchiveLogID: archiveLogID,
			Title:                   title,
			ProcessType:             processType,
			Compliance:              compliance,
			ComplianceScore:         complianceScore,
			Confidence:              confidence,
			UpdatedAt:               now,
		}
		return r.DB.Create(row).Error
	}
	if err != nil {
		return err
	}

	var uuidStrs []string
	_ = json.Unmarshal(existing.ValidArchiveLogIDs, &uuidStrs)
	found := false
	for _, id := range uuidStrs {
		if id == archiveLogID.String() {
			found = true
			break
		}
	}
	if !found {
		uuidStrs = append(uuidStrs, archiveLogID.String())
	}
	b, _ := json.Marshal(uuidStrs)
	return r.WithTenant(c).Model(&model.ArchiveProcessSnapshot{}).Where("id = ?", existing.ID).Updates(map[string]interface{}{
		"valid_archive_log_ids":  datatypes.JSON(b),
		"latest_valid_archive_log_id": archiveLogID,
		"title":                  title,
		"process_type":           processType,
		"compliance":             compliance,
		"compliance_score":       complianceScore,
		"confidence":             confidence,
		"updated_at":             now,
	}).Error
}

// GetByProcessID 单流程快照。
func (r *ArchiveProcessSnapshotRepo) GetByProcessID(c *gin.Context, processID string) (*model.ArchiveProcessSnapshot, error) {
	var row model.ArchiveProcessSnapshot
	err := r.WithTenant(c).Where("process_id = ?", processID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

// GetMapByProcessIDs 批量查询。
func (r *ArchiveProcessSnapshotRepo) GetMapByProcessIDs(c *gin.Context, processIDs []string) (map[string]*model.ArchiveProcessSnapshot, error) {
	if len(processIDs) == 0 {
		return map[string]*model.ArchiveProcessSnapshot{}, nil
	}
	var rows []model.ArchiveProcessSnapshot
	if err := r.WithTenant(c).Where("process_id IN ?", processIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[string]*model.ArchiveProcessSnapshot, len(rows))
	for i := range rows {
		out[rows[i].ProcessID] = &rows[i]
	}
	return out, nil
}
