<script setup lang="ts">
import type { EmbedRefreshEventRequest } from '~/composables/useEmbedEventApi'
import { waitForParentEmbedContext } from '~/composables/useEmbedParent'

definePageMeta({ layout: 'embed' })

const { setupEmbedSession } = useEmbedSession()
const { scheduleEmbedRefresh } = useEmbedEventApi()

const processId = ref('')
const ready = ref(false)
type RunnerAction = {
  action: EmbedRefreshEventRequest['action']
  eventId: string
  processId: string
  workflowId: string
  oaBelongUserId: string
  oaCurrentUserId: string
  occurredAtMs: number
}

function createEventId() {
  if (globalThis.crypto?.randomUUID) return globalThis.crypto.randomUUID()
  return `oa-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

async function dispatch(item: RunnerAction) {
  if (!ready.value) return
  try {
    await scheduleEmbedRefresh({
      process_id: item.processId,
      workflow_id: item.workflowId,
      oa_belong_user_id: item.oaBelongUserId,
      oa_current_user_id: item.oaCurrentUserId,
      occurred_at_ms: item.occurredAtMs,
      action: item.action,
      event_id: item.eventId,
    })
  } catch {
    // runner 不能影响 OA 保存或提交；流程级定时扫描和可见 iframe 会继续兜底。
  } finally {
    window.parent.postMessage({
      type: 'aura-runner-event-ack',
      event_id: item.eventId,
    }, '*')
  }
}

function handleParentMessage(event: MessageEvent) {
  if (event.source !== window.parent) return
  const data = event.data || {}
  if (data.type === 'aura-oa-requestid') {
    const nextProcessId = String(data.requestid || '').trim()
    if (nextProcessId && nextProcessId !== processId.value) {
      processId.value = nextProcessId
    }
    return
  }
  if (data.type !== 'aura-oa-refresh-event') return
  const nextProcessId = String(data.requestid || '').trim()
  if (nextProcessId) processId.value = nextProcessId
  const action = String(data.action || '')
  if (action !== 'save_requested' && action !== 'submit_requested') return
  const eventId = String(data.event_id || createEventId())
  void dispatch({
    action: action as EmbedRefreshEventRequest['action'],
    eventId,
    processId: nextProcessId,
    workflowId: String(data.workflow_id || '').trim(),
    oaBelongUserId: String(data.oa_belong_user_id || '').trim(),
    oaCurrentUserId: String(data.oa_current_user_id || '').trim(),
    occurredAtMs: Number(data.occurred_at_ms || 0),
  })
}

onMounted(async () => {
  window.addEventListener('message', handleParentMessage)

  const parentCtx = await waitForParentEmbedContext({ requireRequestId: false })
  processId.value = parentCtx.requestId
  if (!parentCtx.embedToken) return
  try {
    await setupEmbedSession(parentCtx.embedToken)
  } catch {
    return
  }

  ready.value = true
  window.parent.postMessage({ type: 'aura-runner-ready', requestid: processId.value }, '*')
})

onBeforeUnmount(() => {
  window.removeEventListener('message', handleParentMessage)
})
</script>

<template>
  <div aria-hidden="true" />
</template>
