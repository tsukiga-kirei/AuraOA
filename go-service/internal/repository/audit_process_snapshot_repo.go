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
	"auraoa/go-service/internal/pkg/apptime"
)

// AuditProcessSnapshotRepo 审核有效结论快照数据访问层，按租户隔离。
type AuditProcessSnapshotRepo struct {
	*BaseRepo
}

// NewAuditProcessSnapshotRepo 创建 AuditProcessSnapshotRepo 实例。
func NewAuditProcessSnapshotRepo(db *gorm.DB) *AuditProcessSnapshotRepo {
	return &AuditProcessSnapshotRepo{BaseRepo: NewBaseRepo(db)}
}

// UpsertAppendValid 成功解析后追加日志 id 并更新最新有效结论（按 channel 隔离工作台与嵌入）。
func (r *AuditProcessSnapshotRepo) UpsertAppendValid(c *gin.Context, tenantID uuid.UUID, processID, channel string, logID uuid.UUID, title, processType, recommendation string, score, confidence int) error {
	if channel == "" {
		channel = model.AuditSnapshotChannelWorkbench
	}
	var existing model.AuditProcessSnapshot
	err := r.WithTenant(c).Where("process_id = ? AND channel = ?", processID, channel).First(&existing).Error
	now := apptime.Now()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		ids := []string{logID.String()}
		b, _ := json.Marshal(ids)
		row := &model.AuditProcessSnapshot{
			TenantID:         tenantID,
			ProcessID:        processID,
			Channel:          channel,
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

// GetByProcessID 单流程工作台渠道快照。
func (r *AuditProcessSnapshotRepo) GetByProcessID(c *gin.Context, processID string) (*model.AuditProcessSnapshot, error) {
	return r.GetByProcessIDAndChannel(c, processID, model.AuditSnapshotChannelWorkbench)
}

// GetByProcessIDAndChannel 单流程指定渠道快照。
func (r *AuditProcessSnapshotRepo) GetByProcessIDAndChannel(c *gin.Context, processID, channel string) (*model.AuditProcessSnapshot, error) {
	if channel == "" {
		channel = model.AuditSnapshotChannelWorkbench
	}
	var row model.AuditProcessSnapshot
	err := r.WithTenant(c).Where("process_id = ? AND channel = ?", processID, channel).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &row, err
}

// GetMapByProcessIDs 批量查询多个流程的工作台渠道快照。
func (r *AuditProcessSnapshotRepo) GetMapByProcessIDs(c *gin.Context, processIDs []string) (map[string]*model.AuditProcessSnapshot, error) {
	return r.GetMapByProcessIDsAndChannel(c, processIDs, model.AuditSnapshotChannelWorkbench)
}

// GetMapByProcessIDsAndChannel 批量查询指定渠道快照。
func (r *AuditProcessSnapshotRepo) GetMapByProcessIDsAndChannel(c *gin.Context, processIDs []string, channel string) (map[string]*model.AuditProcessSnapshot, error) {
	if len(processIDs) == 0 {
		return map[string]*model.AuditProcessSnapshot{}, nil
	}
	if channel == "" {
		channel = model.AuditSnapshotChannelWorkbench
	}
	var rows []model.AuditProcessSnapshot
	if err := r.WithTenant(c).Where("process_id IN ? AND channel = ?", processIDs, channel).Find(&rows).Error; err != nil {
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
	Recommendation   string // approve / return / review / "" = 全部
	Channel          string // workbench / embed / "" = 全部
	Keyword          string // 标题/流程编号模糊
	ProcessType      string
	Operator         string // 操作人模糊
	Department       string // 部门精确
	StartDate        *time.Time
	EndDateExclusive *time.Time // 次日零点，不包含该时刻
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

// buildAdminAggregatedBaseQuery 构造数据管理页三级渠道汇聚查询：
// 嵌入通用按流程唯一；系统内与嵌入个性化按（流程 + 操作人）唯一。
// 顶部统计卡片与列表完全基于此基础查询，保证数量 100% 对齐。
func (r *AuditProcessSnapshotRepo) buildAdminAggregatedBaseQuery(c *gin.Context, filter AuditSnapshotFilter) *gorm.DB {
	tenantID, _ := c.Get("tenant_id")
	baseSQL := `
WITH classified_logs AS (
    SELECT
        al.id,
        al.tenant_id,
        al.process_id,
        al.user_id,
        al.title,
        al.process_type,
        al.recommendation,
        al.score,
        al.confidence,
        al.created_at,
        al.updated_at,
        CASE
            WHEN COALESCE(al.trigger_detail, '') = 'personal_embed_manual' THEN 'embed_personal'
            WHEN al.trigger_source IN ('embed_auto', 'embed_manual') THEN 'embed_standard'
            ELSE 'workbench'
        END AS channel,
        CASE
            WHEN al.trigger_source IN ('embed_auto', 'embed_manual') AND COALESCE(al.trigger_detail, '') != 'personal_embed_manual'
                THEN al.process_id || ':embed_standard'
            WHEN COALESCE(al.trigger_detail, '') = 'personal_embed_manual'
                THEN al.process_id || ':' || al.user_id::text || ':embed_personal'
            ELSE
                al.process_id || ':' || al.user_id::text || ':workbench'
        END AS group_key
    FROM audit_logs al
    WHERE al.tenant_id = ?
      AND al.status = 'completed'
      AND COALESCE(al.parse_error, '') = ''
      AND al.recommendation IN ('approve', 'return', 'review')
),
grouped_summary AS (
    SELECT
        group_key,
        jsonb_agg(id::text ORDER BY created_at ASC) AS valid_log_ids,
        COUNT(*) AS audit_count
    FROM classified_logs
    GROUP BY group_key
),
ranked_logs AS (
    SELECT
        cl.*,
        ROW_NUMBER() OVER (PARTITION BY cl.group_key ORDER BY cl.created_at DESC, cl.id DESC) AS rn
    FROM classified_logs cl
)
SELECT
    rl.id,
    rl.tenant_id,
    rl.process_id,
    rl.channel,
    gs.valid_log_ids,
    rl.id AS latest_valid_log_id,
    rl.title,
    rl.process_type,
    rl.recommendation,
    rl.score,
    rl.confidence,
    rl.created_at,
    rl.updated_at,
    CASE
        WHEN rl.channel = 'embed_standard' THEN 'OA 嵌入审核'
        ELSE COALESCE(u.display_name, u.username, '')
    END AS operator,
    CASE
        WHEN rl.channel = 'embed_standard' THEN '系统自动'
        ELSE COALESCE(d.name, '')
    END AS department
FROM ranked_logs rl
JOIN grouped_summary gs ON gs.group_key = rl.group_key
LEFT JOIN users u ON u.id = rl.user_id
LEFT JOIN org_members om ON om.user_id = rl.user_id AND om.tenant_id = rl.tenant_id AND om.status = 'active'
LEFT JOIN departments d ON d.id = om.department_id AND d.tenant_id = rl.tenant_id
WHERE rl.rn = 1
`
	base := r.DB.Table("(?) AS agg", r.DB.Raw(baseSQL, tenantID))

	if filter.Channel != "" {
		if filter.Channel == model.AuditSnapshotChannelEmbed || filter.Channel == model.AuditSnapshotChannelEmbedStandard {
			base = base.Where("channel IN ('embed_standard', 'embed')")
		} else {
			base = base.Where("channel = ?", filter.Channel)
		}
	}
	if filter.Recommendation != "" {
		base = base.Where("recommendation = ?", filter.Recommendation)
	}
	if filter.Keyword != "" {
		like := "%" + filter.Keyword + "%"
		base = base.Where("(title ILIKE ? OR process_id ILIKE ?)", like, like)
	}
	if filter.ProcessType != "" {
		types := strings.Split(filter.ProcessType, ",")
		base = base.Where("process_type IN ?", types)
	}
	if filter.Operator != "" {
		like := "%" + filter.Operator + "%"
		base = base.Where("operator ILIKE ?", like)
	}
	if filter.Department != "" {
		base = base.Where("department = ?", filter.Department)
	}
	if filter.StartDate != nil {
		base = base.Where("updated_at >= ?", filter.StartDate)
	}
	if filter.EndDateExclusive != nil {
		base = base.Where("updated_at < ?", filter.EndDateExclusive)
	}
	return base
}

// ListPagedWithUser 数据管理页：三级渠道聚合快照分页查询。
func (r *AuditProcessSnapshotRepo) ListPagedWithUser(c *gin.Context, filter AuditSnapshotFilter, page, pageSize int) ([]AuditSnapshotListRow, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	base := r.buildAdminAggregatedBaseQuery(c, filter)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []AuditSnapshotListRow
	err := base.Order("updated_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error
	return items, total, err
}

// CountStatsByRecommendation 快照分组统计（与 ListPagedWithUser 复用相同汇聚逻辑，数值 100% 对齐）。
func (r *AuditProcessSnapshotRepo) CountStatsByRecommendation(c *gin.Context, channel string) (*AuditSnapshotStats, error) {
	base := r.buildAdminAggregatedBaseQuery(c, AuditSnapshotFilter{Channel: channel})

	type row struct {
		Recommendation string
		Cnt            int64
	}
	var rows []row
	err := base.Select("recommendation, COUNT(*) as cnt").
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

// ── 仪表盘查询辅助类型 ──────────────────────────────────────────────────────

// DayCount 每日计数（用于 WeeklyTrendByDay）。
type DayCount struct {
	Date  string `gorm:"column:date"`
	Count int64  `gorm:"column:count"`
}

// AuditSnapshotEnrichedRow 带操作人信息的快照行（用于最近动态）。
type AuditSnapshotEnrichedRow struct {
	ID             uuid.UUID `gorm:"column:id"`
	Title          string    `gorm:"column:title"`
	Recommendation string    `gorm:"column:recommendation"`
	Score          int       `gorm:"column:score"`
	UserName       string    `gorm:"column:user_name"`
	CreatedAt      time.Time `gorm:"column:created_at"`
}

// DeptCount 部门计数。
type DeptCount struct {
	Department string `gorm:"column:department"`
	Count      int64  `gorm:"column:count"`
}

// UserRankRow 用户快照数排名行。
type UserRankRow struct {
	Username    string    `gorm:"column:username"`
	DisplayName string    `gorm:"column:display_name"`
	Department  string    `gorm:"column:department"`
	AuditCount  int64     `gorm:"column:audit_count"`
	LastActive  time.Time `gorm:"column:last_active"`
}

// TenantSnapshotCount 按租户统计快照数。
type TenantSnapshotCount struct {
	TenantID uuid.UUID `gorm:"column:tenant_id"`
	Count    int64     `gorm:"column:count"`
}

// ── 仪表盘查询方法 ──────────────────────────────────────────────────────────

// CountThisWeek 本周（按应用配置时区周一 00:00 至今）快照条数。
// userID 非 nil 时 JOIN audit_logs 按 user_id 过滤。
func (r *AuditProcessSnapshotRepo) CountThisWeek(c *gin.Context, userID *uuid.UUID) (int64, error) {
	var count int64

	tenantID, _ := c.Get("tenant_id")
	q := r.DB.Table("audit_process_snapshots AS aps")
	if tenantID != nil && tenantID != "" {
		q = q.Where("aps.tenant_id = ? AND aps.channel = ?", tenantID, model.AuditSnapshotChannelWorkbench)
	}
	if userID != nil {
		q = q.Joins("JOIN audit_logs al ON al.id = aps.latest_valid_log_id").
			Where("al.user_id = ?", *userID)
	}
	err := q.Where("aps.updated_at >= date_trunc('week', CURRENT_TIMESTAMP AT TIME ZONE ?)", apptime.Name()).
		Count(&count).Error
	return count, err
}

// WeeklyTrendByDay 本周每天的快照条数（generate_series 填充无数据日期）。
func (r *AuditProcessSnapshotRepo) WeeklyTrendByDay(c *gin.Context, userID *uuid.UUID) ([]DayCount, error) {
	tenantID, _ := c.Get("tenant_id")

	userFilter := ""
	args := []interface{}{apptime.Name(), apptime.Name(), apptime.Name(), tenantID, apptime.Name()}
	if userID != nil {
		userFilter = "AND al.user_id = ?"
		args = append(args, *userID)
	}

	sql := `
WITH days AS (
  SELECT generate_series(
    date_trunc('week', CURRENT_TIMESTAMP AT TIME ZONE ?)::date,
    (CURRENT_TIMESTAMP AT TIME ZONE ?)::date,
    INTERVAL '1 day'
  )::date AS d
)
SELECT TO_CHAR(days.d, 'MM-DD') AS date,
       COALESCE(b.cnt, 0)::bigint AS count
FROM days
LEFT JOIN (
  SELECT DATE(aps.updated_at AT TIME ZONE ?) AS d,
         COUNT(*)::bigint AS cnt
  FROM audit_process_snapshots aps
  ` + func() string {
		if userID != nil {
			return "JOIN audit_logs al ON al.id = aps.latest_valid_log_id"
		}
		return ""
	}() + `
  WHERE aps.tenant_id = ?
    AND aps.channel = 'workbench'
    AND aps.updated_at >= date_trunc('week', CURRENT_TIMESTAMP AT TIME ZONE ?)
    ` + userFilter + `
  GROUP BY 1
) b ON b.d = days.d
ORDER BY days.d`

	var rows []DayCount
	err := r.DB.Raw(sql, args...).Scan(&rows).Error
	return rows, err
}

// RecentEnriched 最近 N 条快照（带 recommendation + score + 操作人信息）。
func (r *AuditProcessSnapshotRepo) RecentEnriched(c *gin.Context, limit int, userID *uuid.UUID) ([]AuditSnapshotEnrichedRow, error) {
	tenantID, _ := c.Get("tenant_id")

	userFilter := ""
	args := []interface{}{tenantID}
	if userID != nil {
		userFilter = "AND al.user_id = ?"
		args = append(args, *userID)
	}
	args = append(args, limit)

	sql := `
SELECT aps.id,
       aps.title,
       aps.recommendation,
       aps.score,
       COALESCE(u.display_name, u.username, '') AS user_name,
       aps.updated_at AS created_at
FROM audit_process_snapshots aps
LEFT JOIN audit_logs al ON al.id = aps.latest_valid_log_id
LEFT JOIN users u ON u.id = al.user_id
WHERE aps.tenant_id = ?
  AND aps.channel = 'workbench'
  ` + userFilter + `
ORDER BY aps.updated_at DESC
LIMIT ?`

	var rows []AuditSnapshotEnrichedRow
	err := r.DB.Raw(sql, args...).Scan(&rows).Error
	return rows, err
}

// CountByDepartment 按部门统计快照数（tenant_admin 用）。
func (r *AuditProcessSnapshotRepo) CountByDepartment(c *gin.Context) ([]DeptCount, error) {
	tenantID, _ := c.Get("tenant_id")

	sql := `
SELECT COALESCE(d.name, '未分配') AS department,
       COUNT(*)::bigint AS count
FROM audit_process_snapshots aps
JOIN audit_logs al ON al.id = aps.latest_valid_log_id
JOIN users u ON u.id = al.user_id
LEFT JOIN org_members om ON om.user_id = u.id AND om.tenant_id = aps.tenant_id AND om.status = 'active'
LEFT JOIN departments d ON d.id = om.department_id AND d.tenant_id = aps.tenant_id
WHERE aps.tenant_id = ?
  AND aps.channel = 'workbench'
GROUP BY d.name
ORDER BY count DESC`

	var rows []DeptCount
	err := r.DB.Raw(sql, tenantID).Scan(&rows).Error
	return rows, err
}

// CountByUserRanking 按用户统计有效快照数排名。
func (r *AuditProcessSnapshotRepo) CountByUserRanking(c *gin.Context, limit int) ([]UserRankRow, error) {
	tenantID, _ := c.Get("tenant_id")

	sql := `
SELECT u.username,
       u.display_name,
       COALESCE(d.name, '') AS department,
       COUNT(*)::bigint AS audit_count,
       MAX(aps.updated_at) AS last_active
FROM audit_process_snapshots aps
JOIN audit_logs al ON al.id = aps.latest_valid_log_id
JOIN users u ON u.id = al.user_id
LEFT JOIN org_members om ON om.user_id = u.id AND om.tenant_id = aps.tenant_id AND om.status = 'active'
LEFT JOIN departments d ON d.id = om.department_id AND d.tenant_id = aps.tenant_id
WHERE aps.tenant_id = ?
  AND aps.channel = 'workbench'
GROUP BY u.id, u.username, u.display_name, d.name
ORDER BY audit_count DESC, last_active DESC
LIMIT ?`

	var rows []UserRankRow
	err := r.DB.Raw(sql, tenantID, limit).Scan(&rows).Error
	return rows, err
}

// CountByTenantGlobal 全平台按租户统计快照数（system_admin 用，无 tenant_id 过滤）。
func (r *AuditProcessSnapshotRepo) CountByTenantGlobal() ([]TenantSnapshotCount, error) {
	sql := `
SELECT tenant_id,
       COUNT(*)::bigint AS count
FROM audit_process_snapshots
WHERE channel = 'workbench'
GROUP BY tenant_id
ORDER BY count DESC`

	var rows []TenantSnapshotCount
	err := r.DB.Raw(sql).Scan(&rows).Error
	return rows, err
}

// VisibleWorkbenchQuery 合并当前用户的有效个人结果与通用 OA 嵌入结果，个人结果优先。
// 仅用于已通过流程访问控制的工作台查询；每个流程只返回一条，避免跨渠道重复计数。
func (r *AuditProcessSnapshotRepo) VisibleWorkbenchQuery(c *gin.Context, userID uuid.UUID) *gorm.DB {
	return r.VisibleWorkbenchQueryScoped(c, userID, true)
}

// VisibleWorkbenchQueryScoped 按可见范围合并审核结果。
// includeEmbed 用于区分当前待办（可展示该流程的通用嵌入结论）与已完成列表（仅展示本人经手/个人定制结论）。
func (r *AuditProcessSnapshotRepo) VisibleWorkbenchQueryScoped(c *gin.Context, userID uuid.UUID, includeEmbed bool) *gorm.DB {
	tenantID, _ := c.Get("tenant_id")
	if !includeEmbed {
		// 已完成列表：仅展示本人触发的个人定制审核（监控流程）或系统内工作台审核，不泄露他人私有记录或无关通用记录
		candidates := r.DB.Raw(`
SELECT al.id, al.tenant_id, al.process_id, 'embed_personal' AS channel,
 jsonb_build_array(al.id::text) AS valid_log_ids, al.id AS latest_valid_log_id,
 al.title, al.process_type, al.recommendation, al.score, al.confidence, al.created_at, al.updated_at, 0 AS priority
FROM audit_logs al
WHERE al.tenant_id = ? AND al.user_id = ? AND al.trigger_detail = 'personal_embed_manual'
 AND al.status = 'completed' AND COALESCE(al.parse_error, '') = '' AND al.recommendation IN ('approve', 'return', 'review')
UNION ALL
SELECT al.id, al.tenant_id, al.process_id, 'workbench' AS channel,
 jsonb_build_array(al.id::text) AS valid_log_ids, al.id AS latest_valid_log_id,
 al.title, al.process_type, al.recommendation, al.score, al.confidence, al.created_at, al.updated_at, 1 AS priority
FROM audit_logs al
WHERE al.tenant_id = ? AND al.user_id = ? AND al.trigger_source NOT IN ('embed_auto', 'embed_manual')
 AND COALESCE(al.trigger_detail, '') != 'personal_embed_manual'
 AND al.status = 'completed' AND COALESCE(al.parse_error, '') = '' AND al.recommendation IN ('approve', 'return', 'review')
`, tenantID, userID, tenantID, userID)
		ranked := r.DB.Table("(?) AS candidates", candidates).Select("candidates.*, ROW_NUMBER() OVER (PARTITION BY process_id ORDER BY priority ASC, updated_at DESC, id DESC) AS row_num")
		return r.DB.Table("(?) AS visible", ranked).Where("row_num = 1")
	}

	candidates := r.DB.Raw(`
SELECT al.id, al.tenant_id, al.process_id, 'embed_personal' AS channel,
 jsonb_build_array(al.id::text) AS valid_log_ids, al.id AS latest_valid_log_id,
 al.title, al.process_type, al.recommendation, al.score, al.confidence, al.created_at, al.updated_at, 0 AS priority
FROM audit_logs al
WHERE al.tenant_id = ? AND al.user_id = ? AND al.trigger_detail = 'personal_embed_manual'
 AND al.status = 'completed' AND COALESCE(al.parse_error, '') = '' AND al.recommendation IN ('approve', 'return', 'review')
UNION ALL
SELECT al.id, al.tenant_id, al.process_id, 'workbench' AS channel,
 jsonb_build_array(al.id::text) AS valid_log_ids, al.id AS latest_valid_log_id,
 al.title, al.process_type, al.recommendation, al.score, al.confidence, al.created_at, al.updated_at, 1 AS priority
FROM audit_logs al
WHERE al.tenant_id = ? AND al.user_id = ? AND al.trigger_source NOT IN ('embed_auto', 'embed_manual')
 AND COALESCE(al.trigger_detail, '') != 'personal_embed_manual'
 AND al.status = 'completed' AND COALESCE(al.parse_error, '') = '' AND al.recommendation IN ('approve', 'return', 'review')
UNION ALL
SELECT al.id, al.tenant_id, al.process_id, 'embed' AS channel,
 jsonb_build_array(al.id::text) AS valid_log_ids, al.id AS latest_valid_log_id,
 al.title, al.process_type, al.recommendation, al.score, al.confidence, al.created_at, al.updated_at, 2 AS priority
FROM audit_logs al
JOIN process_audit_configs cfg ON cfg.tenant_id = al.tenant_id AND cfg.process_type = al.process_type
WHERE al.tenant_id = ? AND cfg.status = 'active' AND cfg.embed_enabled = true
 AND al.trigger_source IN ('embed_auto', 'embed_manual') AND COALESCE(al.trigger_detail, '') != 'personal_embed_manual'
 AND al.status = 'completed' AND COALESCE(al.parse_error, '') = '' AND al.recommendation IN ('approve', 'return', 'review')
`, tenantID, userID, tenantID, userID, tenantID)
	ranked := r.DB.Table("(?) AS candidates", candidates).Select("candidates.*, ROW_NUMBER() OVER (PARTITION BY process_id ORDER BY priority ASC, updated_at DESC, id DESC) AS row_num")
	return r.DB.Table("(?) AS visible", ranked).Where("row_num = 1")
}

// GetVisibleWorkbenchMap 批量返回个人优先且包含嵌入结果的快照视图。
func (r *AuditProcessSnapshotRepo) GetVisibleWorkbenchMap(c *gin.Context, processIDs []string, userID uuid.UUID) (map[string]*model.AuditProcessSnapshot, error) {
	result := make(map[string]*model.AuditProcessSnapshot)
	if len(processIDs) == 0 {
		return result, nil
	}
	var rows []model.AuditProcessSnapshot
	if err := r.VisibleWorkbenchQuery(c, userID).Where("process_id IN ?", processIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		result[rows[i].ProcessID] = &rows[i]
	}
	return result, nil
}

// CountCombinedUserRanking 将审核、归档、总结完成结果按用户汇总后统一排名。
func (r *AuditProcessSnapshotRepo) CountCombinedUserRanking(c *gin.Context, limit int) ([]CombinedUserRankRow, error) {
	tenantID, _ := c.Get("tenant_id")
	var rows []CombinedUserRankRow
	err := r.DB.Raw(`WITH activity AS (
 SELECT al.user_id, 1::bigint AS audit_count, 0::bigint AS archive_count, 0::bigint AS summary_count, aps.updated_at AS at
 FROM audit_process_snapshots aps JOIN audit_logs al ON al.id = aps.latest_valid_log_id AND al.tenant_id = aps.tenant_id WHERE aps.tenant_id = ?
 UNION ALL
 SELECT al.user_id, 0, 1, 0, aps.updated_at FROM archive_process_snapshots aps JOIN archive_logs al ON al.id = aps.latest_valid_archive_log_id AND al.tenant_id = aps.tenant_id WHERE aps.tenant_id = ?
 UNION ALL
 SELECT user_id, 0, 0, 1, updated_at FROM process_summary_logs WHERE tenant_id = ? AND status = 'completed'
 ) SELECT u.username, u.display_name, COALESCE(d.name, '') AS department,
 SUM(a.audit_count) AS audit_count, SUM(a.archive_count) AS archive_count, SUM(a.summary_count) AS summary_count,
 SUM(a.audit_count + a.archive_count + a.summary_count) AS total, MAX(a.at) AS last_active
 FROM activity a JOIN users u ON u.id = a.user_id
 LEFT JOIN org_members om ON om.user_id = u.id AND om.tenant_id = ? AND om.status = 'active'
 LEFT JOIN departments d ON d.id = om.department_id AND d.tenant_id = om.tenant_id
 GROUP BY u.id, u.username, u.display_name, d.name ORDER BY total DESC, last_active DESC LIMIT ?`, tenantID, tenantID, tenantID, tenantID, limit).Scan(&rows).Error
	return rows, err
}

// CombinedUserRankRow 多业务用户活跃度行。
type CombinedUserRankRow struct {
	UserRankRow
	ArchiveCount int64
	SummaryCount int64
	Total        int64
}
