<script setup lang="ts">
import { ExpandOutlined } from '@ant-design/icons-vue'

const props = withDefaults(defineProps<{
  value?: string
  rows?: number
  placeholder?: string
  disabled?: boolean
  dialogTitle?: string
}>(), {
  value: '',
  rows: 6,
  placeholder: '',
  disabled: false,
  dialogTitle: '',
})

const emit = defineEmits<{
  'update:value': [value: string]
}>()

const { t } = useI18n()
const inlineTextareaRef = ref<any>(null)
const expandedTextareaRef = ref<any>(null)
const expanded = ref(false)
const selectionStart = ref(0)
const selectionEnd = ref(0)
const hasRememberedSelection = ref(false)

const content = computed({
  get: () => props.value,
  set: value => emit('update:value', value),
})

const resolveTextarea = (componentRef: any): HTMLTextAreaElement | null =>
  componentRef?.$el?.querySelector?.('textarea')
  || componentRef?.resizableTextArea?.textArea
  || null

const rememberSelection = (componentRef: any) => {
  const el = resolveTextarea(componentRef)
  if (!el) return
  selectionStart.value = el.selectionStart ?? props.value.length
  selectionEnd.value = el.selectionEnd ?? selectionStart.value
  hasRememberedSelection.value = true
}

const restoreSelection = async (target: 'inline' | 'expanded', start: number, end = start) => {
  await nextTick()
  const componentRef = target === 'expanded' ? expandedTextareaRef.value : inlineTextareaRef.value
  const el = resolveTextarea(componentRef)
  if (!el) return
  el.focus()
  el.setSelectionRange(start, end)
}

// insertAtCursor 在变量按钮夺走焦点后，仍使用最后一次有效选区插入。
const insertAtCursor = (variable: string) => {
  const currentValue = props.value || ''
  const fallbackPosition = currentValue.length
  const start = Math.min(
    hasRememberedSelection.value ? selectionStart.value : fallbackPosition,
    currentValue.length,
  )
  const end = Math.min(
    hasRememberedSelection.value ? selectionEnd.value : fallbackPosition,
    currentValue.length,
  )
  const nextValue = currentValue.slice(0, start) + variable + currentValue.slice(end)
  const nextPosition = start + variable.length

  emit('update:value', nextValue)
  selectionStart.value = nextPosition
  selectionEnd.value = nextPosition
  hasRememberedSelection.value = true
  void restoreSelection(expanded.value ? 'expanded' : 'inline', nextPosition)
}

const openExpandedEditor = () => {
  rememberSelection(inlineTextareaRef.value)
  expanded.value = true
  const start = hasRememberedSelection.value ? selectionStart.value : props.value.length
  const end = hasRememberedSelection.value ? selectionEnd.value : start
  void restoreSelection('expanded', start, end)
}

const closeExpandedEditor = () => {
  rememberSelection(expandedTextareaRef.value)
  expanded.value = false
  const start = hasRememberedSelection.value ? selectionStart.value : props.value.length
  const end = hasRememberedSelection.value ? selectionEnd.value : start
  void restoreSelection('inline', start, end)
}

defineExpose({ insertAtCursor })
</script>

<template>
  <div class="prompt-textarea">
    <a-textarea
      ref="inlineTextareaRef"
      v-model:value="content"
      :rows="rows"
      :placeholder="placeholder"
      :disabled="disabled"
      @input="rememberSelection(inlineTextareaRef)"
      @click="rememberSelection(inlineTextareaRef)"
      @keyup="rememberSelection(inlineTextareaRef)"
      @select="rememberSelection(inlineTextareaRef)"
      @blur="rememberSelection(inlineTextareaRef)"
    />
    <a-tooltip :title="t('admin.ruleConfig.expandPrompt')">
      <button
        type="button"
        class="prompt-textarea__expand"
        :aria-label="t('admin.ruleConfig.expandPrompt')"
        @mousedown.prevent
        @click="openExpandedEditor"
      >
        <ExpandOutlined />
      </button>
    </a-tooltip>

    <a-modal
      v-model:open="expanded"
      :title="dialogTitle || t('admin.ruleConfig.promptEditorTitle')"
      width="min(960px, calc(100vw - 32px))"
      :mask-closable="false"
      @cancel="closeExpandedEditor"
    >
      <a-textarea
        ref="expandedTextareaRef"
        v-model:value="content"
        class="prompt-textarea__expanded-input"
        :placeholder="placeholder"
        :readonly="disabled"
        @input="rememberSelection(expandedTextareaRef)"
        @click="rememberSelection(expandedTextareaRef)"
        @keyup="rememberSelection(expandedTextareaRef)"
        @select="rememberSelection(expandedTextareaRef)"
        @blur="rememberSelection(expandedTextareaRef)"
      />
      <template #footer>
        <a-button type="primary" @click="closeExpandedEditor">
          {{ t('common.close') }}
        </a-button>
      </template>
    </a-modal>
  </div>
</template>

<style scoped>
.prompt-textarea {
  position: relative;
}

.prompt-textarea :deep(textarea) {
  padding-right: 40px;
}

.prompt-textarea__expand {
  position: absolute;
  z-index: 2;
  top: 8px;
  right: 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  padding: 0;
  color: var(--color-text-tertiary);
  background: color-mix(in srgb, var(--color-bg-container) 88%, transparent);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.prompt-textarea__expand:hover {
  color: var(--color-primary);
  background: var(--color-primary-bg);
  border-color: var(--color-primary);
}

.prompt-textarea__expanded-input {
  min-height: min(62vh, 620px);
  resize: vertical;
}

.prompt-textarea__expanded-input:read-only {
  color: var(--color-text-primary);
  background: var(--color-bg-hover);
}
</style>
