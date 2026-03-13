<script setup lang="ts">
import {
  AppstoreOutlined,
  AuditOutlined,
  CheckOutlined,
  ClockCircleOutlined,
  CloseOutlined,
  DashboardOutlined,
  DatabaseOutlined,
  DeleteOutlined,
  EditOutlined,
  FileTextOutlined,
  FolderOpenOutlined,
  InfoCircleOutlined,
  LoadingOutlined,
  LockOutlined,
  NodeIndexOutlined,
  PlusOutlined,
  ReloadOutlined,
  RobotOutlined,
  SafetyCertificateOutlined,
  SaveOutlined,
  SearchOutlined,
  SettingOutlined,
  SwapRightOutlined,
  TeamOutlined,
  ThunderboltOutlined,
  UnlockOutlined,
  UploadOutlined,
  UserOutlined,
} from '@ant-design/icons-vue'
import {message} from 'ant-design-vue'
import type {
  AuditRule as ApiAuditRule,
  ProcessAuditConfig as ApiProcessAuditConfig,
  SystemPromptTemplate
} from '~/composables/useRulesApi'
import type {ArchiveRule, CronTaskConfig, ProcessArchiveConfig} from '~/types/rules'
import {useI18n} from '~/composables/useI18n'
import {usePagination} from '~/composables/usePagination'

definePageMeta({ middleware: 'auth', layout: 'default' })

const { t } = useI18n()
const rulesApi = useRulesApi()
const cronApi = useCronApi()
const archiveApi = useArchiveApi()

//===== 顶级选项卡：审核工作台 vs 定时任务配置 vs 归档复盘 =====
const topTab = ref<'audit' | 'cron' | 'archive'>('audit')

//===== Cron 任务类型配置 =====
const cronConfigs = ref<CronTaskConfig[]>([])
const loadingCron = ref(false)
const selectedCronType = ref<string>('')

const selectedCronConfig = computed(() =>
  cronConfigs.value.find(c => c.task_type === selectedCronType.value)
)

const pushFormatOptions = computed(() => [
  { value: 'html', label: t('admin.ruleConfig.htmlEmail') },
  { value: 'markdown', label: t('admin.ruleConfig.markdown') },
  { value: 'plain', label: t('admin.ruleConfig.plainText') },
])


//每日/每周报告内容模板的模板变量
const cronTemplateVariables = computed(() => {
  const taskType = selectedCronConfig.value?.task_type || ''
  if (taskType === 'audit_daily' || taskType === 'archive_daily') {
    return [
      { key: '{{date}}', desc: t('admin.ruleConfig.varDate') },
      { key: '{{time}}', desc: t('admin.ruleConfig.varTimeCutoff') },
      { key: '{{total}}', desc: t('admin.ruleConfig.varTotalDaily') },
      { key: '{{approved}}', desc: t('admin.ruleConfig.varApproved') },
      { key: '{{rejected}}', desc: t('admin.ruleConfig.varRejected') },
      { key: '{{revised}}', desc: t('admin.ruleConfig.varRevised') },
      { key: '{{pass_rate}}', desc: t('admin.ruleConfig.varPassRate') },
      { key: '{{detail_list}}', desc: t('admin.ruleConfig.varDetailList') },
      { key: '{{statistics}}', desc: t('admin.ruleConfig.varStatistics') },
    ]
  }
  if (taskType === 'audit_weekly' || taskType === 'archive_weekly') {
    return [
      { key: '{{week}}', desc: t('admin.ruleConfig.varWeek') },
      { key: '{{date_range}}', desc: t('admin.ruleConfig.varDateRange') },
      { key: '{{time}}', desc: t('admin.ruleConfig.varTimeGenerated') },
      { key: '{{total}}', desc: t('admin.ruleConfig.varTotalWeekly') },
      { key: '{{trend}}', desc: t('admin.ruleConfig.varTrend') },
      { key: '{{compliance_rate}}', desc: t('admin.ruleConfig.varComplianceRate') },
      { key: '{{compliance_trend}}', desc: t('admin.ruleConfig.varComplianceTrend') },
      { key: '{{detail_list}}', desc: t('admin.ruleConfig.varDetailList') },
      { key: '{{statistics}}', desc: t('admin.ruleConfig.varStatistics') },
    ]
  }
  return []
})

//用于 cron 模板变量插入的文本区域参考
const cronSubjectRef = ref<any>(null)
const cronHeaderRef = ref<any>(null)
const cronBodyRef = ref<any>(null)
const cronFooterRef = ref<any>(null)
const cronActiveField = ref<'subject' | 'header' | 'body_template' | 'footer'>('body_template')

const insertCronVariable = (variable: string) => {
  if (!selectedCronConfig.value) return
  const field = cronActiveField.value
  const refMap: Record<string, any> = {
    subject: cronSubjectRef,
    header: cronHeaderRef,
    body_template: cronBodyRef,
    footer: cronFooterRef,
  }
  const textareaRef = refMap[field]
  const el: HTMLTextAreaElement | HTMLInputElement | null =
    textareaRef?.value?.$el?.querySelector?.('textarea')
    || textareaRef?.value?.$el?.querySelector?.('input')
    || textareaRef?.value?.resizableTextArea?.textArea
    || null
  const currentVal = selectedCronConfig.value.content_template[field] || ''
  if (el) {
    const start = el.selectionStart ?? currentVal.length
    const end = el.selectionEnd ?? currentVal.length
    const newVal = currentVal.slice(0, start) + variable + currentVal.slice(end)
    selectedCronConfig.value.content_template[field] = newVal
    nextTick(() => {
      const pos = start + variable.length
      el.focus()
      el.setSelectionRange(pos, pos)
    })
  } else {
    selectedCronConfig.value.content_template[field] = currentVal + variable
  }
}

const handleSaveCronConfig = async () => {
  if (!selectedCronConfig.value) return
  savingCron.value = true
  try {
    const cfg = selectedCronConfig.value
    const saved = await cronApi.saveConfig(cfg.task_type, {
      push_format: cfg.push_format,
      content_template: cfg.content_template,
      batch_limit: cfg.batch_limit,
    })
    // 更新本地数据
    const idx = cronConfigs.value.findIndex(c => c.task_type === cfg.task_type)
    if (idx >= 0) cronConfigs.value[idx] = saved
    message.success(t('admin.ruleConfig.cronSaved'))
  } catch (e: any) {
    message.error(t('admin.ruleConfig.cronSaveFail') + ': ' + (e.message || ''))
  } finally {
    savingCron.value = false
  }
}

const handleResetCronTemplate = async () => {
  if (!selectedCronConfig.value) return
  try {
    const cfg = selectedCronConfig.value
    const reset = await cronApi.resetConfig(cfg.task_type)
    const idx = cronConfigs.value.findIndex(c => c.task_type === cfg.task_type)
    if (idx >= 0) cronConfigs.value[idx] = reset
    message.success(t('admin.ruleConfig.cronReset'))
  } catch (e: any) {
    message.error(t('admin.ruleConfig.cronResetFail') + ': ' + (e.message || ''))
  }
}

const processConfigs = ref<ApiProcessAuditConfig[]>([])
const selectedProcessId = ref('')
// 当前选中流程的规则列表（从 API 加载）
const currentRules = ref<ApiAuditRule[]>([])
const loadingRules = ref(false)

//=====测试连接状态=====
const testingConnection = ref(false)
const testConnectionResult = ref<{ success: boolean; message: string } | null>(null)
// 基本信息页面的测试连接状态（独立于新增弹框）
const infoTestingConnection = ref(false)
const infoTestConnectionResult = ref<{ success: boolean; message: string } | null>(null)
// 同步字段状态
const syncingFields = ref(false)

//=====添加新流程=====
const showAddProcess = ref(false)
const newProcessForm = ref({ process_type: '', process_type_label: '', main_table_name: '' })

// 新增弹框中的测试连接
const handleTestConnectionInModal = async () => {
  const processType = newProcessForm.value.process_type.trim()
  if (!processType) {
    message.warning(t('admin.ruleConfig.enterProcessName'))
    return
  }
  testingConnection.value = true
  testConnectionResult.value = null
  try {
    const info = await rulesApi.testConnection(processType, newProcessForm.value.main_table_name.trim(), newProcessForm.value.process_type_label?.trim() || '')
    if (info.table_mismatch || info.type_label_mismatch) {
      const msgs = []
      if (info.table_mismatch) {
        msgs.push(t('admin.ruleConfig.tableMismatch', [info.expected_table || '-']))
        if (info.expected_table) {
          newProcessForm.value.main_table_name = info.expected_table
        }
      }
      if (info.type_label_mismatch) {
        msgs.push(t('admin.ruleConfig.typeLabelMismatch', [info.expected_type_label || '-']))
        if (info.expected_type_label) {
          newProcessForm.value.process_type_label = info.expected_type_label
        }
      }
      testConnectionResult.value = {
        success: false,
        message: msgs.join('；'),
      }
    } else {
      testConnectionResult.value = {
        success: true,
        message: t('admin.ruleConfig.testConnectionSuccess', [info.process_name || processType, info.main_table || '-', info.process_type_label || '-']),
      }
      // 自动填充主表名称和流程类型
      if (info.main_table) {
        newProcessForm.value.main_table_name = info.main_table
      }
      if (info.process_type_label) {
        newProcessForm.value.process_type_label = info.process_type_label
      }
    }
  } catch (e: any) {
    testConnectionResult.value = {
      success: false,
      message: t('admin.ruleConfig.testConnectionFail', [e.message || '未知错误']),
    }
  } finally {
    testingConnection.value = false
  }
}

// 基本信息页面的测试连接
const handleTestConnectionInInfo = async () => {
  if (!selectedConfig.value) return
  const processType = selectedConfig.value.process_type.trim()
  if (!processType) {
    message.warning(t('admin.ruleConfig.enterProcessName'))
    return
  }
  infoTestingConnection.value = true
  infoTestConnectionResult.value = null
  try {
    const info = await rulesApi.testConnection(processType, selectedConfig.value.main_table_name.trim(), selectedConfig.value.process_type_label?.trim() || '')
    if (info.table_mismatch || info.type_label_mismatch) {
      const msgs = []
      if (info.table_mismatch) {
        msgs.push(t('admin.ruleConfig.tableMismatch', [info.expected_table || '-']))
        if (info.expected_table && selectedConfig.value) {
          selectedConfig.value.main_table_name = info.expected_table
        }
      }
      if (info.type_label_mismatch) {
        msgs.push(t('admin.ruleConfig.typeLabelMismatch', [info.expected_type_label || '-']))
        if (info.expected_type_label && selectedConfig.value) {
          selectedConfig.value.process_type_label = info.expected_type_label
        }
      }
      infoTestConnectionResult.value = {
        success: false,
        message: msgs.join('；'),
      }
    } else {
      infoTestConnectionResult.value = {
        success: true,
        message: t('admin.ruleConfig.testConnectionSuccess', [info.process_name || processType, info.main_table || '-', info.process_type_label || '-']),
      }
      // 自动填充主表名称和流程类型
      if (info.main_table && selectedConfig.value) {
        selectedConfig.value.main_table_name = info.main_table
      }
      if (info.process_type_label && selectedConfig.value) {
        selectedConfig.value.process_type_label = info.process_type_label
      }
    }
  } catch (e: any) {
    infoTestConnectionResult.value = {
      success: false,
      message: t('admin.ruleConfig.testConnectionFail', [e.message || '未知错误']),
    }
  } finally {
    infoTestingConnection.value = false
  }
}

// 同步 OA 字段
const handleSyncFields = async () => {
  if (!selectedConfig.value) return
  syncingFields.value = true
  try {
    const fields = await rulesApi.fetchFields(selectedConfig.value.id)
    // 更新本地数据
    selectedConfig.value.main_fields = (fields.main_fields || []).map((f: any) => ({ ...f, selected: true }))
    selectedConfig.value.detail_tables = (fields.detail_tables || []).map((dt: any) => ({
      ...dt,
      fields: dt.fields.map((f: any) => ({ ...f, selected: true })),
    }))
    message.success(t('admin.ruleConfig.fetchFieldsSuccess'))
  } catch (e: any) {
    message.error(t('admin.ruleConfig.fetchFieldsFail') + ': ' + (e.message || ''))
  } finally {
    syncingFields.value = false
  }
}

const handleAddProcess = async () => {
  if (!newProcessForm.value.process_type.trim()) {
    message.warning(t('admin.ruleConfig.enterProcessName'))
    return
  }
  try {
    const created = await rulesApi.createConfig({
      process_type: newProcessForm.value.process_type.trim(),
      process_type_label: newProcessForm.value.process_type_label.trim(),
      main_table_name: newProcessForm.value.main_table_name.trim(),
    })
    processConfigs.value.push(created)
    selectedProcessId.value = created.id
    showAddProcess.value = false
    newProcessForm.value = { process_type: '', process_type_label: '', main_table_name: '' }
    testConnectionResult.value = null
    message.success(t('admin.ruleConfig.processAdded'))
  } catch (e: any) {
    message.error(t('admin.ruleConfig.createConfigFail') + ': ' + (e.message || ''))
  }
}

// 删除流程配置
const handleDeleteProcess = async (id: string) => {
  try {
    await rulesApi.deleteConfig(id)
    processConfigs.value = processConfigs.value.filter(c => c.id !== id)
    if (selectedProcessId.value === id) {
      selectedProcessId.value = processConfigs.value[0]?.id || ''
    }
    message.success(t('admin.ruleConfig.deleteConfigSuccess'))
  } catch (e: any) {
    message.error(t('admin.ruleConfig.deleteConfigFail') + ': ' + (e.message || ''))
  }
}
const activeTab = ref('info')

const selectedConfig = computed(() =>
  processConfigs.value.find(c => c.id === selectedProcessId.value)
)

// 当选中流程变化时，从 API 加载该流程的规则
watch(selectedProcessId, async (newId) => {
  if (!newId) { currentRules.value = []; return }
  const cfg = processConfigs.value.find(c => c.id === newId)
  if (!cfg) { currentRules.value = []; return }
  loadingRules.value = true
  try {
    currentRules.value = await rulesApi.listRules(cfg.id)
  } catch (e) {
    console.error('[rules] 加载规则失败', e)
    currentRules.value = []
  } finally {
    loadingRules.value = false
  }
  // 重置基本信息页面的测试连接状态
  infoTestConnectionResult.value = null
})

//===== 字段配置 =====
const fieldTypeLabels = computed<Record<string, string>>(() => ({
  text: t('fieldType.text'), number: t('fieldType.number'), date: t('fieldType.date'), select: t('fieldType.select'), textarea: t('fieldType.textarea'), file: t('fieldType.file'),
}))

const toggleFieldSelection = (field: ProcessField) => {
  if (selectedConfig.value?.field_mode === 'all') return
  field.selected = !field.selected
}

//===== 字段选择器模态 =====
const showFieldPicker = ref(false)
const fieldSearchQuery = ref('')

//当前流程的所有可用字段（主表+明细表），按表分组
interface PickerField {
  field_key: string; field_name: string; field_type: string; selected: boolean
  source: string; sourceLabel: string
}
interface FieldGroup {
  source: string; sourceLabel: string; fields: PickerField[]
}

const groupedAvailableFields = computed<FieldGroup[]>(() => {
  if (!selectedConfig.value) return []
  const groups: FieldGroup[] = []
  const mainFields = selectedConfig.value.main_fields || []
  groups.push({
    source: 'main',
    sourceLabel: t('admin.ruleConfig.mainTableFields'),
    fields: mainFields.map(f => ({ ...f, selected: f.selected ?? false, source: 'main', sourceLabel: t('admin.ruleConfig.mainTableFields') })),
  })
  if (selectedConfig.value.detail_tables) {
    selectedConfig.value.detail_tables.forEach((dt, idx) => {
      groups.push({
        source: dt.table_name,
        sourceLabel: `${t('admin.ruleConfig.detailTableLabel')} ${idx + 1}`,
        fields: dt.fields.map(f => ({ ...f, selected: f.selected ?? false, source: dt.table_name, sourceLabel: `${t('admin.ruleConfig.detailTableLabel')} ${idx + 1}` })),
      })
    })
  }
  return groups
})

const allAvailableFields = computed<PickerField[]>(() =>
  groupedAvailableFields.value.flatMap(g => g.fields)
)

const selectedFieldCount = computed(() =>
  allAvailableFields.value.filter(f => f.selected).length
)

const selectedFieldSearchQuery = ref('')
const leftSelectedKeys = ref<string[]>([])
const rightSelectedKeys = ref<string[]>([])

const unselectedFieldsFlat = computed(() => {
  const q = fieldSearchQuery.value.toLowerCase().trim()
  return allAvailableFields.value.filter(f => {
    if (f.selected) return false
    if (!q) return true
    return f.field_name.toLowerCase().includes(q) || f.field_key.toLowerCase().includes(q)
  })
})
const unselectedPagination = usePagination(unselectedFieldsFlat, 5)

