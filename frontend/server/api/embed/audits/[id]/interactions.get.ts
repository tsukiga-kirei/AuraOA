import { proxyEmbedGet } from '../../../../utils/embedBackend'

export default defineEventHandler(event => {
  const id = getRouterParam(event, 'id') || ''
  const query = getQuery(event)
  return proxyEmbedGet(event, `/api/embed/audits/${encodeURIComponent(id)}/interactions`, {
    page: String(query.page || 1), page_size: String(query.page_size || 20),
  })
})
