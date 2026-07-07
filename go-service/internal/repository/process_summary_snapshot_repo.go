package repository

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"

	"auraoa/go-service/internal/model"
)

// ProcessSummarySnapshotRepo 总结有效结果快照数据访问层。
type ProcessSummarySnapshotRepo struct {
	*BaseRepo
}

func NewProcessSummarySnapshotRepo(db *gorm.DB) *ProcessSummarySnapshotRepo {
	return &ProcessSummarySnapshotRepo{BaseRepo: NewBaseRepo(db)}
}

func (r *ProcessSummarySnapshotRepo) UpsertAppendValid(c *gin.Context, tenantID uuid.UUID, processID string, logID uuid.UUID, title, processType string, blockCount int) error {
	var existing model.ProcessSummarySnapshot
	err := r.WithTenant(c).Where("process_id = ?", processID).First(&existing).Error
	now := time.Now()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ids := []string{logID.String()}
		b, _ := json.Marshal(ids)
		row := &model.ProcessSummarySnapshot{
			TenantID:         tenantID,
			ProcessID:        processID,
			ValidLogIDs:      datatypes.JSON(b),
			LatestValidLogID: logID,
			Title:            title,
			ProcessType:      processType,
			BlockCount:       blockCount,
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
	return r.WithTenant(c).Model(&model.ProcessSummarySnapshot{}).Where("id = ?", existing.ID).Updates(map[string]interface{}{
		"valid_log_ids":       datatypes.JSON(b),
		"latest_valid_log_id": logID,
		"title":               title,
		"process_type":        processType,
		"block_count":         blockCount,
		"updated_at":          now,
	}).Error
}

func (r *ProcessSummarySnapshotRepo) GetByProcessID(c *gin.Context, processID string) (*model.ProcessSummarySnapshot, error) {
	var row model.ProcessSummarySnapshot
	err := r.WithTenant(c).Where("process_id = ?", processID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

type ProcessSummarySnapshotFilter struct {
	Keyword     string
	ProcessType string
	Operator    string
	Department  string
	StartDate   *time.Time
	EndDate     *time.Time
}

type ProcessSummarySnapshotListRow struct {
	model.ProcessSummarySnapshot
	Operator   string `json:"operator" gorm:"column:operator"`
	Department string `json:"department" gorm:"column:department"`
}

type ProcessSummarySnapshotStats struct {
	Total      int64 `json:"total"`
	BlockCount int64 `json:"block_count"`
}

func (r *ProcessSummarySnapshotRepo) ListPagedWithUser(c *gin.Context, filter ProcessSummarySnapshotFilter, page, pageSize int) ([]ProcessSummarySnapshotListRow, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	const t = "process_summary_snapshots"
	tenantID, _ := c.Get("tenant_id")
	base := r.DB.
		Where(t+".tenant_id = ?", tenantID).
		Table(t).
		Select(t + ".*, " +
			"COALESCE(u.display_name, u.username, '') AS operator, " +
			"COALESCE(d.name, '') AS department").
		Joins("LEFT JOIN process_summary_logs psl ON psl.id = " + t + ".latest_valid_log_id").
		Joins("LEFT JOIN users u ON u.id = psl.user_id").
		Joins("LEFT JOIN org_members om ON om.user_id = psl.user_id AND om.tenant_id = " + t + ".tenant_id AND om.status = 'active'").
		Joins("LEFT JOIN departments d ON d.id = om.department_id AND d.tenant_id = " + t + ".tenant_id")

	base = applyProcessSummarySnapshotFilter(base, filter)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []ProcessSummarySnapshotListRow
	err := base.Order(t + ".updated_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error
	return items, total, err
}

func (r *ProcessSummarySnapshotRepo) CountStats(c *gin.Context) (*ProcessSummarySnapshotStats, error) {
	var stats ProcessSummarySnapshotStats
	err := r.WithTenant(c).
		Table("process_summary_snapshots").
		Select("COUNT(*) AS total, COALESCE(SUM(block_count), 0)::bigint AS block_count").
		Scan(&stats).Error
	return &stats, err
}

func applyProcessSummarySnapshotFilter(db *gorm.DB, f ProcessSummarySnapshotFilter) *gorm.DB {
	const t = "process_summary_snapshots."
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		db = db.Where("("+t+"title ILIKE ? OR "+t+"process_id ILIKE ?)", like, like)
	}
	if f.ProcessType != "" {
		types := strings.Split(f.ProcessType, ",")
		db = db.Where(t+"process_type IN ?", types)
	}
	if f.Operator != "" {
		like := "%" + f.Operator + "%"
		db = db.Where("(u.display_name ILIKE ? OR u.username ILIKE ?)", like, like)
	}
	if f.Department != "" {
		db = db.Where("d.name = ?", f.Department)
	}
	if f.StartDate != nil {
		db = db.Where(t+"updated_at >= ?", f.StartDate)
	}
	if f.EndDate != nil {
		db = db.Where(t+"updated_at <= ?", f.EndDate)
	}
	return db
}
