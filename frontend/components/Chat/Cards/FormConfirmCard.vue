<script setup lang="ts">
import { ref, reactive, computed, watch } from 'vue'
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  DownloadOutlined,
  FormOutlined,
  TableOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'

export interface FormConfirmOption {
  label: string
  value: string | number
}

export interface FormConfirmField {
  name: string
  label: string
  type?: 'text' | 'number' | 'date' | 'select' | 'textarea' | 'browser'
  value?: any
  display_value?: string // 浏览按钮展示名称（如人员姓名、部门名称），与底层存储 ID 分离
  disabled?: boolean
  options?: (string | FormConfirmOption)[]
}

export interface FormConfirmPayload {
  variant?: 'summary_confirm' | 'compact_form' | 'data_table'
  workflow_id?: string
  workflow_name?: string
  title?: string
  fields?: FormConfirmField[]
  columns?: any[]
  rows?: Record<string, any>[]
}

const props = defineProps<{
  data: FormConfirmPayload
  disabled?: boolean
}>()

const emit = defineEmits<{
  (e: 'send', text: string): void
}>()

const { t } = useI18n()

const status = ref<'idle' | 'confirmed' | 'cancelled'>('idle')
const formModel = reactive<Record<string, any>>({})
const displayModel = reactive<Record<string, string>>({})

// 选项归一化（兼容纯字符串数组与 {label, value} 对象数组）
const normalizeOptions = (options?: (string | FormConfirmOption)[]) => {
  if (!options || !Array.isArray(options)) return []
  return options.map(opt => {
    if (typeof opt === 'object' && opt !== null) {
      return {
        label: opt.label ?? String(opt.value),
        value: opt.value ?? opt.label,
      }
    }
    return {
      label: String(opt),
      value: String(opt),
    }
  })
}

// 初始化表单字段默认值（兼容浏览按钮 display_value 与底值 value）
const initFormValues = () => {
  if (props.data?.fields && Array.isArray(props.data.fields)) {
    props.data.fields.forEach(f => {
      formModel[f.name] = f.value ?? f.display_value ?? ''
      displayModel[f.name] = f.display_value ?? (f.value !== undefined ? String(f.value) : '')
    })
  }
}

watch(
  () => props.data,
  () => {
    initFormValues()
  },
  { immediate: true, deep: true }
)

const isReadOnly = computed(() => props.disabled || status.value !== 'idle')

const handleConfirm = () => {
  if (isReadOnly.value) return
  status.value = 'confirmed'
  const wfId = props.data?.workflow_id || ''
  const payloadStr = JSON.stringify(formModel)
  const text = `【用户已确认表单】流程ID: ${wfId}, 数据: ${payloadStr}，请立即调用接口创建草稿。`
  emit('send', text)
}

const handleCancel = () => {
  if (isReadOnly.value) return
  status.value = 'cancelled'
  const text = `【用户已取消表单创建】，不需要提交该流程了。`
  emit('send', text)
}

// ── 表格与 CSV 导出 ──
const tableColumns = computed(() => {
  if (!props.data?.columns || !Array.isArray(props.data.columns)) return []
  return props.data.columns.map(col => {
    if (typeof col === 'string') {
      return {
        title: col,
        dataIndex: col,
        key: col,
        sorter: (a: any, b: any) => {
          const valA = a[col] ?? ''
          const valB = b[col] ?? ''
          if (typeof valA === 'number' && typeof valB === 'number') return valA - valB
          return String(valA).localeCompare(String(valB))
        },
      }
    }
    return col
  })
})

