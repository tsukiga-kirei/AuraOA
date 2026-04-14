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

  /** 分页查询审核快照列表，支持多维度筛选 */
  async function listAuditSnapshots(filter: AuditSnapshotFilter = {}): Promise<PagedResult<AuditSnapshotItem>> {
    const params = buildParams(filter)
    const query = new URLSearchParams(params).toString()
    return await authFetch<PagedResult<AuditSnapshotItem>>(`/api/audit/snapshots${query ? `?${query}` : ''}`)
  }

  /** 获取审核快照的汇总统计数据（总数、各状态分布等） */
  async function getAuditSnapshotStats(): Promise<AuditSnapshotStats> {
    return await authFetch<AuditSnapshotStats>('/api/audit/snapshots/stats')
  }

  /**
   * 获取指定流程的完整审核链（历次审核记录按时间排列）。
   * @param processId 流程 ID
   */
  async function getAuditSnapshotChain(processId: string): Promise<{ chain: AuditLogItem[] }> {
    return await authFetch<{ chain: AuditLogItem[] }>(`/api/audit/snapshots/${processId}/chain`)
  }

  // ── 审核日志（保留用于导出） ──────────────────────────────────────────────

  /** 分页查询审核日志列表（主要用于导出场景） */
  async function listAuditLogs(filter: AuditLogFilter = {}): Promise<PagedResult<AuditLogItem>> {
    const params = buildParams(filter)
    const query = new URLSearchParams(params).toString()
    return await authFetch<PagedResult<AuditLogItem>>(`/api/audit/logs${query ? `?${query}` : ''}`)
  }

  /** 获取审核日志统计数据 */
  async function getAuditLogStats(): Promise<AuditLogStats> {
    return await authFetch<AuditLogStats>('/api/audit/logs/stats')
  }

  /** 导出审核日志为 CSV 文件，触发浏览器下载 */
  async function exportAuditLogs(filter: AuditLogFilter = {}) {
    const params = buildParams(filter)
    const url = buildExportUrl('/api/audit/logs/export', params)
    await triggerDownload(url, 'audit_logs.csv')
  }

  // ── 归档快照（数据管理页主数据源） ──────────────────────────────────────────

  /** 分页查询归档快照列表，支持多维度筛选 */
  async function listArchiveSnapshots(filter: ArchiveSnapshotFilter = {}): Promise<PagedResult<ArchiveSnapshotItem>> {
    const params = buildParams(filter)
    const query = new URLSearchParams(params).toString()
    return await authFetch<PagedResult<ArchiveSnapshotItem>>(`/api/archive/snapshots${query ? `?${query}` : ''}`)
  }

  /** 获取归档快照的汇总统计数据 */
  async function getArchiveSnapshotStats(): Promise<ArchiveSnapshotStats> {
    return await authFetch<ArchiveSnapshotStats>('/api/archive/snapshots/stats')
  }

  /**
   * 获取指定流程的完整归档复盘链。
   * @param processId 流程 ID
   */
  async function getArchiveSnapshotChain(processId: string): Promise<{ chain: ArchiveLogItem[] }> {
    return await authFetch<{ chain: ArchiveLogItem[] }>(`/api/archive/snapshots/${processId}/chain`)
  }

  // ── 归档复盘日志（保留用于导出） ──────────────────────────────────────────

  /** 分页查询归档复盘日志列表（主要用于导出场景） */
  async function listArchiveLogs(filter: ArchiveLogFilter = {}): Promise<PagedResult<ArchiveLogItem>> {
    const params = buildParams(filter)
    const query = new URLSearchParams(params).toString()
    return await authFetch<PagedResult<ArchiveLogItem>>(`/api/archive/logs${query ? `?${query}` : ''}`)
  }

  /** 获取归档复盘日志统计数据 */
  async function getArchiveLogStats(): Promise<ArchiveLogStats> {
    return await authFetch<ArchiveLogStats>('/api/archive/logs/stats')
  }

  /** 导出归档复盘日志为 CSV 文件，触发浏览器下载 */
  async function exportArchiveLogs(filter: ArchiveLogFilter = {}) {
    const params = buildParams(filter)
    const url = buildExportUrl('/api/archive/logs/export', params)
    await triggerDownload(url, 'archive_logs.csv')
  }

  // ── 定时任务日志 ──────────────────────────────────────────────────────────────

  /** 分页查询定时任务执行日志 */
  async function listCronLogs(filter: CronLogFilter = {}): Promise<PagedResult<CronLogItem>> {
    const params = buildParams(filter)
    const query = new URLSearchParams(params).toString()
    return await authFetch<PagedResult<CronLogItem>>(`/api/tenant/cron/logs${query ? `?${query}` : ''}`)
  }

  /** 获取定时任务日志统计数据 */
  async function getCronLogStats(): Promise<CronLogStats> {
    return await authFetch<CronLogStats>('/api/tenant/cron/logs/stats')
  }

  /** 导出定时任务日志为 CSV 文件，触发浏览器下载 */
  async function exportCronLogs(filter: CronLogFilter = {}) {
    const params = buildParams(filter)
    const url = buildExportUrl('/api/tenant/cron/logs/export', params)
    await triggerDownload(url, 'cron_logs.csv')
  }

  // ── 工具函数 ──────────────────────────────────────────────────────────────────

  /**
   * 将过滤器对象转换为 URL 查询参数，过滤掉空值。
   * @param filter 过滤条件对象
   * @returns 字符串键值对，可直接传入 URLSearchParams
   */
  function buildParams(filter: Record<string, any>): Record<string, string> {
    const params: Record<string, string> = {}
    for (const [key, value] of Object.entries(filter)) {
      if (value !== undefined && value !== null && value !== '') {
        params[key] = String(value)
      }
    }
    return params
  }

  /**
   * 构建带认证 base URL 的导出请求完整地址。
   * @param path 接口路径
   * @param params 查询参数
   */
  function buildExportUrl(path: string, params: Record<string, string>): string {
    const runtimeConfig = useRuntimeConfig()
    const baseURL = String(runtimeConfig.public.apiBase || '')
    const query = new URLSearchParams(params).toString()
    return `${baseURL}${path}${query ? `?${query}` : ''}`
  }

  /**
   * 使用原生 fetch 下载文件并触发浏览器保存对话框。
   * 从响应头 Content-Disposition 中解析文件名，支持 UTF-8 编码文件名。
   * @param url 下载地址（含完整 base URL 和查询参数）
   * @param fallbackName 无法从响应头解析文件名时的默认文件名
   */
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

      // 优先解析 RFC 5987 编码的 UTF-8 文件名，降级使用普通文件名
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
      // 释放 Blob URL 内存
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
