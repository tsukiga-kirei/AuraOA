package repository

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"auraoa/go-service/internal/model"
	"auraoa/go-service/internal/pkg/apptime"
)

// LLMMessageLogRepo 提供租户大模型消息记录的数据访问方法。
type LLMMessageLogRepo struct {
	*BaseRepo
}

// NewLLMMessageLogRepo 创建一个新的 LLMMessageLogRepo 实例。
func NewLLMMessageLogRepo(db *gorm.DB) *LLMMessageLogRepo {
	return &LLMMessageLogRepo{BaseRepo: NewBaseRepo(db)}
}

// Create 写入一条大模型消息记录。
func (r *LLMMessageLogRepo) Create(log *model.TenantLLMMessageLog) error {
	return r.DB.Create(log).Error
}

// CreateWithPayload 写入一条大模型消息记录及其输入输出内容。
func (r *LLMMessageLogRepo) CreateWithPayload(log *model.TenantLLMMessageLog, payload *model.TenantLLMMessagePayload) error {
	return r.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(log).Error; err != nil {
			return err
		}
		if payload == nil {
			return nil
		}
		payload.LLMMessageLogID = log.ID
		payload.TenantID = log.TenantID
		if payload.CreatedAt.IsZero() {
			payload.CreatedAt = log.CreatedAt
		}
		return tx.Create(payload).Error
	})
}

// TokenUsageSummary Token 消耗统计汇总结构。
type TokenUsageSummary struct {
	TenantID      uuid.UUID `json:"tenant_id"`
	ModelConfigID uuid.UUID `json:"model_config_id"`
	TotalInput    int64     `json:"total_input"`
	TotalOutput   int64     `json:"total_output"`
	TotalTokens   int64     `json:"total_tokens"`
	CallCount     int64     `json:"call_count"`
}

// QueryByTimeRange 按时间范围和可选模型筛选查询 Token 消耗统计。
func (r *LLMMessageLogRepo) QueryByTimeRange(c *gin.Context, startTime, endTime time.Time, modelConfigID *uuid.UUID) ([]TokenUsageSummary, error) {
	query := r.WithTenant(c).Model(&model.TenantLLMMessageLog{}).
		Select("tenant_id, model_config_id, SUM(input_tokens) as total_input, SUM(output_tokens) as total_output, SUM(total_tokens) as total_tokens, COUNT(*) as call_count").
		Where("created_at >= ? AND created_at <= ?", startTime, endTime)

	if modelConfigID != nil {
		query = query.Where("model_config_id = ?", *modelConfigID)
	}

	query = query.Group("tenant_id, model_config_id")

	var summaries []TokenUsageSummary
	if err := query.Find(&summaries).Error; err != nil {
		return nil, err
	}
	return summaries, nil
}

// QueryAllTenantsTokenUsage 查询所有租户的 Token 消耗统计（system_admin 用）。
func (r *LLMMessageLogRepo) QueryAllTenantsTokenUsage(startTime, endTime time.Time) ([]TokenUsageSummary, error) {
	var summaries []TokenUsageSummary
	err := r.DB.Model(&model.TenantLLMMessageLog{}).
		Select("tenant_id, model_config_id, SUM(input_tokens) as total_input, SUM(output_tokens) as total_output, SUM(total_tokens) as total_tokens, COUNT(*) as call_count").
		Where("created_at >= ? AND created_at <= ?", startTime, endTime).
		Group("tenant_id, model_config_id").
		Find(&summaries).Error
	if err != nil {
		return nil, err
	}
	return summaries, nil
}

// DashboardLLMDailyPointRow LLM 按日聚合行（应用配置时区日期）。
type DashboardLLMDailyPointRow struct {
	Date  string `gorm:"column:date"`
	AvgMs int64  `gorm:"column:avg_ms"`
	Calls int64  `gorm:"column:calls"`
}

