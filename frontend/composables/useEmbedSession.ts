export const useEmbedSession = () => {
  const { setEmbedToken, embedAuthHeaders } = useEmbedAuth()

  async function setupEmbedSession(embedToken: string): Promise<void> {
    const token = String(embedToken || '').trim()
    if (!token) {
      throw new Error('缺少嵌入访问令牌')
    }
    setEmbedToken(token)
    await $fetch('/api/embed/session', {
      method: 'POST',
      body: { embed_token: token },
      credentials: 'include',
      headers: embedAuthHeaders(),
    })
  }

  return { setupEmbedSession }
}
