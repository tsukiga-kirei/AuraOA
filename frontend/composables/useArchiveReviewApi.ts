/**
 * useArchiveReviewApi — 归档复盘工作台 API 调用封装
 * 对接后端路由组：
 *   GET  /api/archive/stats              归档统计数据
 *   GET  /api/archive/processes          归档流程列表（分页+筛选）
 *   POST /api/archive/execute            提交单条归档复盘任务
 *   POST /api/archive/batch              批量提交归档复盘
 *   GET  /api/archive/jobs/:id           轮询异步任务状态
 *   POST /api/archive/cancel/:id         取消正在执行的任务
 *   GET  /api/archive/result/:id         获取归档复盘结果
 *   GET  /api/archive/history/:processId 获取流程历史复盘记录
 *   GET  /api/tenant/settings/archive-configs 获取已配置的流程类型列表
 */

import type {
  ArchiveBatchExecuteRequest,
  ArchiveBatchExecuteResponse,
  ArchiveProcessListResponse,
  ArchiveProcessTypeOption,
  ArchiveReviewExecuteRequest,
  ArchiveReviewHistoryItem,
  ArchiveReviewResult,
  ArchiveReviewStats,
  ArchiveReviewSubmitResponse,
} from '~/types/archive-review'

export type {
  ArchiveBatchExecuteRequest,
  ArchiveBatchExecuteResponse,
  ArchiveProcessListResponse,
  ArchiveProcessTypeOption,
  ArchiveReviewExecuteRequest,
  ArchiveReviewHistoryItem,
  ArchiveReviewResult,
  ArchiveReviewStats,
  ArchiveReviewSubmitResponse,
}

// 轮询间隔（毫秒）
const POLL_INTERVAL_MS = 1500
// 前端等待超时（略大于服务端 30 分钟超时，避免前端先放弃）
const ARCHIVE_TIMEOUT_MS = 35 * 60 * 1000