// DashboardLLMWeeklyTrend 最近 n 个应用配置时区自然日 LLM 调用：日均耗时与次数。
func (r *LLMMessageLogRepo) DashboardLLMWeeklyTrend(c *gin.Context, days int) ([]DashboardLLMDailyPointRow, error) {
	if days < 1 {
		days = 7
	}
	tid, ok := c.Get("tenant_id")
	if !ok || tid == nil || tid == "" {
		return nil, ErrNoTenantContext
	}
	tenantUUID, err := uuid.Parse(tid.(string))
	if err != nil {
		return nil, err
	}

	q := `
WITH days AS (
  SELECT generate_series(
    (CURRENT_TIMESTAMP AT TIME ZONE $3)::date - ($2::int - 1),
    (CURRENT_TIMESTAMP AT TIME ZONE $3)::date,
    INTERVAL '1 day'
  )::date AS d
)
SELECT TO_CHAR(days.d, 'MM-DD') AS date,
       COALESCE(b.avg_ms, 0)::bigint AS avg_ms,
       COALESCE(b.calls, 0)::bigint AS calls
FROM days
LEFT JOIN (
  SELECT DATE(created_at AT TIME ZONE $3) AS d,
         COALESCE(AVG(duration_ms), 0)::bigint AS avg_ms,
         COUNT(*)::bigint AS calls
  FROM tenant_llm_message_logs
  WHERE tenant_id = $1
  GROUP BY 1
) b ON b.d = days.d
ORDER BY days.d
`
	var rows []DashboardLLMDailyPointRow
	err = r.DB.Raw(q, tenantUUID, days, apptime.Name()).Scan(&rows).Error
	return rows, err
}

// DashboardLLMWeeklyTrendGlobal 全库最近 n 个应用配置时区自然日 LLM 调用趋势。
func (r *LLMMessageLogRepo) DashboardLLMWeeklyTrendGlobal(days int) ([]DashboardLLMDailyPointRow, error) {
	if days < 1 {
		days = 7
	}
	q := `
WITH days AS (
  SELECT generate_series(
    (CURRENT_TIMESTAMP AT TIME ZONE $2)::date - ($1::int - 1),
    (CURRENT_TIMESTAMP AT TIME ZONE $2)::date,
    INTERVAL '1 day'
  )::date AS d
)
SELECT TO_CHAR(days.d, 'MM-DD') AS date,
       COALESCE(b.avg_ms, 0)::bigint AS avg_ms,
       COALESCE(b.calls, 0)::bigint AS calls
FROM days
LEFT JOIN (
  SELECT DATE(created_at AT TIME ZONE $2) AS d,
         COALESCE(AVG(duration_ms), 0)::bigint AS avg_ms,
         COUNT(*)::bigint AS calls
  FROM tenant_llm_message_logs
  GROUP BY 1
) b ON b.d = days.d
ORDER BY days.d
`
	var rows []DashboardLLMDailyPointRow
	err := r.DB.Raw(q, days, apptime.Name()).Scan(&rows).Error
	return rows, err
}

// DashboardLLMOverallStats 租户 LLM 调用总次数与平均耗时。
func (r *LLMMessageLogRepo) DashboardLLMOverallStats(c *gin.Context) (totalCalls int64, avgMs int64, err error) {
	type row struct {
		Calls int64 `gorm:"column:calls"`
		AvgMs int64 `gorm:"column:avg_ms"`
	}
	var out row
	err = r.WithTenant(c).
		Model(&model.TenantLLMMessageLog{}).
		Select("COUNT(*)::bigint AS calls, COALESCE(AVG(duration_ms), 0)::bigint AS avg_ms").
		Scan(&out).Error
	return out.Calls, out.AvgMs, err
}

// DashboardLLMOverallStatsGlobal 全库 LLM 调用总次数与平均耗时。
func (r *LLMMessageLogRepo) DashboardLLMOverallStatsGlobal() (totalCalls int64, avgMs int64, err error) {
	type row struct {
		Calls int64 `gorm:"column:calls"`
		AvgMs int64 `gorm:"column:avg_ms"`
	}
	var out row
	err = r.DB.
		Model(&model.TenantLLMMessageLog{}).
		Select("COUNT(*)::bigint AS calls, COALESCE(AVG(duration_ms), 0)::bigint AS avg_ms").
		Scan(&out).Error
	return out.Calls, out.AvgMs, err
}

// AIModelCallStats 按模型+调用类型分组的 AI 调用统计行。
type AIModelCallStats struct {
	ModelConfigID string `gorm:"column:model_config_id"`
	ModelName     string `gorm:"column:model_name"`
	DisplayName   string `gorm:"column:display_name"`
	Provider      string `gorm:"column:provider"`
	CallType      string `gorm:"column:call_type"`
	Calls         int64  `gorm:"column:calls"`
	AvgMs         int64  `gorm:"column:avg_ms"`
}

