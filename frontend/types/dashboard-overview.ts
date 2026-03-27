/** GET /api/tenant/settings/dashboard-overview 响应（字段 snake_case 与后端一致） */

export interface DashboardAuditSummary {
  total: number
  approved: number
  returned: number
  archived: number
  review: number
  pending_ai: number
}

export interface DashboardDayCount {
  date: string
  count: number
}

export interface DashboardActivityItem {
  id: string
  kind: 'audit_completed' | 'audit_failed' | 'cron_log' | 'archive_reviewed'
  title: string
  user_name: string
  created_at: string
}

export interface DashboardArchiveRecentRow {
  id: string
  title: string
  compliance: string
  user_name: string
  created_at: string
}

export interface DashboardDeptCount {
  department: string
  count: number
}

export interface DashboardLLMDailyPoint {
  date: string
  avg_ms: number
  calls: number
}

export interface DashboardAIPerformance {
  avg_response_ms: number
  success_rate: number
  total_calls: number
  daily_stats: DashboardLLMDailyPoint[]
}

export interface DashboardTenantUsage {
  token_used: number
  token_quota: number
  storage_used_mb: number
  storage_quota_mb: number
  active_users: number
  total_users: number
}

export interface DashboardUserActivityRow {
  username: string
  display_name: string
  department: string
  audit_count: number
  last_active: string
}

export interface DashboardOverview {
  pending_oa_count: number
  audit_summary: DashboardAuditSummary
  weekly_trend: DashboardDayCount[]
  recent_activity: DashboardActivityItem[]
  archive_recent: DashboardArchiveRecentRow[]
  dept_distribution?: DashboardDeptCount[]
  ai_performance?: DashboardAIPerformance
  tenant_usage?: DashboardTenantUsage
  user_activity?: DashboardUserActivityRow[]
  /** 以下字段仅系统管理员平台仪表盘使用 */
  tenant_total?: number
  tenant_active?: number
  tenant_ranking?: PlatformTenantRankRow[]
}

export interface PlatformTenantRankRow {
  tenant_id: string
  tenant_name: string
  tenant_code: string
  audit_count: number
}

export interface PlatformTokenSummary {
  total_used: number
  total_quota: number
}

/** GET /api/admin/dashboard-overview */
export interface PlatformDashboardOverview {
  tenant_total: number
  tenant_active: number
  pending_oa_count: number
  audit_summary: DashboardAuditSummary
  weekly_trend: DashboardDayCount[]
  recent_activity: DashboardActivityItem[]
  archive_recent: DashboardArchiveRecentRow[]
  tenant_ranking: PlatformTenantRankRow[]
  ai_performance?: DashboardAIPerformance
  token_summary?: PlatformTokenSummary
}
