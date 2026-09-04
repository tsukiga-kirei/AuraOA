export const useEmbedSession = () => {
  const { setEmbedToken, setOAUserId, embedAuthHeaders } = useEmbedAuth()

  async function setupEmbedSession(embedToken: string, oaUserId?: string): Promise<void> {
    const token = String(embedToken || '').trim()
    if (!token) {
      throw new Error('缺少嵌入访问令牌')
    }
    setEmbedToken(token)
    if (oaUserId) {
      setOAUserId(oaUserId)
    }
    await $fetch('/api/embed/session', {
      method: 'POST',
      body: { embed_token: token },
      credentials: 'include',
      headers: embedAuthHeaders(),
    })
  }

  return { setupEmbedSession }
}
