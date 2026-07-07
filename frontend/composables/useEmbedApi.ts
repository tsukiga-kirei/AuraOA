import type { AuditResult } from '~/types/audit'
import type { EmbedContextResponse, EmbedExecuteRequest } from '~/types/embed'

/**
 * useEmbedApi — OA 嵌入展示页 API（经 Nuxt 服务端代理，无需用户登录）
 */
export const useEmbedApi = () => {
  const { embedAuthHeaders } = useEmbedAuth()
  const POLL_INTERVAL_MS = 1500
  const AUDIT_TIMEOUT_MS = 35 * 60 * 1000

  async function embedFetch<T>(
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

  async function getContext(processId: string): Promise<EmbedContextResponse> {
    const q = new URLSearchParams({ process_id: processId })
    return await embedFetch<EmbedContextResponse>(`/api/embed/context?${q.toString()}`)
  }

  async function waitAuditJob(
    jobId: string,
    onProgress?: (st: AuditResult & { progress_steps?: unknown[] }) => void,
  ): Promise<AuditResult> {
    const deadline = Date.now() + AUDIT_TIMEOUT_MS
    while (Date.now() < deadline) {
      const st = await embedFetch<AuditResult & { progress_steps?: unknown[] }>(
        `/api/embed/jobs/${encodeURIComponent(jobId)}`,
      )
      onProgress?.(st)
      const status = st.status
      if (status === 'completed' || status === 'failed') {
        return st as AuditResult
      }
      await new Promise(r => setTimeout(r, POLL_INTERVAL_MS))
    }
    throw new Error('审核等待超时')
  }

  async function executeEmbed(
    req: EmbedExecuteRequest,
    onProgress?: (st: AuditResult & { progress_steps?: unknown[] }) => void,
  ): Promise<AuditResult> {
    const submit = await embedFetch<{ status: string; id: string }>('/api/embed/execute', {
      method: 'POST',
      body: req,
    })
    if (submit.status !== 'pending' || !submit.id) {
      return submit as unknown as AuditResult
    }
    onProgress?.(submit as unknown as AuditResult & { progress_steps?: unknown[] })
    return await waitAuditJob(submit.id, onProgress)
  }

  return { getContext, executeEmbed, waitAuditJob }
}
