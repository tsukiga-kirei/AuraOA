<script setup lang="ts">
import { useAuditDataQueryApi } from '~/composables/useAuditDataQueryApi'
import { message, Modal } from 'ant-design-vue'
import QueryValueSourceEditor from './QueryValueSourceEditor.vue'
import QuerySchemaBrowser from './QuerySchemaBrowser.vue'
import { appDayjs } from '~/utils/appTime'
import { previewQuerySql, querySqlParameterNames } from '~/utils/querySqlEditor'
import {
  DatabaseOutlined,
  PlusOutlined,
  RobotOutlined,
  CheckCircleOutlined,
  PlayCircleOutlined,
  SaveOutlined,
  CopyOutlined,
  SettingOutlined,
  SearchOutlined,
  BookOutlined,
  DeleteOutlined,
  InfoCircleOutlined,
  CodeOutlined,
  HistoryOutlined,
  LinkOutlined,
} from '@ant-design/icons-vue'
import type {
  AuditDataQuery,
  QueryDefinition,
  QueryVersion,
  QueryCheck,
  QueryEvidence,
  QueryPrompts,
  QueryAssist,
  QueryColumn,
  QueryBuilder,
  QueryParameter,
  QueryUsageReference,
  QuerySource,
} from '~/types/audit-data-query'
const props = defineProps<{
  allowUse?: boolean
  initialQueryId?: string
  embedded?: boolean
}>()
const emit = defineEmits<{
  'use-version': [query: AuditDataQuery, version: QueryVersion]
}>()
const { t } = useI18n()
const { menus } = useAuth()
const canEditRules = computed(() => menus.value.some((menu) => menu.path === '/admin/tenant/rules'))
const api = useAuditDataQueryApi()
const makeDefinition = (): QueryDefinition => ({
  mode: 'config',
  ai_context: { oa_module: 'general', notes: '', field_notes: {} },
  name: '',
  description: '',
  sql: '',
  dialect: 'mysql',
  tables: [],
  parameters: [],
  max_rows: 0,
  builder: {
    source_type: 'table',
    archived_only: false,
    table: '',
    columns: [],
    filters: [],
    order_alias: '',
    order_dir: 'DESC',
  },
})
const items = ref<AuditDataQuery[]>([])
const selected = ref<AuditDataQuery>()
const definition = ref<QueryDefinition>(makeDefinition())
const source = ref<QuerySource>()
const category = ref('')
const editorMode = ref('config')
const builderKey = ref(0)
const baseline = ref('')
const keyword = ref('')
const appliedKeyword = ref('')
const loading = ref(false)
const initializing = ref(true)
const loadError = ref('')
const busy = ref('')
const activePanel = ref('basic')
const activeResultPanel = ref('check')
function openResultPanel(panel: string) {
  activeResultPanel.value = panel
  activePanel.value = 'results'
}
const versions = ref<QueryVersion[]>([])
const references = ref<QueryUsageReference[]>([])
const workflowReferences = computed(() => {
  const groups = new Map<
    string,
    { module: QueryUsageReference['module']; process_type: string; count: number }
  >()
  for (const reference of references.value) {
    const key = JSON.stringify([reference.module, reference.process_type])
    const group = groups.get(key)
    if (group) group.count++
    else
      groups.set(key, { module: reference.module, process_type: reference.process_type, count: 1 })
  }
  return [...groups.values()]
})
// 暂存同一条目的变量设置，输入 SQL 的过程中短暂删除占位符也不丢失取值配置。
const parameterSettings = new Map<string, QueryParameter>()
const check = ref<QueryCheck>()
const evidence = ref<QueryEvidence>()
const testValues = reactive<Record<string, string>>({})
const editor = ref<{
  insert: (value: string) => void
  focusPosition: (position: number) => void
}>()
const schema = ref<Record<string, QueryColumn[]>>({})
const loadingTables = ref<string[]>([])
const instruction = ref('')
const aiResult = ref<QueryAssist>()
const aiAction = ref<'generate' | 'explain' | 'repair'>('generate')
const lastError = ref('')
const promptsOpen = ref(false)
const prompts = ref<QueryPrompts>({ generate: '', explain: '', repair: '' })
const currentState = computed(() =>
  JSON.stringify({ definition: definition.value, category: category.value }),
)
const dirty = computed(() => currentState.value !== baseline.value)
const resultColumns = computed(() => check.value?.columns || [])
const executionSQL = computed(() => previewQuerySql(definition.value))
const contextTables = computed(() => {
  const tables = [...definition.value.tables]
  const builder = editorMode.value === 'config' ? definition.value.builder : undefined
  if (builder?.table.trim()) tables.push(builder.table.trim())
  if (builder?.source_type === 'workflow' && builder.archived_only !== false)
    tables.push('workflow_requestbase')
  return [...new Map(tables.map((table) => [table.toLowerCase(), table])).values()]
})
const lockedTables = computed(() =>
  editorMode.value === 'config'
    ? contextTables.value.filter(
        (table) =>
          !definition.value.tables.includes(table) || table === definition.value.builder?.table,
      )
    : [],
)
const aiContext = computed(() => definition.value.ai_context!)
const previewSummary = computed(() =>
  evidence.value
    ? {
        status: evidence.value.status,
        error: evidence.value.error,
        row_count: evidence.value.result?.rows.length || 0,
        truncated: evidence.value.result?.truncated || false,
        columns: evidence.value.result?.columns || [],
      }
    : undefined,
)
const contextPreview = computed(() =>
  JSON.stringify(
    {
      environment: source.value,
      category: category.value,
      target_mode: 'sql',
      definition: { ...definition.value, tables: contextTables.value },
      schema: Object.fromEntries(
        contextTables.value.map((table) => [table, schema.value[table] || []]),
      ),
      instruction: instruction.value,
      database_error: lastError.value,
      check: check.value,
      preview_summary: previewSummary.value,
    },
    null,
    2,
  ),
)
function normalizeDefinition(value: QueryDefinition) {
  value.mode ||= value.builder && !value.builder.sql_mode ? 'config' : 'sql'
  if (value.mode === 'sql') delete value.builder
  else if (value.builder) delete value.builder.sql_mode
  value.ai_context ||= {
    oa_module: value.builder?.source_type === 'workflow' ? 'workflow' : 'general',
    notes: '',
    field_notes: {},
  }
  value.ai_context.field_notes ||= {}
  return value
}
let listSequence = 0
let selectSequence = 0
const clone = <T,>(value: T): T => JSON.parse(JSON.stringify(value))
const errorText = (error: unknown) => (error instanceof Error ? error.message : String(error))
function invalidate() {
  check.value = undefined
  evidence.value = undefined
  aiResult.value = undefined
}
watch(currentState, invalidate)
watch([aiAction, instruction], () => {
  aiResult.value = undefined
})
watch(
  testValues,
  () => {
    evidence.value = undefined
  },
  { deep: true },
)
async function canLeave() {
  if (busy.value) return false
  if (!dirty.value) return true
  return await new Promise<boolean>((resolve) =>
    Modal.confirm({
      title: t('query.unsaved'),
      content: t('query.unsavedHint'),
      okText: t('query.discard'),
      cancelText: t('common.cancel'),
      onOk: () => resolve(true),
      onCancel: () => resolve(false),
    }),
  )
}
async function load() {
  const sequence = ++listSequence
  loading.value = true
  loadError.value = ''
  try {
    const result = await api.listAll(appliedKeyword.value)
    if (sequence !== listSequence) return
    items.value = result
  } catch (error) {
    if (sequence === listSequence) {
      loadError.value = errorText(error)
      message.error(loadError.value)
    }
  } finally {
    if (sequence === listSequence) loading.value = false
  }
}
async function loadVersionsAndReferences(id: string) {
  const [history, refs] = await Promise.all([api.versions(id), api.references(id)])
  return { history, refs }
}
async function selectQuery(item: AuditDataQuery, force = false) {
  if (!force && !(await canLeave())) return
  const sequence = ++selectSequence
  busy.value = 'select'
  try {
    const [query, related] = await Promise.all([
      api.get(item.id),
      loadVersionsAndReferences(item.id),
    ])
    if (sequence !== selectSequence) return
    selected.value = query
    definition.value = normalizeDefinition(clone(query.draft))
    if (source.value) definition.value.dialect = source.value.dialect
    editorMode.value = definition.value.mode!
    activePanel.value = 'basic'
    instruction.value = ''
    category.value = query.category
    baseline.value = currentState.value
    parameterSettings.clear()
    if (editorMode.value === 'sql') syncSQLParameters()
    builderKey.value++
    versions.value = related.history
    references.value = related.refs
    schema.value = {}
    loadingTables.value = []
    lastError.value = ''
    invalidate()
    for (const key of Object.keys(testValues)) delete testValues[key]
  } catch (error) {
    message.error(errorText(error))
  } finally {
    if (sequence === selectSequence) busy.value = ''
  }
}
function setNewDraft(next: QueryDefinition, nextCategory = '', copied = false) {
  ++selectSequence
  selected.value = undefined
  if (source.value) next.dialect = source.value.dialect
  definition.value = normalizeDefinition(next)
  editorMode.value = definition.value.mode!
  instruction.value = ''
  parameterSettings.clear()
  if (editorMode.value === 'sql') syncSQLParameters()
  builderKey.value++
  for (const key of Object.keys(testValues)) delete testValues[key]
  category.value = nextCategory
  versions.value = []
  references.value = []
  schema.value = {}
  loadingTables.value = []
  baseline.value = copied ? '' : currentState.value
  activePanel.value = 'basic'
  invalidate()
}
async function newQuery() {
  if (initializing.value || !(await canLeave())) return
  setNewDraft(makeDefinition())
}
async function requestCopy(item: AuditDataQuery) {
  if (!(await canLeave())) return
  Modal.confirm({
    title: t('query.copyConfirm'),
    content: t('query.copyConfirmHint', item.name),
    okText: t('query.copy'),
    cancelText: t('common.cancel'),
    async onOk() {
      busy.value = 'copy'
      try {
        const original = await api.get(item.id)
        const next = clone(original.draft)
        next.name = t('query.copyName', original.name)
        setNewDraft(next, original.category, true)
      } catch (error) {
        message.error(errorText(error))
        throw error
      } finally {
        busy.value = ''
      }
    },
  })
}
async function save() {
  if (!definition.value.name.trim()) {
    message.warning(t('query.nameRequired'))
    return false
  }
  const payload = clone(definition.value)
  const savedCategory = category.value
  const state = currentState.value
  busy.value = 'save'
  try {
    const item = await api.save(selected.value?.id, savedCategory, payload)
    selected.value = item
    baseline.value = state
    message.success(t('query.saved'))
    await load()
    return true
  } catch (error) {
    message.error(errorText(error))
    return false
  } finally {
    busy.value = ''
  }
}
async function checkSQL() {
  const state = currentState.value
  busy.value = 'check'
  openResultPanel('check')
  try {
    const result = await api.check(clone(definition.value))
    if (state !== currentState.value) return
    check.value = result
    if (result.valid) message.success(t('query.checkPassed'))
    else
      message.error(
        result.issues.map((issue) => issue.message).join('；') || t('query.checkFailed'),
        8,
      )
  } catch (error) {
    message.error(errorText(error))
  } finally {
    busy.value = ''
  }
}
async function preview() {
  if (busy.value) return
  // 先保存当前草稿，服务端以草稿和数据源指纹记录成功验证，发布时再次核对。
  if (dirty.value || !selected.value) {
    if (!(await save())) return
  }
  const state = currentState.value
  const testState = JSON.stringify(testValues)
  const parameters = clone(testValues)
  for (const parameter of definition.value.parameters) {
    if (
      parameter.default_binding &&
      parameter.default_binding.source !== 'field' &&
      parameters[parameter.name] === ''
    )
      delete parameters[parameter.name]
  }
  busy.value = 'preview'
  openResultPanel('test')
  lastError.value = ''
  try {
    const result = await api.preview({
      query_id: selected.value!.id,
      definition: clone(definition.value),
      parameters,
    })
    if (state !== currentState.value || testState !== JSON.stringify(testValues)) return
    evidence.value = result
    if (!result.error && !result.result?.truncated) message.success(t('query.previewSucceeded'))
    selected.value = await api.get(selected.value!.id)
  } catch (error) {
    lastError.value = errorText(error)
    message.error(lastError.value)
  } finally {
    busy.value = ''
  }
}
async function publish() {
  if (!selected.value || dirty.value) {
    message.warning(t('query.saveBeforePublish'))
    return
  }
  busy.value = 'publish'
  try {
    const version = await api.publish(selected.value.id)
    message.success(t('query.published', version.version_no))
    await selectQuery(selected.value, true)
    await load()
    openResultPanel('versions')
  } catch (error) {
    message.error(errorText(error))
  } finally {
    busy.value = ''
  }
}
async function requestDelete() {
  if (!selected.value || busy.value) return
  const item = selected.value
  busy.value = 'delete-check'
  try {
    references.value = await api.references(item.id)
    if (references.value.length) {
      openResultPanel('references')
      message.warning(t('query.deleteReferenced'))
      return
    }
  } catch (error) {
    message.error(errorText(error))
    return
  } finally {
    busy.value = ''
  }
  Modal.confirm({
    title: t('query.deleteConfirm'),
    content: t('query.deleteConfirmHint', item.name),
    okText: t('query.delete'),
    okType: 'danger',
    cancelText: t('common.cancel'),
    async onOk() {
      busy.value = 'delete'
      try {
        await api.remove(item.id)
        setNewDraft(makeDefinition())
        await load()
        if (items.value[0]) await selectQuery(items.value[0], true)
        message.success(t('query.deleted'))
      } catch (error) {
        // 后端也会校验引用；确认框打开期间新增的引用需展示给管理员。
        try {
          references.value = await api.references(item.id)
          if (references.value.length) openResultPanel('references')
        } catch {
          /* 操作失败保留原条目及草稿。 */
        }
        message.error(errorText(error))
        throw error
      } finally {
        busy.value = ''
      }
    },
  })
}
async function removeWorkflowReference(reference: (typeof workflowReferences.value)[number]) {
  if (!selected.value || busy.value || !canEditRules.value) return
  const id = selected.value.id
  busy.value = 'remove-reference'
  try {
    await api.removeReferences(id, reference.module, reference.process_type)
    references.value = references.value.filter(
      (item) => item.module !== reference.module || item.process_type !== reference.process_type,
    )
    references.value = await api.references(id)
    message.success(t('query.referenceRemoved'), 6)
  } catch (error) {
    message.error(errorText(error))
  } finally {
    busy.value = ''
  }
}
function addTable(table: string) {
  if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(table) || contextTables.value.length >= 20) {
    message.warning(t('query.tableNameHint'))
    return
  }
  if (!contextTables.value.some((item) => item.toLowerCase() === table.toLowerCase()))
    definition.value.tables.push(table)
  void loadSchema(table)
}
async function loadSchema(table: string, notify = true) {
  if (loadingTables.value.includes(table)) return
  const sequence = selectSequence
  loadingTables.value.push(table)
  try {
    const result = await api.schema(table)
    if (sequence !== selectSequence || !contextTables.value.includes(table)) return
    schema.value[table] = result.columns
    if (notify) message.success(t('query.schemaLoaded', [table, result.columns.length]))
  } catch (error) {
    if (sequence === selectSequence) message.error(errorText(error))
  } finally {
    if (sequence === selectSequence)
      loadingTables.value = loadingTables.value.filter((item) => item !== table)
  }
}
function rememberSchema(table: string, columns: QueryColumn[]) {
  schema.value[table] = columns
}
function setFieldNote(key: string, value: string) {
  aiContext.value.field_notes ||= {}
  if (value.trim()) aiContext.value.field_notes[key] = value
  else delete aiContext.value.field_notes[key]
}
function removeTable(table: string) {
  definition.value.tables = definition.value.tables.filter((item) => item !== table)
  delete schema.value[table]
  for (const key of Object.keys(aiContext.value.field_notes || {})) {
    if (key.startsWith(table + '.')) delete aiContext.value.field_notes![key]
  }
}
watch(activePanel, (panel) => {
  if (panel === 'ai') {
    for (const table of contextTables.value) if (!schema.value[table]) void loadSchema(table, false)
  }
})
async function focusSQL(position: number) {
  activePanel.value = 'sql'
  await nextTick()
  editor.value?.focusPosition(position)
}
async function confirmModeChange(mode: string) {
  const hasContent =
    !!definition.value.sql.trim() ||
    !!definition.value.builder?.columns.length ||
    !!definition.value.builder?.filters.length
  if (!hasContent || editorMode.value === mode) return true
  return await new Promise<boolean>((resolve) =>
    Modal.confirm({
      title: t('query.modeChangeTitle'),
      content: t(mode === 'sql' ? 'query.modeChangeSQL' : 'query.modeChangeConfig'),
      okText: t('query.modeChangeConfirm'),
      cancelText: t('common.cancel'),
      onOk: () => resolve(true),
      onCancel: () => resolve(false),
    }),
  )
}
async function changeMode(mode: 'config' | 'sql') {
  if (busy.value || mode === editorMode.value || !(await confirmModeChange(mode))) return
  const tables = [...contextTables.value]
  editorMode.value = mode
  definition.value.mode = mode
  if (mode === 'sql') {
    delete definition.value.builder
    definition.value.tables = tables
    syncSQLParameters()
  } else {
    definition.value.sql = ''
    definition.value.parameters = []
    definition.value.tables = []
    definition.value.builder = {
      source_type: 'table',
      archived_only: false,
      table: tables[0] || '',
      columns: [],
      filters: [],
      order_alias: '',
      order_dir: 'DESC',
    }
    parameterSettings.clear()
  }
  builderKey.value++
}
function syncSQLParameters() {
  for (const parameter of definition.value.parameters) {
    parameterSettings.set(parameter.name, clone(parameter))
  }
  definition.value.parameters = querySqlParameterNames(definition.value.sql).map((name, index) => {
    return clone(
      parameterSettings.get(name) || {
        name,
        label: t('query.sqlVariableLabel', index + 1),
        type: 'string' as const,
        default_binding: { source: 'field' as const },
      },
    )
  })
}
async function assist() {
  const state = currentState.value
  const action = aiAction.value
  const requestInstruction = instruction.value
  busy.value = 'assist'
  try {
    const result = await api.assist({
      action,
      category: category.value,
      definition: { ...clone(definition.value), tables: [...contextTables.value] },
      instruction: instruction.value,
      error: lastError.value.slice(0, 4000),
      preview: previewSummary.value,
    })
    if (
      state !== currentState.value ||
      action !== aiAction.value ||
      requestInstruction !== instruction.value
    )
      return
    aiResult.value = result
    if (result.warnings.length) message.warning(result.warnings.join('；'), 8)
  } catch (error) {
    message.error(errorText(error))
  } finally {
    busy.value = ''
  }
}
function updateBuilder(value: QueryBuilder) {
  const previous = definition.value.builder
  if (
    previous &&
    (previous.table !== value.table ||
      previous.source_type !== value.source_type ||
      previous.archived_only !== value.archived_only)
  ) {
    const previousTables = [previous.table]
    if (previous.source_type === 'workflow' && previous.archived_only !== false)
      previousTables.push('workflow_requestbase')
    definition.value.tables = definition.value.tables.filter(
      (table) => !previousTables.includes(table),
    )
  }
  definition.value.parameters = []
  definition.value.builder = value
  if (value.source_type === 'workflow' && aiContext.value.oa_module === 'general')
    aiContext.value.oa_module = 'workflow'
  definition.value.mode = 'config'
  definition.value.sql = ''
}
function applyBuilder(value: QueryDefinition) {
  definition.value = normalizeDefinition(value)
}
function markSQLMode() {
  definition.value.mode = 'sql'
  delete definition.value.builder
}
function updateSQL(value: string) {
  if (definition.value.sql === value) return
  definition.value.sql = value
  syncSQLParameters()
  // 自定义 SQL 只有一个编辑来源，不保留另一份可视化配置。
  definition.value.mode = 'sql'
  delete definition.value.builder
}
async function adoptAI() {
  if (!aiResult.value || busy.value) return
  const result = clone(aiResult.value)
  if (!(await confirmModeChange('sql'))) return
  definition.value = {
    ...clone(definition.value),
    mode: 'sql',
    builder: undefined,
    sql: result.sql,
    parameters: result.parameters,
    tables: result.tables,
  }
  editorMode.value = 'sql'
  syncSQLParameters()
  aiResult.value = undefined
  activePanel.value = 'sql'
  message.success(t('query.aiAdopted'))
}
async function editPrompts() {
  try {
    prompts.value = await api.prompts()
    promptsOpen.value = true
  } catch (error) {
    message.error(errorText(error))
  }
}
async function savePrompts() {
  busy.value = 'prompts'
  try {
    prompts.value = await api.savePrompts(clone(prompts.value))
    promptsOpen.value = false
    message.success(t('query.promptsSaved'))
  } catch (error) {
    message.error(errorText(error))
  } finally {
    busy.value = ''
  }
}
async function restoreVersion(version: QueryVersion) {
  if (!(await canLeave())) return
  definition.value = normalizeDefinition(clone(version.definition))
  if (source.value) definition.value.dialect = source.value.dialect
  editorMode.value = definition.value.mode!
  parameterSettings.clear()
  if (editorMode.value === 'sql') syncSQLParameters()
  builderKey.value++
  schema.value = {}
  for (const key of Object.keys(testValues)) delete testValues[key]
  activePanel.value = 'basic'
}
function search() {
  appliedKeyword.value = keyword.value.trim()
  void load()
}
function formatTime(value: string) {
  const date = appDayjs(value)
  return date.isValid() ? date.format(t('query.dateTimeFormat')) : '-'
}
onMounted(async () => {
  try {
    try {
      source.value = await api.source()
      definition.value.dialect = source.value.dialect
    } catch (error) {
      message.error(errorText(error))
    }
    baseline.value = currentState.value
    await load()
    if (props.initialQueryId) {
      try {
        const query = await api.get(props.initialQueryId)
        await selectQuery(query, true)
      } catch (error) {
        message.error(errorText(error))
        if (items.value[0]) await selectQuery(items.value[0], true)
      }
    } else if (items.value[0]) {
      await selectQuery(items.value[0], true)
    }
  } finally {
    initializing.value = false
  }
})
// 页面离开与工作区切换均保护未保存的 SQL 草稿。
onBeforeRouteLeave(async () => await canLeave())
function beforeUnload(event: BeforeUnloadEvent) {
  if (dirty.value) {
    event.preventDefault()
    event.returnValue = ''
  }
}
onMounted(() => window.addEventListener('beforeunload', beforeUnload))
onBeforeUnmount(() => window.removeEventListener('beforeunload', beforeUnload))
defineExpose({ canLeave, editPrompts, busy })
</script>

