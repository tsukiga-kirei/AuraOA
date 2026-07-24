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
  SnippetsOutlined,
  SwapRightOutlined,
  DownOutlined,
  TeamOutlined,
  ThunderboltOutlined,
  UnlockOutlined,
  UploadOutlined,
  UserOutlined,
} from '@ant-design/icons-vue'
import { message, Modal } from 'ant-design-vue'
import type {
  AuditRule as ApiAuditRule,
  ProcessAuditConfig as ApiProcessAuditConfig,
  SystemPromptTemplate
} from '~/composables/useAuditConfigApi'
import type { ArchiveRule, ProcessArchiveConfig } from '~/types/archive-config'
import type { ProcessInfo } from '~/types/common'
import type { CronTaskConfig } from '~/types/cron'
import type { ProcessSummaryConfig, SummaryBlockConfig } from '~/types/process-summary'
import type { RuleImportCapability, RuleImportDraft, RuleImportSource, SelectableRuleImportDraft } from '~/types/rule-import'
import {useI18n} from '~/composables/useI18n'
import {usePagination} from '~/composables/usePagination'
import {useArchiveConfigApi} from "~/composables/useArchiveConfigApi";
import {useCronApi} from "~/composables/useCronApi";

definePageMeta({ middleware: 'auth', layout: 'default' })

const { t } = useI18n()
const { systemPromptVariables } = usePromptSystemVariables()
const rulesApi = useAuditConfigApi()
const cronApi = useCronApi()
const archiveApi = useArchiveConfigApi()
const summaryApi = useSummaryConfigApi()
const tableNameSQLVariable = '{{table_name}}'
const joinFieldSQLVariable = '{{join_field}}'
const customSQLPlaceholder = `仅允许 SELECT，必须使用 ${tableNameSQLVariable}、${joinFieldSQLVariable} 与 :source_value`

//===== 顶级选项卡：审核工作台 vs 定时任务配置 vs 归档复盘 vs 流程总结 =====
const topTab = ref<'audit' | 'cron' | 'archive' | 'summary'>('audit')

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
const selectedRuleIds = ref<string[]>([])
const batchDeletingRules = ref(false)
const selectedRuleIdSet = computed(() => new Set(selectedRuleIds.value))
const allRulesSelected = computed(() => currentRules.value.length > 0 && selectedRuleIds.value.length === currentRules.value.length)
const rulesSelectionIndeterminate = computed(() => selectedRuleIds.value.length > 0 && !allRulesSelected.value)

const toggleRuleSelection = (id: string, checked: boolean) => {
  selectedRuleIds.value = checked
    ? [...new Set([...selectedRuleIds.value, id])]
    : selectedRuleIds.value.filter(ruleId => ruleId !== id)
}

const toggleAllRules = (checked: boolean) => {
  selectedRuleIds.value = checked ? currentRules.value.map(rule => rule.id) : []
}

//=====测试连接状态=====
const testingConnection = ref(false)
// 基本信息页面的测试连接状态（独立于新增弹框）
const infoTestingConnection = ref(false)
// 同步字段状态
const syncingFields = ref(false)
const syncingArchiveFields = ref(false)

const showTestConnectionFeedback = (success: boolean, content: string) => {
  if (success) message.success(content)
  else message.error(content)
}

//=====添加新流程=====
const showAddProcess = ref(false)
const newProcessForm = ref({ process_type: '', process_type_label: '', main_table_name: '' })
const addProcessSearch = ref({
  keyword: '',
  rows: [] as ProcessInfo[],
  searching: false,
  hasSearched: false,
  selectedId: '',
})

const resetAddProcessSearch = (state: typeof addProcessSearch.value) => {
  state.keyword = ''
  state.rows = []
  state.searching = false
  state.hasSearched = false
  state.selectedId = ''
}

const workflowOptionKey = (item: ProcessInfo) =>
  item.workflow_id || item.process_name || item.process_type

const workflowOptionName = (item: ProcessInfo) =>
  item.process_name || item.process_type || (item.workflow_id && item.workflow_id !== '0' ? `流程 ${item.workflow_id}` : '未命名流程')

const applyWorkflowToAddForm = (
  form: { process_type: string; process_type_label: string; main_table_name: string },
  item: ProcessInfo,
) => {
  form.process_type = item.process_name || item.process_type || ''
  form.process_type_label = item.process_type_label || ''
  form.main_table_name = item.main_table || ''
}

const searchAddProcessWorkflows = async (
  searchApi: (keyword: string) => Promise<ProcessInfo[]>,
  state: typeof addProcessSearch.value,
) => {
  const keyword = state.keyword.trim()
  if (!keyword) {
    message.warning('请输入流程名称、分类或主表名后再搜索')
    state.rows = []
    state.hasSearched = false
    return
  }
  state.searching = true
  try {
    state.rows = await searchApi(keyword)
    state.hasSearched = true
  } catch (e: any) {
    message.error('流程检索失败：' + (e.message || '未知错误'))
  } finally {
    state.searching = false
  }
}

const selectAddProcessWorkflow = (
  form: { process_type: string; process_type_label: string; main_table_name: string },
  state: typeof addProcessSearch.value,
  item: ProcessInfo,
  onSelected?: () => void,
) => {
  state.selectedId = workflowOptionKey(item)
  applyWorkflowToAddForm(form, item)
  onSelected?.()
}

watch(showAddProcess, (open) => {
  if (open) {
    newProcessForm.value = { process_type: '', process_type_label: '', main_table_name: '' }
    resetAddProcessSearch(addProcessSearch.value)
  }
})

