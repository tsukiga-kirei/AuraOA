/** 嵌入页客户端令牌（跨域 iframe 下 Cookie 可能被拦截，需同步走请求头） */
export const useEmbedAuth = () => {
  const token = useState<string>('aura_embed_token_client', () => '')

  function setEmbedToken(value: string) {
    token.value = String(value || '').trim()
  }

  function getEmbedToken() {
    return token.value
  }

  function embedAuthHeaders(): Record<string, string> {
    const t = token.value
    return t ? { 'X-Embed-Token': t } : {}
  }

  function appendEmbedTokenQuery(url: string): string {
    const t = token.value
    if (!t) return url
    const sep = url.includes('?') ? '&' : '?'
    return `${url}${sep}embed_token=${encodeURIComponent(t)}`
  }

  return { setEmbedToken, getEmbedToken, embedAuthHeaders, appendEmbedTokenQuery }
}