<template>
  <div class="query-workspace memory-surface">
    <div v-if="!embedded" class="workspace-heading">
      <div class="heading-content">
        <span class="heading-icon"><DatabaseOutlined /></span>
        <div>
          <h1>{{ t('query.title') }}</h1>
          <p>{{ t('query.subtitle') }}</p>
        </div>
      </div>
    </div>
    <div class="workspace-layout">
      <aside class="query-library panel">
        <div class="library-top">
          <BookOutlined />
          <span>{{ t('query.library') }}</span>
          <button
            type="button"
            class="add-library-btn"
            :disabled="!!busy"
            :title="t('query.new')"
            :aria-label="t('query.new')"
            @click="newQuery()"
          >
            <PlusOutlined />
          </button>
        </div>
        <div class="library-search memory-input-action">
          <a-input
            v-model:value="keyword"
            :placeholder="t('query.search')"
            :aria-label="t('query.search')"
            allow-clear
            @press-enter="search"
            @change="!keyword && search()"
          />
          <a-button class="memory-icon-button" :aria-label="t('common.search')" @click="search">
            <template #icon><SearchOutlined /></template>
          </a-button>
        </div>
        <a-button v-if="loadError" size="small" class="library-retry" @click="load">
          {{ t('query.retry') }}
        </a-button>
        <a-spin :spinning="loading">
          <div class="library-list">
            <div
              v-for="item in items"
              :key="item.id"
              class="library-item"
              :class="{ selected: selected?.id === item.id }"
            >
              <button
                type="button"
                class="library-select"
                :disabled="!!busy"
                :aria-pressed="selected?.id === item.id"
                @click="selectQuery(item)"
              >
                <span class="item-name">{{ item.name }}</span>
                <span class="item-meta">
                  {{ item.category || t('query.uncategorized') }} ·
                  {{
                    item.latest_version ? t('query.version', item.latest_version) : t('query.draft')
                  }}
                </span>
              </button>
              <a-button
                type="text"
                size="small"
                class="library-copy memory-icon-button"
                :disabled="!!busy"
                :title="t('query.copy')"
                :aria-label="t('query.copyItem', item.name)"
                @click="requestCopy(item)"
              >
                <template #icon><CopyOutlined /></template>
              </a-button>
            </div>
            <p v-if="!items.length && !loading" class="library-empty">{{ t('query.empty') }}</p>
          </div>
        </a-spin>
      </aside>
      <main v-if="initializing" class="query-main panel initial-query-loading">
        <a-spin />
      </main>
      <main v-else class="query-main panel">
        <div class="draft-meta">
          <a-space wrap>
            <a-tag class="query-version-tag">
              {{
                selected?.latest_version
                  ? t('query.version', selected.latest_version)
                  : t('query.draft')
              }}
            </a-tag>
            <span v-if="dirty" class="muted">{{ t('query.unsaved') }}</span>
          </a-space>
          <a-space wrap>
            <a-button
              v-if="selected"
              danger
              :loading="busy === 'delete' || busy === 'delete-check'"
              :disabled="!!busy"
              @click="requestDelete"
            >
              <template #icon><DeleteOutlined /></template>
              {{ t('query.delete') }}
            </a-button>
            <a-button type="primary" :loading="busy === 'save'" :disabled="!!busy" @click="save">
              <template #icon><SaveOutlined /></template>
              {{ t('query.save') }}
            </a-button>
          </a-space>
        </div>
        <a-tabs v-model:active-key="activePanel" class="workspace-tabs" :tab-bar-gutter="4">
          <a-tab-pane key="basic"
            ><template #tab
              ><InfoCircleOutlined /><span>{{ t('query.tab.basic') }}</span></template
            >
            <a-form layout="vertical" class="definition-form">
              <a-row :gutter="16">
                <a-col :span="14">
                  <a-form-item :label="t('query.name')" required>
                    <a-input
                      v-model:value="definition.name"
                      :maxlength="200"
                      :placeholder="t('query.namePlaceholder')"
                    />
                  </a-form-item>
                </a-col>
                <a-col :span="10">
                  <a-form-item :label="t('query.category')">
                    <a-input v-model:value="category" :maxlength="100" />
                  </a-form-item>
                </a-col>
              </a-row>
              <a-form-item :label="t('query.description')" required>
                <a-textarea
                  v-model:value="definition.description"
                  :rows="2"
                  :maxlength="8000"
                  :placeholder="t('query.descriptionHint')"
                />
              </a-form-item>
              <a-row :gutter="16">
                <a-col :span="12">
                  <a-form-item :label="t('query.dialect')">
                    <a-input
                      :value="
                        source
                          ? `${source.name} · ${source.dialect === 'dm' ? t('query.dm') : source.dialect === 'mysql' ? 'MySQL' : 'Oracle'}`
                          : t('query.sourceMissing')
                      "
                      readonly
                    />
                    <small class="muted">{{ t('query.sourceAutomatic') }}</small>
                  </a-form-item>
                </a-col>
                <a-col :span="12">
                  <a-form-item :label="t('query.maxRows')">
                    <a-input-number
                      :value="definition.max_rows || undefined"
                      :min="0"
                      :precision="0"
                      :placeholder="t('query.unlimited')"
                      @update:value="
                        (value) => (definition.max_rows = value == null ? 0 : Number(value))
                      "
                      style="width: 100%"
                    />
                  </a-form-item>
                </a-col>
              </a-row>
            </a-form>
            <div class="tab-footer">
              <a-button type="primary" @click="activePanel = 'sql'">{{
                t('query.nextSQL')
              }}</a-button>
            </div>
          </a-tab-pane>
          <a-tab-pane key="sql"
            ><template #tab
              ><CodeOutlined /><span>{{ t('query.tab.sql') }}</span></template
            >
            <a-form layout="vertical" class="definition-form">
              <a-radio-group
                :value="editorMode"
                :disabled="!!busy"
                button-style="solid"
                @change="changeMode($event.target.value)"
              >
                <a-radio-button value="config">{{ t('query.builder.mode') }}</a-radio-button>
                <a-radio-button value="sql">{{ t('query.builder.sqlMode') }}</a-radio-button>
              </a-radio-group>
              <p class="memory-help">{{ t('query.exclusiveModeHint') }}</p>
              <QueryBuilderEditor
                v-if="editorMode === 'config'"
                :key="`query-builder-${builderKey}`"
                :definition="definition"
                :disabled="!!busy || !source"
                @apply="applyBuilder"
                @update:builder="updateBuilder"
                @schema="rememberSchema"
              />
              <QuerySqlEditor
                v-if="editorMode === 'sql'"
                ref="editor"
                class="sql-template"
                :model-value="definition.sql"
                @update:model-value="updateSQL"
                @run="preview"
              />
              <a-collapse
                v-if="definition.sql"
                :default-active-key="['execution-sql']"
                class="builder-notice"
              >
                <a-collapse-panel key="execution-sql" :header="t('query.executionSQL')">
                  <p class="memory-help">
                    {{
                      definition.max_rows > 0
                        ? t('query.rowLimitPreview', definition.max_rows)
                        : t('query.unlimitedPreview')
                    }}
                  </p>
                  <pre class="ai-sql">{{ executionSQL }}</pre>
                </a-collapse-panel>
              </a-collapse>
              <template v-if="editorMode === 'sql' && definition.parameters.length">
                <div class="parameter-heading">
                  <strong>{{ t('query.sqlVariableValues') }}</strong>
                </div>
                <div
                  v-for="parameter in definition.parameters"
                  :key="parameter.name"
                  class="sql-parameter-source"
                >
                  <div class="sql-variable-meta">
                    <code>:{{ parameter.name }}</code>
                    <a-input
                      v-model:value="parameter.label"
                      @update:value="markSQLMode"
                      :disabled="!!busy"
                      :aria-label="t('query.parameterLabel')"
                      :placeholder="t('query.parameterLabel')"
                    />
                    <a-select
                      v-model:value="parameter.type"
                      @update:value="markSQLMode"
                      :disabled="!!busy"
                      :aria-label="t('query.parameterType')"
                    >
                      <a-select-option
                        v-for="type in ['string', 'number', 'date', 'string_list']"
                        :key="type"
                        :value="type"
                      >
                        {{ t('query.type.' + type) }}
                      </a-select-option>
                    </a-select>
                  </div>
                  <QueryValueSourceEditor
                    v-model="parameter.default_binding"
                    @update:model-value="markSQLMode"
                    :type="parameter.type"
                    :disabled="!!busy"
                  />
                </div>
              </template>

              <a-collapse v-if="editorMode === 'sql'" class="builder-notice">
                <a-collapse-panel key="schema" :header="t('query.schema')">
                  <QuerySchemaBrowser
                    :tables="contextTables"
                    :schema="schema"
                    :loading-tables="loadingTables"
                    :field-notes="aiContext.field_notes || {}"
                    :disabled="!!busy"
                    @add="addTable"
                    @remove="removeTable"
                    @load="loadSchema"
                    @note="setFieldNote"
                  />
                </a-collapse-panel>
              </a-collapse>
            </a-form>
          </a-tab-pane>
          <a-tab-pane key="ai">
            <template #tab
              ><RobotOutlined /><span>{{ t('query.assistant') }}</span></template
            >
            <section class="ai-workspace">
              <div class="ai-heading">
                <div>
                  <strong>{{ t('query.assistant') }}</strong>
                  <p class="memory-help">{{ t('query.aiWorkspaceHint') }}</p>
                </div>
                <a-button :disabled="!!busy" @click="editPrompts"
                  ><template #icon><SettingOutlined /></template
                  >{{ t('query.aiPrompts') }}</a-button
                >
              </div>
              <div class="ai-context-summary">
                <div>
                  <small>{{ t('query.oaEnvironment') }}</small
                  ><strong>{{ source?.oa_name || t('query.sourceMissing') }}</strong
                  ><span>{{ source?.dialect }} · {{ source?.timezone }}</span>
                </div>
                <div>
                  <small>{{ t('query.contextBasic') }}</small
                  ><strong>{{ definition.name || t('query.notFilled') }}</strong
                  ><span>{{ category || t('query.uncategorized') }}</span>
                </div>
                <div>
                  <small>{{ t('query.contextQuery') }}</small
                  ><strong>{{
                    t(editorMode === 'config' ? 'query.builder.mode' : 'query.builder.sqlMode')
                  }}</strong
                  ><span
                    >{{ t('query.contextTableCount', contextTables.length) }} ·
                    {{
                      t(
                        'query.contextVariableCount',
                        definition.parameters.length || definition.builder?.filters.length || 0,
                      )
                    }}</span
                  >
                </div>
              </div>
              <a-form layout="vertical" class="ai-context-form">
                <a-form-item :label="t('query.oaModule')">
                  <a-select v-model:value="aiContext.oa_module" :disabled="!!busy"
                    ><a-select-option
                      v-for="module in ['general', 'workflow', 'model']"
                      :key="module"
                      :value="module"
                      >{{ t('query.module.' + module) }}</a-select-option
                    ></a-select
                  >
                </a-form-item>
                <a-form-item :label="t('query.contextNotes')"
                  ><a-textarea
                    v-model:value="aiContext.notes"
                    :disabled="!!busy"
                    :rows="2"
                    :maxlength="8000"
                    :placeholder="t('query.contextNotesHint')"
                /></a-form-item>
              </a-form>
              <h3 class="ai-section-title">{{ t('query.schema') }}</h3>
              <QuerySchemaBrowser
                :tables="contextTables"
                :schema="schema"
                :loading-tables="loadingTables"
                :field-notes="aiContext.field_notes || {}"
                :locked-tables="lockedTables"
                :disabled="!!busy"
                @add="addTable"
                @remove="removeTable"
                @load="loadSchema"
                @note="setFieldNote"
              />
              <div class="ai-request">
                <div class="ai-action-row">
                  <a-select v-model:value="aiAction" :disabled="!!busy"
                    ><a-select-option
                      v-for="action in ['generate', 'explain', 'repair']"
                      :key="action"
                      :value="action"
                      >{{ t('query.action.' + action) }}</a-select-option
                    ></a-select
                  >
                  <span class="memory-help">{{ t('query.aiCustomOnly') }}</span>
                </div>
                <a-textarea
                  v-model:value="instruction"
                  :disabled="!!busy"
                  :rows="4"
                  :maxlength="8000"
                  :placeholder="t('query.aiInstruction')"
                  class="ai-instruction"
                />
                <div class="panel-actions">
                  <a-button
                    type="primary"
                    :loading="busy === 'assist'"
                    :disabled="
                      !!busy ||
                      !source ||
                      (aiAction === 'generate' ? !contextTables.length : !definition.sql.trim())
                    "
                    @click="assist"
                    ><template #icon><RobotOutlined v-if="busy !== 'assist'" /></template
                    >{{ t('query.runAI') }}</a-button
                  >
                </div>
              </div>
              <a-collapse class="context-preview"
                ><a-collapse-panel key="context" :header="t('query.contextPreview')"
                  ><p class="memory-help">{{ t('query.contextPreviewHint') }}</p>
                  <pre class="ai-sql">{{ contextPreview }}</pre>
                </a-collapse-panel></a-collapse
              >
              <section v-if="aiResult" class="ai-candidate">
                <h3 class="ai-section-title">{{ t('query.aiCandidate') }}</h3>
                <div class="ai-explanation">{{ aiResult.explanation }}</div>
                <ul v-if="aiResult.warnings.length" class="ai-warnings">
                  <li v-for="(warning, index) in aiResult.warnings" :key="index">{{ warning }}</li>
                </ul>
                <pre v-if="aiResult.sql" class="ai-sql">{{ aiResult.sql }}</pre>
                <p class="memory-help">{{ t('query.aiCandidateHint') }}</p>
                <a-button
                  v-if="aiAction !== 'explain' && aiResult.sql"
                  type="primary"
                  :disabled="!!busy"
                  @click="adoptAI"
                  >{{ t('query.adoptAI') }}</a-button
                >
              </section>
            </section>
          </a-tab-pane>
          <a-tab-pane key="results">
            <template #tab
              ><CheckCircleOutlined /><span>{{ t('query.tab.results') }}</span></template
            >
            <a-tabs v-model:active-key="activeResultPanel" class="result-tabs" :tab-bar-gutter="28">
              <a-tab-pane key="check"
                ><template #tab
                  ><CheckCircleOutlined /><span>{{ t('query.checkResult') }}</span></template
                >
                <section class="tab-section">
                  <div class="panel-actions">
                    <a-button
                      type="primary"
                      :loading="busy === 'check'"
                      :disabled="!!busy || !source"
                      @click="checkSQL"
                    >
                      <template #icon><CheckCircleOutlined /></template>
                      {{ t('query.check') }}
                    </a-button>
                  </div>
                  <template v-if="check">
                    <button
                      v-for="(issue, index) in check.issues"
                      :key="index"
                      class="check-issue"
                      @click="focusSQL(issue.position)"
                    >
                      {{ issue.message }}
                    </button>
                    <p v-if="resultColumns.length" class="output-columns">
                      {{ t('query.outputColumns') }}：{{ resultColumns.join('、') }}
                    </p>
                  </template>
                  <a-empty v-else :description="t('query.checkEmpty')" /></section
              ></a-tab-pane>
              <a-tab-pane key="test"
                ><template #tab
                  ><PlayCircleOutlined /><span>{{ t('query.test') }}</span></template
                >
                <section class="tab-section">
                  <div class="test-parameters">
                    <a-form-item
                      v-for="parameter in definition.parameters"
                      :key="parameter.name"
                      :label="parameter.label || parameter.name"
                    >
                      <a-input
                        v-model:value="testValues[parameter.name]"
                        :placeholder="
                          parameter.default_binding &&
                          parameter.default_binding.source !== 'field' &&
                          parameter.default_binding.value !== 'process_id'
                            ? t('query.defaultTestValueHint')
                            : t('query.testValueHint')
                        "
                      />
                    </a-form-item>
                  </div>
                  <div class="panel-actions">
                    <a-button
                      type="primary"
                      :loading="busy === 'preview'"
                      :disabled="!!busy || !source"
                      @click="preview"
                    >
                      <template #icon><PlayCircleOutlined /></template>
                      {{ t('query.preview') }}
                    </a-button>
                  </div>
                  <QueryEvidencePreview v-if="evidence" :evidence="evidence" />
                  <a-empty v-else :description="t('query.previewEmpty')" /></section
              ></a-tab-pane>
              <a-tab-pane key="versions"
                ><template #tab
                  ><HistoryOutlined /><span>{{ t('query.versions') }}</span></template
                >
                <section class="tab-section">
                  <div class="panel-actions">
                    <a-button
                      type="primary"
                      :loading="busy === 'publish'"
                      :disabled="!!busy || !selected || dirty || !selected.tested_hash"
                      @click="publish"
                    >
                      {{ t('query.publish') }}
                    </a-button>
                  </div>
                  <a-empty v-if="!versions.length" :description="t('query.noVersions')" />
                  <div v-for="version in versions" :key="version.id" class="version-row">
                    <div>
                      <strong>{{ t('query.version', version.version_no) }}</strong>
                      <span class="muted">· {{ formatTime(version.created_at) }}</span>
                    </div>
                    <a-space>
                      <a-button size="small" @click="restoreVersion(version)">
                        {{ t('query.loadDraft') }}
                      </a-button>
                      <a-button
                        v-if="allowUse && selected"
                        size="small"
                        type="primary"
                        @click="emit('use-version', selected!, version)"
                      >
                        {{ t('query.useVersion') }}
                      </a-button>
                    </a-space>
                  </div>
                </section></a-tab-pane
              >
              <a-tab-pane key="references"
                ><template #tab
                  ><LinkOutlined /><span>{{ t('query.references') }}</span></template
                >
                <section class="tab-section">
                  <a-empty
                    v-if="!workflowReferences.length"
                    :description="t('query.noReferences')"
                  />
                  <div
                    v-for="reference in workflowReferences"
                    :key="JSON.stringify([reference.module, reference.process_type])"
                    class="reference-row"
                  >
                    <div>
                      <strong>{{ reference.process_type }}</strong>
                      <p>
                        {{
                          t(
                            reference.module === 'audit_rules'
                              ? 'query.auditUsage'
                              : 'query.archiveUsage',
                          )
                        }}
                      </p>
                    </div>
                    <a-popconfirm
                      v-if="canEditRules"
                      :title="t('query.removeReferenceConfirm', reference.process_type)"
                      :ok-text="t('query.removeReference')"
                      :cancel-text="t('common.cancel')"
                      :disabled="!!busy"
                      @confirm="removeWorkflowReference(reference)"
                    >
                      <a-button danger :disabled="!!busy">{{
                        t('query.removeReference')
                      }}</a-button>
                    </a-popconfirm>
                  </div>
                </section></a-tab-pane
              >
            </a-tabs>
          </a-tab-pane>
        </a-tabs>
      </main>
    </div>
    <a-modal
      v-model:open="promptsOpen"
      :title="t('query.aiPrompts')"
      :width="850"
      :ok-text="t('query.save')"
      :cancel-text="t('common.cancel')"
      :confirm-loading="busy === 'prompts'"
      @ok="savePrompts"
    >
      <p class="memory-help">{{ t('query.promptHint') }}</p>
      <a-tabs>
        <a-tab-pane
          v-for="action in ['generate', 'explain', 'repair'] as const"
          :key="action"
          :tab="t('query.action.' + action)"
        >
          <a-textarea v-model:value="prompts[action]" :rows="12" :maxlength="12000" />
        </a-tab-pane>
      </a-tabs>
    </a-modal>
  </div>
