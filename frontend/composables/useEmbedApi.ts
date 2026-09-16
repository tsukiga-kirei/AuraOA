import type { AuditResult } from '~/types/audit'
import type { EmbedContextResponse, EmbedExecuteRequest } from '~/types/embed'

/**
 * 校验字符串是否为 ofetch/HTTP 原始技术错误格式（例如: `[POST] "/api/embed/execute": 403` 或 `403 Forbidden`）
 */
export function isRawHttpErrorMessage(msg?: string | null): boolean {
  if (!msg || typeof msg !== 'string') return false
  const trimmed = msg.trim()
  return /^\s*\[[A-Z]+\]\s*["'].*["']\s*:\s*\d+/i.test(trimmed)
    || /^\s*\d{3}\s+(Forbidden|Unauthorized|Not Found|Bad Request|Internal Server Error)/i.test(trimmed)
    || trimmed.startsWith('FetchError:')
}

/**
 * 从网络或代理异常中提取面向用户的清晰业务错误信息
 */
export function extractEmbedErrorMessage(err: unknown, fallback = '请求失败'): string {
  const e = err as any
  if (!e) return fallback

  const body = e?.data ?? e?.response?._data
  const status = Number(e?.statusCode || e?.status || body?.statusCode || body?.status)

  // 1. 优先提取后端业务返回的 message
  const msgFromData = body?.message || body?.statusMessage
  if (typeof msgFromData === 'string' && msgFromData.trim() && !isRawHttpErrorMessage(msgFromData)) {
    return msgFromData.trim()
  }

  // 2. 尝试从 statusMessage 读取
  if (typeof e?.statusMessage === 'string' && e.statusMessage.trim() && !isRawHttpErrorMessage(e.statusMessage)) {
    return e.statusMessage.trim()
  }

  // 3. 尝试从 error.message 读取（排除 ofetch 自动生成的请求方法和路径）
  if (typeof e?.message === 'string' && e.message.trim() && !isRawHttpErrorMessage(e.message)) {
    return e.message.trim()
  }

  // 4. 根据 HTTP 状态码提供业务友好的中文兜底
  if (status === 403) {
    return '当前用户无权执行该操作，请联系管理员分配流程访问权限'
  }
  if (status === 401) {
    return '嵌入访问令牌无效或已过期，请刷新 OA 页面'
  }
  if (status === 404) {
    return '未找到该流程的配置信息'
  }
  if (status === 409) {
    return '审核任务冲突或正在处理中，请稍后重试'
  }
  if (status >= 500) {
    return '服务处理失败，请稍后重试'
  }

  return fallback
}

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
    try {
      const res = await $fetch<T>(path, {
        method: init?.method ?? 'GET',
        body: init?.body as Record<string, unknown> | undefined,
        credentials: 'include',
        headers: embedAuthHeaders(),
      })
      return res as T
    } catch (err: any) {
      const friendlyMsg = extractEmbedErrorMessage(err)
      const error = new Error(friendlyMsg) as any
      error.statusCode = err?.statusCode || err?.status
      error.data = err?.data
      error.originalError = err
      throw error
    }
  }

  async function getContext(processId: string, preferCached = false): Promise<EmbedContextResponse> {
    const q = new URLSearchParams({ process_id: processId })
    if (preferCached) q.set('prefer_cached', 'true')
    const oaUser = useEmbedAuth().getOAUserId()
    if (oaUser) q.set('oa_user_id', oaUser)
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
      if (status === 'completed' || status === 'failed' || status === 'cancelled') {
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
    if (!['pending', 'assembling', 'reasoning', 'extracting'].includes(submit.status) || !submit.id) {
      return submit as unknown as AuditResult
    }
    onProgress?.(submit as unknown as AuditResult & { progress_steps?: unknown[] })
    return await waitAuditJob(submit.id, onProgress)
  }

  return { getContext, executeEmbed, waitAuditJob }
}
