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

// ── 数据管理页快照分页 ──────────────────────────────────────────────────────

// AuditSnapshotFilter 快照分页过滤条件。
type AuditSnapshotFilter struct {
	Recommendation string     // approve / return / review / "" = 全部
	Keyword        string     // 标题/流程编号模糊
	ProcessType    string
	Operator       string     // 操作人模糊
	Department     string     // 部门精确
	StartDate      *time.Time
	EndDate        *time.Time
}

// AuditSnapshotListRow 快照列表行（含操作人+部门）。
type AuditSnapshotListRow struct {
	model.AuditProcessSnapshot
	Operator   string `json:"operator" gorm:"column:operator"`
	Department string `json:"department" gorm:"column:department"`
}

// AuditSnapshotStats 快照分组统计。
type AuditSnapshotStats struct {
	Total        int64 `json:"total"`
	ApproveCount int64 `json:"approve_count"`
	ReturnCount  int64 `json:"return_count"`
	ReviewCount  int64 `json:"review_count"`
}

// ListPagedWithUser 数据管理页：快照分页查询，JOIN 最新审核日志→用户→组织→部门。
func (r *AuditProcessSnapshotRepo) ListPagedWithUser(c *gin.Context, filter AuditSnapshotFilter, page, pageSize int) ([]AuditSnapshotListRow, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 20
	}

	const t = "audit_process_snapshots"
	tenantID, _ := c.Get("tenant_id")
	base := r.DB.
		Where(t+".tenant_id = ?", tenantID).
		Table(t).
		Select(t+".*, "+
			"COALESCE(u.display_name, u.username, '') AS operator, "+
			"COALESCE(d.name, '') AS department").
		Joins("LEFT JOIN audit_logs al ON al.id = "+t+".latest_valid_log_id").
		Joins("LEFT JOIN users u ON u.id = al.user_id").
		Joins("LEFT JOIN org_members om ON om.user_id = al.user_id AND om.tenant_id = "+t+".tenant_id AND om.status = 'active'").
		Joins("LEFT JOIN departments d ON d.id = om.department_id AND d.tenant_id = "+t+".tenant_id")

	base = applyAuditSnapshotFilter(base, filter)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []AuditSnapshotListRow
	err := base.Order(t + ".updated_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error
	return items, total, err
}

// CountStatsByRecommendation 快照分组统计。
func (r *AuditProcessSnapshotRepo) CountStatsByRecommendation(c *gin.Context) (*AuditSnapshotStats, error) {
	type row struct {
		Recommendation string
		Cnt            int64
	}
	var rows []row
	err := r.WithTenant(c).
		Table("audit_process_snapshots").
		Select("recommendation, COUNT(*) as cnt").
		Group("recommendation").
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	stats := &AuditSnapshotStats{}
	for _, rw := range rows {
		stats.Total += rw.Cnt
		switch rw.Recommendation {
		case "approve":
			stats.ApproveCount += rw.Cnt
		case "return":
			stats.ReturnCount += rw.Cnt
		case "review":
			stats.ReviewCount += rw.Cnt
		}
	}
	return stats, nil
}

func applyAuditSnapshotFilter(db *gorm.DB, f AuditSnapshotFilter) *gorm.DB {
	const t = "audit_process_snapshots."
	if f.Recommendation != "" {
		db = db.Where(t+"recommendation = ?", f.Recommendation)
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
