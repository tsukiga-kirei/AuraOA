<script setup lang="ts">
const props = defineProps<{ contentId: string }>()
const { t } = useI18n()
const railRef = ref<HTMLElement | null>(null)
const scrollRange = ref(0)
const scrollPosition = ref(0)
const railHeight = ref(0)
const thumbHeight = ref(0)
const dragging = ref(false)
let dragStartY = 0
let dragStartScroll = 0
let resizeObserver: ResizeObserver | null = null
let frame = 0

const thumbTravel = computed(() => Math.max(0, railHeight.value - thumbHeight.value))
const thumbTop = computed(() => scrollRange.value ? scrollPosition.value / scrollRange.value * thumbTravel.value : 0)

function measure() {
  const root = document.documentElement
  const viewport = root.clientHeight
  railHeight.value = Math.max(0, viewport - 8)
  scrollRange.value = Math.max(0, root.scrollHeight - viewport)
  scrollPosition.value = Math.max(0, Math.min(window.scrollY, scrollRange.value))
  thumbHeight.value = Math.min(railHeight.value, Math.max(28, railHeight.value * viewport / Math.max(viewport, root.scrollHeight)))
}

function scheduleMeasure() {
  if (frame) return
  frame = requestAnimationFrame(() => { frame = 0; measure() })
}

function startDrag(event: PointerEvent) {
  if (event.button !== 0) return
  measure()
  dragging.value = true
  dragStartY = event.clientY
  dragStartScroll = window.scrollY
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
}

function moveDrag(event: PointerEvent) {
  if (!dragging.value || !thumbTravel.value) return
  window.scrollTo({ top: dragStartScroll + (event.clientY - dragStartY) / thumbTravel.value * scrollRange.value, behavior: 'instant' })
}

function stopDrag() { dragging.value = false }

function jumpTo(event: PointerEvent) {
  if (event.button !== 0 || !railRef.value || !thumbTravel.value) return
  const offset = event.clientY - railRef.value.getBoundingClientRect().top - thumbHeight.value / 2
  window.scrollTo({ top: Math.max(0, Math.min(1, offset / thumbTravel.value)) * scrollRange.value, behavior: 'instant' })
}

function onKeydown(event: KeyboardEvent) {
  const page = document.documentElement.clientHeight * .9
  const offsets: Record<string, number> = { ArrowDown: 48, ArrowUp: -48, PageDown: page, PageUp: -page, ' ': event.shiftKey ? -page : page }
  let next: number
  if (event.key === 'Home') next = 0
  else if (event.key === 'End') next = scrollRange.value
  else if (event.key in offsets) next = window.scrollY + offsets[event.key]!
  else return
  event.preventDefault()
  window.scrollTo({ top: Math.max(0, Math.min(scrollRange.value, next)), behavior: 'instant' })
}

onMounted(() => {
  scheduleMeasure()
  window.addEventListener('scroll', scheduleMeasure, { passive: true })
  window.addEventListener('resize', scheduleMeasure, { passive: true })
  resizeObserver = new ResizeObserver(scheduleMeasure)
  const content = document.getElementById(props.contentId)
  if (content) resizeObserver.observe(content)
})

onBeforeUnmount(() => {
  window.removeEventListener('scroll', scheduleMeasure)
  window.removeEventListener('resize', scheduleMeasure)
  resizeObserver?.disconnect()
  if (frame) cancelAnimationFrame(frame)
})
</script>

<template>
  <div v-if="scrollRange > 0" ref="railRef" class="audit-scrollbar" :class="{ 'audit-scrollbar--dragging': dragging }" @pointerdown="jumpTo">
    <div
      class="audit-scrollbar__thumb"
      role="scrollbar"
      tabindex="0"
      aria-orientation="vertical"
      :aria-label="t('embed.auditPageScroll')"
      :aria-controls="contentId"
      :aria-valuemin="0"
      :aria-valuemax="Math.round(scrollRange)"
      :aria-valuenow="Math.round(scrollPosition)"
      :style="{ height: `${thumbHeight}px`, transform: `translateY(${thumbTop}px)` }"
      @pointerdown.stop.prevent="startDrag"
      @pointermove="moveDrag"
      @pointerup="stopDrag"
      @pointercancel="stopDrag"
      @lostpointercapture="stopDrag"
      @keydown="onKeydown"
    />
  </div>
</template>

<style scoped>
/* 覆盖在页面边缘，不保留原生滚动槽，避免 iframe 中出现白色竖条。 */
.audit-scrollbar { position: fixed; z-index: 20; top: 4px; right: 0; bottom: 4px; width: 12px; background: transparent; opacity: 0; pointer-events: none; transition: opacity 160ms ease; }
:global(html.embed-audit-document[data-audit-scrolling] .audit-scrollbar),
.audit-scrollbar--dragging,
.audit-scrollbar:focus-within { opacity: 1; pointer-events: auto; }
.audit-scrollbar__thumb { position: absolute; top: 0; right: 3px; width: 5px; border-radius: 999px; background: color-mix(in srgb, var(--color-text-tertiary) 65%, transparent); cursor: grab; touch-action: none; user-select: none; }
.audit-scrollbar__thumb:hover, .audit-scrollbar--dragging .audit-scrollbar__thumb { background: var(--color-text-tertiary); }
.audit-scrollbar--dragging .audit-scrollbar__thumb { cursor: grabbing; }
.audit-scrollbar__thumb:focus-visible { outline: 2px solid var(--color-primary); outline-offset: 1px; }
@media (prefers-reduced-motion: reduce) { .audit-scrollbar { transition: none; } }
</style>
