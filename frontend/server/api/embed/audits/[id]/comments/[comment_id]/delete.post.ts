import { proxyEmbedPost } from '../../../../../../utils/embedBackend'

export default defineEventHandler(async event => {
  const id = getRouterParam(event, 'id') || ''
  const commentId = getRouterParam(event, 'comment_id') || ''
  return proxyEmbedPost(event, `/api/embed/audits/${encodeURIComponent(id)}/comments/${encodeURIComponent(commentId)}/delete`, {})
})
