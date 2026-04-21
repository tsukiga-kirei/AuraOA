/**
 * useDashboardOverviewApi — 仪表盘概览数据 API 调用封装
 * 对接后端路由：
 *   GET /api/tenant/settings/dashboard-overview  租户仪表盘概览数据
 *   GET /api/admin/dashboard-overview            平台级仪表盘概览数据（系统管理员）
 *   GET /api/admin/system-monitor                系统运行监控数据（系统管理员）
 */

import type { DashboardOverview, PlatformDashboardOverview, SystemMonitorData } from '~/types/dashboard-overview'

export const useDashboardOverviewApi = () => {
  const { authFetch } = useAuth()

  /**
   * 获取当前租户的仪表盘概览数据（审核统计、归档统计、定时任务状态等）。
   * @returns 租户维度的仪表盘聚合数据
   */
  async function fetchDashboardOverview(): Promise<DashboardOverview> {
    return await authFetch<DashboardOverview>('/api/tenant/settings/dashboard-overview')
  }

  /**
   * 获取平台级仪表盘概览数据（所有租户汇总，仅系统管理员可访问）。
   * @returns 平台维度的仪表盘聚合数据
   */
  async function fetchPlatformDashboardOverview(): Promise<PlatformDashboardOverview> {
    return await authFetch<PlatformDashboardOverview>('/api/admin/dashboard-overview')
  }

  /**
   * 获取系统运行监控数据（仅系统管理员可访问）。
   * 对接后端路由：GET /api/admin/system-monitor
   * @returns 系统 CPU、内存、磁盘使用率及关键服务状态
   */
  async function fetchSystemMonitorData(): Promise<SystemMonitorData> {
    return await authFetch<SystemMonitorData>('/api/admin/system-monitor')
  }

  return { fetchDashboardOverview, fetchPlatformDashboardOverview, fetchSystemMonitorData }
}