// 新增弹框中的测试连接
const handleTestConnectionInModal = async () => {
  const processType = newProcessForm.value.process_type.trim()
  if (!processType) {
    message.warning(t('admin.ruleConfig.enterProcessName'))
    return
  }
  testingConnection.value = true
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
      showTestConnectionFeedback(false, msgs.join('；'))
    } else {
      if (info.main_table) {
        newProcessForm.value.main_table_name = info.main_table
      }
      if (info.process_type_label) {
        newProcessForm.value.process_type_label = info.process_type_label
      }
      showTestConnectionFeedback(
        true,
        t('admin.ruleConfig.testConnectionSuccess', [info.process_name || processType, info.main_table || '-', info.process_type_label || '-']),
      )
    }
  } catch (e: any) {
    showTestConnectionFeedback(false, t('admin.ruleConfig.testConnectionFail', [e.message || '未知错误']))
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
      showTestConnectionFeedback(false, msgs.join('；'))
    } else {
      if (info.main_table && selectedConfig.value) {
        selectedConfig.value.main_table_name = info.main_table
      }
      if (info.process_type_label && selectedConfig.value) {
        selectedConfig.value.process_type_label = info.process_type_label
      }
      showTestConnectionFeedback(
        true,
        t('admin.ruleConfig.testConnectionSuccess', [info.process_name || processType, info.main_table || '-', info.process_type_label || '-']),
      )
    }
  } catch (e: any) {
    showTestConnectionFeedback(false, t('admin.ruleConfig.testConnectionFail', [e.message || '未知错误']))
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

// 归档复盘：同步 OA 字段
const handleArchiveSyncFields = async () => {
  if (!selectedArchiveConfig.value) return
  syncingArchiveFields.value = true
  try {
    const fields = await archiveApi.fetchFields(selectedArchiveConfig.value.id)
    // 更新本地数据
    selectedArchiveConfig.value.main_fields = (fields.main_fields || []).map((f: any) => ({ ...f, selected: true }))
    selectedArchiveConfig.value.detail_tables = (fields.detail_tables || []).map((dt: any) => ({
      ...dt,
      fields: (dt.fields || []).map((f: any) => ({ ...f, selected: true })),
    }))
    message.success(t('admin.ruleConfig.fetchFieldsSuccess'))
  } catch (e: any) {
    message.error(t('admin.ruleConfig.fetchFieldsFail') + ': ' + (e.message || ''))
  } finally {
    syncingArchiveFields.value = false
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
      access_control: { allowed_roles: [], allowed_members: [], allowed_departments: [] },
    })
    processConfigs.value.push(created)
    selectedProcessId.value = created.id
    showAddProcess.value = false
    newProcessForm.value = { process_type: '', process_type_label: '', main_table_name: '' }
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

// 快照原始权限，用于保存时检测是否有权限降级
const originalAuditPerms = ref<Record<string, boolean>>({})
watch(selectedProcessId, (newId) => {
  const cfg = processConfigs.value.find(c => c.id === newId)
  originalAuditPerms.value = cfg ? { ...(cfg.user_permissions as any) } : {}
}, { immediate: true })

// 当选中流程变化时，从 API 加载该流程的规则
watch(selectedProcessId, async (newId) => {
  selectedRuleIds.value = []
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
  // 切换流程时无需额外清理测试连接状态
})

//===== 字段配置 =====
const fieldTypeLabels = computed<Record<string, string>>(() => ({
  text: t('fieldType.text'), number: t('fieldType.number'), date: t('fieldType.date'), select: t('fieldType.select'), textarea: t('fieldType.textarea'), file: t('fieldType.file'),
}))



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

const auditContextFieldOptions = computed(() => {
  const includeAll = selectedConfig.value?.field_mode === 'all'
  return allAvailableFields.value
    .filter(f => includeAll || f.selected)
    .map(f => ({
      label: `${f.field_name}（${f.sourceLabel}）`,
      value: `${f.source}:${f.field_key}`,
    }))
})

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

const hasActualExternalContext = (rule: { context_enabled?: boolean; context_mounts?: { enabled?: boolean }[] }) =>
  !!rule.context_enabled && !!rule.context_mounts?.some(mount => mount.enabled !== false)

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
        context_enabled: rule.context_enabled,
        context_mounts: rule.context_mounts,
        // 强制根据级别同步启用状态
        enabled: rule.rule_scope === 'mandatory' ? true : (rule.rule_scope === 'default_off' ? false : true)
      })
      const idx = currentRules.value.findIndex(r => r.id === editingRule.value!.id)
      if (idx >= 0) currentRules.value[idx] = updated
    } else {
      // 创建规则
      // 根据规则级别设置初始状态：强制或默认开启则设为true，默认关闭则设为false
      const initialEnabled = rule.rule_scope !== 'default_off'
      const created = await rulesApi.createRule({
        config_id: selectedConfig.value.id,
        process_type: selectedConfig.value.process_type,
        rule_content: rule.rule_content,
        rule_scope: rule.rule_scope,
        related_flow: rule.related_flow,
        context_enabled: rule.context_enabled,
        context_mounts: rule.context_mounts,
        enabled: initialEnabled,
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
    selectedRuleIds.value = selectedRuleIds.value.filter(ruleId => ruleId !== id)
    message.success(t('admin.ruleConfig.deleted'))
  } catch (e: any) {
    message.error(t('admin.ruleConfig.deleteRuleFail') + ': ' + (e.message || ''))
  }
}

const batchDeleteRules = async () => {
  if (!selectedConfig.value || selectedRuleIds.value.length === 0) return
  const ids = [...selectedRuleIds.value]
  batchDeletingRules.value = true
  try {
    const deletedCount = await rulesApi.batchDeleteRules(selectedConfig.value.id, ids)
    const deletedIDs = new Set(ids)
    currentRules.value = currentRules.value.filter(rule => !deletedIDs.has(rule.id))
    selectedRuleIds.value = []
    message.success(t('admin.ruleConfig.batchDeleteSuccess', `${deletedCount}`))
  } catch (e: any) {
    message.error(t('admin.ruleConfig.batchDeleteFail') + ': ' + (e.message || ''))
  } finally {
    batchDeletingRules.value = false
  }
}

type RuleImportModule = 'audit' | 'archive'
const ruleImportFileInput = ref<HTMLInputElement | null>(null)
const ruleImportTarget = ref<RuleImportModule>('audit')
const ruleImportSource = ref<RuleImportSource>('file_import')
const ruleImportCapability = ref<RuleImportCapability | null>(null)
const ruleImportLoading = ref(false)
const ruleImportSaving = ref(false)
const showRuleImportPreview = ref(false)
const ruleImportFileName = ref('')
const ruleImportWarnings = ref<string[]>([])
const ruleImportDrafts = ref<SelectableRuleImportDraft[]>([])
const showPasteRuleImport = ref(false)
const pastedRuleImportText = ref('')
const ruleImportAccept = computed(() =>
  (ruleImportCapability.value?.supported_types || []).map(type => `.${type}`).join(','),
)
const selectedRuleImportCount = computed(() => ruleImportDrafts.value.filter(rule => rule.selected).length)

const handleImportRules = (module: RuleImportModule) => {
  const config = module === 'audit' ? selectedConfig.value : selectedArchiveConfig.value
  if (!config) return
  ruleImportTarget.value = module
  ruleImportSource.value = 'file_import'
  if (!ruleImportCapability.value?.enabled) {
    message.warning(ruleImportCapability.value?.reason || t('admin.ruleConfig.fileImportUnavailable'))
    return
  }
  ruleImportFileInput.value?.click()
}

const handlePasteImport = (module: RuleImportModule) => {
  const config = module === 'audit' ? selectedConfig.value : selectedArchiveConfig.value
  if (!config) return
  ruleImportTarget.value = module
  ruleImportSource.value = 'paste_import'
  pastedRuleImportText.value = ''
  showPasteRuleImport.value = true
}

const analyzePastedRuleImport = async () => {
  const text = pastedRuleImportText.value.trim()
  const config = ruleImportTarget.value === 'audit' ? selectedConfig.value : selectedArchiveConfig.value
  if (!config || !text) {
    message.warning(t('admin.ruleConfig.pasteImportRequired'))
    return
  }
  ruleImportLoading.value = true
  message.loading({ content: t('admin.ruleConfig.pasteImportAnalyzing'), key: 'rule-import', duration: 0 })
  try {
    const api = ruleImportTarget.value === 'audit' ? rulesApi : archiveApi
    const preview = await api.previewPastedRuleImport(config.id, text)
    ruleImportFileName.value = t('admin.ruleConfig.pasteImportContentName')
    ruleImportWarnings.value = preview.warnings || []
    ruleImportDrafts.value = preview.rules.map(rule => ({ ...rule, selected: true }))
    showPasteRuleImport.value = false
    showRuleImportPreview.value = true
    message.success({ content: t('admin.ruleConfig.fileImportRecognized', `${preview.rules.length}`), key: 'rule-import' })
  } catch (e: any) {
    message.error({ content: t('admin.ruleConfig.pasteImportFailed') + ': ' + (e.message || ''), key: 'rule-import' })
  } finally {
    ruleImportLoading.value = false
  }
}

const handleRuleImportFileSelected = async (event: Event) => {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  const capability = ruleImportCapability.value
  const extension = file.name.includes('.') ? file.name.split('.').pop()!.toLowerCase() : ''
  if (capability && !capability.supported_types.includes(extension)) {
    message.error(t('admin.ruleConfig.fileImportTypeUnsupported', extension || file.type))
    input.value = ''
    return
  }
  if (capability?.max_file_size_mb && file.size > capability.max_file_size_mb * 1024 * 1024) {
    message.error(t('admin.ruleConfig.fileImportTooLarge', `${capability.max_file_size_mb}`))
    input.value = ''
    return
  }
  const config = ruleImportTarget.value === 'audit' ? selectedConfig.value : selectedArchiveConfig.value
  if (!config) return

  ruleImportLoading.value = true
  message.loading({ content: t('admin.ruleConfig.fileImportRecognizing'), key: 'rule-import', duration: 0 })
  try {
    const api = ruleImportTarget.value === 'audit' ? rulesApi : archiveApi
    const preview = await api.previewRuleImport(config.id, file)
    ruleImportFileName.value = preview.file_name
    ruleImportWarnings.value = preview.warnings || []
    ruleImportDrafts.value = preview.rules.map(rule => ({ ...rule, selected: true }))
    showRuleImportPreview.value = true
    message.success({ content: t('admin.ruleConfig.fileImportRecognized', `${preview.rules.length}`), key: 'rule-import' })
  } catch (e: any) {
    message.error({ content: t('admin.ruleConfig.fileImportFailed') + ': ' + (e.message || ''), key: 'rule-import' })
  } finally {
    ruleImportLoading.value = false
    input.value = ''
  }
}

const confirmRuleImport = async () => {
  const config = ruleImportTarget.value === 'audit' ? selectedConfig.value : selectedArchiveConfig.value
  const selected = ruleImportDrafts.value.filter(rule => rule.selected)
  if (!config || selected.length === 0) {
    message.warning(t('admin.ruleConfig.fileImportSelectOne'))
    return
  }
  ruleImportSaving.value = true
  try {
    const drafts: RuleImportDraft[] = selected.map(({ selected: _selected, ...rule }) => rule)
    let importedCount = 0
    if (ruleImportTarget.value === 'audit') {
      const created = await rulesApi.confirmRuleImport(config.id, drafts, ruleImportSource.value)
      currentRules.value.push(...created)
      importedCount = created.length
    } else {
      const created = await archiveApi.confirmRuleImport(config.id, drafts, ruleImportSource.value)
      currentArchiveRules.value.push(...created)
      importedCount = created.length
    }
    showRuleImportPreview.value = false
    message.success(t('admin.ruleConfig.fileImportSaved', `${importedCount}`))
  } catch (e: any) {
    message.error(t('admin.ruleConfig.fileImportSaveFail') + ': ' + (e.message || ''))
  } finally {
    ruleImportSaving.value = false
  }
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

// 提示词编辑器会记住最近一次有效选区，变量按钮点击后仍可在原光标位置插入。
const reasoningTextareaRef = ref<any>(null)
const extractionTextareaRef = ref<any>(null)

const insertReasoningVariable = (variable: string) => {
  reasoningTextareaRef.value?.insertAtCursor(variable)
}

const insertExtractionVariable = (variable: string) => {
  extractionTextareaRef.value?.insertAtCursor(variable)
}

const SUMMARY_DATA_VARIABLE_KEYS = [
  '{{process_meta}}',
  '{{main_table}}',
  '{{detail_tables}}',
  '{{attachments}}',
  '{{flow_history}}',
  '{{flow_graph}}',
] as const

const summaryPromptDataVariables = computed(() => [
  { key: '{{process_meta}}', desc: t('admin.ruleConfig.varProcessMetaDesc') },
  { key: '{{main_table}}', desc: t('admin.ruleConfig.varMainTableDesc') },
  { key: '{{detail_tables}}', desc: t('admin.ruleConfig.varDetailTablesDesc') },
  { key: '{{attachments}}', desc: t('admin.ruleConfig.varAttachmentsDesc') },
  { key: '{{flow_history}}', desc: t('admin.ruleConfig.varFlowHistoryDesc') },
  { key: '{{flow_graph}}', desc: t('admin.ruleConfig.varFlowGraphDesc') },
])

const toggleSummaryBlockDataVariable = (block: SummaryBlockConfig, variable: string) => {
  if (!block.enabled_data_variables) {
    block.enabled_data_variables = []
  }
  const idx = block.enabled_data_variables.indexOf(variable)
  if (idx >= 0) {
    block.enabled_data_variables.splice(idx, 1)
  } else {
    block.enabled_data_variables.push(variable)
  }
}

const summaryBlockTextareaRefs = ref<Record<string, any>>({})

const setSummaryBlockTextareaRef = (blockId: string, el: any) => {
  if (el) {
    summaryBlockTextareaRefs.value[blockId] = el
  }
}

const insertSummaryBlockVariable = (block: SummaryBlockConfig, variable: string) => {
  summaryBlockTextareaRefs.value[block.id]?.insertAtCursor(variable)
}

//=====系统提示词模板=====

const promptTemplates = ref<SystemPromptTemplate[]>([])
const archivePromptTemplates = ref<SystemPromptTemplate[]>([])
const loadingTemplates = ref(false)
// 页面初始化：依次加载组织数据、审核配置、提示词模板、定时任务配置、归档配置
onMounted(async () => {
  loadOrgData()
  // 文件选择器需要在用户点击的同步事件中打开，因此页面初始化时预先读取系统能力。
  rulesApi.getRuleImportCapability()
    .then(capability => { ruleImportCapability.value = capability })
    .catch((e) => {
      console.error('[rules] 加载文件识别导入能力失败', e)
      ruleImportCapability.value = {
        enabled: false,
        max_file_size_mb: 0,
        supported_types: [],
        reason: t('admin.ruleConfig.fileImportCapabilityFail'),
      }
    })
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
    cronConfigs.value = cronList.map(c => ({
      ...c,
      batch_limit: c.batch_limit ?? 10
    }))
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
  // 加载流程总结配置
  loadingSummary.value = true
  try {
    const summaryList = await summaryApi.listConfigs()
    summaryConfigs.value = summaryList.map(normalizeSummaryConfigForUI)
    if (summaryConfigs.value.length > 0) selectedSummaryId.value = summaryConfigs.value[0].id
  } catch (e) { console.error('[rules] 加载流程总结配置失败', e) }
  finally { loadingSummary.value = false }
})

const getTemplateContent = (promptKey: string) => {
  return promptTemplates.value.find(t => t.prompt_key === promptKey)?.content || ''
}

const handleStrictnessChange = (value: string) => {
  if (!selectedConfig.value) return
  selectedConfig.value.ai_config.audit_strictness = value as any
  selectedConfig.value.ai_config.system_reasoning_prompt = getTemplateContent(`audit_system_reasoning_${value}`)
  selectedConfig.value.ai_config.system_extraction_prompt = getTemplateContent(`audit_system_extraction_${value}`)
  selectedConfig.value.ai_config.user_reasoning_prompt = getTemplateContent(`audit_user_reasoning_${value}`)
  selectedConfig.value.ai_config.user_extraction_prompt = getTemplateContent(`audit_user_extraction_${value}`)
}

const resetSystemPrompts = () => {
  if (!selectedConfig.value) return
  const strictness = selectedConfig.value.ai_config.audit_strictness || 'standard'
  selectedConfig.value.ai_config.system_reasoning_prompt = getTemplateContent(`audit_system_reasoning_${strictness}`)
  selectedConfig.value.ai_config.system_extraction_prompt = getTemplateContent(`audit_system_extraction_${strictness}`)
  message.success(t('admin.ruleConfig.systemPromptsReset'))
}

const resetUserPrompts = () => {
  if (!selectedConfig.value) return
  const strictness = selectedConfig.value.ai_config.audit_strictness || 'standard'
  selectedConfig.value.ai_config.user_reasoning_prompt = getTemplateContent(`audit_user_reasoning_${strictness}`)
  selectedConfig.value.ai_config.user_extraction_prompt = getTemplateContent(`audit_user_extraction_${strictness}`)
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

//===== 流程总结配置 =====
const summaryConfigs = ref<ProcessSummaryConfig[]>([])
const loadingSummary = ref(false)
const selectedSummaryId = ref('')
const summaryActiveTab = ref('info')
const showAddSummaryProcess = ref(false)
const newSummaryProcessForm = ref({ process_type: '', process_type_label: '', main_table_name: '' })
const addSummaryProcessSearch = ref({
  keyword: '',
  rows: [] as ProcessInfo[],
  searching: false,
  hasSearched: false,
  selectedId: '',
})
const summaryTestingConnection = ref(false)

watch(showAddSummaryProcess, (open) => {
  if (open) {
    newSummaryProcessForm.value = { process_type: '', process_type_label: '', main_table_name: '' }
    resetAddProcessSearch(addSummaryProcessSearch.value)
  }
})
const summaryInfoTestingConnection = ref(false)
const syncingSummaryFields = ref(false)
const savingSummary = ref(false)
const showSummaryFieldPicker = ref(false)
const editingSummaryBlockId = ref<string>('')
const summaryFieldSearchQuery = ref('')
const summarySelectedFieldSearchQuery = ref('')
const summaryLeftSelectedKeys = ref<string[]>([])
const summaryRightSelectedKeys = ref<string[]>([])

const fixedSummarySystemPrompt = `你是企业 OA 审批流程的总结助手。你的任务是基于给定流程字段、明细、附件识别内容和审批流信息，生成给审批人快速阅读的中文总结。

必须遵守：
1. 只根据输入内容总结，不要编造不存在的事实。
2. 涉及金额、日期、人员、供应商、项目、附件结论等关键信息时尽量保留原值。
3. 若字段为空或附件解析失败，需要明确写“未提供”或“附件解析失败”，不要猜测。
4. 输出必须是 JSON 对象，不要输出 Markdown 代码块、不要添加 JSON 外的解释。
5. JSON 格式固定为：{"content":"一段可直接展示的总结","points":["要点1","要点2"]}。`

const selectedSummaryConfig = computed(() =>
  summaryConfigs.value.find(c => c.id === selectedSummaryId.value)
)

interface SummaryFieldOption {
  label: string
  value: string
  field_type: string
  sourceLabel: string
}

interface SummaryPickerField {
  field_key: string
  field_name: string
  field_type: string
  source: string
  sourceLabel: string
  value: string
}

const summaryAllAvailableFields = computed<SummaryPickerField[]>(() => {
  if (!selectedSummaryConfig.value) return []
  const fields: SummaryPickerField[] = []
  for (const f of selectedSummaryConfig.value.main_fields || []) {
    fields.push({
      ...f,
      source: 'main',
      sourceLabel: t('admin.ruleConfig.mainTableFields'),
      value: `main:${f.field_key}`,
    })
  }
  ;(selectedSummaryConfig.value.detail_tables || []).forEach((dt, idx) => {
    const sourceLabel = dt.table_label || `${t('admin.ruleConfig.detailTableLabel')} ${idx + 1}`
    for (const f of dt.fields || []) {
      fields.push({
        ...f,
        source: dt.table_name,
        sourceLabel,
        value: `${dt.table_name}:${f.field_key}`,
      })
    }
  })
  return fields
})

const summaryFieldOptions = computed<SummaryFieldOption[]>(() => {
  return summaryAllAvailableFields.value.map(f => ({
    label: `${f.field_name}（${f.sourceLabel}）`,
    value: f.value,
    field_type: f.field_type,
    sourceLabel: f.sourceLabel,
  }))
})

const getSummaryFieldLabel = (refKey: string) =>
  summaryFieldOptions.value.find(f => f.value === refKey)?.label || refKey

const summaryPageFieldSearchQuery = ref('')
const summaryPageFieldsFlat = computed(() => {
  const q = summaryPageFieldSearchQuery.value.toLowerCase().trim()
  return summaryAllAvailableFields.value.filter(f => {
    if (!q) return true
    return f.field_name.toLowerCase().includes(q) || f.field_key.toLowerCase().includes(q) || f.sourceLabel.toLowerCase().includes(q)
  })
})
const summaryPageFieldsPagination = usePagination(summaryPageFieldsFlat, 8)

const editingSummaryBlock = computed(() =>
  selectedSummaryConfig.value?.summary_blocks.find(b => b.id === editingSummaryBlockId.value)
)

const editingSummarySelectedSet = computed(() =>
  new Set(editingSummaryBlock.value?.selected_fields || [])
)

const summaryUnselectedFieldsFlat = computed(() => {
  const q = summaryFieldSearchQuery.value.toLowerCase().trim()
  return summaryAllAvailableFields.value.filter(f => {
    if (editingSummarySelectedSet.value.has(f.value)) return false
    if (!q) return true
    return f.field_name.toLowerCase().includes(q) || f.field_key.toLowerCase().includes(q) || f.sourceLabel.toLowerCase().includes(q)
  })
})
const summaryUnselectedPagination = usePagination(summaryUnselectedFieldsFlat, 5)

const summarySelectedFieldsFlat = computed(() => {
  const q = summarySelectedFieldSearchQuery.value.toLowerCase().trim()
  return summaryAllAvailableFields.value.filter(f => {
    if (!editingSummarySelectedSet.value.has(f.value)) return false
    if (!q) return true
    return f.field_name.toLowerCase().includes(q) || f.field_key.toLowerCase().includes(q) || f.sourceLabel.toLowerCase().includes(q)
  })
})
const summarySelectedPagination = usePagination(summarySelectedFieldsFlat, 5)

const openSummaryFieldPicker = (block: SummaryBlockConfig) => {
  editingSummaryBlockId.value = block.id
  summaryFieldSearchQuery.value = ''
  summarySelectedFieldSearchQuery.value = ''
  summaryLeftSelectedKeys.value = []
  summaryRightSelectedKeys.value = []
  showSummaryFieldPicker.value = true
}

const pickSummaryField = (field: { value: string }) => {
  if (!editingSummaryBlock.value) return
  if (!editingSummaryBlock.value.selected_fields.includes(field.value)) {
    editingSummaryBlock.value.selected_fields.push(field.value)
  }
}

const unpickSummaryField = (field: { value: string }) => {
  if (!editingSummaryBlock.value) return
  editingSummaryBlock.value.selected_fields = editingSummaryBlock.value.selected_fields.filter(v => v !== field.value)
}

const toggleSummaryLeftSelectAll = () => {
  if (summaryLeftSelectedKeys.value.length === summaryUnselectedFieldsFlat.value.length && summaryUnselectedFieldsFlat.value.length > 0) {
    summaryLeftSelectedKeys.value = []
  } else {
    summaryLeftSelectedKeys.value = summaryUnselectedFieldsFlat.value.map(f => f.value)
  }
}

const toggleSummaryLeftSelect = (fieldId: string) => {
  const idx = summaryLeftSelectedKeys.value.indexOf(fieldId)
  if (idx >= 0) summaryLeftSelectedKeys.value.splice(idx, 1)
  else summaryLeftSelectedKeys.value.push(fieldId)
}

const summaryBatchPick = () => {
  summaryUnselectedFieldsFlat.value.filter(f => summaryLeftSelectedKeys.value.includes(f.value)).forEach(pickSummaryField)
  summaryLeftSelectedKeys.value = []
}

const toggleSummaryRightSelectAll = () => {
  if (summaryRightSelectedKeys.value.length === summarySelectedFieldsFlat.value.length && summarySelectedFieldsFlat.value.length > 0) {
    summaryRightSelectedKeys.value = []
  } else {
    summaryRightSelectedKeys.value = summarySelectedFieldsFlat.value.map(f => f.value)
  }
}

const toggleSummaryRightSelect = (fieldId: string) => {
  const idx = summaryRightSelectedKeys.value.indexOf(fieldId)
  if (idx >= 0) summaryRightSelectedKeys.value.splice(idx, 1)
  else summaryRightSelectedKeys.value.push(fieldId)
}

const summaryBatchUnpick = () => {
  summarySelectedFieldsFlat.value.filter(f => summaryRightSelectedKeys.value.includes(f.value)).forEach(unpickSummaryField)
  summaryRightSelectedKeys.value = []
}

function normalizeSummaryContextMountForUI(mount: any) {
  if (!mount || typeof mount !== 'object') return mount
  if (mount.type === 'workflow') {
    const targetProcessType = mount.workflow?.target_process_type || ''
    return {
      ...mount,
      source_splitter: ',',
      workflow: {
        ...(mount.workflow || {}),
        include_basic: mount.workflow?.include_basic ?? true,
        basic_fields: mount.workflow?.basic_fields || ['archived', 'title', 'applicant', 'department', 'process_type', 'current_node', 'submit_time'],
        data_mode: targetProcessType || mount.workflow?.target_workflow_id ? 'selected_fields' : 'all_fields',
        target_process_type: targetProcessType,
        target_workflow_id: mount.workflow?.target_workflow_id || '',
        target_process_label: mount.workflow?.target_process_label || '',
        target_main_table: mount.workflow?.target_main_table || '',
        selected_fields: mount.workflow?.selected_fields || [],
        fallback_strategy: mount.workflow?.fallback_strategy || 'basic_with_notice',
        max_rows: mount.workflow?.max_rows || 20,
      },
    }
  }
  if (mount.type === 'model') {
    return {
      ...mount,
      model: {
        ...(mount.model || {}),
        join_field: mount.model?.join_field || 'id',
        mode: mount.model?.mode || 'exists',
        return_fields: mount.model?.return_fields || [],
        max_rows: mount.model?.max_rows ?? 5,
      },
    }
  }
  return mount
}

const normalizeSummaryConfigForUI = (cfg: ProcessSummaryConfig): ProcessSummaryConfig => ({
  ...cfg,
  main_fields: cfg.main_fields || [],
  detail_tables: cfg.detail_tables || [],
  summary_blocks: (cfg.summary_blocks?.length ? cfg.summary_blocks : [createSummaryBlock()]).map((b, idx) => {
    const includeMeta = b.include_meta !== false
    let enabledDataVariables = Array.isArray(b.enabled_data_variables) ? [...b.enabled_data_variables] : []
    // 兼容旧配置：仅关闭 include_meta 时，默认仍传入除流程基础信息外的全部数据
    if (!includeMeta && enabledDataVariables.length === 0 && b.enabled_data_variables === undefined) {
      enabledDataVariables = SUMMARY_DATA_VARIABLE_KEYS.filter(key => key !== '{{process_meta}}')
    }
    return {
      ...b,
      id: b.id || createClientId(),
      title: b.title || '流程摘要',
      user_prompt: b.user_prompt || '',
      include_meta: includeMeta,
      enabled_data_variables: enabledDataVariables,
      field_mode: b.field_mode || 'all',
      selected_fields: b.selected_fields || [],
      context_mounts: Array.isArray(b.context_mounts) ? b.context_mounts.map(normalizeSummaryContextMountForUI) : [],
      enabled: b.enabled !== false,
      sort_order: b.sort_order || idx + 1,
    }
  }),
  embed_enabled: cfg.embed_enabled ?? true,
  embed_config: {
    auto_summary_on_open: cfg.embed_config?.auto_summary_on_open ?? true,
    auto_summary_on_stale: cfg.embed_config?.auto_summary_on_stale ?? true,
  },
  status: cfg.status || 'active',
})

function createSummaryBlock(): SummaryBlockConfig {
  return {
    id: createClientId(),
    title: '流程摘要',
    user_prompt: '请概括流程背景、关键申请内容、金额/日期/对象等核心信息，并列出审批人最需要关注的要点。',
    include_meta: true,
    enabled_data_variables: [],
    field_mode: 'all',
    selected_fields: [],
    context_mounts: [],
    enabled: true,
    sort_order: 1,
  }
}

const summaryContextTesting = ref<Record<string, boolean>>({})
const summaryContextPreviews = ref<Record<string, string>>({})
const summaryContextExpanded = ref<Record<string, boolean>>({})
const summaryWorkflowFieldLoading = ref<Record<string, boolean>>({})
const summaryWorkflowFieldOptions = ref<Record<string, { label: string; value: string }[]>>({})
const summaryWorkflowConfigOpen = ref(false)
const summaryWorkflowConfigBlockId = ref('')
const summaryWorkflowKeyword = ref('')
const summaryWorkflowRows = ref<ProcessInfo[]>([])
const summaryWorkflowSelectedID = ref('')
const summaryWorkflowSearching = ref(false)
const summaryWorkflowHasSearched = ref(false)
const summaryWorkflowDraftFields = ref<string[]>([])

const summaryWorkflowBasicOptions = [
  { label: '是否归档', value: 'archived' },
  { label: '流程标题', value: 'title' },
  { label: '发起人', value: 'applicant' },
  { label: '发起部门', value: 'department' },
  { label: '流程类型', value: 'process_type' },
  { label: '当前节点', value: 'current_node' },
  { label: '提交时间', value: 'submit_time' },
]
const summaryWorkflowBasicAllValues = summaryWorkflowBasicOptions.map(o => o.value)

const isSummaryWorkflowBasicAllSelected = (block: SummaryBlockConfig) => {
  const fields = getSummaryContextMount(block, 'workflow')?.workflow?.basic_fields || []
  return summaryWorkflowBasicAllValues.every(v => fields.includes(v))
}

const toggleSummaryWorkflowBasicAll = (block: SummaryBlockConfig, checked: boolean) => {
  const mount = getSummaryContextMount(block, 'workflow')
  if (!mount?.workflow) return
  mount.workflow.basic_fields = checked ? [...summaryWorkflowBasicAllValues] : []
  mount.workflow.include_basic = checked
}

const summaryContextFieldOptionsForBlock = (block: SummaryBlockConfig) => {
  const allowed = block.field_mode === 'selected' ? new Set(block.selected_fields || []) : null
  return summaryFieldOptions.value
    .filter(f => !allowed || allowed.has(f.value))
    .map(f => ({ label: f.label, value: f.value }))
}

const getSummaryContextMount = (block: SummaryBlockConfig, type: 'workflow' | 'model') =>
  (block.context_mounts || []).find(m => m.type === type)

const createSummaryWorkflowMount = () => ({
  type: 'workflow' as const,
  enabled: true,
  name: '关联流程',
  source_field: '',
  source_splitter: ',',
  workflow: {
    include_basic: true,
    basic_fields: ['archived', 'title', 'applicant', 'department', 'process_type', 'current_node', 'submit_time'],
    data_mode: 'all_fields' as const,
    target_process_type: '',
    target_workflow_id: '',
    target_process_label: '',
    target_main_table: '',
    fallback_strategy: 'basic_with_notice' as const,
    max_rows: 20,
    selected_fields: [],
  },
})

const createSummaryModelMount = () => ({
  type: 'model' as const,
  enabled: true,
  name: '关联建模表',
  source_field: '',
  model: {
    table_name: '',
    join_field: 'id',
    mode: 'exists' as const,
    return_fields: [],
    max_rows: 5,
  },
})

const toggleSummaryContextMount = (block: SummaryBlockConfig, type: 'workflow' | 'model', checked: boolean) => {
  if (!block.context_mounts) block.context_mounts = []
  const idx = block.context_mounts.findIndex(m => m.type === type)
  if (checked && idx < 0) block.context_mounts.push(type === 'workflow' ? createSummaryWorkflowMount() : createSummaryModelMount())
  if (!checked && idx >= 0) block.context_mounts.splice(idx, 1)
  if (checked) summaryContextExpanded.value[summaryContextKey(block, type)] = true
}

const isSummaryContextExpanded = (block: SummaryBlockConfig, type: 'workflow' | 'model') =>
  summaryContextExpanded.value[summaryContextKey(block, type)] !== false

const toggleSummaryContextExpanded = (block: SummaryBlockConfig, type: 'workflow' | 'model') => {
  const key = summaryContextKey(block, type)
  summaryContextExpanded.value[key] = !isSummaryContextExpanded(block, type)
}

const summaryContextCollapsedHint = (block: SummaryBlockConfig, type: 'workflow' | 'model') => {
  const mount = getSummaryContextMount(block, type)
  if (!mount) return ''
  if (type === 'workflow') {
    const sourceLabel = summaryFieldOptions.value.find(f => f.value === mount.source_field)?.label
    if (mount.workflow?.data_mode === 'selected_fields') {
      return [sourceLabel || '未选来源字段', summaryWorkflowConfigSummary(block) || '未配置目标流程'].join(' · ')
    }
    return [sourceLabel || '未选来源字段', '全部字段'].join(' · ')
  }
  const table = mount.model?.table_name?.trim()
  return table ? `表 ${table}` : '未填建模表名'
}

const setSummaryModelReturnFields = (block: SummaryBlockConfig, value: string) => {
  const mount = getSummaryContextMount(block, 'model')
  if (mount?.model) mount.model.return_fields = value.split(',').map(v => v.trim()).filter(Boolean)
}

const setSummaryModelAllRows = (block: SummaryBlockConfig, checked: boolean) => {
  const mount = getSummaryContextMount(block, 'model')
  if (mount?.model) mount.model.max_rows = checked ? -1 : 5
}

const summaryContextKey = (block: SummaryBlockConfig, type: 'workflow' | 'model') => `${block.id}:${type}`

const testSummaryContext = async (block: SummaryBlockConfig, type: 'workflow' | 'model') => {
  const mount = getSummaryContextMount(block, type)
  if (!mount) return
  const key = summaryContextKey(block, type)
  summaryContextTesting.value[key] = true
  summaryContextPreviews.value[key] = ''
  try {
    const resp = await summaryApi.testContextConfig([mount])
    summaryContextPreviews.value[key] = resp.context_text || ''
  } catch (e: any) {
    message.error('关联数据测试失败：' + (e.message || '未知错误'))
  } finally {
    summaryContextTesting.value[key] = false
  }
}

const loadSummaryWorkflowFields = async (block: SummaryBlockConfig, target?: ProcessInfo, options?: { silent?: boolean }) => {
  const mount = getSummaryContextMount(block, 'workflow')
  const processType = target?.process_name || target?.process_type || mount?.workflow?.target_process_type?.trim()
  const workflowID = target?.workflow_id || mount?.workflow?.target_workflow_id || ''
  if (!mount?.workflow || (!processType && !workflowID)) {
    message.warning('请先选择目标流程')
    return
  }
  summaryWorkflowFieldLoading.value[block.id] = true
  try {
    const fields = await summaryApi.fetchWorkflowFields(processType || '', workflowID)
    summaryWorkflowFieldOptions.value[block.id] = [
      ...(fields.main_fields || []).map(f => ({ label: `${f.field_name}（主表）`, value: `main:${f.field_key}` })),
      ...(fields.detail_tables || []).flatMap((dt, idx) => (dt.fields || []).map(f => ({
        label: `${f.field_name}（${dt.table_label || `明细表 ${idx + 1}`}）`,
        value: `${dt.table_name}:${f.field_key}`,
      }))),
    ]
    if (!options?.silent) message.success('目标流程字段已加载')
  } catch (e: any) {
    message.error('目标流程字段加载失败：' + (e.message || '未知错误'))
  } finally {
    summaryWorkflowFieldLoading.value[block.id] = false
  }
}

const setSummaryWorkflowTargetMode = (block: SummaryBlockConfig, specified: boolean) => {
  const mount = getSummaryContextMount(block, 'workflow')
  if (!mount?.workflow) return
  if (specified) {
    mount.workflow.data_mode = 'selected_fields'
    openSummaryWorkflowConfigModal(block)
  } else {
    mount.workflow.target_process_type = ''
    mount.workflow.target_workflow_id = ''
    mount.workflow.target_process_label = ''
    mount.workflow.target_main_table = ''
    mount.workflow.selected_fields = []
    mount.workflow.data_mode = 'all_fields'
    summaryWorkflowFieldOptions.value[block.id] = []
  }
}

const summaryTargetWorkflowSummary = (block: SummaryBlockConfig) => {
  const wf = getSummaryContextMount(block, 'workflow')?.workflow
  if (!wf?.target_process_type && !wf?.target_workflow_id) return ''
  return [wf.target_process_type, wf.target_process_label, wf.target_main_table].filter(Boolean).join(' / ')
}

const summaryWorkflowConfigSummary = (block: SummaryBlockConfig) => {
  const wf = getSummaryContextMount(block, 'workflow')?.workflow
  if (!wf || wf.data_mode !== 'selected_fields') return ''
  const target = summaryTargetWorkflowSummary(block)
  const fieldCount = wf.selected_fields?.length || 0
  if (!target) return '尚未配置'
  return `${target} · ${fieldCount} 个字段`
}

const currentSummaryWorkflowBlock = computed(() =>
  selectedSummaryConfig.value?.summary_blocks.find(block => block.id === summaryWorkflowConfigBlockId.value)
)

const openSummaryWorkflowConfigModal = async (block: SummaryBlockConfig) => {
  const wf = getSummaryContextMount(block, 'workflow')?.workflow
  if (!wf) return
  summaryWorkflowConfigBlockId.value = block.id
  summaryWorkflowKeyword.value = ''
  summaryWorkflowRows.value = []
  summaryWorkflowHasSearched.value = false
  summaryWorkflowSelectedID.value = wf.target_workflow_id || wf.target_process_type || ''
  summaryWorkflowDraftFields.value = [...(wf.selected_fields || [])]
  summaryWorkflowConfigOpen.value = true
  // 已配置过的目标流程：仅静默加载字段选项，不自动检索 OA
  if ((wf.target_process_type || wf.target_workflow_id) && !summaryWorkflowFieldOptions.value[block.id]?.length) {
    await loadSummaryWorkflowFields(block, undefined, { silent: true })
  }
}

const selectSummaryWorkflowInModal = async (item: ProcessInfo) => {
  const nextID = item.workflow_id || item.process_name || item.process_type
  if (summaryWorkflowSelectedID.value === nextID) return
  summaryWorkflowSelectedID.value = nextID
  summaryWorkflowDraftFields.value = []
  const block = currentSummaryWorkflowBlock.value
  if (!block) return
  await loadSummaryWorkflowFields(block, item, { silent: true })
}

const searchSummaryWorkflows = async () => {
  const keyword = summaryWorkflowKeyword.value.trim()
  if (!keyword) {
    message.warning('请输入流程名称、分类或主表名后再搜索')
    summaryWorkflowRows.value = []
    summaryWorkflowHasSearched.value = false
    return
  }
  summaryWorkflowSearching.value = true
  try {
    summaryWorkflowRows.value = await summaryApi.searchWorkflows(keyword)
    summaryWorkflowHasSearched.value = true
  } catch (e: any) {
    message.error('流程检索失败：' + (e.message || '未知错误'))
  } finally {
    summaryWorkflowSearching.value = false
  }
}

const summaryWorkflowDisplayName = (item: ProcessInfo) =>
  item.process_name || item.process_type || (item.workflow_id && item.workflow_id !== '0' ? `流程 ${item.workflow_id}` : '未命名流程')

const confirmSummaryWorkflowConfig = async () => {
  const block = currentSummaryWorkflowBlock.value
  const mount = block ? getSummaryContextMount(block, 'workflow') : undefined
  if (!block || !mount?.workflow) {
    message.warning('请先搜索并选择目标流程')
    return Promise.reject()
  }

  const row = summaryWorkflowRows.value.find(item =>
    (item.workflow_id || item.process_name || item.process_type) === summaryWorkflowSelectedID.value
  )
  if (row) {
    mount.workflow.target_process_type = row.process_name || row.process_type
    mount.workflow.target_workflow_id = row.workflow_id || ''
    mount.workflow.target_process_label = row.process_type_label || ''
    mount.workflow.target_main_table = row.main_table || ''
  } else if (!mount.workflow.target_process_type && !mount.workflow.target_workflow_id) {
    message.warning('请先搜索并选择目标流程')
    return Promise.reject()
  }

  mount.workflow.data_mode = 'selected_fields'
  mount.workflow.selected_fields = [...summaryWorkflowDraftFields.value]
  summaryWorkflowConfigOpen.value = false
}

const insertSummaryCustomSQLVariable = (block: SummaryBlockConfig, variable: string) => {
  const mount = getSummaryContextMount(block, 'model')
  if (!mount?.model) return
  const current = mount.model.custom_sql || ''
  mount.model.custom_sql = current ? `${current} ${variable}` : variable
}

function createClientId() {
  return globalThis.crypto?.randomUUID?.() || `block_${Date.now()}_${Math.random().toString(16).slice(2)}`
}

const handleSummaryTestConnectionInModal = async () => {
  const processType = newSummaryProcessForm.value.process_type.trim()
  if (!processType) {
    message.warning(t('admin.ruleConfig.enterProcessName'))
    return
  }
  summaryTestingConnection.value = true
  try {
    const info = await summaryApi.testConnection(processType, newSummaryProcessForm.value.main_table_name.trim(), newSummaryProcessForm.value.process_type_label?.trim() || '')
    if (info.table_mismatch || info.type_label_mismatch) {
      const msgs = []
      if (info.table_mismatch) {
        msgs.push(t('admin.ruleConfig.tableMismatch', [info.expected_table || '-']))
        if (info.expected_table) newSummaryProcessForm.value.main_table_name = info.expected_table
      }
      if (info.type_label_mismatch) {
        msgs.push(t('admin.ruleConfig.typeLabelMismatch', [info.expected_type_label || '-']))
        if (info.expected_type_label) newSummaryProcessForm.value.process_type_label = info.expected_type_label
      }
      showTestConnectionFeedback(false, msgs.join('；'))
    } else {
      if (info.main_table) newSummaryProcessForm.value.main_table_name = info.main_table
      if (info.process_type_label) newSummaryProcessForm.value.process_type_label = info.process_type_label
      showTestConnectionFeedback(
        true,
        t('admin.ruleConfig.testConnectionSuccess', [info.process_name || processType, info.main_table || '-', info.process_type_label || '-']),
      )
    }
  } catch (e: any) {
    showTestConnectionFeedback(false, t('admin.ruleConfig.testConnectionFail', [e.message || '未知错误']))
  } finally {
    summaryTestingConnection.value = false
  }
}

const handleAddSummaryProcess = async () => {
  if (!newSummaryProcessForm.value.process_type.trim()) {
    message.warning(t('admin.ruleConfig.enterProcessName'))
    return
  }
  try {
    const created = await summaryApi.createConfig({
      process_type: newSummaryProcessForm.value.process_type.trim(),
      process_type_label: newSummaryProcessForm.value.process_type_label.trim(),
      main_table_name: newSummaryProcessForm.value.main_table_name.trim(),
      embed_enabled: true,
      embed_config: { auto_summary_on_open: true, auto_summary_on_stale: true },
      summary_blocks: [createSummaryBlock()],
    })
    const normalized = normalizeSummaryConfigForUI(created)
    summaryConfigs.value.push(normalized)
    selectedSummaryId.value = normalized.id
    showAddSummaryProcess.value = false
    newSummaryProcessForm.value = { process_type: '', process_type_label: '', main_table_name: '' }
    message.success(t('admin.ruleConfig.processAdded'))
  } catch (e: any) {
    message.error(t('admin.ruleConfig.createConfigFail') + ': ' + (e.message || ''))
  }
}

const handleDeleteSummaryProcess = async (id: string) => {
  try {
    await summaryApi.deleteConfig(id)
    summaryConfigs.value = summaryConfigs.value.filter(c => c.id !== id)
    if (selectedSummaryId.value === id) {
      selectedSummaryId.value = summaryConfigs.value[0]?.id || ''
    }
    message.success(t('admin.ruleConfig.deleteConfigSuccess'))
  } catch (e: any) {
    message.error(t('admin.ruleConfig.deleteConfigFail') + ': ' + (e.message || ''))
  }
}

const handleSummaryTestConnectionInInfo = async () => {
  if (!selectedSummaryConfig.value) return
  const processType = selectedSummaryConfig.value.process_type.trim()
  if (!processType) {
    message.warning(t('admin.ruleConfig.enterProcessName'))
    return
  }
  summaryInfoTestingConnection.value = true
  try {
    const info = await summaryApi.testConnection(processType, selectedSummaryConfig.value.main_table_name.trim(), selectedSummaryConfig.value.process_type_label?.trim() || '')
    if (info.table_mismatch || info.type_label_mismatch) {
      const msgs = []
      if (info.table_mismatch) {
        msgs.push(t('admin.ruleConfig.tableMismatch', [info.expected_table || '-']))
        if (info.expected_table) selectedSummaryConfig.value.main_table_name = info.expected_table
      }
      if (info.type_label_mismatch) {
        msgs.push(t('admin.ruleConfig.typeLabelMismatch', [info.expected_type_label || '-']))
        if (info.expected_type_label) selectedSummaryConfig.value.process_type_label = info.expected_type_label
      }
      showTestConnectionFeedback(false, msgs.join('；'))
    } else {
      if (info.main_table) selectedSummaryConfig.value.main_table_name = info.main_table
      if (info.process_type_label) selectedSummaryConfig.value.process_type_label = info.process_type_label
      showTestConnectionFeedback(
        true,
        t('admin.ruleConfig.testConnectionSuccess', [info.process_name || processType, info.main_table || '-', info.process_type_label || '-']),
      )
    }
  } catch (e: any) {
    showTestConnectionFeedback(false, t('admin.ruleConfig.testConnectionFail', [e.message || '未知错误']))
  } finally {
    summaryInfoTestingConnection.value = false
  }
}

const handleSummarySyncFields = async () => {
  if (!selectedSummaryConfig.value) return
  syncingSummaryFields.value = true
  try {
    const fields = await summaryApi.fetchFields(selectedSummaryConfig.value.id)
    selectedSummaryConfig.value.main_fields = (fields.main_fields || []).map((f: any) => ({ ...f, selected: true }))
    selectedSummaryConfig.value.detail_tables = (fields.detail_tables || []).map((dt: any) => ({
      ...dt,
      fields: (dt.fields || []).map((f: any) => ({ ...f, selected: true })),
    }))
    message.success(t('admin.ruleConfig.fetchFieldsSuccess'))
  } catch (e: any) {
    message.error(t('admin.ruleConfig.fetchFieldsFail') + ': ' + (e.message || ''))
  } finally {
    syncingSummaryFields.value = false
  }
}

const addSummaryBlock = () => {
  if (!selectedSummaryConfig.value) return
  const block = createSummaryBlock()
  block.title = `总结块 ${selectedSummaryConfig.value.summary_blocks.length + 1}`
  block.sort_order = selectedSummaryConfig.value.summary_blocks.length + 1
  selectedSummaryConfig.value.summary_blocks.push(block)
}

const removeSummaryBlock = (blockId: string) => {
  if (!selectedSummaryConfig.value) return
  selectedSummaryConfig.value.summary_blocks = selectedSummaryConfig.value.summary_blocks.filter(b => b.id !== blockId)
  selectedSummaryConfig.value.summary_blocks.forEach((b, idx) => { b.sort_order = idx + 1 })
}

const handleSaveSummaryConfig = async () => {
  if (!selectedSummaryConfig.value) return
  const cfg = selectedSummaryConfig.value
  if (!cfg.summary_blocks.some(block => block.enabled)) {
    message.warning('至少需要启用一个总结块')
    return
  }
  const emptySelectedBlock = cfg.summary_blocks.find(block => block.enabled && block.field_mode === 'selected' && block.selected_fields.length === 0)
  if (emptySelectedBlock) {
    message.warning(`总结块「${emptySelectedBlock.title || '未命名'}」需要至少选择一个字段`)
    return
  }
  savingSummary.value = true
  try {
    const updated = await summaryApi.updateConfig(cfg.id, {
      process_type: cfg.process_type,
      process_type_label: cfg.process_type_label,
      main_table_name: cfg.main_table_name,
      main_fields: cfg.main_fields,
      detail_tables: cfg.detail_tables,
      summary_blocks: cfg.summary_blocks.map((b, idx) => ({ ...b, sort_order: idx + 1 })),
      embed_enabled: cfg.embed_enabled ?? true,
      embed_config: cfg.embed_config,
      status: cfg.status,
    })
    const normalized = normalizeSummaryConfigForUI(updated)
    const idx = summaryConfigs.value.findIndex(c => c.id === cfg.id)
    if (idx >= 0) summaryConfigs.value[idx] = normalized
    message.success('流程总结配置已保存')
  } catch (e: any) {
    message.error(t('admin.ruleConfig.updateConfigFail') + ': ' + (e.message || ''))
  } finally {
    savingSummary.value = false
  }
}

// 快照原始权限，用于保存时检测是否有权限降级
const originalArchivePerms = ref<Record<string, boolean>>({})
watch(selectedArchiveId, (newId) => {
  const cfg = archiveConfigs.value.find(c => c.id === newId)
  originalArchivePerms.value = cfg ? { ...(cfg.user_permissions as any) } : {}
}, { immediate: true })

// 当选中归档流程变化时，从 API 加载该流程的规则
const currentArchiveRules = ref<ArchiveRule[]>([])
const loadingArchiveRules = ref(false)
const selectedArchiveRuleIds = ref<string[]>([])
const batchDeletingArchiveRules = ref(false)
const selectedArchiveRuleIdSet = computed(() => new Set(selectedArchiveRuleIds.value))
const allArchiveRulesSelected = computed(() => currentArchiveRules.value.length > 0 && selectedArchiveRuleIds.value.length === currentArchiveRules.value.length)
const archiveRulesSelectionIndeterminate = computed(() => selectedArchiveRuleIds.value.length > 0 && !allArchiveRulesSelected.value)

const toggleArchiveRuleSelection = (id: string, checked: boolean) => {
  selectedArchiveRuleIds.value = checked
    ? [...new Set([...selectedArchiveRuleIds.value, id])]
    : selectedArchiveRuleIds.value.filter(ruleId => ruleId !== id)
}

const toggleAllArchiveRules = (checked: boolean) => {
  selectedArchiveRuleIds.value = checked ? currentArchiveRules.value.map(rule => rule.id) : []
}

watch(selectedArchiveId, async (newId) => {
  selectedArchiveRuleIds.value = []
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
const addArchiveProcessSearch = ref({
  keyword: '',
  rows: [] as ProcessInfo[],
  searching: false,
  hasSearched: false,
  selectedId: '',
})

watch(showAddArchiveProcess, (open) => {
  if (open) {
    newArchiveProcessForm.value = { process_type: '', process_type_label: '', main_table_name: '' }
    resetAddProcessSearch(addArchiveProcessSearch.value)
  }
})

const handleTestConnectionInArchiveModal = async () => {
  const processType = newArchiveProcessForm.value.process_type.trim()
  if (!processType) {
    message.warning(t('admin.ruleConfig.enterProcessName'))
    return
  }
  archiveTestingConnection.value = true
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
      showTestConnectionFeedback(false, msgs.join('；'))
    } else {
      if (info.main_table) newArchiveProcessForm.value.main_table_name = info.main_table
      if (info.process_type_label) newArchiveProcessForm.value.process_type_label = info.process_type_label
      showTestConnectionFeedback(
        true,
        t('admin.ruleConfig.testConnectionSuccess', [info.process_name || processType, info.main_table || '-', info.process_type_label || '-']),
      )
    }
  } catch (e: any) {
    showTestConnectionFeedback(false, t('admin.ruleConfig.testConnectionFail', [e.message || '未知错误']))
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

const archiveContextFieldOptions = computed(() => {
  const includeAll = selectedArchiveConfig.value?.field_mode === 'all'
  return archiveAllAvailableFields.value
    .filter(f => includeAll || f.selected)
    .map(f => ({
      label: `${f.field_name}（${f.sourceLabel}）`,
      value: `${f.source}:${f.field_key}`,
    }))
})



const archiveSelectedFieldSearchQuery = ref('')
const archiveLeftSelectedKeys = ref<string[]>([])
const archiveRightSelectedKeys = ref<string[]>([])

const archiveUnselectedFieldsFlat = computed(() => {
  const q = archiveFieldSearchQuery.value.toLowerCase().trim()
  return archiveAllAvailableFields.value.filter(f => {
    if (f.selected) return false
    if (!q) return true
    return f.field_name.toLowerCase().includes(q) || f.field_key.toLowerCase().includes(q)
  })
})
const archiveUnselectedPagination = usePagination(archiveUnselectedFieldsFlat, 5)

const toggleArchiveLeftSelectAll = () => {
  if (archiveLeftSelectedKeys.value.length === archiveUnselectedFieldsFlat.value.length && archiveUnselectedFieldsFlat.value.length > 0) {
    archiveLeftSelectedKeys.value = []
  } else {
    archiveLeftSelectedKeys.value = archiveUnselectedFieldsFlat.value.map(f => f.field_key + '_' + f.source)
  }
}

const toggleArchiveLeftSelect = (fieldId: string) => {
  const idx = archiveLeftSelectedKeys.value.indexOf(fieldId)
  if (idx >= 0) archiveLeftSelectedKeys.value.splice(idx, 1)
  else archiveLeftSelectedKeys.value.push(fieldId)
}

const archiveBatchPick = () => {
  archiveUnselectedFieldsFlat.value.filter(f => archiveLeftSelectedKeys.value.includes(f.field_key + '_' + f.source)).forEach(archivePickField)
  archiveLeftSelectedKeys.value = []
}

const archiveSelectedFieldsFlat = computed(() => {
  const q = archiveSelectedFieldSearchQuery.value.toLowerCase().trim()
  return archiveAllAvailableFields.value.filter(f => {
    if (!f.selected) return false
    if (!q) return true
    return f.field_name.toLowerCase().includes(q) || f.field_key.toLowerCase().includes(q) || f.sourceLabel.toLowerCase().includes(q)
  })
})
const archiveSelectedPagination = usePagination(archiveSelectedFieldsFlat, 5)

const toggleArchiveRightSelectAll = () => {
  if (archiveRightSelectedKeys.value.length === archiveSelectedFieldsFlat.value.length && archiveSelectedFieldsFlat.value.length > 0) {
    archiveRightSelectedKeys.value = []
  } else {
    archiveRightSelectedKeys.value = archiveSelectedFieldsFlat.value.map(f => f.field_key + '_' + f.source)
  }
}

const toggleArchiveRightSelect = (fieldId: string) => {
  const idx = archiveRightSelectedKeys.value.indexOf(fieldId)
  if (idx >= 0) archiveRightSelectedKeys.value.splice(idx, 1)
  else archiveRightSelectedKeys.value.push(fieldId)
}

const archiveBatchUnpick = () => {
  archiveSelectedFieldsFlat.value.filter(f => archiveRightSelectedKeys.value.includes(f.field_key + '_' + f.source)).forEach(archiveUnpickField)
  archiveRightSelectedKeys.value = []
}

const archivePageSelectedFieldSearchQuery = ref('')
const archivePageSelectedFieldsFlat = computed(() => {
  const q = archivePageSelectedFieldSearchQuery.value.toLowerCase().trim()
  return archiveAllAvailableFields.value.filter(f => {
    if (!f.selected) return false
    if (!q) return true
    return f.field_name.toLowerCase().includes(q) || f.field_key.toLowerCase().includes(q) || f.sourceLabel.toLowerCase().includes(q)
  })
})
const archivePageSelectedPagination = usePagination(archivePageSelectedFieldsFlat, 5)

const openArchiveFieldPicker = () => {
  archiveFieldSearchQuery.value = ''
  archiveSelectedFieldSearchQuery.value = ''
  archiveLeftSelectedKeys.value = []
  archiveRightSelectedKeys.value = []
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
        context_enabled: rule.context_enabled,
        context_mounts: rule.context_mounts,
        // 强制根据级别同步启用状态
        enabled: rule.rule_scope === 'mandatory' ? true : (rule.rule_scope === 'default_off' ? false : true)
      })
      const idx = currentArchiveRules.value.findIndex(r => r.id === editingArchiveRule.value!.id)
      if (idx >= 0) currentArchiveRules.value[idx] = updated
    } else {
      // 创建规则
      // 根据规则级别设置初始状态
      const initialEnabled = rule.rule_scope !== 'default_off'
      const created = await archiveApi.createRule({
        config_id: selectedArchiveConfig.value.id,
        process_type: selectedArchiveConfig.value.process_type,
        rule_content: rule.rule_content,
        rule_scope: rule.rule_scope,
        related_flow: rule.related_flow,
        context_enabled: rule.context_enabled,
        context_mounts: rule.context_mounts,
        enabled: initialEnabled,
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
    selectedArchiveRuleIds.value = selectedArchiveRuleIds.value.filter(ruleId => ruleId !== id)
    message.success(t('admin.ruleConfig.deleted'))
  } catch (e: any) {
    message.error(t('admin.ruleConfig.deleteRuleFail') + ': ' + (e.message || ''))
  }
}

const batchDeleteArchiveRules = async () => {
  if (!selectedArchiveConfig.value || selectedArchiveRuleIds.value.length === 0) return
  const ids = [...selectedArchiveRuleIds.value]
  batchDeletingArchiveRules.value = true
  try {
    const deletedCount = await archiveApi.batchDeleteRules(selectedArchiveConfig.value.id, ids)
    const deletedIDs = new Set(ids)
    currentArchiveRules.value = currentArchiveRules.value.filter(rule => !deletedIDs.has(rule.id))
    selectedArchiveRuleIds.value = []
    message.success(t('admin.ruleConfig.batchDeleteSuccess', `${deletedCount}`))
  } catch (e: any) {
    message.error(t('admin.ruleConfig.batchDeleteFail') + ': ' + (e.message || ''))
  } finally {
    batchDeletingArchiveRules.value = false
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

const insertArchiveReasoningVariable = (variable: string) => {
  archiveReasoningTextareaRef.value?.insertAtCursor(variable)
}

const insertArchiveExtractionVariable = (variable: string) => {
  archiveExtractionTextareaRef.value?.insertAtCursor(variable)
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

// 归档基本信息也提供测试连接
const handleArchiveTestConnectionInInfo = async () => {
  if (!selectedArchiveConfig.value) return
  const processType = selectedArchiveConfig.value.process_type.trim()
  if (!processType) {
    message.warning(t('admin.ruleConfig.enterProcessName'))
    return
  }
  archiveInfoTestingConnection.value = true
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
      showTestConnectionFeedback(false, msgs.join('；'))
    } else {
      if (info.main_table && selectedArchiveConfig.value) selectedArchiveConfig.value.main_table_name = info.main_table
      if (info.process_type_label && selectedArchiveConfig.value) selectedArchiveConfig.value.process_type_label = info.process_type_label
      showTestConnectionFeedback(
        true,
        t('admin.ruleConfig.testConnectionSuccess', [info.process_name || processType, info.main_table || '-', info.process_type_label || '-']),
      )
    }
  } catch (e: any) {
    showTestConnectionFeedback(false, t('admin.ruleConfig.testConnectionFail', [e.message || '未知错误']))
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

const ensureArchiveAC = () => {
  if (!selectedArchiveConfig.value) return null
  const ac = selectedArchiveConfig.value.access_control
  if (!ac || typeof ac !== 'object') {
    selectedArchiveConfig.value.access_control = { allowed_roles: [], allowed_members: [], allowed_departments: [] }
  } else {
    if (!Array.isArray(ac.allowed_roles)) ac.allowed_roles = []
    if (!Array.isArray(ac.allowed_members)) ac.allowed_members = []
    if (!Array.isArray(ac.allowed_departments)) ac.allowed_departments = []
  }
  return selectedArchiveConfig.value.access_control!
}

const toggleArchiveRole = (roleId: string) => {
  const ac = ensureArchiveAC()
  if (!ac) return
  const idx = ac.allowed_roles.indexOf(roleId)
  if (idx >= 0) ac.allowed_roles.splice(idx, 1)
  else ac.allowed_roles.push(roleId)
}

const toggleArchiveMember = (memberId: string) => {
  const ac = ensureArchiveAC()
  if (!ac) return
  const idx = ac.allowed_members.indexOf(memberId)
  if (idx >= 0) ac.allowed_members.splice(idx, 1)
  else ac.allowed_members.push(memberId)
}

const toggleArchiveDept = (deptId: string) => {
  const ac = ensureArchiveAC()
  if (!ac) return
  const idx = ac.allowed_departments.indexOf(deptId)
  if (idx >= 0) ac.allowed_departments.splice(idx, 1)
  else ac.allowed_departments.push(deptId)
}

const handleSaveArchiveConfig = async () => {
  if (!selectedArchiveConfig.value) return
  const cfg = selectedArchiveConfig.value

  // 检测权限降级，需要用户确认
  const disabledKeys = getDowngradedPermKeys(originalArchivePerms.value, cfg.user_permissions as any)
  if (disabledKeys.length > 0) {
    const confirmed = await confirmPermDowngrade(disabledKeys, archivePermissionLabels.value)
    if (!confirmed) return
  }

  savingArchive.value = true
  try {
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
    originalArchivePerms.value = { ...(cfg.user_permissions as any) }
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

//===== 审核工作台访问控制（角色/成员/部门）=====
const auditRoleSearch = ref('')
const auditMemberSearch = ref('')
const auditDeptSearch = ref('')

const filteredAuditRoles = computed(() => {
  const q = auditRoleSearch.value.toLowerCase().trim()
  if (!q) return roles.value
  return roles.value.filter(r => r.name.toLowerCase().includes(q))
})

const filteredAuditMembers = computed(() => {
  const q = auditMemberSearch.value.toLowerCase().trim()
  if (!q) return members.value
  return members.value.filter(m => m.name.toLowerCase().includes(q) || m.department_name.toLowerCase().includes(q))
})

const filteredAuditDepts = computed(() => {
  const q = auditDeptSearch.value.toLowerCase().trim()
  if (!q) return departments.value
  return departments.value.filter(d => d.name.toLowerCase().includes(q))
})

const ensureAuditAC = () => {
  if (!selectedConfig.value) return null
  const ac = selectedConfig.value.access_control
  if (!ac || typeof ac !== 'object') {
    selectedConfig.value.access_control = { allowed_roles: [], allowed_members: [], allowed_departments: [] }
  } else {
    if (!Array.isArray(ac.allowed_roles)) ac.allowed_roles = []
    if (!Array.isArray(ac.allowed_members)) ac.allowed_members = []
    if (!Array.isArray(ac.allowed_departments)) ac.allowed_departments = []
  }
  return selectedConfig.value.access_control!
}

const toggleAuditRole = (roleId: string) => {
  const ac = ensureAuditAC()
  if (!ac) return
  const idx = ac.allowed_roles.indexOf(roleId)
  if (idx >= 0) ac.allowed_roles.splice(idx, 1)
  else ac.allowed_roles.push(roleId)
}

const toggleAuditMember = (memberId: string) => {
  const ac = ensureAuditAC()
  if (!ac) return
  const idx = ac.allowed_members.indexOf(memberId)
  if (idx >= 0) ac.allowed_members.splice(idx, 1)
  else ac.allowed_members.push(memberId)
}

const toggleAuditDept = (deptId: string) => {
  const ac = ensureAuditAC()
  if (!ac) return
  const idx = ac.allowed_departments.indexOf(deptId)
  if (idx >= 0) ac.allowed_departments.splice(idx, 1)
  else ac.allowed_departments.push(deptId)
}

const saving = ref(false)
const savingCron = ref(false)
const savingArchive = ref(false)

/**
 * 比较原始权限与当前权限，返回从 true 变为 false 的权限 key 列表。
 */
function getDowngradedPermKeys(
  original: Record<string, boolean>,
  current: Record<string, boolean>,
): string[] {
  return Object.keys(original).filter(k => original[k] === true && current[k] === false)
}

/**
 * 若有权限被关闭，弹出确认对话框；用户取消则返回 false，确认则返回 true。
 */
function confirmPermDowngrade(
  disabledKeys: string[],
  labels: Record<string, { label: string }>,
): Promise<boolean> {
  const names = disabledKeys.map(k => labels[k]?.label ?? k).join('、')
  return new Promise(resolve => {
    Modal.confirm({
      title: t('admin.ruleConfig.permDowngradeTitle'),
      content: t('admin.ruleConfig.permDowngradeContent', [names]),
      okText: t('common.confirm'),
      cancelText: t('common.cancel'),
      onOk: () => resolve(true),
      onCancel: () => resolve(false),
    })
  })
}

const handleSave = async () => {
  if (!selectedConfig.value) return
  const cfg = selectedConfig.value

  // 检测权限降级，需要用户确认
  const disabledKeys = getDowngradedPermKeys(originalAuditPerms.value, cfg.user_permissions as any)
  if (disabledKeys.length > 0) {
    const confirmed = await confirmPermDowngrade(disabledKeys, permissionLabels.value)
    if (!confirmed) return
  }

  saving.value = true
  try {
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
      access_control: cfg.access_control ?? { allowed_roles: [], allowed_members: [], allowed_departments: [] },
      embed_enabled: cfg.embed_enabled ?? false,
      status: cfg.status,
    })
    // 更新本地数据并刷新快照
    const idx = processConfigs.value.findIndex(c => c.id === cfg.id)
    if (idx !== -1) processConfigs.value[idx] = updated
    originalAuditPerms.value = { ...(cfg.user_permissions as any) }
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

    <!--顶级选项卡：审核工作台 / 定时任务配置 / 归档复盘 / 流程总结-->
    <div class="top-tab-nav">
      <button
        v-for="tab in [
          { key: 'audit', label: t('admin.ruleConfig.tabAudit'), icon: DashboardOutlined },
          { key: 'cron', label: t('admin.ruleConfig.tabCron'), icon: ClockCircleOutlined },
          { key: 'archive', label: t('admin.ruleConfig.tabArchive'), icon: FolderOpenOutlined },
          { key: 'summary', label: '流程总结', icon: FileTextOutlined },
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
              <div class="test-connection-hint" style="margin-top: 4px; font-size: 12px; color: var(--color-text-tertiary);">
                {{ t('admin.ruleConfig.testConnectionHint') }}
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
            <div class="rules-toolbar-summary">
              <a-checkbox
                :checked="allRulesSelected"
                :indeterminate="rulesSelectionIndeterminate"
                :disabled="currentRules.length === 0 || batchDeletingRules"
                @change="(event: any) => toggleAllRules(!!event.target.checked)"
              >
                {{ t('admin.ruleConfig.selectAll') }}
              </a-checkbox>
              <span class="rules-count">{{ t('admin.ruleConfig.totalRules', `${currentRules.length}`) }}</span>
              <span v-if="selectedRuleIds.length" class="rules-selected-count">
                {{ t('admin.ruleConfig.selectedRules', `${selectedRuleIds.length}`) }}
              </span>
            </div>
            <div class="rules-toolbar-actions">
              <template v-if="selectedRuleIds.length">
                <a-popconfirm
                  :title="t('admin.ruleConfig.batchDeleteConfirm', `${selectedRuleIds.length}`)"
                  @confirm="batchDeleteRules"
                >
                  <a-button danger :loading="batchDeletingRules">
                    <DeleteOutlined /> {{ t('admin.ruleConfig.batchDelete') }}
                  </a-button>
                </a-popconfirm>
              </template>
              <template v-else>
                <a-button :disabled="ruleImportLoading" @click="handlePasteImport('audit')">
                  <SnippetsOutlined /> {{ t('admin.ruleConfig.pasteImport') }}
                </a-button>
                <a-button
                  :loading="ruleImportLoading && ruleImportTarget === 'audit'"
                  :disabled="ruleImportLoading || !ruleImportCapability?.enabled"
                  :title="ruleImportCapability?.reason || ''"
                  @click="handleImportRules('audit')"
                >
                  <UploadOutlined /> {{ t('admin.ruleConfig.fileImport') }}
                </a-button>
                <a-button type="primary" @click="openRuleEditor()">
                  <PlusOutlined /> {{ t('admin.ruleConfig.manualAddBtn') }}
                </a-button>
              </template>
            </div>
          </div>

          <div class="rules-list">
            <a-spin v-if="loadingRules" style="display: block; text-align: center; padding: 24px;" />
            <div v-for="rule in currentRules" :key="rule.id" class="rule-card">
              <div class="rule-card-left">
                <a-checkbox
                  class="rule-select-checkbox"
                  :checked="selectedRuleIdSet.has(rule.id)"
                  :disabled="batchDeletingRules"
                  :aria-label="t('admin.ruleConfig.selectRule')"
                  @change="(event: any) => toggleRuleSelection(rule.id, !!event.target.checked)"
                />
                <div class="rule-scope-badge" :style="{ color: scopeConfig[rule.rule_scope]?.color, background: scopeConfig[rule.rule_scope]?.bg }">
                  <component :is="scopeConfig[rule.rule_scope]?.icon" />
                  {{ scopeConfig[rule.rule_scope]?.label }}
                </div>
                <div class="rule-card-body">
                  <div class="rule-card-content">{{ rule.rule_content }}</div>
                  <div class="rule-card-meta">
                    <span v-if="rule.source === 'file_import'" class="rule-source-tag">{{ t('admin.ruleConfig.fileImportTag') }}</span>
                    <span v-else-if="rule.source === 'paste_import'" class="rule-source-tag">{{ t('admin.ruleConfig.pasteImportTag') }}</span>
                    <span v-else class="rule-source-tag rule-source-tag--manual">{{ t('admin.ruleConfig.manualAddTag') }}</span>
                    <span v-if="rule.related_flow" class="rule-flow-tag">
                      <NodeIndexOutlined /> {{ t('admin.ruleConfig.relatedFlow') }}
                    </span>
                    <span v-if="hasActualExternalContext(rule)" class="rule-flow-tag">
                      <NodeIndexOutlined /> 外部关联
                    </span>
                  </div>
                </div>
              </div>
              <div class="rule-card-actions">
                <a-switch
                  :checked="rule.enabled"
                  :disabled="rule.rule_scope === 'mandatory'"
                  size="small"
                  @change="(checked: any) => { rulesApi.updateRule(rule.id, { enabled: !!checked }).then(updated => { const idx = currentRules.findIndex(r => r.id === rule.id); if (idx >= 0) currentRules[idx] = updated }) }"
                />
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
                <PromptTextarea
                  v-model:value="selectedConfig.ai_config.system_reasoning_prompt"
                  :rows="6"
                  :placeholder="t('admin.ruleConfig.systemReasoningPlaceholder')"
                  :dialog-title="t('admin.ruleConfig.systemReasoningPrompt')"
                />
              </div>

              <div class="ai-form-group">
                <div class="prompt-section-header">
                  <div class="prompt-section-title">
                    <span class="prompt-phase-badge prompt-phase-badge--extraction">{{ t('admin.ruleConfig.phase2Label') }}</span>
                    <label class="ai-form-label">{{ t('admin.ruleConfig.systemExtractionPrompt') }}</label>
                  </div>
                </div>
                <PromptTextarea
                  v-model:value="selectedConfig.ai_config.system_extraction_prompt"
                  :rows="6"
                  :placeholder="t('admin.ruleConfig.systemExtractionPlaceholder')"
                  :dialog-title="t('admin.ruleConfig.systemExtractionPrompt')"
                  disabled
                />
                <div class="system-prompt-readonly-hint">{{ t('admin.ruleConfig.systemPromptReadonly') }}</div>
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
                <PromptVariableBar
                  :data-variables="reasoningPromptVariables"
                  :system-variables="systemPromptVariables"
                  @insert="insertReasoningVariable"
                />
                <PromptTextarea
                  ref="reasoningTextareaRef"
                  v-model:value="selectedConfig.ai_config.user_reasoning_prompt"
                  :rows="8"
                  :placeholder="t('admin.ruleConfig.userReasoningPlaceholder')"
                  :dialog-title="t('admin.ruleConfig.userReasoningPrompt')"
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
                <PromptVariableBar
                  :data-variables="extractionPromptVariables"
                  :system-variables="systemPromptVariables"
                  @insert="insertExtractionVariable"
                />
                <PromptTextarea
                  ref="extractionTextareaRef"
                  v-model:value="selectedConfig.ai_config.user_extraction_prompt"
                  :rows="6"
                  :placeholder="t('admin.ruleConfig.userExtractionPlaceholder')"
                  :dialog-title="t('admin.ruleConfig.userExtractionPrompt')"
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

          <div class="permission-item" style="margin-top: 20px;">
            <div class="permission-info">
              <div class="permission-label">{{ t('admin.ruleConfig.embedEnabled') }}</div>
              <div class="permission-desc">{{ t('admin.ruleConfig.embedEnabledDesc') }}</div>
            </div>
            <a-switch
              v-model:checked="selectedConfig.embed_enabled"
              :checked-children="t('admin.ruleConfig.switchAllow')"
              :un-checked-children="t('admin.ruleConfig.switchDeny')"
            />
          </div>

          <!-- 审核工作台访问控制 -->
          <div class="section-header" style="margin-top: 28px;">
            <div>
              <h4 class="section-title">{{ t('admin.ruleConfig.auditAccessTitle') }}</h4>
              <p class="section-desc">{{ t('admin.ruleConfig.auditAccessDesc') }}</p>
            </div>
          </div>

          <div class="access-control-section">
            <div class="access-control-group">
              <div class="access-control-label"><TeamOutlined /> {{ t('admin.ruleConfig.auditAllowedRoles') }}</div>
              <div class="access-control-search">
                <a-input v-model:value="auditRoleSearch" :placeholder="t('admin.ruleConfig.auditAccessSearch')" allow-clear size="small" style="max-width: 280px;">
                  <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
                </a-input>
              </div>
              <div class="access-control-tags" style="gap: 8px;">
                <div
                  v-for="role in filteredAuditRoles"
                  :key="role.id"
                  class="access-tag"
                  :class="{ 'access-tag--active': (selectedConfig.access_control?.allowed_roles || []).includes(role.id) }"
                  @click="toggleAuditRole(role.id)"
                >
                  <CheckOutlined v-if="(selectedConfig.access_control?.allowed_roles || []).includes(role.id)" class="access-tag-check" />
                  {{ role.name }}
                </div>
              </div>
            </div>
            <div class="access-control-group" style="margin-top: 16px;">
              <div class="access-control-label"><UserOutlined /> {{ t('admin.ruleConfig.auditAllowedMembers') }}</div>
              <div class="access-control-search">
                <a-input v-model:value="auditMemberSearch" :placeholder="t('admin.ruleConfig.auditAccessSearch')" allow-clear size="small" style="max-width: 280px;">
                  <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
                </a-input>
              </div>
              <div class="access-control-tags" style="gap: 8px;">
                <div
                  v-for="member in filteredAuditMembers"
                  :key="member.id"
                  class="access-tag"
                  :class="{ 'access-tag--active': (selectedConfig.access_control?.allowed_members || []).includes(member.id) }"
                  @click="toggleAuditMember(member.id)"
                >
                  <CheckOutlined v-if="(selectedConfig.access_control?.allowed_members || []).includes(member.id)" class="access-tag-check" />
                  {{ member.name }}
                  <span class="access-tag-dept">{{ member.department_name }}</span>
                </div>
              </div>
            </div>
            <div class="access-control-group" style="margin-top: 16px;">
              <div class="access-control-label"><AppstoreOutlined /> {{ t('admin.ruleConfig.auditAllowedDepts') }}</div>
              <div class="access-control-search">
                <a-input v-model:value="auditDeptSearch" :placeholder="t('admin.ruleConfig.auditAccessSearch')" allow-clear size="small" style="max-width: 280px;">
                  <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
                </a-input>
              </div>
              <div class="access-control-tags" style="gap: 8px;">
                <div
                  v-for="dept in filteredAuditDepts"
                  :key="dept.id"
                  class="access-tag"
                  :class="{ 'access-tag--active': (selectedConfig.access_control?.allowed_departments || []).includes(dept.id) }"
                  @click="toggleAuditDept(dept.id)"
                >
                  <CheckOutlined v-if="(selectedConfig.access_control?.allowed_departments || []).includes(dept.id)" class="access-tag-check" />
                  {{ dept.name }}
                  <span class="access-tag-dept">{{ dept.member_count }}人</span>
                </div>
              </div>
            </div>
          </div>
        </div>

        <div v-if="activeTab !== 'rules'" class="config-actions">
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

    <!-- ==================== 流程总结配置 ==================== -->
    <div v-if="topTab === 'summary'" class="main-layout">
      <div class="process-nav">
        <div class="process-nav-header">
          <FileTextOutlined />
          <span>总结流程</span>
          <button class="add-process-btn" @click="showAddSummaryProcess = true" title="新增流程">
            <PlusOutlined />
          </button>
        </div>
        <a-spin v-if="loadingSummary" style="display: block; padding: 20px;" />
        <div
          v-for="cfg in summaryConfigs"
          :key="cfg.id"
          class="process-nav-item"
          :class="{ 'process-nav-item--active': selectedSummaryId === cfg.id }"
          @click="selectedSummaryId = cfg.id"
        >
          <div style="flex: 1; min-width: 0;">
            <div class="process-nav-name">{{ cfg.process_type }}</div>
            <div v-if="cfg.process_type_label" class="process-nav-path">{{ cfg.process_type_label }}</div>
          </div>
          <a-popconfirm title="确认删除该总结配置？" @confirm.stop="handleDeleteSummaryProcess(cfg.id)" placement="right">
            <button class="icon-btn icon-btn--danger icon-btn--sm" @click.stop style="opacity: 0.5; flex-shrink: 0;">
              <DeleteOutlined />
            </button>
          </a-popconfirm>
        </div>
      </div>

      <div v-if="selectedSummaryConfig" class="config-panel">
        <div class="config-panel-header">
          <h2 class="config-panel-title">{{ selectedSummaryConfig.process_type }}</h2>
          <p v-if="selectedSummaryConfig.process_type_label" class="config-panel-subtitle">{{ selectedSummaryConfig.process_type_label }}</p>
        </div>

        <div class="tab-nav">
          <button
            v-for="tab in [
              { key: 'info', label: t('admin.ruleConfig.infoTab'), icon: InfoCircleOutlined },
              { key: 'fields', label: '引入字段', icon: AppstoreOutlined },
              { key: 'ai', label: 'AI 总结', icon: RobotOutlined },
              { key: 'embed', label: 'OA 嵌入', icon: SafetyCertificateOutlined },
            ]"
            :key="tab.key"
            class="tab-btn"
            :class="{ 'tab-btn--active': summaryActiveTab === tab.key }"
            @click="summaryActiveTab = tab.key"
          >
            <component :is="tab.icon" />
            {{ tab.label }}
          </button>
        </div>

        <div v-if="summaryActiveTab === 'info'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">基础信息</h4>
              <p class="section-desc">维护流程名称、分类和主表映射，保持与 OA 流程一致</p>
            </div>
          </div>

          <a-form layout="vertical" class="info-form">
            <a-form-item label="流程名称">
              <a-input v-model:value="selectedSummaryConfig.process_type" placeholder="OA 流程名称" />
            </a-form-item>
            <a-form-item label="流程分类">
              <a-input v-model:value="selectedSummaryConfig.process_type_label" placeholder="流程分类名称" />
            </a-form-item>
            <a-form-item label="主表名称">
              <div style="display: flex; gap: 8px;">
                <a-input v-model:value="selectedSummaryConfig.main_table_name" placeholder="OA 主表名称" style="flex: 1;" />
                <a-button :loading="summaryInfoTestingConnection" @click="handleSummaryTestConnectionInInfo">
                  <template #icon><DatabaseOutlined /></template>
                  {{ summaryInfoTestingConnection ? '测试中' : '测试连接' }}
                </a-button>
              </div>
              <div class="test-connection-hint" style="margin-top: 4px; font-size: 12px; color: var(--color-text-tertiary);">
                {{ t('admin.ruleConfig.testConnectionHint') }}
              </div>
            </a-form-item>
          </a-form>
        </div>

        <div v-if="summaryActiveTab === 'fields'" class="tab-content">
          <div class="section-header" style="display: flex; justify-content: space-between; align-items: center;">
            <div>
              <h4 class="section-title">引入字段</h4>
              <p class="section-desc">从 OA 同步主表、明细表和附件字段，供总结块按需引用</p>
            </div>
            <a-button :loading="syncingSummaryFields" @click="handleSummarySyncFields">
              <template #icon><DatabaseOutlined /></template>
              {{ syncingSummaryFields ? '同步中' : '同步字段' }}
            </a-button>
          </div>

          <div style="margin-bottom: 12px; max-width: 300px;">
            <a-input
              v-model:value="summaryPageFieldSearchQuery"
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
                <tr v-for="field in summaryPageFieldsPagination.paged.value" :key="field.value">
                  <td style="padding-left: 24px; font-weight: 500;">{{ field.field_name }}</td>
                  <td class="text-mono" style="font-size: 13px;">{{ field.field_key }}</td>
                  <td><span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span></td>
                  <td class="text-secondary" style="font-size: 13px;">{{ field.sourceLabel }}</td>
                </tr>
                <tr v-if="summaryPageFieldsPagination.paged.value.length === 0">
                  <td colspan="4" class="empty-cell">{{ summaryPageFieldSearchQuery ? (t('admin.ruleConfig.noSearchResult') || '未找到匹配字段') : '暂无字段，请先同步字段' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div class="pagination-wrapper" style="margin-top: 12px; text-align: right;">
            <a-pagination
              v-model:current="summaryPageFieldsPagination.current.value"
              v-model:page-size="summaryPageFieldsPagination.pageSize.value"
              :total="summaryPageFieldsPagination.total.value"
              size="small"
              show-size-changer
              show-quick-jumper
              :page-size-options="['8', '20', '50']"
              @change="summaryPageFieldsPagination.onChange"
              @showSizeChange="summaryPageFieldsPagination.onChange"
            />
          </div>
        </div>

        <div v-if="summaryActiveTab === 'ai'" class="tab-content">
          <div class="section-header" style="display: flex; justify-content: space-between; align-items: center;">
            <div>
              <h4 class="section-title">AI 总结</h4>
              <p class="section-desc">系统提示词只读展示；每个总结块独立配置字段范围和用户提示词</p>
            </div>
            <a-button type="primary" @click="addSummaryBlock">
              <PlusOutlined /> 新增块
            </a-button>
          </div>

          <div class="ai-prompt-section" style="margin-top: 0;">
            <div class="ai-prompt-section-header">
              <div class="ai-prompt-section-tag ai-prompt-section-tag--system">{{ t('admin.ruleConfig.systemPromptTag') }}</div>
            </div>
            <p class="ai-prompt-section-desc">流程总结使用后端固定系统提示词，前台仅展示，不允许修改。</p>
            <PromptTextarea
              :value="fixedSummarySystemPrompt"
              :rows="7"
              :dialog-title="t('admin.ruleConfig.summarySystemPrompt')"
              disabled
            />
            <div class="system-prompt-readonly-hint">系统提示词由后端固定维护，前台仅允许查看，不参与保存。</div>
          </div>

          <div class="summary-block-list" style="margin-top: 16px;">
            <div
              v-for="(block, idx) in selectedSummaryConfig.summary_blocks"
              :key="block.id"
              class="summary-block-card"
            >
              <div class="summary-block-head">
                <div class="summary-block-index">{{ idx + 1 }}</div>
                <a-input v-model:value="block.title" placeholder="块标题" style="max-width: 280px;" />
                <a-switch v-model:checked="block.enabled" checked-children="启用" un-checked-children="停用" />
                <a-popconfirm
                  v-if="selectedSummaryConfig.summary_blocks.length > 1"
                  title="确认删除该总结块？"
                  @confirm="removeSummaryBlock(block.id)"
                >
                  <button class="icon-btn icon-btn--danger"><DeleteOutlined /></button>
                </a-popconfirm>
              </div>

              <div class="summary-block-option-row">
                <div class="summary-block-option-copy">
                  <div class="summary-block-option-title">{{ t('admin.ruleConfig.summaryIncludeAllData') }}</div>
                  <div class="summary-block-option-desc">{{ t('admin.ruleConfig.summaryIncludeAllDataDesc') }}</div>
                </div>
                <a-switch
                  v-model:checked="block.include_meta"
                  checked-children="全部"
                  un-checked-children="自定义"
                />
              </div>

              <div v-if="!block.include_meta" class="summary-custom-data-hint">
                {{ t('admin.ruleConfig.summaryCustomDataHint') }}
              </div>

              <div class="field-mode-switch" style="margin-top: 12px;">
                <div
                  class="field-mode-option"
                  :class="{ 'field-mode-option--active': block.field_mode === 'all' }"
                  @click="block.field_mode = 'all'"
                >
                  <div class="field-mode-radio" />
                  <div>
                    <div class="field-mode-label">全部字段</div>
                    <div class="field-mode-desc">主表、明细和附件内容全部进入该块</div>
                  </div>
                </div>
                <div
                  class="field-mode-option"
                  :class="{ 'field-mode-option--active': block.field_mode === 'selected' }"
                  @click="block.field_mode = 'selected'"
                >
                  <div class="field-mode-radio" />
                  <div>
                    <div class="field-mode-label">指定字段</div>
                    <div class="field-mode-desc">仅使用弹窗中勾选的字段；附件字段被选中时带入识别文本</div>
                  </div>
                </div>
              </div>

              <div v-if="block.field_mode === 'selected'" class="summary-field-picker">
                <a-form-item label="选择字段" class="summary-field-select-item">
                  <a-select
                    v-model:value="block.selected_fields"
                    mode="multiple"
                    :options="summaryFieldOptions"
                    show-search
                    allow-clear
                    option-filter-prop="label"
                    placeholder="搜索并选择字段"
                    :max-tag-count="3"
                  />
                  <div class="field-count" style="margin-top: 6px; margin-bottom: 0;">
                    已选择 {{ block.selected_fields.length }} / {{ summaryAllAvailableFields.length }} 个字段
                  </div>
                </a-form-item>
              </div>

              <div class="summary-context-panel">
                <div class="summary-block-option-row">
                  <div class="summary-block-option-copy">
                    <div class="summary-block-option-title">外部关联数据</div>
                    <div class="summary-block-option-desc">本总结块生成前先查询关联流程或建模表，再把结果提供给 AI</div>
                  </div>
                  <div class="summary-context-switches">
                    <label class="context-switch">
                      <a-switch
                        :checked="!!getSummaryContextMount(block, 'workflow')"
                        @change="(checked: any) => toggleSummaryContextMount(block, 'workflow', !!checked)"
                      />
                      <span>关联流程</span>
                    </label>
                    <label class="context-switch">
                      <a-switch
                        :checked="!!getSummaryContextMount(block, 'model')"
                        @change="(checked: any) => toggleSummaryContextMount(block, 'model', !!checked)"
                      />
                      <span>关联建模表</span>
                    </label>
                  </div>
                </div>

                <div v-if="getSummaryContextMount(block, 'workflow')" class="summary-context-box">
                  <button
                    type="button"
                    class="summary-context-box-head"
                    @click="toggleSummaryContextExpanded(block, 'workflow')"
                  >
                    <span class="context-panel-title">关联流程</span>
                    <span v-if="!isSummaryContextExpanded(block, 'workflow')" class="summary-context-collapsed-hint">
                      {{ summaryContextCollapsedHint(block, 'workflow') }}
                    </span>
                    <DownOutlined class="summary-context-chevron" :class="{ 'summary-context-chevron--collapsed': !isSummaryContextExpanded(block, 'workflow') }" />
                  </button>
                  <div v-show="isSummaryContextExpanded(block, 'workflow')" class="summary-context-box-body">
                  <a-row :gutter="12">
                    <a-col :span="24">
                      <a-form-item label="来源字段">
                        <a-select
                          v-model:value="getSummaryContextMount(block, 'workflow')!.source_field"
                          :options="summaryContextFieldOptionsForBlock(block)"
                          show-search
                          option-filter-prop="label"
                          placeholder="选择已引入字段"
                        />
                      </a-form-item>
                    </a-col>
                  </a-row>
                  <a-form-item class="summary-basic-fields-item" :colon="false">
                    <template #label>
                      <div class="basic-fields-label-row">
                        <span>流程基础信息</span>
                        <a-checkbox
                          :checked="isSummaryWorkflowBasicAllSelected(block)"
                          :indeterminate="!!getSummaryContextMount(block, 'workflow')!.workflow!.basic_fields?.length && !isSummaryWorkflowBasicAllSelected(block)"
                          @change="(e: any) => toggleSummaryWorkflowBasicAll(block, !!e?.target?.checked)"
                        >
                          全选
                        </a-checkbox>
                      </div>
                    </template>
                    <div class="basic-fields-panel">
                      <a-checkbox-group
                        v-model:value="getSummaryContextMount(block, 'workflow')!.workflow!.basic_fields"
                        class="basic-fields-group"
                        @change="(vals: any) => {
                          const mount = getSummaryContextMount(block, 'workflow')
                          if (mount?.workflow) mount.workflow.include_basic = Array.isArray(vals) && vals.length > 0
                        }"
                      >
                        <label
                          v-for="opt in summaryWorkflowBasicOptions"
                          :key="opt.value"
                          class="basic-field-chip"
                          :class="{ active: getSummaryContextMount(block, 'workflow')!.workflow!.basic_fields?.includes(opt.value) }"
                        >
                          <a-checkbox :value="opt.value">{{ opt.label }}</a-checkbox>
                        </label>
                      </a-checkbox-group>
                      <div class="basic-fields-hint">
                        仅勾选的项会注入 AI；全部取消则不传基础信息
                      </div>
                    </div>
                  </a-form-item>
                  <a-form-item label="引用流程表单数据" class="summary-workflow-data-mode-item">
                    <div class="field-mode-switch summary-context-mode-switch">
                      <div
                        class="field-mode-option"
                        :class="{ 'field-mode-option--active': getSummaryContextMount(block, 'workflow')!.workflow?.data_mode !== 'selected_fields' }"
                        @click="setSummaryWorkflowTargetMode(block, false)"
                      >
                        <div class="field-mode-radio" />
                        <div class="field-mode-label">自动读取全部字段</div>
                      </div>
                      <div
                        class="field-mode-option"
                        :class="{ 'field-mode-option--active': getSummaryContextMount(block, 'workflow')!.workflow?.data_mode === 'selected_fields' }"
                        @click="setSummaryWorkflowTargetMode(block, true)"
                      >
                        <div class="field-mode-radio" />
                        <div class="field-mode-label">指定目标流程并选择字段</div>
                      </div>
                    </div>
                  </a-form-item>
                  <div v-if="getSummaryContextMount(block, 'workflow')!.workflow?.data_mode === 'selected_fields'" class="summary-workflow-target-compact">
                    <div class="summary-workflow-target-meta">
                      <div class="target-flow-label">目标流程引用</div>
                      <div class="target-flow-value">{{ summaryWorkflowConfigSummary(block) || '尚未配置' }}</div>
                    </div>
                    <a-button @click="openSummaryWorkflowConfigModal(block)">配置</a-button>
                  </div>
                  <div class="context-actions">
                    <a-button :loading="summaryContextTesting[summaryContextKey(block, 'workflow')]" @click="testSummaryContext(block, 'workflow')">测试关联流程</a-button>
                  </div>
                  <pre v-if="summaryContextPreviews[summaryContextKey(block, 'workflow')]" class="context-preview">{{ summaryContextPreviews[summaryContextKey(block, 'workflow')] }}</pre>
                  </div>
                </div>

                <div v-if="getSummaryContextMount(block, 'model')" class="summary-context-box">
                  <button
                    type="button"
                    class="summary-context-box-head"
                    @click="toggleSummaryContextExpanded(block, 'model')"
                  >
                    <span class="context-panel-title">关联建模表</span>
                    <span v-if="!isSummaryContextExpanded(block, 'model')" class="summary-context-collapsed-hint">
                      {{ summaryContextCollapsedHint(block, 'model') }}
                    </span>
                    <DownOutlined class="summary-context-chevron" :class="{ 'summary-context-chevron--collapsed': !isSummaryContextExpanded(block, 'model') }" />
                  </button>
                  <div v-show="isSummaryContextExpanded(block, 'model')" class="summary-context-box-body">
                  <div class="summary-model-config-grid">
                    <a-form-item label="来源字段">
                      <a-select
                        v-model:value="getSummaryContextMount(block, 'model')!.source_field"
                        :options="summaryContextFieldOptionsForBlock(block)"
                        show-search
                        placeholder="选择已引入字段"
                      />
                    </a-form-item>
                    <a-form-item label="建模表名">
                      <a-input v-model:value="getSummaryContextMount(block, 'model')!.model!.table_name" />
                    </a-form-item>
                    <a-form-item label="关联字段">
                      <a-input v-model:value="getSummaryContextMount(block, 'model')!.model!.join_field" placeholder="默认 id" />
                    </a-form-item>
                    <a-form-item label="查询方式">
                      <a-select v-model:value="getSummaryContextMount(block, 'model')!.model!.mode" show-search option-filter-prop="label">
                        <a-select-option value="exists">是否存在</a-select-option>
                        <a-select-option value="count">存在条数</a-select-option>
                        <a-select-option value="rows">返回匹配数据</a-select-option>
                        <a-select-option value="custom_sql">自定义 SQL</a-select-option>
                      </a-select>
                    </a-form-item>
                    <a-form-item v-if="getSummaryContextMount(block, 'model')!.model?.mode === 'rows'" class="summary-model-return-fields" :label="t('externalContext.modelReturnFields')">
                      <a-input
                        :value="getSummaryContextMount(block, 'model')!.model!.return_fields?.join(',')"
                        :placeholder="t('externalContext.modelReturnFieldsPlaceholder')"
                        @update:value="(v: string) => setSummaryModelReturnFields(block, v)"
                      />
                    </a-form-item>
                  </div>
                  <div v-if="getSummaryContextMount(block, 'model')!.model?.mode === 'rows'" class="summary-model-footer">
                    <a-form-item label="返回行数上限">
                      <a-space>
                        <a-input-number
                          :value="getSummaryContextMount(block, 'model')!.model!.max_rows === -1 ? undefined : getSummaryContextMount(block, 'model')!.model!.max_rows"
                          :min="1"
                          :max="50"
                          :disabled="getSummaryContextMount(block, 'model')!.model!.max_rows === -1"
                          style="width: 160px;"
                          @update:value="(value: number | null) => { const mount = getSummaryContextMount(block, 'model'); if (mount?.model && value) mount.model.max_rows = value }"
                        />
                        <a-checkbox :checked="getSummaryContextMount(block, 'model')!.model!.max_rows === -1" @change="(e: any) => setSummaryModelAllRows(block, !!e?.target?.checked)">
                          {{ t('common.all') }}
                        </a-checkbox>
                      </a-space>
                    </a-form-item>
                    <div class="context-actions summary-model-actions">
                      <a-button :loading="summaryContextTesting[summaryContextKey(block, 'model')]" @click="testSummaryContext(block, 'model')">测试建模查询</a-button>
                    </div>
                  </div>
                  <a-form-item v-if="getSummaryContextMount(block, 'model')!.model?.mode === 'custom_sql'" label="自定义 SQL" class="summary-model-sql-item">
                    <div class="sql-variable-bar">
                      <span>插入变量：</span>
                      <a-button size="small" @click="insertSummaryCustomSQLVariable(block, tableNameSQLVariable)">{{ tableNameSQLVariable }}</a-button>
                      <a-button size="small" @click="insertSummaryCustomSQLVariable(block, joinFieldSQLVariable)">{{ joinFieldSQLVariable }}</a-button>
                    </div>
                    <a-textarea v-model:value="getSummaryContextMount(block, 'model')!.model!.custom_sql" :rows="4" :placeholder="customSQLPlaceholder" />
                  </a-form-item>
                  <div v-if="getSummaryContextMount(block, 'model')!.model?.mode !== 'rows'" class="context-actions">
                    <a-button :loading="summaryContextTesting[summaryContextKey(block, 'model')]" @click="testSummaryContext(block, 'model')">测试建模查询</a-button>
                  </div>
                  <pre v-if="summaryContextPreviews[summaryContextKey(block, 'model')]" class="context-preview">{{ summaryContextPreviews[summaryContextKey(block, 'model')] }}</pre>
                  </div>
                </div>
              </div>

              <a-form layout="vertical" style="margin-top: 12px;">
                <a-form-item label="用户提示词">
                  <PromptVariableBar
                    :data-variables="summaryPromptDataVariables"
                    :system-variables="systemPromptVariables"
                    :disable-data-variables="block.include_meta"
                    :data-variable-mode="block.include_meta ? 'insert' : 'toggle'"
                    :selected-data-variables="block.enabled_data_variables || []"
                    @insert="insertSummaryBlockVariable(block, $event)"
                    @toggle-data-variable="toggleSummaryBlockDataVariable(block, $event)"
                  />
                  <PromptTextarea
                    :ref="(el) => setSummaryBlockTextareaRef(block.id, el)"
                    v-model:value="block.user_prompt"
                    :rows="4"
                    placeholder="输入该块的总结需求、判断重点或输出口径"
                    :dialog-title="`${block.title} · ${t('admin.ruleConfig.userPromptTag')}`"
                  />
                </a-form-item>
              </a-form>
            </div>
          </div>
        </div>

        <div v-if="summaryActiveTab === 'embed'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">OA 嵌入总结</h4>
              <p class="section-desc">控制 /embed/summary 的可见性和自动总结触发策略</p>
            </div>
          </div>
          <div class="permissions-list">
            <div class="permission-item">
              <div class="permission-info">
                <div class="permission-label">OA 嵌入总结</div>
                <div class="permission-desc">/embed/summary 使用该开关控制可见性</div>
              </div>
              <a-switch
                v-model:checked="selectedSummaryConfig.embed_enabled"
                checked-children="启用"
                un-checked-children="停用"
              />
            </div>
            <div class="permission-item">
              <div class="permission-info">
                <div class="permission-label">打开时自动总结</div>
                <div class="permission-desc">没有历史结果时自动发起总结</div>
              </div>
              <a-switch
                v-model:checked="selectedSummaryConfig.embed_config!.auto_summary_on_open"
                checked-children="启用"
                un-checked-children="停用"
              />
            </div>
            <div class="permission-item">
              <div class="permission-info">
                <div class="permission-label">流程变化后自动刷新</div>
                <div class="permission-desc">字段变化、节点变化，或流程被退回后重新提交时重新总结</div>
              </div>
              <a-switch
                v-model:checked="selectedSummaryConfig.embed_config!.auto_summary_on_stale"
                checked-children="启用"
                un-checked-children="停用"
              />
            </div>
          </div>
        </div>

        <div class="config-actions">
          <a-button type="primary" size="large" :disabled="savingSummary" @click="handleSaveSummaryConfig">
            <LoadingOutlined v-if="savingSummary" spin />
            <SaveOutlined v-else />
            保存总结配置
          </a-button>
        </div>
      </div>

      <div v-else class="config-empty">
        <a-empty description="请选择或新增总结流程" />
      </div>
    </div>

    <!--添加总结流程模态-->
    <a-modal
      v-model:open="showAddSummaryProcess"
      title="新增总结流程"
      @ok="handleAddSummaryProcess"
      ok-text="确认"
      cancel-text="取消"
      :width="640"
    >
      <div class="add-process-modal">
        <section class="add-process-section">
          <div class="add-process-section-head">
            <div class="add-process-section-title">从 OA 搜索并填充</div>
            <div class="add-process-section-desc">搜索后点选流程，自动填入下方信息</div>
          </div>
          <div class="workflow-picker-search">
            <a-input
              v-model:value="addSummaryProcessSearch.keyword"
              placeholder="输入流程名称、流程分类或主表名搜索"
              allow-clear
              @press-enter="searchAddProcessWorkflows(summaryApi.searchWorkflows, addSummaryProcessSearch)"
            />
            <a-button
              type="primary"
              :loading="addSummaryProcessSearch.searching"
              @click="searchAddProcessWorkflows(summaryApi.searchWorkflows, addSummaryProcessSearch)"
            >
              搜索
            </a-button>
          </div>
          <a-spin :spinning="addSummaryProcessSearch.searching">
            <div class="workflow-result-list workflow-result-list--add">
              <button
                v-for="item in addSummaryProcessSearch.rows"
                :key="workflowOptionKey(item)"
                type="button"
                class="workflow-result-item"
                :class="{ active: addSummaryProcessSearch.selectedId === workflowOptionKey(item) }"
                @click="selectAddProcessWorkflow(newSummaryProcessForm, addSummaryProcessSearch, item)"
              >
                <span class="workflow-result-radio"></span>
                <span class="workflow-result-main">
                  <span class="workflow-result-name">{{ workflowOptionName(item) }}</span>
                  <span class="workflow-result-meta">
                    <span v-if="item.process_type_label">{{ item.process_type_label }}</span>
                    <span v-if="item.main_table">{{ item.main_table }}</span>
                    <span v-if="item.workflow_id && item.workflow_id !== '0'">ID {{ item.workflow_id }}</span>
                  </span>
                </span>
              </button>
              <a-empty
                v-if="!addSummaryProcessSearch.rows.length && !addSummaryProcessSearch.searching"
                :description="addSummaryProcessSearch.hasSearched ? '未找到匹配流程' : '输入关键词后点击搜索'"
              />
            </div>
          </a-spin>
        </section>

        <div class="add-process-divider">
          <span>或手动填写</span>
        </div>

        <section class="add-process-section">
          <div class="add-process-section-head">
            <div class="add-process-section-title">流程信息</div>
            <div class="add-process-section-desc">可直接编辑，也可点测试连接校验</div>
          </div>
          <a-form layout="vertical">
            <a-form-item label="流程名称" required>
              <a-input v-model:value="newSummaryProcessForm.process_type" placeholder="请输入 OA 流程名称" />
            </a-form-item>
            <a-form-item label="流程分类">
              <a-input v-model:value="newSummaryProcessForm.process_type_label" placeholder="可选，OA 流程分类名称" />
            </a-form-item>
            <a-form-item label="主表名称">
              <div style="display: flex; gap: 8px;">
                <a-input v-model:value="newSummaryProcessForm.main_table_name" placeholder="可选，测试连接后自动填充" style="flex: 1;" />
                <a-button
                  :loading="summaryTestingConnection"
                  @click="handleSummaryTestConnectionInModal"
                  :disabled="!newSummaryProcessForm.process_type.trim()"
                >
                  <template #icon><DatabaseOutlined /></template>
                  {{ summaryTestingConnection ? '测试中' : '测试连接' }}
                </a-button>
              </div>
            </a-form-item>
          </a-form>
        </section>
      </div>
    </a-modal>

    <!--总结块字段选择器模态-->
    <a-modal
      v-model:open="showSummaryFieldPicker"
      :title="editingSummaryBlock ? `选择字段：${editingSummaryBlock.title || '未命名总结块'}` : '选择字段'"
      :width="720"
      :footer="null"
      @cancel="showSummaryFieldPicker = false"
    >
      <div class="field-picker-modal">
        <div class="field-picker-left">
          <div class="field-picker-panel-header" style="justify-content: flex-start; gap: 8px;">
            <a-checkbox
              :checked="summaryLeftSelectedKeys.length === summaryUnselectedFieldsFlat.length && summaryUnselectedFieldsFlat.length > 0"
              :indeterminate="summaryLeftSelectedKeys.length > 0 && summaryLeftSelectedKeys.length < summaryUnselectedFieldsFlat.length"
              @change="toggleSummaryLeftSelectAll"
            />
            <span style="flex: 1;">{{ t('admin.ruleConfig.availableFields') }} <span class="field-count" style="margin-left:4px; font-weight:normal;">({{ summaryUnselectedFieldsFlat.length }})</span></span>
            <a-button type="primary" size="small" :disabled="summaryLeftSelectedKeys.length === 0" @click="summaryBatchPick">
              {{ t('admin.ruleConfig.add') }}
            </a-button>
          </div>
          <div class="field-picker-search">
            <a-input
              v-model:value="summaryFieldSearchQuery"
              :placeholder="t('admin.ruleConfig.searchFieldPlaceholder')"
              allow-clear
              size="small"
            >
              <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
            </a-input>
          </div>
          <div class="field-picker-list" style="padding: 12px 16px;">
            <div
              v-for="field in summaryUnselectedPagination.paged.value"
              :key="field.value"
              class="field-picker-item"
              @click="toggleSummaryLeftSelect(field.value)"
              style="display: flex; gap: 12px; justify-content: flex-start; margin-bottom: 8px;"
            >
              <div class="field-picker-item-checkbox" @click.stop="toggleSummaryLeftSelect(field.value)">
                <a-checkbox :checked="summaryLeftSelectedKeys.includes(field.value)" />
              </div>
              <div class="field-picker-item-info" style="flex: 1;">
                <div class="field-picker-item-name">{{ field.field_name }} <span class="field-source-tag" style="font-size: 11px; color: var(--color-text-tertiary); font-weight: normal; margin-left: 4px;">({{ field.sourceLabel }})</span></div>
                <div class="field-picker-item-meta">
                  <span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span>
                  <span class="field-key">{{ field.field_key }}</span>
                </div>
              </div>
              <button class="icon-btn icon-btn--sm" @click.stop="pickSummaryField(field)" style="margin-left: auto;">
                <SwapRightOutlined />
              </button>
            </div>
            <div v-if="!summaryUnselectedFieldsFlat.length" class="field-picker-empty">
              {{ summaryFieldSearchQuery ? t('admin.ruleConfig.noSearchResult') : t('admin.ruleConfig.allFieldsAdded') }}
            </div>
          </div>
          <div class="pagination-wrapper" style="padding: 12px 16px; border-top: 1px solid var(--color-border-light);">
            <a-pagination
              v-model:current="summaryUnselectedPagination.current.value"
              v-model:page-size="summaryUnselectedPagination.pageSize.value"
              :total="summaryUnselectedPagination.total.value"
              size="small"
              show-size-changer
              :page-size-options="['5', '20', '50']"
              @change="summaryUnselectedPagination.onChange"
              @showSizeChange="summaryUnselectedPagination.onChange"
            />
          </div>
        </div>
        <div class="field-picker-right">
          <div class="field-picker-panel-header" style="justify-content: flex-start; gap: 8px;">
            <a-checkbox
              :checked="summaryRightSelectedKeys.length === summarySelectedFieldsFlat.length && summarySelectedFieldsFlat.length > 0"
              :indeterminate="summaryRightSelectedKeys.length > 0 && summaryRightSelectedKeys.length < summarySelectedFieldsFlat.length"
              @change="toggleSummaryRightSelectAll"
            />
            <span style="flex: 1;">{{ t('admin.ruleConfig.selectedFields') }} <span class="field-picker-count" style="margin-left:4px;">{{ summarySelectedFieldsFlat.length }}</span></span>
            <a-button danger size="small" :disabled="summaryRightSelectedKeys.length === 0" @click="summaryBatchUnpick">
              {{ t('admin.ruleConfig.remove') }}
            </a-button>
          </div>
          <div class="field-picker-search">
            <a-input
              v-model:value="summarySelectedFieldSearchQuery"
              :placeholder="t('admin.ruleConfig.searchFieldPlaceholder')"
              allow-clear
              size="small"
            >
              <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
            </a-input>
          </div>
          <div class="field-picker-list" style="padding: 12px 16px;">
            <div
              v-for="field in summarySelectedPagination.paged.value"
              :key="field.value"
              class="field-picker-item field-picker-item--selected"
              @click="toggleSummaryRightSelect(field.value)"
              style="display: flex; gap: 12px; justify-content: flex-start; margin-bottom: 8px;"
            >
              <div class="field-picker-item-checkbox" @click.stop="toggleSummaryRightSelect(field.value)">
                <a-checkbox :checked="summaryRightSelectedKeys.includes(field.value)" />
              </div>
              <div class="field-picker-item-info" style="flex: 1;">
                <div class="field-picker-item-name">{{ field.field_name }} <span class="field-source-tag" style="font-size: 11px; color: var(--color-text-tertiary); font-weight: normal; margin-left: 4px;">({{ field.sourceLabel }})</span></div>
                <div class="field-picker-item-meta">
                  <span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span>
                  <span class="field-key">{{ field.field_key }}</span>
                </div>
              </div>
              <button class="field-picker-remove" @click.stop="unpickSummaryField(field)" style="margin-left: auto;">
                <CloseOutlined />
              </button>
            </div>
            <div v-if="!summarySelectedFieldsFlat.length" class="field-picker-empty">
              {{ summarySelectedFieldSearchQuery ? t('admin.ruleConfig.noSearchResult') : t('admin.ruleConfig.noFieldsSelected') }}
            </div>
          </div>
          <div class="pagination-wrapper" style="padding: 12px 16px; border-top: 1px solid var(--color-border-light);">
            <a-pagination
              v-model:current="summarySelectedPagination.current.value"
              v-model:page-size="summarySelectedPagination.pageSize.value"
              :total="summarySelectedPagination.total.value"
              size="small"
              show-size-changer
              :page-size-options="['5', '20', '50']"
              @change="summarySelectedPagination.onChange"
              @showSizeChange="summarySelectedPagination.onChange"
            />
          </div>
        </div>
      </div>
    </a-modal>

    <a-modal
      v-model:open="summaryWorkflowConfigOpen"
      title="配置目标流程引用"
      :width="720"
      ok-text="确定"
      cancel-text="取消"
      :confirm-loading="currentSummaryWorkflowBlock ? !!summaryWorkflowFieldLoading[currentSummaryWorkflowBlock.id] : false"
      @ok="confirmSummaryWorkflowConfig"
    >
      <div v-if="currentSummaryWorkflowBlock" class="workflow-config-modal">
        <div
          v-if="summaryTargetWorkflowSummary(currentSummaryWorkflowBlock) && !summaryWorkflowRows.length"
          class="summary-workflow-current"
        >
          <div class="target-flow-label">当前已选流程</div>
          <div class="target-flow-value">{{ summaryTargetWorkflowSummary(currentSummaryWorkflowBlock) }}</div>
        </div>
        <div class="workflow-picker">
          <div class="workflow-picker-search">
            <a-input
              v-model:value="summaryWorkflowKeyword"
              placeholder="输入流程名称、流程分类或主表名搜索"
              allow-clear
              @press-enter="searchSummaryWorkflows"
            />
            <a-button type="primary" :loading="summaryWorkflowSearching" @click="searchSummaryWorkflows">
              搜索
            </a-button>
          </div>
          <a-spin :spinning="summaryWorkflowSearching">
            <div class="workflow-result-list workflow-result-list--modal">
              <button
                v-for="item in summaryWorkflowRows"
                :key="item.workflow_id || item.process_name || item.process_type"
                type="button"
                class="workflow-result-item"
                :class="{ active: summaryWorkflowSelectedID === (item.workflow_id || item.process_name || item.process_type) }"
                @click="selectSummaryWorkflowInModal(item)"
              >
                <span class="workflow-result-radio"></span>
                <span class="workflow-result-main">
                  <span class="workflow-result-name">{{ summaryWorkflowDisplayName(item) }}</span>
                  <span class="workflow-result-meta">
                    <span v-if="item.process_type_label">{{ item.process_type_label }}</span>
                    <span v-if="item.main_table">{{ item.main_table }}</span>
                    <span v-if="item.workflow_id && item.workflow_id !== '0'">ID {{ item.workflow_id }}</span>
                  </span>
                </span>
              </button>
              <a-empty
                v-if="!summaryWorkflowRows.length && !summaryWorkflowSearching"
                :description="summaryWorkflowHasSearched ? '未找到匹配流程' : '输入关键词后点击搜索'"
              />
            </div>
          </a-spin>
        </div>
        <a-form layout="vertical" class="workflow-config-form">
          <a-form-item label="引用流程字段">
            <a-select
              v-model:value="summaryWorkflowDraftFields"
              :options="summaryWorkflowFieldOptions[currentSummaryWorkflowBlock.id] || []"
              mode="multiple"
              show-search
              option-filter-prop="label"
              placeholder="先选择目标流程，再勾选要提供给 AI 的字段"
            />
          </a-form-item>
          <a-form-item label="流程类型不一致时">
            <a-select v-model:value="getSummaryContextMount(currentSummaryWorkflowBlock, 'workflow')!.workflow!.fallback_strategy">
              <a-select-option value="basic_with_notice">仅提供基础信息并提示模型类型不一致</a-select-option>
              <a-select-option value="all_fields">尝试读取实际流程全部字段</a-select-option>
              <a-select-option value="ignore">忽略该引用流程</a-select-option>
            </a-select>
          </a-form-item>
        </a-form>
      </div>
    </a-modal>

    <!--规则编辑器模式-->
    <RuleEditor
      :open="showRuleEditor"
      :rule="editingRule"
      :field-options="auditContextFieldOptions"
      context-test-endpoint="/api/tenant/rules/context/test"
      workflow-fields-endpoint="/api/tenant/rules/context/workflow-fields"
      workflow-search-endpoint="/api/tenant/rules/context/workflow-search"
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
      :width="640"
    >
      <div class="add-process-modal">
        <section class="add-process-section">
          <div class="add-process-section-head">
            <div class="add-process-section-title">从 OA 搜索并填充</div>
            <div class="add-process-section-desc">搜索后点选流程，自动填入下方信息</div>
          </div>
          <div class="workflow-picker-search">
            <a-input
              v-model:value="addProcessSearch.keyword"
              placeholder="输入流程名称、流程分类或主表名搜索"
              allow-clear
              @press-enter="searchAddProcessWorkflows(rulesApi.searchWorkflows, addProcessSearch)"
            />
            <a-button
              type="primary"
              :loading="addProcessSearch.searching"
              @click="searchAddProcessWorkflows(rulesApi.searchWorkflows, addProcessSearch)"
            >
              搜索
            </a-button>
          </div>
          <a-spin :spinning="addProcessSearch.searching">
            <div class="workflow-result-list workflow-result-list--add">
              <button
                v-for="item in addProcessSearch.rows"
                :key="workflowOptionKey(item)"
                type="button"
                class="workflow-result-item"
                :class="{ active: addProcessSearch.selectedId === workflowOptionKey(item) }"
                @click="selectAddProcessWorkflow(newProcessForm, addProcessSearch, item)"
              >
                <span class="workflow-result-radio"></span>
                <span class="workflow-result-main">
                  <span class="workflow-result-name">{{ workflowOptionName(item) }}</span>
                  <span class="workflow-result-meta">
                    <span v-if="item.process_type_label">{{ item.process_type_label }}</span>
                    <span v-if="item.main_table">{{ item.main_table }}</span>
                    <span v-if="item.workflow_id && item.workflow_id !== '0'">ID {{ item.workflow_id }}</span>
                  </span>
                </span>
              </button>
              <a-empty
                v-if="!addProcessSearch.rows.length && !addProcessSearch.searching"
                :description="addProcessSearch.hasSearched ? '未找到匹配流程' : '输入关键词后点击搜索'"
              />
            </div>
          </a-spin>
        </section>

        <div class="add-process-divider">
          <span>或手动填写</span>
        </div>

        <section class="add-process-section">
          <div class="add-process-section-head">
            <div class="add-process-section-title">流程信息</div>
            <div class="add-process-section-desc">可直接编辑，也可点测试连接校验</div>
          </div>
          <a-form layout="vertical">
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
            </a-form-item>
          </a-form>
        </section>
      </div>
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
        <div v-if="selectedCronConfig?.task_type === 'audit_batch' || selectedCronConfig?.task_type === 'archive_batch'" class="tab-content">
          <div class="section-header">
            <div>
              <h4 class="section-title">{{ t('admin.ruleConfig.batchAuditConfigTitle') }}</h4>
              <p class="section-desc">{{ t('admin.ruleConfig.batchAuditConfigDesc') }}</p>
            </div>
          </div>
            <div class="ai-form-group">
              <label class="ai-form-label">{{ t('admin.ruleConfig.batchLimitLabel') }}</label>
              <a-input-number
                v-model:value="selectedCronConfig!.batch_limit"
                :min="1"
                :max="50"
                size="large"
                style="width: 200px;"
              />
              <p class="section-desc" style="margin-top: 4px;">{{ t('admin.ruleConfig.batchLimitDesc') }}</p>
            </div>
        </div>

        <!--========== daily / weekly：带有变量插入的内容模板==========-->
        <div v-if="selectedCronConfig?.task_type !== 'audit_batch' && selectedCronConfig?.task_type !== 'archive_batch'" class="tab-content">
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
                :class="{ 'push-format-option--active': selectedCronConfig!.push_format === fmt.value }"
                @click="selectedCronConfig!.push_format = fmt.value as any"
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
                v-model:value="selectedCronConfig!.content_template.subject"
                size="large"
                :placeholder="t('admin.ruleConfig.emailSubjectPlaceholder')"
                @focus="cronActiveField = 'subject'"
              />
            </div>
            <div class="ai-form-group">
              <label class="ai-form-label">{{ t('admin.ruleConfig.emailHeaderLabel') }}</label>
              <a-input
                ref="cronHeaderRef"
                v-model:value="selectedCronConfig!.content_template.header"
                size="large"
                :placeholder="t('admin.ruleConfig.emailHeaderPlaceholder')"
                @focus="cronActiveField = 'header'"
              />
            </div>
            <div class="ai-form-group">
              <label class="ai-form-label">{{ t('admin.ruleConfig.emailBodyLabel') }}</label>
              <a-textarea
                ref="cronBodyRef"
                v-model:value="selectedCronConfig!.content_template.body_template"
                :rows="6"
                :placeholder="t('admin.ruleConfig.emailBodyPlaceholder')"
                @focus="cronActiveField = 'body_template'"
              />
            </div>
            <div class="ai-form-group">
              <label class="ai-form-label">{{ t('admin.ruleConfig.emailFooterLabel') }}</label>
              <a-input
                ref="cronFooterRef"
                v-model:value="selectedCronConfig!.content_template.footer"
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
          <div style="flex: 1; min-width: 0;">
            <div class="process-nav-name">{{ cfg.process_type }}</div>
            <div v-if="cfg.process_type_label" class="process-nav-path">{{ cfg.process_type_label }}</div>
          </div>
          <a-popconfirm :title="t('admin.ruleConfig.deleteConfigConfirm')" @confirm.stop="handleDeleteArchiveProcess(cfg.id)" placement="right">
            <button class="icon-btn icon-btn--danger icon-btn--sm" @click.stop style="opacity: 0.5; flex-shrink: 0;">
              <DeleteOutlined />
            </button>
          </a-popconfirm>
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
            <a-button :loading="syncingArchiveFields" @click="handleArchiveSyncFields">
              <template #icon><DatabaseOutlined /></template>
              {{ syncingArchiveFields ? t('admin.ruleConfig.syncingFields') : t('admin.ruleConfig.syncFields') }}
            </a-button>
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

            <div v-if="archivePageSelectedFieldsFlat.length > 0 || archivePageSelectedFieldSearchQuery" class="page-selected-fields-container" style="margin-top: 16px;">
              <div style="margin-bottom: 12px; max-width: 300px;">
                <a-input
                  v-model:value="archivePageSelectedFieldSearchQuery"
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
                    <tr v-for="field in archivePageSelectedPagination.paged.value" :key="field.field_key + field.source">
                      <td style="padding-left: 24px; font-weight: 500;">{{ field.field_name }}</td>
                      <td class="text-mono" style="font-size: 13px;">{{ field.field_key }}</td>
                      <td><span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span></td>
                      <td class="text-secondary" style="font-size: 13px;">{{ field.sourceLabel }}</td>
                    </tr>
                    <tr v-if="archivePageSelectedPagination.paged.value.length === 0">
                      <td colspan="4" class="empty-cell">{{ t('admin.ruleConfig.noSearchResult') || '未找到匹配字段' }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
              <div class="pagination-wrapper" style="margin-top: 12px; text-align: right;">
                <a-pagination
                  v-model:current="archivePageSelectedPagination.current.value"
                  v-model:page-size="archivePageSelectedPagination.pageSize.value"
                  :total="archivePageSelectedPagination.total.value"
                  size="small"
                  show-size-changer
                  show-quick-jumper
                  :page-size-options="['5', '20', '50']"
                  @change="archivePageSelectedPagination.onChange"
                  @showSizeChange="archivePageSelectedPagination.onChange"
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
            <div class="rules-toolbar-summary">
              <a-checkbox
                :checked="allArchiveRulesSelected"
                :indeterminate="archiveRulesSelectionIndeterminate"
                :disabled="currentArchiveRules.length === 0 || batchDeletingArchiveRules"
                @change="(event: any) => toggleAllArchiveRules(!!event.target.checked)"
              >
                {{ t('admin.ruleConfig.selectAll') }}
              </a-checkbox>
              <span class="rules-count">{{ t('admin.ruleConfig.totalRules', `${currentArchiveRules.length}`) }}</span>
              <span v-if="selectedArchiveRuleIds.length" class="rules-selected-count">
                {{ t('admin.ruleConfig.selectedRules', `${selectedArchiveRuleIds.length}`) }}
              </span>
            </div>
            <div class="rules-toolbar-actions">
              <template v-if="selectedArchiveRuleIds.length">
                <a-popconfirm
                  :title="t('admin.ruleConfig.batchDeleteConfirm', `${selectedArchiveRuleIds.length}`)"
                  @confirm="batchDeleteArchiveRules"
                >
                  <a-button danger :loading="batchDeletingArchiveRules">
                    <DeleteOutlined /> {{ t('admin.ruleConfig.batchDelete') }}
                  </a-button>
                </a-popconfirm>
              </template>
              <template v-else>
                <a-button :disabled="ruleImportLoading" @click="handlePasteImport('archive')">
                  <SnippetsOutlined /> {{ t('admin.ruleConfig.pasteImport') }}
                </a-button>
                <a-button
                  :loading="ruleImportLoading && ruleImportTarget === 'archive'"
                  :disabled="ruleImportLoading || !ruleImportCapability?.enabled"
                  :title="ruleImportCapability?.reason || ''"
                  @click="handleImportRules('archive')"
                >
                  <UploadOutlined /> {{ t('admin.ruleConfig.fileImport') }}
                </a-button>
                <a-button type="primary" @click="openArchiveRuleEditor()">
                  <PlusOutlined /> {{ t('admin.ruleConfig.manualAdd') }}
                </a-button>
              </template>
            </div>
          </div>

          <div class="rules-list">
            <div v-if="loadingArchiveRules" class="rules-loading">
              <a-spin size="small" />
            </div>
            <div v-for="rule in currentArchiveRules" :key="rule.id" class="rule-card">
              <div class="rule-card-left">
                <a-checkbox
                  class="rule-select-checkbox"
                  :checked="selectedArchiveRuleIdSet.has(rule.id)"
                  :disabled="batchDeletingArchiveRules"
                  :aria-label="t('admin.ruleConfig.selectRule')"
                  @change="(event: any) => toggleArchiveRuleSelection(rule.id, !!event.target.checked)"
                />
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
                    <span v-if="hasActualExternalContext(rule)" class="rule-flow-tag">
                      <NodeIndexOutlined /> 外部关联
                    </span>
                    <span v-if="rule.source === 'file_import'" class="rule-source-tag">{{ t('admin.ruleConfig.fileImportTag') }}</span>
                    <span v-else-if="rule.source === 'paste_import'" class="rule-source-tag">{{ t('admin.ruleConfig.pasteImportTag') }}</span>
                    <span v-else class="rule-source-tag rule-source-tag--manual">{{ t('admin.ruleConfig.manualAddTag') }}</span>
                  </div>
                </div>
              </div>
              <div class="rule-card-actions">
                <a-switch
                  :checked="rule.enabled"
                  :disabled="rule.rule_scope === 'mandatory'"
                  size="small"
                  @change="(checked: any) => { archiveApi.updateRule(rule.id, { enabled: !!checked }).then(updated => { const idx = currentArchiveRules.findIndex(r => r.id === rule.id); if (idx >= 0) currentArchiveRules[idx] = updated }) }"
                />
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
                <PromptTextarea
                  v-model:value="selectedArchiveConfig.ai_config.system_reasoning_prompt"
                  :rows="6"
                  :placeholder="t('admin.ruleConfig.systemReasoningPlaceholder')"
                  :dialog-title="t('admin.ruleConfig.systemReasoningPrompt')"
                />
              </div>

              <div class="ai-form-group">
                <div class="prompt-section-header">
                  <div class="prompt-section-title">
                    <span class="prompt-phase-badge prompt-phase-badge--extraction">{{ t('admin.ruleConfig.phase2Label') }}</span>
                    <label class="ai-form-label">{{ t('admin.ruleConfig.systemExtractionPrompt') }}</label>
                  </div>
                </div>
                <PromptTextarea
                  v-model:value="selectedArchiveConfig.ai_config.system_extraction_prompt"
                  :rows="6"
                  :placeholder="t('admin.ruleConfig.systemExtractionPlaceholder')"
                  :dialog-title="t('admin.ruleConfig.systemExtractionPrompt')"
                  disabled
                />
                <div class="system-prompt-readonly-hint">{{ t('admin.ruleConfig.systemPromptReadonly') }}</div>
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
                <PromptVariableBar
                  :data-variables="archiveReasoningPromptVariables"
                  :system-variables="systemPromptVariables"
                  @insert="insertArchiveReasoningVariable"
                />
                <PromptTextarea
                  ref="archiveReasoningTextareaRef"
                  v-model:value="selectedArchiveConfig.ai_config.user_reasoning_prompt"
                  :rows="8"
                  :placeholder="t('admin.ruleConfig.userReasoningPlaceholder')"
                  :dialog-title="t('admin.ruleConfig.userReasoningPrompt')"
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
                <PromptVariableBar
                  :data-variables="archiveExtractionPromptVariables"
                  :system-variables="systemPromptVariables"
                  @insert="insertArchiveExtractionVariable"
                />
                <PromptTextarea
                  ref="archiveExtractionTextareaRef"
                  v-model:value="selectedArchiveConfig.ai_config.user_extraction_prompt"
                  :rows="6"
                  :placeholder="t('admin.ruleConfig.userExtractionPlaceholder')"
                  :dialog-title="t('admin.ruleConfig.userExtractionPrompt')"
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

        <div v-if="archiveActiveTab !== 'rules'" class="config-actions">
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
      :field-options="archiveContextFieldOptions"
      context-test-endpoint="/api/tenant/archive/context/test"
      workflow-fields-endpoint="/api/tenant/archive/context/workflow-fields"
      workflow-search-endpoint="/api/tenant/archive/context/workflow-search"
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
      :width="640"
    >
      <div class="add-process-modal">
        <section class="add-process-section">
          <div class="add-process-section-head">
            <div class="add-process-section-title">从 OA 搜索并填充</div>
            <div class="add-process-section-desc">搜索后点选流程，自动填入下方信息</div>
          </div>
          <div class="workflow-picker-search">
            <a-input
              v-model:value="addArchiveProcessSearch.keyword"
              placeholder="输入流程名称、流程分类或主表名搜索"
              allow-clear
              @press-enter="searchAddProcessWorkflows(archiveApi.searchWorkflows, addArchiveProcessSearch)"
            />
            <a-button
              type="primary"
              :loading="addArchiveProcessSearch.searching"
              @click="searchAddProcessWorkflows(archiveApi.searchWorkflows, addArchiveProcessSearch)"
            >
              搜索
            </a-button>
          </div>
          <a-spin :spinning="addArchiveProcessSearch.searching">
            <div class="workflow-result-list workflow-result-list--add">
              <button
                v-for="item in addArchiveProcessSearch.rows"
                :key="workflowOptionKey(item)"
                type="button"
                class="workflow-result-item"
                :class="{ active: addArchiveProcessSearch.selectedId === workflowOptionKey(item) }"
                @click="selectAddProcessWorkflow(newArchiveProcessForm, addArchiveProcessSearch, item)"
              >
                <span class="workflow-result-radio"></span>
                <span class="workflow-result-main">
                  <span class="workflow-result-name">{{ workflowOptionName(item) }}</span>
                  <span class="workflow-result-meta">
                    <span v-if="item.process_type_label">{{ item.process_type_label }}</span>
                    <span v-if="item.main_table">{{ item.main_table }}</span>
                    <span v-if="item.workflow_id && item.workflow_id !== '0'">ID {{ item.workflow_id }}</span>
                  </span>
                </span>
              </button>
              <a-empty
                v-if="!addArchiveProcessSearch.rows.length && !addArchiveProcessSearch.searching"
                :description="addArchiveProcessSearch.hasSearched ? '未找到匹配流程' : '输入关键词后点击搜索'"
              />
            </div>
          </a-spin>
        </section>

        <div class="add-process-divider">
          <span>或手动填写</span>
        </div>

        <section class="add-process-section">
          <div class="add-process-section-head">
            <div class="add-process-section-title">流程信息</div>
            <div class="add-process-section-desc">可直接编辑，也可点测试连接校验</div>
          </div>
          <a-form layout="vertical">
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
            </a-form-item>
          </a-form>
        </section>
      </div>
    </a-modal>

    <input
      ref="ruleImportFileInput"
      type="file"
      :accept="ruleImportAccept"
      class="rule-import-file-input"
      @change="handleRuleImportFileSelected"
    />

    <!--粘贴导入：文本直接交给 AI，不依赖 MinerU。-->
    <a-modal
      v-model:open="showPasteRuleImport"
      :title="t('admin.ruleConfig.pasteImportTitle')"
      :width="760"
      :ok-text="t('admin.ruleConfig.pasteImportAnalyze')"
      :cancel-text="t('admin.ruleConfig.cancel')"
      :confirm-loading="ruleImportLoading"
      :ok-button-props="{ disabled: !pastedRuleImportText.trim() }"
      @ok="analyzePastedRuleImport"
    >
      <p class="rule-import-help rule-import-paste-help">{{ t('admin.ruleConfig.pasteImportHelp') }}</p>
      <a-textarea
        v-model:value="pastedRuleImportText"
        :placeholder="t('admin.ruleConfig.pasteImportPlaceholder')"
        :rows="14"
        :maxlength="120000"
        show-count
      />
    </a-modal>

    <!--规则文件识别导入预览：审核与归档共用，确认后才批量写库-->
    <a-modal
      v-model:open="showRuleImportPreview"
      :title="t('admin.ruleConfig.fileImportPreviewTitle')"
      :width="900"
      :ok-text="t('admin.ruleConfig.fileImportConfirm', `${selectedRuleImportCount}`)"
      :cancel-text="t('admin.ruleConfig.cancel')"
      :confirm-loading="ruleImportSaving"
      :ok-button-props="{ disabled: selectedRuleImportCount === 0 }"
      @ok="confirmRuleImport"
    >
      <div class="rule-import-summary">
        <div>
          <div class="rule-import-file-name"><FileTextOutlined /> {{ ruleImportFileName }}</div>
          <div class="rule-import-help">{{ t('admin.ruleConfig.fileImportPreviewHelp') }}</div>
        </div>
        <span class="rule-import-count">{{ t('admin.ruleConfig.fileImportSelectedCount', [`${selectedRuleImportCount}`, `${ruleImportDrafts.length}`]) }}</span>
      </div>

      <a-alert
        v-for="warning in ruleImportWarnings"
        :key="warning"
        type="warning"
        show-icon
        :message="warning"
        class="rule-import-warning"
      />

      <div class="rule-import-list">
        <div v-for="(rule, index) in ruleImportDrafts" :key="index" class="rule-import-item" :class="{ 'rule-import-item--disabled': !rule.selected }">
          <div class="rule-import-item-head">
            <a-checkbox v-model:checked="rule.selected">{{ t('admin.ruleConfig.fileImportRuleIndex', `${index + 1}`) }}</a-checkbox>
            <span class="rule-import-confidence">{{ t('admin.ruleConfig.fileImportConfidence', `${Math.round(rule.confidence * 100)}`) }}</span>
          </div>
          <a-textarea v-model:value="rule.rule_content" :rows="3" :disabled="!rule.selected" />
          <div class="rule-import-options">
            <label>
              <span>{{ t('admin.ruleConfig.ruleLevel') }}</span>
              <a-select v-model:value="rule.rule_scope" :disabled="!rule.selected" style="width: 140px;">
                <a-select-option value="mandatory">{{ t('admin.ruleConfig.mandatory') }}</a-select-option>
                <a-select-option value="default_on">{{ t('admin.ruleConfig.defaultOn') }}</a-select-option>
                <a-select-option value="default_off">{{ t('admin.ruleConfig.defaultOff') }}</a-select-option>
              </a-select>
            </label>
            <label><a-switch v-model:checked="rule.related_flow" :disabled="!rule.selected" size="small" /> {{ t('admin.ruleConfig.relatedFlow') }}</label>
            <label><a-switch v-model:checked="rule.context_recommended" :disabled="!rule.selected" size="small" /> {{ t('admin.ruleConfig.externalContextRecommendation') }}</label>
          </div>
          <div v-if="rule.reasoning" class="rule-import-reasoning">
            <RobotOutlined /> {{ rule.reasoning }}
          </div>
          <div v-if="rule.context_recommended" class="rule-import-context-hint">
            {{ t('admin.ruleConfig.fileImportContextHint') }}
          </div>
        </div>
      </div>
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
          <div class="field-picker-panel-header" style="justify-content: flex-start; gap: 8px;">
            <a-checkbox
              :checked="archiveLeftSelectedKeys.length === archiveUnselectedFieldsFlat.length && archiveUnselectedFieldsFlat.length > 0"
              :indeterminate="archiveLeftSelectedKeys.length > 0 && archiveLeftSelectedKeys.length < archiveUnselectedFieldsFlat.length"
              @change="toggleArchiveLeftSelectAll"
            />
            <span style="flex: 1;">{{ t('admin.ruleConfig.availableFields') }} <span class="field-count" style="margin-left:4px; font-weight:normal;">({{ archiveUnselectedFieldsFlat.length }})</span></span>
            <a-button type="primary" size="small" :disabled="archiveLeftSelectedKeys.length === 0" @click="archiveBatchPick">
              {{ t('admin.ruleConfig.add') }}
            </a-button>
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
          <div class="field-picker-list" style="padding: 12px 16px;">
            <div
              v-for="field in archiveUnselectedPagination.paged.value"
              :key="field.field_key + field.source"
              class="field-picker-item"
              @click="toggleArchiveLeftSelect(field.field_key + '_' + field.source)"
              style="display: flex; gap: 12px; justify-content: flex-start; margin-bottom: 8px;"
            >
              <div class="field-picker-item-checkbox" @click.stop="toggleArchiveLeftSelect(field.field_key + '_' + field.source)">
                <a-checkbox :checked="archiveLeftSelectedKeys.includes(field.field_key + '_' + field.source)" />
              </div>
              <div class="field-picker-item-info" style="flex: 1;">
                <div class="field-picker-item-name">{{ field.field_name }} <span class="field-source-tag" style="font-size: 11px; color: var(--color-text-tertiary); font-weight: normal; margin-left: 4px;">({{ field.sourceLabel }})</span></div>
                <div class="field-picker-item-meta">
                  <span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span>
                  <span class="field-key">{{ field.field_key }}</span>
                </div>
              </div>
              <button class="icon-btn icon-btn--sm" @click.stop="archivePickField(field)" style="margin-left: auto;">
                <SwapRightOutlined />
              </button>
            </div>
            <div v-if="!archiveUnselectedFieldsFlat.length" class="field-picker-empty">
              {{ archiveFieldSearchQuery ? t('admin.ruleConfig.noSearchResult') : t('admin.ruleConfig.allFieldsAdded') }}
            </div>
          </div>
          <div class="pagination-wrapper" style="padding: 12px 16px; border-top: 1px solid var(--color-border-light);">
            <a-pagination
              v-model:current="archiveUnselectedPagination.current.value"
              v-model:page-size="archiveUnselectedPagination.pageSize.value"
              :total="archiveUnselectedPagination.total.value"
              size="small"
              show-size-changer
              :page-size-options="['5', '20', '50']"
              @change="archiveUnselectedPagination.onChange"
              @showSizeChange="archiveUnselectedPagination.onChange"
            />
          </div>
        </div>
        <div class="field-picker-right">
          <div class="field-picker-panel-header" style="justify-content: flex-start; gap: 8px;">
            <a-checkbox
              :checked="archiveRightSelectedKeys.length === archiveSelectedFieldsFlat.length && archiveSelectedFieldsFlat.length > 0"
              :indeterminate="archiveRightSelectedKeys.length > 0 && archiveRightSelectedKeys.length < archiveSelectedFieldsFlat.length"
              @change="toggleArchiveRightSelectAll"
            />
            <span style="flex: 1;">{{ t('admin.ruleConfig.selectedFields') }} <span class="field-picker-count" style="margin-left:4px;">{{ archiveSelectedFieldCount }}</span></span>
            <a-button danger size="small" :disabled="archiveRightSelectedKeys.length === 0" @click="archiveBatchUnpick">
              {{ t('admin.ruleConfig.remove') }}
            </a-button>
          </div>
          <div class="field-picker-search">
            <a-input
              v-model:value="archiveSelectedFieldSearchQuery"
              :placeholder="t('admin.ruleConfig.searchFieldPlaceholder')"
              allow-clear
              size="small"
            >
              <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
            </a-input>
          </div>
          <div class="field-picker-list" style="padding: 12px 16px;">
            <div
              v-for="field in archiveSelectedPagination.paged.value"
              :key="field.field_key + field.source"
              class="field-picker-item field-picker-item--selected"
              @click="toggleArchiveRightSelect(field.field_key + '_' + field.source)"
              style="display: flex; gap: 12px; justify-content: flex-start; margin-bottom: 8px;"
            >
              <div class="field-picker-item-checkbox" @click.stop="toggleArchiveRightSelect(field.field_key + '_' + field.source)">
                <a-checkbox :checked="archiveRightSelectedKeys.includes(field.field_key + '_' + field.source)" />
              </div>
              <div class="field-picker-item-info" style="flex: 1;">
                <div class="field-picker-item-name">{{ field.field_name }} <span class="field-source-tag" style="font-size: 11px; color: var(--color-text-tertiary); font-weight: normal; margin-left: 4px;">({{ field.sourceLabel }})</span></div>
                <div class="field-picker-item-meta">
                  <span class="field-type-tag">{{ fieldTypeLabels[field.field_type] || field.field_type }}</span>
                  <span class="field-key">{{ field.field_key }}</span>
                </div>
              </div>
              <button class="field-picker-remove" @click.stop="archiveUnpickField(field)" style="margin-left: auto;">
                <CloseOutlined />
              </button>
            </div>
            <div v-if="!archiveSelectedFieldsFlat.length" class="field-picker-empty">
              {{ archiveSelectedFieldSearchQuery ? t('admin.ruleConfig.noSearchResult') : t('admin.ruleConfig.noFieldsSelected') }}
            </div>
          </div>
          <div class="pagination-wrapper" style="padding: 12px 16px; border-top: 1px solid var(--color-border-light);">
            <a-pagination
              v-model:current="archiveSelectedPagination.current.value"
              v-model:page-size="archiveSelectedPagination.pageSize.value"
              :total="archiveSelectedPagination.total.value"
              size="small"
              show-size-changer
              :page-size-options="['5', '20', '50']"
              @change="archiveSelectedPagination.onChange"
              @showSizeChange="archiveSelectedPagination.onChange"
            />
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
  display: flex; justify-content: space-between; align-items: center; gap: 12px; flex-wrap: nowrap; margin-bottom: 14px;
}
.rules-toolbar-summary { display: flex; align-items: center; gap: 12px; flex-wrap: nowrap; white-space: nowrap; }
.rules-count { font-size: 13px; color: var(--color-text-tertiary); }
.rules-selected-count {
  padding: 2px 8px; border-radius: var(--radius-full);
  background: var(--color-primary-bg); color: var(--color-primary); font-size: 12px; font-weight: 600;
}
.rules-toolbar-actions { display: flex; gap: 8px; flex-wrap: nowrap; flex-shrink: 0; }

.rules-list { display: flex; flex-direction: column; gap: 10px; }
.rule-card {
  display: flex; align-items: center; justify-content: space-between;
  padding: 14px 18px; background: var(--color-bg-page); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); transition: all var(--transition-fast); gap: 16px;
}
.rule-card:hover { box-shadow: var(--shadow-sm); }
.rule-card-left { display: flex; align-items: flex-start; gap: 12px; flex: 1; min-width: 0; }
.rule-select-checkbox { margin-top: 3px; flex-shrink: 0; }
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

.rule-import-file-input { display: none; }
.rule-import-summary {
  display: flex; align-items: flex-start; justify-content: space-between; gap: 16px;
  margin-bottom: 14px; padding: 12px 14px; border-radius: var(--radius-md);
  background: var(--color-bg-hover);
}
.rule-import-file-name {
  display: flex; align-items: center; gap: 8px; font-weight: 600; color: var(--color-text-primary);
}
.rule-import-help { margin-top: 4px; font-size: 12px; color: var(--color-text-tertiary); }
.rule-import-paste-help { margin: 0 0 12px; line-height: 1.6; }
.rule-import-count {
  padding: 3px 10px; border-radius: var(--radius-full); white-space: nowrap;
  background: var(--color-primary-bg); color: var(--color-primary); font-size: 12px; font-weight: 600;
}
.rule-import-warning { margin-bottom: 10px; }
.rule-import-list {
  display: flex; flex-direction: column; gap: 12px; max-height: 58vh; overflow-y: auto; padding-right: 4px;
}
.rule-import-item {
  padding: 14px; border: 1px solid var(--color-border-light); border-radius: var(--radius-md);
  background: var(--color-bg-card); transition: opacity var(--transition-fast);
}
.rule-import-item--disabled { opacity: .55; }
.rule-import-item-head {
  display: flex; align-items: center; justify-content: space-between; margin-bottom: 10px;
}
.rule-import-confidence { font-size: 12px; color: var(--color-text-tertiary); }
.rule-import-options {
  display: flex; flex-wrap: wrap; align-items: center; gap: 18px; margin-top: 12px;
}
.rule-import-options label { display: flex; align-items: center; gap: 8px; color: var(--color-text-secondary); font-size: 13px; }
.rule-import-reasoning {
  display: flex; align-items: flex-start; gap: 6px; margin-top: 10px; font-size: 12px;
  line-height: 1.6; color: var(--color-text-tertiary);
}
.rule-import-context-hint {
  margin-top: 8px; padding: 7px 10px; border-radius: var(--radius-sm);
  background: var(--color-warning-bg); color: var(--color-warning); font-size: 12px;
}

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
  .summary-block-option-row { flex-direction: column; align-items: flex-start; }
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
.system-prompt-readonly-hint {
  margin-top: 6px; font-size: 12px; color: var(--color-text-quaternary);
  display: flex; align-items: center; gap: 4px;
  padding: 4px 8px; background: var(--color-bg-hover); border-radius: var(--radius-sm);
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

.summary-block-list { display: flex; flex-direction: column; gap: 14px; }
.summary-block-card {
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  background: var(--color-bg-card);
  padding: 16px;
}
.summary-block-head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.summary-block-option-row {
  margin-top: 14px;
  padding: 12px 14px;
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  background: var(--color-bg-hover);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.summary-block-option-copy {
  min-width: 0;
}
.summary-block-option-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-primary);
}
.summary-block-option-desc {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--color-text-tertiary);
}
.summary-context-panel {
  margin-top: 12px;
  border-top: 1px dashed var(--color-border-light);
  padding-top: 12px;
}
.summary-context-switches {
  display: flex;
  gap: 14px;
  align-items: center;
}
.context-switch {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--color-text-secondary);
}
.summary-context-box {
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  padding: 0;
  margin-top: 10px;
  background: var(--color-bg-hover);
  overflow: hidden;
}
.summary-context-box-head {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 12px;
  border: 0;
  background: transparent;
  text-align: left;
  cursor: pointer;
}
.summary-context-box-head .context-panel-title {
  margin-bottom: 0;
  flex-shrink: 0;
}
.summary-context-collapsed-hint {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  color: var(--color-text-tertiary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.summary-context-chevron {
  flex-shrink: 0;
  font-size: 12px;
  color: var(--color-text-tertiary);
  transition: transform .18s ease;
}
.summary-context-chevron--collapsed {
  transform: rotate(-90deg);
}
.summary-context-box-body {
  padding: 0 12px 12px;
}
.summary-context-box-body > .context-panel-title {
  display: none;
}
.summary-model-config-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0 12px;
}
.summary-model-config-grid :deep(.ant-form-item) {
  margin-bottom: 12px;
}
.summary-model-return-fields {
  grid-column: span 2;
}
.summary-model-footer {
  display: flex;
  align-items: flex-end;
  gap: 20px;
}
.summary-model-footer :deep(.ant-form-item) {
  margin-bottom: 0;
}
.summary-model-actions {
  margin: 0;
  padding-bottom: 0;
}
.summary-model-sql-item {
  margin-top: 12px;
  margin-bottom: 12px;
}
.summary-workflow-target-compact {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  margin-top: 4px;
  margin-bottom: 12px;
  border: 1px dashed var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-card);
}
.summary-workflow-target-meta {
  min-width: 0;
  flex: 1;
}
.context-actions {
  margin-top: 4px;
}
.summary-workflow-current {
  margin-bottom: 12px;
  padding: 10px 12px;
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  background: var(--color-bg-hover);
}
.workflow-config-modal .workflow-config-form {
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid var(--color-border-light);
}
.workflow-result-list--modal {
  max-height: 240px;
}
.workflow-result-list--add {
  max-height: 200px;
  margin-top: 8px;
}
.add-process-modal {
  display: flex;
  flex-direction: column;
  gap: 4px;
  margin-top: 8px;
}
.add-process-section {
  padding: 14px;
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  background: var(--color-bg-hover);
}
.add-process-section-head {
  margin-bottom: 12px;
}
.add-process-section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
}
.add-process-section-desc {
  margin-top: 4px;
  font-size: 12px;
  color: var(--color-text-tertiary);
}
.add-process-divider {
  display: flex;
  align-items: center;
  gap: 12px;
  margin: 10px 0;
  color: var(--color-text-tertiary);
  font-size: 12px;
}
.add-process-divider::before,
.add-process-divider::after {
  content: '';
  flex: 1;
  height: 1px;
  background: var(--color-border-light);
}
.add-process-section :deep(.ant-form-item:last-child) {
  margin-bottom: 0;
}
.summary-context-mode-switch {
  margin-bottom: 0;
}
.summary-context-mode-switch .field-mode-option {
  padding: 10px 12px;
  align-items: center;
}
.summary-context-mode-switch .field-mode-label {
  font-size: 13px;
  font-weight: 500;
}
.summary-field-select-item {
  margin-bottom: 0;
}
.summary-field-select-item :deep(.ant-form-item-label) {
  padding-bottom: 4px;
}
.summary-field-select-item :deep(.ant-form-item-label > label) {
  font-size: 13px;
  color: var(--color-text-secondary);
}
.summary-workflow-data-mode-item {
  margin-bottom: 12px;
}
.summary-basic-fields-item {
  margin-bottom: 12px;
}
.summary-basic-fields-item :deep(.ant-form-item-label) {
  width: 100%;
}
.summary-basic-fields-item :deep(.ant-form-item-label > label) {
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
  border-radius: var(--radius-md);
  background: var(--color-bg-card);
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
  background: var(--color-bg-hover);
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
.context-panel-title {
  font-weight: 600;
  margin-bottom: 10px;
  color: var(--color-text-primary);
}
.target-flow-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px;
  margin-bottom: 14px;
  border: 1px solid var(--color-border-light);
  border-radius: 8px;
  background: var(--color-bg-card);
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
.context-test {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-top: 10px;
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
.sql-variable-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
  font-size: 12px;
  color: var(--color-text-tertiary);
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
.summary-custom-data-hint {
  margin-top: 8px;
  padding: 8px 12px;
  font-size: 12px;
  line-height: 1.5;
  color: var(--color-info);
  background: var(--color-info-bg);
  border-radius: var(--radius-sm);
}
.summary-block-index {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--color-primary-bg);
  color: var(--color-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 13px;
  flex-shrink: 0;
}
.summary-field-picker {
  margin-top: 12px;
  padding: 12px;
  border: 1px dashed var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg-hover);
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

/* 数据表格样式 */
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
  .summary-model-config-grid { grid-template-columns: 1fr; }
  .summary-model-return-fields { grid-column: auto; }
  .summary-model-footer { align-items: flex-start; flex-direction: column; gap: 8px; }
}
</style>
