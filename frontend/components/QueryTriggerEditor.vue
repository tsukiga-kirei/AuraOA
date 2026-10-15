<script setup lang="ts">
import type { QueryFieldOption, QueryTrigger } from '~/types/audit-data-query'
import { queryTriggerType, queryTriggerOperators, queryTriggerNeedsValue, queryTriggerOperatorKey } from '~/utils/queryTrigger'

const props = defineProps<{
  modelValue?: QueryTrigger
  fieldOptions: QueryFieldOption[]
  disabled?: boolean
}>()
const emit = defineEmits<{ 'update:modelValue': [value: QueryTrigger | undefined] }>()
const { t } = useI18n()
const valueTypes: QueryTrigger['value_type'][] = ['text', 'number', 'date', 'choice', 'file']
const operators = computed(() => props.modelValue ? queryTriggerOperators(props.modelValue.value_type) : [])
const detailHint = computed(() => {
  const trigger = props.modelValue
  if (!trigger?.field || trigger.field.toLowerCase().startsWith('main:')) return ''
  if (trigger.operator === 'is_empty') return 'query.triggerDetailEmpty'
  return ['ne', 'not_contains'].includes(trigger.operator) ? 'query.triggerDetailNegative' : 'query.triggerDetailAny'
})
function toggle(enabled: boolean) {
  emit('update:modelValue', enabled ? { field: '', value_type: 'text', operator: 'eq', value: '' } : undefined)
}
function update(patch: Partial<QueryTrigger>) {
  if (props.modelValue) emit('update:modelValue', { ...props.modelValue, ...patch })
}
function changeType(value_type: QueryTrigger['value_type']) {
  update({ value_type, operator: queryTriggerOperators(value_type)[0], value: '' })
}
function changeField(value: string | undefined) {
  const field = props.fieldOptions.find(option => option.value === value)
  const value_type = queryTriggerType(field)
  update({ field: value || '', field_label: field?.label || '', value_type, operator: queryTriggerOperators(value_type)[0], value: '' })
}
function changeOperator(operator: QueryTrigger['operator']) {
  update({ operator, value: ['is_empty', 'not_empty'].includes(operator) ? '' : props.modelValue?.value })
}
</script>

<template>
  <div class="query-trigger-editor">
    <div class="trigger-heading">
      <span>{{ t('query.triggerTitle') }}</span>
      <a-switch size="small" :checked="!!modelValue" :disabled="disabled"
        :aria-label="t('query.triggerTitle')" @change="value => toggle(!!value)" />
    </div>
    <div v-if="modelValue" class="trigger-controls">
      <a-select :value="modelValue.field || undefined" :options="fieldOptions" show-search
        option-filter-prop="label" allow-clear :disabled="disabled"
        :placeholder="t('query.selectField')" :aria-label="t('query.triggerField')" @change="value => changeField(value == null ? undefined : String(value))" />
      <div class="trigger-comparison">
        <a-select :value="modelValue.value_type === 'boolean' ? undefined : modelValue.value_type" :disabled="disabled"
          :placeholder="t('query.triggerSelectType')"
          :aria-label="t('query.triggerValueType')" @change="value => changeType(value as QueryTrigger['value_type'])">
          <a-select-option v-for="type in valueTypes" :key="type" :value="type">{{ t('query.triggerType.' + type) }}</a-select-option>
        </a-select>
        <a-select :value="modelValue.operator" :disabled="disabled"
          :aria-label="t('query.triggerOperator')" @change="value => changeOperator(value as QueryTrigger['operator'])">
          <a-select-option v-for="operator in operators" :key="operator" :value="operator">{{ t(queryTriggerOperatorKey(modelValue, operator)) }}</a-select-option>
        </a-select>
        <template v-if="queryTriggerNeedsValue(modelValue)">
          <a-input-number v-if="modelValue.value_type === 'number'" :value="modelValue.value || undefined"
            string-mode :disabled="disabled" :aria-label="t('query.triggerValue')" :placeholder="t('query.triggerValue')"
            @update:value="value => update({ value: value == null ? '' : String(value) })" />
          <a-date-picker v-else-if="modelValue.value_type === 'date'" :value="modelValue.value || undefined"
            value-format="YYYY-MM-DD" :disabled="disabled" :aria-label="t('query.triggerValue')"
            :placeholder="t('query.triggerDate')" @update:value="value => update({ value: value ? String(value) : '' })" />
          <a-input v-else :value="modelValue.value" :disabled="disabled || modelValue.value_type === 'boolean'" :maxlength="1000"
            :aria-label="t('query.triggerValue')"
            :placeholder="t(modelValue.value_type === 'choice' ? 'query.triggerChoiceValue' : 'query.triggerValue')"
            @update:value="value => update({ value: String(value ?? '') })" />
        </template>
      </div>
      <small v-if="modelValue.value_type === 'boolean'" class="memory-help">{{ t('query.triggerLegacyHint') }}</small>
      <small v-else-if="modelValue.value_type === 'choice' && queryTriggerNeedsValue(modelValue)" class="memory-help">{{ t('query.triggerChoiceHint') }}</small>
      <small v-else class="memory-help">{{ t('query.triggerTypeHint') }}</small>
      <small v-if="detailHint" class="memory-help">{{ t(detailHint) }}</small>
    </div>
  </div>
</template>

<style scoped>
.query-trigger-editor {
  margin: 12px 0 16px;
  padding: 12px;
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  background: var(--color-bg-card);
}
.trigger-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  color: var(--color-text-secondary);
  font-size: 13px;
}
.trigger-controls {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-top: 12px;
}
.trigger-comparison {
  display: grid;
  grid-template-columns: 100px 120px minmax(0, 1fr);
  gap: 10px;
}
.trigger-comparison > * {
  min-width: 0;
  width: 100%;
}
.trigger-controls .memory-help {
  margin: -2px 0 0;
}
</style>
