<script setup lang="ts">
import type { EmbedRefreshEventRequest } from '~/composables/useEmbedEventApi'
import { waitForParentEmbedContext } from '~/composables/useEmbedParent'

definePageMeta({ layout: 'embed' })

const { setupEmbedSession } = useEmbedSession()
const { scheduleEmbedRefresh } = useEmbedEventApi()

const processId = ref('')
const ready = ref(false)
const pendingActions: Array<{
  action: EmbedRefreshEventRequest['action']
  eventId: string
}> = []

function createEventId() {
  if (globalThis.crypto?.randomUUID) return globalThis.crypto.randomUUID()
  return `oa-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

async function dispatch(action: EmbedRefreshEventRequest['action'], eventId = createEventId()) {
  if (!ready.value || !processId.value) {
    pendingActions.push({ action, eventId })
    return
  }
  try {
    await scheduleEmbedRefresh({
      process_id: processId.value,
      action,
      event_id: eventId,
    })
  } catch {
    // runner 不能影响 OA 保存或提交；流程级定时扫描和可见 iframe 会继续兜底。
  } finally {
    window.parent.postMessage({
      type: 'aura-runner-event-ack',
      event_id: eventId,
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
  const action = String(data.action || 'save_or_submit') as EmbedRefreshEventRequest['action']
  const eventId = String(data.event_id || createEventId())
  void dispatch(action, eventId)
}

onMounted(async () => {
  window.addEventListener('message', handleParentMessage)
  window.parent.postMessage({ type: 'aura-runner-ready' }, '*')

  const parentCtx = await waitForParentEmbedContext()
  processId.value = parentCtx.requestId
  if (!parentCtx.embedToken || !processId.value) return
  try {
    await setupEmbedSession(parentCtx.embedToken)
  } catch {
    return
  }

  ready.value = true
  const queued = pendingActions.splice(0)
  for (const item of queued) {
    await dispatch(item.action, item.eventId)
  }
  window.parent.postMessage({ type: 'aura-runner-ready', requestid: processId.value }, '*')
})

onBeforeUnmount(() => {
  window.removeEventListener('message', handleParentMessage)
})
</script>

<template>
  <div aria-hidden="true" />
</template>
