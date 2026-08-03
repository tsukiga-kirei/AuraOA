export interface EmbedRefreshEventRequest {
  process_id: string
  workflow_id: string
  oa_belong_user_id: string
  oa_current_user_id: string
  occurred_at_ms: number
  action: 'save_requested' | 'submit_requested'
  event_id?: string
}

export interface EmbedRefreshEventResponse {
  process_id: string
  action: string
  event_id: string
  scheduled_modules: string[]
  resolution_pending: boolean
}

/**
 * useEmbedEventApi — OA 隐藏 runner 事件接口。
 * POST /api/embed/events 只安排后台检查，不等待审核或总结完成。
 */
export const useEmbedEventApi = () => {
  const { embedAuthHeaders } = useEmbedAuth()

  async function scheduleEmbedRefresh(req: EmbedRefreshEventRequest): Promise<void> {
    const response = await fetch('/api/embed/events', {
      method: 'POST',
      credentials: 'include',
      keepalive: true,
      headers: {
        'Content-Type': 'application/json',
        ...embedAuthHeaders(),
      },
      body: JSON.stringify(req),
    })
    if (!response.ok) {
      throw new Error(`embed event rejected: ${response.status}`)
    }
  }

  return { scheduleEmbedRefresh }
}
