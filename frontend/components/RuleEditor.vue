<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { ExternalContextMount } from '~/types/external-context'

const { t } = useI18n()
const { authFetch } = useAuth()

type FieldOption = { value: string; label: string }
type ProcessFields = {
  main_fields?: { field_key: string; field_name: string; field_type?: string }[]
  detail_tables?: { table_name: string; table_label?: string; fields?: { field_key: string; field_name: string; field_type?: string }[] }[]
}
type ProcessInfo = {
  workflow_id?: string
  process_type: string
  process_name: string
  process_type_label?: string
  main_table?: string
}

// props：弹窗开关 / 待编辑的规则数据（新增时为 null）
const props = defineProps<{
  open: boolean
  rule?: {
    id?: string
    rule_content: string
    rule_scope: string
    related_flow?: boolean
    context_enabled?: boolean
    context_mounts?: ExternalContextMount[]
  } | null
  fieldOptions?: FieldOption[]
  contextTestEndpoint?: string
  workflowFieldsEndpoint?: string
  workflowSearchEndpoint?: string
}>()

// emit：关闭弹窗 / 提交保存的规则数据
const emit = defineEmits<{
  close: []
  save: [rule: any]
}>()

const form = ref({
  rule_content: '',
  rule_scope: 'default_on',
  related_flow: false,
  context_enabled: false,
  context_mounts: [] as ExternalContextMount[],
})

const testingWorkflowContext = ref(false)
const testingModelContext = ref(false)
const loadingWorkflowFields = ref(false)
const searchingWorkflows = ref(false)
const hasSearchedWorkflows = ref(false)
const contextPreview = ref<Record<string, string>>({})
const workflowTargetFieldOptions = ref<FieldOption[]>([])
const targetWorkflowConfigOpen = ref(false)
const targetWorkflowKeyword = ref('')
const targetWorkflowRows = ref<ProcessInfo[]>([])
const selectedTargetWorkflowID = ref('')
const targetWorkflowDraftFields = ref<string[]>([])
const targetWorkflowDraftFallback = ref<'basic_with_notice' | 'all_fields' | 'ignore'>('basic_with_notice')

// 弹窗打开时初始化表单：编辑模式填充已有数据，新增模式重置为默认值
watch(() => props.open, (isOpen) => {
  if (isOpen) {
    if (props.rule) {
      form.value = {
        rule_content: props.rule.rule_content,
        rule_scope: props.rule.rule_scope,
        related_flow: (props.rule as any).related_flow ?? false,
        context_enabled: (props.rule as any).context_enabled ?? false,
        context_mounts: normalizeMounts((props.rule as any).context_mounts || []),
      }
    } else {
      form.value = {
        rule_content: '',
        rule_scope: 'default_on',
        related_flow: false,
        context_enabled: false,
        context_mounts: [],
      }
    }
    contextPreview.value = {}
    workflowTargetFieldOptions.value = []
  }
})

// 规则生效范围选项：强制执行 / 默认开启 / 默认关闭
const scopeOptions = computed(() => [
  { value: 'mandatory', label: t('ruleEditor.mandatory') },
  { value: 'default_on', label: t('ruleEditor.defaultOn') },
  { value: 'default_off', label: t('ruleEditor.defaultOff') },
])

const fieldOptions = computed(() => props.fieldOptions || [])
const tableNameSQLVariable = '{{table_name}}'
const joinFieldSQLVariable = '{{join_field}}'
const customSQLPlaceholder = computed(() => t('externalContext.customSQLPlaceholder', [tableNameSQLVariable, joinFieldSQLVariable]))

const workflowBasicOptions = computed(() => [
  { label: t('externalContext.basic.archived'), value: 'archived' },
  { label: t('externalContext.basic.title'), value: 'title' },
  { label: t('externalContext.basic.applicant'), value: 'applicant' },
  { label: t('externalContext.basic.department'), value: 'department' },
  { label: t('externalContext.basic.processType'), value: 'process_type' },
  { label: t('externalContext.basic.currentNode'), value: 'current_node' },
  { label: t('externalContext.basic.submitTime'), value: 'submit_time' },
])
const workflowBasicAllValues = ['archived', 'title', 'applicant', 'department', 'process_type', 'current_node', 'submit_time']

