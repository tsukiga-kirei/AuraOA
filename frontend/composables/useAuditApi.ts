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
} from '~/types/audit'

export type {
  OAProcessItem, AuditResult, AuditChainItem,
  BatchAuditResponse, AuditTab, AuditExecuteRequest,
  BatchAuditRequest, AuditStats,
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

  async function executeAudit(req: AuditExecuteRequest): Promise<AuditResult> {
    return await authFetch<AuditResult>('/api/audit/execute', {
      method: 'POST',
      body: req,
    })
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

  return {
    getStats,
    listProcesses,
    executeAudit,
    batchAudit,
    getAuditChain,
    getAuditResult,
  }
}
