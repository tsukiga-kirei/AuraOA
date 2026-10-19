<script setup lang="ts">
import { LoadingOutlined } from '@ant-design/icons-vue'
import { useI18n } from '~/composables/useI18n'
import { renderSafeMarkdown } from '~/utils/markdown'

const props = withDefaults(defineProps<{
  title?: string
  text?: string
  emptyText?: string
  maxHeight?: string
  pageFlow?: boolean
  loading?: boolean
}>(), {
  title: '',
  text: '',
  emptyText: '等待模型输出',
  maxHeight: '260px',
  pageFlow: false,
  loading: false,
})

const { t } = useI18n()
const bodyRef = ref<HTMLElement | null>(null)
const pinnedToBottom = ref(true)
const isScrolling = ref(false)
let scrollTimer: ReturnType<typeof setTimeout> | null = null

const html = computed(() => {
  try {
    return renderSafeMarkdown(props.text || '')
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
  if (props.pageFlow) return
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
  <div class="ai-stream" :class="{ 'ai-stream--page-flow': pageFlow }">
    <div v-if="title" class="ai-stream__header">
      <span class="ai-stream__title">
        <span v-if="$slots.icon" class="ai-stream__icon" aria-hidden="true"><slot name="icon" /></span>
        {{ title }}
      </span>
      <span v-if="loading" class="ai-stream__status">
        <LoadingOutlined spin />
        <span>{{ t('dashboard.aiAnalyzingSub', '实时分析中') }}</span>
      </span>
    </div>
    <div
      ref="bodyRef"
      class="ai-stream__body"
      :class="{ 'ai-stream__body--scrolling': isScrolling }"
      :style="{ maxHeight: pageFlow ? 'none' : maxHeight }"
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
  margin-bottom: 14px;
  overflow: hidden;
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  background: var(--color-bg-card);
}

.ai-stream--page-flow {
  overflow: visible;
  margin-bottom: 10px;
  border-radius: 12px;
}

.ai-stream__header {
  display: flex;
  width: 100%;
  min-height: 44px;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 14px;
  border-bottom: 1px solid var(--color-border-light);
  background: var(--color-bg-card);
  color: var(--color-text-primary);
  box-sizing: border-box;
}

.ai-stream--page-flow .ai-stream__header {
  position: sticky;
  top: 0;
  z-index: 4;
  border-radius: 12px 12px 0 0;
  box-shadow: 0 3px 8px rgb(15 23 42 / 3%);
}

.ai-stream__title {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 13.5px;
  font-weight: 600;
  line-height: 1.45;
  color: var(--color-text-primary);
}

.ai-stream__icon {
  display: inline-flex;
  color: var(--color-primary);
  font-size: 16px;
}

.ai-stream__status {
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  gap: 6px;
  color: var(--color-primary);
  font-size: 12px;
  font-weight: 400;
}

.ai-stream__body {
  min-height: 80px;
  overflow: auto;
  padding: 12px 14px;
  background: color-mix(in srgb, var(--color-bg-page) 58%, var(--color-bg-card));
  scrollbar-width: thin;
  scrollbar-color: transparent transparent;
  transition: scrollbar-color var(--transition-fast);
  box-sizing: border-box;
}

.ai-stream--page-flow .ai-stream__body {
  overflow: visible;
  border-radius: 0 0 12px 12px;
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
  line-height: 1.6;
  overflow-wrap: anywhere;
}

.ai-stream__content :deep(> :first-child) {
  margin-top: 0;
}

.ai-stream__content :deep(> :last-child) {
  margin-bottom: 0;
}

.ai-stream__content :deep(h1),
.ai-stream__content :deep(h2),
.ai-stream__content :deep(h3),
.ai-stream__content :deep(h4),
.ai-stream__content :deep(h5),
.ai-stream__content :deep(h6) {
  margin: 10px 0 4px;
  color: var(--color-text-primary);
  font-weight: 600;
  line-height: 1.45;
}

.ai-stream__content :deep(h1) { font-size: 16px; }
.ai-stream__content :deep(h2) { font-size: 15px; }
.ai-stream__content :deep(h3),
.ai-stream__content :deep(h4),
.ai-stream__content :deep(h5),
.ai-stream__content :deep(h6) { font-size: 13.5px; }

.ai-stream__content :deep(p) {
  margin: 4px 0;
}

.ai-stream__content :deep(ul),
.ai-stream__content :deep(ol) {
  margin: 4px 0;
  padding-left: 20px;
}

.ai-stream__content :deep(li) {
  margin: 2px 0;
  padding-left: 2px;
}

.ai-stream__content :deep(pre) {
  overflow-x: auto;
  margin: 6px 0;
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
}

.ai-stream__content :deep(code) {
  font-family: var(--font-mono);
  font-size: 12px;
}

.ai-stream__content :deep(blockquote) {
  margin: 6px 0;
  padding: 4px 10px;
  border-left: 3px solid var(--color-primary);
  background: var(--color-bg-elevated);
}

.ai-stream__content :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 6px 0;
}

.ai-stream__content :deep(th),
.ai-stream__content :deep(td) {
  border: 1px solid var(--color-border-light);
  padding: 4px 8px;
  font-size: 12px;
}

.ai-stream__content :deep(th) {
  background: var(--color-bg-elevated);
  font-weight: 600;
}

.ai-stream__empty {
  display: flex;
  min-height: 80px;
  align-items: center;
  justify-content: center;
  color: var(--color-text-tertiary);
  font-size: 13px;
}

@media (max-width: 640px) {
  .ai-stream__header {
    min-height: 48px;
    padding: 12px 14px;
  }

  .ai-stream__body {
    padding: 14px;
  }
}
</style>
