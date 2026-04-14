<script setup lang="ts">
const { t } = useI18n()

// props：弹窗开关 / 待编辑的规则数据（新增时为 null）
const props = defineProps<{
  open: boolean
  rule?: {
    id?: string
    rule_content: string
    rule_scope: string
    related_flow?: boolean
  } | null
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
})

// 弹窗打开时初始化表单：编辑模式填充已有数据，新增模式重置为默认值
watch(() => props.open, (isOpen) => {
  if (isOpen) {
    if (props.rule) {
      form.value = {
        rule_content: props.rule.rule_content,
        rule_scope: props.rule.rule_scope,
        related_flow: (props.rule as any).related_flow ?? false,
      }
    } else {
      form.value = {
        rule_content: '',
        rule_scope: 'default_on',
        related_flow: false,
      }
    }
  }
})

// 规则生效范围选项：强制执行 / 默认开启 / 默认关闭
const scopeOptions = computed(() => [
  { value: 'mandatory', label: t('ruleEditor.mandatory') },
  { value: 'default_on', label: t('ruleEditor.defaultOn') },
  { value: 'default_off', label: t('ruleEditor.defaultOff') },
])

// 提交表单数据给父组件处理
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
