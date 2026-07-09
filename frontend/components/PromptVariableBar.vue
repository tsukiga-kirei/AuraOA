<script setup lang="ts">
import { CheckOutlined } from '@ant-design/icons-vue'
import type { PromptVariableItem } from '~/composables/usePromptSystemVariables'

const props = withDefaults(defineProps<{
  dataVariables: PromptVariableItem[]
  systemVariables: PromptVariableItem[]
  disableDataVariables?: boolean
  dataVariableMode?: 'insert' | 'toggle'
  selectedDataVariables?: string[]
}>(), {
  disableDataVariables: false,
  dataVariableMode: 'insert',
  selectedDataVariables: () => [],
})

const emit = defineEmits<{
  insert: [variable: string]
  'toggle-data-variable': [variable: string]
}>()

const { t } = useI18n()

const isDataVariableSelected = (key: string) => props.selectedDataVariables.includes(key)

const handleDataVariableClick = (key: string) => {
  if (props.disableDataVariables) return
  if (props.dataVariableMode === 'toggle') {
    emit('toggle-data-variable', key)
    return
  }
  emit('insert', key)
}
</script>

<template>
  <div class="prompt-variable-bars">
    <div v-if="dataVariables.length" class="prompt-variables">
      <span
        class="prompt-variables-hint"
        :class="{ 'prompt-variables-hint--disabled': disableDataVariables }"
      >
        {{ disableDataVariables
          ? t('admin.ruleConfig.insertDataVariableDisabled')
          : (dataVariableMode === 'toggle'
            ? t('admin.ruleConfig.selectDataVariable')
            : t('admin.ruleConfig.insertDataVariable')) }}：
      </span>
      <a-tooltip
        v-for="v in dataVariables"
        :key="v.key"
        :title="disableDataVariables ? t('admin.ruleConfig.insertDataVariableDisabledHint') : v.desc"
      >
        <button
          class="variable-btn variable-btn--data"
          :class="{
            'variable-btn--selected': dataVariableMode === 'toggle' && isDataVariableSelected(v.key),
            'variable-btn--disabled': disableDataVariables,
          }"
          :disabled="disableDataVariables"
          @click="handleDataVariableClick(v.key)"
        >
          <CheckOutlined v-if="dataVariableMode === 'toggle' && isDataVariableSelected(v.key)" class="variable-btn-check" />
          {{ v.key }}
        </button>
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

.prompt-variables-hint--disabled {
  color: var(--color-text-quaternary);
}

.variable-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  font-family: monospace;
  padding: 2px 8px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.variable-btn-check {
  font-size: 10px;
}

.variable-btn--data {
  background: var(--color-bg-hover);
  color: var(--color-primary);
}

.variable-btn--data:hover:not(:disabled) {
  background: var(--color-primary-bg);
  border-color: var(--color-primary);
}

.variable-btn--data.variable-btn--selected {
  background: var(--color-primary-bg);
  border-color: var(--color-primary);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--color-primary) 25%, transparent);
}

.variable-btn--disabled,
.variable-btn--data:disabled {
  opacity: 0.45;
  cursor: not-allowed;
  background: var(--color-bg-hover);
  border-color: var(--color-border-light);
  color: var(--color-text-quaternary);
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
