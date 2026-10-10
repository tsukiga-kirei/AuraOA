<script setup lang="ts">
import { message } from 'ant-design-vue'
import { PlusOutlined, DeleteOutlined } from '@ant-design/icons-vue'
import { useAuditDataQueryApi } from '~/composables/useAuditDataQueryApi'
import { compileQueryBuilder } from '~/utils/queryBuilder'
import type { QueryBuilder, QueryColumn, QueryDefinition } from '~/types/audit-data-query'
const props = defineProps<{
  definition: QueryDefinition
  disabled?: boolean
}>()
const emit = defineEmits<{
  apply: [value: QueryDefinition]
  'update:builder': [value: QueryBuilder]
}>()
const { t } = useI18n()
const api = useAuditDataQueryApi()
const clone = <T,>(value: T): T => JSON.parse(JSON.stringify(value))
const builder = ref<QueryBuilder>(
  clone(
    props.definition.builder || {
      source_type: 'table',
      table: props.definition.tables[0] || '',
      columns: [],
      filters: [],
      order_alias: '',
      order_dir: 'DESC',
    },
  ),
)
const columns = ref<QueryColumn[]>([])
const loading = ref(false)
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
watch(builder, (value) => emit('update:builder', clone(value)), { deep: true })
watch(
  () => builder.value.table,
  () => {
    ++sequence
    columns.value = []
  },
)
async function loadSchema(notify = true) {
  const seq = ++sequence
  loading.value = true
  try {
    const result = await api.schema(builder.value.table.trim())
    if (seq !== sequence) return
    columns.value = result.columns
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
  builder.value.columns.push({ field: '', alias: '', aggregate: '' })
}
function selectField(index: number, value: string) {
  const column = columns.value.find((item) => item.key === value)
  if (!builder.value.columns[index].alias && column && /[\u3400-\u9fff]/.test(column.label))
    builder.value.columns[index].alias = column.label
}
function addFilter() {
  builder.value.filters.push({
    field: '',
    operator: '=',
    parameter: '',
    label: '',
    type: 'string',
  })
}
function apply() {
  try {
    const compiled = compileQueryBuilder(builder.value, props.definition.dialect, columns.value)
    emit('apply', {
      ...clone(props.definition),
      ...compiled,
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
      <small v-if="builder.source_type === 'workflow'" class="muted">{{
        t('query.builder.archived')
      }}</small>
    </a-form-item>
    <div class="section-title">
      <strong>{{ t('query.builder.columns') }}</strong
      ><a-button
        size="small"
        :disabled="disabled || !columns.length || builder.columns.length >= 40"
        @click="addColumn"
        ><template #icon><PlusOutlined /></template>{{ t('query.add') }}</a-button
      >
    </div>
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
    <div class="section-title">
      <strong>{{ t('query.builder.filters') }}</strong
      ><a-button
        size="small"
        :disabled="disabled || !columns.length || builder.filters.length >= 20"
        @click="addFilter"
        ><template #icon><PlusOutlined /></template>{{ t('query.add') }}</a-button
      >
    </div>
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
        @change="filter.type = filter.operator === 'IN' ? 'string_list' : 'string'"
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
          v-model:value="filter.parameter"
          :disabled="disabled"
          :placeholder="t('query.parameterName')"
        />
        <a-input
          v-model:value="filter.label"
          :disabled="disabled"
          :placeholder="t('query.parameterLabel')"
        />
        <a-select v-model:value="filter.type" :disabled="disabled || filter.operator === 'IN'"
          ><a-select-option
            v-for="value in ['string', 'number', 'date', 'string_list']"
            :key="value"
            :value="value"
            >{{ t('query.type.' + value) }}</a-select-option
          ></a-select
        >
      </template>
    </div>
    <a-form-item :label="t('query.builder.order')">
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
    </a-form-item>
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
.section-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
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
.filter-row > :nth-child(4) {
  grid-column: 1;
}
.filter-row > :nth-child(6) {
  grid-column: 1 / 3;
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
