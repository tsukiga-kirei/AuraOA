/** 嵌入页服务端代理：携带令牌访问 Go /api/embed，令牌不暴露给浏览器 */
export function getEmbedBackend() {
  const config = useRuntimeConfig()
  const token = String(config.embedAccessToken || '')
  const tenantCode = String(config.embedTenantCode || '')
  if (!token || !tenantCode) {
    throw createError({
      statusCode: 503,
      statusMessage: '嵌入审核未配置（请设置 EMBED_ACCESS_TOKEN 与 EMBED_TENANT_CODE）',
    })
  }
  return {
    apiBase: String(config.public.apiBase || 'http://localhost:8080').replace(/\/$/, ''),
    headers: {
      'X-Embed-Token': token,
      'X-Tenant-Code': tenantCode,
    } as Record<string, string>,
  }
}

function rethrowEmbedProxyError(e: unknown, fallbackStatus = 502): never {
  const err = e as {
    statusCode?: number
    status?: number
    data?: { code?: number; message?: string }
    response?: { _data?: { code?: number; message?: string } }
  }
  const body = err.data ?? err.response?._data
  const message = body?.message || (e instanceof Error ? e.message : '请求失败')
  const status = err.statusCode ?? err.status ?? fallbackStatus
  throw createError({ statusCode: status, statusMessage: message })
}

export async function proxyEmbedGet<T>(path: string, query?: Record<string, string | undefined>): Promise<T> {
  const { apiBase, headers } = getEmbedBackend()
  let res: { code: number; message: string; data: T }
  try {
    res = await $fetch(`${apiBase}${path}`, { headers, query })
  } catch (e: unknown) {
    rethrowEmbedProxyError(e)
  }
  if (res.code !== 0) {
    throw createError({ statusCode: 400, statusMessage: res.message || '请求失败' })
  }
  return res.data
}

export async function proxyEmbedPost<T>(path: string, body: unknown): Promise<T> {
  const { apiBase, headers } = getEmbedBackend()
  let res: { code: number; message: string; data: T }
  try {
    res = await $fetch(`${apiBase}${path}`, {
      method: 'POST',
      headers,
      body: body as Record<string, unknown> | BodyInit | null | undefined,
    })
  } catch (e: unknown) {
    const err = e as { data?: { code?: number; data?: T } }
    const data = err.data
    if (data && typeof data === 'object' && data.code === 0 && data.data) {
      return data.data
    }
    rethrowEmbedProxyError(e)
  }
  if (res.code !== 0) {
    throw createError({ statusCode: 400, statusMessage: res.message || '请求失败' })
  }
  return res.data
}
