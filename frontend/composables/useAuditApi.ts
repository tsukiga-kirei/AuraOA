/**
 * useAuditApi — 审核工作台 API 调用封装
 * 对接后端路由组：
 *   GET  /api/audit/stats              审核统计数据
 *   GET  /api/audit/processes          审核流程列表（分页+筛选）
 *   POST /api/audit/execute            提交单条审核任务
 *   POST /api/audit/batch              批量提交审核
 *   GET  /api/audit/jobs/:id           轮询异步任务状态
 *   POST /api/audit/cancel/:id         取消正在执行的任务
 *   GET  /api/audit/chain/:processId   获取流程审核链
 *   GET  /api/audit/result/:id         获取审核结果
 *   GET  /api/tenant/settings/processes 获取已配置的流程类型列表
 */

import type {
  OAProcessItem,
  AuditResult,
  AuditChainItem,
  BatchAuditResponse,
  AuditTab,
  AuditExecuteRequest,
  BatchAuditRequest,
  AuditStats,
  AuditSubmitResponse,
  AuditRunStatus,
} from '~/types/audit'

export type {
  OAProcessItem, AuditResult, AuditChainItem,
  BatchAuditResponse, AuditTab, AuditExecuteRequest,
  BatchAuditRequest, AuditStats, AuditSubmitResponse, AuditRunStatus,
}

export const useAuditApi = () => {
  const { authFetch } = useAuth()

  /**
   * 获取审核统计数据（待审核数、已完成数、各状态分布等）。
   * @param params 可选的时间范围筛选参数
   */
  async function getStats(params?: {
    start_date?: string
    end_date?: string
  }): Promise<AuditStats> {
    const query = new URLSearchParams()
    if (params?.start_date) query.set('start_date', params.start_date)
    if (params?.end_date) query.set('end_date', params.end_date)
    const qs = query.toString()
    return await authFetch<AuditStats>(qs ? `/api/audit/stats?${qs}` : '/api/audit/stats')
  }

  /**
   * 分页查询审核流程列表，支持按 tab 分类和多维度筛选。
   * @param tab 当前 tab（待审核/已完成/全部等）
   * @param params 筛选和分页参数
   */
  async function listProcesses(tab: AuditTab, params?: {
    keyword?: string
    applicant?: string
    process_type?: string
    department?: string
    audit_status?: string
    page?: number
    page_size?: number
    start_date?: string
    end_date?: string
  }): Promise<{ items: OAProcessItem[]; total: number; page?: number; page_size?: number }> {
    const query = new URLSearchParams({ tab })
    if (params?.keyword) query.set('keyword', params.keyword)
    if (params?.applicant) query.set('applicant', params.applicant)
    if (params?.process_type) query.set('process_type', params.process_type)
    if (params?.department) query.set('department', params.department)
    if (params?.audit_status) query.set('audit_status', params.audit_status)
    if (params?.page) query.set('page', String(params.page))
    if (params?.page_size) query.set('page_size', String(params.page_size))
    if (params?.start_date) query.set('start_date', params.start_date)
    if (params?.end_date) query.set('end_date', params.end_date)
    return await authFetch<{ items: OAProcessItem[]; total: number; page?: number; page_size?: number }>(
      `/api/audit/processes?${query.toString()}`,
    )
  }

  // 轮询间隔（毫秒）
  const POLL_INTERVAL_MS = 1500
  // 前端等待超时（略大于服务端 30 分钟非终态超时，避免前端先放弃而后端仍可能完成）
  const AUDIT_TIMEOUT_MS = 35 * 60 * 1000

  /**
   * 轮询异步审核任务直到完成或失败。
   * 每隔 POLL_INTERVAL_MS 毫秒查询一次任务状态，超过 AUDIT_TIMEOUT_MS 则抛出超时错误。
   * @param jobId 任务 ID
   * @param onProgress 进度回调，每次轮询都会调用（可用于更新进度步骤）
   * @returns 最终的审核结果
   */
  async function waitAuditJob(
    jobId: string,
    onProgress?: (st: AuditResult & { progress_steps?: unknown[]; updated_at?: string }) => void,
  ): Promise<AuditResult> {
    const deadline = Date.now() + AUDIT_TIMEOUT_MS
    while (Date.now() < deadline) {
      const st = await authFetch<AuditResult & { progress_steps?: unknown[]; updated_at?: string }>(
        `/api/audit/jobs/${encodeURIComponent(jobId)}`,
      )
      onProgress?.(st)
      const status = st.status as AuditRunStatus | undefined
      if (status === 'completed' || status === 'failed') {
        return st as AuditResult
      }
      await new Promise(r => setTimeout(r, POLL_INTERVAL_MS))
    }
    throw new Error('审核等待超时，请稍后刷新列表查看结果')
  }

  /**
   * 提交审核任务并等待结果（内部轮询 Redis Stream 异步任务）。
   * 后端先返回 pending 状态的任务 ID，前端持续轮询直到完成。
   * @param req 审核请求参数（流程 ID、配置 ID 等）
   * @param onProgress 进度回调（可选，用于展示审核步骤进度）
   * @returns 审核最终结果
   */
  async function executeAudit(
    req: AuditExecuteRequest,
    onProgress?: (st: AuditResult & { progress_steps?: unknown[] }) => void,
  ): Promise<AuditResult> {
    const submit = await authFetch<AuditSubmitResponse>('/api/audit/execute', {
      method: 'POST',
      body: req,
    })
    // 若后端直接返回终态（非 pending），则无需轮询
    if (submit.status !== 'pending' || !submit.id) {
      return submit as unknown as AuditResult
    }
    return await waitAuditJob(submit.id, onProgress)
  }

  /**
   * 批量提交审核任务（后端异步处理，不等待结果）。
   * @param req 批量请求参数（流程 ID 列表等）
   */
  async function batchAudit(req: BatchAuditRequest): Promise<BatchAuditResponse> {
    return await authFetch<BatchAuditResponse>('/api/audit/batch', {
      method: 'POST',
      body: req,
    })
  }

  /**
   * 获取指定流程的完整审核链（历次审核记录按时间排列）。
   * @param processId 流程 ID
   */
  async function getAuditChain(processId: string): Promise<AuditChainItem[]> {
    return await authFetch<AuditChainItem[]>(`/api/audit/chain/${encodeURIComponent(processId)}`)
  }

  /**
   * 获取指定审核日志的详细结果（用于结果详情页）。
   * @param auditLogId 审核日志 ID
   */
  async function getAuditResult(auditLogId: string): Promise<AuditResult> {
    return await authFetch<AuditResult>(`/api/audit/result/${encodeURIComponent(auditLogId)}`)
  }

  /**
   * 获取当前租户已配置的流程类型列表（用于筛选下拉框）。
   * 返回已启用的审核配置，每项包含流程类型编码、显示名和配置 ID。
   */
  async function getProcessTypes(): Promise<{ process_type: string; process_type_label: string; config_id: string }[]> {
    return await authFetch<{ process_type: string; process_type_label: string; config_id: string }[]>('/api/tenant/settings/processes')
  }

  /**
   * 取消正在执行的审核任务。
   * @param auditLogId 审核日志 ID
   */
  async function cancelAuditJob(auditLogId: string): Promise<void> {
    await authFetch(`/api/audit/cancel/${encodeURIComponent(auditLogId)}`, { method: 'POST' })
  }

  return {
    getStats,
    listProcesses,
    executeAudit,
    waitAuditJob,
    cancelAuditJob,
    batchAudit,
    getAuditChain,
    getAuditResult,
    getProcessTypes,
  }
}