// DashboardAIPerformanceByModel 按模型+调用类型分组的 AI 性能统计（system_admin 用）。
func (r *LLMMessageLogRepo) DashboardAIPerformanceByModel() ([]AIModelCallStats, error) {
	sql := `
SELECT tl.model_config_id::text AS model_config_id,
       COALESCE(amc.model_name, '') AS model_name,
       COALESCE(amc.display_name, '') AS display_name,
       COALESCE(amc.provider, '') AS provider,
       tl.call_type,
       COUNT(*)::bigint AS calls,
       COALESCE(AVG(tl.duration_ms), 0)::bigint AS avg_ms
FROM tenant_llm_message_logs tl
LEFT JOIN ai_model_configs amc ON amc.id = tl.model_config_id
WHERE tl.model_config_id IS NOT NULL
GROUP BY tl.model_config_id, amc.model_name, amc.display_name, amc.provider, tl.call_type
ORDER BY calls DESC`

	var rows []AIModelCallStats
	err := r.DB.Raw(sql).Scan(&rows).Error
	return rows, err
}

// LLMLogFilter 数据管理页 AI 调用记录筛选条件。
type LLMLogFilter struct {
	RequestType string
	CallType    string
	Keyword     string
	Operator    string
	StartDate   *time.Time
	EndDate     *time.Time
}

// LLMProcessListRow 按流程聚合的列表行。
type LLMProcessListRow struct {
	ProcessID      string    `json:"process_id"`
	ProcessTitle   string    `json:"process_title"`
	CallCount      int64     `json:"call_count"`
	TotalTokens    int64     `json:"total_tokens"`
	LatestCallAt   time.Time `json:"latest_call_at"`
	LatestUserName string    `json:"latest_user_name"`
}

// LLMLogListRow 单条调用记录（不含大文本 payload）。
type LLMLogListRow struct {
	model.TenantLLMMessageLog
	UserName         string `gorm:"column:user_name" json:"user_name"`
	ModelName        string `gorm:"column:model_name" json:"model_name"`
	ModelDisplayName string `gorm:"column:model_display_name" json:"model_display_name"`
}

// LLMLogStats AI 调用记录统计（按流程维度）。
type LLMLogStats struct {
	Total        int64 `json:"total"`
	AuditCount   int64 `json:"audit_count"`
	ArchiveCount int64 `json:"archive_count"`
	SummaryCount int64 `json:"summary_count"`
}

// LLMLogDetailWithPayload 详情含输入输出提示词。
type LLMLogDetailWithPayload struct {
	LLMLogListRow
	SystemPrompt    string `json:"system_prompt"`
	UserPrompt      string `json:"user_prompt"`
	ResponseContent string `json:"response_content"`
}