const isWorkflowBasicAllSelected = computed(() => {
  const fields = workflowMount.value?.workflow?.basic_fields || []
  return workflowBasicAllValues.every(v => fields.includes(v))
})

const toggleWorkflowBasicAll = (checked: boolean) => {
  const mount = workflowMount.value
  if (!mount?.workflow) return
  mount.workflow.basic_fields = checked ? [...workflowBasicAllValues] : []
  mount.workflow.include_basic = checked
}

const workflowMount = computed(() => form.value.context_mounts.find(m => m.type === 'workflow'))
const modelMount = computed(() => form.value.context_mounts.find(m => m.type === 'model'))
const workflowDataModeForUI = (mount?: Partial<ExternalContextMount>) =>
  mount?.workflow?.target_process_type || mount?.workflow?.target_workflow_id ? 'selected_fields' : 'all_fields'

const normalizeMounts = (mounts: ExternalContextMount[]) => {
  return mounts.map(m => {
    if (m.type === 'workflow') return normalizeWorkflowMount(m)
    if (m.type === 'model') return normalizeModelMount(m)
    return m
  })
}

const normalizeWorkflowMount = (mount?: Partial<ExternalContextMount>): ExternalContextMount => ({
  type: 'workflow',
  enabled: mount?.enabled ?? true,
  name: mount?.name || '关联流程',
  source_field: mount?.source_field || '',
  source_splitter: mount?.source_splitter || ',',
  workflow: {
    include_basic: mount?.workflow?.include_basic ?? true,
    basic_fields: mount?.workflow?.basic_fields || ['archived', 'title', 'applicant', 'department', 'process_type', 'current_node', 'submit_time'],
    data_mode: workflowDataModeForUI(mount),
    target_process_type: mount?.workflow?.target_process_type || '',
    target_workflow_id: mount?.workflow?.target_workflow_id || '',
    target_process_label: mount?.workflow?.target_process_label || '',
    target_main_table: mount?.workflow?.target_main_table || '',
    selected_fields: mount?.workflow?.selected_fields || [],
    fallback_strategy: mount?.workflow?.fallback_strategy || 'basic_with_notice',
    max_rows: mount?.workflow?.max_rows || 20,
  },
})

const normalizeModelMount = (mount?: Partial<ExternalContextMount>): ExternalContextMount => ({
  type: 'model',
  enabled: mount?.enabled ?? true,
  name: mount?.name || '关联建模表',
  source_field: mount?.source_field || '',
  model: {
    table_name: mount?.model?.table_name || '',
    join_field: mount?.model?.join_field || 'id',
    mode: mount?.model?.mode || 'exists',
    return_fields: mount?.model?.return_fields || [],
    max_rows: mount?.model?.max_rows ?? 5,
    order_by: mount?.model?.order_by || '',
    order_dir: mount?.model?.order_dir || 'ASC',
    custom_sql: mount?.model?.custom_sql || '',
  },
})

const toggleMount = (type: 'workflow' | 'model', checked: boolean) => {
  const idx = form.value.context_mounts.findIndex(m => m.type === type)
  if (checked && idx < 0) {
    form.value.context_mounts.push(type === 'workflow' ? normalizeWorkflowMount() : normalizeModelMount())
  } else if (!checked && idx >= 0) {
    form.value.context_mounts.splice(idx, 1)
  }
  form.value.context_enabled = form.value.context_mounts.length > 0
}

const fieldListFromText = (text?: string) => (text || '').split(',').map(v => v.trim()).filter(Boolean)

const setModelReturnFields = (value: string) => {
  const mount = modelMount.value
  if (mount?.model) mount.model.return_fields = fieldListFromText(value)
}

const setModelMaxRows = (value: number | string | null) => {
  const mount = modelMount.value
  const rows = Number(value)
  if (mount?.model && Number.isFinite(rows) && rows > 0) mount.model.max_rows = rows
}

const setModelAllRows = (checked: boolean) => {
  const mount = modelMount.value
  if (mount?.model) mount.model.max_rows = checked ? -1 : 5
}