</template>

<style scoped>
.result-tabs > :deep(.ant-tabs-nav) {
  margin: 0 0 24px;
  padding: 0;
  background: transparent;
  border-radius: 0;
}
.result-tabs > :deep(.ant-tabs-nav .ant-tabs-tab) {
  padding: 16px 0;
  border-radius: 0;
  box-shadow: none;
  background: transparent;
}
.result-tabs > :deep(.ant-tabs-nav .ant-tabs-ink-bar) {
  display: block;
}

.query-workspace {
  color: var(--color-text-primary);
}
.workspace-heading {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
  margin-bottom: 24px;
}
.heading-content {
  display: flex;
  gap: 14px;
  align-items: center;
}
.heading-icon {
  display: grid;
  place-items: center;
  width: 48px;
  height: 48px;
  border-radius: 12px;
  background: var(--color-primary-bg);
  color: var(--color-primary);
  font-size: 24px;
}
h1 {
  font-size: 24px;
  font-weight: 600;
  margin: 0 0 4px;
}
.heading-content p {
  margin: 0;
  color: var(--color-text-secondary);
  font-size: 13px;
}
.workspace-layout {
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr);
  gap: 20px;
  align-items: start;
}
.panel {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-lg);
}
.query-assistant {
  padding: 16px;
}
.query-library {
  overflow: hidden;
  position: sticky;
  top: 20px;
}
.library-top,
.schema-heading,
.parameter-heading,
.version-row,
.draft-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}
.library-top {
  padding: 14px 16px;
  border-bottom: 1px solid var(--color-border-light);
  justify-content: flex-start;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
}
.add-library-btn {
  margin-left: auto;
  width: 26px;
  height: 26px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px dashed var(--color-border);
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-text-tertiary);
  font-size: 12px;
  cursor: pointer;
  transition: all var(--transition-fast);
}
.add-library-btn:hover:not(:disabled) {
  border-color: var(--color-primary);
  color: var(--color-primary);
  background: var(--color-primary-bg);
}
.add-library-btn:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}
.library-search {
  padding: 12px 16px;
  border-bottom: 1px solid var(--color-border-light);
}
.library-list {
  margin: 0;
  max-height: calc(100vh - 340px);
  overflow-y: auto;
}
.library-retry {
  margin: 12px 16px;
}
.library-empty {
  padding: 20px 16px;
  margin: 0;
  color: var(--color-text-tertiary);
  font-size: 12px;
  line-height: 1.7;
}
.library-item {
  display: flex;
  align-items: center;
  gap: 8px;
  text-align: left;
  width: 100%;
  border: 0;
  border-bottom: 1px solid var(--color-border-light);
  background: transparent;
  color: var(--color-text-primary);
  padding: 0 12px 0 0;
  transition: all var(--transition-fast);
}
.library-select {
  display: flex;
  flex: 1;
  min-width: 0;
  flex-direction: column;
  gap: 2px;
  padding: 12px 16px;
  border: 0;
  text-align: left;
  background: transparent;
  color: inherit;
  cursor: pointer;
}
.library-select:disabled {
  cursor: not-allowed;
}
.library-copy {
  flex-shrink: 0;
  color: var(--color-text-tertiary);
}
.library-item:last-child {
  border-bottom: 0;
}
.library-item:hover {
  background: var(--color-bg-hover);
}
.library-item.selected {
  background: var(--color-primary-bg);
  border-left: 3px solid var(--color-primary);
}
.library-select:focus-visible,
.add-library-btn:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: -2px;
}
.item-name {
  font-size: 14px;
  font-weight: 500;
  overflow-wrap: anywhere;
}
.item-meta,
.muted {
  font-size: 12px;
  color: var(--color-text-tertiary);
  line-height: 1.7;
}
.draft-meta {
  padding: 20px 20px 0;
  flex-wrap: wrap;
}
.panel-actions {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.panel-actions {
  margin: 16px 0;
}
.sql-template {
  margin-top: 16px;
}
.initial-query-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 360px;
}
.definition-form {
  padding: 0;
}
.sql-parameter-source {
  display: flex;
  flex-direction: column;
  gap: 8px;
  margin-bottom: 16px;
}
.parameter-heading {
  margin: 20px 0 12px;
}
.sql-variable-meta {
  display: grid;
  grid-template-columns: minmax(100px, 1fr) minmax(120px, 2fr) 110px;
  gap: 12px;
  align-items: center;
}
.sql-variable-meta code {
  overflow-wrap: anywhere;
  color: var(--color-text-secondary);
}
.query-main {
  min-width: 0;
}
.workspace-tabs {
  padding: 20px 20px 32px;
}
/* 导航沿用规则配置页的内容宽度与边框，不将灰色背景拉满编辑区。 */
.workspace-tabs > :deep(.ant-tabs-nav) {
  width: fit-content;
  max-width: 100%;
  margin: 0 0 24px;
  background: var(--color-bg-hover);
  padding: 4px;
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-lg);
}
.workspace-tabs > :deep(.ant-tabs-nav .ant-tabs-nav-wrap) {
  flex: 0 1 auto;
}
.workspace-tabs > :deep(.ant-tabs-nav::before) {
  border-bottom: 0;
}
.workspace-tabs > :deep(.ant-tabs-nav .ant-tabs-tab) {
  padding: 7px 14px;
  border-radius: var(--radius-md);
  font-size: 13.5px;
}
.workspace-tabs > :deep(.ant-tabs-nav .ant-tabs-tab-active) {
  background: var(--color-bg-card);
  box-shadow: var(--shadow-xs);
}
.workspace-tabs > :deep(.ant-tabs-nav .ant-tabs-tab-active .ant-tabs-tab-btn) {
  font-weight: 600;
}
.workspace-tabs > :deep(.ant-tabs-nav .ant-tabs-tab-btn) {
  display: flex;
  align-items: center;
  gap: 6px;
}
.workspace-tabs > :deep(.ant-tabs-nav .ant-tabs-tab-btn .anticon) {
  margin-right: 0;
}
.workspace-tabs > :deep(.ant-tabs-nav .ant-tabs-ink-bar) {
  display: none;
}
.tab-section,
.ai-workspace {
  padding: 0;
}
.tab-section {
  min-height: 240px;
}
.tab-section > :deep(.ant-empty) {
  margin: 32px 0;
}
.tab-footer {
  display: flex;
  justify-content: flex-end;
  padding: 0;
  margin-top: 24px;
}
.ai-heading {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}
.ai-heading > div {
  min-width: 0;
}
.ai-heading > .ant-btn {
  flex-shrink: 0;
}
.ai-context-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 20px;
}
.ai-context-summary > div {
  display: flex;
  flex-direction: column;
  gap: 4px;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  padding: 12px;
  overflow-wrap: anywhere;
}
.ai-context-summary small,
.ai-context-summary span {
  color: var(--color-text-tertiary);
  font-size: 12px;
}
.ai-context-summary strong {
  font-size: 13px;
  font-weight: 500;
}
.ai-context-form {
  display: grid;
  grid-template-columns: 180px minmax(0, 1fr);
  gap: 16px;
}
.ai-section-title {
  font-size: 14px;
  font-weight: 600;
  margin: 20px 0 12px;
}
.ai-request {
  margin-top: 24px;
}
.ai-action-row {
  display: flex;
  align-items: center;
  gap: 12px;
}
.ai-action-row .ant-select {
  width: 160px;
  flex-shrink: 0;
}
.ai-candidate {
  padding: 0 0 4px;
}
.ai-warnings {
  padding-left: 20px;
  font-size: 13px;
  line-height: 1.8;
  color: var(--color-text-secondary);
}
.test-parameters {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 0 16px;
  margin-top: 16px;
}
.check-issue {
  display: block;
  width: 100%;
  text-align: left;
  background: var(--color-bg-secondary);
  color: var(--color-text-secondary);
  border: 1px solid var(--color-border);
  padding: 10px 12px;
  border-radius: 6px;
  cursor: pointer;
  margin-top: 8px;
}
.output-columns {
  font-size: 13px;
  overflow-wrap: anywhere;
}
.version-row,
.reference-row {
  padding: 12px 0;
  border-bottom: 1px solid var(--color-border-light);
}
.reference-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.reference-row > div {
  min-width: 0;
  overflow-wrap: anywhere;
}
.reference-row .ant-btn {
  flex-shrink: 0;
}
.reference-row p {
  font-size: 13px;
  color: var(--color-text-secondary);
}
.section-heading {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--color-primary);
}
.schema-table {
  margin-top: 12px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--color-border-light);
}
.schema-column {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  width: 100%;
  text-align: left;
  background: transparent;
  border: 0;
  padding: 5px 8px;
  color: var(--color-text-secondary);
  cursor: pointer;
  overflow-wrap: anywhere;
}
.schema-column:hover {
  background: var(--color-bg-hover);
}
.builder-notice {
  margin-top: 12px;
}
.schema-column small {
  color: var(--color-text-tertiary);
}
.ai-instruction {
  margin: 12px 0;
}
.ai-explanation {
  font-size: 13px;
  line-height: 1.8;
  white-space: pre-wrap;
}
.ai-sql {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  background: var(--color-bg-secondary);
  color: var(--color-text-primary);
  border-radius: 6px;
  padding: 12px;
  font-size: 12px;
  max-height: 260px;
  overflow: auto;
}
@media (min-width: 1700px) {
  .workspace-layout {
    grid-template-columns: 260px minmax(0, 1fr);
  }
}
@media (max-width: 1380px) {
  .workspace-layout {
    grid-template-columns: 240px minmax(0, 1fr);
  }
  .query-assistant {
    grid-column: 2;
  }
}
@media (max-width: 760px) {
  .query-library {
    position: static;
  }
  .workspace-heading {
    align-items: flex-start;
    flex-direction: column;
  }
  .workspace-layout {
    display: flex;
    flex-direction: column;
  }
  .panel {
    width: 100%;
    box-sizing: border-box;
  }
  .query-library .library-list {
    min-height: 0;
    max-height: 200px;
    overflow: auto;
  }
  .query-main {
    min-width: 0;
  }
  .query-assistant {
    order: 1;
  }
  .workspace-tabs {
    padding: 16px 16px 32px;
  }
  .ai-context-summary,
  .ai-context-form {
    grid-template-columns: minmax(0, 1fr);
  }
  .ai-heading {
    flex-wrap: wrap;
  }
  .test-parameters {
    grid-template-columns: 1fr 1fr;
  }
  .sql-variable-meta {
    grid-template-columns: minmax(0, 1fr) 110px;
  }
  .sql-variable-meta code {
    grid-column: 1 / -1;
  }
  .draft-meta {
    padding: 12px;
  }
  h1 {
    font-size: 20px;
  }
}
</style>
