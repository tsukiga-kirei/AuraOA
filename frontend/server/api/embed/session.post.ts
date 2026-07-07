export default defineEventHandler(async (event) => {
  const body = await readBody<{ embed_token?: string }>(event)
  const token = String(body?.embed_token || '').trim()
  if (!token) {
    throw createError({ statusCode: 400, statusMessage: '缺少嵌入访问令牌' })
  }

  setCookie(event, 'aura_embed_token', token, {
    httpOnly: true,
    sameSite: 'lax',
    secure: process.env.NODE_ENV === 'production',
    path: '/',
    maxAge: 12 * 60 * 60,
  })

  return { ok: true }
})
