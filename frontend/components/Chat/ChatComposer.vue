<script setup lang="ts">
import {
  SendOutlined,
  PauseCircleOutlined,
} from '@ant-design/icons-vue'

const props = defineProps<{
  submitting: boolean
  placeholder?: string
}>()

const emit = defineEmits<{
  (e: 'submit', content: string): void
  (e: 'stop'): void
}>()

const { t } = useI18n()
const inputContent = ref('')
const textareaRef = ref<HTMLTextAreaElement | null>(null)
const isComposing = ref(false)

const handleKeyDown = (e: KeyboardEvent) => {
  if (e.key === 'Enter' && !e.shiftKey) {
    if (isComposing.value) return
    e.preventDefault()
    handleSubmit()
  }
}

const handleSubmit = () => {
  if (props.submitting) {
    emit('stop')
    return
  }
  const text = inputContent.value.trim()
  if (!text) return
  emit('submit', text)
  inputContent.value = ''
  if (textareaRef.value) {
    textareaRef.value.style.height = 'auto'
  }
}

const handleInput = () => {
  if (!textareaRef.value) return
  textareaRef.value.style.height = 'auto'
  textareaRef.value.style.height = `${Math.min(textareaRef.value.scrollHeight, 140)}px`
}

const appendPrompt = (prompt: string) => {
  if (inputContent.value) {
    inputContent.value += ' ' + prompt
  } else {
    inputContent.value = prompt
  }
  nextTick(() => {
    handleInput()
    textareaRef.value?.focus()
  })
}

defineExpose({
  appendPrompt,
})
</script>

<template>
  <div class="chat-composer-wrapper">
    <div class="chat-composer-inner">
      <textarea
        ref="textareaRef"
        v-model="inputContent"
        rows="1"
        class="chat-composer-textarea"
        :placeholder="placeholder || t('chat.composerPlaceholder')"
        @compositionstart="isComposing = true"
        @compositionend="isComposing = false"
        @keydown="handleKeyDown"
        @input="handleInput"
      />
      <button
        type="button"
        class="chat-composer-btn"
        :class="{ 'is-submitting': submitting }"
        :disabled="!submitting && !inputContent.trim()"
        :title="submitting ? t('chat.stopGenerating') : t('chat.sendQuestion')"
        :aria-label="submitting ? t('chat.stopGenerating') : t('chat.sendQuestion')"
        @click="handleSubmit"
      >
        <PauseCircleOutlined v-if="submitting" />
        <SendOutlined v-else />
      </button>
    </div>
  </div>
</template>

<style scoped>
.chat-composer-wrapper {
  position: relative;
  width: min(100%, 820px);
  margin: 0 auto;
  padding: 12px 16px 20px;
}
.chat-composer-inner {
  display: flex;
  align-items: flex-end;
  background: #ffffff;
  border: 1px solid #d9d9d9;
  border-radius: 12px;
  padding: 8px 12px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.06);
  transition: border-color 0.2s, box-shadow 0.2s;
}
.chat-composer-inner:focus-within {
  border-color: #1890ff;
  box-shadow: 0 4px 16px rgba(24, 144, 255, 0.12);
}
.chat-composer-textarea {
  flex: 1;
  border: none;
  outline: none;
  resize: none;
  background: transparent;
  font-size: 14px;
  line-height: 1.5;
  color: #1f2937;
  max-height: 140px;
  padding: 4px 0;
}
.chat-composer-btn {
  width: 34px;
  height: 34px;
  display: grid;
  place-items: center;
  border-radius: 8px;
  border: none;
  background: #1890ff;
  color: #ffffff;
  cursor: pointer;
  margin-left: 8px;
  font-size: 16px;
  transition: background-color 0.2s, opacity 0.2s;
}
.chat-composer-btn:hover:not(:disabled) {
  background: #40a9ff;
}
.chat-composer-btn:disabled {
  background: #d9d9d9;
  cursor: not-allowed;
  opacity: 0.6;
}
.chat-composer-btn.is-submitting {
  background: #ff4d4f;
}
</style>