const toggleLeftSelectAll = () => {
  if (leftSelectedKeys.value.length === unselectedFieldsFlat.value.length && unselectedFieldsFlat.value.length > 0) {
    leftSelectedKeys.value = []
  } else {
    leftSelectedKeys.value = unselectedFieldsFlat.value.map(f => f.field_key + '_' + f.source)
  }
}

const toggleLeftSelect = (fieldId: string) => {
  const idx = leftSelectedKeys.value.indexOf(fieldId)
  if (idx >= 0) leftSelectedKeys.value.splice(idx, 1)
  else leftSelectedKeys.value.push(fieldId)
}

const batchPick = () => {
  unselectedFieldsFlat.value.filter(f => leftSelectedKeys.value.includes(f.field_key + '_' + f.source)).forEach(pickField)
  leftSelectedKeys.value = []
}

const selectedFieldsFlat = computed(() => {
  const q = selectedFieldSearchQuery.value.toLowerCase().trim()
  return allAvailableFields.value.filter(f => {
    if (!f.selected) return false
    if (!q) return true
    return f.field_name.toLowerCase().includes(q) || f.field_key.toLowerCase().includes(q) || f.sourceLabel.toLowerCase().includes(q)
  })
})
const selectedPagination = usePagination(selectedFieldsFlat, 5)

const toggleRightSelectAll = () => {
  if (rightSelectedKeys.value.length === selectedFieldsFlat.value.length && selectedFieldsFlat.value.length > 0) {
    rightSelectedKeys.value = []
  } else {
    rightSelectedKeys.value = selectedFieldsFlat.value.map(f => f.field_key + '_' + f.source)
  }
}

const toggleRightSelect = (fieldId: string) => {
  const idx = rightSelectedKeys.value.indexOf(fieldId)
  if (idx >= 0) rightSelectedKeys.value.splice(idx, 1)
  else rightSelectedKeys.value.push(fieldId)
}

const batchUnpick = () => {
  selectedFieldsFlat.value.filter(f => rightSelectedKeys.value.includes(f.field_key + '_' + f.source)).forEach(unpickField)
  rightSelectedKeys.value = []
}

const pageSelectedFieldSearchQuery = ref('')
const pageSelectedFieldsFlat = computed(() => {
  const q = pageSelectedFieldSearchQuery.value.toLowerCase().trim()
  return allAvailableFields.value.filter(f => {
    if (!f.selected) return false
    if (!q) return true
    return f.field_name.toLowerCase().includes(q) || f.field_key.toLowerCase().includes(q) || f.sourceLabel.toLowerCase().includes(q)
  })
})
const pageSelectedPagination = usePagination(pageSelectedFieldsFlat, 5)


//按表分组的已筛选未选定字段（选择器左侧）
const groupedUnselectedFields = computed<FieldGroup[]>(() => {
  const q = fieldSearchQuery.value.toLowerCase().trim()
  return groupedAvailableFields.value
    .map(g => ({
      ...g,
      fields: g.fields.filter(f => {
        if (f.selected) return false
        if (!q) return true
        return f.field_name.toLowerCase().includes(q) || f.field_key.toLowerCase().includes(q)
      }),
    }))
    .filter(g => g.fields.length > 0)
})



const openFieldPicker = () => {
  fieldSearchQuery.value = ''
  selectedFieldSearchQuery.value = ''
  leftSelectedKeys.value = []
  rightSelectedKeys.value = []
  showFieldPicker.value = true
}

const pickField = (field: { field_key: string; source: string }) => {
  if (!selectedConfig.value) return
  //在 main_fields 中查找并切换
  const mainFields = selectedConfig.value.main_fields || []
  const mf = mainFields.find(f => f.field_key === field.field_key)
  if (mf && field.source === 'main') { mf.selected = true; return }
  //查找详细表格
  if (selectedConfig.value.detail_tables) {
    for (const dt of selectedConfig.value.detail_tables) {
      if (dt.table_name === field.source) {
        const df = dt.fields.find(f => f.field_key === field.field_key)
        if (df) { df.selected = true; return }
      }
    }
  }
}

const unpickField = (field: { field_key: string; source: string }) => {
  if (!selectedConfig.value) return
  const mainFields = selectedConfig.value.main_fields || []
  const mf = mainFields.find(f => f.field_key === field.field_key)
  if (mf && field.source === 'main') { mf.selected = false; return }
  if (selectedConfig.value.detail_tables) {
    for (const dt of selectedConfig.value.detail_tables) {
      if (dt.table_name === field.source) {
        const df = dt.fields.find(f => f.field_key === field.field_key)
        if (df) { df.selected = false; return }
      }
    }
  }
}

//=====规则配置=====
const scopeConfig = computed(() => ({
  mandatory: { label: t('admin.ruleConfig.mandatory'), color: 'var(--color-danger)', bg: 'var(--color-danger-bg)', icon: LockOutlined },
  default_on: { label: t('admin.ruleConfig.defaultOn'), color: 'var(--color-primary)', bg: 'var(--color-primary-bg)', icon: UnlockOutlined },
  default_off: { label: t('admin.ruleConfig.defaultOff'), color: 'var(--color-text-tertiary)', bg: 'var(--color-bg-hover)', icon: UnlockOutlined },
}))

const showRuleEditor = ref(false)
const editingRule = ref<ApiAuditRule | AuditRule | null>(null)

const openRuleEditor = (rule?: ApiAuditRule | AuditRule) => {
  editingRule.value = rule || null
  showRuleEditor.value = true
}

const handleSaveRule = async (rule: any) => {
  if (!selectedConfig.value) return
  try {
    if (editingRule.value) {
      // 更新规则
      const updated = await rulesApi.updateRule(editingRule.value.id, {
        rule_content: rule.rule_content,
        rule_scope: rule.rule_scope,
        related_flow: rule.related_flow,
      })
      const idx = currentRules.value.findIndex(r => r.id === editingRule.value!.id)
      if (idx >= 0) currentRules.value[idx] = updated
    } else {
      // 创建规则
      const created = await rulesApi.createRule({
        config_id: selectedConfig.value.id,
        process_type: selectedConfig.value.process_type,
        rule_content: rule.rule_content,
        rule_scope: rule.rule_scope,
        related_flow: rule.related_flow,
      })
      currentRules.value.push(created)
    }
    showRuleEditor.value = false
    editingRule.value = null
    message.success(t('admin.ruleConfig.ruleSaved'))
  } catch (e: any) {
    const key = editingRule.value ? 'admin.ruleConfig.updateRuleFail' : 'admin.ruleConfig.createRuleFail'
    message.error(t(key) + ': ' + (e.message || ''))
  }
}

const deleteRule = async (id: string) => {
  try {
    await rulesApi.deleteRule(id)
    currentRules.value = currentRules.value.filter(r => r.id !== id)
    message.success(t('admin.ruleConfig.deleted'))
  } catch (e: any) {
    message.error(t('admin.ruleConfig.deleteRuleFail') + ': ' + (e.message || ''))
  }
}

const handleImportRules = () => {
  message.info(t('admin.ruleConfig.fileImportDev'))
}

const kbModes = computed(() => [
  { key: 'rules_only', icon: FileTextOutlined, title: t('admin.ruleConfig.rulesOnlyTitle'), desc: t('admin.ruleConfig.rulesOnlyDesc'), available: true },
  { key: 'rag_only', icon: DatabaseOutlined, title: t('admin.ruleConfig.ragOnlyTitle'), desc: t('admin.ruleConfig.ragOnlyDesc'), available: false },
  { key: 'hybrid', icon: ThunderboltOutlined, title: t('admin.ruleConfig.hybridTitle'), desc: t('admin.ruleConfig.hybridDesc'), available: false },
])

//=====人工智能配置=====
const strictnessOptions = computed(() => [
  { value: 'strict', label: t('admin.ruleConfig.strict'), desc: t('admin.ruleConfig.strictDescNew') },
  { value: 'standard', label: t('admin.ruleConfig.standard'), desc: t('admin.ruleConfig.standardDescNew') },
  { value: 'loose', label: t('admin.ruleConfig.loose'), desc: t('admin.ruleConfig.looseDescNew') },
])




//用户推理提示词可用变量
const reasoningPromptVariables = computed(() => [
  { key: '{{main_table}}', desc: t('admin.ruleConfig.varMainTableDesc') },
  { key: '{{detail_tables}}', desc: t('admin.ruleConfig.varDetailTablesDesc') },
  { key: '{{rules}}', desc: t('admin.ruleConfig.varRulesDesc') },
  { key: '{{flow_history}}', desc: t('admin.ruleConfig.varFlowHistoryDesc') },
  { key: '{{flow_graph}}', desc: t('admin.ruleConfig.varFlowGraphDesc') },
  { key: '{{current_node}}', desc: t('admin.ruleConfig.varCurrentNodeDesc') },
])

//用户提取提示词可用变量
const extractionPromptVariables = computed(() => [
  { key: '{{reasoning_result}}', desc: t('admin.ruleConfig.varReasoningResultDesc') },
  { key: '{{rules}}', desc: t('admin.ruleConfig.varRulesDesc') },
])

//用于光标位置插入的文本区域引用
const reasoningTextareaRef = ref<any>(null)
const extractionTextareaRef = ref<any>(null)

const insertAtCursor = (textareaRef: any, field: 'user_reasoning_prompt' | 'user_extraction_prompt', variable: string) => {
  if (!selectedConfig.value) return
  //从ant-design-vue的a-textarea获取原生textarea元素
  const el: HTMLTextAreaElement | null = textareaRef?.value?.$el?.querySelector?.('textarea')
    || textareaRef?.value?.resizableTextArea?.textArea
    || null
  const currentVal = selectedConfig.value.ai_config[field] || ''
  if (el) {
    const start = el.selectionStart ?? currentVal.length
    const end = el.selectionEnd ?? currentVal.length
    const newVal = currentVal.slice(0, start) + variable + currentVal.slice(end)
    selectedConfig.value.ai_config[field] = newVal
    //Vue重新渲染后恢复光标位置
    nextTick(() => {
      const pos = start + variable.length
      el.focus()
      el.setSelectionRange(pos, pos)
    })
  } else {
    //后备：追加到最后
    selectedConfig.value.ai_config[field] = currentVal + variable
  }
}

const insertReasoningVariable = (variable: string) => {
  insertAtCursor(reasoningTextareaRef, 'user_reasoning_prompt', variable)
}

const insertExtractionVariable = (variable: string) => {
  insertAtCursor(extractionTextareaRef, 'user_extraction_prompt', variable)
}

//=====系统提示词模板=====

const promptTemplates = ref<SystemPromptTemplate[]>([])
const archivePromptTemplates = ref<SystemPromptTemplate[]>([])
const loadingTemplates = ref(false)
onMounted(async () => {
  loadOrgData()
  // 加载审核工作台配置
  try {
    const configs = await rulesApi.listConfigs()
    processConfigs.value = configs
    if (configs.length > 0) selectedProcessId.value = configs[0].id
  } catch (e) { console.error('[rules] 加载流程配置失败', e) }
  // 加载提示词模板（审核工作台）
  loadingTemplates.value = true
  try {
    promptTemplates.value = await rulesApi.listPromptTemplates()
  } catch (e) { console.error('[rules] 加载提示词模板失败', e) }
  finally { loadingTemplates.value = false }
  // 加载定时任务类型配置
  loadingCron.value = true
  try {
    const cronList = await cronApi.listConfigs()
    cronConfigs.value = cronList
    if (cronList.length > 0) selectedCronType.value = cronList[0].task_type
  } catch (e) { console.error('[rules] 加载定时任务配置失败', e) }
  finally { loadingCron.value = false }
  // 加载归档复盘配置
  loadingArchive.value = true
  try {
    const archiveList = await archiveApi.listConfigs()
    archiveConfigs.value = archiveList
    if (archiveList.length > 0) selectedArchiveId.value = archiveList[0].id
    // 同时加载归档专用提示词模板
    archivePromptTemplates.value = await archiveApi.listPromptTemplates()
  } catch (e) { console.error('[rules] 加载归档配置失败', e) }
  finally { loadingArchive.value = false }
})

const getTemplateContent = (promptKey: string) => {
  return promptTemplates.value.find(t => t.prompt_key === promptKey)?.content || ''
}

const handleStrictnessChange = (value: string) => {
  if (!selectedConfig.value) return
  selectedConfig.value.ai_config.audit_strictness = value as any
  selectedConfig.value.ai_config.system_reasoning_prompt = getTemplateContent(`system_reasoning_${value}`)
  selectedConfig.value.ai_config.system_extraction_prompt = getTemplateContent(`system_extraction_${value}`)
  selectedConfig.value.ai_config.user_reasoning_prompt = getTemplateContent(`user_reasoning_${value}`)
  selectedConfig.value.ai_config.user_extraction_prompt = getTemplateContent(`user_extraction_${value}`)
}

const resetSystemPrompts = () => {
  if (!selectedConfig.value) return
  const strictness = selectedConfig.value.ai_config.audit_strictness || 'standard'
  selectedConfig.value.ai_config.system_reasoning_prompt = getTemplateContent(`system_reasoning_${strictness}`)
  selectedConfig.value.ai_config.system_extraction_prompt = getTemplateContent(`system_extraction_${strictness}`)
  message.success(t('admin.ruleConfig.systemPromptsReset'))
}

const resetUserPrompts = () => {
  if (!selectedConfig.value) return
  const strictness = selectedConfig.value.ai_config.audit_strictness || 'standard'
  selectedConfig.value.ai_config.user_reasoning_prompt = getTemplateContent(`user_reasoning_${strictness}`)
  selectedConfig.value.ai_config.user_extraction_prompt = getTemplateContent(`user_extraction_${strictness}`)
  message.success(t('admin.ruleConfig.userPromptsReset'))
}

//=====用户权限=====
//===== 存档审核配置 =====
const { departments, roles, members, loadAll: loadOrgData } = useOrgApi()
const archiveConfigs = ref<ProcessArchiveConfig[]>([])
const loadingArchive = ref(false)
const selectedArchiveId = ref('')
const archiveActiveTab = ref('info')

const selectedArchiveConfig = computed(() =>
  archiveConfigs.value.find(c => c.id === selectedArchiveId.value)
)

// 当选中归档流程变化时，从 API 加载该流程的规则
const currentArchiveRules = ref<ArchiveRule[]>([])
const loadingArchiveRules = ref(false)

watch(selectedArchiveId, async (newId) => {
  if (!newId) { currentArchiveRules.value = []; return }
  const cfg = archiveConfigs.value.find(c => c.id === newId)
  if (!cfg) { currentArchiveRules.value = []; return }
  loadingArchiveRules.value = true
  try {
    currentArchiveRules.value = await archiveApi.listRules(cfg.id)
  } catch (e) {
    console.error('[rules] 加载归档规则失败', e)
    currentArchiveRules.value = []
  } finally {
    loadingArchiveRules.value = false
  }
})

//=====添加新的归档进程=====
const showAddArchiveProcess = ref(false)
const newArchiveProcessForm = ref({ process_type: '', process_type_label: '', main_table_name: '' })
const archiveTestingConnection = ref(false)
const archiveTestConnectionResult = ref<{ success: boolean; message: string } | null>(null)

const handleTestConnectionInArchiveModal = async () => {
  const processType = newArchiveProcessForm.value.process_type.trim()
  if (!processType) {
    message.warning(t('admin.ruleConfig.enterProcessName'))
    return
  }
  archiveTestingConnection.value = true
  archiveTestConnectionResult.value = null
  try {
    const info = await archiveApi.testConnection(processType, newArchiveProcessForm.value.main_table_name.trim(), newArchiveProcessForm.value.process_type_label?.trim() || '')
    if (info.table_mismatch || info.type_label_mismatch) {
      const msgs = []
      if (info.table_mismatch && info.expected_table) {
        msgs.push(t('admin.ruleConfig.tableMismatch', [info.expected_table]))
        newArchiveProcessForm.value.main_table_name = info.expected_table
      }
      if (info.type_label_mismatch && info.expected_type_label) {
        msgs.push(t('admin.ruleConfig.typeLabelMismatch', [info.expected_type_label]))
        newArchiveProcessForm.value.process_type_label = info.expected_type_label
      }
      archiveTestConnectionResult.value = { success: false, message: msgs.join('；') }
    } else {
      archiveTestConnectionResult.value = {
        success: true,
        message: t('admin.ruleConfig.testConnectionSuccess', [info.process_name || processType, info.main_table || '-', info.process_type_label || '-']),
      }
      if (info.main_table) newArchiveProcessForm.value.main_table_name = info.main_table
      if (info.process_type_label) newArchiveProcessForm.value.process_type_label = info.process_type_label
    }
  } catch (e: any) {
    archiveTestConnectionResult.value = { success: false, message: t('admin.ruleConfig.testConnectionFail', [e.message || '未知错误']) }
  } finally {
    archiveTestingConnection.value = false
  }
}

