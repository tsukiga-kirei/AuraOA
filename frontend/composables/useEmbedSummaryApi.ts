import type { SummaryResult } from '~/types/process-summary'
import type { EmbedProcessSummary } from '~/types/embed'

export interface EmbedSummaryContextResponse {
  personal_result?: SummaryResult | null
  visible_block_ids?: string[]
  supported: boolean
  reason?: 'not_found_in_oa' | 'no_config' | 'config_inactive' | 'embed_disabled'
  message?: string
  process?: EmbedProcessSummary
  embed_enabled?: boolean
  has_summary?: boolean
  stale?: boolean
  stale_block_ids?: string[]
  should_auto_summary?: boolean
  last_summary_at?: string
  running_job_id?: string
  summary_result?: SummaryResult | null
  auto_retry_blocked?: boolean
	config_version_no?: number
	config_upgrade_available?: boolean
}

export interface EmbedSummaryExecuteRequest {
  process_id: string
  process_type?: string
  title?: string
  trigger_source?: 'summary_embed_auto' | 'summary_embed_manual'
  trigger_detail?: 'visible_open' | 'manual'
	use_latest_config?: boolean
}

export const useEmbedSummaryApi = () => {
  const { embedAuthHeaders } = useEmbedAuth()
  const POLL_INTERVAL_MS = 1500
  const SUMMARY_TIMEOUT_MS = 35 * 60 * 1000

  async function embedSummaryFetch<T>(
    path: string,
    init?: { method?: 'GET' | 'POST'; body?: unknown },
  ): Promise<T> {
    const res = await $fetch<T>(path, {
      method: init?.method ?? 'GET',
      body: init?.body as Record<string, unknown> | undefined,
      credentials: 'include',
      headers: embedAuthHeaders(),
    })
    return res as T
  }

  async function getSummaryContext(processId: string, preferCached = false): Promise<EmbedSummaryContextResponse> {
    const q = new URLSearchParams({ process_id: processId })
    if (preferCached) q.set('prefer_cached', 'true')
    return await embedSummaryFetch<EmbedSummaryContextResponse>(`/api/embed/summary/context?${q.toString()}`)
  }

  async function waitSummaryJob(
    jobId: string,
    onProgress?: (st: SummaryResult) => void,
  ): Promise<SummaryResult> {
    const deadline = Date.now() + SUMMARY_TIMEOUT_MS
    while (Date.now() < deadline) {
      const st = await embedSummaryFetch<SummaryResult>(`/api/embed/summary/jobs/${encodeURIComponent(jobId)}`)
      onProgress?.(st)
      if (st.status === 'completed' || st.status === 'failed' || st.status === 'cancelled') {
        return st
      }
      await new Promise(r => setTimeout(r, POLL_INTERVAL_MS))
    }
    throw new Error('总结等待超时')
  }

  async function executeSummaryEmbed(
    req: EmbedSummaryExecuteRequest,
    onProgress?: (st: SummaryResult) => void,
  ): Promise<SummaryResult> {
    const submit = await embedSummaryFetch<SummaryResult & { status: string; id: string }>('/api/embed/summary/execute', {
      method: 'POST',
      body: req,
    })
    if (!['pending', 'assembling', 'reasoning', 'extracting'].includes(submit.status) || !submit.id) {
      return submit
    }
    onProgress?.(submit)
    return await waitSummaryJob(submit.id, onProgress)
  }

  return { getSummaryContext, executeSummaryEmbed, waitSummaryJob }
}
