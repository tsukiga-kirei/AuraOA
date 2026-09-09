/** 对接 /api/chat/*，为全局导航和聊天画布共享会话状态。 */
import type { ChatSessionItem, ChatSessionDetail, EffectiveAgentItem } from '~/types/chat'

export const useChatSession = () => {
  const { authFetch, activeRole, currentUser, accessRevision } = useAuth()
  const { t } = useI18n()
  const state = useState('chat-workspace', () => ({
    identity: '', sessions: [] as ChatSessionItem[], effectiveAgents: [] as EffectiveAgentItem[],
    currentSessionId: null as string | null, currentDetail: null as ChatSessionDetail | null,
    loading: false, detailLoading: false, initialized: false, total: 0, page: 1, keyword: '', searchRequest: 0, error: '', selectedAgentCode: '',
  }))
  const identity = computed(() => JSON.stringify([currentUser.value?.username, activeRole.value]))
  watch(identity, key => {
    if (state.value.identity === key) return
    Object.assign(state.value, { identity: key, sessions: [], effectiveAgents: [], currentSessionId: null, currentDetail: null, initialized: false, loading: false, total: 0, page: 1, error: '', selectedAgentCode: '' })
  }, { immediate: true, flush: 'sync' })
  const { sessions, effectiveAgents, currentSessionId, currentDetail, loading, detailLoading, error, total, selectedAgentCode } = toRefs(state.value)
  const fetchEffectiveAgents = async () => {
    const key = identity.value
    const data = await authFetch<EffectiveAgentItem[]>('/api/chat/agents')
    if (key === identity.value) effectiveAgents.value = data || []
  }
  const fetchSessions = async (keyword = state.value.keyword, more = false) => {
    if (more && loading.value) return
    const key = identity.value
    const request = ++state.value.searchRequest
    const page = more ? state.value.page + 1 : 1
    loading.value = true; error.value = ''
    try {
      const query = new URLSearchParams({ page: String(page), page_size: '30', keyword })
      const data = await authFetch<{ items: ChatSessionItem[]; total: number }>(`/api/chat/sessions?${query}`)
      if (key !== identity.value || request !== state.value.searchRequest) return
      sessions.value = more ? [...sessions.value, ...(data.items || [])] : data.items || []
      total.value = data.total; state.value.page = page; state.value.keyword = keyword
    } catch (err: any) { if (key === identity.value && request === state.value.searchRequest) error.value = err.message }
    finally { if (key === identity.value && request === state.value.searchRequest) loading.value = false }
  }
  const initialize = async (force = false) => {
    if (!force && (state.value.initialized || loading.value)) return
    state.value.initialized = true
    const key = identity.value
    try { await Promise.all([fetchEffectiveAgents(), fetchSessions()]) }
    catch (err: any) { if (key === identity.value) { error.value = err.message; state.value.initialized = false } }
  }
  watch(identity, () => { initialize() })
  watch(accessRevision, (revision, previous) => {
    if (!previous || revision === previous) return
    void initialize(true)
  })
  const selectSession = async (id: string) => {
    const key = identity.value
    currentSessionId.value = id; currentDetail.value = null; detailLoading.value = true; error.value = ''
    try {
      const detail = await authFetch<ChatSessionDetail>(`/api/chat/sessions/${id}`)
      if (key === identity.value && currentSessionId.value === id) { currentDetail.value = detail; selectedAgentCode.value = detail.session.agent_code }
    } catch (err: any) { if (currentSessionId.value === id && key === identity.value) error.value = err.message }
    finally { if (currentSessionId.value === id && key === identity.value) detailLoading.value = false }
  }
  const newConversation = (code = effectiveAgents.value[0]?.agent_code || '') => {
    currentSessionId.value = null; currentDetail.value = null; selectedAgentCode.value = code; error.value = ''
  }
  const createSession = async (agentCode = selectedAgentCode.value || effectiveAgents.value[0]?.agent_code, title?: string) => {
    if (!agentCode) { error.value = t('chat.noAgents'); return null }
    const key = identity.value
    try {
      const session = await authFetch<ChatSessionItem>('/api/chat/sessions', { method: 'POST', body: { agent_code: agentCode, title } })
      if (key !== identity.value) return null
      sessions.value.unshift(session); total.value++
      currentSessionId.value = session.id; currentDetail.value = { session, messages: [] }; selectedAgentCode.value = session.agent_code
      return session
    } catch (err: any) { error.value = err.message; return null }
  }
  const renameSession = async (id: string, title: string) => {
    await authFetch(`/api/chat/sessions/${id}`, { method: 'PATCH', body: { title } })
    const session = sessions.value.find(item => item.id === id)
    if (session) session.title = title
    if (currentDetail.value?.session.id === id) currentDetail.value.session.title = title
  }
  const deleteSession = async (id: string) => {
    await authFetch(`/api/chat/sessions/${id}`, { method: 'DELETE' })
    sessions.value = sessions.value.filter(item => item.id !== id); total.value--
    if (currentSessionId.value === id) newConversation()
  }
  const updateMessageFeedback = async (
    messageId: string,
    feedback: 'like' | 'dislike' | null,
    feedbackComment?: string | null,
  ) => {
    await authFetch(`/api/chat/messages/${messageId}/feedback`, {
      method: 'POST',
      body: { feedback, feedback_comment: feedbackComment },
    })
    if (currentDetail.value?.messages) {
      const msg = currentDetail.value.messages.find(m => m.id === messageId)
      if (msg) {
        msg.feedback = feedback
        msg.feedback_at = feedback ? new Date().toISOString() : null
        msg.feedback_comment = feedback === 'dislike' ? (feedbackComment ?? null) : null
      }
    }
  }
  return { sessions, effectiveAgents, currentSessionId, currentDetail, loading, detailLoading, error, total, selectedAgentCode, initialize, newConversation, fetchEffectiveAgents, fetchSessions, selectSession, createSession, renameSession, deleteSession, updateMessageFeedback }
}
