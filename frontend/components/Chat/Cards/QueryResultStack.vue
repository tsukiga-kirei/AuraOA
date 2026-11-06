<script setup lang="ts">
import { DatabaseOutlined } from '@ant-design/icons-vue'

const props = defineProps<{
  result: {
    query_count?: number
    queries?: Array<Record<string, any>>
  }
}>()

const { t } = useI18n()
const expanded = ref<Record<number, boolean>>({})
const previewLimit = 8

const queries = computed(() => props.result?.queries || [])

const visibleRows = (item: Record<string, any>, index: number) => {
  const rows = Array.isArray(item.rows) ? item.rows : []
  if (expanded.value[index] || rows.length <= previewLimit) return rows
  return rows.slice(0, previewLimit)
}

const formatCell = (value: unknown) => {
  if (value === null || value === undefined || value === '') return '—'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

const toggle = (index: number) => {
  expanded.value = { ...expanded.value, [index]: !expanded.value[index] }
}
</script>

<template>
  <div class="query-stack">
    <div class="query-stack-head">
      <DatabaseOutlined />
      <span>{{ t('chat.queryStack.title', '一次返回多组查询') }}</span>
      <span class="query-count">{{ result.query_count || queries.length }} {{ t('chat.queryStack.group', '组') }}</span>
    </div>

    <article v-for="(item, index) in queries" :key="index" class="query-card" :class="{ 'is-error': Boolean(item.error) }">
      <div class="query-card-top">
        <span class="query-index">{{ index + 1 }}</span>
        <span class="query-title">{{ item.title || t('chat.queryStack.untitled', `查询 ${index + 1}`) }}</span>
        <span v-if="!item.error" class="query-meta">{{ item.total_rows || 0 }} {{ t('chat.queryStack.rowUnit', '行') }}</span>
        <span v-if="item.truncated" class="query-badge">{{ t('chat.queryStack.truncated', '已按行数上限截断') }}</span>
      </div>

      <div v-if="item.error" class="query-error">{{ item.error }}</div>

      <template v-else>
        <div v-if="item.columns?.length" class="query-table-wrap">
          <table>
            <thead>
              <tr>
                <th v-for="col in item.columns" :key="col">{{ col }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="!visibleRows(item, index).length">
                <td :colspan="item.columns.length" class="query-empty">{{ t('chat.queryStack.empty', '这条查询没有返回数据') }}</td>
              </tr>
              <tr v-for="(row, rowIndex) in visibleRows(item, index)" :key="rowIndex">
                <td v-for="col in item.columns" :key="col">{{ formatCell(row?.[col]) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="query-foot">
          <span class="query-hint">{{ t('chat.queryStack.hint', '完整结果已交给模型，这里展示便于核对的表格。') }}</span>
          <a-button
            v-if="(item.rows?.length || 0) > previewLimit"
            type="link"
            size="small"
            class="query-expand"
            @click="toggle(index)"
          >
            {{ expanded[index] ? t('chat.queryStack.collapse', '收起') : t('chat.queryStack.expand', '展开全部') }}
          </a-button>
        </div>
      </template>
    </article>
  </div>
</template>

<style scoped>
.query-stack {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin: 6px 0;
}
.query-stack-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-radius: 8px;
  background: var(--color-primary-bg);
  color: var(--color-text-primary);
  border: 1px solid var(--color-border-light);
  font-size: 13px;
  font-weight: 600;
}
.query-stack-head :deep(.anticon) {
  color: var(--color-primary);
}
.query-count {
  margin-left: auto;
  font-weight: 500;
  color: var(--color-primary);
}
.query-card {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border-light);
  border-radius: 8px;
  padding: 12px;
}
.query-card.is-error {
  border-color: var(--color-danger);
}
.query-card-top {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.query-index {
  flex: none;
  width: 20px;
  height: 20px;
  border-radius: 999px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--color-primary-bg);
  color: var(--color-primary);
  font-size: 12px;
  font-weight: 600;
}
.query-title {
  min-width: 0;
  color: var(--color-text-primary);
  font-weight: 600;
  font-size: 13px;
}
.query-meta {
  flex: none;
  color: var(--color-text-secondary);
  font-size: 12px;
}
.query-badge {
  flex: none;
  padding: 0 6px;
  border-radius: 4px;
  background: var(--color-warning-bg);
  color: var(--color-warning);
  font-size: 12px;
  line-height: 20px;
}
.query-error {
  margin-top: 8px;
  padding: 8px 10px;
  border-radius: 6px;
  background: var(--color-danger-bg);
  color: var(--color-danger);
  font-size: 12px;
  line-height: 1.5;
}
.query-table-wrap {
  margin-top: 8px;
  overflow: auto;
  border: 1px solid var(--color-border-light);
  border-radius: 6px;
}
table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
}
th {
  background: var(--color-bg-page);
  color: var(--color-text-secondary);
  text-align: left;
  padding: 6px 8px;
  font-weight: 500;
  white-space: nowrap;
}
td {
  padding: 6px 8px;
  border-top: 1px solid var(--color-border-light);
  color: var(--color-text-primary);
  white-space: nowrap;
}
.query-empty {
  color: var(--color-text-secondary);
  text-align: center;
}
.query-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-top: 8px;
}
.query-hint {
  color: var(--color-text-secondary);
  font-size: 12px;
  line-height: 1.5;
}
.query-expand {
  padding-inline: 0;
  height: auto;
}
</style>