const handleAddArchiveProcess = async () => {
  if (!newArchiveProcessForm.value.process_type.trim()) {
    message.warning(t('admin.ruleConfig.enterProcessName'))
    return
  }
  try {
    const created = await archiveApi.createConfig({
      process_type: newArchiveProcessForm.value.process_type.trim(),
      process_type_label: newArchiveProcessForm.value.process_type_label.trim(),
      main_table_name: newArchiveProcessForm.value.main_table_name.trim(),
      access_control: { allowed_roles: [], allowed_members: [], allowed_departments: [] },
    })
    archiveConfigs.value.push(created)
    selectedArchiveId.value = created.id
    showAddArchiveProcess.value = false
    newArchiveProcessForm.value = { process_type: '', process_type_label: '', main_table_name: '' }
    archiveTestConnectionResult.value = null
    message.success(t('admin.ruleConfig.processAdded'))
  } catch (e: any) {
    message.error(t('admin.ruleConfig.createConfigFail') + ': ' + (e.message || ''))
  }
}

const handleDeleteArchiveProcess = async (id: string) => {
  try {
    await archiveApi.deleteConfig(id)
    archiveConfigs.value = archiveConfigs.value.filter(c => c.id !== id)
    if (selectedArchiveId.value === id) {
      selectedArchiveId.value = archiveConfigs.value[0]?.id || ''
    }
    message.success(t('admin.ruleConfig.deleteConfigSuccess'))
  } catch (e: any) {
    message.error(t('admin.ruleConfig.deleteConfigFail') + ': ' + (e.message || ''))
  }
}

//===== 存档字段选择器 =====
const showArchiveFieldPicker = ref(false)
const archiveFieldSearchQuery = ref('')

interface ArchivePickerField {
  field_key: string; field_name: string; field_type: string; selected: boolean
  source: string; sourceLabel: string
}
interface ArchiveFieldGroup {
  source: string; sourceLabel: string; fields: ArchivePickerField[]
}

const archiveGroupedAvailableFields = computed<ArchiveFieldGroup[]>(() => {
  if (!selectedArchiveConfig.value) return []
  const groups: ArchiveFieldGroup[] = []
  const mainFields = selectedArchiveConfig.value.main_fields || []
  groups.push({
    source: 'main',
    sourceLabel: t('admin.ruleConfig.mainTableFields'),
    fields: mainFields.map(f => ({ ...f, selected: !!f.selected, source: 'main', sourceLabel: t('admin.ruleConfig.mainTableFields') })),
  })
  if (selectedArchiveConfig.value.detail_tables) {
    selectedArchiveConfig.value.detail_tables.forEach((dt, idx) => {
      groups.push({
        source: dt.table_name,
        sourceLabel: `${t('admin.ruleConfig.detailTableLabel')} ${idx + 1}`,
        fields: (dt.fields || []).map(f => ({ ...f, selected: !!f.selected, source: dt.table_name, sourceLabel: `${t('admin.ruleConfig.detailTableLabel')} ${idx + 1}` })),
      })
    })
  }
  return groups
})

const archiveAllAvailableFields = computed<ArchivePickerField[]>(() =>
  archiveGroupedAvailableFields.value.flatMap(g => g.fields)
)

const archiveSelectedFieldCount = computed(() =>
  archiveAllAvailableFields.value.filter(f => f.selected).length
)

const archiveGroupedUnselected = computed<ArchiveFieldGroup[]>(() => {
  const q = archiveFieldSearchQuery.value.toLowerCase().trim()
  return archiveGroupedAvailableFields.value
    .map(g => ({
      ...g,
      fields: g.fields.filter(f => {
        if (f.selected) return false
        if (!q) return true
        return f.field_name.toLowerCase().includes(q) || f.field_key.toLowerCase().includes(q)
      }),
    }))
    .filter(g => g.fields.length > 0)
})

const archiveGroupedSelected = computed<ArchiveFieldGroup[]>(() =>
  archiveGroupedAvailableFields.value
    .map(g => ({ ...g, fields: g.fields.filter(f => f.selected) }))
    .filter(g => g.fields.length > 0)
)

const openArchiveFieldPicker = () => {
  archiveFieldSearchQuery.value = ''
  showArchiveFieldPicker.value = true
}

const archivePickField = (field: { field_key: string; source: string }) => {
  if (!selectedArchiveConfig.value) return
  const mainFields = selectedArchiveConfig.value.main_fields || []
  const mf = mainFields.find(f => f.field_key === field.field_key)
  if (mf && field.source === 'main') { mf.selected = true; return }
  if (selectedArchiveConfig.value.detail_tables) {
    for (const dt of selectedArchiveConfig.value.detail_tables) {
      if (dt.table_name === field.source) {
        const df = dt.fields.find(f => f.field_key === field.field_key)
        if (df) { df.selected = true; return }
      }
    }
  }
}

const archiveUnpickField = (field: { field_key: string; source: string }) => {
  if (!selectedArchiveConfig.value) return
  const mainFields = selectedArchiveConfig.value.main_fields || []
  const mf = mainFields.find(f => f.field_key === field.field_key)
  if (mf && field.source === 'main') { mf.selected = false; return }
  if (selectedArchiveConfig.value.detail_tables) {
    for (const dt of selectedArchiveConfig.value.detail_tables) {
      if (dt.table_name === field.source) {
        const df = dt.fields.find(f => f.field_key === field.field_key)
        if (df) { df.selected = false; return }
      }
    }
  }
}

//=====存档规则=====
const showArchiveRuleEditor = ref(false)
const editingArchiveRule = ref<ArchiveRule | null>(null)

const openArchiveRuleEditor = (rule?: ArchiveRule) => {
  editingArchiveRule.value = rule || null
  showArchiveRuleEditor.value = true
}

const handleSaveArchiveRule = async (rule: any) => {
  if (!selectedArchiveConfig.value) return
  try {
    if (editingArchiveRule.value) {
      const updated = await archiveApi.updateRule(editingArchiveRule.value.id, {
        rule_content: rule.rule_content,
        rule_scope: rule.rule_scope,
        related_flow: rule.related_flow,
      })
      const idx = currentArchiveRules.value.findIndex(r => r.id === editingArchiveRule.value!.id)
      if (idx >= 0) currentArchiveRules.value[idx] = updated
    } else {
      const created = await archiveApi.createRule({
        config_id: selectedArchiveConfig.value.id,
        process_type: selectedArchiveConfig.value.process_type,
        rule_content: rule.rule_content,
        rule_scope: rule.rule_scope,
        related_flow: rule.related_flow,
      })
      currentArchiveRules.value.push(created)
    }
    showArchiveRuleEditor.value = false
    editingArchiveRule.value = null
    message.success(t('admin.ruleConfig.ruleSaved'))
  } catch (e: any) {
    const key = editingArchiveRule.value ? 'admin.ruleConfig.updateRuleFail' : 'admin.ruleConfig.createRuleFail'
    message.error(t(key) + ': ' + (e.message || ''))
  }
}

const deleteArchiveRule = async (id: string) => {
  try {
    await archiveApi.deleteRule(id)
    currentArchiveRules.value = currentArchiveRules.value.filter(r => r.id !== id)
    message.success(t('admin.ruleConfig.deleted'))
  } catch (e: any) {
    message.error(t('admin.ruleConfig.deleteRuleFail') + ': ' + (e.message || ''))
  }
}

//=====存档AI提示变量（与审计工作台相同）=====
const archiveReasoningPromptVariables = computed(() => [
  { key: '{{main_table}}', desc: t('admin.ruleConfig.varMainTableDesc') },
  { key: '{{detail_tables}}', desc: t('admin.ruleConfig.varDetailTablesDesc') },
  { key: '{{rules}}', desc: t('admin.ruleConfig.varRulesDesc') },
  { key: '{{flow_history}}', desc: t('admin.ruleConfig.varFlowHistoryDesc') },
  { key: '{{flow_graph}}', desc: t('admin.ruleConfig.varFlowGraphDesc') },
  { key: '{{current_node}}', desc: t('admin.ruleConfig.varCurrentNodeDesc') },
])
const archiveExtractionPromptVariables = computed(() => [
  { key: '{{reasoning_result}}', desc: t('admin.ruleConfig.varReasoningResultDesc') },
  { key: '{{rules}}', desc: t('admin.ruleConfig.varRulesDesc') },
])

const archiveReasoningTextareaRef = ref<any>(null)
const archiveExtractionTextareaRef = ref<any>(null)

const insertArchiveAtCursor = (textareaRef: any, field: 'user_reasoning_prompt' | 'user_extraction_prompt', variable: string) => {
  if (!selectedArchiveConfig.value) return
  const el: HTMLTextAreaElement | null = textareaRef?.value?.$el?.querySelector?.('textarea')
    || textareaRef?.value?.resizableTextArea?.textArea || null
  const currentVal = selectedArchiveConfig.value.ai_config[field] || ''
  if (el) {
    const start = el.selectionStart ?? currentVal.length
    const end = el.selectionEnd ?? currentVal.length
    const newVal = currentVal.slice(0, start) + variable + currentVal.slice(end)
    selectedArchiveConfig.value.ai_config[field] = newVal
    nextTick(() => { const pos = start + variable.length; el.focus(); el.setSelectionRange(pos, pos) })
  } else {
    selectedArchiveConfig.value.ai_config[field] = currentVal + variable
  }
}

// 归档复盘：恢复默认提示词模板
const getArchiveTemplateContent = (promptKey: string) => {
  return archivePromptTemplates.value.find(t => t.prompt_key === promptKey)?.content || ''
}

const resetArchiveSystemPrompts = () => {
  if (!selectedArchiveConfig.value) return
  const strictness = selectedArchiveConfig.value.ai_config.audit_strictness || 'standard'
  selectedArchiveConfig.value.ai_config.system_reasoning_prompt = getArchiveTemplateContent(`archive_system_reasoning_${strictness}`)
  selectedArchiveConfig.value.ai_config.system_extraction_prompt = getArchiveTemplateContent(`archive_system_extraction_${strictness}`)
  message.success(t('admin.ruleConfig.systemPromptsReset'))
}

const resetArchiveUserPrompts = () => {
  if (!selectedArchiveConfig.value) return
  const strictness = selectedArchiveConfig.value.ai_config.audit_strictness || 'standard'
  selectedArchiveConfig.value.ai_config.user_reasoning_prompt = getArchiveTemplateContent(`archive_user_reasoning_${strictness}`)
  selectedArchiveConfig.value.ai_config.user_extraction_prompt = getArchiveTemplateContent(`archive_user_extraction_${strictness}`)
  message.success(t('admin.ruleConfig.userPromptsReset'))
}

const handleArchiveStrictnessChange = (value: string) => {
  if (!selectedArchiveConfig.value) return
  selectedArchiveConfig.value.ai_config.audit_strictness = value as any
  // 更新尺度时，同时重置系统和用户提示词为该尺度下的默认值
  resetArchiveSystemPrompts()
  resetArchiveUserPrompts()
}

const archiveInfoTestingConnection = ref(false)
const archiveInfoTestConnectionResult = ref<{ success: boolean; message: string } | null>(null)

// 归档基本信息也提供测试连接
const handleArchiveTestConnectionInInfo = async () => {
  if (!selectedArchiveConfig.value) return
  const processType = selectedArchiveConfig.value.process_type.trim()
  if (!processType) {
    message.warning(t('admin.ruleConfig.enterProcessName'))
    return
  }
  archiveInfoTestingConnection.value = true
  archiveInfoTestConnectionResult.value = null
  try {
    const info = await archiveApi.testConnection(processType, selectedArchiveConfig.value.main_table_name.trim(), selectedArchiveConfig.value.process_type_label?.trim() || '')
    if (info.table_mismatch || info.type_label_mismatch) {
      const msgs = []
      if (info.table_mismatch) {
        msgs.push(t('admin.ruleConfig.tableMismatch', [info.expected_table || '-']))
        if (info.expected_table && selectedArchiveConfig.value) {
          selectedArchiveConfig.value.main_table_name = info.expected_table
        }
      }
      if (info.type_label_mismatch) {
        msgs.push(t('admin.ruleConfig.typeLabelMismatch', [info.expected_type_label || '-']))
        if (info.expected_type_label && selectedArchiveConfig.value) {
          selectedArchiveConfig.value.process_type_label = info.expected_type_label
        }
      }
      archiveInfoTestConnectionResult.value = { success: false, message: msgs.join('；') }
    } else {
      archiveInfoTestConnectionResult.value = {
        success: true,
        message: t('admin.ruleConfig.testConnectionSuccess', [info.process_name || processType, info.main_table || '-', info.process_type_label || '-']),
      }
      if (info.main_table && selectedArchiveConfig.value) selectedArchiveConfig.value.main_table_name = info.main_table
      if (info.process_type_label && selectedArchiveConfig.value) selectedArchiveConfig.value.process_type_label = info.process_type_label
    }
  } catch (e: any) {
    archiveInfoTestConnectionResult.value = { success: false, message: t('admin.ruleConfig.testConnectionFail', [e.message || '未知错误']) }
  } finally {
    archiveInfoTestingConnection.value = false
  }
}


//=====归档权限（用户定制+访问控制）=====
const archivePermissionLabels = computed(() => ({
  allow_custom_fields: { label: t('admin.ruleConfig.customReviewFields'), desc: t('admin.ruleConfig.customReviewFieldsDesc') },
  allow_custom_rules: { label: t('admin.ruleConfig.customReviewRules'), desc: t('admin.ruleConfig.customReviewRulesDesc') },
  allow_modify_strictness: { label: t('admin.ruleConfig.modReviewStrictness'), desc: t('admin.ruleConfig.modReviewStrictnessDesc') },
}))

//访问控制：角色和成员
const archiveRoleSearch = ref('')
const archiveMemberSearch = ref('')
const archiveDeptSearch = ref('')

const filteredArchiveRoles = computed(() => {
  const q = archiveRoleSearch.value.toLowerCase().trim()
  if (!q) return roles.value
  return roles.value.filter(r => r.name.toLowerCase().includes(q))
})

const filteredArchiveMembers = computed(() => {
  const q = archiveMemberSearch.value.toLowerCase().trim()
  if (!q) return members.value
  return members.value.filter(m => m.name.toLowerCase().includes(q) || m.department_name.toLowerCase().includes(q))
})

const filteredArchiveDepts = computed(() => {
  const q = archiveDeptSearch.value.toLowerCase().trim()
  if (!q) return departments.value
  return departments.value.filter(d => d.name.toLowerCase().includes(q))
})

const toggleArchiveRole = (roleId: string) => {
  if (!selectedArchiveConfig.value) return
  if (!selectedArchiveConfig.value.access_control) {
    selectedArchiveConfig.value.access_control = { allowed_roles: [], allowed_members: [], allowed_departments: [] }
  }
  const list = selectedArchiveConfig.value.access_control.allowed_roles
  const idx = list.indexOf(roleId)
  if (idx >= 0) list.splice(idx, 1)
  else list.push(roleId)
}

const toggleArchiveMember = (memberId: string) => {
  if (!selectedArchiveConfig.value) return
  if (!selectedArchiveConfig.value.access_control) {
    selectedArchiveConfig.value.access_control = { allowed_roles: [], allowed_members: [], allowed_departments: [] }
  }
  const list = selectedArchiveConfig.value.access_control.allowed_members
  const idx = list.indexOf(memberId)
  if (idx >= 0) list.splice(idx, 1)
  else list.push(memberId)
}

const toggleArchiveDept = (deptId: string) => {
  if (!selectedArchiveConfig.value) return
  if (!selectedArchiveConfig.value.access_control) {
    selectedArchiveConfig.value.access_control = { allowed_roles: [], allowed_members: [], allowed_departments: [] }
  }
  const list = selectedArchiveConfig.value.access_control.allowed_departments
  const idx = list.indexOf(deptId)
  if (idx >= 0) list.splice(idx, 1)
  else list.push(deptId)
}

const handleSaveArchiveConfig = async () => {
  if (!selectedArchiveConfig.value) return
  savingArchive.value = true
  try {
    const cfg = selectedArchiveConfig.value
    const updated = await archiveApi.updateConfig(cfg.id, {
      process_type: cfg.process_type,
      process_type_label: cfg.process_type_label,
      main_table_name: cfg.main_table_name,
      main_fields: cfg.main_fields,
      detail_tables: cfg.detail_tables,
      field_mode: cfg.field_mode,
      kb_mode: cfg.kb_mode,
      ai_config: cfg.ai_config,
      user_permissions: cfg.user_permissions,
      access_control: cfg.access_control,
      status: cfg.status,
    })
    const idx = archiveConfigs.value.findIndex(c => c.id === cfg.id)
    if (idx >= 0) archiveConfigs.value[idx] = updated
    message.success(t('admin.ruleConfig.archiveSaved'))
  } catch (e: any) {
    message.error(t('admin.ruleConfig.updateConfigFail') + ': ' + (e.message || ''))
  } finally {
    savingArchive.value = false
  }
}