const testContext = async (mount: ExternalContextMount, key: 'workflow' | 'model') => {
  if (!props.contextTestEndpoint) return
  if (key === 'workflow') testingWorkflowContext.value = true
  if (key === 'model') testingModelContext.value = true
  contextPreview.value[key] = ''
  try {
    const resp = await authFetch<{ context_text: string }>(props.contextTestEndpoint, {
      method: 'POST',
      body: { context_mounts: [mount] },
    })
    contextPreview.value[key] = resp.context_text || ''
  } catch (e: any) {
    message.error(t('externalContext.testFailed', [e?.message || t('externalContext.unknownError')]))
  } finally {
    if (key === 'workflow') testingWorkflowContext.value = false
    if (key === 'model') testingModelContext.value = false
  }
}

const loadWorkflowTargetFields = async (target?: ProcessInfo, options?: { silent?: boolean }) => {
  const mount = workflowMount.value
  const processType = target?.process_name || target?.process_type || mount?.workflow?.target_process_type?.trim()
  const workflowID = target?.workflow_id || mount?.workflow?.target_workflow_id || ''
  if (!props.workflowFieldsEndpoint || !mount?.workflow || (!processType && !workflowID)) {
    message.warning(t('externalContext.selectTargetFirst'))
    return
  }
  loadingWorkflowFields.value = true
  try {
    const fields = await authFetch<ProcessFields>(props.workflowFieldsEndpoint, {
      method: 'POST',
      body: { process_type: processType || '', workflow_id: workflowID },
    })
    workflowTargetFieldOptions.value = buildFieldOptions(fields)
    if (!workflowTargetFieldOptions.value.length) message.warning(t('externalContext.noTargetFields'))
    else if (!options?.silent) message.success(t('externalContext.targetFieldsLoaded'))
  } catch (e: any) {
    message.error(t('externalContext.targetFieldsLoadFailed', [e?.message || t('externalContext.unknownError')]))
  } finally {
    loadingWorkflowFields.value = false
  }
}

const buildFieldOptions = (fields: ProcessFields): FieldOption[] => {
  const out: FieldOption[] = []
  for (const f of fields.main_fields || []) {
    out.push({ label: `${f.field_name}（${t('externalContext.mainTable')}）`, value: `main:${f.field_key}` })
  }
  ;(fields.detail_tables || []).forEach((dt, idx) => {
    const label = dt.table_label || t('externalContext.detailTable', [idx + 1])
    for (const f of dt.fields || []) {
      out.push({ label: `${f.field_name}（${label}）`, value: `${dt.table_name}:${f.field_key}` })
    }
  })
  return out
}

const setWorkflowTargetMode = (specified: boolean) => {
  const mount = workflowMount.value
  if (!mount?.workflow) return
  if (specified) {
    mount.workflow.data_mode = 'selected_fields'
    openTargetWorkflowConfigModal()
  } else {
    mount.workflow.target_process_type = ''
    mount.workflow.target_workflow_id = ''
    mount.workflow.target_process_label = ''
    mount.workflow.target_main_table = ''
    mount.workflow.selected_fields = []
    mount.workflow.data_mode = 'all_fields'
    workflowTargetFieldOptions.value = []
  }
}

const openTargetWorkflowConfigModal = async () => {
  const mount = workflowMount.value
  const wf = mount?.workflow
  if (!wf) return
  targetWorkflowKeyword.value = ''
  targetWorkflowRows.value = []
  hasSearchedWorkflows.value = false
  selectedTargetWorkflowID.value = wf.target_workflow_id || wf.target_process_type || ''
  targetWorkflowDraftFields.value = [...(wf.selected_fields || [])]
  targetWorkflowDraftFallback.value = (wf.fallback_strategy as any) || 'basic_with_notice'
  targetWorkflowConfigOpen.value = true
  if ((wf.target_process_type || wf.target_workflow_id) && !workflowTargetFieldOptions.value.length) {
    await loadWorkflowTargetFields(undefined, { silent: true })
  }
}

