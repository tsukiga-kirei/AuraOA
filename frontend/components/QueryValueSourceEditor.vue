<script setup lang="ts">
import type { QueryBinding, QueryParameter } from '~/types/audit-data-query'

const props = defineProps<{
  modelValue?: QueryBinding
  type: QueryParameter['type']
  disabled?: boolean
  fieldOptions?: { label: string; value: string }[]
}>()
const emit = defineEmits<{ 'update:modelValue': [value: QueryBinding] }>()
const { t } = useI18n()
const binding = computed<QueryBinding>(() => props.modelValue || { source: 'field' })
function update(patch: Partial<QueryBinding>) {
  const next = patch.source ? { source: patch.source } : binding.value
  emit('update:modelValue', { ...next, ...patch })
}
</script>

<template>
  <div class="value-source-editor">
    <a-select
      :value="binding.source"
      :disabled="disabled"
      :aria-label="t('query.parameterSource')"
      @change="(value) => update({ source: value as QueryBinding['source'] })"
    >
      <a-select-option value="constant">{{ t('query.source.constant') }}</a-select-option>
      <a-select-option value="field">{{ t('query.source.field') }}</a-select-option>
      <a-select-option value="system">{{ t('query.source.system') }}</a-select-option>
    </a-select>
    <a-select
      v-if="binding.source === 'field' && fieldOptions !== undefined"
      :value="binding.field || undefined"
      :options="fieldOptions"
      :disabled="disabled"
      show-search
      allow-clear
      :placeholder="t('query.selectField')"
      @change="(value) => update({ field: value == null ? '' : String(value) })"
    />
    <p v-else-if="binding.source === 'field'" class="memory-help deferred-field">
      {{ t('query.builder.deferredField') }}
    </p>
    <a-select
      v-else-if="binding.source === 'system'"
      :value="binding.value || undefined"
      :disabled="disabled"
      :placeholder="t('query.selectSystemValue')"
      @change="(value) => update({ value: String(value) })"
    >
      <a-select-option
        v-for="value in ['process_id', 'current_date', 'year_start', 'month_start', 'current_year']"
        :key="value"
        :value="value"
        >{{ t('query.system.' + value) }}</a-select-option
      >
    </a-select>
    <a-input
      v-else
      :value="binding.value"
      :disabled="disabled"
      :placeholder="type === 'string_list' ? t('query.listConstant') : t('query.constant')"
      @update:value="(value) => update({ value: String(value ?? '') })"
    />
    <a-input-number
      v-if="type === 'date'"
      :value="binding.days_offset || 0"
      :min="-3660"
      :max="3660"
      :disabled="disabled"
      :addon-before="t('query.daysOffset')"
      @update:value="(value) => update({ days_offset: Number(value || 0) })"
    />
  </div>
</template>

<style scoped>
.value-source-editor {
  display: grid;
  grid-template-columns: 130px minmax(0, 1fr);
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.value-source-editor :deep(.ant-select) {
  min-width: 0;
}
.value-source-editor :deep(.ant-input-number-group-wrapper) {
  grid-column: 2;
}
.deferred-field {
  margin: 0;
}
@media (max-width: 600px) {
  .value-source-editor {
    grid-template-columns: 1fr;
  }
  .value-source-editor :deep(.ant-input-number-group-wrapper) {
    grid-column: 1;
  }
}
</style>
