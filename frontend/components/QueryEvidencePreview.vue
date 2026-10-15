<script setup lang="ts">
import type { QueryEvidence } from '~/types/audit-data-query'
import { message } from 'ant-design-vue'
import { appDayjs } from '~/utils/appTime'
import { queryTriggerOperatorKey, queryTriggerNeedsValue } from '~/utils/queryTrigger'
const props = defineProps<{ evidence: QueryEvidence }>()
const { t } = useI18n()
const triggerSummary = computed(() => {
  const trigger = props.evidence.trigger
  if (!trigger) return ''
  const value = trigger.value_type === 'boolean'
    ? t(trigger.value === 'true' ? 'query.triggerYes' : 'query.triggerNo')
    : trigger.value || ''
  return [trigger.field_label || trigger.field, t(queryTriggerOperatorKey(trigger)),
    queryTriggerNeedsValue(trigger) ? value : ''].filter(Boolean).join(' ')
})
function formatTime(value: string, dateOnly = false) {
  const date = appDayjs(value)
  return date.isValid()
    ? date.format(t(dateOnly ? 'query.dateFormat' : 'query.dateTimeFormat'))
    : '-'
}
watch(
  () => props.evidence,
  (value) => {
    if (value.error) message.error(value.error, 8)
    else if (value.result?.truncated) message.warning(t('query.truncated'), 8)
  },
  { immediate: true },
)
const columns = computed(() =>
  (props.evidence.result?.columns || []).map((column) => ({
    title: column.label,
    dataIndex: column.key,
    key: column.key,
    ellipsis: true,
    width: 180,
  })),
)
const displayRows = computed(() =>
  (props.evidence.result?.rows || []).map((row, i) => {
    const display: Record<string, unknown> = { ...row, _row: i }
    for (const column of props.evidence.result?.columns || []) {
      const value = row[column.key]
      if (
        typeof value === 'string' &&
        /DATE|TIME/i.test(column.data_type) &&
        appDayjs(value).isValid()
      )
        display[column.key] = formatTime(value, /^\d{4}-\d{2}-\d{2}$/.test(value))
    }
    return display
  }),
)
</script>
<template>
  <a-tabs class="memory-surface">
    <a-tab-pane key="data" :tab="t('query.result')">
      <a-alert v-if="evidence.status === 'skipped'" type="info" show-icon
        :message="t('query.triggerSkipped')" :description="t('query.triggerSkippedHint')" class="trigger-result" />
      <div v-if="evidence.trigger" class="query-metrics">
        <a-tag v-if="evidence.trigger_matched === true" color="success">{{ t('query.triggerMatched') }}</a-tag>
        {{ triggerSummary }}
      </div>
      <a-tag v-if="evidence.error" color="error">{{ t('query.resultFailed') }}</a-tag>
      <template v-if="evidence.result">
        <div class="query-metrics">
          <a-tag v-if="evidence.result.truncated" color="warning">{{
            t('query.resultIncomplete')
          }}</a-tag>
          {{ t('query.resultSummary', [evidence.result.rows.length, evidence.result.duration_ms]) }}
          · {{ formatTime(evidence.queried_at) }}
        </div>
        <a-table
          size="small"
          :columns="columns"
          :data-source="displayRows"
          row-key="_row"
          :pagination="false"
          :scroll="{ x: 'max-content', y: 280 }"
          :locale="{ emptyText: t('query.noRows') }"
        />
      </template>
    </a-tab-pane>
    <a-tab-pane key="parameters" :tab="t('query.boundParameters')">
      <pre class="query-preview">{{ JSON.stringify(evidence.parameters, null, 2) }}</pre>
    </a-tab-pane>
    <a-tab-pane key="model" :tab="t('query.modelPreview')">
      <pre class="query-preview">{{ evidence.model_text }}</pre>
    </a-tab-pane>
  </a-tabs>
</template>
<style scoped>
.trigger-result {
  margin-bottom: 12px;
}
.query-metrics {
  color: var(--color-text-secondary);
  font-size: 12px;
  margin-bottom: 12px;
}
.query-preview {
  padding: 16px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md, 8px);
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  max-height: 400px;
  overflow: auto;
  font:
    12px/1.7 ui-monospace,
    SFMono-Regular,
    Menlo,
    monospace;
}
</style>
