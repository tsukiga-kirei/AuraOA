<script setup lang="ts">
import { useAuditDataQueryApi } from '~/composables/useAuditDataQueryApi'
import { message } from 'ant-design-vue'
import QueryValueSourceEditor from './QueryValueSourceEditor.vue'
import { appDayjs } from '~/utils/appTime'
import { DatabaseOutlined, PlusOutlined } from '@ant-design/icons-vue'
import type {
  QueryReference,
  QueryDefinition,
  AuditDataQuery,
  QueryVersion,
  QueryBinding,
  QueryEvidence,
} from '~/types/audit-data-query'
const props = defineProps<{
  modelValue: QueryReference[]
  fieldOptions: { label: string; value: string }[]
}>()
const emit = defineEmits<{ 'update:modelValue': [value: QueryReference[]] }>()
const { t } = useI18n()
const { menus } = useAuth()
const api = useAuditDataQueryApi()
const canManage = computed(() => menus.value.some((menu) => menu.path === '/admin/tenant/memory'))
const definitions = ref<Record<string, QueryDefinition>>({})
const libraryOpen = ref(false)
const libraryItems = ref<AuditDataQuery[]>([])
const libraryLoading = ref(false)
const libraryKeyword = ref('')
const selectedLibraryId = ref('')
const pickerVersions = ref<QueryVersion[]>([])
const pickerVersionNo = ref<number>()
const versionsLoading = ref(false)
const pickerError = ref('')
let pickerSequence = 0
const filteredLibrary = computed(() => {
  const keyword = libraryKeyword.value.trim().toLowerCase()
  return libraryItems.value.filter(
    (item) => !keyword || `${item.name} ${item.category}`.toLowerCase().includes(keyword),
  )
})
const pickerQuery = computed(() =>
  libraryItems.value.find((item) => item.id === selectedLibraryId.value),
)
const pickerVersion = computed(() =>
  pickerVersions.value.find((item) => item.version_no === pickerVersionNo.value),
)
const processId = ref('')
const evidence = ref<QueryEvidence>()
const previewBusy = ref('')
const keyOf = (reference: QueryReference) => `${reference.query_id}:${reference.version_no}`
const clone = <T,>(value: T): T => JSON.parse(JSON.stringify(value))
const errorText = (error: unknown) => (error instanceof Error ? error.message : String(error))
let sequence = 0
watch(
  () => [canManage.value, props.modelValue.map(keyOf).join(',')],
  async () => {
    const seq = ++sequence
    if (!canManage.value) return
    try {
      const pairs = await Promise.all(
        props.modelValue.map(async (reference) => {
          const versions = await api.versions(reference.query_id)
          return [
            keyOf(reference),
            versions.find((version) => version.version_no === reference.version_no)?.definition,
          ] as const
        }),
      )
      if (seq !== sequence) return
      for (const [key, definition] of pairs) if (definition) definitions.value[key] = definition
    } catch (error) {
      message.error(errorText(error))
    }
  },
  { immediate: true },
)
watch(
  () => props.modelValue,
  () => {
    evidence.value = undefined
  },
  { deep: true },
)
watch(processId, () => {
  evidence.value = undefined
})
function changeReference(index: number, patch: Partial<QueryReference>) {
  const items = clone(props.modelValue)
  items[index] = { ...items[index], ...patch }
  emit('update:modelValue', items)
}
function binding(reference: QueryReference, name: string): QueryBinding {
  return (
    reference.bindings[name] ||
    definitions.value[keyOf(reference)]?.parameters.find((parameter) => parameter.name === name)
      ?.default_binding || { source: 'field', field: '' }
  )
}
function changeBinding(index: number, name: string, patch: Partial<QueryBinding>) {
  const reference = props.modelValue[index]
  const old = binding(reference, name)
  const next = patch.source ? { source: patch.source, field: '', value: '', days_offset: 0 } : old
  changeReference(index, { bindings: { ...reference.bindings, [name]: { ...next, ...patch } } })
}
function canInherit(reference: QueryReference, name: string) {
  const value = definitions.value[keyOf(reference)]?.parameters.find((p) => p.name === name)?.default_binding
  return !!value && value.source !== 'field'
}
function overrideBinding(index: number, name: string) {
  changeReference(index, { bindings: { ...props.modelValue[index].bindings, [name]: clone(binding(props.modelValue[index], name)) } })
}
function resetBinding(index: number, name: string) {
  const bindings = { ...props.modelValue[index].bindings }
  delete bindings[name]
  changeReference(index, { bindings })
}
function bindingSummary(value: QueryBinding) {
  const text = value.source === 'system' ? t('query.system.' + value.value) : value.value || '—'
  const offset = value.days_offset || 0
  return offset ? `${text} · ${t(offset < 0 ? 'query.daysEarlier' : 'query.daysLater', Math.abs(offset))}` : text
}
function remove(index: number) {
  const items = clone(props.modelValue)
  items.splice(index, 1)
  emit('update:modelValue', items)
}
async function openLibrary(id?: string) {
  const seq = ++pickerSequence
  libraryOpen.value = true
  libraryLoading.value = true
  libraryKeyword.value = ''
  libraryItems.value = []
  selectedLibraryId.value = ''
  pickerVersions.value = []
  pickerVersionNo.value = undefined
  versionsLoading.value = false
  pickerError.value = ''
  try {
    const items = await api.listAll()
    if (seq !== pickerSequence) return
    libraryItems.value = items.filter((item) => !item.archived && item.latest_version > 0)
    libraryLoading.value = false
    if (id && libraryItems.value.some((item) => item.id === id)) await selectLibraryEntry(id)
  } catch (error) {
    if (seq === pickerSequence) {
      pickerError.value = errorText(error)
      message.error(pickerError.value)
    }
  } finally {
    if (seq === pickerSequence) libraryLoading.value = false
  }
}
function closeLibrary() {
  ++pickerSequence
  libraryOpen.value = false
}
async function selectLibraryEntry(id: string) {
  const seq = ++pickerSequence
  selectedLibraryId.value = id
  pickerVersions.value = []
  pickerVersionNo.value = undefined
  pickerError.value = ''
  versionsLoading.value = true
  try {
    const versions = await api.versions(id)
    if (seq !== pickerSequence) return
    pickerVersions.value = [...versions].sort((a, b) => b.version_no - a.version_no)
    pickerVersionNo.value = pickerVersions.value[0]?.version_no
  } catch (error) {
    if (seq === pickerSequence) {
      pickerError.value = errorText(error)
      message.error(pickerError.value)
    }
  } finally {
    if (seq === pickerSequence) {
      versionsLoading.value = false
      libraryLoading.value = false
    }
  }
}
function selectPublishedVersion() {
  if (pickerQuery.value && pickerVersion.value) useVersion(pickerQuery.value, pickerVersion.value)
}
function formatTime(value: string) {
  const date = appDayjs(value)
  return date.isValid() ? date.format(t('query.dateTimeFormat')) : '-'
}
async function useVersion(query: AuditDataQuery, version: QueryVersion) {
  const existingIndex = props.modelValue.findIndex((reference) => reference.query_id === query.id)
  const previous = props.modelValue[existingIndex]
  if (previous?.version_no === version.version_no) {
    message.warning(t('query.alreadyReferenced'))
    return
  }
  if (existingIndex < 0 && props.modelValue.length >= 8) {
    message.warning(t('query.referenceLimit'))
    return
  }
  const bindings: Record<string, QueryBinding> = {}
  for (const parameter of version.definition.parameters) {
    const explicit = previous?.bindings[parameter.name]
    if (explicit) bindings[parameter.name] = clone(explicit)
    else if (!parameter.default_binding || parameter.default_binding.source === 'field')
      bindings[parameter.name] = clone(parameter.default_binding || { source: 'field', field: '' })
    // 固定值和系统值直接沿用发布版本，只有用户覆盖时才写入规则绑定。
  }
  const reference: QueryReference = {
    query_id: query.id,
    version_no: version.version_no,
    enabled: previous?.enabled ?? true,
    bindings,
  }
  definitions.value[keyOf(reference)] = clone(version.definition)
  const items = clone(props.modelValue)
  if (existingIndex >= 0) items[existingIndex] = reference
  else items.push(reference)
  emit('update:modelValue', items)
  libraryOpen.value = false
}
async function preview(reference: QueryReference) {
  if (!processId.value.trim()) {
    message.warning(t('query.processRequired'))
    return
  }
  const state = JSON.stringify(reference)
  const process = processId.value.trim()
  previewBusy.value = keyOf(reference)
  try {
    const result = await api.preview({
      query_id: reference.query_id,
      version_no: reference.version_no,
      process_id: process,
      bindings: clone(reference.bindings),
    })
    if (
      process === processId.value.trim() &&
      props.modelValue.some((item) => JSON.stringify(item) === state)
    )
      evidence.value = result
  } catch (error) {
    message.error(errorText(error))
  } finally {
    previewBusy.value = ''
  }
}
</script>
<template>
  <section class="rule-data-queries memory-surface">
    <div class="query-section-heading">
      <strong>
        <DatabaseOutlined />
        {{ t('query.ruleTitle') }}
      </strong>
      <a-button
        v-if="canManage"
        size="small"
        :disabled="modelValue.length >= 8"
        @click="openLibrary()"
      >
        <template #icon><PlusOutlined /></template>
        {{ t('query.selectLibrary') }}
      </a-button>
    </div>
    <p class="query-section-description">{{ t('query.ruleHint') }}</p>
    <a-alert v-if="!canManage" type="info" :message="t('query.permissionHint')" show-icon />
    <div v-for="(reference, index) in modelValue" :key="keyOf(reference)" class="query-reference">
      <div class="query-reference-heading">
        <strong>{{ definitions[keyOf(reference)]?.name || reference.query_id }}</strong>
        <a-space>
          <a-tag>{{ t('query.version', reference.version_no) }}</a-tag>
          <a-switch
            :checked="reference.enabled"
            :disabled="!canManage"
            size="small"
            @change="(value) => changeReference(index, { enabled: !!value })"
          />
          <a-button
            v-if="canManage"
            type="text"
            size="small"
            @click="openLibrary(reference.query_id)"
          >
            {{ t('query.changeVersion') }}
          </a-button>
          <a-button v-if="canManage" type="text" danger size="small" @click="remove(index)">
            {{ t('query.remove') }}
          </a-button>
        </a-space>
      </div>
      <p class="query-section-description">{{ definitions[keyOf(reference)]?.description }}</p>
      <div
        v-for="parameter in definitions[keyOf(reference)]?.parameters || []"
        :key="parameter.name"
        class="query-binding"
      >
        <span class="binding-label">{{ parameter.label }}</span>
        <div class="binding-content">
          <div v-if="canInherit(reference, parameter.name) && !reference.bindings[parameter.name]" class="inherited-binding">
            <div class="inherited-value">
              <span class="binding-origin">{{ t('query.inherited') }}</span>
              <span>{{ bindingSummary(binding(reference, parameter.name)) }}</span>
            </div>
            <a-button v-if="canManage" type="link" size="small" @click="overrideBinding(index, parameter.name)">{{ t('query.overrideBinding') }}</a-button>
          </div>
          <template v-else>
            <div v-if="canInherit(reference, parameter.name)" class="binding-override-heading">
              <span class="binding-origin">{{ t('query.ruleOverride') }}</span>
              <a-button v-if="canManage" type="link" size="small" @click="resetBinding(index, parameter.name)">{{ t('query.resetBinding') }}</a-button>
            </div>
            <QueryValueSourceEditor
              :model-value="binding(reference, parameter.name)"
              :type="parameter.type"
              :field-options="fieldOptions"
              :disabled="!canManage"
              @update:model-value="(value) => changeBinding(index, parameter.name, value)"
            />
          </template>
        </div>
      </div>
      <a-button
        v-if="canManage"
        size="small"
        :loading="previewBusy === keyOf(reference)"
        :disabled="!!previewBusy"
        @click="preview(reference)"
      >
        {{ t('query.previewWithProcess') }}
      </a-button>
    </div>
    <a-input
      v-if="modelValue.length && canManage"
      v-model:value="processId"
      :placeholder="t('query.processPlaceholder')"
      class="test-process"
    />
    <QueryEvidencePreview v-if="evidence" :evidence="evidence" />
    <a-modal
      :open="libraryOpen"
      :title="t('query.selectLibrary')"
      :width="880"
      :ok-text="t('query.confirmSelection')"
      :cancel-text="t('common.cancel')"
      :ok-button-props="{ disabled: !pickerVersion || libraryLoading || versionsLoading }"
      @cancel="closeLibrary"
      @ok="selectPublishedVersion"
    >
      <p class="query-section-description">{{ t('query.pickerHint') }}</p>
      <a-input
        v-model:value="libraryKeyword"
        :placeholder="t('query.search')"
        :aria-label="t('query.search')"
        allow-clear
      />
      <a-spin :spinning="libraryLoading">
        <div class="published-picker">
          <div class="published-list">
            <button
              v-for="item in filteredLibrary"
              :key="item.id"
              type="button"
              class="published-item"
              :class="{ selected: item.id === selectedLibraryId }"
              :aria-pressed="item.id === selectedLibraryId"
              @click="selectLibraryEntry(item.id)"
            >
              <strong>{{ item.name }}</strong>
              <span
                >{{ item.category || t('query.uncategorized') }} ·
                {{ t('query.version', item.latest_version) }}</span
              >
            </button>
            <p v-if="!filteredLibrary.length && !libraryLoading" class="query-section-description">
              {{ pickerError ? t('query.pickerFailed') : t('query.noPublishedEntries') }}
            </p>
            <a-button v-if="pickerError" size="small" @click="openLibrary()">{{
              t('query.retry')
            }}</a-button>
          </div>
          <a-spin :spinning="versionsLoading">
            <div v-if="pickerVersion" class="published-detail">
              <strong>{{ pickerVersion.definition.name }}</strong>
              <a-select v-model:value="pickerVersionNo" :aria-label="t('query.selectVersion')">
                <a-select-option
                  v-for="version in pickerVersions"
                  :key="version.version_no"
                  :value="version.version_no"
                >
                  {{ t('query.version', version.version_no) }} ·
                  {{ formatTime(version.created_at) }}
                </a-select-option>
              </a-select>
              <p class="query-section-description">{{ pickerVersion.definition.description }}</p>
              <strong v-if="pickerVersion.definition.parameters.length">{{
                t('query.valueBindings')
              }}</strong>
              <div
                v-for="parameter in pickerVersion.definition.parameters"
                :key="parameter.name"
                class="published-parameter"
              >
                <span>{{ parameter.label }}</span>
                <span>{{
                  t('query.source.' + (parameter.default_binding?.source || 'field'))
                }}</span>
              </div>
              <a-collapse>
                <a-collapse-panel key="sql" :header="t('query.sql')">
                  <pre class="published-sql">{{ pickerVersion.definition.sql }}</pre>
                </a-collapse-panel>
              </a-collapse>
            </div>
            <a-empty v-else :description="t('query.pickEntryHint')" />
          </a-spin>
        </div>
      </a-spin>
    </a-modal>
  </section>
