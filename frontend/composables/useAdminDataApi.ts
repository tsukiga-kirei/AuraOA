/**
 * useAdminDataApi — 数据管理页面 API 封装
 * 对接后端路由组：
 *   GET /api/audit/snapshots             审核快照分页（数据管理页主表）
 *   GET /api/audit/snapshots/stats       审核快照统计
 *   GET /api/audit/snapshots/:id/chain   审核链详情
 *   GET /api/audit/logs/export           审核日志导出
 *   GET /api/archive/snapshots           归档快照分页
 *   GET /api/archive/snapshots/stats     归档快照统计
 *   GET /api/archive/snapshots/:id/chain 归档复盘链详情
 *   GET /api/archive/logs/export         归档复盘日志导出
 *   GET /api/tenant/cron/logs            定时任务日志分页
 *   GET /api/tenant/cron/logs/stats      定时任务日志统计
 *   GET /api/tenant/cron/logs/export     定时任务日志导出
 */

import type {
  AuditLogFilter,
  AuditLogStats,
  AuditLogItem,
  AuditSnapshotFilter,
  AuditSnapshotStats,
  AuditSnapshotItem,
  ArchiveLogFilter,
  ArchiveLogStats,
  ArchiveLogItem,
  ArchiveSnapshotFilter,
  ArchiveSnapshotStats,
  ArchiveSnapshotItem,
  CronLogFilter,
  CronLogStats,
  CronLogItem,
  PagedResult,
} from '~/types/admin-data'

