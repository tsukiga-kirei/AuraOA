/**
 * useSummaryWorkbenchApi — 流程总结前台工作台 API。
 * 对接 /api/summary/processes、/stats、/execute、/jobs 与 /history。
 */
import type {
  SummaryResult,
  SummaryWorkbenchListResponse,
  SummaryWorkbenchStats,
} from '~/types/process-summary'

export interface SummaryWorkbenchQuery {
  keyword?: string
  applicant?: string
  department?: string
  process_type?: string
  summary_status?: string
  start_date?: string
  end_date?: string
  page?: number
  page_size?: number
}

export const useSummaryWorkbenchApi = () => {
  const { authFetch } = useAuth()
  const { t } = useI18n()
  const pollIntervalMs = 1500
  const timeoutMs = 35 * 60 * 1000

  const queryString = (query: SummaryWorkbenchQuery) => {
    const params = new URLSearchParams()
    Object.entries(query).forEach(([key, value]) => {
      if (value !== undefined && value !== '') params.set(key, String(value))
    })
    return params.toString()
  }

  async function listProcesses(query: SummaryWorkbenchQuery): Promise<SummaryWorkbenchListResponse> {
    return await authFetch<SummaryWorkbenchListResponse>(`/api/summary/processes?${queryString(query)}`)
  }

  async function getStats(query: SummaryWorkbenchQuery): Promise<SummaryWorkbenchStats> {
    return await authFetch<SummaryWorkbenchStats>(`/api/summary/stats?${queryString(query)}`)
  }

  async function getJob(jobId: string): Promise<SummaryResult> {
    return await authFetch<SummaryResult>(`/api/summary/jobs/${encodeURIComponent(jobId)}`)
  }

  async function waitJob(jobId: string, onProgress?: (result: SummaryResult) => void): Promise<SummaryResult> {
    const deadline = Date.now() + timeoutMs
    while (Date.now() < deadline) {
      const result = await getJob(jobId)
      onProgress?.(result)
      if (['completed', 'failed', 'cancelled'].includes(result.status || '')) return result
      await new Promise(resolve => setTimeout(resolve, pollIntervalMs))
    }
    throw new Error(t('summary.timeout'))
  }

  async function execute(
    process: { process_id: string; process_type: string; title: string },
    useLatestConfig = false,
    onProgress?: (result: SummaryResult) => void,
  ): Promise<SummaryResult> {
    const submitted = await authFetch<SummaryResult & { id: string }>('/api/summary/execute', {
      method: 'POST',
      body: { ...process, use_latest_config: useLatestConfig },
    })
    onProgress?.(submitted)
    if (!submitted.id || !['pending', 'assembling', 'reasoning', 'extracting'].includes(submitted.status || '')) {
      return submitted
    }
    return await waitJob(submitted.id, onProgress)
  }

  async function getHistory(processId: string) {
    return await authFetch<{ chain: unknown[] }>(`/api/summary/history/${encodeURIComponent(processId)}`)
  }

  return { listProcesses, getStats, execute, getJob, waitJob, getHistory }
}