const selectTargetWorkflowInModal = async (item: ProcessInfo) => {
  const nextID = item.workflow_id || item.process_name || item.process_type
  if (selectedTargetWorkflowID.value === nextID) return
  selectedTargetWorkflowID.value = nextID
  targetWorkflowDraftFields.value = []
  await loadWorkflowTargetFields(item, { silent: true })
}

const searchTargetWorkflows = async () => {
  if (!props.workflowSearchEndpoint) {
    message.warning(t('externalContext.searchEndpointMissing'))
    return
  }
  const keyword = targetWorkflowKeyword.value.trim()
  if (!keyword) {
    message.warning(t('externalContext.searchKeywordRequired'))
    targetWorkflowRows.value = []
    hasSearchedWorkflows.value = false
    return
  }
  searchingWorkflows.value = true
  try {
    targetWorkflowRows.value = await authFetch<ProcessInfo[]>(props.workflowSearchEndpoint, {
      method: 'POST',
      body: { keyword },
    })
    hasSearchedWorkflows.value = true
  } catch (e: any) {
    message.error(t('externalContext.searchFailed', [e?.message || t('externalContext.unknownError')]))
  } finally {
    searchingWorkflows.value = false
  }
}

const targetWorkflowDisplayName = (item: ProcessInfo) =>
  item.process_name || item.process_type || (item.workflow_id && item.workflow_id !== '0'
    ? t('externalContext.workflowWithId', [item.workflow_id])
    : t('externalContext.unnamedWorkflow'))

const confirmTargetWorkflowConfig = async () => {
  const mount = workflowMount.value
  if (!mount?.workflow) {
    message.warning(t('externalContext.searchAndSelectTarget'))
    return Promise.reject()
  }
  const row = targetWorkflowRows.value.find(item =>
    (item.workflow_id || item.process_name || item.process_type) === selectedTargetWorkflowID.value
  )
  if (row) {
    mount.workflow.target_process_type = row.process_name || row.process_type
    mount.workflow.target_workflow_id = row.workflow_id || ''
    mount.workflow.target_process_label = row.process_type_label || ''
    mount.workflow.target_main_table = row.main_table || ''
  } else if (!mount.workflow.target_process_type && !mount.workflow.target_workflow_id) {
    message.warning(t('externalContext.searchAndSelectTarget'))
    return Promise.reject()
  }
  mount.workflow.data_mode = 'selected_fields'
  mount.workflow.selected_fields = [...targetWorkflowDraftFields.value]
  mount.workflow.fallback_strategy = targetWorkflowDraftFallback.value
  targetWorkflowConfigOpen.value = false
}

const targetWorkflowSummary = computed(() => {
  const wf = workflowMount.value?.workflow
  if (!wf?.target_process_type && !wf?.target_workflow_id) return ''
  return [wf.target_process_type, wf.target_process_label, wf.target_main_table].filter(Boolean).join(' / ')
})

const targetWorkflowConfigSummary = computed(() => {
  const wf = workflowMount.value?.workflow
  if (!wf || wf.data_mode !== 'selected_fields') return ''
  const target = targetWorkflowSummary.value
  const fieldCount = wf.selected_fields?.length || 0
  if (!target) return t('externalContext.notConfigured')
  return `${target} · ${t('externalContext.fieldCount', [fieldCount])}`
})

const insertCustomSQLVariable = (variable: string) => {
  const mount = modelMount.value
  if (!mount?.model) return
  const current = mount.model.custom_sql || ''
  mount.model.custom_sql = current ? `${current} ${variable}` : variable
}

// 提交表单数据给父组件处理
const handleSave = () => {
  form.value.context_enabled = form.value.context_mounts.length > 0
  emit('save', { ...form.value })
}
</script>