export function useAdminDataApi() {
  const { authFetch, token } = useAuth()

  // ── 审核快照（数据管理页主数据源） ──────────────────────────────────────────

  async function listAuditSnapshots(filter: AuditSnapshotFilter = {}): Promise<PagedResult<AuditSnapshotItem>> {
    const params = buildParams(filter)
    const query = new URLSearchParams(params).toString()
    return await authFetch<PagedResult<AuditSnapshotItem>>(`/api/audit/snapshots${query ? `?${query}` : ''}`)
  }

  async function getAuditSnapshotStats(): Promise<AuditSnapshotStats> {
    return await authFetch<AuditSnapshotStats>('/api/audit/snapshots/stats')
  }

  async function getAuditSnapshotChain(processId: string): Promise<{ chain: AuditLogItem[] }> {
    return await authFetch<{ chain: AuditLogItem[] }>(`/api/audit/snapshots/${processId}/chain`)
  }

  // ── 审核日志（保留用于导出） ──────────────────────────────────────────────

  async function listAuditLogs(filter: AuditLogFilter = {}): Promise<PagedResult<AuditLogItem>> {
    const params = buildParams(filter)
    const query = new URLSearchParams(params).toString()
    return await authFetch<PagedResult<AuditLogItem>>(`/api/audit/logs${query ? `?${query}` : ''}`)
  }

  async function getAuditLogStats(): Promise<AuditLogStats> {
    return await authFetch<AuditLogStats>('/api/audit/logs/stats')
  }

  async function exportAuditLogs(filter: AuditLogFilter = {}) {
    const params = buildParams(filter)
    const url = buildExportUrl('/api/audit/logs/export', params)
    await triggerDownload(url, 'audit_logs.csv')
  }

  // ── 归档快照（数据管理页主数据源） ──────────────────────────────────────────

  async function listArchiveSnapshots(filter: ArchiveSnapshotFilter = {}): Promise<PagedResult<ArchiveSnapshotItem>> {
    const params = buildParams(filter)
    const query = new URLSearchParams(params).toString()
    return await authFetch<PagedResult<ArchiveSnapshotItem>>(`/api/archive/snapshots${query ? `?${query}` : ''}`)
  }

  async function getArchiveSnapshotStats(): Promise<ArchiveSnapshotStats> {
    return await authFetch<ArchiveSnapshotStats>('/api/archive/snapshots/stats')
  }

  async function getArchiveSnapshotChain(processId: string): Promise<{ chain: ArchiveLogItem[] }> {
    return await authFetch<{ chain: ArchiveLogItem[] }>(`/api/archive/snapshots/${processId}/chain`)
  }

  // ── 归档复盘日志（保留用于导出） ──────────────────────────────────────────

  async function listArchiveLogs(filter: ArchiveLogFilter = {}): Promise<PagedResult<ArchiveLogItem>> {
    const params = buildParams(filter)
    const query = new URLSearchParams(params).toString()
    return await authFetch<PagedResult<ArchiveLogItem>>(`/api/archive/logs${query ? `?${query}` : ''}`)
  }

  async function getArchiveLogStats(): Promise<ArchiveLogStats> {
    return await authFetch<ArchiveLogStats>('/api/archive/logs/stats')
  }

  async function exportArchiveLogs(filter: ArchiveLogFilter = {}) {
    const params = buildParams(filter)
    const url = buildExportUrl('/api/archive/logs/export', params)
    await triggerDownload(url, 'archive_logs.csv')
  }

  // ── 定时任务日志 ──────────────────────────────────────────────────────────────

  async function listCronLogs(filter: CronLogFilter = {}): Promise<PagedResult<CronLogItem>> {
    const params = buildParams(filter)
    const query = new URLSearchParams(params).toString()
    return await authFetch<PagedResult<CronLogItem>>(`/api/tenant/cron/logs${query ? `?${query}` : ''}`)
  }

  async function getCronLogStats(): Promise<CronLogStats> {
    return await authFetch<CronLogStats>('/api/tenant/cron/logs/stats')
  }

  async function exportCronLogs(filter: CronLogFilter = {}) {
    const params = buildParams(filter)
    const url = buildExportUrl('/api/tenant/cron/logs/export', params)
    await triggerDownload(url, 'cron_logs.csv')
  }

  // ── 工具函数 ──────────────────────────────────────────────────────────────────

  function buildParams(filter: Record<string, any>): Record<string, string> {
    const params: Record<string, string> = {}
    for (const [key, value] of Object.entries(filter)) {
      if (value !== undefined && value !== null && value !== '') {
        params[key] = String(value)
      }
    }
    return params
  }

  function buildExportUrl(path: string, params: Record<string, string>): string {
    const runtimeConfig = useRuntimeConfig()
    const baseURL = String(runtimeConfig.public.apiBase || '')
    const query = new URLSearchParams(params).toString()
    return `${baseURL}${path}${query ? `?${query}` : ''}`
  }

  async function triggerDownload(url: string, fallbackName: string) {
    const accessToken = token.value || (process.client ? localStorage.getItem('token') || '' : '')

    const res = await fetch(url, {
      headers: accessToken
          ? { Authorization: `Bearer ${accessToken}` }
          : {},
    })

    if (!res.ok) {
      throw new Error('导出失败')
    }

    const blob = await res.blob()
    const blobUrl = URL.createObjectURL(blob)

    try {
      const a = document.createElement('a')
      a.href = blobUrl

      const contentDisposition = res.headers.get('Content-Disposition') || ''
      const utf8Match = contentDisposition.match(/filename\*=UTF-8''([^;]+)/i)
      const normalMatch = contentDisposition.match(/filename=\"?([^\";]+)\"?/i)

      const filename = utf8Match?.[1]
          ? decodeURIComponent(utf8Match[1])
          : normalMatch?.[1] || fallbackName

      a.download = filename
      document.body.appendChild(a)
      a.click()
      document.body.removeChild(a)
    } finally {
      URL.revokeObjectURL(blobUrl)
    }
  }

  return {
    // 审核快照
    listAuditSnapshots,
    getAuditSnapshotStats,
    getAuditSnapshotChain,
    // 审核日志（保留）
    listAuditLogs,
    getAuditLogStats,
    exportAuditLogs,
    // 归档快照
    listArchiveSnapshots,
    getArchiveSnapshotStats,
    getArchiveSnapshotChain,
    // 归档日志（保留）
    listArchiveLogs,
    getArchiveLogStats,
    exportArchiveLogs,
    // 定时任务
    listCronLogs,
    getCronLogStats,
    exportCronLogs,
  }
}