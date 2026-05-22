import { getEmbedBackend } from '../../../utils/embedBackend'

export default defineEventHandler(async (event) => {
  const id = getRouterParam(event, 'id')
  if (!id) {
    throw createError({ statusCode: 400, statusMessage: '任务 ID 无效' })
  }

  const { apiBase, headers } = getEmbedBackend()
  const upstream = await fetch(`${apiBase}/api/embed/stream/${encodeURIComponent(id)}`, {
    headers,
  })

  if (!upstream.ok || !upstream.body) {
    throw createError({
      statusCode: upstream.status || 502,
      statusMessage: upstream.statusText || '流式审核连接失败',
    })
  }

  return new Response(upstream.body, {
    status: upstream.status,
    headers: {
      'Content-Type': 'text/event-stream',
      'Cache-Control': 'no-cache',
      Connection: 'keep-alive',
    },
  })
})
