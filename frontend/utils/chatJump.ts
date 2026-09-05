/**
 * 对话跳转线与平滑滚动导航辅助工具
 * 参考 global-investment-copilot 的 MessageJumpRail 实现
 */

export interface JumpTurnItem {
  id: string
  title: string
  preview: string
}

export function plainChatPreview(text: string, maxLength = 160): string {
  const cleaned = String(text || '')
    .replace(/```[\s\S]*?```/g, ' ')
    .replace(/[#>*_`~]+/g, '')
    .replace(/\s+/g, ' ')
    .trim()
  if (!cleaned) return ''
  if (cleaned.length <= maxLength) return cleaned
  return `${cleaned.slice(0, maxLength).trim()}…`
}

export function buildJumpTurns(messages: Array<{ id: string; role: string; content: string; streaming?: boolean }> = []): JumpTurnItem[] {
  const turns: JumpTurnItem[] = []
  for (let index = 0; index < messages.length; index += 1) {
    const message = messages[index]
    if (message.role !== 'user') continue
    let preview = ''
    let streaming = false
    for (let cursor = index + 1; cursor < messages.length; cursor += 1) {
      const next = messages[cursor]
      if (next.role === 'user') break
      if (next.role === 'assistant') {
        streaming = Boolean(next.streaming)
        preview = next.content || ''
        if (preview || streaming) break
      }
    }
    turns.push({
      id: message.id,
      title: String(message.content || '').trim() || '提问',
      preview: preview
        ? plainChatPreview(preview, 168)
        : (streaming ? '正在智能生成中…' : '尚未返回正文'),
    })
  }
  return turns
}
