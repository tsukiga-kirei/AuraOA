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

const POLL_INTERVAL_MS = 1500
const ARCHIVE_TIMEOUT_MS = 35 * 60 * 1000

export const useArchiveReviewApi = () => {
  const { authFetch } = useAuth()

  async function getStats(): Promise<ArchiveReviewStats> {
    return await authFetch<ArchiveReviewStats>('/api/archive/stats')
  }

  async function listProcesses(params?: {
    keyword?: string
    applicant?: string
    process_type?: string
    department?: string
    audit_status?: string
    page?: number
    page_size?: number
  }): Promise<ArchiveProcessListResponse> {
    const query = new URLSearchParams()
    if (params?.keyword) query.set('keyword', params.keyword)
    if (params?.applicant) query.set('applicant', params.applicant)
    if (params?.process_type) query.set('process_type', params.process_type)
    if (params?.department) query.set('department', params.department)
    if (params?.audit_status) query.set('audit_status', params.audit_status)
    if (params?.page) query.set('page', String(params.page))
    if (params?.page_size) query.set('page_size', String(params.page_size))
    const qs = query.toString()
    return await authFetch<ArchiveProcessListResponse>(qs ? `/api/archive/processes?${qs}` : '/api/archive/processes')
  }

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

  async function executeReview(
    req: ArchiveReviewExecuteRequest,
    onProgress?: (st: ArchiveReviewResult) => void,
  ): Promise<ArchiveReviewResult> {
    const submit = await authFetch<ArchiveReviewSubmitResponse>('/api/archive/execute', {
      method: 'POST',
      body: req,
    })
    if (submit.status !== 'pending' || !submit.id) {
      return submit as unknown as ArchiveReviewResult
    }
    return await waitArchiveJob(submit.id, onProgress)
  }

  async function batchReview(req: ArchiveBatchExecuteRequest): Promise<ArchiveBatchExecuteResponse> {
    return await authFetch<ArchiveBatchExecuteResponse>('/api/archive/batch', {
      method: 'POST',
      body: req,
    })
  }

  async function cancelArchiveJob(jobId: string): Promise<void> {
    await authFetch(`/api/archive/cancel/${encodeURIComponent(jobId)}`, {
      method: 'POST',
    })
  }

  async function getArchiveResult(id: string): Promise<ArchiveReviewResult> {
    return await authFetch<ArchiveReviewResult>(`/api/archive/result/${encodeURIComponent(id)}`)
  }

  async function getArchiveHistory(processId: string): Promise<ArchiveReviewHistoryItem[]> {
    return await authFetch<ArchiveReviewHistoryItem[]>(`/api/archive/history/${encodeURIComponent(processId)}`)
  }

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
