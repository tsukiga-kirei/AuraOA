import type { PermissionGroup } from '~/types/auth'

export type OverviewWidgetId =
  | 'audit_summary'
  | 'pending_tasks'
  | 'weekly_trend'
  | 'cron_tasks'
  | 'archive_review'
  | 'dept_distribution'
  | 'recent_activity'
  | 'ai_performance'
  | 'tenant_usage'
  | 'user_activity'
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

/** 当前已接入真实数据的仪表盘组件（系统监控类已移至业务 TODO，自此处移除） */
export const OVERVIEW_WIDGETS: OverviewWidgetDef[] = [
  { id: 'audit_summary', titleKey: 'overview.widgetTitle.audit_summary', descriptionKey: 'overview.widgetDesc.audit_summary', requiredPermissions: ['business', 'tenant_admin', 'system_admin'], defaultEnabled: true, size: 'lg' },
  { id: 'pending_tasks', titleKey: 'overview.widgetTitle.pending_tasks', descriptionKey: 'overview.widgetDesc.pending_tasks', requiredPermissions: ['business', 'tenant_admin'], defaultEnabled: true, size: 'sm' },
  { id: 'weekly_trend', titleKey: 'overview.widgetTitle.weekly_trend', descriptionKey: 'overview.widgetDesc.weekly_trend', requiredPermissions: ['business', 'tenant_admin', 'system_admin'], defaultEnabled: true, size: 'md' },
  { id: 'cron_tasks', titleKey: 'overview.widgetTitle.cron_tasks', descriptionKey: 'overview.widgetDesc.cron_tasks', requiredPermissions: ['business', 'tenant_admin'], defaultEnabled: true, size: 'md' },
  { id: 'archive_review', titleKey: 'overview.widgetTitle.archive_review', descriptionKey: 'overview.widgetDesc.archive_review', requiredPermissions: ['business', 'tenant_admin', 'system_admin'], defaultEnabled: true, size: 'md' },
  { id: 'dept_distribution', titleKey: 'overview.widgetTitle.dept_distribution', descriptionKey: 'overview.widgetDesc.dept_distribution', requiredPermissions: ['tenant_admin'], defaultEnabled: true, size: 'md' },
  { id: 'recent_activity', titleKey: 'overview.widgetTitle.recent_activity', descriptionKey: 'overview.widgetDesc.recent_activity', requiredPermissions: ['business', 'tenant_admin', 'system_admin'], defaultEnabled: true, size: 'md' },
  { id: 'ai_performance', titleKey: 'overview.widgetTitle.ai_performance', descriptionKey: 'overview.widgetDesc.ai_performance', requiredPermissions: ['tenant_admin', 'system_admin'], defaultEnabled: true, size: 'md' },
  { id: 'tenant_usage', titleKey: 'overview.widgetTitle.tenant_usage', descriptionKey: 'overview.widgetDesc.tenant_usage', requiredPermissions: ['tenant_admin', 'system_admin'], defaultEnabled: true, size: 'md' },
  { id: 'user_activity', titleKey: 'overview.widgetTitle.user_activity', descriptionKey: 'overview.widgetDesc.user_activity', requiredPermissions: ['tenant_admin'], defaultEnabled: true, size: 'md' },
  { id: 'platform_tenant_stats', titleKey: 'overview.widgetTitle.platform_tenant_stats', descriptionKey: 'overview.widgetDesc.platform_tenant_stats', requiredPermissions: ['system_admin'], defaultEnabled: true, size: 'md' },
  { id: 'platform_tenant_ranking', titleKey: 'overview.widgetTitle.platform_tenant_ranking', descriptionKey: 'overview.widgetDesc.platform_tenant_ranking', requiredPermissions: ['system_admin'], defaultEnabled: true, size: 'md' },
]

export const OVERVIEW_WIDGET_ID_SET = new Set<OverviewWidgetId>(OVERVIEW_WIDGETS.map(w => w.id))
