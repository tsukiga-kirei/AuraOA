<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { ExternalContextMount } from '~/types/external-context'

const { t } = useI18n()
const { authFetch } = useAuth()

type FieldOption = { value: string; label: string }

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

const testProcessId = ref('')
const testingContext = ref(false)
const contextPreview = ref('')

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
    testProcessId.value = ''
    contextPreview.value = ''
  }
})

// 规则生效范围选项：强制执行 / 默认开启 / 默认关闭
const scopeOptions = computed(() => [
  { value: 'mandatory', label: t('ruleEditor.mandatory') },
  { value: 'default_on', label: t('ruleEditor.defaultOn') },
  { value: 'default_off', label: t('ruleEditor.defaultOff') },
])

const fieldOptions = computed(() => props.fieldOptions || [])

const workflowMount = computed(() => form.value.context_mounts.find(m => m.type === 'workflow'))
const modelMount = computed(() => form.value.context_mounts.find(m => m.type === 'model'))

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
    basic_fields: mount?.workflow?.basic_fields || ['title', 'applicant', 'department', 'process_type', 'current_node', 'submit_time'],
    data_mode: mount?.workflow?.data_mode || 'none',
    target_process_type: mount?.workflow?.target_process_type || '',
    selected_fields: mount?.workflow?.selected_fields || [],
    fallback_strategy: mount?.workflow?.fallback_strategy || 'basic_with_notice',
    max_refs: mount?.workflow?.max_refs || 5,
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
    max_rows: mount?.model?.max_rows || 5,
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

const setWorkflowSelectedFields = (values: string[]) => {
  const mount = workflowMount.value
  if (mount?.workflow) mount.workflow.selected_fields = values
}

const testContext = async () => {
  if (!props.contextTestEndpoint) return
  if (!testProcessId.value.trim()) {
    message.warning('请输入用于测试的当前流程 requestid')
    return
  }
  testingContext.value = true
  contextPreview.value = ''
  try {
    const resp = await authFetch<{ context_text: string }>(props.contextTestEndpoint, {
      method: 'POST',
      body: { process_id: testProcessId.value.trim(), context_mounts: form.value.context_mounts },
    })
    contextPreview.value = resp.context_text || ''
  } catch (e: any) {
    message.error(`关联数据测试失败：${e?.message || '未知错误'}`)
  } finally {
    testingContext.value = false
  }
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

      <a-divider orientation="left">外部关联数据</a-divider>
      <div class="context-switches">
        <label class="context-switch">
          <a-switch :checked="!!workflowMount" @change="(checked: any) => toggleMount('workflow', !!checked)" />
          <span>关联流程</span>
        </label>
        <label class="context-switch">
          <a-switch :checked="!!modelMount" @change="(checked: any) => toggleMount('model', !!checked)" />
          <span>关联建模表</span>
        </label>
      </div>

      <div v-if="workflowMount" class="context-panel">
        <div class="context-panel-title">关联流程</div>
        <a-row :gutter="12">
          <a-col :span="12">
            <a-form-item label="来源字段">
              <a-select v-model:value="workflowMount.source_field" :options="fieldOptions" show-search placeholder="选择已引入字段" />
            </a-form-item>
          </a-col>
          <a-col :span="6">
            <a-form-item label="分隔符">
              <a-input v-model:value="workflowMount.source_splitter" />
            </a-form-item>
          </a-col>
          <a-col :span="6">
            <a-form-item label="最多流程数">
              <a-input-number v-model:value="workflowMount.workflow!.max_refs" :min="1" :max="20" style="width: 100%;" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="8">
            <a-form-item label="引用表单数据">
              <a-select v-model:value="workflowMount.workflow!.data_mode">
                <a-select-option value="none">不引用表单数据</a-select-option>
                <a-select-option value="all_fields">全部字段</a-select-option>
                <a-select-option value="selected_fields">指定字段</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="目标流程">
              <a-input v-model:value="workflowMount.workflow!.target_process_type" placeholder="可选，用于指定字段校验" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="兜底策略">
              <a-select v-model:value="workflowMount.workflow!.fallback_strategy">
                <a-select-option value="basic_with_notice">仅基础信息并提示</a-select-option>
                <a-select-option value="all_fields">尝试全部字段</a-select-option>
                <a-select-option value="ignore">忽略该流程</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item v-if="workflowMount.workflow?.data_mode === 'selected_fields'" label="引用流程字段">
          <a-select
            mode="multiple"
            :value="workflowMount.workflow.selected_fields"
            :options="fieldOptions"
            show-search
            placeholder="选择目标流程字段；字段名需与目标流程配置一致"
            @update:value="setWorkflowSelectedFields"
          />
        </a-form-item>
      </div>

      <div v-if="modelMount" class="context-panel">
        <div class="context-panel-title">关联建模表</div>
        <a-row :gutter="12">
          <a-col :span="12">
            <a-form-item label="来源字段">
              <a-select v-model:value="modelMount.source_field" :options="fieldOptions" show-search placeholder="选择已引入字段" />
            </a-form-item>
          </a-col>
          <a-col :span="12">
            <a-form-item label="建模表名">
              <a-input v-model:value="modelMount.model!.table_name" placeholder="例如 uf_supplier_license" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-row :gutter="12">
          <a-col :span="8">
            <a-form-item label="建模关联字段">
              <a-input v-model:value="modelMount.model!.join_field" placeholder="默认 id" />
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="查询方式">
              <a-select v-model:value="modelMount.model!.mode">
                <a-select-option value="exists">是否存在</a-select-option>
                <a-select-option value="count">存在条数</a-select-option>
                <a-select-option value="rows">当前行数据</a-select-option>
                <a-select-option value="custom_sql">自定义 SQL</a-select-option>
              </a-select>
            </a-form-item>
          </a-col>
          <a-col :span="8">
            <a-form-item label="最多返回行">
              <a-input-number v-model:value="modelMount.model!.max_rows" :min="1" :max="50" style="width: 100%;" />
            </a-form-item>
          </a-col>
        </a-row>
        <a-form-item v-if="modelMount.model?.mode === 'rows'" label="返回字段">
          <a-input :value="modelMount.model.return_fields?.join(',')" placeholder="逗号分隔；留空表示全部字段" @update:value="setModelReturnFields" />
        </a-form-item>
        <a-form-item v-if="modelMount.model?.mode === 'custom_sql'" label="自定义 SQL">
          <a-textarea v-model:value="modelMount.model.custom_sql" :rows="4" placeholder="仅允许 SELECT，并使用 :source_value 参数" />
        </a-form-item>
      </div>

      <div v-if="form.context_mounts.length" class="context-test">
        <a-input v-model:value="testProcessId" placeholder="输入当前流程 requestid 测试关联数据" style="max-width: 320px;" />
        <a-button :loading="testingContext" @click="testContext">测试</a-button>
      </div>
      <pre v-if="contextPreview" class="context-preview">{{ contextPreview }}</pre>
    </a-form>
  </a-modal>
</template>

<style scoped>
.context-switches {
  display: flex;
  gap: 16px;
  margin-bottom: 12px;
}
.context-switch {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--color-text-secondary);
}
.context-panel {
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  padding: 12px;
  margin-bottom: 12px;
  background: var(--color-bg-page);
}
.context-panel-title {
  font-weight: 600;
  margin-bottom: 10px;
  color: var(--color-text-primary);
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
</style>
