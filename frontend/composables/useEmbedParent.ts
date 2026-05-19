/**
 * 嵌入页与 OA 父页面通信：获取泛微 requestid
 *
 * 消息协议（与 docs/oa-configurations/assets/aura-embed-notify.js 一致）：
 * - iframe → parent: { type: 'aura-oa-request-requestid' }
 * - parent → iframe: { type: 'aura-oa-requestid', requestid: '598488' }
 * - parent → iframe: { type: 'aura-oa-url', url: '...' }（可选，从 URL 解析）
 */

export const EMBED_MSG_REQUEST_REQUESTID = 'aura-oa-request-requestid'
export const EMBED_MSG_REQUESTID = 'aura-oa-requestid'
export const EMBED_MSG_URL = 'aura-oa-url'

export function parseRequestIdFromUrl(url: string): string {
  if (!url) return ''
  try {
    const u = new URL(url)
    const fromSearch = u.searchParams.get('requestid')
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
 * 等待 requestid：同源读 parent → 主动 postMessage 请求 → 监听父页推送
 */
export function waitForParentRequestId(options?: { intervalMs?: number; maxAttempts?: number }): Promise<string> {
  if (typeof window === 'undefined') return Promise.resolve('')

  const intervalMs = options?.intervalMs ?? 300
  const maxAttempts = options?.maxAttempts ?? 200

  const immediate = readRequestIdFromParent()
  if (immediate) return Promise.resolve(immediate)

  return new Promise((resolve) => {
    let attempts = 0
    requestRequestIdFromParent()

    const onMessage = (event: MessageEvent) => {
      const data = event.data
      if (!data || typeof data !== 'object') return
      if (data.type === EMBED_MSG_REQUESTID && data.requestid) {
        finish(String(data.requestid))
        return
      }
      if (data.type === EMBED_MSG_URL && data.url) {
        const parsed = parseRequestIdFromUrl(String(data.url))
        if (parsed) finish(parsed)
      }
    }

    const timer = setInterval(() => {
      attempts++
      const found = readRequestIdFromParent()
      if (found) {
        finish(found)
        return
      }
      // 每隔约 3 秒再向父页要一次（WfForm 可能尚未就绪）
      if (attempts % 10 === 0) {
        requestRequestIdFromParent()
      }
      if (attempts >= maxAttempts) {
        finish('')
      }
    }, intervalMs)

    function finish(id: string) {
      clearInterval(timer)
      window.removeEventListener('message', onMessage)
      resolve(id)
    }

    window.addEventListener('message', onMessage)
  })
}
