<script setup lang="ts">
import { message } from 'ant-design-vue'
import {
  PlusOutlined,
  ReloadOutlined,
  CopyOutlined,
  DeleteOutlined,
  TableOutlined,
} from '@ant-design/icons-vue'
import type { QueryColumn } from '~/types/audit-data-query'
const props = defineProps<{
  tables: string[]
  schema: Record<string, QueryColumn[]>
  loadingTables: string[]
  fieldNotes: Record<string, string>
  lockedTables?: string[]
  disabled?: boolean
}>()
const emit = defineEmits<{
  add: [table: string]
  remove: [table: string]
  load: [table: string]
  note: [key: string, value: string]
}>()
const { t } = useI18n()
const tableName = ref('')
const activeTable = ref('')
const keyword = ref('')
watch(
  () => props.tables,
  (tables) => {
    if (!tables.includes(activeTable.value)) activeTable.value = tables[0] || ''
  },
  { immediate: true },
)
watch(activeTable, () => {
  keyword.value = ''
})
const columns = computed(() => {
  const query = keyword.value.trim().toLowerCase()
  return (props.schema[activeTable.value] || []).filter((column) =>
    [
      column.key,
      column.label,
      column.data_type,
      props.fieldNotes[`${activeTable.value}.${column.key}`] || '',
    ].some((value) => (value || '').toLowerCase().includes(query)),
  )
})
function addTable() {
  const table = tableName.value.trim()
  if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(table)) {
    message.warning(t('query.tableNameHint'))
    return
  }
  emit('add', table)
  activeTable.value = table
  tableName.value = ''
}
async function copyField(field: string) {
  try {
    await navigator.clipboard.writeText(`${activeTable.value}.${field}`)
    message.success(t('query.fieldCopied'))
  } catch {
    message.error(t('query.copyFailed'))
  }
}
</script>
<template>
  <section class="schema-browser">
    <div class="schema-toolbar memory-input-action">
      <a-input
        v-model:value="tableName"
        :disabled="disabled"
        :placeholder="t('query.tableName')"
        @press-enter="addTable"
      />
      <a-button :disabled="disabled || tables.length >= 20" @click="addTable">
        <template #icon><PlusOutlined /></template>{{ t('query.addTable') }}
      </a-button>
    </div>
    <p class="memory-help">{{ t('query.schemaBrowserHint') }}</p>
    <a-empty v-if="!tables.length" :description="t('query.noTables')" />
    <div v-else class="schema-layout">
      <div class="schema-table-list" :aria-label="t('query.schema')">
        <button
          v-for="table in tables"
          :key="table"
          type="button"
          class="schema-table-item"
          :class="{ selected: activeTable === table }"
          :aria-pressed="activeTable === table"
          @click="activeTable = table"
        >
          <TableOutlined />
          <span
            ><strong>{{ table }}</strong
            ><small>{{
              schema[table]
                ? t('query.fieldCount', schema[table].length)
                : t('query.schemaNotLoaded')
            }}</small></span
          >
        </button>
      </div>
      <div class="schema-detail">
        <div class="schema-detail-heading">
          <strong>{{ activeTable }}</strong>
          <a-space>
            <a-button
              size="small"
              :loading="loadingTables.includes(activeTable)"
              :disabled="disabled"
              @click="emit('load', activeTable)"
            >
              <template #icon><ReloadOutlined /></template>{{ t('query.builder.load') }}
            </a-button>
            <a-button
              v-if="!lockedTables?.includes(activeTable)"
              size="small"
              danger
              :disabled="disabled"
              :aria-label="t('query.remove')"
              @click="emit('remove', activeTable)"
            >
              <template #icon><DeleteOutlined /></template>
            </a-button>
          </a-space>
        </div>
        <a-input
          v-model:value="keyword"
          allow-clear
          :placeholder="t('query.searchFields')"
          :aria-label="t('query.searchFields')"
          class="field-search"
        />
        <!-- 表头与内容共用列宽，只滚动同一张表，避免输入框撑宽后表头错位。 -->
        <a-spin v-if="schema[activeTable]" :spinning="loadingTables.includes(activeTable)">
          <div
            class="field-table-scroll"
            role="region"
            :aria-label="t('query.schema')"
            tabindex="0"
          >
            <table class="field-table">
              <colgroup>
                <col class="physical-column" />
                <col />
                <col class="type-column" />
              </colgroup>
              <thead>
                <tr>
                  <th scope="col">{{ t('query.fieldPhysical') }}</th>
                  <th scope="col">{{ t('query.fieldDescription') }}</th>
                  <th scope="col">{{ t('query.fieldType') }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="record in columns" :key="record.key">
                  <td>
                    <div class="field-key">
                      <code>{{ record.key }}</code>
                      <a-button
                        type="text"
                        size="small"
                        :aria-label="t('query.copyField')"
                        :title="t('query.copyField')"
                        @click="copyField(record.key)"
                      >
                        <template #icon><CopyOutlined /></template>
                      </a-button>
                    </div>
                  </td>
                  <td>
                    <span v-if="record.label !== record.key" class="field-label">{{
                      record.label
                    }}</span>
                    <a-input
                      :value="fieldNotes[`${activeTable}.${record.key}`] || ''"
                      :disabled="disabled"
                      :maxlength="200"
                      :placeholder="t('query.fieldNotePlaceholder')"
                      :aria-label="t('query.fieldDescription')"
                      @update:value="emit('note', `${activeTable}.${record.key}`, $event || '')"
                    />
                  </td>
                  <td class="field-data-type">{{ record.data_type || '-' }}</td>
                </tr>
                <tr v-if="!columns.length">
                  <td colspan="3" class="field-empty">{{ t('query.noRows') }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </a-spin>
        <div v-else class="schema-empty">
          <a-spin v-if="loadingTables.includes(activeTable)" />
          <a-empty v-else :description="t('query.readFieldsHint')" />
        </div>
      </div>
    </div>
  </section>
</template>
<style scoped>
.schema-browser {
  min-width: 0;
}
.schema-toolbar {
  width: 100%;
}
.schema-layout {
  display: grid;
  grid-template-columns: 190px minmax(0, 1fr);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  overflow: hidden;
}
.schema-table-list {
  background: var(--color-bg-secondary);
  border-right: 1px solid var(--color-border-light);
  min-height: 0;
  max-height: 460px;
  overflow: auto;
}
.schema-table-item {
  width: 100%;
  display: flex;
  align-items: flex-start;
  gap: 8px;
  border: 0;
  border-bottom: 1px solid var(--color-border-light);
  padding: 12px;
  color: var(--color-text-secondary);
  background: transparent;
  text-align: left;
  cursor: pointer;
}
.schema-table-item .anticon {
  margin-top: 3px;
  flex-shrink: 0;
}
.schema-table-item span {
  min-width: 0;
}
.schema-table-item strong {
  display: block;
  overflow-wrap: anywhere;
  font-size: 12px;
  font-weight: 500;
}
.schema-table-item small {
  font-size: 11px;
  color: var(--color-text-tertiary);
}
.schema-table-item.selected {
  background: var(--color-primary-bg);
  color: var(--color-primary);
  box-shadow: inset 3px 0 var(--color-primary);
}
.schema-table-item:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: -2px;
}
.schema-detail {
  padding: 12px;
  min-width: 0;
}
.schema-detail-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}
.schema-detail-heading > strong {
  min-width: 0;
  overflow-wrap: anywhere;
  font-size: 13px;
}
.field-search {
  margin-bottom: 12px;
}
.field-table-scroll {
  max-height: 320px;
  overflow: auto;
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
}
.field-table-scroll:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}
.field-table {
  width: 100%;
  min-width: 520px;
  table-layout: fixed;
  border-spacing: 0;
  border-collapse: separate;
  text-align: left;
}
.physical-column {
  width: 34%;
}
.type-column {
  width: 22%;
}
.field-table th,
.field-table td {
  padding: 10px 12px;
  border-bottom: 1px solid var(--color-border-light);
  vertical-align: middle;
}
.field-table th {
  position: sticky;
  top: 0;
  z-index: 1;
  background: var(--color-bg-page);
  color: var(--color-text-secondary);
  font-size: 13px;
  font-weight: 600;
}
.field-table td {
  background: var(--color-bg-card);
  color: var(--color-text-primary);
  font-size: 13px;
}
.field-table tr:last-child td {
  border-bottom: 0;
}
.field-table tbody tr:hover td {
  background: var(--color-bg-hover);
}
.field-table :deep(.ant-input) {
  width: 100%;
  min-width: 0;
}
.field-data-type {
  overflow-wrap: anywhere;
}
.field-table .field-empty {
  padding: 32px 12px;
  text-align: center;
  color: var(--color-text-tertiary);
}
.field-key {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.field-key code {
  overflow-wrap: anywhere;
  font-size: 12px;
}
.field-label {
  display: block;
  color: var(--color-text-secondary);
  margin-bottom: 4px;
  font-size: 12px;
}
.schema-empty {
  display: grid;
  place-items: center;
  min-height: 200px;
}
@media (max-width: 1000px) {
  .schema-layout {
    grid-template-columns: minmax(0, 1fr);
  }
  .schema-table-list {
    min-height: 0;
    display: flex;
    border-right: 0;
    max-height: 160px;
    overflow-x: auto;
  }
  .schema-table-item {
    width: 190px;
    flex-shrink: 0;
  }
}
</style>