<template>
  <a-modal
    :open="open"
    :title="rule ? t('ruleEditor.editRule') : t('ruleEditor.addRule')"
    @cancel="emit('close')"
    @ok="handleSave"
    :okText="t('ruleEditor.save')"
    :cancelText="t('ruleEditor.cancel')"
    :width="760"
  >
    <a-form layout="vertical" style="margin-top: 16px;">
      <a-form-item :label="t('ruleEditor.ruleContent')">
        <a-textarea
          v-model:value="form.rule_content"
          :rows="3"
          :placeholder="t('ruleEditor.ruleContentPlaceholder')"
          size="large"
        />
      </a-form-item>
      <a-form-item :label="t('ruleEditor.ruleLevel')">
        <a-radio-group v-model:value="form.rule_scope" button-style="solid">
          <a-radio-button v-for="opt in scopeOptions" :key="opt.value" :value="opt.value">
            {{ opt.label }}
          </a-radio-button>
        </a-radio-group>
      </a-form-item>
      <a-form-item :label="t('ruleEditor.relatedFlow')">
        <div style="display: flex; align-items: center; gap: 12px;">
          <a-switch v-model:checked="form.related_flow" />
          <span style="font-size: 13px; color: var(--color-text-tertiary);">{{ t('ruleEditor.relatedFlowDesc') }}</span>
        </div>
      </a-form-item>

      <a-divider orientation="left">{{ t('externalContext.title') }}</a-divider>
      <div class="context-toggle-row">
        <button type="button" class="context-toggle-card" :class="{ active: !!workflowMount }" @click="toggleMount('workflow', !workflowMount)">
          <span class="context-toggle-title">{{ t('externalContext.workflow') }}</span>
          <span @click.stop>
            <a-switch :checked="!!workflowMount" @change="(checked: any) => toggleMount('workflow', !!checked)" />
          </span>
        </button>
        <button type="button" class="context-toggle-card" :class="{ active: !!modelMount }" @click="toggleMount('model', !modelMount)">
          <span class="context-toggle-title">{{ t('externalContext.model') }}</span>
          <span @click.stop>
            <a-switch :checked="!!modelMount" @change="(checked: any) => toggleMount('model', !!checked)" />
          </span>
        </button>
      </div>

      <div v-if="workflowMount" class="context-panel">
        <div class="context-panel-head">
          <div>
            <div class="context-panel-title">{{ t('externalContext.workflow') }}</div>
            <div class="context-panel-subtitle">{{ t('externalContext.workflowDesc') }}</div>
          </div>
        </div>
        <a-row :gutter="12">
          <a-col :span="24">
            <a-form-item :label="t('externalContext.sourceField')">
              <a-select v-model:value="workflowMount.source_field" :options="fieldOptions" show-search :placeholder="t('externalContext.sourceFieldPlaceholder')" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item class="basic-fields-item" :colon="false">
          <template #label>
            <div class="basic-fields-label-row">
              <span>{{ t('externalContext.basicInfo') }}</span>
              <a-checkbox
                :checked="isWorkflowBasicAllSelected"
                :indeterminate="!!workflowMount.workflow!.basic_fields?.length && !isWorkflowBasicAllSelected"
                @change="(e: any) => toggleWorkflowBasicAll(!!e?.target?.checked)"
              >
                {{ t('externalContext.selectAll') }}
              </a-checkbox>
            </div>
          </template>
          <div class="basic-fields-panel">
            <a-checkbox-group
              v-model:value="workflowMount.workflow!.basic_fields"
              class="basic-fields-group"
              @change="(vals: any) => {
                if (workflowMount?.workflow) workflowMount.workflow.include_basic = Array.isArray(vals) && vals.length > 0
              }"
            >
              <label
                v-for="opt in workflowBasicOptions"
                :key="opt.value"
                class="basic-field-chip"
                :class="{ active: workflowMount.workflow!.basic_fields?.includes(opt.value) }"
              >
                <a-checkbox :value="opt.value">{{ opt.label }}</a-checkbox>
              </label>
            </a-checkbox-group>
            <div class="basic-fields-hint">{{ t('externalContext.basicInfoHint') }}</div>
          </div>
        </a-form-item>
        <a-form-item :label="t('externalContext.workflowFormData')">
          <div class="context-mode-switch">
            <button type="button" :class="{ active: workflowMount.workflow?.data_mode !== 'selected_fields' }" @click="setWorkflowTargetMode(false)">
              {{ t('externalContext.allFieldsAuto') }}
            </button>
            <button type="button" :class="{ active: workflowMount.workflow?.data_mode === 'selected_fields' }" @click="setWorkflowTargetMode(true)">
              {{ t('externalContext.selectTargetAndFields') }}
            </button>
          </div>
        </a-form-item>
        <div v-if="workflowMount.workflow?.data_mode === 'selected_fields'" class="target-flow-compact">
          <div class="target-flow-meta">
            <div class="target-flow-label">{{ t('externalContext.targetWorkflow') }}</div>
            <div class="target-flow-value">{{ targetWorkflowConfigSummary || t('externalContext.notConfigured') }}</div>
          </div>
          <a-button @click="openTargetWorkflowConfigModal">{{ t('externalContext.configure') }}</a-button>
        </div>
        <div class="context-actions">
          <a-button :loading="testingWorkflowContext" @click="testContext(workflowMount, 'workflow')">{{ t('externalContext.test') }}</a-button>
        </div>
        <pre v-if="contextPreview.workflow" class="context-preview">{{ contextPreview.workflow }}</pre>
      </div>

      <div v-if="modelMount" class="context-panel">
        <div class="context-panel-title">{{ t('externalContext.model') }}</div>
        <a-row :gutter="12">
          <a-col :span="12">
            <a-form-item :label="t('externalContext.sourceField')">
              <a-select v-model:value="modelMount.source_field" :options="fieldOptions" show-search :placeholder="t('externalContext.sourceFieldPlaceholder')" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item :label="t('externalContext.modelTable')">
              <a-input v-model:value="modelMount.model!.table_name" :placeholder="t('externalContext.modelTablePlaceholder')" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="8">
            <a-form-item :label="t('externalContext.modelJoinField')">
              <a-input v-model:value="modelMount.model!.join_field" :placeholder="t('externalContext.defaultId')" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item :label="t('externalContext.queryMode')">
              <a-select v-model:value="modelMount.model!.mode">
                <a-select-option value="exists">{{ t('externalContext.mode.exists') }}</a-select-option>
                <a-select-option value="count">{{ t('externalContext.mode.count') }}</a-select-option>
                <a-select-option value="rows">{{ t('externalContext.mode.rows') }}</a-select-option>
                <a-select-option value="custom_sql">{{ t('externalContext.mode.customSQL') }}</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item v-if="modelMount.model?.mode === 'rows'" :label="t('externalContext.modelReturnFields')">
          <a-input :value="modelMount.model.return_fields?.join(',')" :placeholder="t('externalContext.modelReturnFieldsPlaceholder')" @update:value="setModelReturnFields" />
        </a-form-item>
        <a-form-item v-if="modelMount.model?.mode === 'rows'" :label="t('externalContext.maxRows')">
          <a-space>
            <a-input-number
              :value="modelMount.model!.max_rows === -1 ? undefined : modelMount.model!.max_rows"
              :min="1"
              :max="50"
              :disabled="modelMount.model!.max_rows === -1"
              style="width: 160px;"
              @update:value="setModelMaxRows"
            />
            <a-checkbox :checked="modelMount.model!.max_rows === -1" @change="(e: any) => setModelAllRows(!!e?.target?.checked)">
              {{ t('common.all') }}
            </a-checkbox>
          </a-space>
        </a-form-item>
        <a-form-item v-if="modelMount.model?.mode === 'custom_sql'" :label="t('externalContext.customSQL')">
          <div class="sql-variable-bar">
            <span>{{ t('externalContext.insertVariable') }}：</span>
            <a-button size="small" @click="insertCustomSQLVariable(tableNameSQLVariable)">{{ tableNameSQLVariable }}</a-button>
            <a-button size="small" @click="insertCustomSQLVariable(joinFieldSQLVariable)">{{ joinFieldSQLVariable }}</a-button>
          </div>
          <a-textarea v-model:value="modelMount.model.custom_sql" :rows="4" :placeholder="customSQLPlaceholder" />
        </a-form-item>
        <div class="context-actions">
          <a-button :loading="testingModelContext" @click="testContext(modelMount, 'model')">{{ t('externalContext.testModel') }}</a-button>
        </div>
        <pre v-if="contextPreview.model" class="context-preview">{{ contextPreview.model }}</pre>
      </div>
    </a-form>
  </a-modal>

  <a-modal
    v-model:open="targetWorkflowConfigOpen"
    :title="t('externalContext.configureTarget')"
    :width="720"
    :ok-text="t('common.confirm')"
    :cancel-text="t('common.cancel')"
    :confirm-loading="loadingWorkflowFields"
    @ok="confirmTargetWorkflowConfig"
  >
    <div class="workflow-config-modal">
      <div v-if="targetWorkflowSummary && !targetWorkflowRows.length" class="workflow-current">
        <div class="target-flow-label">{{ t('externalContext.currentTarget') }}</div>
        <div class="target-flow-value">{{ targetWorkflowSummary }}</div>
      </div>
      <div class="workflow-picker">
        <div class="workflow-picker-search">
          <a-input
            v-model:value="targetWorkflowKeyword"
            :placeholder="t('externalContext.searchPlaceholder')"
            allow-clear
            @press-enter="searchTargetWorkflows"
          />
          <a-button type="primary" :loading="searchingWorkflows" @click="searchTargetWorkflows">
            {{ t('common.search') }}
          </a-button>
        </div>
        <a-spin :spinning="searchingWorkflows">
          <div class="workflow-result-list workflow-result-list--modal">
            <button
              v-for="item in targetWorkflowRows"
              :key="item.workflow_id || item.process_name || item.process_type"
              type="button"
              class="workflow-result-item"
              :class="{ active: selectedTargetWorkflowID === (item.workflow_id || item.process_name || item.process_type) }"
              @click="selectTargetWorkflowInModal(item)"
            >
              <span class="workflow-result-radio"></span>
              <span class="workflow-result-main">
                <span class="workflow-result-name">{{ targetWorkflowDisplayName(item) }}</span>
                <span class="workflow-result-meta">
                  <span v-if="item.process_type_label">{{ item.process_type_label }}</span>
                  <span v-if="item.main_table">{{ item.main_table }}</span>
                  <span v-if="item.workflow_id && item.workflow_id !== '0'">ID {{ item.workflow_id }}</span>
                </span>
              </span>
            </button>
            <a-empty
              v-if="!targetWorkflowRows.length && !searchingWorkflows"
              :description="hasSearchedWorkflows ? t('externalContext.noSearchResult') : t('externalContext.searchEmptyHint')"
            />
          </div>
        </a-spin>
      </div>
      <a-form layout="vertical" class="workflow-config-form">
        <a-form-item :label="t('externalContext.workflowFields')">
          <a-select
            v-model:value="targetWorkflowDraftFields"
            :options="workflowTargetFieldOptions"
            mode="multiple"
            show-search
            option-filter-prop="label"
            :placeholder="t('externalContext.workflowFieldsPlaceholder')"
          />
        </a-form-item>
        <a-form-item :label="t('externalContext.typeMismatch')">
          <a-select v-model:value="targetWorkflowDraftFallback">
            <a-select-option value="basic_with_notice">{{ t('externalContext.fallback.basic') }}</a-select-option>
            <a-select-option value="all_fields">{{ t('externalContext.fallback.allFields') }}</a-select-option>
            <a-select-option value="ignore">{{ t('externalContext.fallback.ignore') }}</a-select-option>
          </a-select>
        </a-form-item>
      </a-form>
    </div>
  </a-modal>
