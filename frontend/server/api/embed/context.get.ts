import { proxyEmbedGet } from '../../utils/embedBackend'

export default defineEventHandler(async (event) => {
  const query = getQuery(event)
  const processId = String(query.process_id || query.requestid || '').trim()
  if (!processId) {
    throw createError({ statusCode: 400, statusMessage: 'process_id 不能为空' })
  }
  const q: Record<string, string | undefined> = { process_id: processId }
  const oaUserId = String(query.oa_user_id || query.oa_current_user_id || '').trim()
  if (oaUserId) q.oa_user_id = oaUserId
  return await proxyEmbedGet(event, '/api/embed/context', q)
})
