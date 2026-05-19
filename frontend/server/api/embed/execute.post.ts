import { proxyEmbedPost } from '../../utils/embedBackend'

export default defineEventHandler(async (event) => {
  const body = await readBody(event)
  return await proxyEmbedPost('/api/embed/execute', body)
})
