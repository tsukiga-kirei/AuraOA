import type { SummaryResult } from '~/types/process-summary'
import type { EmbedProcessSummary } from '~/types/embed'

export interface EmbedSummaryContextResponse {
  supported: boolean
  reason?: 'not_found_in_oa' | 'no_config' | 'config_inactive' | 'embed_disabled'
  message?: string
  process?: EmbedProcessSummary
  embed_enabled?: boolean
  has_summary?: boolean
  stale?: boolean
  should_auto_summary?: boolean
  last_summary_at?: string
  running_job_id?: string
  summary_result?: SummaryResult | null
}

export interface EmbedSummaryExecuteRequest {
  process_id: string
  process_type?: string
  title?: string
  trigger_source?: 'summary_embed_auto' | 'summary_embed_manual'
}

async function embedSummaryFetch<T>(
  path: string,
  init?: { method?: 'GET' | 'POST'; body?: unknown },
): Promise<T> {
  return await $fetch<T>(path, {
    method: init?.method ?? 'GET',
    body: init?.body as Record<string, unknown> | undefined,
  })
}

export const useEmbedSummaryApi = () => {
  const POLL_INTERVAL_MS = 1500
  const SUMMARY_TIMEOUT_MS = 35 * 60 * 1000

  async function getSummaryContext(processId: string): Promise<EmbedSummaryContextResponse> {
    const q = new URLSearchParams({ process_id: processId })
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
      if (st.status === 'completed' || st.status === 'failed') {
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
    if (submit.status !== 'pending' || !submit.id) {
      return submit
    }
    onProgress?.(submit)
    return await waitSummaryJob(submit.id, onProgress)
  }

  return { getSummaryContext, executeSummaryEmbed, waitSummaryJob }
}
