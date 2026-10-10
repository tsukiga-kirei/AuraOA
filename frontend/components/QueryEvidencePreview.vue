<script setup lang="ts">
import type { QueryEvidence } from '~/types/audit-data-query'
const props = defineProps<{ evidence: QueryEvidence }>()
const { t } = useI18n()
const columns = computed(() =>
  (props.evidence.result?.columns || []).map((column) => ({
    title: column.label,
    dataIndex: column.key,
    key: column.key,
    ellipsis: true,
    width: 180,
  })),
)
</script>
<template>
  <a-tabs class="memory-surface">
    <a-tab-pane key="data" :tab="t('query.result')">
      <a-alert v-if="evidence.error" type="error" :message="evidence.error" show-icon />
      <template v-if="evidence.result">
        <a-alert
          v-if="evidence.result.truncated"
          type="warning"
          :message="t('query.truncated')"
          show-icon
          class="query-notice"
        />
        <div class="query-metrics">
          {{ t('query.resultSummary', [evidence.result.rows.length, evidence.result.duration_ms]) }}
          · {{ evidence.queried_at }}
        </div>
        <a-table
          size="small"
          :columns="columns"
          :data-source="evidence.result.rows.map((row, i) => ({ ...row, _row: i }))"
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
.query-notice {
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
