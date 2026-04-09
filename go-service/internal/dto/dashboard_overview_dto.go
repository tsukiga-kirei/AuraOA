package dto

// ─── 租户级仪表盘（business / tenant_admin） ───

// DashboardOverviewResponse 仪表盘聚合数据（GET /api/tenant/settings/dashboard-overview）。
type DashboardOverviewResponse struct {
	// 本周概览（替代原 AuditSummary）
	WeeklyOverview *WeeklyOverviewData `json:"weekly_overview"`

	// 待办任务（仅 business）
	PendingTasks *PendingTasksData `json:"pending_tasks,omitempty"`

	// 审核趋势 — 按日按功能分组（堆叠柱状图数据）
	WeeklyTrend []WeeklyTrendDayData `json:"weekly_trend"`

	// 最近动态（带详细标注）
	RecentActivity []ActivityItemEnriched `json:"recent_activity"`

	// 定时任务列表（仅 business）
	CronTasks []CronTaskPreview `json:"cron_tasks,omitempty"`

	// 部门分布（仅 tenant_admin）
	DeptDistribution []DeptDistributionData `json:"dept_distribution,omitempty"`

	// 用户活跃排名（仅 tenant_admin）
	UserActivity []DashboardUserActivityRow `json:"user_activity,omitempty"`
}

// WeeklyOverviewData 本周概览（周一 00:00 至当前）。
type WeeklyOverviewData struct {
	Total        int64 `json:"total"`         // 三项之和
	AuditCount   int64 `json:"audit_count"`   // 审核工作台快照本周条数
	ArchiveCount int64 `json:"archive_count"` // 归档复盘快照本周条数
	CronCount    int64 `json:"cron_count"`    // 定时任务本周执行次数
}

// PendingTasksData 待办任务（区分类型）。
type PendingTasksData struct {
	AuditPending   int64 `json:"audit_pending"`   // 审核工作台待办
	ArchivePending int64 `json:"archive_pending"`  // 归档复盘待办
	Total          int64 `json:"total"`
}

// WeeklyTrendDayData 每天按功能分组的数据（堆叠柱状图）。
type WeeklyTrendDayData struct {
	Date         string `json:"date"`          // MM-DD
	AuditCount   int64  `json:"audit_count"`   // 审核工作台
	CronCount    int64  `json:"cron_count"`    // 定时任务
	ArchiveCount int64  `json:"archive_count"` // 归档复盘
}

// ActivityItemEnriched 带详细标注的动态条目。
type ActivityItemEnriched struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`       // audit | archive | cron
	Title     string `json:"title"`
	UserName  string `json:"user_name"`
	CreatedAt string `json:"created_at"` // RFC3339

	// 审核工作台标注
	Recommendation string `json:"recommendation,omitempty"` // approve/return/review
	Score          int    `json:"score,omitempty"`

	// 归档复盘标注
	Compliance      string `json:"compliance,omitempty"`
	ComplianceScore int    `json:"compliance_score,omitempty"`

	// 定时任务标注
	CronStatus string `json:"cron_status,omitempty"` // success/failed/running
	TaskLabel  string `json:"task_label,omitempty"`
}

// CronTaskPreview 定时任务预览（仅 business）。
type CronTaskPreview struct {
	ID             string `json:"id"`
	TaskLabel      string `json:"task_label"`
	TaskType       string `json:"task_type"`
	Description    string `json:"description"`
	CronExpression string `json:"cron_expression"`
	IsActive       bool   `json:"is_active"`
}

// DeptDistributionData 部门分布（区分三个功能）。
type DeptDistributionData struct {
	Department   string `json:"department"`
	AuditCount   int64  `json:"audit_count"`
	CronCount    int64  `json:"cron_count"`
	ArchiveCount int64  `json:"archive_count"`
	Total        int64  `json:"total"`
}

// DashboardUserActivityRow 用户审核活跃度排行。
type DashboardUserActivityRow struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Department  string `json:"department"`
	AuditCount  int64  `json:"audit_count"`
	LastActive  string `json:"last_active"` // RFC3339
}

// ─── 系统管理员平台仪表盘（system_admin） ───

// PlatformDashboardOverviewResponse 系统管理员平台仪表盘（GET /api/admin/dashboard-overview）。
type PlatformDashboardOverviewResponse struct {
	// 租户规模（含人员数量）
	TenantStats *PlatformTenantStatsData `json:"tenant_stats"`

	// AI 模型表现（按模型+调用类型分组）
	AIPerformance *PlatformAIPerformanceData `json:"ai_performance"`

	// 租户资源用量（按租户分列）
	TenantUsageList []TenantUsageRow `json:"tenant_usage_list"`

	// 租户审核排名（含失败记录）
	TenantRanking []PlatformTenantRankRowEnriched `json:"tenant_ranking"`
}

// PlatformTenantStatsData 租户规模（含人员数量）。
type PlatformTenantStatsData struct {
	TenantTotal    int64            `json:"tenant_total"`
	TenantActive   int64            `json:"tenant_active"`
	ActiveCriteria string           `json:"active_criteria"` // 活跃判断标准说明
	Tenants        []TenantStatsRow `json:"tenants"`
}

// TenantStatsRow 租户规模明细行。
type TenantStatsRow struct {
	TenantID   string `json:"tenant_id"`
	TenantName string `json:"tenant_name"`
	TenantCode string `json:"tenant_code"`
	UserCount  int64  `json:"user_count"` // 注册人员数量
	IsActive   bool   `json:"is_active"`
}

// PlatformAIPerformanceData AI 模型表现（按模型+调用类型分组）。
type PlatformAIPerformanceData struct {
	Models []AIModelPerformanceRow `json:"models"`
}

// AIModelPerformanceRow 单个 AI 模型的性能数据。
type AIModelPerformanceRow struct {
	ModelConfigID string `json:"model_config_id"`
	ModelName     string `json:"model_name"`
	DisplayName   string `json:"display_name"`
	Provider      string `json:"provider"`

	// 按调用类型分组
	ReasoningStats  AICallTypeStats `json:"reasoning_stats"`
	StructuredStats AICallTypeStats `json:"structured_stats"`

	// 总体
	OverallSuccessRate float64 `json:"overall_success_rate"`
	TotalCalls         int64   `json:"total_calls"`
}

// AICallTypeStats 单种调用类型的统计。
type AICallTypeStats struct {
	Calls       int64   `json:"calls"`
	SuccessRate float64 `json:"success_rate"`
	AvgMs       int64   `json:"avg_ms"`
}

// TenantUsageRow 按租户分列的资源用量。
type TenantUsageRow struct {
	TenantID   string `json:"tenant_id"`
	TenantName string `json:"tenant_name"`
	TenantCode string `json:"tenant_code"`
	TokenUsed  int64  `json:"token_used"`
	TokenQuota int64  `json:"token_quota"`
}

// PlatformTenantRankRowEnriched 租户审核排名（含失败记录）。
type PlatformTenantRankRowEnriched struct {
	TenantID      string `json:"tenant_id"`
	TenantName    string `json:"tenant_name"`
	TenantCode    string `json:"tenant_code"`
	AuditCount    int64  `json:"audit_count"`    // 审核快照数
	ArchiveCount  int64  `json:"archive_count"`  // 归档复盘快照数
	CronCount     int64  `json:"cron_count"`     // 定时任务执行数
	AuditFailed   int64  `json:"audit_failed"`   // 审核失败数
	ArchiveFailed int64  `json:"archive_failed"` // 归档复盘失败数
}
