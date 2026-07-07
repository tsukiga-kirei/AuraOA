export const useEmbedSession = () => {
  async function setupEmbedSession(embedToken: string): Promise<void> {
    const token = String(embedToken || '').trim()
    if (!token) {
      throw new Error('缺少嵌入访问令牌')
    }
    await $fetch('/api/embed/session', {
      method: 'POST',
      body: { embed_token: token },
    })
  }

  return { setupEmbedSession }
}
