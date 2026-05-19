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

export async function proxyEmbedGet<T>(path: string, query?: Record<string, string | undefined>): Promise<T> {
  const { apiBase, headers } = getEmbedBackend()
  const res = await $fetch<{ code: number; message: string; data: T }>(`${apiBase}${path}`, {
    headers,
    query,
  })
  if (res.code !== 0) {
    throw createError({ statusCode: 400, statusMessage: res.message || '请求失败' })
  }
  return res.data
}

export async function proxyEmbedPost<T>(path: string, body: unknown): Promise<T> {
  const { apiBase, headers } = getEmbedBackend()
  try {
    const res = await $fetch<{ code: number; message: string; data: T }>(`${apiBase}${path}`, {
      method: 'POST',
      headers,
      body,
    })
    if (res.code !== 0) {
      throw createError({ statusCode: 400, statusMessage: res.message || '请求失败' })
    }
    return res.data
  } catch (e: any) {
    // Go 异步审核返回 202，$fetch 可能走 response 分支
    const data = e?.data
    if (data && typeof data === 'object' && data.code === 0 && data.data) {
      return data.data as T
    }
    throw e
  }
}
