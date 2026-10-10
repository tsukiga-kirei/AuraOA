<script setup lang="ts">
import { highlightQuerySql, formatQuerySql } from '~/utils/querySqlEditor'
const props = defineProps<{ modelValue: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: string]; run: [] }>()
const { t } = useI18n()
const input = ref<HTMLTextAreaElement>()
const backdrop = ref<HTMLElement>()
function syncScroll() {
  if (input.value && backdrop.value) {
    backdrop.value.scrollTop = input.value.scrollTop
    backdrop.value.scrollLeft = input.value.scrollLeft
  }
}
function insert(value: string) {
  const el = input.value
  const start = el?.selectionStart ?? props.modelValue.length
  const end = el?.selectionEnd ?? start
  emit('update:modelValue', props.modelValue.slice(0, start) + value + props.modelValue.slice(end))
  nextTick(() => {
    el?.focus()
    el?.setSelectionRange(start + value.length, start + value.length)
  })
}
function focusPosition(position: number) {
  const index = Math.max(0, position - 1)
  input.value?.focus()
  input.value?.setSelectionRange(index, index + 1)
}
defineExpose({ insert, focusPosition })
</script>
<template>
  <div class="sql-editor">
    <div class="editor-toolbar">
      <span>{{ t('query.sql') }}</span>
      <a-button size="small" @click="emit('update:modelValue', formatQuerySql(modelValue))">
        {{ t('query.format') }}
      </a-button>
    </div>
    <div class="editor-surface">
      <pre ref="backdrop" aria-hidden="true" v-html="highlightQuerySql(modelValue) + '\n'" />
      <textarea
        ref="input"
        :value="modelValue"
        :aria-label="t('query.sql')"
        spellcheck="false"
        @input="emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)"
        @scroll="syncScroll"
        @keydown.tab.prevent="insert('  ')"
        @keydown.ctrl.enter.prevent="emit('run')"
        @keydown.meta.enter.prevent="emit('run')"
      />
    </div>
    <div class="editor-hint">{{ t('query.sqlHint') }}</div>
  </div>
</template>
<style scoped>
.sql-editor {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md, 8px);
  overflow: hidden;
  background: var(--color-bg-input);
}
.editor-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: var(--color-bg-secondary);
  color: var(--color-text-secondary);
  border-bottom: 1px solid var(--color-border-light);
}
.editor-surface {
  position: relative;
  height: 300px;
}
.editor-surface pre,
.editor-surface textarea {
  position: absolute;
  inset: 0;
  margin: 0;
  padding: 14px;
  font:
    13px/1.7 ui-monospace,
    SFMono-Regular,
    Menlo,
    monospace;
  white-space: pre;
  overflow: auto;
  tab-size: 2;
  border: 0;
  border-radius: 0;
  box-sizing: border-box;
  width: 100%;
  height: 100%;
  letter-spacing: normal;
}
.editor-surface pre {
  pointer-events: none;
  color: var(--color-text-primary);
  background: transparent;
}
.editor-surface textarea {
  color: transparent;
  caret-color: var(--color-primary);
  background: transparent;
  resize: none;
  outline: none;
}
.editor-surface:focus-within {
  box-shadow: inset 0 0 0 2px var(--color-primary-ring);
}
.editor-surface :deep(.sql-keyword) {
  color: var(--color-primary);
  font-weight: 600;
}
.editor-surface :deep(.sql-literal) {
  color: var(--color-success);
}
.editor-surface :deep(.sql-parameter) {
  color: var(--color-warning);
}
.editor-hint {
  padding: 7px 12px;
  font-size: 12px;
  color: var(--color-text-tertiary);
  border-top: 1px solid var(--color-border-light);
}
</style>
