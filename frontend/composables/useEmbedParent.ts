/**
 * 从 OA 父页面 URL 解析泛微 requestid（固定嵌入地址，不传 query 参数）。
 */

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

/** 尝试读取 iframe 父页面完整 URL 中的 requestid（需同源） */
export function readRequestIdFromParent(): string {
  if (typeof window === 'undefined' || window.parent === window) return ''
  try {
    return parseRequestIdFromUrl(window.parent.location.href)
  } catch {
    return ''
  }
}

/**
 * 轮询父页面 URL 直至解析到 requestid；跨域时等待 postMessage（OA 可选兜底）。
 */
export function waitForParentRequestId(options?: { intervalMs?: number; maxAttempts?: number }): Promise<string> {
  if (typeof window === 'undefined') return Promise.resolve('')

  const intervalMs = options?.intervalMs ?? 300
  const maxAttempts = options?.maxAttempts ?? 200

  const immediate = readRequestIdFromParent()
  if (immediate) return Promise.resolve(immediate)

  return new Promise((resolve) => {
    let attempts = 0

    const onMessage = (event: MessageEvent) => {
      const data = event.data
      if (!data || typeof data !== 'object') return
      if (data.type === 'aura-oa-requestid' && data.requestid) {
        finish(String(data.requestid))
        return
      }
      if (data.type === 'aura-oa-url' && data.url) {
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
