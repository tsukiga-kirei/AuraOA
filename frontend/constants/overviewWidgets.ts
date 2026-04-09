import type { PermissionGroup } from '~/types/auth'

export type OverviewWidgetId =
  | 'weekly_overview'
  | 'pending_tasks'
  | 'weekly_trend'
  | 'cron_tasks'
  | 'recent_activity'
  | 'dept_distribution'
  | 'user_activity'
  | 'ai_performance'
  | 'tenant_usage'
  | 'platform_tenant_stats'
  | 'platform_tenant_ranking'

export interface OverviewWidgetDef {
  id: OverviewWidgetId
  /** i18n 键，用于标题 */
  titleKey: string
  descriptionKey: string
  requiredPermissions: PermissionGroup[]
  defaultEnabled: boolean
  size: 'sm' | 'md' | 'lg'
}

/** 组件与页面权限的映射关系（business 角色专用） */
export const WIDGET_PAGE_PERMISSION_MAP: Partial<Record<OverviewWidgetId, string>> = {
  weekly_overview: '',
  pending_tasks: '',
  weekly_trend: '',
  cron_tasks: '/cron',
  recent_activity: '',
}

/** 仪表盘组件注册表 */
export const OVERVIEW_WIDGETS: OverviewWidgetDef[] = [
  // business + tenant_admin
  { id: 'weekly_overview', titleKey: 'overview.widgetTitle.weekly_overview', descriptionKey: 'overview.widgetDesc.weekly_overview', requiredPermissions: ['business', 'tenant_admin'], defaultEnabled: true, size: 'lg' },
  { id: 'pending_tasks', titleKey: 'overview.widgetTitle.pending_tasks', descriptionKey: 'overview.widgetDesc.pending_tasks', requiredPermissions: ['business'], defaultEnabled: true, size: 'sm' },
  { id: 'weekly_trend', titleKey: 'overview.widgetTitle.weekly_trend', descriptionKey: 'overview.widgetDesc.weekly_trend', requiredPermissions: ['business', 'tenant_admin'], defaultEnabled: true, size: 'md' },
  { id: 'cron_tasks', titleKey: 'overview.widgetTitle.cron_tasks', descriptionKey: 'overview.widgetDesc.cron_tasks', requiredPermissions: ['business'], defaultEnabled: true, size: 'md' },
  { id: 'recent_activity', titleKey: 'overview.widgetTitle.recent_activity', descriptionKey: 'overview.widgetDesc.recent_activity', requiredPermissions: ['business', 'tenant_admin'], defaultEnabled: true, size: 'md' },
  // tenant_admin only
  { id: 'dept_distribution', titleKey: 'overview.widgetTitle.dept_distribution', descriptionKey: 'overview.widgetDesc.dept_distribution', requiredPermissions: ['tenant_admin'], defaultEnabled: true, size: 'md' },
  { id: 'user_activity', titleKey: 'overview.widgetTitle.user_activity', descriptionKey: 'overview.widgetDesc.user_activity', requiredPermissions: ['tenant_admin'], defaultEnabled: true, size: 'md' },
  // system_admin only
  { id: 'platform_tenant_stats', titleKey: 'overview.widgetTitle.platform_tenant_stats', descriptionKey: 'overview.widgetDesc.platform_tenant_stats', requiredPermissions: ['system_admin'], defaultEnabled: true, size: 'md' },
  { id: 'ai_performance', titleKey: 'overview.widgetTitle.ai_performance', descriptionKey: 'overview.widgetDesc.ai_performance', requiredPermissions: ['system_admin'], defaultEnabled: true, size: 'lg' },
  { id: 'tenant_usage', titleKey: 'overview.widgetTitle.tenant_usage', descriptionKey: 'overview.widgetDesc.tenant_usage', requiredPermissions: ['system_admin'], defaultEnabled: true, size: 'md' },
  { id: 'platform_tenant_ranking', titleKey: 'overview.widgetTitle.platform_tenant_ranking', descriptionKey: 'overview.widgetDesc.platform_tenant_ranking', requiredPermissions: ['system_admin'], defaultEnabled: true, size: 'lg' },
]

export const OVERVIEW_WIDGET_ID_SET = new Set<OverviewWidgetId>(OVERVIEW_WIDGETS.map(w => w.id))
