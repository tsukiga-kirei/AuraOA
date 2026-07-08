<script setup lang="ts">
import { marked } from 'marked'

const props = withDefaults(defineProps<{
  title?: string
  text?: string
  emptyText?: string
  maxHeight?: string
}>(), {
  title: '',
  text: '',
  emptyText: '等待模型输出',
  maxHeight: '260px',
})

const bodyRef = ref<HTMLElement | null>(null)
const pinnedToBottom = ref(true)
const isScrolling = ref(false)
let scrollTimer: ReturnType<typeof setTimeout> | null = null

const html = computed(() => {
  try {
    return marked.parse(props.text || '') as string
  } catch {
    return (props.text || '').replace(/[&<>"']/g, char => ({
      '&': '&amp;',
      '<': '&lt;',
      '>': '&gt;',
      '"': '&quot;',
      "'": '&#39;',
    }[char] || char))
  }
})

const scrollToBottom = () => {
  const el = bodyRef.value
  if (!el) return
  el.scrollTop = el.scrollHeight
}

const handleScroll = () => {
  const el = bodyRef.value
  if (!el) return
  pinnedToBottom.value = el.scrollHeight - el.scrollTop - el.clientHeight < 32
  isScrolling.value = true
  if (scrollTimer) clearTimeout(scrollTimer)
  scrollTimer = setTimeout(() => {
    isScrolling.value = false
  }, 900)
}

watch(
  () => props.text,
  async () => {
    if (!pinnedToBottom.value) return
    await nextTick()
    scrollToBottom()
  },
  { flush: 'post' },
)

onMounted(async () => {
  await nextTick()
  scrollToBottom()
})

onBeforeUnmount(() => {
  if (scrollTimer) clearTimeout(scrollTimer)
})
</script>

<template>
  <div class="ai-stream">
    <div v-if="title" class="ai-stream__header">{{ title }}</div>
    <div
      ref="bodyRef"
      class="ai-stream__body"
      :class="{ 'ai-stream__body--scrolling': isScrolling }"
      :style="{ maxHeight }"
      @scroll="handleScroll"
    >
      <div v-if="text" class="markdown-body ai-stream__content" v-html="html" />
      <div v-else class="ai-stream__empty">{{ emptyText }}</div>
    </div>
  </div>
</template>

<style scoped>
.ai-stream {
  width: 100%;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-card);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.55);
  overflow: hidden;
}

.ai-stream__header {
  padding: 9px 12px;
  border-bottom: 1px solid var(--color-border-light);
  color: var(--color-text-primary);
  font-size: 13px;
  font-weight: 600;
  line-height: 1.4;
  background: color-mix(in srgb, var(--color-bg-hover) 62%, var(--color-bg-card));
}

.ai-stream__body {
  min-height: 116px;
  overflow: auto;
  padding: 12px 14px;
  background: color-mix(in srgb, var(--color-bg-page) 70%, var(--color-bg-card));
  scrollbar-width: thin;
  scrollbar-color: transparent transparent;
  transition: scrollbar-color var(--transition-fast);
}

.ai-stream__body--scrolling {
  scrollbar-color: color-mix(in srgb, var(--color-text-tertiary) 72%, transparent) transparent;
}

.ai-stream__body::-webkit-scrollbar {
  width: 6px;
  height: 6px;
}

.ai-stream__body::-webkit-scrollbar-track {
  background: transparent;
}

.ai-stream__body::-webkit-scrollbar-thumb {
  background: transparent;
  border-radius: var(--radius-full);
}

.ai-stream__body--scrolling::-webkit-scrollbar-thumb {
  background: color-mix(in srgb, var(--color-text-tertiary) 72%, transparent);
}

.ai-stream__content {
  color: var(--color-text-secondary);
  font-size: 13px;
  line-height: 1.75;
}

.ai-stream__empty {
  display: flex;
  min-height: 92px;
  align-items: center;
  justify-content: center;
  color: var(--color-text-tertiary);
  font-size: 13px;
}
</style>
