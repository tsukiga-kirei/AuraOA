<script setup lang="ts">
import { message } from 'ant-design-vue'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons-vue'
import { useAuditDataQueryApi } from '~/composables/useAuditDataQueryApi'
import QueryValueSourceEditor from './QueryValueSourceEditor.vue'
import { compileQueryBuilder } from '~/utils/queryBuilder'
import type {
  QueryBuilder,
  QueryColumn,
  QueryDefinition,
  QueryBinding,
} from '~/types/audit-data-query'
const props = defineProps<{
  definition: QueryDefinition
  disabled?: boolean
}>()
const emit = defineEmits<{
  apply: [value: QueryDefinition]
  'update:builder': [value: QueryBuilder]
  schema: [table: string, columns: QueryColumn[]]
}>()
const { t } = useI18n()
const api = useAuditDataQueryApi()
const clone = <T,>(value: T): T => JSON.parse(JSON.stringify(value))
const builder = ref<QueryBuilder>(
  clone(
    props.definition.builder || {
      source_type: 'table',
      archived_only: false,
      table: props.definition.tables[0] || '',
      columns: [],
      filters: [],
      order_alias: '',
      order_dir: 'DESC',
    },
  ),
)
// 旧配置保留归档口径，新建配置默认不附加状态条件。
builder.value.archived_only ??= builder.value.source_type === 'workflow'
for (const filter of builder.value.filters) {
  const parameter = props.definition.parameters.find((item) => item.name === filter.parameter)
  filter.binding ??= clone(parameter?.default_binding || { source: 'field' })
  filter.parameter ||= nextParameterName()
}
const columns = ref<QueryColumn[]>([])
const loading = ref(false)
const expandedSections = ref(['columns', 'filters', 'order'])
let sequence = 0
const fieldOptions = computed(() =>
  columns.value.map((column) => ({
    value: column.key,
    label: column.label === column.key ? column.key : `${column.label} (${column.key})`,
  })),
)
const operatorOptions = [
  '=',
  '<>',
  '>',
  '>=',
  '<',
  '<=',
  'LIKE',
  'IN',
  'IS NULL',
  'IS NOT NULL',
].map((value) => ({ value, label: value }))
// 配置变更同步写回草稿，避免紧接着保存或切换模式时遗漏最新字段。
watch(builder, (value) => emit('update:builder', clone(value)), { deep: true, flush: 'sync' })
watch(
  () => builder.value.table,
  () => {
    ++sequence
    columns.value = []
    loading.value = false
  },
)
async function loadSchema(notify = true) {
  const seq = ++sequence
  loading.value = true
  try {
    const result = await api.schema(builder.value.table.trim())
    if (seq !== sequence) return
    columns.value = result.columns
    emit('schema', builder.value.table.trim(), result.columns)
    if (result.dialect !== props.definition.dialect) {
      message.error(t('query.sourceChanged'))
      return
    }
    if (notify)
      message.success(t('query.schemaLoaded', [builder.value.table, result.columns.length]))
  } catch (cause) {
    if (seq === sequence) message.error(cause instanceof Error ? cause.message : String(cause))
  } finally {
    if (seq === sequence) loading.value = false
  }
}
function addColumn() {
  if (!expandedSections.value.includes('columns')) expandedSections.value.push('columns')
  builder.value.columns.push({ field: '', alias: '', aggregate: '' })
}
function selectField(index: number, value: string) {
  const column = columns.value.find((item) => item.key === value)
  if (!builder.value.columns[index].alias && column && /[\u3400-\u9fff]/.test(column.label))
    builder.value.columns[index].alias = column.label
}
function nextParameterName() {
  let index = 1
  const used = new Set(builder.value.filters.map((filter) => filter.parameter))
  while (used.has(`filter_${index}`)) index++
  return `filter_${index}`
}
function changeFilterOperator(index: number) {
  const filter = builder.value.filters[index]
  filter.type = filter.operator === 'IN' ? 'string_list' : 'string'
  if (filter.binding) delete filter.binding.days_offset
}
function changeFilterType(index: number) {
  const filter = builder.value.filters[index]
  if (filter.type !== 'date' && filter.binding) delete filter.binding.days_offset
}
function setFilterBinding(index: number, value: QueryBinding) {
  const filter = builder.value.filters[index]
  filter.binding = value
  if (filter.operator !== 'IN' && value.source === 'system') {
    filter.type = value.value && value.value !== 'process_id' ? 'date' : 'string'
  }
  if (filter.type !== 'date') delete filter.binding.days_offset
}
function prepareFilterLabels() {
  builder.value.filters.forEach((filter, index) => {
    if (!filter.label.trim()) {
      const column = columns.value.find((item) => item.key === filter.field)
      filter.label =
        column && /[\u3400-\u9fff]/.test(column.label)
          ? column.label
          : t('query.builder.filterValue', index + 1)
    }
  })
}
function addFilter() {
  if (!expandedSections.value.includes('filters')) expandedSections.value.push('filters')
  builder.value.filters.push({
    field: '',
    operator: '=',
    parameter: nextParameterName(),
    binding: { source: 'constant', value: '' },
    label: '',
    type: 'string',
  })
}
function apply() {
  try {
    prepareFilterLabels()
    const compiled = compileQueryBuilder(builder.value, props.definition.dialect, columns.value)
    emit('apply', {
      ...clone(props.definition),
      ...compiled,
      mode: 'config',
      builder: clone(builder.value),
    })
    message.success(t('query.builder.applied'))
  } catch (cause) {
    const code = cause instanceof Error ? cause.message : 'schema'
    message.error(t(`query.builder.error.${code}`))
  }
}
onMounted(() => {
  if (builder.value.table) loadSchema(false)
})
</script>
<template>
  <div class="query-builder">
    <p class="memory-help builder-help">{{ t('query.builder.hint') }}</p>
    <a-form-item :label="t('query.builder.source')">
      <a-select v-model:value="builder.source_type" :disabled="disabled">
        <a-select-option value="table">{{ t('query.builder.table') }}</a-select-option>
        <a-select-option value="workflow">{{ t('query.builder.workflow') }}</a-select-option>
      </a-select>
    </a-form-item>
    <a-form-item :label="t('query.builder.tableName')" required>
      <div class="memory-input-action">
        <a-input
          v-model:value="builder.table"
          :disabled="disabled"
          :placeholder="t('query.tableName')"
        />
        <a-button :loading="loading" :disabled="disabled" @click="loadSchema()">{{
          t('query.builder.load')
        }}</a-button>
      </div>
    </a-form-item>
    <div v-if="builder.source_type === 'workflow'" class="workflow-scope">
      <div class="workflow-scope-heading">
        <strong>{{ t('query.builder.archivedOnly') }}</strong>
        <a-switch v-model:checked="builder.archived_only" :disabled="disabled" />
      </div>
      <p class="memory-help">{{ t('query.builder.archived') }}</p>
    </div>
    <a-collapse v-model:active-key="expandedSections" class="builder-sections" :bordered="false">
      <a-collapse-panel key="columns">
        <template #header
          ><strong>{{ t('query.builder.columns') }}</strong></template
        >
        <template #extra>
          <a-button
            size="small"
            :disabled="disabled || !columns.length || builder.columns.length >= 40"
            @click.stop="addColumn"
            ><template #icon><PlusOutlined /></template>{{ t('query.add') }}</a-button
          >
        </template>
        <div v-for="(column, index) in builder.columns" :key="index" class="config-row">
          <a-select
            v-model:value="column.field"
            show-search
            :options="
              column.aggregate === 'COUNT'
                ? [{ value: '*', label: '*' }, ...fieldOptions]
                : fieldOptions
            "
            :placeholder="t('query.builder.field')"
            :disabled="disabled"
            @change="(value) => selectField(index, String(value ?? ''))"
          />
          <a-select v-model:value="column.aggregate" :disabled="disabled"
            ><a-select-option value="">{{ t('query.builder.detail') }}</a-select-option
            ><a-select-option
              v-for="value in ['SUM', 'COUNT', 'MIN', 'MAX', 'AVG']"
              :key="value"
              :value="value"
              >{{ t('query.builder.' + value) }}</a-select-option
            ></a-select
          >
          <a-input
            v-model:value="column.alias"
            :disabled="disabled"
            :maxlength="100"
            :placeholder="t('query.builder.alias')"
          />
          <a-button
            type="text"
            :aria-label="t('query.remove')"
            :disabled="disabled"
            @click="builder.columns.splice(index, 1)"
            ><DeleteOutlined
          /></a-button>
        </div>
      </a-collapse-panel>
      <a-collapse-panel key="filters">
        <template #header
          ><strong>{{ t('query.builder.filters') }}</strong></template
        >
        <template #extra>
          <a-button
            size="small"
            :disabled="disabled || !columns.length || builder.filters.length >= 20"
            @click.stop="addFilter"
            ><template #icon><PlusOutlined /></template>{{ t('query.add') }}</a-button
          >
        </template>
        <div v-for="(filter, index) in builder.filters" :key="index" class="filter-row">
          <a-select
            v-model:value="filter.field"
            show-search
            :options="fieldOptions"
            :placeholder="t('query.builder.field')"
            :disabled="disabled"
          />
          <a-select
            v-model:value="filter.operator"
            :options="operatorOptions"
            :disabled="disabled"
            @change="changeFilterOperator(index)"
          />
          <a-button
            type="text"
            :aria-label="t('query.remove')"
            :disabled="disabled"
            @click="builder.filters.splice(index, 1)"
            ><DeleteOutlined
          /></a-button>
          <template v-if="!filter.operator.startsWith('IS ')">
            <a-input
              v-model:value="filter.label"
              :disabled="disabled"
              :placeholder="t('query.builder.valuePurpose')"
              class="filter-purpose"
            />
            <QueryValueSourceEditor
              :model-value="filter.binding"
              :type="filter.type"
              @update:model-value="(value) => setFilterBinding(index, value)"
              :disabled="disabled"
              class="filter-value-source"
            />
            <a-select
              v-model:value="filter.type"
              class="filter-type"
              @change="changeFilterType(index)"
              :disabled="disabled || filter.operator === 'IN'"
              ><a-select-option
                v-for="value in ['string', 'number', 'date', 'string_list']"
                :key="value"
                :value="value"
                >{{ t('query.type.' + value) }}</a-select-option
              ></a-select
            >
          </template>
        </div>
      </a-collapse-panel>
      <a-collapse-panel key="order">
        <template #header
          ><strong>{{ t('query.builder.order') }}</strong></template
        >
        <div class="query-order">
          <a-select v-model:value="builder.order_alias" :disabled="disabled"
            ><a-select-option value="">{{ t('query.builder.noOrder') }}</a-select-option
            ><a-select-option
              v-for="column in builder.columns.filter((item) => item.alias)"
              :key="column.alias"
              :value="column.alias"
              >{{ column.alias }}</a-select-option
            ></a-select
          >
          <a-select v-model:value="builder.order_dir" :disabled="disabled || !builder.order_alias"
            ><a-select-option value="DESC">{{ t('query.builder.desc') }}</a-select-option
            ><a-select-option value="ASC">{{ t('query.builder.asc') }}</a-select-option></a-select
          >
        </div>
      </a-collapse-panel>
    </a-collapse>
    <a-button type="primary" :disabled="disabled || loading || !columns.length" @click="apply">{{
      t('query.builder.apply')
    }}</a-button>
    <p class="muted">{{ t('query.builder.parameterHint') }}</p>
  </div>
