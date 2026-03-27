package dto

// DashboardOverviewResponse 仪表盘聚合数据（GET /api/tenant/settings/dashboard-overview）。
type DashboardOverviewResponse struct {
	PendingOACount int `json:"pending_oa_count"`

	AuditSummary DashboardAuditSummary `json:"audit_summary"`

	WeeklyTrend []DashboardDayCount `json:"weekly_trend"`

	RecentActivity []DashboardActivityItem `json:"recent_activity"`

	ArchiveRecent []DashboardArchiveRow `json:"archive_recent"`

	DeptDistribution []DashboardDeptCount `json:"dept_distribution,omitempty"`

	AIPerformance *DashboardAIPerformance `json:"ai_performance,omitempty"`

	TenantUsage *DashboardTenantUsage `json:"tenant_usage,omitempty"`

	UserActivity []DashboardUserActivityRow `json:"user_activity,omitempty"`
}

// DashboardAuditSummary 审核概览数字。
type DashboardAuditSummary struct {
	Total     int64 `json:"total"`
	Approved  int64 `json:"approved"`
	Returned  int64 `json:"returned"`
	Archived  int64 `json:"archived"`
	Review    int64 `json:"review"`
	PendingAI int64 `json:"pending_ai"`
}

// DashboardDayCount 按日聚合（标签一般为 MM-DD）。
type DashboardDayCount struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

// DashboardActivityItem 最近动态（前端按 kind 做 i18n）。
type DashboardActivityItem struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"` // audit_completed | audit_failed | cron_log | archive_reviewed
	Title     string `json:"title"`
	UserName  string `json:"user_name"`
	CreatedAt string `json:"created_at"` // RFC3339
}

// DashboardArchiveRow 归档复盘小部件行。
type DashboardArchiveRow struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Compliance  string `json:"compliance"`
	UserName    string `json:"user_name"`
	CreatedAt   string `json:"created_at"`
}

// DashboardDeptCount 部门审核分布。
type DashboardDeptCount struct {
	Department string `json:"department"`
	Count      int64  `json:"count"`
}

// DashboardAIPerformance AI 调用聚合（租户管理员）。
type DashboardAIPerformance struct {
	AvgResponseMs int64                    `json:"avg_response_ms"`
	SuccessRate   float64                  `json:"success_rate"`
	TotalCalls    int64                    `json:"total_calls"`
	DailyStats    []DashboardLLMDailyPoint `json:"daily_stats"`
}

// DashboardLLMDailyPoint 按日 LLM 调用统计。
type DashboardLLMDailyPoint struct {
	Date  string `json:"date"`
	AvgMs int64  `json:"avg_ms"`
	Calls int64  `json:"calls"`
}

// DashboardTenantUsage 租户资源用量（租户管理员）。
type DashboardTenantUsage struct {
	TokenUsed     int64 `json:"token_used"`
	TokenQuota    int64 `json:"token_quota"`
	StorageUsedMB int64 `json:"storage_used_mb"`
	StorageQuotaMB int64 `json:"storage_quota_mb"`
	ActiveUsers   int64 `json:"active_users"`
	TotalUsers    int64 `json:"total_users"`
}

// DashboardUserActivityRow 用户审核活跃度排行。
type DashboardUserActivityRow struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Department  string `json:"department"`
	AuditCount  int64  `json:"audit_count"`
	LastActive  string `json:"last_active"` // RFC3339
}

// PlatformTenantRankRow 全平台：按已完成审核数排名的租户。
type PlatformTenantRankRow struct {
	TenantID   string `json:"tenant_id"`
	TenantName string `json:"tenant_name"`
	TenantCode string `json:"tenant_code"`
	AuditCount int64  `json:"audit_count"`
}

// PlatformTokenSummary 全平台 Token 汇总（各租户表之和）。
type PlatformTokenSummary struct {
	TotalUsed  int64 `json:"total_used"`
	TotalQuota int64 `json:"total_quota"`
}

// PlatformDashboardOverviewResponse 系统管理员平台仪表盘（GET /api/admin/dashboard-overview）。
type PlatformDashboardOverviewResponse struct {
	TenantTotal  int64 `json:"tenant_total"`
	TenantActive int64 `json:"tenant_active"`

	PendingOACount int `json:"pending_oa_count"` // 平台视图固定为 0

	AuditSummary DashboardAuditSummary `json:"audit_summary"`

	WeeklyTrend []DashboardDayCount `json:"weekly_trend"`

	RecentActivity []DashboardActivityItem `json:"recent_activity"`

	ArchiveRecent []DashboardArchiveRow `json:"archive_recent"`

	TenantRanking []PlatformTenantRankRow `json:"tenant_ranking"`

	AIPerformance *DashboardAIPerformance `json:"ai_performance"`

	TokenSummary *PlatformTokenSummary `json:"token_summary"`
}
