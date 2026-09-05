/** 按完整 SSE 帧解析；事件名、UTF-8 字符和 CRLF 均可跨网络分片。 */
export async function readSSE(stream: ReadableStream<Uint8Array>, onEvent: (event: string, data: any) => void) {
  const reader = stream.getReader()
  const decoder = new TextDecoder()
  let buffer = ''
  let event = 'message'
  let data: string[] = []
  const line = (value: string) => {
    if (!value) {
      if (data.length) onEvent(event, JSON.parse(data.join('\n')))
      data = []; event = 'message'; return
    }
    if (value.startsWith('event:')) event = value.slice(6).trim()
    if (value.startsWith('data:')) data.push(value.slice(5).replace(/^ /, ''))
  }
  try {
    while (true) {
      const chunk = await reader.read()
      buffer += decoder.decode(chunk.value, { stream: !chunk.done })
      let index: number
      while ((index = buffer.indexOf('\n')) >= 0) {
        line(buffer.slice(0, index).replace(/\r$/, ''))
        buffer = buffer.slice(index + 1)
      }
      if (chunk.done) break
    }
    if (buffer) line(buffer.replace(/\r$/, ''))
    line('')
  } finally { reader.releaseLock() }
}
