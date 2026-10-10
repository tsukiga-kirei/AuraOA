<script setup lang="ts">
import { useAuditDataQueryApi } from '~/composables/useAuditDataQueryApi'
import { message, Modal } from 'ant-design-vue'
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
  max_rows: 20,
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
const sourceError = ref('')
const category = ref('')
const editorMode = ref('config')
const builderKey = ref(0)
const baseline = ref('')
const keyword = ref('')
const appliedKeyword = ref('')
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const loading = ref(false)
const loadError = ref('')
const busy = ref('')
const activePanel = ref('test')
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
    const result = await api.list(appliedKeyword.value, page.value, pageSize.value)
    if (sequence !== listSequence) return
    items.value = result.items
    total.value = result.total
    page.value = result.page
  } catch (error) {
    if (sequence === listSequence) loadError.value = errorText(error)
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
    editorMode.value = query.draft.builder ? 'config' : 'sql'
    builderKey.value++
    category.value = query.category
    baseline.value = JSON.stringify({ definition: query.draft, category: query.category })
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
async function newQuery(copy = false) {
  if (!(await canLeave())) return
  ++selectSequence
  const next = copy ? clone(definition.value) : makeDefinition()
  if (copy) next.name = t('query.copyName', next.name)
  selected.value = undefined
  if (source.value) next.dialect = source.value.dialect
  definition.value = next
  editorMode.value = next.builder ? 'config' : 'sql'
  builderKey.value++
  for (const key of Object.keys(testValues)) delete testValues[key]
  category.value = copy ? category.value : ''
  versions.value = []
  references.value = []
  schema.value = {}
  baseline.value = copy ? '' : currentState.value
  invalidate()
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
    if (state === currentState.value) check.value = result
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
    if (state === currentState.value) aiResult.value = result
  } catch (error) {
    message.error(errorText(error))
  } finally {
    busy.value = ''
  }
}
function updateBuilder(value: QueryBuilder) {
  definition.value.builder = value
  definition.value.sql = ''
}
function applyBuilder(value: QueryDefinition) {
  definition.value = value
}
function updateSQL(value: string) {
  definition.value.sql = value
  delete definition.value.builder
}
function adoptAI() {
  delete definition.value.builder
  editorMode.value = 'sql'
  if (!aiResult.value) return
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
  editorMode.value = definition.value.builder ? 'config' : 'sql'
  builderKey.value++
  schema.value = {}
  for (const key of Object.keys(testValues)) delete testValues[key]
  activePanel.value = 'test'
}
function search() {
  appliedKeyword.value = keyword.value.trim()
  page.value = 1
  void load()
}
function changePage(next: number, size: number) {
  page.value = size === pageSize.value ? next : 1
  pageSize.value = size
  void load()
}
onMounted(async () => {
  try {
    source.value = await api.source()
    definition.value.dialect = source.value.dialect
  } catch (error) {
    sourceError.value = errorText(error)
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
    <a-alert
      v-if="sourceError"
      type="error"
      :message="sourceError"
      show-icon
      class="source-error"
    />
    <div class="workspace-layout">
      <aside class="query-library panel">
        <div class="library-top">
          <strong>{{ t('query.library') }}</strong>
          <a-button size="small" type="primary" :disabled="!!busy" @click="newQuery()">
            <template #icon><PlusOutlined /></template>
            {{ t('query.new') }}
          </a-button>
        </div>
        <div class="memory-input-action">
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
        <a-alert v-if="loadError" type="error" :message="loadError" show-icon>
          <template #action>
            <a-button size="small" @click="load">{{ t('query.retry') }}</a-button>
          </template>
        </a-alert>
        <a-spin :spinning="loading">
          <div class="library-list">
            <button
              v-for="item in items"
              :key="item.id"
              class="library-item"
              :disabled="!!busy"
              :class="{ selected: selected?.id === item.id }"
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
            <a-empty v-if="!items.length && !loading" :description="t('query.empty')" />
          </div>
        </a-spin>
        <div class="pagination-wrapper">
          <a-pagination
            size="small"
            :current="page"
            :page-size="pageSize"
            :total="total"
            :page-size-options="['10', '20', '50']"
            show-size-changer
            @change="changePage"
          />
        </div>
      </aside>
      <main class="query-main panel">
        <div class="main-toolbar">
          <a-space wrap>
            <a-tag class="query-version-tag">
              {{
                selected?.latest_version
                  ? t('query.version', selected.latest_version)
                  : t('query.draft')
              }}
            </a-tag>
            <a-tag v-if="dirty" color="warning">{{ t('query.unsaved') }}</a-tag>
          </a-space>
          <a-space wrap>
            <a-button :disabled="!!busy" @click="newQuery(true)">
              <template #icon><CopyOutlined /></template>
              {{ t('query.copy') }}
            </a-button>
            <a-button :loading="busy === 'check'" :disabled="!!busy || !source" @click="checkSQL">
              <template #icon><CheckCircleOutlined /></template>
              {{ t('query.check') }}
            </a-button>
            <a-button :loading="busy === 'preview'" :disabled="!!busy || !source" @click="preview">
              <template #icon><PlayCircleOutlined /></template>
              {{ t('query.preview') }}
            </a-button>
            <a-button :loading="busy === 'save'" :disabled="!!busy" @click="save">
              <template #icon><SaveOutlined /></template>
              {{ t('query.save') }}
            </a-button>
            <a-button
              type="primary"
              :loading="busy === 'publish'"
              :disabled="!!busy || !selected || dirty || !selected.tested_hash"
              @click="publish"
            >
              {{ t('query.publish') }}
            </a-button>
          </a-space>
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
                  v-model:value="definition.max_rows"
                  :min="1"
                  :max="100"
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
            <a-alert
              v-if="!definition.builder && definition.sql"
              type="warning"
              :message="t('query.builder.convertHint')"
              show-icon
              class="builder-notice"
            />
            <QueryBuilderEditor
              :key="builderKey"
              :definition="definition"
              :disabled="!!busy || !source"
              @apply="applyBuilder"
              @update:builder="updateBuilder"
            />
            <a-collapse v-if="definition.sql" class="builder-notice"
              ><a-collapse-panel key="sql" :header="t('query.builder.generatedSQL')">
                <pre class="ai-sql">{{ definition.sql }}</pre>
              </a-collapse-panel></a-collapse
            >
          </template>
          <QuerySqlEditor
            v-else
            ref="editor"
            :model-value="definition.sql"
            @update:model-value="updateSQL"
            @run="preview"
          />
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
          <a-tab-pane key="test" :tab="t('query.test')">
            <a-alert type="info" :message="t('query.manualTestHint')" show-icon />
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
            <a-alert v-if="lastError" type="error" :message="lastError" show-icon />
            <QueryEvidencePreview v-if="evidence" :evidence="evidence" />
            <a-empty v-else-if="!lastError" :description="t('query.previewEmpty')" />
          </a-tab-pane>
          <a-tab-pane key="check" :tab="t('query.checkResult')">
            <template v-if="check">
              <a-alert
                :type="check.valid ? 'success' : 'error'"
                :message="t(check.valid ? 'query.checkPassed' : 'query.checkFailed')"
                show-icon
              />
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
          <a-tab-pane key="versions" :tab="t('query.versions')">
            <a-empty v-if="!versions.length" :description="t('query.noVersions')" />
            <div v-for="version in versions" :key="version.id" class="version-row">
              <div>
                <strong>{{ t('query.version', version.version_no) }}</strong>
                <span class="muted">· {{ version.created_at }}</span>
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
              <a-alert
                v-for="(warning, index) in aiResult.warnings"
                :key="index"
                type="warning"
                :message="warning"
                class="ai-warning"
              />
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
      <a-alert type="info" :message="t('query.promptHint')" show-icon />
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
.source-error {
  margin-bottom: 16px;
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
  grid-template-columns: 240px minmax(380px, 1fr) 280px;
  gap: 20px;
  align-items: start;
}
.panel {
  background: var(--color-bg-card);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-lg);
}
.query-library,
.query-assistant {
  padding: 16px;
}
.library-top,
.schema-heading,
.parameter-heading,
.version-row,
.main-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}
.library-top {
  margin-bottom: 16px;
  flex-wrap: wrap;
}
.library-list {
  min-height: 260px;
  margin: 12px -8px;
}
.library-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  text-align: left;
  width: 100%;
  border: 1px solid transparent;
  background: transparent;
  color: var(--color-text-primary);
  padding: 12px;
  border-radius: 8px;
  cursor: pointer;
  margin-bottom: 4px;
}
.library-item:hover {
  background: var(--color-bg-hover);
}
.library-item.selected {
  background: var(--color-primary-bg);
  border-color: var(--color-primary-ring);
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
.main-toolbar {
  padding: 16px;
  border-bottom: 1px solid var(--color-border-light);
  flex-wrap: wrap;
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
  background: var(--color-error-bg);
  color: var(--color-error);
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
.ai-warning {
  margin-top: 10px;
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
.pagination-wrapper {
  margin-top: 12px;
}
.query-library :deep(.ant-pagination) {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
@media (min-width: 1700px) {
  .workspace-layout {
    grid-template-columns: 260px minmax(480px, 1fr) 320px;
  }
}
@media (max-width: 1380px) {
  .workspace-layout {
    grid-template-columns: 220px minmax(0, 1fr);
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
  .main-toolbar {
    padding: 12px;
  }
  h1 {
    font-size: 20px;
  }
}
</style>
