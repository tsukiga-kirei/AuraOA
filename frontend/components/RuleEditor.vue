<script setup lang="ts">
const { t } = useI18n()

const props = defineProps<{
  open: boolean
  rule?: {
    id?: string
    rule_content: string
    rule_scope: string
    related_flow?: boolean
  } | null
}>()

const emit = defineEmits<{
  close: []
  save: [rule: any]
}>()

const form = ref({
  rule_content: '',
  rule_scope: 'default_on',
  related_flow: false,
})

watch(() => props.rule, (val) => {
  if (val) {
    form.value = { rule_content: val.rule_content, rule_scope: val.rule_scope, related_flow: (val as any).related_flow ?? false }
  } else {
    form.value = { rule_content: '', rule_scope: 'default_on', related_flow: false }
  }
}, { immediate: true })

const scopeOptions = computed(() => [
  { value: 'mandatory', label: t('ruleEditor.mandatory') },
  { value: 'default_on', label: t('ruleEditor.defaultOn') },
  { value: 'default_off', label: t('ruleEditor.defaultOff') },
])

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
    </a-form>
  </a-modal>
</template>