const permissionLabels = computed(() => ({
  allow_custom_fields: { label: t('admin.ruleConfig.allowCustomFields'), desc: t('admin.ruleConfig.allowCustomFieldsDesc') },
  allow_custom_rules: { label: t('admin.ruleConfig.allowCustomRules'), desc: t('admin.ruleConfig.allowCustomRulesDesc') },
  allow_modify_strictness: { label: t('admin.ruleConfig.allowModStrictness'), desc: t('admin.ruleConfig.allowModStrictnessDesc') },
}))

const saving = ref(false)
const savingCron = ref(false)
const savingArchive = ref(false)

const handleSave = async () => {
  if (!selectedConfig.value) return
  saving.value = true
  try {
    const cfg = selectedConfig.value
    const updated = await rulesApi.updateConfig(cfg.id, {
      process_type: cfg.process_type,
      process_type_label: cfg.process_type_label,
      main_table_name: cfg.main_table_name,
      main_fields: cfg.main_fields,
      detail_tables: cfg.detail_tables,
      field_mode: cfg.field_mode,
      kb_mode: cfg.kb_mode,
      ai_config: cfg.ai_config,
      user_permissions: cfg.user_permissions,
      status: cfg.status,
    })
    // 更新本地数据
    const idx = processConfigs.value.findIndex(c => c.id === cfg.id)
    if (idx !== -1) processConfigs.value[idx] = updated
    message.success(t('admin.ruleConfig.configSaved'))
  } catch (e: any) {
    message.error(t('admin.ruleConfig.updateConfigFail') + ': ' + (e.message || ''))
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <div class="tenant-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('admin.ruleConfig.title') }}</h1>
        <p class="page-subtitle">{{ t('admin.ruleConfig.subtitle') }}</p>
      </div>
    </div>

    <!--顶级选项卡：审核工作台 / 定时任务配置 / 归档复盘-->
    <div class="top-tab-nav">
      <button
        v-for="tab in [
          { key: 'audit', label: t('admin.ruleConfig.tabAudit'), icon: DashboardOutlined },
          { key: 'cron', label: t('admin.ruleConfig.tabCron'), icon: ClockCircleOutlined },
          { key: 'archive', label: t('admin.ruleConfig.tabArchive'), icon: FolderOpenOutlined },
        ]"
        :key="tab.key"
        class="top-tab-btn"
        :class="{ 'top-tab-btn--active': topTab === tab.key }"
        @click="topTab = tab.key as any"
      >
        <component :is="tab.icon" />
        {{ tab.label }}
      </button>
    </div>

    <!-- ==================== 审核工作台配置 ==================== -->
    <div v-if="topTab === 'audit'" class="main-layout">
      <!--左：进程列表-->
      <div class="process-nav">
        <div class="process-nav-header">
          <SettingOutlined />
          <span>{{ t('admin.ruleConfig.auditProcess') }}</span>
          <button class="add-process-btn" @click="showAddProcess = true" :title="t('admin.ruleConfig.addProcess')">
            <PlusOutlined />
          </button>
        </div>
        <div
          v-for="cfg in processConfigs"
          :key="cfg.id"
          class="process-nav-item"
          :class="{ 'process-nav-item--active': selectedProcessId === cfg.id }"
          @click="selectedProcessId = cfg.id"
        >
          <div style="flex: 1; min-width: 0;">
            <div class="process-nav-name">{{ cfg.process_type }}</div>
            <div v-if="cfg.process_type_label" class="process-nav-path">{{ cfg.process_type_label }}</div>
          </div>
          <a-popconfirm :title="t('admin.ruleConfig.deleteConfigConfirm')" @confirm.stop="handleDeleteProcess(cfg.id)" placement="right">
            <button class="icon-btn icon-btn--danger icon-btn--sm" @click.stop style="opacity: 0.5; flex-shrink: 0;">
              <DeleteOutlined />
            </button>
          </a-popconfirm>
        </div>
      </div>

      <!--右：配置面板-->
      <div v-if="selectedConfig" class="config-panel">
        <div class="config-panel-header">
          <h2 class="config-panel-title">{{ selectedConfig.process_type }}</h2>
          <p v-if="selectedConfig.process_type_label" class="config-panel-subtitle">{{ selectedConfig.process_type_label }}</p>
        </div>

        <!--子选项卡-->
        <div class="tab-nav">
          <button
            v-for="tab in [
              { key: 'info', label: t('admin.ruleConfig.infoTab'), icon: InfoCircleOutlined },
              { key: 'fields', label: t('admin.ruleConfig.tabFields'), icon: AppstoreOutlined },
              { key: 'rules', label: t('admin.ruleConfig.tabRules'), icon: AuditOutlined },
              { key: 'ai', label: t('admin.ruleConfig.tabAI'), icon: RobotOutlined },
              { key: 'permissions', label: t('admin.ruleConfig.tabPerms'), icon: SafetyCertificateOutlined },
            ]"
            :key="tab.key"
            class="tab-btn"
            :class="{ 'tab-btn--active': activeTab === tab.key }"
            @click="activeTab = tab.key"
          >
            <component :is="tab.icon" />
            {{ tab.label }}
          </button>
        </div>

        <!--========== 信息选项卡 ==========-->
        <div v-if="activeTab === 'info'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">{{ t('admin.ruleConfig.infoTitle') }}</h4>
              <p class="section-desc">{{ t('admin.ruleConfig.infoDesc') }}</p>
            </div>
          </div>
          <a-form layout="vertical" class="info-form">
            <a-form-item :label="t('admin.ruleConfig.processNameLabel')">
              <a-input v-model:value="selectedConfig!.process_type" :placeholder="t('admin.ruleConfig.processNameInputPlaceholder')" />
            </a-form-item>
            <a-form-item :label="t('admin.ruleConfig.processTypeLabel')">
              <a-input
                :value="selectedConfig!.process_type_label ?? ''"
                @update:value="(v: string) => { if (selectedConfig) selectedConfig.process_type_label = v }"
                :placeholder="t('admin.ruleConfig.processTypeLabelPlaceholder')"
              />
            </a-form-item>
            <a-form-item :label="t('admin.ruleConfig.mainTableLabel')">
              <div style="display: flex; gap: 8px;">
                <a-input v-model:value="selectedConfig!.main_table_name" :placeholder="t('admin.ruleConfig.mainTableInputPlaceholder')" style="flex: 1;" />
                <a-button
                  :loading="infoTestingConnection"
                  @click="handleTestConnectionInInfo"
                >
                  <template #icon><DatabaseOutlined /></template>
                  {{ infoTestingConnection ? t('admin.ruleConfig.testingConnection') : t('admin.ruleConfig.testConnection') }}
                </a-button>
              </div>
              <div v-if="infoTestConnectionResult" style="margin-top: 8px;">
                <a-alert
                  :type="infoTestConnectionResult.success ? 'success' : 'error'"
                  :message="infoTestConnectionResult.message"
                  show-icon
                  closable
                  @close="infoTestConnectionResult = null"
                />
              </div>
            </a-form-item>
          </a-form>
        </div>

        <!--========== 字段选项卡 ==========-->
        <div v-if="activeTab === 'fields'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">{{ t('admin.ruleConfig.fieldTitle') }}</h4>
              <p class="section-desc">{{ t('admin.ruleConfig.fieldDesc') }}</p>
            </div>
            <a-button :loading="syncingFields" @click="handleSyncFields">
              <template #icon><DatabaseOutlined /></template>
              {{ syncingFields ? t('admin.ruleConfig.syncingFields') : t('admin.ruleConfig.syncFields') }}
            </a-button>
          </div>

          <div class="field-mode-switch">
            <div
              class="field-mode-option"
              :class="{ 'field-mode-option--active': selectedConfig.field_mode === 'selected' }"
              @click="selectedConfig.field_mode = 'selected'"
            >
              <div class="field-mode-radio" />
              <div>
                <div class="field-mode-label">{{ t('admin.ruleConfig.selectFields') }}</div>
                <div class="field-mode-desc">{{ t('admin.ruleConfig.selectFieldsDesc') }}</div>
              </div>
            </div>
            <div
              class="field-mode-option"
              :class="{ 'field-mode-option--active': selectedConfig.field_mode === 'all' }"
              @click="selectedConfig.field_mode = 'all'"
            >
              <div class="field-mode-radio" />
              <div>
                <div class="field-mode-label">{{ t('admin.ruleConfig.allFields') }}</div>
                <div class="field-mode-desc">{{ t('admin.ruleConfig.allFieldsDesc') }}</div>
              </div>
            </div>
          </div>

          <!--选定字段显示+选择器触发器-->
          <template v-if="selectedConfig.field_mode === 'selected'">
            <div class="field-picker-toolbar">
              <span class="field-count">{{ t('admin.ruleConfig.selectedCount', [`${selectedFieldCount}`, `${allAvailableFields.length}`]) }}</span>
              <a-button type="primary" @click="openFieldPicker">
                <AppstoreOutlined /> {{ t('admin.ruleConfig.selectFieldsModal') }}
              </a-button>
            </div>

            <div v-if="pageSelectedFieldsFlat.length > 0 || pageSelectedFieldSearchQuery" class="page-selected-fields-container" style="margin-top: 16px;">
              <div style="margin-bottom: 12px; max-width: 300px;">
                <a-input
                  v-model:value="pageSelectedFieldSearchQuery"
                  :placeholder="t('admin.ruleConfig.searchFieldPlaceholder')"
                  allow-clear
                >
                  <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
                </a-input>
              </div>
              <div class="data-table-card">
                <table class="data-table">
                  <thead>
                    <tr>
                      <th style="padding-left: 24px;">字段名称</th>
                      <th>字段标识</th>
                      <th>字段类型</th>
                      <th>归属表</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="field in pageSelectedPagination.paged.value" :key="field.field_key + field.source">
                      <td style="padding-left: 24px; font-weight: 500;">{{ field.field_name }}</td>
                      <td class="text-mono" style="font-size: 13px;">{{ field.field_key }}</td>
                      <td><span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span></td>
                      <td class="text-secondary" style="font-size: 13px;">{{ field.sourceLabel }}</td>
                    </tr>
                    <tr v-if="pageSelectedPagination.paged.value.length === 0">
                      <td colspan="4" class="empty-cell">{{ t('admin.ruleConfig.noSearchResult') || '未找到匹配字段' }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <div class="pagination-wrapper" style="margin-top: 12px; text-align: right;">
                <a-pagination
                  v-model:current="pageSelectedPagination.current.value"
                  v-model:page-size="pageSelectedPagination.pageSize.value"
                  :total="pageSelectedPagination.total.value"
                  size="small"
                  show-size-changer
                  show-quick-jumper
                  :page-size-options="['5', '20', '50']"
                  @change="pageSelectedPagination.onChange"
                  @showSizeChange="pageSelectedPagination.onChange"
                />
              </div>
            </div>
            <div v-else class="field-empty-hint" style="margin-top: 16px;">
              {{ t('admin.ruleConfig.noFieldsSelected') }}
            </div>
          </template>

          <template v-else>
            <div class="field-count" style="margin-top: 8px;">
              {{ t('admin.ruleConfig.allFieldsHint') }}
            </div>
          </template>
        </div>

        <!--========== 规则选项卡 ==========-->
        <div v-if="activeTab === 'rules'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">{{ t('admin.ruleConfig.rulesTitle') }}</h4>
              <p class="section-desc">{{ t('admin.ruleConfig.rulesDesc') }}</p>
            </div>
          </div>

          <!--KB 模式选择器-->
          <div class="kb-modes">
            <div
              v-for="mode in kbModes"
              :key="mode.key"
              class="kb-mode-card"
              :class="{
                'kb-mode-card--active': selectedConfig.kb_mode === mode.key,
                'kb-mode-card--disabled': !mode.available,
              }"
              @click="mode.available && (selectedConfig.kb_mode = mode.key as any)"
            >
              <div class="kb-mode-icon"><component :is="mode.icon" /></div>
              <div class="kb-mode-info">
                <div class="kb-mode-title">{{ mode.title }}</div>
                <div class="kb-mode-desc">{{ mode.desc }}</div>
              </div>
              <div v-if="selectedConfig.kb_mode === mode.key" class="kb-mode-check">✓</div>
              <div v-if="!mode.available" class="kb-mode-badge">{{ t('admin.ruleConfig.comingSoon') }}</div>
            </div>
          </div>

          <div class="rules-toolbar">
            <span class="rules-count">{{ t('admin.ruleConfig.totalRules', `${currentRules.length}`) }}</span>
            <div class="rules-toolbar-actions">
              <a-button @click="handleImportRules">
                <UploadOutlined /> {{ t('admin.ruleConfig.fileImport') }}
              </a-button>
              <a-button type="primary" @click="openRuleEditor()">
                <PlusOutlined /> {{ t('admin.ruleConfig.manualAddBtn') }}
              </a-button>
            </div>
          </div>

          <div class="rules-list">
            <a-spin v-if="loadingRules" style="display: block; text-align: center; padding: 24px;" />
            <div v-for="rule in currentRules" :key="rule.id" class="rule-card">
              <div class="rule-card-left">
                <div class="rule-scope-badge" :style="{ color: scopeConfig[rule.rule_scope]?.color, background: scopeConfig[rule.rule_scope]?.bg }">
                  <component :is="scopeConfig[rule.rule_scope]?.icon" />
                  {{ scopeConfig[rule.rule_scope]?.label }}
                </div>
                <div class="rule-card-body">
                  <div class="rule-card-content">{{ rule.rule_content }}</div>
                  <div class="rule-card-meta">
                    <span v-if="rule.source === 'file_import'" class="rule-source-tag">{{ t('admin.ruleConfig.fileImportTag') }}</span>
                    <span v-else class="rule-source-tag rule-source-tag--manual">{{ t('admin.ruleConfig.manualAddTag') }}</span>
                    <span v-if="rule.related_flow" class="rule-flow-tag">
                      <NodeIndexOutlined /> {{ t('admin.ruleConfig.relatedFlow') }}
                    </span>
                  </div>
                </div>
              </div>
              <div class="rule-card-actions">
                <a-switch :checked="rule.enabled" size="small" @change="(checked: any) => { rulesApi.updateRule(rule.id, { enabled: !!checked }).then(updated => { const idx = currentRules.findIndex(r => r.id === rule.id); if (idx >= 0) currentRules[idx] = updated }) }" />
                <button class="icon-btn" @click="openRuleEditor(rule)"><EditOutlined /></button>
                <a-popconfirm :title="t('admin.ruleConfig.deleteRuleConfirm')" @confirm="deleteRule(rule.id)">
                  <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
                </a-popconfirm>
              </div>
            </div>
          </div>
        </div>

        <!--========== AI 标签==========-->
        <div v-if="activeTab === 'ai'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">{{ t('admin.ruleConfig.aiTitle') }}</h4>
              <p class="section-desc">{{ t('admin.ruleConfig.aiDescNew') }}</p>
            </div>
          </div>

          <div class="ai-form">
            <!--审核尺度-->
            <div class="ai-form-group">
              <label class="ai-form-label">{{ t('admin.ruleConfig.strictness') }}</label>
              <div class="strictness-options">
                <div
                  v-for="opt in strictnessOptions"
                  :key="opt.value"
                  class="strictness-option"
                  :class="{ 'strictness-option--active': selectedConfig.ai_config.audit_strictness === opt.value }"
                  @click="handleStrictnessChange(opt.value)"
                >
                  <div class="strictness-option-radio" />
                  <div>
                    <div class="strictness-option-label">{{ opt.label }}</div>
                    <div class="strictness-option-desc">{{ opt.desc }}</div>
                  </div>
                </div>
              </div>
            </div>

            <!--系统提示词区域-->
            <div class="ai-prompt-section">
              <div class="ai-prompt-section-header">
                <div class="ai-prompt-section-tag ai-prompt-section-tag--system">{{ t('admin.ruleConfig.systemPromptTag') }}</div>
                <a-button size="small" type="link" @click="resetSystemPrompts">
                  <ReloadOutlined /> {{ t('admin.ruleConfig.resetSystemPresets') }}
                </a-button>
              </div>
              <p class="ai-prompt-section-desc">{{ t('admin.ruleConfig.systemPromptDesc') }}</p>

              <div class="ai-form-group">
                <div class="prompt-section-header">
                  <div class="prompt-section-title">
                    <span class="prompt-phase-badge prompt-phase-badge--reasoning">{{ t('admin.ruleConfig.phase1Label') }}</span>
                    <label class="ai-form-label">{{ t('admin.ruleConfig.systemReasoningPrompt') }}</label>
                  </div>
                </div>
                <a-textarea
                  v-model:value="selectedConfig.ai_config.system_reasoning_prompt"
                  :rows="6"
                  :placeholder="t('admin.ruleConfig.systemReasoningPlaceholder')"
                />
              </div>

              <div class="ai-form-group">
                <div class="prompt-section-header">
                  <div class="prompt-section-title">
                    <span class="prompt-phase-badge prompt-phase-badge--extraction">{{ t('admin.ruleConfig.phase2Label') }}</span>
                    <label class="ai-form-label">{{ t('admin.ruleConfig.systemExtractionPrompt') }}</label>
                  </div>
                </div>
                <a-textarea
                  v-model:value="selectedConfig.ai_config.system_extraction_prompt"
                  :rows="6"
                  :placeholder="t('admin.ruleConfig.systemExtractionPlaceholder')"
                />
              </div>
            </div>

            <!--用户提示词区域-->
            <div class="ai-prompt-section">
              <div class="ai-prompt-section-header">
                <div class="ai-prompt-section-tag ai-prompt-section-tag--user">{{ t('admin.ruleConfig.userPromptTag') }}</div>
                <a-button size="small" type="link" @click="resetUserPrompts">
                  <ReloadOutlined /> {{ t('admin.ruleConfig.resetUserPresets') }}
                </a-button>
              </div>
              <p class="ai-prompt-section-desc">{{ t('admin.ruleConfig.userPromptDesc') }}</p>

              <div class="ai-form-group">
                <div class="prompt-section-header">
                  <div class="prompt-section-title">
                    <span class="prompt-phase-badge prompt-phase-badge--reasoning">{{ t('admin.ruleConfig.phase1Label') }}</span>
                    <label class="ai-form-label">{{ t('admin.ruleConfig.userReasoningPrompt') }}</label>
                  </div>
                  <div class="prompt-section-desc">{{ t('admin.ruleConfig.userReasoningPromptDesc') }}</div>
                </div>
                <div class="prompt-variables">
                  <span class="prompt-variables-hint">{{ t('admin.ruleConfig.insertVariable') }}：</span>
                  <a-tooltip v-for="v in reasoningPromptVariables" :key="v.key" :title="v.desc">
                    <button
                      class="variable-btn"
                      @click="insertReasoningVariable(v.key)"
                    >{{ v.key }}</button>
                  </a-tooltip>
                </div>
                <a-textarea
                  ref="reasoningTextareaRef"
                  v-model:value="selectedConfig.ai_config.user_reasoning_prompt"
                  :rows="8"
                  :placeholder="t('admin.ruleConfig.userReasoningPlaceholder')"
                />
              </div>

              <div class="ai-form-group">
                <div class="prompt-section-header">
                  <div class="prompt-section-title">
                    <span class="prompt-phase-badge prompt-phase-badge--extraction">{{ t('admin.ruleConfig.phase2Label') }}</span>
                    <label class="ai-form-label">{{ t('admin.ruleConfig.userExtractionPrompt') }}</label>
                  </div>
                  <div class="prompt-section-desc">{{ t('admin.ruleConfig.userExtractionPromptDesc') }}</div>
                </div>
                <div class="prompt-variables">
                  <span class="prompt-variables-hint">{{ t('admin.ruleConfig.insertVariable') }}：</span>
                  <a-tooltip v-for="v in extractionPromptVariables" :key="v.key" :title="v.desc">
                    <button
                      class="variable-btn"
                      @click="insertExtractionVariable(v.key)"
                    >{{ v.key }}</button>
                  </a-tooltip>
                </div>
                <a-textarea
                  ref="extractionTextareaRef"
                  v-model:value="selectedConfig.ai_config.user_extraction_prompt"
                  :rows="6"
                  :placeholder="t('admin.ruleConfig.userExtractionPlaceholder')"
                />
              </div>
            </div>
          </div>
        </div>

        <!--========== 权限选项卡 ==========-->
        <div v-if="activeTab === 'permissions'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">{{ t('admin.ruleConfig.permTitle') }}</h4>
              <p class="section-desc">{{ t('admin.ruleConfig.permDesc') }}</p>
            </div>
          </div>

          <div class="permissions-list">
            <div
              v-for="(perm, key) in permissionLabels"
              :key="key"
              class="permission-item"
            >
              <div class="permission-info">
                <div class="permission-label">{{ perm.label }}</div>
                <div class="permission-desc">{{ perm.desc }}</div>
              </div>
              <a-switch
                v-model:checked="(selectedConfig.user_permissions as any)[key]"
                :checked-children="t('admin.ruleConfig.switchAllow')"
                :un-checked-children="t('admin.ruleConfig.switchDeny')"
              />
            </div>
          </div>
        </div>

        <div class="config-actions">
          <a-button type="primary" size="large" :disabled="saving" @click="handleSave">
            <LoadingOutlined v-if="saving" spin />
            <SaveOutlined v-else />
            {{ t('admin.ruleConfig.saveConfig') }}
          </a-button>
        </div>
      </div>

      <div v-else class="config-empty">
        <a-empty :description="t('admin.ruleConfig.selectProcess')" />
      </div>
    </div>

    <!--规则编辑器模式-->
    <RuleEditor
      :open="showRuleEditor"
      :rule="editingRule"
      @close="showRuleEditor = false; editingRule = null"
      @save="handleSaveRule"
    />

    <!--添加流程模态-->
    <a-modal
      v-model:open="showAddProcess"
      :title="t('admin.ruleConfig.addProcessTitle')"
      @ok="handleAddProcess"
      :ok-text="t('admin.ruleConfig.confirm')"
      :cancel-text="t('admin.ruleConfig.cancel')"
    >
      <a-form layout="vertical" style="margin-top: 16px;">
        <a-form-item :label="t('admin.ruleConfig.processName')" required>
          <a-input v-model:value="newProcessForm.process_type" :placeholder="t('admin.ruleConfig.processNamePlaceholder')" />
        </a-form-item>
        <a-form-item :label="t('admin.ruleConfig.processTypeLabel')">
          <a-input v-model:value="newProcessForm.process_type_label" :placeholder="t('admin.ruleConfig.processTypeLabelPlaceholder')" />
        </a-form-item>
        <a-form-item :label="t('admin.ruleConfig.mainTableName')">
          <div style="display: flex; gap: 8px;">
            <a-input v-model:value="newProcessForm.main_table_name" :placeholder="t('admin.ruleConfig.mainTableNamePlaceholder')" style="flex: 1;" />
            <a-button
              :loading="testingConnection"
              @click="handleTestConnectionInModal"
              :disabled="!newProcessForm.process_type.trim()"
            >
              <template #icon><DatabaseOutlined /></template>
              {{ testingConnection ? t('admin.ruleConfig.testingConnection') : t('admin.ruleConfig.testConnection') }}
            </a-button>
          </div>
          <div class="test-connection-hint" style="margin-top: 4px; font-size: 12px; color: var(--color-text-tertiary);">
            {{ t('admin.ruleConfig.testConnectionHint') }}
          </div>
          <div v-if="testConnectionResult" style="margin-top: 8px;">
            <a-alert
              :type="testConnectionResult.success ? 'success' : 'error'"
              :message="testConnectionResult.message"
              show-icon
              closable
              @close="testConnectionResult = null"
            />
          </div>
        </a-form-item>
      </a-form>
    </a-modal>

    <!--字段选择器模态-->
    <a-modal
      v-model:open="showFieldPicker"
      :title="t('admin.ruleConfig.selectFieldsModal')"
      :width="720"
      :footer="null"
      @cancel="showFieldPicker = false"
    >
      <div class="field-picker-modal">
        <div class="field-picker-left">
          <div class="field-picker-panel-header" style="justify-content: flex-start; gap: 8px;">
            <a-checkbox
              :checked="leftSelectedKeys.length === unselectedFieldsFlat.length && unselectedFieldsFlat.length > 0"
              :indeterminate="leftSelectedKeys.length > 0 && leftSelectedKeys.length < unselectedFieldsFlat.length"
              @change="toggleLeftSelectAll"
            />
            <span style="flex: 1;">{{ t('admin.ruleConfig.availableFields') }} <span class="field-count" style="margin-left:4px; font-weight:normal;">({{ unselectedFieldsFlat.length }})</span></span>
            <a-button type="primary" size="small" :disabled="leftSelectedKeys.length === 0" @click="batchPick">
              {{ t('admin.ruleConfig.add') }}
            </a-button>
          </div>
          <div class="field-picker-search">
            <a-input
              v-model:value="fieldSearchQuery"
              :placeholder="t('admin.ruleConfig.searchFieldPlaceholder')"
              allow-clear
              size="small"
            >
              <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
            </a-input>
          </div>
          <div class="field-picker-list" style="padding: 12px 16px;">
            <div
              v-for="field in unselectedPagination.paged.value"
              :key="field.field_key + field.source"
              class="field-picker-item"
              @click="toggleLeftSelect(field.field_key + '_' + field.source)"
              style="display: flex; gap: 12px; justify-content: flex-start; margin-bottom: 8px;"
            >
              <div class="field-picker-item-checkbox" @click.stop="toggleLeftSelect(field.field_key + '_' + field.source)">
                <a-checkbox :checked="leftSelectedKeys.includes(field.field_key + '_' + field.source)" />
              </div>
              <div class="field-picker-item-info" style="flex: 1;">
                <div class="field-picker-item-name">{{ field.field_name }} <span class="field-source-tag" style="font-size: 11px; color: var(--color-text-tertiary); font-weight: normal; margin-left: 4px;">({{ field.sourceLabel }})</span></div>
                <div class="field-picker-item-meta">
                  <span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span>
                  <span class="field-key">{{ field.field_key }}</span>
                </div>
              </div>
              <button class="icon-btn icon-btn--sm" @click.stop="pickField(field)" style="margin-left: auto;">
                <SwapRightOutlined />
              </button>
            </div>
            <div v-if="!unselectedFieldsFlat.length" class="field-picker-empty">
              {{ fieldSearchQuery ? t('admin.ruleConfig.noSearchResult') : t('admin.ruleConfig.allFieldsAdded') }}
            </div>
          </div>
          <div class="pagination-wrapper" style="padding: 12px 16px; border-top: 1px solid var(--color-border-light);">
            <a-pagination
              v-model:current="unselectedPagination.current.value"
              v-model:page-size="unselectedPagination.pageSize.value"
              :total="unselectedPagination.total.value"
              size="small"
              show-size-changer
              :page-size-options="['5', '20', '50']"
              @change="unselectedPagination.onChange"
              @showSizeChange="unselectedPagination.onChange"
            />
          </div>
        </div>
        <div class="field-picker-right">
          <div class="field-picker-panel-header" style="justify-content: flex-start; gap: 8px;">
            <a-checkbox
              :checked="rightSelectedKeys.length === selectedFieldsFlat.length && selectedFieldsFlat.length > 0"
              :indeterminate="rightSelectedKeys.length > 0 && rightSelectedKeys.length < selectedFieldsFlat.length"
              @change="toggleRightSelectAll"
            />
            <span style="flex: 1;">{{ t('admin.ruleConfig.selectedFields') }} <span class="field-picker-count" style="margin-left:4px;">{{ selectedFieldCount }}</span></span>
            <a-button danger size="small" :disabled="rightSelectedKeys.length === 0" @click="batchUnpick">
              {{ t('admin.ruleConfig.remove') }}
            </a-button>
          </div>
          <div class="field-picker-search">
            <a-input
              v-model:value="selectedFieldSearchQuery"
              :placeholder="t('admin.ruleConfig.searchFieldPlaceholder')"
              allow-clear
              size="small"
            >
              <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
            </a-input>
          </div>
          <div class="field-picker-list" style="padding: 12px 16px;">
            <div
              v-for="field in selectedPagination.paged.value"
              :key="field.field_key + field.source"
              class="field-picker-item field-picker-item--selected"
              @click="toggleRightSelect(field.field_key + '_' + field.source)"
              style="display: flex; gap: 12px; justify-content: flex-start; margin-bottom: 8px;"
            >
              <div class="field-picker-item-checkbox" @click.stop="toggleRightSelect(field.field_key + '_' + field.source)">
                <a-checkbox :checked="rightSelectedKeys.includes(field.field_key + '_' + field.source)" />
              </div>
              <div class="field-picker-item-info" style="flex: 1;">
                <div class="field-picker-item-name">{{ field.field_name }} <span class="field-source-tag" style="font-size: 11px; color: var(--color-text-tertiary); font-weight: normal; margin-left: 4px;">({{ field.sourceLabel }})</span></div>
                <div class="field-picker-item-meta">
                  <span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span>
                  <span class="field-key">{{ field.field_key }}</span>
                </div>
              </div>
              <button class="field-picker-remove" @click.stop="unpickField(field)" style="margin-left: auto;">
                <CloseOutlined />
              </button>
            </div>
            <div v-if="!selectedFieldsFlat.length" class="field-picker-empty">
              {{ selectedFieldSearchQuery ? t('admin.ruleConfig.noSearchResult') : t('admin.ruleConfig.noFieldsSelected') }}
            </div>
          </div>
          <div class="pagination-wrapper" style="padding: 12px 16px; border-top: 1px solid var(--color-border-light);">
            <a-pagination
              v-model:current="selectedPagination.current.value"
              v-model:page-size="selectedPagination.pageSize.value"
              :total="selectedPagination.total.value"
              size="small"
              show-size-changer
              :page-size-options="['5', '20', '50']"
              @change="selectedPagination.onChange"
              @showSizeChange="selectedPagination.onChange"
            />
          </div>
        </div>
      </div>
    </a-modal>

    <!-- ==================== 定时任务配置 ==================== -->
    <div v-if="topTab === 'cron'" class="main-layout">
      <!--左：任务类型列表-->
      <div class="process-nav">
        <div class="process-nav-header">
          <ClockCircleOutlined />
          <span>{{ t('admin.ruleConfig.cronTaskTypes') }}</span>
        </div>
        <div
          v-for="cfg in cronConfigs"
          :key="cfg.task_type"
          class="process-nav-item"
          :class="{ 'process-nav-item--active': selectedCronType === cfg.task_type }"
          @click="selectedCronType = cfg.task_type"
        >
          <div class="process-nav-name">{{ cfg.label_zh }}</div>
          <div class="process-nav-path">
            <span :class="cfg.is_enabled ? 'status-dot status-dot--active' : 'status-dot'" />
            {{ cfg.is_enabled ? t('admin.ruleConfig.cronEnabled') : t('admin.ruleConfig.cronDisabled') }}
          </div>
        </div>
      </div>

      <!--右：cron 配置面板-->
      <div v-if="selectedCronConfig" class="config-panel">
        <div class="config-panel-header" style="display: flex; justify-content: space-between; align-items: flex-start;">
          <div>
            <h2 class="config-panel-title">{{ selectedCronConfig.label_zh }}</h2>
            <p class="config-panel-subtitle">{{ selectedCronConfig.description_zh }}</p>
          </div>
          <a-switch
            :checked="selectedCronConfig.is_enabled"
            :checked-children="t('admin.ruleConfig.cronEnabled')"
            :un-checked-children="t('admin.ruleConfig.cronDisabled')"
            @change="(checked: any) => { if (checked) handleSaveCronConfig(); else handleResetCronTemplate(); }"
          />
        </div>

        <!--========== audit_batch / archive_batch：仅批量限制配置==========-->
        <div v-if="selectedCronConfig.task_type === 'audit_batch' || selectedCronConfig.task_type === 'archive_batch'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">{{ t('admin.ruleConfig.batchAuditConfigTitle') }}</h4>
              <p class="section-desc">{{ t('admin.ruleConfig.batchAuditConfigDesc') }}</p>
            </div>
          </div>
          <div class="ai-form">
            <div class="ai-form-group">
              <label class="ai-form-label">{{ t('admin.ruleConfig.batchLimitLabel') }}</label>
              <a-input-number
                v-model:value="selectedCronConfig.batch_limit"
                :min="1"
                :max="50"
                size="large"
                style="width: 200px;"
              />
              <p class="section-desc" style="margin-top: 4px;">{{ t('admin.ruleConfig.batchLimitDesc') }}</p>
            </div>
          </div>
        </div>

        <!--========== daily / weekly：带有变量插入的内容模板==========-->
        <div v-if="selectedCronConfig.task_type !== 'audit_batch' && selectedCronConfig.task_type !== 'archive_batch'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">{{ t('admin.ruleConfig.pushTemplateTitle') }}</h4>
              <p class="section-desc">{{ t('admin.ruleConfig.pushTemplateDesc') }}</p>
            </div>
          </div>

          <!--可变插入栏-->
          <div class="prompt-variables" style="margin-bottom: 16px;">
            <span class="prompt-variables-hint">{{ t('admin.ruleConfig.insertVariable') }}：</span>
            <a-tooltip v-for="v in cronTemplateVariables" :key="v.key" :title="v.desc">
              <button class="variable-btn" @click="insertCronVariable(v.key)">{{ v.key }}</button>
            </a-tooltip>
          </div>

          <!--推送格式-->
          <div class="ai-form-group" style="margin-bottom: 20px;">
            <label class="ai-form-label">{{ t('admin.ruleConfig.pushFormatLabel') }}</label>
            <div class="push-format-options">
              <div
                v-for="fmt in pushFormatOptions"
                :key="fmt.value"
                class="push-format-option"
                :class="{ 'push-format-option--active': selectedCronConfig.push_format === fmt.value }"
                @click="selectedCronConfig.push_format = fmt.value as any"
              >
                <div class="push-format-radio" />
                <span>{{ fmt.label }}</span>
              </div>
            </div>
          </div>

          <div class="ai-form">
            <div class="ai-form-group">
              <label class="ai-form-label">{{ t('admin.ruleConfig.emailSubjectLabel') }}</label>
              <a-input
                ref="cronSubjectRef"
                v-model:value="selectedCronConfig.content_template.subject"
                size="large"
                :placeholder="t('admin.ruleConfig.emailSubjectPlaceholder')"
                @focus="cronActiveField = 'subject'"
              />
            </div>
            <div class="ai-form-group">
              <label class="ai-form-label">{{ t('admin.ruleConfig.emailHeaderLabel') }}</label>
              <a-input
                ref="cronHeaderRef"
                v-model:value="selectedCronConfig.content_template.header"
                size="large"
                :placeholder="t('admin.ruleConfig.emailHeaderPlaceholder')"
                @focus="cronActiveField = 'header'"
              />
            </div>
            <div class="ai-form-group">
              <label class="ai-form-label">{{ t('admin.ruleConfig.emailBodyLabel') }}</label>
              <a-textarea
                ref="cronBodyRef"
                v-model:value="selectedCronConfig.content_template.body_template"
                :rows="6"
                :placeholder="t('admin.ruleConfig.emailBodyPlaceholder')"
                @focus="cronActiveField = 'body_template'"
              />
            </div>
            <div class="ai-form-group">
              <label class="ai-form-label">{{ t('admin.ruleConfig.emailFooterLabel') }}</label>
              <a-input
                ref="cronFooterRef"
                v-model:value="selectedCronConfig.content_template.footer"
                size="large"
                :placeholder="t('admin.ruleConfig.emailFooterPlaceholder')"
                @focus="cronActiveField = 'footer'"
              />
            </div>
          </div>
        </div>

        <div class="config-actions">
          <a-button type="primary" size="large" :disabled="savingCron" @click="handleSaveCronConfig">
            <LoadingOutlined v-if="savingCron" spin />
            <SaveOutlined v-else />
            {{ t('admin.ruleConfig.cronSaveConfig') }}
          </a-button>
          <a-popconfirm
            :title="t('admin.ruleConfig.cronResetConfirm')"
            @confirm="handleResetCronTemplate"
          >
            <a-button size="large" style="margin-left: 12px;">
              <ReloadOutlined />
              {{ t('admin.ruleConfig.cronResetBtn') }}
            </a-button>
          </a-popconfirm>
        </div>
      </div>

      <div v-else class="config-empty">
        <a-empty :description="t('admin.ruleConfig.cronSelectHint')" />
      </div>
    </div>

    <!-- ==================== 归档复盘配置 ==================== -->
    <div v-if="topTab === 'archive'" class="main-layout">
      <!--左：进程列表-->
      <div class="process-nav">
        <div class="process-nav-header">
          <FolderOpenOutlined />
          <span>{{ t('admin.ruleConfig.archiveProcess') }}</span>
          <button class="add-process-btn" @click="showAddArchiveProcess = true" :title="t('admin.ruleConfig.addArchiveProcess')">
            <PlusOutlined />
          </button>
        </div>
        <div
          v-for="cfg in archiveConfigs"
          :key="cfg.id"
          class="process-nav-item"
          :class="{ 'process-nav-item--active': selectedArchiveId === cfg.id }"
          @click="selectedArchiveId = cfg.id"
        >
          <div class="process-nav-name">{{ cfg.process_type }}</div>
          <div v-if="cfg.process_type_label" class="process-nav-path">{{ cfg.process_type_label }}</div>
        </div>
      </div>

      <!--右：存档配置面板-->
      <div v-if="selectedArchiveConfig" class="config-panel">
        <div class="config-panel-header">
          <h2 class="config-panel-title">{{ selectedArchiveConfig.process_type }}</h2>
          <p v-if="selectedArchiveConfig.process_type_label" class="config-panel-subtitle">{{ selectedArchiveConfig.process_type_label }}</p>
        </div>

        <!--子选项卡：删除霓虹流规则，与审核工作台景观-->
        <div class="tab-nav">
          <button
            v-for="tab in [
              { key: 'info', label: t('admin.ruleConfig.infoTab'), icon: InfoCircleOutlined },
              { key: 'fields', label: t('admin.ruleConfig.tabFields'), icon: AppstoreOutlined },
              { key: 'rules', label: t('admin.ruleConfig.tabRules'), icon: AuditOutlined },
              { key: 'ai', label: t('admin.ruleConfig.tabAI'), icon: RobotOutlined },
              { key: 'permissions', label: t('admin.ruleConfig.tabPerms'), icon: SafetyCertificateOutlined },
            ]"
            :key="tab.key"
            class="tab-btn"
            :class="{ 'tab-btn--active': archiveActiveTab === tab.key }"
            @click="archiveActiveTab = tab.key"
          >
            <component :is="tab.icon" />
            {{ tab.label }}
          </button>
        </div>

        <!--========== 信息选项卡 ==========-->
        <div v-if="archiveActiveTab === 'info'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">{{ t('admin.ruleConfig.infoTitle') }}</h4>
              <p class="section-desc">{{ t('admin.ruleConfig.archiveInfoDesc') }}</p>
            </div>
          </div>
          <a-form layout="vertical" class="info-form">
            <a-form-item :label="t('admin.ruleConfig.processNameLabel')">
              <a-input v-model:value="selectedArchiveConfig!.process_type" :placeholder="t('admin.ruleConfig.processNameInputPlaceholder')" />
            </a-form-item>
            <a-form-item :label="t('admin.ruleConfig.processTypeLabel')">
              <a-input
                :value="selectedArchiveConfig!.process_type_label ?? ''"
                @update:value="(v: string) => { if (selectedArchiveConfig) selectedArchiveConfig.process_type_label = v }"
                :placeholder="t('admin.ruleConfig.processTypeLabelPlaceholder')"
              />
            </a-form-item>
            <a-form-item :label="t('admin.ruleConfig.mainTableLabel')">
              <div style="display: flex; gap: 8px;">
                <a-input v-model:value="selectedArchiveConfig!.main_table_name" :placeholder="t('admin.ruleConfig.mainTableInputPlaceholder')" style="flex: 1;" />
                <a-button
                  :loading="archiveInfoTestingConnection"
                  @click="handleArchiveTestConnectionInInfo"
                  :disabled="!selectedArchiveConfig!.process_type.trim()"
                >
                  <template #icon><DatabaseOutlined /></template>
                  {{ archiveInfoTestingConnection ? t('admin.ruleConfig.testingConnection') : t('admin.ruleConfig.testConnection') }}
                </a-button>
              </div>
              <div class="test-connection-hint" style="margin-top: 4px; font-size: 12px; color: var(--color-text-tertiary);">
                {{ t('admin.ruleConfig.testConnectionHint') }}
              </div>
              <div v-if="archiveInfoTestConnectionResult" style="margin-top: 8px;">
                <a-alert
                  :type="archiveInfoTestConnectionResult.success ? 'success' : 'error'"
                  :message="archiveInfoTestConnectionResult.message"
                  show-icon
                  closable
                  @close="archiveInfoTestConnectionResult = null"
                />
              </div>
            </a-form-item>
          </a-form>
        </div>

        <!--========== 字段选项卡 ==========-->
        <div v-if="archiveActiveTab === 'fields'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">{{ t('admin.ruleConfig.fieldTitle') }}</h4>
              <p class="section-desc">{{ t('admin.ruleConfig.archiveFieldDesc') }}</p>
            </div>
          </div>

          <div class="field-mode-switch">
            <div
              class="field-mode-option"
              :class="{ 'field-mode-option--active': selectedArchiveConfig.field_mode === 'selected' }"
              @click="selectedArchiveConfig.field_mode = 'selected'"
            >
              <div class="field-mode-radio" />
              <div>
                <div class="field-mode-label">{{ t('admin.ruleConfig.selectFields') }}</div>
                <div class="field-mode-desc">{{ t('admin.ruleConfig.selectFieldsDesc') }}</div>
              </div>
            </div>
            <div
              class="field-mode-option"
              :class="{ 'field-mode-option--active': selectedArchiveConfig.field_mode === 'all' }"
              @click="selectedArchiveConfig.field_mode = 'all'"
            >
              <div class="field-mode-radio" />
              <div>
                <div class="field-mode-label">{{ t('admin.ruleConfig.allFields') }}</div>
                <div class="field-mode-desc">{{ t('admin.ruleConfig.allFieldsDesc') }}</div>
              </div>
            </div>
          </div>

          <!--选定字段显示+选择器触发器-->
          <template v-if="selectedArchiveConfig.field_mode === 'selected'">
            <div class="field-picker-toolbar">
              <span class="field-count">{{ t('admin.ruleConfig.selectedCount', [`${archiveSelectedFieldCount}`, `${archiveAllAvailableFields.length}`]) }}</span>
              <a-button type="primary" @click="openArchiveFieldPicker">
                <AppstoreOutlined /> {{ t('admin.ruleConfig.selectFieldsModal') }}
              </a-button>
            </div>

            <!--按表分组的选定字段-->
            <template v-if="archiveGroupedSelected.length">
              <div v-for="group in archiveGroupedSelected" :key="group.source" class="selected-field-group">
                <div class="field-group-label">{{ group.sourceLabel }}</div>
                <div class="selected-fields-display">
                  <div
                    v-for="field in group.fields"
                    :key="field.field_key + field.source"
                    class="selected-field-tag"
                  >
                    <span class="selected-field-name">{{ field.field_name }}</span>
                    <span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span>
                  </div>
                </div>
              </div>
            </template>
            <div v-else class="field-empty-hint">
              {{ t('admin.ruleConfig.noFieldsSelected') }}
            </div>
          </template>

          <template v-else>
            <div class="field-count" style="margin-top: 8px;">
              {{ t('admin.ruleConfig.allFieldsHint') }}
            </div>
          </template>
        </div>

        <!--========== 规则选项卡 ==========-->
        <div v-if="archiveActiveTab === 'rules'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">{{ t('admin.ruleConfig.rulesTitle') }}</h4>
              <p class="section-desc">{{ t('admin.ruleConfig.reviewRulesDesc') }}</p>
            </div>
          </div>

          <!--KB 模式选择器-->
          <div class="kb-modes">
            <div
              v-for="mode in kbModes"
              :key="mode.key"
              class="kb-mode-card"
              :class="{
                'kb-mode-card--active': selectedArchiveConfig.kb_mode === mode.key,
                'kb-mode-card--disabled': !mode.available,
              }"
              @click="mode.available && (selectedArchiveConfig.kb_mode = mode.key as any)"
            >
              <div class="kb-mode-icon"><component :is="mode.icon" /></div>
              <div class="kb-mode-info">
                <div class="kb-mode-title">{{ mode.title }}</div>
                <div class="kb-mode-desc">{{ mode.desc }}</div>
              </div>
              <div v-if="selectedArchiveConfig.kb_mode === mode.key" class="kb-mode-check">✓</div>
              <div v-if="!mode.available" class="kb-mode-badge">{{ t('admin.ruleConfig.comingSoon') }}</div>
            </div>
          </div>

          <div class="rules-toolbar">
            <span class="rules-count">{{ t('admin.ruleConfig.totalRules', `${currentArchiveRules.length}`) }}</span>
            <div class="rules-toolbar-actions">
              <a-button @click="handleImportRules">
                <UploadOutlined /> {{ t('admin.ruleConfig.fileImport') }}
              </a-button>
              <a-button type="primary" @click="openArchiveRuleEditor()">
                <PlusOutlined /> {{ t('admin.ruleConfig.manualAdd') }}
              </a-button>
            </div>
          </div>

          <div class="rules-list">
            <div v-if="loadingArchiveRules" class="rules-loading">
              <a-spin size="small" />
            </div>
            <div v-for="rule in currentArchiveRules" :key="rule.id" class="rule-card">
              <div class="rule-card-left">
                <div class="rule-scope-badge" :style="{ color: scopeConfig[rule.rule_scope]?.color, background: scopeConfig[rule.rule_scope]?.bg }">
                  <component :is="scopeConfig[rule.rule_scope]?.icon" />
                  {{ scopeConfig[rule.rule_scope]?.label }}
                </div>
                <div class="rule-card-body">
                  <div class="rule-card-content">{{ rule.rule_content }}</div>
                  <div class="rule-card-meta">
                    <span v-if="rule.related_flow" class="rule-flow-tag">
                      <NodeIndexOutlined /> {{ t('admin.ruleConfig.relatedFlow') }}
                    </span>
                    <span v-if="rule.source === 'file_import'" class="rule-source-tag">{{ t('admin.ruleConfig.fileImportTag') }}</span>
                    <span v-else class="rule-source-tag rule-source-tag--manual">{{ t('admin.ruleConfig.manualAddTag') }}</span>
                  </div>
                </div>
              </div>
              <div class="rule-card-actions">
                <a-switch v-model:checked="rule.enabled" size="small" />
                <button class="icon-btn" @click="openArchiveRuleEditor(rule)"><EditOutlined /></button>
                <a-popconfirm :title="t('admin.ruleConfig.deleteRuleConfirm')" @confirm="deleteArchiveRule(rule.id)">
                  <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
                </a-popconfirm>
              </div>
            </div>
          </div>
        </div>

        <!--========== AI选项卡（两级提示）==========-->
        <div v-if="archiveActiveTab === 'ai'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">{{ t('admin.ruleConfig.aiTitle') }}</h4>
              <p class="section-desc">{{ t('admin.ruleConfig.aiDescNew') }}</p>
            </div>
          </div>

          <div class="ai-form">
            <!--严格性-->
            <div class="ai-form-group">
              <label class="ai-form-label">{{ t('admin.ruleConfig.strictness') }}</label>
              <div class="strictness-options">
                <div
                  v-for="opt in strictnessOptions"
                  :key="opt.value"
                  class="strictness-option"
                  :class="{ 'strictness-option--active': selectedArchiveConfig.ai_config.audit_strictness === opt.value }"
                  @click="handleArchiveStrictnessChange(opt.value)"
                >
                  <div class="strictness-option-radio" />
                  <div>
                    <div class="strictness-option-label">{{ opt.label }}</div>
                    <div class="strictness-option-desc">{{ opt.desc }}</div>
                  </div>
                </div>
              </div>
              <!--当前尺度标签-->
              <div class="strictness-hint">
                {{ t('admin.ruleConfig.strictnessHint') }}
              </div>
            </div>

            <!--系统提示词区域-->
            <div class="ai-prompt-section">
              <div class="ai-prompt-section-header">
                <div class="ai-prompt-section-tag ai-prompt-section-tag--system">{{ t('admin.ruleConfig.systemPromptTag') }}</div>
                <a-button size="small" type="link" @click="resetArchiveSystemPrompts">
                  <ReloadOutlined /> {{ t('admin.ruleConfig.resetSystemPresets') }}
                </a-button>
              </div>
              <p class="ai-prompt-section-desc">{{ t('admin.ruleConfig.systemPromptDesc') }}</p>

              <div class="ai-form-group">
                <div class="prompt-section-header">
                  <div class="prompt-section-title">
                    <span class="prompt-phase-badge prompt-phase-badge--reasoning">{{ t('admin.ruleConfig.phase1Label') }}</span>
                    <label class="ai-form-label">{{ t('admin.ruleConfig.systemReasoningPrompt') }}</label>
                  </div>
                </div>
                <a-textarea
                  v-model:value="selectedArchiveConfig.ai_config.system_reasoning_prompt"
                  :rows="6"
                  :placeholder="t('admin.ruleConfig.systemReasoningPlaceholder')"
                />
              </div>

              <div class="ai-form-group">
                <div class="prompt-section-header">
                  <div class="prompt-section-title">
                    <span class="prompt-phase-badge prompt-phase-badge--extraction">{{ t('admin.ruleConfig.phase2Label') }}</span>
                    <label class="ai-form-label">{{ t('admin.ruleConfig.systemExtractionPrompt') }}</label>
                  </div>
                </div>
                <a-textarea
                  v-model:value="selectedArchiveConfig.ai_config.system_extraction_prompt"
                  :rows="6"
                  :placeholder="t('admin.ruleConfig.systemExtractionPlaceholder')"
                />
              </div>
            </div>

            <!--用户提示词区域-->
            <div class="ai-prompt-section">
              <div class="ai-prompt-section-header">
                <div class="ai-prompt-section-tag ai-prompt-section-tag--user">{{ t('admin.ruleConfig.userPromptTag') }}</div>
                <a-button size="small" type="link" @click="resetArchiveUserPrompts">
                  <ReloadOutlined /> {{ t('admin.ruleConfig.resetUserPresets') }}
                </a-button>
              </div>
              <p class="ai-prompt-section-desc">{{ t('admin.ruleConfig.userPromptDesc') }}</p>

              <div class="ai-form-group">
                <div class="prompt-section-header">
                  <div class="prompt-section-title">
                    <span class="prompt-phase-badge prompt-phase-badge--reasoning">{{ t('admin.ruleConfig.phase1Label') }}</span>
                    <label class="ai-form-label">{{ t('admin.ruleConfig.userReasoningPrompt') }}</label>
                  </div>
                  <div class="prompt-section-desc">{{ t('admin.ruleConfig.userReasoningPromptDesc') }}</div>
                </div>
                <div class="prompt-variables">
                  <span class="prompt-variables-hint">{{ t('admin.ruleConfig.insertVariable') }}：</span>
                  <a-tooltip v-for="v in archiveReasoningPromptVariables" :key="v.key" :title="v.desc">
                    <button class="variable-btn" @click="insertArchiveAtCursor(archiveReasoningTextareaRef, 'user_reasoning_prompt', v.key)">{{ v.key }}</button>
                  </a-tooltip>
                </div>
                <a-textarea
                  ref="archiveReasoningTextareaRef"
                  v-model:value="selectedArchiveConfig.ai_config.user_reasoning_prompt"
                  :rows="8"
                  :placeholder="t('admin.ruleConfig.userReasoningPlaceholder')"
                />
              </div>

              <div class="ai-form-group">
                <div class="prompt-section-header">
                  <div class="prompt-section-title">
                    <span class="prompt-phase-badge prompt-phase-badge--extraction">{{ t('admin.ruleConfig.phase2Label') }}</span>
                    <label class="ai-form-label">{{ t('admin.ruleConfig.userExtractionPrompt') }}</label>
                  </div>
                  <div class="prompt-section-desc">{{ t('admin.ruleConfig.userExtractionPromptDesc') }}</div>
                </div>
                <div class="prompt-variables">
                  <span class="prompt-variables-hint">{{ t('admin.ruleConfig.insertVariable') }}：</span>
                  <a-tooltip v-for="v in archiveExtractionPromptVariables" :key="v.key" :title="v.desc">
                    <button class="variable-btn" @click="insertArchiveAtCursor(archiveExtractionTextareaRef, 'user_extraction_prompt', v.key)">{{ v.key }}</button>
                  </a-tooltip>
                </div>
                <a-textarea
                  ref="archiveExtractionTextareaRef"
                  v-model:value="selectedArchiveConfig.ai_config.user_extraction_prompt"
                  :rows="6"
                  :placeholder="t('admin.ruleConfig.userExtractionPlaceholder')"
                />
              </div>
            </div>
          </div>
        </div>

        <!--========== 权限选项卡（用户自定义权限 + 访问控制）==========-->
        <div v-if="archiveActiveTab === 'permissions'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">{{ t('admin.ruleConfig.archivePermTitle') }}</h4>
              <p class="section-desc">{{ t('admin.ruleConfig.archivePermDesc') }}</p>
            </div>
          </div>

          <div class="permissions-list">
            <div v-for="(perm, key) in archivePermissionLabels" :key="key" class="permission-item">
              <div class="permission-info">
                <div class="permission-label">{{ perm.label }}</div>
                <div class="permission-desc">{{ perm.desc }}</div>
              </div>
              <a-switch
                v-model:checked="(selectedArchiveConfig.user_permissions as any)[key]"
                :checked-children="t('admin.ruleConfig.allow')"
                :un-checked-children="t('admin.ruleConfig.deny')"
              />
            </div>
          </div>

          <!-- 访问控制 -->
          <div class="section-header" style="margin-top: 28px;">
            <div>
              <h4 class="section-title">{{ t('admin.ruleConfig.archiveAccessTitle') }}</h4>
              <p class="section-desc">{{ t('admin.ruleConfig.archiveAccessDesc') }}</p>
            </div>
          </div>

          <div class="access-control-section">
            <div class="access-control-group">
              <div class="access-control-label"><TeamOutlined /> {{ t('admin.ruleConfig.archiveAllowedRoles') }}</div>
              <div class="access-control-search">
                <a-input v-model:value="archiveRoleSearch" :placeholder="t('admin.ruleConfig.archiveAccessSearch')" allow-clear size="small" style="max-width: 280px;">
                  <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
                </a-input>
              </div>
              <div class="access-control-tags" style="gap: 8px;">
                <div
                  v-for="role in filteredArchiveRoles"
                  :key="role.id"
                  class="access-tag"
                  :class="{ 'access-tag--active': (selectedArchiveConfig.access_control?.allowed_roles || []).includes(role.id) }"
                  @click="toggleArchiveRole(role.id)"
                >
                  <CheckOutlined v-if="(selectedArchiveConfig.access_control?.allowed_roles || []).includes(role.id)" class="access-tag-check" />
                  {{ role.name }}
                </div>
              </div>
            </div>
            <div class="access-control-group" style="margin-top: 16px;">
              <div class="access-control-label"><UserOutlined /> {{ t('admin.ruleConfig.archiveAllowedMembers') }}</div>
              <div class="access-control-search">
                <a-input v-model:value="archiveMemberSearch" :placeholder="t('admin.ruleConfig.archiveAccessSearch')" allow-clear size="small" style="max-width: 280px;">
                  <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
                </a-input>
              </div>
              <div class="access-control-tags" style="gap: 8px;">
                <div
                  v-for="member in filteredArchiveMembers"
                  :key="member.id"
                  class="access-tag"
                  :class="{ 'access-tag--active': (selectedArchiveConfig.access_control?.allowed_members || []).includes(member.id) }"
                  @click="toggleArchiveMember(member.id)"
                >
                  <CheckOutlined v-if="(selectedArchiveConfig.access_control?.allowed_members || []).includes(member.id)" class="access-tag-check" />
                  {{ member.name }}
                  <span class="access-tag-dept">{{ member.department_name }}</span>
                </div>
              </div>
            </div>
            <div class="access-control-group" style="margin-top: 16px;">
              <div class="access-control-label"><AppstoreOutlined /> {{ t('admin.ruleConfig.archiveAllowedDepts') }}</div>
              <div class="access-control-search">
                <a-input v-model:value="archiveDeptSearch" :placeholder="t('admin.ruleConfig.archiveAccessSearch')" allow-clear size="small" style="max-width: 280px;">
                  <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
                </a-input>
              </div>
              <div class="access-control-tags" style="gap: 8px;">
                <div
                  v-for="dept in filteredArchiveDepts"
                  :key="dept.id"
                  class="access-tag"
                  :class="{ 'access-tag--active': (selectedArchiveConfig.access_control?.allowed_departments || []).includes(dept.id) }"
                  @click="toggleArchiveDept(dept.id)"
                >
                  <CheckOutlined v-if="(selectedArchiveConfig.access_control?.allowed_departments || []).includes(dept.id)" class="access-tag-check" />
                  {{ dept.name }}
                  <span class="access-tag-dept">{{ dept.member_count }}人</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div class="config-actions">
          <a-button type="primary" size="large" :disabled="savingArchive" @click="handleSaveArchiveConfig">
            <LoadingOutlined v-if="savingArchive" spin />
            <SaveOutlined v-else />
            {{ t('admin.ruleConfig.saveConfig') }}
          </a-button>
        </div>
      </div>

      <div v-else class="config-empty">
        <a-empty :description="t('admin.ruleConfig.selectArchiveProcess')" />
      </div>
    </div>

    <!--存档规则编辑器模式-->
    <RuleEditor
      :open="showArchiveRuleEditor"
      :rule="editingArchiveRule"
      @close="showArchiveRuleEditor = false; editingArchiveRule = null"
      @save="handleSaveArchiveRule"
    />

    <!--添加归档流程模式-->
    <a-modal
      v-model:open="showAddArchiveProcess"
      :title="t('admin.ruleConfig.addArchiveProcessTitle')"
      @ok="handleAddArchiveProcess"
      :ok-text="t('admin.ruleConfig.confirm')"
      :cancel-text="t('admin.ruleConfig.cancel')"
    >
      <a-form layout="vertical" style="margin-top: 16px;">
        <a-form-item :label="t('admin.ruleConfig.processName')" required>
          <a-input v-model:value="newArchiveProcessForm.process_type" :placeholder="t('admin.ruleConfig.processNamePlaceholder')" />
        </a-form-item>
        <a-form-item :label="t('admin.ruleConfig.processTypeLabel')">
          <a-input v-model:value="newArchiveProcessForm.process_type_label" :placeholder="t('admin.ruleConfig.processTypeLabelPlaceholder')" />
        </a-form-item>
        <a-form-item :label="t('admin.ruleConfig.mainTableName')">
          <div style="display: flex; gap: 8px;">
            <a-input v-model:value="newArchiveProcessForm.main_table_name" :placeholder="t('admin.ruleConfig.mainTableNamePlaceholder')" style="flex: 1;" />
            <a-button
              :loading="archiveTestingConnection"
              @click="handleTestConnectionInArchiveModal"
              :disabled="!newArchiveProcessForm.process_type.trim()"
            >
              <template #icon><DatabaseOutlined /></template>
              {{ archiveTestingConnection ? t('admin.ruleConfig.testingConnection') : t('admin.ruleConfig.testConnection') }}
            </a-button>
          </div>
          <div class="test-connection-hint" style="margin-top: 4px; font-size: 12px; color: var(--color-text-tertiary);">
            {{ t('admin.ruleConfig.testConnectionHint') }}
          </div>
          <div v-if="archiveTestConnectionResult" style="margin-top: 8px;">
            <a-alert
              :type="archiveTestConnectionResult.success ? 'success' : 'error'"
              :message="archiveTestConnectionResult.message"
              show-icon
              closable
              @close="archiveTestConnectionResult = null"
            />
          </div>
        </a-form-item>
      </a-form>
    </a-modal>

    <!--归档字段选择器模式-->
    <a-modal
      v-model:open="showArchiveFieldPicker"
      :title="t('admin.ruleConfig.selectFieldsModal')"
      :width="720"
      :footer="null"
      @cancel="showArchiveFieldPicker = false"
    >
      <div class="field-picker-modal">
        <div class="field-picker-left">
          <div class="field-picker-panel-header">
            <span>{{ t('admin.ruleConfig.availableFields') }}</span>
          </div>
          <div class="field-picker-search">
            <a-input
              v-model:value="archiveFieldSearchQuery"
              :placeholder="t('admin.ruleConfig.searchFieldPlaceholder')"
              allow-clear
              size="small"
            >
              <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
            </a-input>
          </div>
          <div class="field-picker-list">
            <template v-for="group in archiveGroupedUnselected" :key="group.source">
              <div class="field-picker-group-label">{{ group.sourceLabel }}</div>
              <div
                v-for="field in group.fields"
                :key="field.field_key + field.source"
                class="field-picker-item"
                @click="archivePickField(field)"
              >
                <div class="field-picker-item-info">
                  <div class="field-picker-item-name">{{ field.field_name }}</div>
                  <div class="field-picker-item-meta">
                    <span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span>
                    <span class="field-key">{{ field.field_key }}</span>
                  </div>
                </div>
                <SwapRightOutlined class="field-picker-arrow" />
              </div>
            </template>
            <div v-if="!archiveGroupedUnselected.length" class="field-picker-empty">
              {{ archiveFieldSearchQuery ? t('admin.ruleConfig.noSearchResult') : t('admin.ruleConfig.allFieldsAdded') }}
            </div>
          </div>
        </div>
        <div class="field-picker-right">
          <div class="field-picker-panel-header">
            <span>{{ t('admin.ruleConfig.selectedFields') }}</span>
            <span class="field-picker-count">{{ archiveSelectedFieldCount }}</span>
          </div>
          <div class="field-picker-list">
            <template v-for="group in archiveGroupedSelected" :key="group.source">
              <div class="field-picker-group-label">{{ group.sourceLabel }}</div>
              <div
                v-for="field in group.fields"
                :key="field.field_key + field.source"
                class="field-picker-item field-picker-item--selected"
              >
                <div class="field-picker-item-info">
                  <div class="field-picker-item-name">{{ field.field_name }}</div>
                  <div class="field-picker-item-meta">
                    <span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span>
                    <span class="field-key">{{ field.field_key }}</span>
                  </div>
                </div>
                <button class="field-picker-remove" @click="archiveUnpickField(field)">
                  <CloseOutlined />
                </button>
              </div>
            </template>
            <div v-if="!archiveGroupedSelected.length" class="field-picker-empty">
              {{ t('admin.ruleConfig.noFieldsSelected') }}
            </div>
          </div>
        </div>
      </div>
    </a-modal>

  </div>
