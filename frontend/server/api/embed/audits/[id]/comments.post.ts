import { proxyEmbedPost } from '../../../../utils/embedBackend'

export default defineEventHandler(async event => {
  const id = getRouterParam(event, 'id') || ''
  return proxyEmbedPost(event, `/api/embed/audits/${encodeURIComponent(id)}/comments`, await readBody(event))
})
