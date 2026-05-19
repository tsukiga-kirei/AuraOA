/**
 * 从 OA 父页面 URL 解析泛微 requestid。
 */

const REQUEST_ID_KEYS = ['requestid', 'requestId', 'RequestId']

export function parseRequestIdFromUrl(url: string): string {
  if (!url) return ''
  try {
    const base = typeof window !== 'undefined' ? window.location.origin : 'http://localhost'
    const u = new URL(url, base)
    for (const key of REQUEST_ID_KEYS) {
      const fromSearch = u.searchParams.get(key)
      if (fromSearch) return fromSearch.trim()
    }
    const hash = u.hash || ''
    const hashQuery = hash.includes('?') ? hash.split('?').slice(1).join('?') : ''
    if (hashQuery) {
      const params = new URLSearchParams(hashQuery)
      for (const key of REQUEST_ID_KEYS) {
        const v = params.get(key)
        if (v) return v.trim()
      }
    }
  } catch {
    // 降级：正则匹配
    const m = url.match(/[?&#]requestid=(\d+)/i)
    if (m?.[1]) return m[1]
  }
  return ''
}

/** 尝试读取 iframe 父页面 URL 中的 requestid（需同源）。 */
export function readRequestIdFromParent(): string {
  if (typeof window === 'undefined') return ''
  try {
    if (window.parent === window) return ''
    return parseRequestIdFromUrl(window.parent.location.href)
  } catch {
    return ''
  }
}

/** 轮询父页面 URL，直到解析到 requestid 或超时。 */
export function waitForParentRequestId(options?: {
  intervalMs?: number
  maxAttempts?: number
}): Promise<string> {
  const intervalMs = options?.intervalMs ?? 300
  const maxAttempts = options?.maxAttempts ?? 200

  return new Promise((resolve) => {
    let attempts = 0
    const tryRead = () => {
      const id = readRequestIdFromParent()
      if (id) {
        resolve(id)
        return true
      }
      return false
    }

    if (tryRead()) return

    const timer = setInterval(() => {
      attempts++
      if (tryRead() || attempts >= maxAttempts) {
        clearInterval(timer)
        if (!readRequestIdFromParent()) resolve('')
      }
    }, intervalMs)
  })
}

export type EmbedParentMessage =
  | { type: 'aura-oa-requestid'; requestid: string }
  | { type: 'aura-oa-context'; url?: string; requestid?: string }

export function parseEmbedParentMessage(data: unknown): string {
  if (!data || typeof data !== 'object') return ''
  const msg = data as EmbedParentMessage
  if (msg.type === 'aura-oa-requestid' && msg.requestid) {
    return String(msg.requestid).trim()
  }
  if (msg.type === 'aura-oa-context') {
    if (msg.requestid) return String(msg.requestid).trim()
    if (msg.url) return parseRequestIdFromUrl(msg.url)
  }
  return ''
}
