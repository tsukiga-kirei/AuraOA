import { proxyEmbedGet } from '../../../utils/embedBackend'

export default defineEventHandler(async (event) => {
  const query = getQuery(event)
  const processId = String(query.process_id || query.requestid || '').trim()
  if (!processId) {
    throw createError({ statusCode: 400, statusMessage: 'process_id 不能为空' })
  }
  return await proxyEmbedGet(event, '/api/embed/summary/context', { process_id: processId })
})
