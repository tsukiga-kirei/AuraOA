// useAuditApi — 审核工作台 API 调用

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

  async function getStats(): Promise<AuditStats> {
    return await authFetch<AuditStats>('/api/audit/stats')
  }

  async function listProcesses(tab: AuditTab, params?: {
    keyword?: string
    applicant?: string
    process_type?: string
    department?: string
    audit_status?: string
    page?: number
    page_size?: number
  }): Promise<{ items: OAProcessItem[]; total: number }> {
    const query = new URLSearchParams({ tab })
    if (params?.keyword) query.set('keyword', params.keyword)
    if (params?.applicant) query.set('applicant', params.applicant)
    if (params?.process_type) query.set('process_type', params.process_type)
    if (params?.department) query.set('department', params.department)
    if (params?.audit_status) query.set('audit_status', params.audit_status)
    if (params?.page) query.set('page', String(params.page))
    if (params?.page_size) query.set('page_size', String(params.page_size))
    return await authFetch<{ items: OAProcessItem[]; total: number }>(`/api/audit/processes?${query.toString()}`)
  }

  const POLL_INTERVAL_MS = 1500
  /** 略大于服务端非终态超时（30m），避免前端先放弃而后端仍可能完成 */
  const AUDIT_TIMEOUT_MS = 35 * 60 * 1000

  /** 轮询异步任务直到完成或失败 */
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

  /** 提交审核并等待结果（内部轮询 Redis Stream 异步任务） */
  async function executeAudit(
    req: AuditExecuteRequest,
    onProgress?: (st: AuditResult & { progress_steps?: unknown[] }) => void,
  ): Promise<AuditResult> {
    const submit = await authFetch<AuditSubmitResponse>('/api/audit/execute', {
      method: 'POST',
      body: req,
    })
    if (submit.status !== 'pending' || !submit.id) {
      return submit as unknown as AuditResult
    }
    return await waitAuditJob(submit.id, onProgress)
  }

  async function batchAudit(req: BatchAuditRequest): Promise<BatchAuditResponse> {
    return await authFetch<BatchAuditResponse>('/api/audit/batch', {
      method: 'POST',
      body: req,
    })
  }

  async function getAuditChain(processId: string): Promise<AuditChainItem[]> {
    return await authFetch<AuditChainItem[]>(`/api/audit/chain/${encodeURIComponent(processId)}`)
  }

  async function getAuditResult(auditLogId: string): Promise<AuditResult> {
    return await authFetch<AuditResult>(`/api/audit/result/${encodeURIComponent(auditLogId)}`)
  }

  /** 获取当前租户已配置的流程类型列表（用于筛选下拉） */
  async function getProcessTypes(): Promise<{ process_type: string; process_type_label: string; config_id: string }[]> {
    return await authFetch<{ process_type: string; process_type_label: string; config_id: string }[]>('/api/tenant/settings/processes')
  }

  return {
    getStats,
    listProcesses,
    executeAudit,
    waitAuditJob,
    batchAudit,
    getAuditChain,
    getAuditResult,
    getProcessTypes,
  }
}
