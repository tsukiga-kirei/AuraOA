<script setup lang="ts">
import { useAuditDataQueryApi } from '~/composables/useAuditDataQueryApi'
import { message } from 'ant-design-vue'
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
const initialQueryId = ref<string>()
const workspace = ref<{ canLeave: () => Promise<boolean> }>()
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
  return reference.bindings[name] || { source: 'field', field: '' }
}
function changeBinding(index: number, name: string, patch: Partial<QueryBinding>) {
  const reference = props.modelValue[index]
  const old = binding(reference, name)
  const next = patch.source ? { source: patch.source, field: '', value: '', days_offset: 0 } : old
  changeReference(index, { bindings: { ...reference.bindings, [name]: { ...next, ...patch } } })
}
function remove(index: number) {
  const items = clone(props.modelValue)
  items.splice(index, 1)
  emit('update:modelValue', items)
}
function openLibrary(id?: string) {
  initialQueryId.value = id
  libraryOpen.value = true
}
async function closeLibrary() {
  if (!workspace.value || (await workspace.value.canLeave())) libraryOpen.value = false
}
async function useVersion(query: AuditDataQuery, version: QueryVersion) {
  if (workspace.value && !(await workspace.value.canLeave())) return
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
    bindings[parameter.name] = clone(
      previous?.bindings[parameter.name] || { source: 'field', field: '' },
    )
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
            {{ t('query.openWorkspace') }}
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
        <span class="binding-label">
          {{ parameter.label }}
          <small>:{{ parameter.name }}</small>
        </span>
        <a-select
          :value="binding(reference, parameter.name).source"
          :disabled="!canManage"
          :aria-label="t('query.parameterSource')"
          @change="
            (value) =>
              changeBinding(index, parameter.name, { source: value as QueryBinding['source'] })
          "
        >
          <a-select-option value="field">{{ t('query.source.field') }}</a-select-option>
          <a-select-option value="system">{{ t('query.source.system') }}</a-select-option>
          <a-select-option value="constant">{{ t('query.source.constant') }}</a-select-option>
        </a-select>
        <a-select
          v-if="binding(reference, parameter.name).source === 'field'"
          :value="binding(reference, parameter.name).field"
          :disabled="!canManage"
          :options="fieldOptions"
          show-search
          :placeholder="t('query.selectField')"
          @change="(value) => changeBinding(index, parameter.name, { field: String(value) })"
        />
        <a-select
          v-else-if="binding(reference, parameter.name).source === 'system'"
          :value="binding(reference, parameter.name).value"
          :disabled="!canManage"
          @change="(value) => changeBinding(index, parameter.name, { value: String(value) })"
        >
          <a-select-option
            v-for="value in ['process_id', 'current_date', 'year_start', 'month_start']"
            :key="value"
            :value="value"
          >
            {{ t('query.system.' + value) }}
          </a-select-option>
        </a-select>
        <a-input
          v-else
          :value="binding(reference, parameter.name).value"
          :disabled="!canManage"
          :placeholder="t('query.constant')"
          @update:value="(value) => changeBinding(index, parameter.name, { value: String(value) })"
        />
        <a-input-number
          v-if="parameter.type === 'date'"
          :value="binding(reference, parameter.name).days_offset || 0"
          :min="-3660"
          :max="3660"
          :disabled="!canManage"
          :addon-before="t('query.daysOffset')"
          @update:value="
            (value) => changeBinding(index, parameter.name, { days_offset: Number(value) })
          "
        />
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
    <a-drawer
      :open="libraryOpen"
      :title="t('query.title')"
      width="95vw"
      :mask-closable="false"
      :keyboard="false"
      destroy-on-close
      @close="closeLibrary"
    >
      <AuditDataQueryWorkspace
        ref="workspace"
        allow-use
        :initial-query-id="initialQueryId"
        @use-version="useVersion"
      />
    </a-drawer>
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
  grid-template-columns: 140px 120px minmax(160px, 1fr);
  gap: 10px;
  align-items: center;
  margin-bottom: 10px;
}
.binding-label {
  font-size: 13px;
  color: var(--color-text-secondary);
}
.binding-label small {
  display: block;
  color: var(--color-text-tertiary);
  overflow-wrap: anywhere;
}
.test-process {
  margin: 8px 0;
}
.query-binding :deep(.ant-input-number-group-wrapper) {
  grid-column: 3;
}
@media (max-width: 760px) {
  .query-binding {
    grid-template-columns: 1fr 1fr;
  }
  .binding-label {
    grid-column: 1/-1;
  }
  .query-binding :deep(.ant-input-number-group-wrapper) {
    grid-column: 1/-1;
  }
}
</style>
