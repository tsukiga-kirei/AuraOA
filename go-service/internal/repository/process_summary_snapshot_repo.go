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
	now := apptime.Now()
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

// GetMapByProcessIDs 批量查询流程总结快照。
func (r *ProcessSummarySnapshotRepo) GetMapByProcessIDs(c *gin.Context, processIDs []string) (map[string]*model.ProcessSummarySnapshot, error) {
	result := make(map[string]*model.ProcessSummarySnapshot)
	if len(processIDs) == 0 {
		return result, nil
	}
	var rows []model.ProcessSummarySnapshot
	if err := r.WithTenant(c).Where("process_id IN ?", processIDs).Find(&rows).Error; err != nil {
		return nil, err
	}
	for i := range rows {
		result[rows[i].ProcessID] = &rows[i]
	}
	return result, nil
}

type ProcessSummarySnapshotFilter struct {
	Channel          string // workbench / embed / "" = 全部
	Keyword          string
	ProcessType      string
	Operator         string
	Department       string
	StartDate        *time.Time
	EndDateExclusive *time.Time // 次日零点，不包含该时刻
}

type ProcessSummarySnapshotListRow struct {
	model.ProcessSummarySnapshot
	Channel    string     `json:"channel" gorm:"column:channel"`
	UserID     *uuid.UUID `json:"user_id,omitempty" gorm:"column:user_id"`
	Operator   string     `json:"operator" gorm:"column:operator"`
	Department string     `json:"department" gorm:"column:department"`
}

type ProcessSummarySnapshotStats struct {
	Total      int64 `json:"total"`
	BlockCount int64 `json:"block_count"`
}

// adminSummaryGroups 从有效日志重建展示分组，历史记录自动纳入，无需改写共享快照。
// 嵌入按流程汇总，系统内按流程与操作人汇总，各组使用最新有效记录。
func (r *ProcessSummarySnapshotRepo) adminSummaryGroups(c *gin.Context) *gorm.DB {
	tenantID, _ := c.Get("tenant_id")
	const sql = `WITH classified AS (
 SELECT psl.*,
 CASE WHEN trigger_source IN ('summary_embed_auto', 'summary_embed_manual') THEN 'embed' ELSE 'workbench' END AS channel,
 CASE WHEN trigger_source IN ('summary_embed_auto', 'summary_embed_manual') THEN NULL::uuid ELSE user_id END AS group_user_id
 FROM process_summary_logs psl
 WHERE tenant_id = ? AND status = 'completed' AND COALESCE(parse_error, '') = ''
 ), ranked AS (
 SELECT classified.*, ROW_NUMBER() OVER (PARTITION BY process_id, channel, group_user_id ORDER BY created_at DESC, id DESC) AS rn,
 jsonb_agg(id::text) OVER (PARTITION BY process_id, channel, group_user_id ORDER BY created_at ASC, id ASC ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) AS valid_log_ids
 FROM classified
 )
 SELECT r.id, r.tenant_id, r.process_id, r.channel, r.group_user_id AS user_id,
 r.valid_log_ids, r.id AS latest_valid_log_id, r.title, r.process_type,
 CASE WHEN jsonb_typeof(r.summary_result->'blocks') = 'array' THEN jsonb_array_length(r.summary_result->'blocks') ELSE 0 END AS block_count,
 r.created_at, r.updated_at,
 CASE WHEN r.channel = 'embed' THEN 'OA 嵌入总结' ELSE COALESCE(u.display_name, u.username, '') END AS operator,
 CASE WHEN r.channel = 'embed' THEN '系统自动' ELSE COALESCE(d.name, '') END AS department
 FROM ranked r
 LEFT JOIN users u ON u.id = r.group_user_id
 LEFT JOIN org_members om ON om.user_id = r.group_user_id AND om.tenant_id = r.tenant_id AND om.status = 'active'
 LEFT JOIN departments d ON d.id = om.department_id AND d.tenant_id = r.tenant_id
 WHERE r.rn = 1`
	return r.DB.Table("(?) AS summary_groups", r.DB.Raw(sql, tenantID))
}

func (r *ProcessSummarySnapshotRepo) ListPagedWithUser(c *gin.Context, filter ProcessSummarySnapshotFilter, page, pageSize int) ([]ProcessSummarySnapshotListRow, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	base := applyProcessSummarySnapshotFilter(r.adminSummaryGroups(c), filter)
	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []ProcessSummarySnapshotListRow
	err := base.Order("created_at DESC, id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error
	return items, total, err
}

func (r *ProcessSummarySnapshotRepo) CountStats(c *gin.Context, channel string) (*ProcessSummarySnapshotStats, error) {
	base := applyProcessSummarySnapshotFilter(r.adminSummaryGroups(c), ProcessSummarySnapshotFilter{Channel: channel})
	var stats ProcessSummarySnapshotStats
	err := base.Select("COUNT(*) AS total, COALESCE(SUM(block_count), 0)::bigint AS block_count").Scan(&stats).Error
	return &stats, err
}

func applyProcessSummarySnapshotFilter(db *gorm.DB, f ProcessSummarySnapshotFilter) *gorm.DB {
	if f.Channel != "" {
		db = db.Where("channel = ?", f.Channel)
	}
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		db = db.Where("(title ILIKE ? OR process_id ILIKE ?)", like, like)
	}
	if f.ProcessType != "" {
		db = db.Where("process_type IN ?", strings.Split(f.ProcessType, ","))
	}
	if f.Operator != "" {
		db = db.Where("operator ILIKE ?", "%"+f.Operator+"%")
	}
	if f.Department != "" {
		db = db.Where("department = ?", f.Department)
	}
	if f.StartDate != nil {
		db = db.Where("updated_at >= ?", f.StartDate)
	}
	if f.EndDateExclusive != nil {
		db = db.Where("updated_at < ?", f.EndDateExclusive)
	}
	return db
}