</template>

<style scoped>
.page-header { margin-bottom: 24px; }
.page-title { font-size: 24px; font-weight: 700; color: var(--color-text-primary); margin: 0; }
.page-subtitle { font-size: 14px; color: var(--color-text-tertiary); margin: 4px 0 0; }

/*顶级选项卡*/
.top-tab-nav {
  display: flex; gap: 4px; background: var(--color-bg-hover); padding: 4px;
  border-radius: var(--radius-lg); margin-bottom: 24px; width: fit-content;
}
.top-tab-btn {
  display: flex; align-items: center; gap: 6px;
  padding: 8px 24px; border: none; background: transparent; border-radius: var(--radius-md);
  font-size: 14px; font-weight: 500; color: var(--color-text-secondary); cursor: pointer;
  transition: all var(--transition-fast);
}
.top-tab-btn:hover { color: var(--color-text-primary); }
.top-tab-btn--active { background: var(--color-bg-card); color: var(--color-primary); box-shadow: var(--shadow-xs); }

/*主要布局*/
.main-layout { display: grid; grid-template-columns: 240px 1fr; gap: 20px; align-items: start; }

/*流程导航*/
.process-nav {
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); overflow: hidden; position: sticky; top: 20px;
}
.process-nav-header {
  padding: 14px 16px; border-bottom: 1px solid var(--color-border-light);
  font-size: 14px; font-weight: 600; color: var(--color-text-primary);
  display: flex; align-items: center; gap: 8px;
}
.add-process-btn {
  margin-left: auto; width: 26px; height: 26px; border-radius: var(--radius-md);
  border: 1px dashed var(--color-border); background: transparent; cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  color: var(--color-text-tertiary); font-size: 12px; transition: all var(--transition-fast);
}
.add-process-btn:hover { border-color: var(--color-primary); color: var(--color-primary); background: var(--color-primary-bg); }
.process-nav-item {
  padding: 12px 16px; cursor: pointer; transition: all var(--transition-fast);
  border-bottom: 1px solid var(--color-border-light);
  display: flex; align-items: center; gap: 8px;
}
.process-nav-item:last-child { border-bottom: none; }
.process-nav-item:hover { background: var(--color-bg-hover); }
.process-nav-item--active { background: var(--color-primary-bg); border-left: 3px solid var(--color-primary); }
.process-nav-name { font-size: 14px; font-weight: 500; color: var(--color-text-primary); margin-bottom: 2px; }
.process-nav-path { font-size: 12px; color: var(--color-text-tertiary); }