const exportCsv = () => {
  const rows = props.data?.rows || []
  const cols = props.data?.columns || []
  if (!rows.length) {
    message.warning(t('common.noData', '暂无可导出的数据'))
    return
  }

  const colHeaders: string[] = cols.map(c => (typeof c === 'string' ? c : c.title || c.dataIndex))
  const colKeys: string[] = cols.map(c => (typeof c === 'string' ? c : c.dataIndex || c.key))

  const escapeCsv = (val: any) => {
    if (val === null || val === undefined) return ''
    const s = String(val).replace(/"/g, '""')
    return `"${s}"`
  }

  const csvContent =
    '\uFEFF' +
    [colHeaders.map(escapeCsv).join(','), ...rows.map(row => colKeys.map(k => escapeCsv(row[k])).join(','))].join(
      '\n'
    )

  const blob = new Blob([csvContent], { type: 'text/csv;charset=utf-8;' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `${props.data?.title || 'query_result'}_${Date.now()}.csv`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
  message.success(t('common.exported', '已成功导出 CSV'))
}
</script>

<template>
  <div class="form-confirm-card" :class="[`form-confirm-card--${data.variant || 'summary_confirm'}`]">
    <!-- 1. 表格变体 (data_table) -->
    <template v-if="data.variant === 'data_table'">
      <div class="card-header">
        <div class="header-left">
          <TableOutlined class="header-icon" />
          <span class="header-title">{{ data.title || t('chat.formConfirm.title', '数据查询结果') }}</span>
          <span v-if="data.rows" class="row-count-badge">{{ data.rows.length }} 条记录</span>
        </div>
        <div class="header-right">
          <a-button size="small" @click="exportCsv">
            <template #icon><DownloadOutlined /></template>
            {{ t('chat.formConfirm.exportCsv', '导出 CSV') }}
          </a-button>
        </div>
      </div>
      <div class="table-container">
        <a-table
          :data-source="data.rows || []"
          :columns="tableColumns"
          size="small"
          :pagination="{ pageSize: 10, size: 'small', showTotal: total => `共 ${total} 条` }"
          :scroll="{ x: 'max-content' }"
          bordered
        />
      </div>
    </template>

    <!-- 2. 表单确认变体 (summary_confirm / compact_form) -->
    <template v-else>
      <div class="card-header">
        <div class="header-left">
          <FormOutlined class="header-icon" />
          <span class="header-title">{{ data.workflow_name || t('chat.formConfirm.title', '流程预填信息确认') }}</span>
          <a-tag v-if="data.workflow_id" color="blue" class="workflow-id-tag">ID: {{ data.workflow_id }}</a-tag>
        </div>
        <div class="header-right">
          <a-tag v-if="status === 'confirmed'" color="success">
            <template #icon><CheckCircleOutlined /></template>
            {{ t('chat.formConfirm.confirmed', '已确认提交') }}
          </a-tag>
          <a-tag v-else-if="status === 'cancelled'" color="default">
            <template #icon><CloseCircleOutlined /></template>
            {{ t('chat.formConfirm.cancelled', '已取消填单') }}
          </a-tag>
          <span v-else class="status-hint">请核对预填信息</span>
        </div>
      </div>

      <div class="form-body">
        <div v-for="field in data.fields || []" :key="field.name" class="form-row">
          <div class="field-label">{{ field.label || field.name }}:</div>
          <div class="field-control">
            <a-textarea
              v-if="field.type === 'textarea'"
              v-model:value="formModel[field.name]"
              :disabled="field.disabled || isReadOnly"
              :rows="2"
              class="custom-field"
            />
            <a-select
              v-else-if="field.type === 'select'"
              v-model:value="formModel[field.name]"
              :disabled="field.disabled || isReadOnly"
              class="custom-field w-full"
              :options="normalizeOptions(field.options)"
            />
            <a-date-picker
              v-else-if="field.type === 'date'"
              v-model:value="formModel[field.name]"
              value-format="YYYY-MM-DD"
              :disabled="field.disabled || isReadOnly"
              class="custom-field w-full"
            />
            <a-input-number
              v-else-if="field.type === 'number'"
              v-model:value="formModel[field.name]"
              :disabled="field.disabled || isReadOnly"
              class="custom-field w-full"
            />
            <div v-else-if="field.display_value || field.type === 'browser'" class="browse-display-wrap">
              <a-input
                v-model:value="displayModel[field.name]"
                :disabled="field.disabled || isReadOnly"
                class="custom-field"
                @change="() => {
                  formModel[field.name] = displayModel[field.name]
                }"
              />
              <span v-if="field.value && displayModel[field.name] && String(field.value) !== displayModel[field.name]" class="browse-id-badge">
                ID: {{ field.value }}
              </span>
            </div>
            <a-input
              v-else
              v-model:value="formModel[field.name]"
              :disabled="field.disabled || isReadOnly"
              class="custom-field"
            />
          </div>
        </div>
      </div>

      <div v-if="status === 'idle'" class="form-actions">
        <a-button :disabled="disabled" @click="handleCancel">
          {{ t('chat.formConfirm.cancel', '取消') }}
        </a-button>
        <a-button type="primary" :disabled="disabled" @click="handleConfirm">
          <template #icon><CheckCircleOutlined /></template>
          {{ t('chat.formConfirm.confirm', '确认创建') }}
        </a-button>
      </div>
    </template>
  </div>
</template>

<style scoped>
.form-confirm-card {
  margin: 12px 0;
  padding: 16px;
  background: var(--color-bg-subtle, #faf8f5);
  border: 1px solid var(--color-border-subtle, #ebe5df);
  border-radius: 8px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.03);
  transition: all 0.2s ease;
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
  padding-bottom: 10px;
  border-bottom: 1px dashed var(--color-border-subtle, #ebe5df);
}

.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-icon {
  font-size: 15px;
  color: var(--color-primary, #9a7342);
}

.header-title {
  font-weight: 600;
  font-size: 14px;
  color: var(--color-text-primary, #302d2b);
}

.workflow-id-tag {
  font-size: 12px;
}

.row-count-badge {
  font-size: 12px;
  color: var(--color-text-tertiary, #8c8680);
  background: var(--color-bg-card, #ffffff);
  padding: 2px 8px;
  border-radius: 10px;
  border: 1px solid var(--color-border-light, #ebe5df);
}

.status-hint {
  font-size: 12px;
  color: var(--color-text-tertiary, #8c8680);
}

.form-body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.form-row {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  font-size: 13px;
}

.field-label {
  width: 100px;
  flex-shrink: 0;
  color: var(--color-text-secondary, #605a54);
  padding-top: 5px;
  text-align: right;
  font-weight: 500;
}

.field-control {
  flex: 1;
  min-width: 0;
}

.w-full {
  width: 100%;
}

.custom-field {
  border-radius: 6px;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  margin-top: 16px;
  padding-top: 12px;
  border-top: 1px dashed var(--color-border-subtle, #ebe5df);
}

.table-container {
  overflow-x: auto;
}
</style>
