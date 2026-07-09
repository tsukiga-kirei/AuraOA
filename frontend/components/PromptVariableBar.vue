<script setup lang="ts">
import type { PromptVariableItem } from '~/composables/usePromptSystemVariables'

defineProps<{
  dataVariables: PromptVariableItem[]
  systemVariables: PromptVariableItem[]
}>()

const emit = defineEmits<{
  insert: [variable: string]
}>()

const { t } = useI18n()
</script>

<template>
  <div class="prompt-variable-bars">
    <div v-if="dataVariables.length" class="prompt-variables">
      <span class="prompt-variables-hint">{{ t('admin.ruleConfig.insertDataVariable') }}：</span>
      <a-tooltip v-for="v in dataVariables" :key="v.key" :title="v.desc">
        <button class="variable-btn variable-btn--data" @click="emit('insert', v.key)">{{ v.key }}</button>
      </a-tooltip>
    </div>
    <div v-if="systemVariables.length" class="prompt-variables">
      <span class="prompt-variables-hint prompt-variables-hint--system">{{ t('admin.ruleConfig.insertSystemVariable') }}：</span>
      <a-tooltip v-for="v in systemVariables" :key="v.key" :title="v.desc">
        <button class="variable-btn variable-btn--system" @click="emit('insert', v.key)">{{ v.key }}</button>
      </a-tooltip>
    </div>
  </div>
</template>

<style scoped>
.prompt-variable-bars {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 8px;
}

.prompt-variables {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}

.prompt-variables-hint {
  font-size: 12px;
  color: var(--color-text-tertiary);
}

.prompt-variables-hint--system {
  color: var(--color-warning, #d48806);
}

.variable-btn {
  font-size: 11px;
  font-family: monospace;
  padding: 2px 8px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.variable-btn--data {
  background: var(--color-bg-hover);
  color: var(--color-primary);
}

.variable-btn--data:hover {
  background: var(--color-primary-bg);
  border-color: var(--color-primary);
}

.variable-btn--system {
  background: var(--color-warning-bg, #fffbe6);
  color: var(--color-warning, #d48806);
  border-color: color-mix(in srgb, var(--color-warning, #d48806) 35%, var(--color-border));
}

.variable-btn--system:hover {
  background: color-mix(in srgb, var(--color-warning-bg, #fffbe6) 70%, var(--color-warning, #d48806));
  border-color: var(--color-warning, #d48806);
}
</style>
