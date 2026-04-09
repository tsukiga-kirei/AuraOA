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

// ── 仪表盘查询辅助类型 ──────────────────────────────────────────────────────

// ArchiveSnapshotEnrichedRow 带操作人信息的归档快照行（用于最近动态）。
type ArchiveSnapshotEnrichedRow struct {
	ID              uuid.UUID `gorm:"column:id"`
	Title           string    `gorm:"column:title"`
	Compliance      string    `gorm:"column:compliance"`
	ComplianceScore int       `gorm:"column:compliance_score"`
	UserName        string    `gorm:"column:user_name"`
	CreatedAt       time.Time `gorm:"column:created_at"`
}

// TenantFailedCount 按租户统计失败数。
type TenantFailedCount struct {
	TenantID uuid.UUID `gorm:"column:tenant_id"`
	Count    int64     `gorm:"column:count"`
}

// ── 仪表盘查询方法 ──────────────────────────────────────────────────────────

// CountThisWeek 本周（周一 00:00 UTC 至今）归档快照条数。
// userID 非 nil 时 JOIN archive_logs 按 user_id 过滤。
func (r *ArchiveProcessSnapshotRepo) CountThisWeek(c *gin.Context, userID *uuid.UUID) (int64, error) {
	var count int64

	tenantID, _ := c.Get("tenant_id")
	q := r.DB.Table("archive_process_snapshots AS aps")
	if tenantID != nil && tenantID != "" {
		q = q.Where("aps.tenant_id = ?", tenantID)
	}
	if userID != nil {
		q = q.Joins("JOIN archive_logs arl ON arl.id = aps.latest_valid_archive_log_id").
			Where("arl.user_id = ?", *userID)
	}
	err := q.Where("aps.updated_at >= date_trunc('week', CURRENT_TIMESTAMP AT TIME ZONE 'UTC')").
		Count(&count).Error
	return count, err
}

// WeeklyTrendByDay 本周每天的归档快照条数（generate_series 填充无数据日期）。
func (r *ArchiveProcessSnapshotRepo) WeeklyTrendByDay(c *gin.Context, userID *uuid.UUID) ([]DayCount, error) {
	tenantID, _ := c.Get("tenant_id")

	userFilter := ""
	args := []interface{}{tenantID}
	if userID != nil {
		userFilter = "AND arl.user_id = ?"
		args = append(args, *userID)
	}

	sql := `
WITH days AS (
  SELECT generate_series(
    date_trunc('week', CURRENT_DATE AT TIME ZONE 'UTC')::date,
    (CURRENT_DATE AT TIME ZONE 'UTC')::date,
    INTERVAL '1 day'
  )::date AS d
)
SELECT TO_CHAR(days.d, 'MM-DD') AS date,
       COALESCE(b.cnt, 0)::bigint AS count
FROM days
LEFT JOIN (
  SELECT DATE(aps.updated_at AT TIME ZONE 'UTC') AS d,
         COUNT(*)::bigint AS cnt
  FROM archive_process_snapshots aps
  ` + func() string {
		if userID != nil {
			return "JOIN archive_logs arl ON arl.id = aps.latest_valid_archive_log_id"
		}
		return ""
	}() + `
  WHERE aps.tenant_id = ?
    AND aps.updated_at >= date_trunc('week', CURRENT_TIMESTAMP AT TIME ZONE 'UTC')
    ` + userFilter + `
  GROUP BY 1
) b ON b.d = days.d
ORDER BY days.d`

	var rows []DayCount
	err := r.DB.Raw(sql, args...).Scan(&rows).Error
	return rows, err
}

// RecentEnriched 最近 N 条归档快照（带 compliance + compliance_score + 操作人信息）。
func (r *ArchiveProcessSnapshotRepo) RecentEnriched(c *gin.Context, limit int, userID *uuid.UUID) ([]ArchiveSnapshotEnrichedRow, error) {
	tenantID, _ := c.Get("tenant_id")

	userFilter := ""
	args := []interface{}{tenantID}
	if userID != nil {
		userFilter = "AND arl.user_id = ?"
		args = append(args, *userID)
	}
	args = append(args, limit)

	sql := `
SELECT aps.id,
       aps.title,
       aps.compliance,
       aps.compliance_score,
       COALESCE(u.display_name, u.username, '') AS user_name,
       aps.updated_at AS created_at
FROM archive_process_snapshots aps
LEFT JOIN archive_logs arl ON arl.id = aps.latest_valid_archive_log_id
LEFT JOIN users u ON u.id = arl.user_id
WHERE aps.tenant_id = ?
  ` + userFilter + `
ORDER BY aps.updated_at DESC
LIMIT ?`

	var rows []ArchiveSnapshotEnrichedRow
	err := r.DB.Raw(sql, args...).Scan(&rows).Error
	return rows, err
}

// CountByDepartment 按部门统计归档快照数（tenant_admin 用）。
func (r *ArchiveProcessSnapshotRepo) CountByDepartment(c *gin.Context) ([]DeptCount, error) {
	tenantID, _ := c.Get("tenant_id")

	sql := `
SELECT COALESCE(d.name, '未分配') AS department,
       COUNT(*)::bigint AS count
FROM archive_process_snapshots aps
JOIN archive_logs arl ON arl.id = aps.latest_valid_archive_log_id
JOIN users u ON u.id = arl.user_id
LEFT JOIN org_members om ON om.user_id = u.id AND om.tenant_id = aps.tenant_id AND om.status = 'active'
LEFT JOIN departments d ON d.id = om.department_id AND d.tenant_id = aps.tenant_id
WHERE aps.tenant_id = ?
GROUP BY d.name
ORDER BY count DESC`

	var rows []DeptCount
	err := r.DB.Raw(sql, tenantID).Scan(&rows).Error
	return rows, err
}

// CountByTenantGlobal 全平台按租户统计归档快照数（system_admin 用，无 tenant_id 过滤）。
func (r *ArchiveProcessSnapshotRepo) CountByTenantGlobal() ([]TenantSnapshotCount, error) {
	sql := `
SELECT tenant_id,
       COUNT(*)::bigint AS count
FROM archive_process_snapshots
GROUP BY tenant_id
ORDER BY count DESC`

	var rows []TenantSnapshotCount
	err := r.DB.Raw(sql).Scan(&rows).Error
	return rows, err
}

// CountFailedByTenantGlobal 全平台按租户统计归档失败数（从 archive_logs 查 status='failed'）。
func (r *ArchiveProcessSnapshotRepo) CountFailedByTenantGlobal() ([]TenantFailedCount, error) {
	sql := `
SELECT tenant_id,
       COUNT(*)::bigint AS count
FROM archive_logs
WHERE status = 'failed'
GROUP BY tenant_id
ORDER BY count DESC`

	var rows []TenantFailedCount
	err := r.DB.Raw(sql).Scan(&rows).Error
	return rows, err
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