/*配置面板*/
.config-panel {
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); padding: 24px;
}
.config-panel-header { margin-bottom: 20px; }
.config-panel-title { font-size: 18px; font-weight: 600; color: var(--color-text-primary); margin: 0 0 4px; }
.config-panel-subtitle { font-size: 13px; color: var(--color-text-tertiary); margin: 0; }
.config-empty {
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); padding: 48px;
}

/*选项卡*/
.tab-nav {
  display: flex; gap: 4px; background: var(--color-bg-hover); padding: 4px;
  border-radius: var(--radius-lg); margin-bottom: 24px; width: fit-content;
}
.tab-btn {
  display: flex; align-items: center; gap: 6px;
  padding: 8px 20px; border: none; background: transparent; border-radius: var(--radius-md);
  font-size: 14px; font-weight: 500; color: var(--color-text-secondary); cursor: pointer;
  transition: all var(--transition-fast);
}
.tab-btn:hover { color: var(--color-text-primary); }
.tab-btn--active { background: var(--color-bg-card); color: var(--color-primary); box-shadow: var(--shadow-xs); }

/*部分*/
.section-header { margin-bottom: 16px; }
.section-title { font-size: 15px; font-weight: 600; color: var(--color-text-primary); margin: 0 0 4px; }
.section-desc { font-size: 13px; color: var(--color-text-tertiary); margin: 0; }

