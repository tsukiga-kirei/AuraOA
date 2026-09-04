/**
 * useChatStream — 处理智能体单轮对话流式调用（SSE）
 * 包含事件监听：step_start, tool_call, tool_result, token, answer, done, error
 * 并维护当前响应消息的思考过程、工具调用列表与 Markdown 内容
 */

import type { ChatMessageItem, ChatToolExecution } from '~/types/chat'

export interface UseChatStreamOptions {
  onDone?: () => void
  onError?: (err: string) => void
}

export const useChatStream = (options: UseChatStreamOptions = {}) => {
  const { accessToken, effectiveTenantCode } = useAuth()
  const streaming = ref(false)
  const abortController = ref<AbortController | null>(null)

  const sendStreamMessage = async (
    sessionId: string,
    content: string,
    messageListRef: Ref<ChatMessageItem[]>,
  ) => {
    if (!content.trim() || streaming.value) return

    // 1. 推入用户消息
    const userMsgId = 'user-' + Date.now()
    const userMsg: ChatMessageItem = {
      id: userMsgId,
      session_id: sessionId,
      sender_type: 'user',
      content: content.trim(),
      created_at: new Date().toISOString(),
    }
    messageListRef.value.push(userMsg)

    // 2. 推入待响应的 assistant 消息占位
    const assistantMsgId = 'asst-' + Date.now()
    const assistantMsg: ChatMessageItem = {
      id: assistantMsgId,
      session_id: sessionId,
      sender_type: 'assistant',
      content: '',
      reasoning_content: '',
      tool_executions: [],
      created_at: new Date().toISOString(),
      streaming: true,
    }
    messageListRef.value.push(assistantMsg)

    streaming.value = true
    const controller = new AbortController()
    abortController.value = controller

    try {
      const response = await fetch(`/api/chat/sessions/${sessionId}/messages/stream`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${accessToken.value}`,
          'X-Tenant-Code': effectiveTenantCode.value || '',
        },
        body: JSON.stringify({ content: content.trim() }),
        signal: controller.signal,
      })

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }

      const reader = response.body?.getReader()
      if (!reader) throw new Error('流式读取器未就绪')

      const decoder = new TextDecoder('utf-8')
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        let currentEvent = ''
        for (const line of lines) {
          const trimmed = line.trim()
          if (!trimmed) continue
          if (trimmed.startsWith('event:')) {
            currentEvent = trimmed.slice(6).trim()
          } else if (trimmed.startsWith('data:')) {
            const dataStr = trimmed.slice(5).trim()
            try {
              const parsed = JSON.parse(dataStr)
              handleStreamEvent(currentEvent, parsed, assistantMsg)
            } catch (e) {
              console.warn('解析 SSE 数据行失败:', dataStr, e)
            }
          }
        }
      }
    } catch (err: any) {
      if (err.name === 'AbortError') {
        console.log('用户已中止对话生成')
      } else {
        console.error('对话生成异常:', err)
        assistantMsg.content += `\n\n> ⚠️ 生成异常: ${err.message || '网络中断'}`
        options.onError?.(err.message)
      }
    } finally {
      assistantMsg.streaming = false
      streaming.value = false
      abortController.value = null
      options.onDone?.()
    }
  }

  const handleStreamEvent = (event: string, data: any, msg: ChatMessageItem) => {
    switch (event) {
      case 'step_start':
        // 新一轮循环开始
        break
      case 'tool_call': {
        // 工具调用请求发起
        const execution: ChatToolExecution = {
          tool_code: data.tool_code,
          tool_name: data.tool_name || data.tool_code,
          arguments: data.arguments || {},
          result: null,
          execution_ms: 0,
        }
        if (!msg.tool_executions) msg.tool_executions = []
        msg.tool_executions.push(execution)
        break
      }
      case 'tool_result': {
        // 工具调用完成返回
        if (!msg.tool_executions) msg.tool_executions = []
        const found = msg.tool_executions.find(t => t.tool_code === data.tool_code && !t.result && !t.error)
        if (found) {
          found.result = data.result
          found.error = data.error
          found.execution_ms = data.execution_ms || 0
        } else {
          msg.tool_executions.push({
            tool_code: data.tool_code,
            tool_name: data.tool_code,
            arguments: {},
            result: data.result,
            error: data.error,
            execution_ms: data.execution_ms || 0,
          })
        }
        break
      }
      case 'token':
        // 正文增量打字
        if (data.delta) {
          msg.content += data.delta
        }
        break
      case 'answer':
        // 全量最终正文校准
        if (data.content) {
          msg.content = data.content
        }
        break
      case 'done':
        msg.token_cost = data.token_cost
        break
      case 'error':
        msg.content += `\n\n> ⚠️ 出错: ${data.message || '未知错误'}`
        break
    }
  }

  const stopStreaming = () => {
    if (abortController.value) {
      abortController.value.abort()
      abortController.value = null
      streaming.value = false
    }
  }

  return {
    streaming,
    sendStreamMessage,
    stopStreaming,
  }
}
