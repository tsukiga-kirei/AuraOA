package repository

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"oa-smart-audit/go-service/internal/model"
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

// DashboardLLMDailyPointRow LLM 按日聚合行（UTC 日期）。
type DashboardLLMDailyPointRow struct {
	Date  string `gorm:"column:date"`
	AvgMs int64  `gorm:"column:avg_ms"`
	Calls int64  `gorm:"column:calls"`
}

// DashboardLLMWeeklyTrend 最近 n 个 UTC 自然日 LLM 调用：日均耗时与次数。
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
    (CURRENT_DATE AT TIME ZONE 'UTC')::date - ($2::int - 1),
    (CURRENT_DATE AT TIME ZONE 'UTC')::date,
    INTERVAL '1 day'
  )::date AS d
)
SELECT TO_CHAR(days.d, 'MM-DD') AS date,
       COALESCE(b.avg_ms, 0)::bigint AS avg_ms,
       COALESCE(b.calls, 0)::bigint AS calls
FROM days
LEFT JOIN (
  SELECT DATE(created_at AT TIME ZONE 'UTC') AS d,
         COALESCE(AVG(duration_ms), 0)::bigint AS avg_ms,
         COUNT(*)::bigint AS calls
  FROM tenant_llm_message_logs
  WHERE tenant_id = $1
  GROUP BY 1
) b ON b.d = days.d
ORDER BY days.d
`
	var rows []DashboardLLMDailyPointRow
	err = r.DB.Raw(q, tenantUUID, days).Scan(&rows).Error
	return rows, err
}

// DashboardLLMWeeklyTrendGlobal 全库最近 n 个 UTC 自然日 LLM 调用趋势。
func (r *LLMMessageLogRepo) DashboardLLMWeeklyTrendGlobal(days int) ([]DashboardLLMDailyPointRow, error) {
	if days < 1 {
		days = 7
	}
	q := `
WITH days AS (
  SELECT generate_series(
    (CURRENT_DATE AT TIME ZONE 'UTC')::date - ($1::int - 1),
    (CURRENT_DATE AT TIME ZONE 'UTC')::date,
    INTERVAL '1 day'
  )::date AS d
)
SELECT TO_CHAR(days.d, 'MM-DD') AS date,
       COALESCE(b.avg_ms, 0)::bigint AS avg_ms,
       COALESCE(b.calls, 0)::bigint AS calls
FROM days
LEFT JOIN (
  SELECT DATE(created_at AT TIME ZONE 'UTC') AS d,
         COALESCE(AVG(duration_ms), 0)::bigint AS avg_ms,
         COUNT(*)::bigint AS calls
  FROM tenant_llm_message_logs
  GROUP BY 1
) b ON b.d = days.d
ORDER BY days.d
`
	var rows []DashboardLLMDailyPointRow
	err := r.DB.Raw(q, days).Scan(&rows).Error
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
