<script setup lang="ts">
import { useAuditDataQueryApi } from '~/composables/useAuditDataQueryApi'
import { message, Modal } from 'ant-design-vue'
import { appDayjs } from '~/utils/appTime'
import { previewQuerySql } from '~/utils/querySqlEditor'
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
const api = useAuditDataQueryApi()
const makeDefinition = (): QueryDefinition => ({
  name: '',
  description: '',
  sql: '',
  dialect: 'mysql',
  tables: [],
  parameters: [],
  max_rows: 0,
  builder: {
    source_type: 'table',
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
const source = ref<{ name: string; dialect: QueryDefinition['dialect'] }>()
const category = ref('')
const editorMode = ref('config')
const builderKey = ref(0)
const baseline = ref('')
const keyword = ref('')
const appliedKeyword = ref('')
const loading = ref(false)
const loadError = ref('')
const busy = ref('')
const activePanel = ref('check')
const versions = ref<QueryVersion[]>([])
const references = ref<
  { id: string; process_type: string; rule_content: string; module: string }[]
>([])
const check = ref<QueryCheck>()
const evidence = ref<QueryEvidence>()
const testValues = reactive<Record<string, string>>({})
const editor = ref<{
  insert: (value: string) => void
  focusPosition: (position: number) => void
}>()
const tableInput = ref('')
const schema = ref<Record<string, QueryColumn[]>>({})
const schemaBusy = ref(false)
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
watch(aiAction, () => {
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
    definition.value = clone(query.draft)
    if (source.value) definition.value.dialect = source.value.dialect
    editorMode.value = query.draft.builder && !query.draft.builder.sql_mode ? 'config' : 'sql'
    builderKey.value++
    category.value = query.category
    baseline.value = currentState.value
    versions.value = related.history
    references.value = related.refs
    schema.value = {}
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
  definition.value = next
  editorMode.value = next.builder && !next.builder.sql_mode ? 'config' : 'sql'
  builderKey.value++
  for (const key of Object.keys(testValues)) delete testValues[key]
  category.value = nextCategory
  versions.value = []
  references.value = []
  schema.value = {}
  baseline.value = copied ? '' : currentState.value
  activePanel.value = 'check'
  invalidate()
}
async function newQuery() {
  if (!(await canLeave())) return
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
  activePanel.value = 'check'
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
  const parameters = clone(testValues)
  busy.value = 'preview'
  activePanel.value = 'test'
  lastError.value = ''
  try {
    const result = await api.preview({
      query_id: selected.value!.id,
      definition: clone(definition.value),
      parameters,
    })
    if (state !== currentState.value || JSON.stringify(parameters) !== JSON.stringify(testValues))
      return
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
    activePanel.value = 'versions'
  } catch (error) {
    message.error(errorText(error))
  } finally {
    busy.value = ''
  }
}
async function archive() {
  if (!selected.value || !(await canLeave())) return
  busy.value = 'archive'
  try {
    await api.archive(selected.value.id)
    baseline.value = currentState.value
    busy.value = ''
    await newQuery()
    await load()
    message.success(t('query.archived'))
  } catch (error) {
    message.error(errorText(error))
  } finally {
    busy.value = ''
  }
}
function addTable() {
  const table = tableInput.value.trim()
  if (!/^[A-Za-z_][A-Za-z0-9_]*$/.test(table)) {
    message.warning(t('query.tableNameHint'))
    return
  }
  if (!definition.value.tables.includes(table)) definition.value.tables.push(table)
  tableInput.value = ''
  void loadSchema(table)
}
async function loadSchema(table: string) {
  const sequence = selectSequence
  schemaBusy.value = true
  try {
    const result = await api.schema(table)
    if (sequence !== selectSequence || !definition.value.tables.includes(table)) return
    schema.value[table] = result.columns
    message.success(t('query.schemaLoaded', [table, result.columns.length]))
    if (!selected.value && !definition.value.sql)
      definition.value.dialect = result.dialect as QueryDefinition['dialect']
  } catch (error) {
    message.error(errorText(error))
  } finally {
    schemaBusy.value = false
  }
}
function removeTable(table: string) {
  definition.value.tables = definition.value.tables.filter((item) => item !== table)
  delete schema.value[table]
}
function addParameter() {
  definition.value.parameters.push({ name: '', label: '', type: 'string' })
}
async function assist() {
  const state = currentState.value
  const action = aiAction.value
  busy.value = 'assist'
  try {
    const result = await api.assist(
      action,
      clone(definition.value),
      instruction.value,
      lastError.value,
    )
    if (state !== currentState.value) return
    aiResult.value = result
    if (result.warnings.length) message.warning(result.warnings.join('；'), 8)
  } catch (error) {
    message.error(errorText(error))
  } finally {
    busy.value = ''
  }
}
function updateBuilder(value: QueryBuilder) {
  definition.value.builder = { ...value, sql_mode: false }
  definition.value.sql = ''
}
function applyBuilder(value: QueryDefinition) {
  definition.value = value
}
function updateSQL(value: string) {
  if (definition.value.sql === value) return
  definition.value.sql = value
  // 手工 SQL 和配置分别保留，切回配置时由用户明确生成，避免丢失已选字段。
  if (definition.value.builder) definition.value.builder.sql_mode = true
}
function adoptAI() {
  if (!aiResult.value) return
  if (definition.value.builder) definition.value.builder.sql_mode = true
  editorMode.value = 'sql'
  const result = clone(aiResult.value)
  definition.value.sql = result.sql
  definition.value.parameters = result.parameters
  aiResult.value = undefined
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
  definition.value = clone(version.definition)
  if (source.value) definition.value.dialect = source.value.dialect
  editorMode.value =
    definition.value.builder && !definition.value.builder.sql_mode ? 'config' : 'sql'
  builderKey.value++
  schema.value = {}
  for (const key of Object.keys(testValues)) delete testValues[key]
  activePanel.value = 'check'
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
    }
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
      <a-button :disabled="!!busy" @click="editPrompts">
        <template #icon><SettingOutlined /></template>
        {{ t('query.aiPrompts') }}
      </a-button>
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
      <main class="query-main panel">
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
          <a-button type="primary" :loading="busy === 'save'" :disabled="!!busy" @click="save">
            <template #icon><SaveOutlined /></template>
            {{ t('query.save') }}
          </a-button>
        </div>
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
          <a-radio-group v-model:value="editorMode" button-style="solid">
            <a-radio-button value="config">{{ t('query.builder.mode') }}</a-radio-button>
            <a-radio-button value="sql">{{ t('query.builder.sqlMode') }}</a-radio-button>
          </a-radio-group>
          <template v-if="editorMode === 'config'">
            <p
              v-if="definition.sql && (!definition.builder || definition.builder.sql_mode)"
              class="memory-help builder-notice"
            >
              {{ t('query.builder.convertHint') }}
            </p>
            <QueryBuilderEditor
              :key="`query-builder-${builderKey}`"
              :definition="definition"
              :disabled="!!busy || !source"
              @apply="applyBuilder"
              @update:builder="updateBuilder"
            />
          </template>
          <QuerySqlEditor
            v-else
            ref="editor"
            class="sql-template"
            :model-value="definition.sql"
            @update:model-value="updateSQL"
            @run="preview"
          />
          <a-collapse v-if="definition.sql" class="builder-notice">
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
          <div class="parameter-heading">
            <strong>{{ t('query.parameterDefinition') }}</strong>
            <a-button v-if="editorMode === 'sql'" size="small" @click="addParameter">
              <template #icon><PlusOutlined /></template>
              {{ t('query.addParameter') }}
            </a-button>
          </div>
          <div
            v-for="(parameter, index) in definition.parameters"
            :key="index"
            class="parameter-row"
            :class="{ 'parameter-row--generated': editorMode === 'config' }"
          >
            <a-input
              v-model:value="parameter.name"
              :readonly="editorMode === 'config'"
              :aria-label="t('query.parameterName')"
              :placeholder="t('query.parameterName')"
            />
            <a-input
              v-model:value="parameter.label"
              :readonly="editorMode === 'config'"
              :aria-label="t('query.parameterLabel')"
              :placeholder="t('query.parameterLabel')"
            />
            <a-select
              v-model:value="parameter.type"
              :disabled="editorMode === 'config'"
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
            <a-button
              size="small"
              v-if="editorMode === 'sql'"
              :disabled="!parameter.name"
              @click="editor?.insert(':' + parameter.name)"
            >
              {{ t('query.insert') }}
            </a-button>
            <a-button
              v-if="editorMode === 'sql'"
              size="small"
              danger
              @click="definition.parameters.splice(index, 1)"
            >
              {{ t('query.remove') }}
            </a-button>
          </div>
        </a-form>
        <a-tabs v-model:active-key="activePanel" class="result-panels">
          <a-tab-pane key="check" :tab="t('query.checkResult')">
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
                @click="editor?.focusPosition(issue.position)"
              >
                {{ issue.message }}
              </button>
              <p v-if="resultColumns.length" class="output-columns">
                {{ t('query.outputColumns') }}：{{ resultColumns.join('、') }}
              </p>
              <p class="muted">{{ t('query.checkScope') }}</p>
            </template>
            <a-empty v-else :description="t('query.checkEmpty')" />
          </a-tab-pane>
          <a-tab-pane key="test" :tab="t('query.test')">
            <p class="memory-help">{{ t('query.manualTestHint') }}</p>
            <div class="test-parameters">
              <a-form-item
                v-for="parameter in definition.parameters"
                :key="parameter.name"
                :label="parameter.label || parameter.name"
              >
                <a-input
                  v-model:value="testValues[parameter.name]"
                  :placeholder="t('query.testValueHint')"
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
            <a-empty v-else :description="t('query.previewEmpty')" />
          </a-tab-pane>
          <a-tab-pane key="versions" :tab="t('query.versions')">
            <div class="panel-actions">
              <a-button
                type="primary"
                :loading="busy === 'publish'"
                :disabled="!!busy || !selected || dirty || !selected.tested_hash"
                @click="publish"
              >
                {{ t('query.publish') }}
              </a-button>
              <span class="muted">{{ t('query.publishHint') }}</span>
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
          </a-tab-pane>
          <a-tab-pane key="references" :tab="t('query.references')">
            <a-empty v-if="!references.length" :description="t('query.noReferences')" />
            <div
              v-for="reference in references"
              :key="reference.module + reference.id"
              class="reference-row"
            >
              <strong>{{ reference.process_type }}</strong>
              <p>{{ reference.rule_content }}</p>
            </div>
            <a-popconfirm v-if="selected" :title="t('query.archiveConfirm')" @confirm="archive">
              <a-button danger :disabled="!!busy">{{ t('query.archive') }}</a-button>
            </a-popconfirm>
          </a-tab-pane>
        </a-tabs>
      </main>
      <aside class="query-assistant panel">
        <div class="section-heading">
          <RobotOutlined v-if="busy !== 'assist'" />
          <strong>{{ t('query.assistant') }}</strong>
        </div>
        <a-tabs>
          <a-tab-pane key="schema" :tab="t('query.schema')">
            <div class="memory-input-action">
              <a-input
                v-model:value="tableInput"
                :placeholder="t('query.tableName')"
                @press-enter="addTable"
              />
              <a-button type="primary" @click="addTable">{{ t('query.add') }}</a-button>
            </div>
            <p class="muted">{{ t('query.tableScopeHint') }}</p>
            <a-spin :spinning="schemaBusy">
              <div v-for="table in definition.tables" :key="table" class="schema-table">
                <div class="schema-heading">
                  <a-button type="link" size="small" @click="loadSchema(table)">
                    {{ table }}
                  </a-button>
                  <a-button type="text" size="small" danger @click="removeTable(table)">
                    {{ t('query.remove') }}
                  </a-button>
                </div>
                <button
                  v-for="column in schema[table] || []"
                  :key="column.key"
                  class="schema-column"
                  @click="editor?.insert(column.label)"
                >
                  <span>{{ column.label }}</span>
                  <small>{{ column.data_type }}</small>
                </button>
              </div>
            </a-spin>
          </a-tab-pane>
          <a-tab-pane key="ai" :tab="t('query.ai')">
            <a-select v-model:value="aiAction" :disabled="!!busy" style="width: 100%">
              <a-select-option
                v-for="action in ['generate', 'explain', 'repair']"
                :key="action"
                :value="action"
              >
                {{ t('query.action.' + action) }}
              </a-select-option>
            </a-select>
            <a-textarea
              v-model:value="instruction"
              :rows="5"
              :maxlength="8000"
              :placeholder="t('query.aiInstruction')"
              class="ai-instruction"
            />
            <a-button
              type="primary"
              block
              :loading="busy === 'assist'"
              :disabled="!!busy || !source"
              @click="assist"
            >
              <template #icon><RobotOutlined /></template>
              {{ t('query.runAI') }}
            </a-button>
            <p class="muted">{{ t('query.aiHint') }}</p>
            <template v-if="aiResult">
              <div class="ai-explanation">{{ aiResult.explanation }}</div>
              <pre v-if="aiResult.sql" class="ai-sql">{{ aiResult.sql }}</pre>
              <a-button v-if="aiAction !== 'explain' && aiResult.sql" block @click="adoptAI">
                {{ t('query.adoptAI') }}
              </a-button>
            </template>
          </a-tab-pane>
        </a-tabs>
      </aside>
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
  grid-template-columns: 240px minmax(380px, 1fr) 280px;
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
  padding: 16px 16px 0;
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
.definition-form {
  padding: 20px 20px 0;
}
.parameter-heading {
  margin: 20px 0 12px;
}
.parameter-row {
  display: grid;
  grid-template-columns: minmax(75px, 1fr) minmax(75px, 1fr) 100px auto auto;
  gap: 8px;
  margin-bottom: 10px;
}
.parameter-row--generated {
  grid-template-columns: minmax(75px, 1fr) minmax(75px, 1fr) 100px;
}
.result-panels {
  padding: 4px 20px 16px;
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
    grid-template-columns: 260px minmax(480px, 1fr) 320px;
  }
}
@media (max-width: 1380px) {
  .workspace-layout {
    grid-template-columns: 240px minmax(0, 1fr);
  }
  .query-assistant {
    grid-column: 2;
  }
  .parameter-row {
    grid-template-columns: 1fr 1fr 100px;
  }
  .parameter-row .ant-btn {
    justify-self: start;
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
  .definition-form {
    padding: 16px;
  }
  .result-panels {
    padding: 0 16px 16px;
  }
  .parameter-row,
  .test-parameters {
    grid-template-columns: 1fr 1fr;
  }
  .draft-meta {
    padding: 12px;
  }
  h1 {
    font-size: 20px;
  }
}
</style>
