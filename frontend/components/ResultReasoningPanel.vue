<script setup lang="ts">
import { DownOutlined, UpOutlined } from '@ant-design/icons-vue'
import { useI18n } from '~/composables/useI18n'
import { renderSafeMarkdown } from '~/utils/markdown'

const props = withDefaults(defineProps<{
  title: string
  text: string
  defaultExpanded?: boolean
  maxHeight?: string
  pageFlow?: boolean
}>(), {
  defaultExpanded: false,
  maxHeight: '360px',
  pageFlow: false,
})

const { t } = useI18n()
const expanded = ref(props.defaultExpanded)
const panelId = useId()
const panelRef = ref<HTMLElement | null>(null)

function toggle() {
  // 收起吸顶的长正文前，先保留标题在视口中的位置，避免跳到后续内容。
  if (props.pageFlow && expanded.value && panelRef.value && panelRef.value.getBoundingClientRect().top < 0) {
    panelRef.value.scrollIntoView({ block: 'start', behavior: 'instant' })
  }
  expanded.value = !expanded.value
}

const html = computed(() => renderSafeMarkdown(props.text || ''))
</script>

<template>
  <section ref="panelRef" class="result-reasoning-panel" :class="{ 'result-reasoning-panel--expanded': expanded, 'result-reasoning-panel--page-flow': pageFlow }">
    <button
      type="button"
      class="result-reasoning-panel__toggle"
      :aria-controls="panelId"
      :aria-expanded="expanded"
      @click="toggle"
    >
      <span class="result-reasoning-panel__title"><span v-if="$slots.icon" class="result-reasoning-panel__icon" aria-hidden="true"><slot name="icon" /></span>{{ title }}</span>
      <span class="result-reasoning-panel__action">
        {{ t(expanded ? 'resultDetails.collapse' : 'resultDetails.expand') }}
        <UpOutlined v-if="expanded" />
        <DownOutlined v-else />
      </span>
    </button>
    <Transition name="reasoning-reveal">
    <div
      v-if="expanded"
      :id="panelId"
      class="result-reasoning-panel__body"
      :style="{ maxHeight: pageFlow ? 'none' : maxHeight }"
    >
      <div class="markdown-body result-reasoning-panel__content" v-html="html" />
    </div>
    </Transition>
  </section>
</template>

<style scoped>
.result-reasoning-panel {
  margin-bottom: 24px;
  overflow: hidden;
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  background: var(--color-bg-card);
}

.result-reasoning-panel__toggle {
  display: flex;
  width: 100%;
  min-height: 52px;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 13px 16px;
  border: 0;
  background: var(--color-bg-card);
  color: var(--color-text-primary);
  cursor: pointer;
  font: inherit;
  text-align: left;
}

.result-reasoning-panel__toggle:hover {
  background: var(--color-bg-hover);
}

.result-reasoning-panel__toggle:focus-visible {
  position: relative;
  z-index: 1;
  outline: 2px solid var(--color-primary);
  outline-offset: -2px;
}

.result-reasoning-panel__title {
  font-size: 14px;
  font-weight: 600;
  line-height: 1.5;
}

.result-reasoning-panel__icon { color: var(--color-primary); font-size: 16px; display: inline-flex; }
.result-reasoning-panel--page-flow .result-reasoning-panel__title { display: inline-flex; align-items: center; gap: 9px; }
.result-reasoning-panel--page-flow { overflow: visible; margin-bottom: 12px; border-radius: 12px; }
.result-reasoning-panel--page-flow .result-reasoning-panel__toggle { position: sticky; top: 0; z-index: 4; border-radius: 12px; transition: background 160ms ease; }
.result-reasoning-panel--page-flow.result-reasoning-panel--expanded .result-reasoning-panel__toggle { border-radius: 12px 12px 0 0; box-shadow: 0 3px 8px rgb(15 23 42 / 3%); }
.result-reasoning-panel--page-flow .result-reasoning-panel__body { overflow: visible; border-radius: 0 0 12px 12px; }
.result-reasoning-panel--page-flow .reasoning-reveal-enter-active { transition: opacity 180ms ease, transform 180ms ease; }
.result-reasoning-panel--page-flow .reasoning-reveal-enter-from { opacity: 0; transform: translateY(-4px); }
@media (prefers-reduced-motion: reduce) { .reasoning-reveal-enter-active { transition: none; } }

.result-reasoning-panel__action {
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  gap: 6px;
  color: var(--color-text-tertiary);
  font-size: 12px;
  font-weight: 400;
}

.result-reasoning-panel--expanded .result-reasoning-panel__toggle {
  border-bottom: 1px solid var(--color-border-light);
}

.result-reasoning-panel__body {
  overflow: auto;
  padding: 16px 18px;
  background: color-mix(in srgb, var(--color-bg-page) 58%, var(--color-bg-card));
}

.result-reasoning-panel__content {
  color: var(--color-text-secondary);
  font-size: 13px;
  line-height: 1.75;
  overflow-wrap: anywhere;
}

.result-reasoning-panel__content :deep(> :first-child) {
  margin-top: 0;
}

.result-reasoning-panel__content :deep(> :last-child) {
  margin-bottom: 0;
}

.result-reasoning-panel__content :deep(h1),
.result-reasoning-panel__content :deep(h2),
.result-reasoning-panel__content :deep(h3),
.result-reasoning-panel__content :deep(h4),
.result-reasoning-panel__content :deep(h5),
.result-reasoning-panel__content :deep(h6) {
  margin: 14px 0 6px;
  color: var(--color-text-primary);
  font-weight: 600;
  line-height: 1.5;
}

.result-reasoning-panel__content :deep(h1) { font-size: 18px; }
.result-reasoning-panel__content :deep(h2) { font-size: 16px; }
.result-reasoning-panel__content :deep(h3),
.result-reasoning-panel__content :deep(h4),
.result-reasoning-panel__content :deep(h5),
.result-reasoning-panel__content :deep(h6) { font-size: 14px; }

.result-reasoning-panel__content :deep(p) {
  margin: 8px 0;
}

.result-reasoning-panel__content :deep(ul),
.result-reasoning-panel__content :deep(ol) {
  margin: 8px 0;
  padding-left: 22px;
}

.result-reasoning-panel__content :deep(li) {
  margin: 4px 0;
  padding-left: 2px;
}

.result-reasoning-panel__content :deep(pre) {
  overflow-x: auto;
  margin: 10px 0;
  padding: 12px;
  border-radius: var(--radius-sm);
  background: var(--color-bg-elevated);
}

.result-reasoning-panel__content :deep(code) {
  font-family: var(--font-mono);
}

.result-reasoning-panel__content :deep(blockquote) {
  margin: 10px 0;
  padding: 6px 12px;
  border-left: 3px solid var(--color-primary);
  background: var(--color-bg-elevated);
}

@media (max-width: 640px) {
  .result-reasoning-panel__toggle {
    min-height: 48px;
    padding: 12px 14px;
  }

  .result-reasoning-panel__body {
    padding: 14px;
  }
}
</style>
