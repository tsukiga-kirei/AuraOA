<script setup lang="ts">
import { computed, onBeforeUnmount, ref } from 'vue'
import { CheckOutlined, CopyOutlined } from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { writeClipboardText } from '~/utils/clipboard'

const props = defineProps<{ text?: string }>()
const { t } = useI18n()
const copied = ref(false)
const copying = ref(false)
const label = computed(() => t(copied.value ? 'common.copied' : 'common.copy'))
let resetTimer: ReturnType<typeof setTimeout> | undefined

async function copy() {
  if (!props.text || copying.value) return
  copying.value = true
  try {
    await writeClipboardText(props.text)
    copied.value = true
    message.success(t('common.copySuccess'))
    clearTimeout(resetTimer)
    resetTimer = setTimeout(() => { copied.value = false }, 2000)
  } catch {
    message.error(t('common.copyFailed'))
  } finally {
    copying.value = false
  }
}

onBeforeUnmount(() => clearTimeout(resetTimer))
</script>

<template>
  <div class="copyable-content">
    <div v-if="text" class="copyable-content-toolbar">
      <a-tooltip :title="label">
        <button
          type="button"
          class="copyable-content-button"
          :class="{ 'is-copied': copied }"
          :aria-label="label"
          :disabled="copying"
          @click.stop="copy"
        >
          <CheckOutlined v-if="copied" />
          <CopyOutlined v-else />
        </button>
      </a-tooltip>
    </div>
    <slot />
  </div>
</template>

<style scoped>
.copyable-content { min-width: 0; }
.copyable-content-toolbar { display: flex; align-items: center; margin-bottom: 4px; }
.copyable-content-button {
  display: inline-flex; align-items: center; justify-content: center;
  width: 28px; height: 28px; padding: 0; border: none;
  border-radius: var(--radius-sm); background: transparent;
  color: var(--color-text-tertiary); cursor: pointer;
}
.copyable-content-button:hover { color: var(--color-primary); background: var(--color-bg-hover); }
.copyable-content-button:focus-visible { outline: 2px solid var(--color-primary); outline-offset: 2px; }
.copyable-content-button.is-copied { color: var(--color-success); }
.copyable-content-button:disabled { cursor: wait; }
</style>