// ListProcessesPaged 数据管理页：按流程聚合分页查询。
func (r *LLMMessageLogRepo) ListProcessesPaged(c *gin.Context, filter LLMLogFilter, page, pageSize int) ([]LLMProcessListRow, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	const t = "tenant_llm_message_logs"
	tenantID, _ := c.Get("tenant_id")
	base := r.DB.
		Table(t+" AS l").
		Where("l.tenant_id = ? AND l.process_id IS NOT NULL AND l.process_id <> ''", tenantID).
		Joins("LEFT JOIN users u ON u.id = l.user_id")
	base = applyLLMLogFilter(base, filter)

	countSub := base.Session(&gorm.Session{}).
		Select("l.process_id").
		Group("l.process_id")
	var total int64
	if err := r.DB.Table("(?) AS grouped", countSub).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var items []LLMProcessListRow
	err := base.
		Select(`l.process_id,
			(ARRAY_AGG(l.process_title ORDER BY l.created_at DESC))[1] AS process_title,
			COUNT(*)::bigint AS call_count,
			COALESCE(SUM(l.total_tokens), 0)::bigint AS total_tokens,
			MAX(l.created_at) AS latest_call_at,
			(ARRAY_AGG(COALESCE(u.display_name, u.username, '') ORDER BY l.created_at DESC))[1] AS latest_user_name`).
		Group("l.process_id").
		Order("latest_call_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&items).Error
	return items, total, err
}

// ListCallsByProcessID 查询指定流程的全部 AI 调用记录（时间倒序，含提示词）。
func (r *LLMMessageLogRepo) ListCallsByProcessID(c *gin.Context, processID string) ([]LLMLogDetailWithPayload, error) {
	const t = "tenant_llm_message_logs"
	tenantID, _ := c.Get("tenant_id")

	var items []LLMLogDetailWithPayload
	err := r.DB.
		Table(t).
		Select(t+".*, "+
			"COALESCE(u.display_name, u.username, '') AS user_name, "+
			"COALESCE(amc.model_name, '') AS model_name, "+
			"COALESCE(amc.display_name, '') AS model_display_name, "+
			"COALESCE(p.system_prompt, '') AS system_prompt, "+
			"COALESCE(p.user_prompt, '') AS user_prompt, "+
			"COALESCE(p.response_content, '') AS response_content").
		Joins("LEFT JOIN users u ON u.id = "+t+".user_id").
		Joins("LEFT JOIN ai_model_configs amc ON amc.id = "+t+".model_config_id").
		Joins("LEFT JOIN tenant_llm_message_payloads p ON p.llm_message_log_id = "+t+".id").
		Where(t+".tenant_id = ? AND "+t+".process_id = ?", tenantID, processID).
		Order(t + ".created_at DESC").
		Find(&items).Error
	return items, err
}

// CountStats 统计租户 AI 调用流程数量（按场景分布）。
func (r *LLMMessageLogRepo) CountStats(c *gin.Context) (*LLMLogStats, error) {
	type row struct {
		Total        int64 `gorm:"column:total"`
		AuditCount   int64 `gorm:"column:audit_count"`
		ArchiveCount int64 `gorm:"column:archive_count"`
		SummaryCount int64 `gorm:"column:summary_count"`
	}
	var out row
	err := r.WithTenant(c).
		Model(&model.TenantLLMMessageLog{}).
		Where("process_id IS NOT NULL AND process_id <> ''").
		Select(`
			COUNT(DISTINCT process_id)::bigint AS total,
			COUNT(DISTINCT process_id) FILTER (WHERE request_type = 'audit')::bigint AS audit_count,
			COUNT(DISTINCT process_id) FILTER (WHERE request_type = 'archive')::bigint AS archive_count,
			COUNT(DISTINCT process_id) FILTER (WHERE request_type = 'summary')::bigint AS summary_count`).
		Scan(&out).Error
	if err != nil {
		return nil, err
	}
	return &LLMLogStats{
		Total:        out.Total,
		AuditCount:   out.AuditCount,
		ArchiveCount: out.ArchiveCount,
		SummaryCount: out.SummaryCount,
	}, nil
}

// GetByIDWithPayload 按 ID 查询单条 AI 调用记录及输入输出内容。
func (r *LLMMessageLogRepo) GetByIDWithPayload(c *gin.Context, id uuid.UUID) (*LLMLogDetailWithPayload, error) {
	const t = "tenant_llm_message_logs"
	tenantID, _ := c.Get("tenant_id")

	var detail LLMLogDetailWithPayload
	err := r.DB.
		Table(t).
		Select(t+".*, "+
			"COALESCE(u.display_name, u.username, '') AS user_name, "+
			"COALESCE(amc.model_name, '') AS model_name, "+
			"COALESCE(amc.display_name, '') AS model_display_name, "+
			"COALESCE(p.system_prompt, '') AS system_prompt, "+
			"COALESCE(p.user_prompt, '') AS user_prompt, "+
			"COALESCE(p.response_content, '') AS response_content").
		Joins("LEFT JOIN users u ON u.id = "+t+".user_id").
		Joins("LEFT JOIN ai_model_configs amc ON amc.id = "+t+".model_config_id").
		Joins("LEFT JOIN tenant_llm_message_payloads p ON p.llm_message_log_id = "+t+".id").
		Where(t+".tenant_id = ? AND "+t+".id = ?", tenantID, id).
		First(&detail).Error
	if err != nil {
		return nil, err
	}
	return &detail, nil
}

func applyLLMLogFilter(db *gorm.DB, f LLMLogFilter) *gorm.DB {
	const t = "tenant_llm_message_logs"
	if f.RequestType != "" {
		db = db.Where("EXISTS (SELECT 1 FROM "+t+" x WHERE x.tenant_id = l.tenant_id AND x.process_id = l.process_id AND x.request_type = ?)", f.RequestType)
	}
	if f.CallType != "" {
		db = db.Where("l.call_type = ?", f.CallType)
	}
	if f.Keyword != "" {
		like := "%" + f.Keyword + "%"
		db = db.Where("(l.process_id ILIKE ? OR l.process_title ILIKE ?)", like, like)
	}
	if f.Operator != "" {
		like := "%" + f.Operator + "%"
		db = db.Where("(u.display_name ILIKE ? OR u.username ILIKE ?)", like, like)
	}
	if f.StartDate != nil {
		db = db.Where("l.created_at >= ?", f.StartDate)
	}
	if f.EndDate != nil {
		db = db.Where("l.created_at <= ?", f.EndDate)
	}
	return db
}