/*现场模式开关*/
.field-mode-switch { display: flex; gap: 12px; margin-bottom: 16px; }
.field-mode-option {
  display: flex; align-items: center; gap: 12px; padding: 12px 16px; flex: 1;
  border: 2px solid var(--color-border-light); border-radius: var(--radius-md);
  cursor: pointer; transition: all var(--transition-fast);
}
.field-mode-option:hover { border-color: var(--color-primary-lighter); }
.field-mode-option--active { border-color: var(--color-primary); background: var(--color-primary-bg); }
.field-mode-radio {
  width: 18px; height: 18px; border-radius: 50%; border: 2px solid var(--color-border);
  flex-shrink: 0; transition: all var(--transition-fast);
}
.field-mode-option--active .field-mode-radio { border-color: var(--color-primary); border-width: 5px; }
.field-mode-label { font-size: 14px; font-weight: 500; color: var(--color-text-primary); }
.field-mode-desc { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }

.field-count { font-size: 13px; color: var(--color-text-tertiary); margin-bottom: 12px; }

/*场网格*/
.field-type-tag {
  font-size: 10px; font-weight: 600; padding: 1px 6px; border-radius: var(--radius-sm);
  background: var(--color-bg-hover); color: var(--color-text-tertiary);
}
.field-key { font-size: 11px; color: var(--color-text-tertiary); font-family: monospace; }

