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

// ── 数据管理页快照分页 ──────────────────────────────────────────────────────

// ArchiveSnapshotFilter 归档快照分页过滤条件。
type ArchiveSnapshotFilter struct {
	Compliance  string     // compliant / partially_compliant / non_compliant / "" = 全部
	Keyword     string
	ProcessType string
	Operator    string
	Department  string
	StartDate   *time.Time
	EndDate     *time.Time
}

// ArchiveSnapshotListRow 归档快照列表行（含操作人+部门）。
type ArchiveSnapshotListRow struct {
	model.ArchiveProcessSnapshot
	Operator   string `json:"operator" gorm:"column:operator"`
	Department string `json:"department" gorm:"column:department"`
}

// ArchiveSnapshotStats 归档快照分组统计。
type ArchiveSnapshotStats struct {
	Total        int64 `json:"total"`
	Compliant    int64 `json:"compliant"`
	Partial      int64 `json:"partial"`
	NonCompliant int64 `json:"non_compliant"`
}

// ListPagedWithUser 数据管理页：归档快照分页查询。
func (r *ArchiveProcessSnapshotRepo) ListPagedWithUser(c *gin.Context, filter ArchiveSnapshotFilter, page, pageSize int) ([]ArchiveSnapshotListRow, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}

	const t = "archive_process_snapshots"
	tenantID, _ := c.Get("tenant_id")
	base := r.DB.
		Where(t+".tenant_id = ?", tenantID).
		Table(t).
		Select(t+".*, "+
			"COALESCE(u.display_name, u.username, '') AS operator, "+
			"COALESCE(d.name, '') AS department").
		Joins("LEFT JOIN archive_logs arl ON arl.id = "+t+".latest_valid_archive_log_id").
		Joins("LEFT JOIN users u ON u.id = arl.user_id").
		Joins("LEFT JOIN org_members om ON om.user_id = arl.user_id AND om.tenant_id = "+t+".tenant_id AND om.status = 'active'").
		Joins("LEFT JOIN departments d ON d.id = om.department_id AND d.tenant_id = "+t+".tenant_id")

	base = applyArchiveSnapshotFilter(base, filter)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []ArchiveSnapshotListRow
	err := base.Order(t + ".updated_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error
	return items, total, err
}

// CountStatsByCompliance 归档快照分组统计。
func (r *ArchiveProcessSnapshotRepo) CountStatsByCompliance(c *gin.Context) (*ArchiveSnapshotStats, error) {
	type row struct {
		Compliance string
		Cnt        int64
	}
	var rows []row
	err := r.WithTenant(c).
		Table("archive_process_snapshots").
		Select("compliance, COUNT(*) as cnt").
		Group("compliance").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	stats := &ArchiveSnapshotStats{}
	for _, rw := range rows {
		stats.Total += rw.Cnt
		switch rw.Compliance {
		case "compliant":
			stats.Compliant += rw.Cnt
		case "partially_compliant":
			stats.Partial += rw.Cnt
		case "non_compliant":
			stats.NonCompliant += rw.Cnt
		}
	}
	return stats, nil
}

func applyArchiveSnapshotFilter(db *gorm.DB, f ArchiveSnapshotFilter) *gorm.DB {
	const t = "archive_process_snapshots."
	if f.Compliance != "" {
		db = db.Where(t+"compliance = ?", f.Compliance)
	}
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		db = db.Where("("+t+"title ILIKE ? OR "+t+"process_id ILIKE ?)", like, like)
	}
	if f.ProcessType != "" {
		db = db.Where(t+"process_type = ?", f.ProcessType)
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
