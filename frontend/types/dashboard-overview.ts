/** GET /api/tenant/settings/dashboard-overview 响应（字段 snake_case 与后端一致） */

// ── 租户级仪表盘（business / tenant_admin）──

/** 本周概览 */
export interface WeeklyOverviewData {
  total: number
  audit_count: number
  archive_count: number
  cron_count: number
}

/** 待办任务（区分类型） */
export interface PendingTasksData {
  audit_pending: number
  archive_pending: number
  total: number
}

/** 审核趋势（堆叠柱状图） */
export interface WeeklyTrendDayData {
  date: string
  audit_count: number
  cron_count: number
  archive_count: number
}

/** 最近动态（增强版，带详细标注） */
export interface ActivityItemEnriched {
  id: string
  kind: 'audit' | 'archive' | 'cron'
  title: string
  user_name: string
  created_at: string
  // 审核工作台标注
  recommendation?: string
  score?: number
  // 归档复盘标注
  compliance?: string
  compliance_score?: number
  // 定时任务标注
  cron_status?: string
  task_label?: string
}

/** 定时任务预览（仅 business） */
export interface CronTaskPreview {
  id: string
  task_label: string
  task_type: string
  description: string
  cron_expression: string
  is_active: boolean
}

/** 部门分布（三功能分组） */
export interface DeptDistributionData {
  department: string
  audit_count: number
  cron_count: number
  archive_count: number
  total: number
}

/** 用户审核活跃度排行 */
export interface DashboardUserActivityRow {
  username: string
  display_name: string
  department: string
  audit_count: number
  last_active: string
}

/** 租户级响应 */
export interface DashboardOverview {
  weekly_overview: WeeklyOverviewData
  pending_tasks?: PendingTasksData
  weekly_trend: WeeklyTrendDayData[]
  recent_activity: ActivityItemEnriched[]
  cron_tasks?: CronTaskPreview[]
  dept_distribution?: DeptDistributionData[]
  user_activity?: DashboardUserActivityRow[]
}

// ── 系统管理员平台仪表盘（system_admin）──

/** 租户规模明细行 */
export interface TenantStatsRow {
  tenant_id: string
  tenant_name: string
  tenant_code: string
  user_count: number
  is_active: boolean
}

/** 租户规模 */
export interface PlatformTenantStatsData {
  tenant_total: number
  tenant_active: number
  active_criteria: string
  tenants: TenantStatsRow[]
}

/** 单种调用类型统计 */
export interface AICallTypeStats {
  calls: number
  success_rate: number
  avg_ms: number
}

/** 单个 AI 模型性能数据 */
export interface AIModelPerformanceRow {
  model_config_id: string
  model_name: string
  display_name: string
  provider: string
  reasoning_stats: AICallTypeStats
  structured_stats: AICallTypeStats
  overall_success_rate: number
  total_calls: number
}

/** AI 模型表现 */
export interface PlatformAIPerformanceData {
  models: AIModelPerformanceRow[]
}

/** 按租户分列的资源用量 */
export interface TenantUsageRow {
  tenant_id: string
  tenant_name: string
  tenant_code: string
  token_used: number
  token_quota: number
}

/** 租户审核排名（含失败记录） */
export interface PlatformTenantRankRowEnriched {
  tenant_id: string
  tenant_name: string
  tenant_code: string
  audit_count: number
  archive_count: number
  cron_count: number
  audit_failed: number
  archive_failed: number
}

/** GET /api/admin/dashboard-overview */
export interface PlatformDashboardOverview {
  tenant_stats: PlatformTenantStatsData
  ai_performance: PlatformAIPerformanceData
  tenant_usage_list: TenantUsageRow[]
  tenant_ranking: PlatformTenantRankRowEnriched[]
}
