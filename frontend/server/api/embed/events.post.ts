import { proxyEmbedPost } from '../../utils/embedBackend'

export default defineEventHandler(async (event) => {
  const rawBody = await readBody<Record<string, unknown> | string>(event)
  const body = typeof rawBody === 'string'
    ? Object.fromEntries(new URLSearchParams(rawBody))
    : { ...(rawBody || {}) }
  const bodyToken = String(body.embed_token || '').trim()
  delete body.embed_token
  const occurredAtMs = Number(body.occurred_at_ms || 0)
  body.occurred_at_ms = Number.isFinite(occurredAtMs) ? occurredAtMs : 0
  return await proxyEmbedPost(event, '/api/embed/events', body, bodyToken)
})
