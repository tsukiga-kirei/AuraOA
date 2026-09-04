import type { H3Event } from 'h3'
import { ofetch } from 'ofetch'

/** 嵌入页服务端代理：从 Cookie、请求头、查询参数或调用方已剥离的表单字段读取租户令牌。 */
export function getEmbedBackend(event: H3Event, bodyToken = '') {
  const config = useRuntimeConfig()
  const query = getQuery(event)
  const queryToken = typeof query.embed_token === 'string' ? query.embed_token : ''
  const token = String(
    getCookie(event, 'aura_embed_token')
      || getRequestHeader(event, 'x-embed-token')
      || queryToken
      || bodyToken
      || '',
  ).trim()
  if (!token) {
    throw createError({
      statusCode: 401,
      statusMessage: '缺少嵌入访问令牌，请确认 OA 父页面已配置 embed_token',
    })
  }
  const headers: Record<string, string> = {
    'X-Embed-Token': token,
  }
  const oaUserId = String(
    getRequestHeader(event, 'x-embed-oa-user-id')
      || query.oa_user_id
      || query.oa_current_user_id
      || '',
  ).trim()
  if (oaUserId) {
    headers['X-Embed-OA-User-ID'] = oaUserId
  }
  return {
    apiBase: String(config.internalApiBase).replace(/\/$/, ''),
    headers,
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

export async function proxyEmbedGet<T>(event: H3Event, path: string, query?: Record<string, string | undefined>): Promise<T> {
  const { apiBase, headers } = getEmbedBackend(event)
  let res: { code: number; message: string; data: T }
  try {
    res = await ofetch<{ code: number; message: string; data: T }>(`${apiBase}${path}`, { headers, query })
  } catch (e: unknown) {
    rethrowEmbedProxyError(e)
  }
  if (res.code !== 0) {
    throw createError({ statusCode: 400, statusMessage: res.message || '请求失败' })
  }
  return res.data
}

export async function proxyEmbedPost<T>(event: H3Event, path: string, body: unknown, bodyToken = ''): Promise<T> {
  const { apiBase, headers } = getEmbedBackend(event, bodyToken)
  let res: { code: number; message: string; data: T }
  try {
    res = await ofetch<{ code: number; message: string; data: T }>(`${apiBase}${path}`, {
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
