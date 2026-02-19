<script setup lang="ts">
const { t } = useI18n()

const props = defineProps<{
  open: boolean
  rule?: {
    id?: string
    rule_content: string
    rule_scope: string
    priority: number
    process_type: string
  } | null
}>()

const emit = defineEmits<{
  close: []
  save: [rule: any]
}>()

const form = ref({
  rule_content: '',
  rule_scope: 'default_on',
  priority: 50,
  process_type: '',
})

watch(() => props.rule, (val) => {
  if (val) {
    form.value = { ...val }
  } else {
    form.value = { rule_content: '', rule_scope: 'default_on', priority: 50, process_type: '' }
  }
}, { immediate: true })

const scopeOptions = computed(() => [
  { value: 'mandatory', label: t('ruleEditor.mandatory') },
  { value: 'default_on', label: t('ruleEditor.defaultOn') },
  { value: 'default_off', label: t('ruleEditor.defaultOff') },
])

// Process types from mock data
const { mockProcessAuditConfigs } = useMockData()
const processTypes = computed(() =>
  mockProcessAuditConfigs.map(c => c.process_type)
)

const handleSave = () => {
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
    :width="520"
  >
    <a-form layout="vertical" style="margin-top: 16px;">
      <a-form-item :label="t('ruleEditor.processType')">
        <a-select
          v-model:value="form.process_type"
          :placeholder="t('ruleEditor.processTypePlaceholder')"
          size="large"
          allow-clear
        >
          <a-select-option v-for="pt in processTypes" :key="pt" :value="pt">{{ pt }}</a-select-option>
        </a-select>
      </a-form-item>
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
      <a-form-item :label="t('ruleEditor.priority')">
        <a-slider v-model:value="form.priority" :min="0" :max="100" />
        <div style="display: flex; justify-content: space-between; font-size: 12px; color: var(--color-text-tertiary);">
          <span>{{ t('ruleEditor.priorityLow') }}</span>
          <span>{{ t('ruleEditor.priorityCurrent') }}: {{ form.priority }}</span>
          <span>{{ t('ruleEditor.priorityHigh') }}</span>
        </div>
      </a-form-item>
    </a-form>
  </a-modal>
</template>
