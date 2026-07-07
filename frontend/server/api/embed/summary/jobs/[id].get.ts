import { proxyEmbedGet } from '../../../../utils/embedBackend'

export default defineEventHandler(async (event) => {
  const id = getRouterParam(event, 'id')
  if (!id) {
    throw createError({ statusCode: 400, statusMessage: '任务 ID 无效' })
  }
  return await proxyEmbedGet(event, `/api/embed/summary/jobs/${encodeURIComponent(id)}`)
})