/*规则*/
.rules-toolbar {
  display: flex; justify-content: space-between; align-items: center; margin-bottom: 14px;
}
.rules-count { font-size: 13px; color: var(--color-text-tertiary); }
.rules-toolbar-actions { display: flex; gap: 8px; }

.rules-list { display: flex; flex-direction: column; gap: 10px; }
.rule-card {
  display: flex; align-items: center; justify-content: space-between;
  padding: 14px 18px; background: var(--color-bg-page); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); transition: all var(--transition-fast); gap: 16px;
}
.rule-card:hover { box-shadow: var(--shadow-sm); }
.rule-card-left { display: flex; align-items: flex-start; gap: 12px; flex: 1; min-width: 0; }
.rule-scope-badge {
  display: inline-flex; align-items: center; gap: 4px; font-size: 11px; font-weight: 600;
  padding: 4px 10px; border-radius: var(--radius-full); white-space: nowrap; flex-shrink: 0;
}
.rule-card-content { font-size: 14px; font-weight: 500; color: var(--color-text-primary); margin-bottom: 4px; }
.rule-card-meta { display: flex; align-items: center; gap: 8px; font-size: 12px; color: var(--color-text-tertiary); }
.rule-source-tag {
  font-size: 10px; font-weight: 500; padding: 1px 6px; border-radius: var(--radius-sm);
  background: var(--color-info-bg); color: var(--color-info);
}
.rule-source-tag--manual { background: var(--color-bg-hover); color: var(--color-text-tertiary); }
.rule-card-actions { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }

.icon-btn {
  width: 32px; height: 32px; border: 1px solid var(--color-border); background: transparent;
  border-radius: var(--radius-md); cursor: pointer; display: flex; align-items: center;
  justify-content: center; color: var(--color-text-tertiary); transition: all var(--transition-fast);
}
.icon-btn:hover { border-color: var(--color-primary); color: var(--color-primary); }
.icon-btn--danger:hover { border-color: var(--color-danger); color: var(--color-danger); }
.icon-btn--sm { width: 24px; height: 24px; font-size: 12px; }

/*知识库模式*/
.kb-modes { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; margin-bottom: 20px; }
.kb-mode-card {
  display: flex; align-items: center; gap: 12px; padding: 14px;
  background: var(--color-bg-page); border-radius: var(--radius-md);
  border: 2px solid var(--color-border-light); cursor: pointer;
  transition: all var(--transition-fast); position: relative;
}
.kb-mode-card:hover:not(.kb-mode-card--disabled) { border-color: var(--color-primary-lighter); }
.kb-mode-card--active { border-color: var(--color-primary); background: var(--color-primary-bg); }
.kb-mode-card--disabled { opacity: 0.5; cursor: not-allowed; }
.kb-mode-icon {
  width: 36px; height: 36px; border-radius: var(--radius-md); background: var(--color-bg-card);
  display: flex; align-items: center; justify-content: center; font-size: 16px;
  color: var(--color-primary); flex-shrink: 0;
}
.kb-mode-title { font-size: 13px; font-weight: 600; color: var(--color-text-primary); }
.kb-mode-desc { font-size: 11px; color: var(--color-text-tertiary); margin-top: 1px; }
.kb-mode-check {
  position: absolute; top: 8px; right: 8px; width: 20px; height: 20px; border-radius: 50%;
  background: var(--color-primary); color: #fff; font-size: 11px;
  display: flex; align-items: center; justify-content: center;
}
.kb-mode-badge {
  position: absolute; top: 8px; right: 8px; font-size: 10px; font-weight: 600;
  padding: 2px 6px; border-radius: var(--radius-full);
  background: var(--color-bg-hover); color: var(--color-text-tertiary);
}

/*人工智能表格*/
.ai-form { display: flex; flex-direction: column; gap: 20px; }
.ai-form-group { display: flex; flex-direction: column; gap: 6px; }
.ai-form-label { font-size: 13px; font-weight: 600; color: var(--color-text-primary); }

/*严格性*/
.strictness-options { display: flex; gap: 10px; }
.strictness-option {
  display: flex; align-items: center; gap: 10px; padding: 10px 14px; flex: 1;
  border: 2px solid var(--color-border-light); border-radius: var(--radius-md);
  cursor: pointer; transition: all var(--transition-fast);
}
.strictness-option:hover { border-color: var(--color-primary-lighter); }
.strictness-option--active { border-color: var(--color-primary); background: var(--color-primary-bg); }
.strictness-option-radio {
  width: 16px; height: 16px; border-radius: 50%; border: 2px solid var(--color-border);
  flex-shrink: 0; transition: all var(--transition-fast);
}
.strictness-option--active .strictness-option-radio { border-color: var(--color-primary); border-width: 5px; }
.strictness-option-label { font-size: 13px; font-weight: 500; color: var(--color-text-primary); }
.strictness-option-desc { font-size: 11px; color: var(--color-text-tertiary); margin-top: 1px; }

/*权限*/
.permissions-list { display: flex; flex-direction: column; gap: 12px; }
.permission-item {
  display: flex; align-items: center; justify-content: space-between; gap: 16px;
  padding: 16px 20px; background: var(--color-bg-page); border-radius: var(--radius-md);
  border: 1px solid var(--color-border-light);
}
.permission-label { font-size: 14px; font-weight: 500; color: var(--color-text-primary); }
.permission-desc { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }

.config-actions { margin-top: 24px; display: flex; justify-content: flex-end; }

@media (max-width: 768px) {
  .main-layout { grid-template-columns: 1fr; }
  .field-mode-switch { flex-direction: column; }
  .kb-modes { grid-template-columns: 1fr; }
  .strictness-options { flex-direction: column; }
  .tab-nav {
    width: 100%;
    overflow-x: auto;
    -webkit-overflow-scrolling: touch;
    scrollbar-width: none;
  }
  .tab-nav::-webkit-scrollbar { display: none; }
  .tab-btn { flex-shrink: 0; }
  .push-format-options { flex-direction: column; }
  .permission-item { flex-direction: column; align-items: flex-start; gap: 8px; padding: 12px 14px; }
  .config-panel { padding: 16px; }
}
@media (max-width: 480px) {
  .page-title { font-size: 20px; }
  .tab-btn { padding: 6px 10px; font-size: 12px; }
}



.status-dot {
  display: inline-block; width: 6px; height: 6px; border-radius: 50%;
  background: var(--color-text-tertiary); margin-right: 4px;
}
.status-dot--active { background: var(--color-success); }

.push-format-options { display: flex; gap: 10px; }
.push-format-option {
  display: flex; align-items: center; gap: 10px; padding: 10px 16px; flex: 1;
  border: 2px solid var(--color-border-light); border-radius: var(--radius-md);
  cursor: pointer; transition: all var(--transition-fast);
  font-size: 13px; font-weight: 500; color: var(--color-text-primary);
}
.push-format-option:hover { border-color: var(--color-primary-lighter); }
.push-format-option--active { border-color: var(--color-primary); background: var(--color-primary-bg); }
.push-format-radio {
  width: 16px; height: 16px; border-radius: 50%; border: 2px solid var(--color-border);
  flex-shrink: 0; transition: all var(--transition-fast);
}
.push-format-option--active .push-format-radio { border-color: var(--color-primary); border-width: 5px; }

.field-group-label {
  font-size: 13px; font-weight: 600; color: var(--color-text-secondary);
  margin: 16px 0 8px; padding-left: 4px;
  border-left: 3px solid var(--color-primary);
}
.rule-flow-tag {
  display: inline-flex; align-items: center; gap: 4px;
  font-size: 11px; font-weight: 500; padding: 1px 8px;
  border-radius: var(--radius-full);
  background: var(--color-info-bg); color: var(--color-info);
}

.prompt-variables { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; margin-bottom: 8px; }
.prompt-variables-hint { font-size: 12px; color: var(--color-text-tertiary); }
.variable-btn {
  font-size: 11px; font-family: monospace; padding: 2px 8px;
  border: 1px solid var(--color-border); border-radius: var(--radius-sm);
  background: var(--color-bg-hover); color: var(--color-primary);
  cursor: pointer; transition: all var(--transition-fast);
}
.variable-btn:hover { background: var(--color-primary-bg); border-color: var(--color-primary); }

/*提示部分样式*/
.prompt-section-header { margin-bottom: 8px; }
.prompt-section-title { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; }
.prompt-section-desc { font-size: 12px; color: var(--color-text-tertiary); line-height: 1.5; }
.prompt-phase-badge {
  display: inline-flex; align-items: center; font-size: 11px; font-weight: 600;
  padding: 2px 10px; border-radius: var(--radius-full); white-space: nowrap;
}
.prompt-phase-badge--reasoning { background: var(--color-primary-bg); color: var(--color-primary); }
.prompt-phase-badge--extraction { background: var(--color-info-bg); color: var(--color-info); }
.strictness-hint {
  margin-top: 8px; font-size: 12px; color: var(--color-text-tertiary);
  padding: 8px 12px; background: var(--color-bg-hover); border-radius: var(--radius-sm);
  line-height: 1.5;
}

/*提示词区域分组*/
.ai-prompt-section {
  margin-top: 20px; padding: 16px; background: var(--color-bg-page);
  border-radius: var(--radius-md); border: 1px solid var(--color-border-light);
}
.ai-prompt-section-header {
  display: flex; align-items: center; justify-content: space-between; margin-bottom: 4px;
}
.ai-prompt-section-tag {
  display: inline-flex; align-items: center; font-size: 13px; font-weight: 600;
  padding: 2px 12px; border-radius: var(--radius-full);
}
.ai-prompt-section-tag--system { background: var(--color-warning-bg, #fffbe6); color: var(--color-warning, #d48806); }
.ai-prompt-section-tag--user { background: var(--color-primary-bg); color: var(--color-primary); }
.ai-prompt-section-desc {
  font-size: 12px; color: var(--color-text-tertiary); margin: 0 0 12px; line-height: 1.5;
}

/*字段选择器工具栏*/
.field-picker-toolbar {
  display: flex; justify-content: space-between; align-items: center; margin-bottom: 14px;
}

/*信息表*/
.info-form { max-width: 480px; }
.info-form :deep(.ant-form-item) { margin-bottom: 16px; }

/*显示选定的字段*/
.selected-fields-display {
  display: flex; flex-wrap: wrap; gap: 8px;
}
.selected-field-tag {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 6px 12px; border-radius: var(--radius-md);
  background: var(--color-primary-bg); border: 1px solid var(--color-primary-lighter);
  font-size: 13px; color: var(--color-text-primary);
}
.selected-field-name { font-weight: 500; }
.selected-field-group { margin-bottom: 12px; }
.field-empty-hint {
  padding: 24px; text-align: center; color: var(--color-text-tertiary);
  font-size: 13px; background: var(--color-bg-hover); border-radius: var(--radius-md);
}

/*字段选择器模态*/
.field-picker-modal {
  display: grid; grid-template-columns: 1fr 1fr; gap: 16px;
  min-height: 400px; margin-top: 12px;
}
.field-picker-left, .field-picker-right {
  border: 1px solid var(--color-border-light); border-radius: var(--radius-md);
  display: flex; flex-direction: column; overflow: hidden;
}
.field-picker-panel-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 14px; background: var(--color-bg-hover);
  font-size: 13px; font-weight: 600; color: var(--color-text-primary);
  border-bottom: 1px solid var(--color-border-light);
}
.field-picker-count {
  font-size: 11px; font-weight: 500; padding: 1px 8px;
  border-radius: var(--radius-full); background: var(--color-primary-bg); color: var(--color-primary);
}
.field-picker-search { padding: 8px 10px; border-bottom: 1px solid var(--color-border-light); }
.field-picker-list { flex: 1; overflow-y: auto; padding: 4px; }
.field-picker-item {
  display: flex; align-items: center; justify-content: space-between;
  padding: 8px 10px; border-radius: var(--radius-sm); cursor: pointer;
  transition: all var(--transition-fast); gap: 8px;
}
.field-picker-item:hover { background: var(--color-bg-hover); }
.field-picker-item--selected { cursor: default; }
.field-picker-item--selected:hover { background: transparent; }
.field-picker-item-name { font-size: 13px; font-weight: 500; color: var(--color-text-primary); }
.field-picker-item-meta { display: flex; align-items: center; gap: 6px; margin-top: 2px; }
.field-picker-group-label {
  font-size: 12px; font-weight: 600; color: var(--color-text-secondary);
  padding: 6px 10px 2px; margin-top: 4px;
  border-left: 3px solid var(--color-primary);
}
.field-picker-arrow { color: var(--color-primary); font-size: 14px; flex-shrink: 0; }
.field-picker-remove {
  width: 22px; height: 22px; border: none; background: transparent;
  border-radius: var(--radius-sm); cursor: pointer; display: flex;
  align-items: center; justify-content: center;
  color: var(--color-text-tertiary); font-size: 11px;
  transition: all var(--transition-fast); flex-shrink: 0;
}
.field-picker-remove:hover { background: var(--color-danger-bg); color: var(--color-danger); }
.field-picker-empty {
  padding: 32px 16px; text-align: center; color: var(--color-text-tertiary); font-size: 13px;
}

/*访问控制*/
.access-control-section { display: flex; flex-direction: column; gap: 0; }
.access-control-group { }
.access-control-label {
  font-size: 13px; font-weight: 600; color: var(--color-text-secondary);
  display: flex; align-items: center; gap: 6px; margin-bottom: 10px;
}
.access-control-search { margin-bottom: 8px; }
.access-control-tags { display: flex; flex-wrap: wrap; gap: 8px; }
.access-tag {
  display: inline-flex; align-items: center; gap: 5px;
  padding: 5px 12px; border-radius: var(--radius-full);
  border: 1px solid var(--color-border-light); background: var(--color-bg-hover);
  font-size: 12px; font-weight: 500; color: var(--color-text-secondary);
  cursor: pointer; transition: all var(--transition-fast);
}
.access-tag:hover { border-color: var(--color-primary-lighter); color: var(--color-primary); }
.access-tag--active { border-color: var(--color-primary); background: var(--color-primary-bg); color: var(--color-primary); }
.access-tag-check { font-size: 10px; }
.access-tag-dept {
  font-size: 10px; color: var(--color-text-tertiary); margin-left: 2px;
  padding-left: 6px; border-left: 1px solid var(--color-border-light);
}

/* Imported data-table styles */
.data-table-card {
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); overflow: hidden;
}
.data-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.data-table th {
  padding: 12px 16px; text-align: left; font-weight: 600; color: var(--color-text-secondary);
  background: var(--color-bg-page); border-bottom: 1px solid var(--color-border-light);
  font-size: 12px; text-transform: uppercase; letter-spacing: 0.04em; white-space: nowrap;
}
.data-table td {
  padding: 12px 16px; border-bottom: 1px solid var(--color-border-light);
  color: var(--color-text-primary);
}
.data-table tbody tr:hover { background: var(--color-bg-hover); }
.data-table tbody tr:last-child td { border-bottom: none; }
.text-secondary { color: var(--color-text-tertiary); }
.text-mono { font-family: monospace; font-size: 12px; color: var(--color-text-secondary); }
.empty-cell { text-align: center; padding: 32px 16px !important; color: var(--color-text-tertiary); }

.icon-btn {
  width: 28px; height: 28px; border: 1px solid var(--color-border); background: transparent;
  border-radius: var(--radius-sm); cursor: pointer; display: flex; align-items: center;
  justify-content: center; color: var(--color-text-tertiary); transition: all var(--transition-fast);
}
.icon-btn:hover { border-color: var(--color-primary); color: var(--color-primary); }

.pagination-wrapper { padding: 16px 0; display: flex; justify-content: flex-end; }

/*过渡*/
.fade-in { animation: fadeIn 0.3s ease-out; }

@media (max-width: 768px) {
  .data-table-card { overflow-x: auto; -webkit-overflow-scrolling: touch; }
  .data-table { min-width: 700px; }
  .page-title { font-size: 20px; }
  .tab-nav { width: 100%; overflow-x: auto; -webkit-overflow-scrolling: touch; }
  .tab-btn { flex-shrink: 0; padding: 8px 14px; font-size: 13px; }
}
</style>
