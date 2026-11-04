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
import {
  normalizeFormConfirmPayload,
  type FormConfirmField,
  type FormConfirmPayload,
} from '~/utils/chatUiProtocol'

const props = defineProps<{
  // 解析器已归一化；这里仍做一次幂等归一化，保证历史消息与外部直传数据也能正确渲染
  data: Record<string, any>
  disabled?: boolean
}>()

const emit = defineEmits<{
  (e: 'send', text: string): void
}>()

const { t } = useI18n()

const payload = computed<FormConfirmPayload>(() => normalizeFormConfirmPayload(props.data))

const status = ref<'idle' | 'confirmed' | 'cancelled'>('idle')
const formModel = reactive<Record<string, any>>({})
const displayModel = reactive<Record<string, string>>({})

// 初始化表单字段默认值（兼容浏览按钮 display_value 与底值 value）
const initFormValues = () => {
  Object.keys(formModel).forEach(key => delete formModel[key])
  Object.keys(displayModel).forEach(key => delete displayModel[key])
  payload.value.fields.forEach((field: FormConfirmField) => {
    const initial = field.value ?? field.display_value ?? ''
    formModel[field.name] = field.type === 'number' && initial === '' ? undefined : initial
    displayModel[field.name] = field.display_value ?? (field.value !== undefined ? String(field.value) : '')
  })
}

watch(payload, initFormValues, { immediate: true })

const isReadOnly = computed(() => props.disabled || status.value !== 'idle')

const handleConfirm = () => {
  if (isReadOnly.value) return
  status.value = 'confirmed'
  const wfId = payload.value.workflow_id || ''
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
const tableColumns = computed(() => payload.value.columns)

const detailPagination = (rowCount: number) => (rowCount > 5 ? { pageSize: 5, size: 'small' as const } : false)

const exportCsv = () => {
  const rows = payload.value.rows || []
  const cols = payload.value.columns || []
  if (!rows.length) {
    message.warning(t('common.noData', '暂无可导出的数据'))
    return
  }

  const colHeaders: string[] = cols.map((c: any) => (typeof c === 'string' ? c : c.title || c.dataIndex))
  const colKeys: string[] = cols.map((c: any) => (typeof c === 'string' ? c : c.dataIndex || c.key))

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
  link.download = `${payload.value.title || payload.value.workflow_name || 'query_result'}_${Date.now()}.csv`
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  URL.revokeObjectURL(url)
  message.success(t('common.exported', '已成功导出 CSV'))
}
</script>

<template>
  <div class="form-confirm-card" :class="[`form-confirm-card--${payload.variant}`]">
    <!-- 1. 表格变体 (data_table) -->
    <template v-if="payload.variant === 'data_table'">
      <div class="card-header">
        <div class="header-left">
          <TableOutlined class="header-icon" />
          <span class="header-title">{{ payload.title || payload.workflow_name || t('chat.formConfirm.title', '数据查询结果') }}</span>
          <span v-if="payload.rows.length" class="row-count-badge">{{ payload.rows.length }} 条记录</span>
        </div>
        <div class="header-right">
          <a-button size="small" @click="exportCsv">
            <template #icon><DownloadOutlined /></template>
            {{ t('chat.formConfirm.exportCsv', '导出 CSV') }}
          </a-button>
        </div>
      </div>
      <p v-if="payload.description" class="card-description">{{ payload.description }}</p>
      <div class="table-container">
        <a-table
          :data-source="payload.rows"
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
          <span class="header-title">{{ payload.workflow_name || payload.title || t('chat.formConfirm.title', '流程预填信息确认') }}</span>
          <a-tag v-if="payload.workflow_id" color="blue" class="workflow-id-tag">ID: {{ payload.workflow_id }}</a-tag>
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

      <p v-if="payload.description" class="card-description">{{ payload.description }}</p>

      <div class="form-body">
        <div v-for="field in payload.fields" :key="field.name" class="form-row">
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
              :options="field.options || []"
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

      <!-- 内嵌明细子表（报销单等带明细的流程），只读展示 -->
      <div v-for="(table, tIdx) in payload.details" :key="`detail-${tIdx}`" class="detail-block">
        <div class="detail-title">{{ table.title || t('chat.formConfirm.detailTable', '明细') }}</div>
        <a-table
          :data-source="table.rows"
          :columns="table.columns"
          size="small"
          :pagination="detailPagination(table.rows.length)"
          :scroll="{ x: 'max-content' }"
          bordered
        />
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

.card-description {
  margin: 0 0 12px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--color-text-secondary, #605a54);
}

.detail-block {
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px dashed var(--color-border-subtle, #ebe5df);
}

.detail-title {
  margin-bottom: 8px;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-secondary, #605a54);
}
</style>
