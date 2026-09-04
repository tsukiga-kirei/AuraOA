/** 嵌入页客户端令牌与人员上下文（跨域 iframe 下 Cookie 可能被拦截，需同步走请求头） */
export const useEmbedAuth = () => {
  const token = useState<string>('aura_embed_token_client', () => '')
  const oaUserId = useState<string>('aura_embed_oa_user_id', () => '')

  function setEmbedToken(value: string) {
    token.value = String(value || '').trim()
  }

  function getEmbedToken() {
    return token.value
  }

  function setOAUserId(value: string) {
    oaUserId.value = String(value || '').trim()
  }

  function getOAUserId() {
    return oaUserId.value
  }

  function embedAuthHeaders(): Record<string, string> {
    const headers: Record<string, string> = {}
    if (token.value) headers['X-Embed-Token'] = token.value
    if (oaUserId.value) headers['X-Embed-OA-User-ID'] = oaUserId.value
    return headers
  }

  function appendEmbedTokenQuery(url: string): string {
    const t = token.value
    if (!t) return url
    const sep = url.includes('?') ? '&' : '?'
    let res = `${url}${sep}embed_token=${encodeURIComponent(t)}`
    if (oaUserId.value) {
      res += `&oa_user_id=${encodeURIComponent(oaUserId.value)}`
    }
    return res
  }

  return { setEmbedToken, getEmbedToken, setOAUserId, getOAUserId, embedAuthHeaders, appendEmbedTokenQuery }
}