</template>

<style scoped>
.context-toggle-row {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 14px;
}
.context-toggle-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 48px;
  padding: 0 14px;
  border: 1px solid var(--color-border-light);
  border-radius: 8px;
  background: var(--color-bg-card);
  color: var(--color-text-primary);
  cursor: pointer;
  transition: border-color .18s ease, background .18s ease, box-shadow .18s ease;
}
.context-toggle-card.active {
  border-color: rgba(91, 99, 211, .45);
  background: rgba(91, 99, 211, .06);
  box-shadow: 0 6px 18px rgba(20, 30, 70, .06);
}
.context-toggle-title {
  font-size: 14px;
  font-weight: 600;
}
.context-panel {
  border: 1px solid var(--color-border-light);
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 14px;
  background: var(--color-bg-card);
}
.context-panel-head {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 12px;
  margin-bottom: 14px;
}
.context-panel-title {
  font-size: 15px;
  font-weight: 700;
  color: var(--color-text-primary);
}
.context-panel-subtitle {
  margin-top: 4px;
  font-size: 12px;
  color: var(--color-text-tertiary);
}
.basic-fields-item :deep(.ant-form-item-label) {
  width: 100%;
}
.basic-fields-item :deep(.ant-form-item-label > label) {
  width: 100%;
}
.basic-fields-label-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  width: 100%;
}
.basic-fields-panel {
  padding: 10px 12px;
  border: 1px solid var(--color-border-light);
  border-radius: 8px;
  background: var(--color-bg-page);
}
.basic-fields-group {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.basic-field-chip {
  display: inline-flex;
  align-items: center;
  margin: 0;
  padding: 4px 10px;
  border: 1px solid var(--color-border-light);
  border-radius: 999px;
  background: var(--color-bg-card);
  cursor: pointer;
  transition: border-color .15s ease, background .15s ease;
}
.basic-field-chip:hover {
  border-color: rgba(91, 99, 211, .35);
}
.basic-field-chip.active {
  border-color: rgba(91, 99, 211, .45);
  background: rgba(91, 99, 211, .08);
}
.basic-field-chip :deep(.ant-checkbox-wrapper) {
  margin: 0;
}
.basic-fields-hint {
  margin-top: 8px;
  font-size: 12px;
  color: var(--color-text-tertiary);
}
.context-mode-switch {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  padding: 4px;
  border: 1px solid var(--color-border-light);
  border-radius: 8px;
  background: var(--color-bg-page);
}
.context-mode-switch button {
  min-height: 38px;
  border: 0;
  border-radius: 6px;
  background: transparent;
  color: var(--color-text-secondary);
  font-weight: 600;
  cursor: pointer;
}
.context-mode-switch button.active {
  background: var(--color-bg-card);
  color: var(--color-primary);
  box-shadow: 0 4px 12px rgba(20, 30, 70, .08);
}
.target-flow-compact {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  margin-bottom: 12px;
  border: 1px dashed var(--color-border);
  border-radius: 8px;
  background: var(--color-bg-page);
}
.target-flow-meta {
  min-width: 0;
  flex: 1;
}
.target-flow-label {
  font-size: 12px;
  color: var(--color-text-tertiary);
}
.target-flow-value {
  margin-top: 4px;
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
}
.workflow-current {
  margin-bottom: 12px;
  padding: 10px 12px;
  border: 1px solid var(--color-border-light);
  border-radius: 8px;
  background: var(--color-bg-page);
}
.workflow-config-form {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--color-border-light);
}
.workflow-result-list--modal {
  max-height: 240px;
}
.context-actions {
  display: flex;
  margin-top: 10px;
}
.sql-variable-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  font-size: 12px;
  color: var(--color-text-tertiary);
}
.context-test {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-top: 8px;
}
.context-preview {
  margin-top: 10px;
  max-height: 220px;
  overflow: auto;
  white-space: pre-wrap;
  background: var(--color-bg-page);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-sm);
  padding: 10px;
  font-size: 12px;
  line-height: 1.6;
}
.workflow-picker-search {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}
.workflow-picker-search :deep(.ant-input-affix-wrapper) {
  flex: 1;
  min-width: 0;
}
.workflow-result-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-height: 420px;
  overflow: auto;
}
.workflow-result-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 12px;
  border: 1px solid var(--color-border-light);
  border-radius: 8px;
  background: var(--color-bg-card);
  text-align: left;
  cursor: pointer;
}
.workflow-result-item.active {
  border-color: rgba(91, 99, 211, .55);
  background: rgba(91, 99, 211, .06);
}
.workflow-result-radio {
  width: 16px;
  height: 16px;
  border: 2px solid var(--color-border);
  border-radius: 50%;
  flex: 0 0 auto;
}
.workflow-result-item.active .workflow-result-radio {
  border: 5px solid var(--color-primary);
}
.workflow-result-main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.workflow-result-name {
  font-weight: 700;
  color: var(--color-text-primary);
}
.workflow-result-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  font-size: 12px;
  color: var(--color-text-tertiary);
}
</style>
