/**
 * 嵌入页与 OA 父页面通信：获取泛微 requestid
 *
 * 消息协议（与 docs/oa-configurations/assets/aura-embed-notify.js 一致）：
 * - iframe → parent: { type: 'aura-oa-request-requestid' }
 * - parent → iframe: { type: 'aura-oa-requestid', requestid: '598488', embed_token: '...' }
 * - parent → iframe: { type: 'aura-oa-url', url: '...' }（可选，从 URL 解析）
 */

export const EMBED_MSG_REQUEST_REQUESTID = 'aura-oa-request-requestid'
export const EMBED_MSG_REQUESTID = 'aura-oa-requestid'
export const EMBED_MSG_URL = 'aura-oa-url'

export interface EmbedParentContext {
  requestId: string
  embedToken: string
}

export function parseRequestIdFromUrl(url: string): string {
  if (!url) return ''
  try {
    const u = new URL(url)
    const fromSearch = u.searchParams.get('requestid') || u.searchParams.get('process_id')
    if (fromSearch) return fromSearch
    const hash = u.hash || ''
    const qIdx = hash.indexOf('?')
    if (qIdx >= 0) {
      const fromHash = new URLSearchParams(hash.slice(qIdx + 1)).get('requestid')
      if (fromHash) return fromHash
    }
  } catch {
    const m = url.match(/[?&#]requestid=(\d+)/i)
    if (m?.[1]) return m[1]
  }
  return ''
}

/** 从 URL 查询参数读取 embed_token（本地联调或 OA 在 iframe src 上拼接时使用） */
export function readEmbedTokenFromUrl(url: string): string {
  if (!url) return ''
  try {
    const u = new URL(url)
    return String(u.searchParams.get('embed_token') || u.searchParams.get('embedToken') || '').trim()
  } catch {
    return ''
  }
}

/** 从当前嵌入页地址读取 requestid + embed_token */
export function readEmbedContextFromSelfUrl(): EmbedParentContext {
  if (typeof window === 'undefined') return { requestId: '', embedToken: '' }
  const href = window.location.href
  return {
    requestId: parseRequestIdFromUrl(href),
    embedToken: readEmbedTokenFromUrl(href),
  }
}

/** 尝试读取 iframe 父页面 URL 中的 requestid（仅同源） */
export function readRequestIdFromParent(): string {
  if (typeof window === 'undefined' || window.parent === window) return ''
  try {
    return parseRequestIdFromUrl(window.parent.location.href)
  } catch {
    return ''
  }
}

/** 向 OA 父页请求 requestid（跨域时由父页脚本 WfForm + postMessage 响应） */
export function requestRequestIdFromParent(): void {
  if (typeof window === 'undefined' || window.parent === window) return
  window.parent.postMessage({ type: EMBED_MSG_REQUEST_REQUESTID }, '*')
}

/**
 * 等待 OA 父页上下文：当前页 URL → postMessage → 同源 parent URL
 */
export function waitForParentEmbedContext(options?: { intervalMs?: number; maxAttempts?: number; requireRequestId?: boolean }): Promise<EmbedParentContext> {
  if (typeof window === 'undefined') {
    return Promise.resolve({ requestId: '', embedToken: '' })
  }

  const intervalMs = options?.intervalMs ?? 300
  const maxAttempts = options?.maxAttempts ?? 200
  const requireRequestId = options?.requireRequestId ?? true

  const fromSelf = readEmbedContextFromSelfUrl()
  if (fromSelf.embedToken && (!requireRequestId || fromSelf.requestId)) {
    return Promise.resolve(fromSelf)
  }

  return new Promise((resolve) => {
    let attempts = 0
    let latestRequestId = fromSelf.requestId
    let latestToken = fromSelf.embedToken
    requestRequestIdFromParent()

    const tryFinish = () => {
      if (latestToken && (!requireRequestId || latestRequestId)) {
        finish({
          requestId: latestRequestId,
          embedToken: latestToken,
        })
        return true
      }
      return false
    }

    const onMessage = (event: MessageEvent) => {
      const data = event.data
      if (!data || typeof data !== 'object') return
      if (data.type === EMBED_MSG_REQUESTID) {
        latestRequestId = String(data.requestid || '')
        latestToken = String(data.embed_token || data.embedToken || '').trim()
        tryFinish()
        return
      }
      if (data.type === EMBED_MSG_URL && data.url) {
        const parsed = parseRequestIdFromUrl(String(data.url))
        if (parsed) {
          latestRequestId = parsed
          tryFinish()
        }
      }
    }

    const timer = setInterval(() => {
      attempts++
      const parentRequestId = readRequestIdFromParent()
      if (parentRequestId) {
        latestRequestId = parentRequestId
      }
      if (tryFinish()) {
        return
      }
      // 每隔约 3 秒再向父页要一次（WfForm 可能尚未就绪）
      if (attempts % 10 === 0) {
        requestRequestIdFromParent()
      }
      if (attempts >= maxAttempts) {
        finish({
          requestId: latestRequestId,
          embedToken: latestToken,
        })
      }
    }, intervalMs)

    function finish(ctx: EmbedParentContext) {
      clearInterval(timer)
      window.removeEventListener('message', onMessage)
      resolve(ctx)
    }

    window.addEventListener('message', onMessage)
  })
}

export function waitForParentRequestId(options?: { intervalMs?: number; maxAttempts?: number }): Promise<string> {
  return waitForParentEmbedContext(options).then(ctx => ctx.requestId)
}