</template>
<style scoped>
.rule-data-queries {
  margin-top: 24px;
  padding-top: 20px;
  border-top: 1px solid var(--color-border-light);
}
.query-section-heading,
.query-reference-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.query-section-heading strong {
  color: var(--color-text-primary);
}
.query-section-description {
  color: var(--color-text-tertiary);
  font-size: 12px;
  line-height: 1.7;
  margin: 8px 0 12px;
}
.query-reference {
  margin: 12px 0;
  padding: 16px;
  background: var(--color-bg-secondary);
  border: 1px solid var(--color-border);
  border-radius: 8px;
}
.query-binding {
  display: grid;
  grid-template-columns: 140px minmax(0, 1fr);
  gap: 10px;
  align-items: center;
  margin-bottom: 10px;
}
.binding-label {
  font-size: 13px;
  color: var(--color-text-secondary);
}
.binding-content {
  min-width: 0;
}
.inherited-binding,
.binding-override-heading {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.inherited-binding {
  min-height: 40px;
  padding: 4px 10px;
  background: var(--color-bg-card);
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
}
.inherited-value {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  overflow-wrap: anywhere;
  color: var(--color-text-primary);
  font-size: 13px;
}
.binding-origin {
  color: var(--color-text-tertiary);
  font-size: 12px;
}
.binding-override-heading {
  margin-bottom: 6px;
}
.test-process {
  margin: 8px 0;
}
/* 选择器只读展示发布版本，不嵌入维护工作区。 */
.published-picker {
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr);
  gap: 20px;
  margin-top: 16px;
  min-height: 300px;
}
.published-list {
  max-height: 460px;
  overflow-y: auto;
  border-right: 1px solid var(--color-border-light);
}
.published-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 100%;
  padding: 14px 12px;
  text-align: left;
  border: 0;
  border-bottom: 1px solid var(--color-border-light);
  border-left: 3px solid transparent;
  background: transparent;
  color: var(--color-text-primary);
  cursor: pointer;
}
.published-item span {
  color: var(--color-text-tertiary);
  font-size: 12px;
}
.published-item:hover {
  background: var(--color-bg-hover);
}
.published-item.selected {
  background: var(--color-primary-bg);
  border-left-color: var(--color-primary);
}
.published-item:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: -2px;
}
.published-detail {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: 460px;
  overflow-y: auto;
}
.published-parameter {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  color: var(--color-text-secondary);
  font-size: 13px;
}
.published-sql {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  color: var(--color-text-secondary);
}
@media (max-width: 760px) {
  .query-binding,
  .published-picker {
    grid-template-columns: 1fr;
  }
  .published-list {
    max-height: 220px;
    border-right: 0;
    border-bottom: 1px solid var(--color-border-light);
  }
}
</style>
