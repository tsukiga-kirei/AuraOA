/**
 * useChatSession — 统一管理 AI 对话会话列表、切换、新建、重命名、删除及智能体列表
 * 对接后端路由 /api/chat/*
 */

import type { ChatSessionItem, ChatSessionDetail, EffectiveAgentItem } from '~/types/chat'

export const useChatSession = () => {
  const { authFetch } = useAuth()

  const sessions = ref<ChatSessionItem[]>([])
  const effectiveAgents = ref<EffectiveAgentItem[]>([])
  const currentSessionId = ref<string | null>(null)
  const currentDetail = ref<ChatSessionDetail | null>(null)
  const loading = ref(false)
  const detailLoading = ref(false)

  // 1. 获取当前用户可用的有效智能体列表
  const fetchEffectiveAgents = async () => {
    try {
      const data = await authFetch<EffectiveAgentItem[]>('/api/chat/agents')
      effectiveAgents.value = data || []
    } catch (err) {
      console.error('获取有效智能体失败', err)
      effectiveAgents.value = []
    }
  }

  // 2. 获取会话列表
  const fetchSessions = async (keyword?: string) => {
    loading.value = true
    try {
      const query = new URLSearchParams({ page: '1', page_size: '50' })
      if (keyword) query.set('keyword', keyword)
      const data = await authFetch<{ items: ChatSessionItem[]; total: number }>(`/api/chat/sessions?${query.toString()}`)
      sessions.value = data.items || []
    } catch (err) {
      console.error('获取会话列表失败', err)
      sessions.value = []
    } finally {
      loading.value = false
    }
  }

  // 3. 获取单会话详情及历史记录
  const selectSession = async (sessionId: string) => {
    currentSessionId.value = sessionId
    detailLoading.value = true
    try {
      const detail = await authFetch<ChatSessionDetail>(`/api/chat/sessions/${sessionId}`)
      currentDetail.value = detail
    } catch (err) {
      console.error('加载会话详情失败', err)
      currentDetail.value = null
    } finally {
      detailLoading.value = false
    }
  }

  // 4. 创建新会话
  const createSession = async (agentCode?: string, title?: string): Promise<ChatSessionItem | null> => {
    try {
      const created = await authFetch<ChatSessionItem>('/api/chat/sessions', {
        method: 'POST',
        body: {
          agent_code: agentCode,
          title: title || '新会话',
        },
      })
      sessions.value.unshift(created)
      await selectSession(created.id)
      return created
    } catch (err) {
      console.error('创建会话失败', err)
      return null
    }
  }

  // 5. 重命名会话
  const renameSession = async (sessionId: string, newTitle: string) => {
    try {
      await authFetch(`/api/chat/sessions/${sessionId}`, {
        method: 'PATCH',
        body: { title: newTitle },
      })
      const found = sessions.value.find(s => s.id === sessionId)
      if (found) found.title = newTitle
      if (currentDetail.value?.session.id === sessionId) {
        currentDetail.value.session.title = newTitle
      }
    } catch (err) {
      console.error('重命名会话失败', err)
      throw err
    }
  }

  // 6. 删除会话
  const deleteSession = async (sessionId: string) => {
    try {
      await authFetch(`/api/chat/sessions/${sessionId}`, {
        method: 'DELETE',
      })
      sessions.value = sessions.value.filter(s => s.id !== sessionId)
      if (currentSessionId.value === sessionId) {
        if (sessions.value.length > 0) {
          await selectSession(sessions.value[0].id)
        } else {
          currentSessionId.value = null
          currentDetail.value = null
        }
      }
    } catch (err) {
      console.error('删除会话失败', err)
      throw err
    }
  }

  // 会话按时间分组：今天、近7天、更早
  const groupedSessions = computed(() => {
    const today: ChatSessionItem[] = []
    const last7Days: ChatSessionItem[] = []
    const earlier: ChatSessionItem[] = []

    const now = new Date().getTime()
    const oneDay = 24 * 60 * 60 * 1000
    const sevenDays = 7 * oneDay

    sessions.value.forEach(s => {
      const timeStr = s.last_message_at || s.created_at
      const msgTime = new Date(timeStr).getTime()
      const diff = now - msgTime
      if (diff < oneDay) {
        today.push(s)
      } else if (diff < sevenDays) {
        last7Days.push(s)
      } else {
        earlier.push(s)
      }
    })

    return {
      today,
      last7Days,
      earlier,
    }
  })

  return {
    sessions,
    effectiveAgents,
    currentSessionId,
    currentDetail,
    loading,
    detailLoading,
    groupedSessions,
    fetchEffectiveAgents,
    fetchSessions,
    selectSession,
    createSession,
    renameSession,
    deleteSession,
  }
}
