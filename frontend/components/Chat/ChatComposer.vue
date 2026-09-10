<script setup lang="ts">
import {
  SendOutlined,
  PauseCircleOutlined,
} from '@ant-design/icons-vue'

const props = withDefaults(defineProps<{
  submitting: boolean
  disabled?: boolean
  placeholder?: string
  variant?: 'default' | 'hero'
}>(), {
  variant: 'default',
})

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
    if (isComposing.value || e.isComposing || props.submitting) return
    e.preventDefault()
    handleSubmit()
  }
}

const handleSubmit = () => {
  if (props.submitting) {
    emit('stop')
    return
  }
  if (props.disabled) return
  const text = inputContent.value.trim()
  if (!text) return
  emit('submit', text)
  inputContent.value = ''
  nextTick(resizeToContent)
}

const LINE_HEIGHT = 34
const MAX_HEIGHT = 168
const resizeToContent = () => {
  const el = textareaRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = `${Math.min(Math.max(el.scrollHeight, LINE_HEIGHT), MAX_HEIGHT)}px`
}

const handleInput = () => {
  resizeToContent()
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

onMounted(resizeToContent)

defineExpose({
  appendPrompt,
})
</script>

<template>
  <div class="chat-composer-wrapper" :class="{ 'chat-composer--hero': variant === 'hero' }">
    <div class="chat-composer-inner">
      <textarea
        ref="textareaRef"
        v-model="inputContent"
        rows="1"
        :disabled="disabled"
        :aria-label="placeholder || t('chat.composerPlaceholder')"
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
        :class="{ 'is-submitting': submitting, 'is-active': !!inputContent.trim() }"
        :disabled="disabled || (!submitting && !inputContent.trim())"
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
.chat-composer-wrapper { width:min(100%,780px); margin:auto; }
.chat-composer-inner { display:flex; align-items:flex-end; border:1px solid var(--color-border); background:var(--color-bg-page); border-radius:22px; padding:6px 8px 6px 16px; box-shadow:0 4px 20px rgba(0,0,0,.025); transition:border-color .2s, box-shadow .2s; }
.chat-composer-inner:focus-within { border-color:var(--color-primary); }
.chat-composer-textarea { flex:1; min-width:0; min-height:34px; border:0; outline:none; resize:none; background:none; color:var(--color-text-primary); font:inherit; font-size:14px; line-height:22px; max-height:168px; padding:6px 0; overflow-y:auto; }
.chat-composer-textarea::placeholder { color:var(--color-text-tertiary); }
.chat-composer-btn { display:grid; place-items:center; flex-shrink:0; width:34px; height:34px; border:0; border-radius:50%; background:var(--color-text-primary); color:var(--color-bg-card); margin-left:10px; font-size:16px; cursor:pointer; transition:all .2s cubic-bezier(0.2, 0.8, 0.2, 1); }
.chat-composer-btn:disabled { opacity:.25; cursor:default; }
.chat-composer-btn.is-active { background:var(--color-primary); color:white; }
.chat-composer-btn.is-submitting { background:var(--color-primary); color:white; }

/* Hero 初始化大卡片样式 */
.chat-composer--hero .chat-composer-inner {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border);
  border-radius: 20px;
  padding: 10px 12px 10px 20px;
  box-shadow: 0 8px 30px rgba(15, 23, 42, 0.06), 0 1px 3px rgba(15, 23, 42, 0.04);
}
.chat-composer--hero .chat-composer-inner:focus-within {
  border-color: var(--color-primary);
  box-shadow: 0 8px 30px var(--color-primary-ring);
}
.chat-composer--hero .chat-composer-textarea {
  font-size: 15px;
  line-height: 24px;
  min-height: 40px;
  padding: 8px 0;
}
.chat-composer--hero .chat-composer-btn {
  width: 40px;
  height: 40px;
  font-size: 18px;
}
</style>
