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
const headings = computed(() => [
  { title: t('query.fieldPhysical'), dataIndex: 'key', key: 'key', width: 180 },
  { title: t('query.fieldDescription'), key: 'description', width: 230 },
  { title: t('query.fieldType'), dataIndex: 'data_type', key: 'data_type', width: 110 },
])
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
        <a-table
          v-if="schema[activeTable]"
          size="small"
          :data-source="columns"
          :columns="headings"
          :pagination="false"
          row-key="key"
          :scroll="{ x: 520, y: 320 }"
          :loading="loadingTables.includes(activeTable)"
        >
          <template #bodyCell="{ column, record }">
            <div v-if="column.key === 'key'" class="field-key">
              <code>{{ record.key }}</code>
              <a-button
                type="text"
                size="small"
                :aria-label="t('query.copyField')"
                :title="t('query.copyField')"
                @click="copyField(record.key)"
                ><template #icon><CopyOutlined /></template
              ></a-button>
            </div>
            <div v-else-if="column.key === 'description'">
              <span v-if="record.label !== record.key" class="field-label">{{ record.label }}</span>
              <a-input
                :value="fieldNotes[`${activeTable}.${record.key}`] || ''"
                :disabled="disabled"
                :maxlength="200"
                :placeholder="t('query.fieldNotePlaceholder')"
                :aria-label="t('query.fieldDescription')"
                @update:value="emit('note', `${activeTable}.${record.key}`, $event)"
              />
            </div>
          </template>
        </a-table>
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
  max-width: 680px;
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
  max-height: 420px;
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
