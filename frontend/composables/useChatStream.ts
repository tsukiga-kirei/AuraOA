/** 对接 POST /api/chat/sessions/:id/messages/stream；事件字段与后端一致。 */
import type { ChatMessageItem } from '~/types/chat'
import { readSSE } from '~/utils/sse'

/** 生成兼容安全与非安全上下文的 UUID */
function generateUUID(): string {
  if (typeof globalThis !== 'undefined' && globalThis.crypto?.randomUUID) {
    return globalThis.crypto.randomUUID()
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0
    const v = c === 'x' ? r : (r & 0x3) | 0x8
    return v.toString(16)
  })
}

export const useChatStream = (options: { onDone?: () => void; onError?: (error: string) => void } = {}) => {
  const { authStreamFetch } = useAuth()
  const { t } = useI18n()
  const { sessions, currentDetail } = useChatSession()
  const streaming = ref(false)
  let controller: AbortController | null = null
  const sendStreamMessage = async (sessionId: string, content: string, messages: Ref<ChatMessageItem[]>) => {
    if (!content.trim() || streaming.value) return
    messages.value.push({ id: generateUUID(), session_id: sessionId, role: 'user', content: content.trim(), created_at: new Date().toISOString() })
    const msg = reactive<ChatMessageItem>({ id: generateUUID(), session_id: sessionId, role: 'assistant', content: '', reasoning_content: '', tool_calls: [], streaming: true, status: 'running', created_at: new Date().toISOString() })
    messages.value.push(msg)
    streaming.value = true
    const startTime = Date.now()
    const activeController = new AbortController()
    controller = activeController
    let finished = false
    try {
      const response = await authStreamFetch(`/api/chat/sessions/${sessionId}/messages/stream`, {
        method: 'POST', headers: { 'Content-Type': 'application/json', Accept: 'text/event-stream' },
        body: JSON.stringify({ content: content.trim() }), signal: activeController.signal,
      })
      if (!response.body) throw new Error(t('chat.connectionLost'))
      await readSSE(response.body, (event, data) => {
        if (event === 'reset') { msg.content = data.content || ''; msg.reasoning_content = data.reasoning_content || '' }
        if (event === 'delta') msg.content += data.content || ''
        if (event === 'reasoning') msg.reasoning_content += data.content || ''
        if (event === 'tool_start') {
          msg.tool_calls!.push({
            tool_code: data?.tool_code || '',
            tool_call_id: data?.tool_call_id || generateUUID(),
            ui_kind: data?.ui_kind || '',
            status: data?.status || 'running',
            arguments: data?.arguments,
            payload: data?.payload,
            thought: data?.thought,
          })
        }
        if (event === 'tool_result') {
          const tool = msg.tool_calls!.find(item => item.tool_call_id === data?.tool_call_id)
          if (tool) Object.assign(tool, data || {})
          else msg.tool_calls!.push({
            tool_code: data?.tool_code || '',
            tool_call_id: data?.tool_call_id || generateUUID(),
            ui_kind: data?.ui_kind || '',
            status: data?.status || 'success',
            arguments: data?.arguments,
            payload: data?.payload,
            thought: data?.thought,
          })
        }
        if (event === 'session' && data.title) {
          const session = sessions.value.find(item => item.id === sessionId)
          if (session) session.title = data.title
          if (currentDetail.value?.session.id === sessionId) currentDetail.value.session.title = data.title
        }
        if (event === 'done') { finished = true; msg.status = 'success'; msg.token_usage = data.token_usage; msg.duration_ms = data?.duration_ms || (Date.now() - startTime) }
        if (event === 'interrupted') { finished = true; msg.status = 'interrupted'; msg.duration_ms = Date.now() - startTime }
        if (event === 'error') { finished = true; msg.status = 'error'; msg.error = data.message || t('chat.connectionLost'); msg.duration_ms = Date.now() - startTime }
      })
      if (!finished) throw new Error(t('chat.connectionLost'))
    } catch (error: any) {
      if (activeController.signal.aborted) msg.status = 'interrupted'
      else { msg.status = 'error'; msg.error = error.message || t('chat.connectionLost'); options.onError?.(msg.error!) }
      if (!msg.duration_ms) msg.duration_ms = Date.now() - startTime
    } finally {
      if (!msg.duration_ms) msg.duration_ms = Date.now() - startTime
      msg.streaming = false
      msg.tool_calls?.forEach(tool => { if (tool.status === 'running') tool.status = 'error' })
      streaming.value = false
      if (controller === activeController) controller = null
      options.onDone?.()
    }
  }
  const stopStreaming = () => controller?.abort()
  onBeforeUnmount(stopStreaming)
  return { streaming, sendStreamMessage, stopStreaming }
}