</template>
<style scoped>
.query-builder {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-top: 12px;
}
.query-builder :deep(.ant-form-item) {
  margin-bottom: 0;
}
.builder-help {
  margin: 0;
}
.builder-sections {
  background: transparent;
}
.builder-sections :deep(.ant-collapse-header) {
  align-items: center;
  padding: 12px 0;
}
.builder-sections :deep(.ant-collapse-content-box) {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 0 0 12px;
}
.builder-sections :deep(.ant-collapse-item) {
  border-bottom-color: var(--color-border-light);
}
.query-order {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 110px;
  gap: 12px;
}
.query-order :deep(.ant-select) {
  min-width: 0;
}
.config-row {
  display: grid;
  grid-template-columns: minmax(100px, 1fr) 100px minmax(100px, 1fr) 32px;
  gap: 8px;
}
.filter-row {
  display: grid;
  grid-template-columns: minmax(110px, 1fr) minmax(110px, 1fr) 40px;
  gap: 8px;
  padding: 12px;
  border: 1px solid var(--color-border);
  border-radius: 8px;
}
.filter-purpose,
.filter-value-source {
  grid-column: 1 / -1;
}
.filter-type {
  grid-column: 1 / 3;
}
.workflow-scope {
  padding: 12px 0;
  border-bottom: 1px solid var(--color-border-light);
}
.workflow-scope-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 8px;
}
.muted {
  color: var(--color-text-tertiary);
  font-size: 12px;
}
@media (max-width: 760px) {
  .config-row {
    grid-template-columns: minmax(90px, 1fr) 100px 32px;
  }
  .config-row .ant-input {
    grid-row: 2;
    grid-column: 1 / 3;
  }
  .config-row .ant-btn {
    grid-column: 3;
    grid-row: 1;
  }
}
</style>