export const useArchiveReviewApi = () => {
  const { authFetch } = useAuth()

  /**
   * 获取归档复盘统计数据（总数、各状态分布、时间范围内的趋势等）。
   * @param params 可选的时间范围筛选参数
   */
  async function getStats(params?: {
    start_date?: string
    end_date?: string
  }): Promise<ArchiveReviewStats> {
    const query = new URLSearchParams()
    if (params?.start_date) query.set('start_date', params.start_date)
    if (params?.end_date) query.set('end_date', params.end_date)
    const qs = query.toString()
    return await authFetch<ArchiveReviewStats>(qs ? `/api/archive/stats?${qs}` : '/api/archive/stats')
  }

  /**
   * 分页查询归档流程列表，支持关键词、申请人、流程类型、部门、状态等多维度筛选。
   * @param params 筛选和分页参数
   */
  async function listProcesses(params?: {
    keyword?: string
    applicant?: string
    process_type?: string
    department?: string
    audit_status?: string
    page?: number
    page_size?: number
    start_date?: string
    end_date?: string
  }): Promise<ArchiveProcessListResponse> {
    const query = new URLSearchParams()
    if (params?.keyword) query.set('keyword', params.keyword)
    if (params?.applicant) query.set('applicant', params.applicant)
    if (params?.process_type) query.set('process_type', params.process_type)
    if (params?.department) query.set('department', params.department)
    if (params?.audit_status) query.set('audit_status', params.audit_status)
    if (params?.page) query.set('page', String(params.page))
    if (params?.page_size) query.set('page_size', String(params.page_size))
    if (params?.start_date) query.set('start_date', params.start_date)
    if (params?.end_date) query.set('end_date', params.end_date)
    const qs = query.toString()
    return await authFetch<ArchiveProcessListResponse>(qs ? `/api/archive/processes?${qs}` : '/api/archive/processes')
  }

  /**
   * 轮询异步归档任务直到完成或失败。
   * 每隔 POLL_INTERVAL_MS 毫秒查询一次任务状态，超过 ARCHIVE_TIMEOUT_MS 则抛出超时错误。
   * @param jobId 任务 ID
   * @param onProgress 进度回调，每次轮询都会调用（可用于更新进度条）
   * @returns 最终的任务结果
   */
  async function waitArchiveJob(
    jobId: string,
    onProgress?: (st: ArchiveReviewResult) => void,
  ): Promise<ArchiveReviewResult> {
    const deadline = Date.now() + ARCHIVE_TIMEOUT_MS
    while (Date.now() < deadline) {
      const st = await authFetch<ArchiveReviewResult>(`/api/archive/jobs/${encodeURIComponent(jobId)}`)
      onProgress?.(st)
      if (st.status === 'completed' || st.status === 'failed') {
        return st
      }
      await new Promise(resolve => setTimeout(resolve, POLL_INTERVAL_MS))
    }
    throw new Error('归档复盘等待超时，请稍后刷新列表查看结果')
  }

  /**
   * 提交单条归档复盘任务并等待结果（内部轮询异步任务状态）。
   * 后端先返回 pending 状态的任务 ID，前端持续轮询直到完成。
   * @param req 归档复盘请求参数
   * @param onProgress 进度回调（可选）
   * @returns 归档复盘最终结果
   */
  async function executeReview(
    req: ArchiveReviewExecuteRequest,
    onProgress?: (st: ArchiveReviewResult) => void,
  ): Promise<ArchiveReviewResult> {
    const submit = await authFetch<ArchiveReviewSubmitResponse>('/api/archive/execute', {
      method: 'POST',
      body: req,
    })
    // 若后端直接返回终态（非 pending），则无需轮询
    if (submit.status !== 'pending' || !submit.id) {
      return submit as unknown as ArchiveReviewResult
    }
    return await waitArchiveJob(submit.id, onProgress)
  }

  /**
   * 批量提交归档复盘任务（后端异步处理，不等待结果）。
   * @param req 批量请求参数（流程 ID 列表等）
   */
  async function batchReview(req: ArchiveBatchExecuteRequest): Promise<ArchiveBatchExecuteResponse> {
    return await authFetch<ArchiveBatchExecuteResponse>('/api/archive/batch', {
      method: 'POST',
      body: req,
    })
  }

  /**
   * 取消正在执行的归档复盘任务。
   * @param jobId 任务 ID
   */
  async function cancelArchiveJob(jobId: string): Promise<void> {
    await authFetch(`/api/archive/cancel/${encodeURIComponent(jobId)}`, {
      method: 'POST',
    })
  }

  /**
   * 获取指定归档复盘任务的最终结果（用于结果详情页）。
   * @param id 归档日志 ID 或任务 ID
   */
  async function getArchiveResult(id: string): Promise<ArchiveReviewResult> {
    return await authFetch<ArchiveReviewResult>(`/api/archive/result/${encodeURIComponent(id)}`)
  }

  /**
   * 获取指定流程的历史归档复盘记录列表（按时间倒序）。
   * @param processId 流程 ID
   */
  async function getArchiveHistory(processId: string): Promise<ArchiveReviewHistoryItem[]> {
    return await authFetch<ArchiveReviewHistoryItem[]>(`/api/archive/history/${encodeURIComponent(processId)}`)
  }

  /**
   * 获取当前租户已配置的归档流程类型列表（用于筛选下拉框）。
   * 返回的是已启用的归档配置，每项包含流程类型编码和显示名。
   */
  async function getProcessTypes(): Promise<ArchiveProcessTypeOption[]> {
    return await authFetch<ArchiveProcessTypeOption[]>('/api/tenant/settings/archive-configs')
  }

  return {
    getStats,
    listProcesses,
    waitArchiveJob,
    executeReview,
    batchReview,
    cancelArchiveJob,
    getArchiveResult,
    getArchiveHistory,
    getProcessTypes,
  }
}
