<script setup lang="ts">
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

const scopeOptions = [
  { value: 'mandatory', label: '强制执行' },
  { value: 'default_on', label: '默认开启' },
  { value: 'default_off', label: '默认关闭' },
]

const processTypes = [
  '采购审批', '费用报销', '合同审批', '人事审批', '预算审批', '工程审批',
]

const handleSave = () => {
  emit('save', { ...form.value })
}
</script>

<template>
  <a-modal
    :open="open"
    :title="rule ? '编辑规则' : '新增规则'"
    @cancel="emit('close')"
    @ok="handleSave"
    :okText="'保存'"
    :cancelText="'取消'"
    :width="520"
  >
    <a-form layout="vertical" style="margin-top: 16px;">
      <a-form-item label="流程类型">
        <a-select
          v-model:value="form.process_type"
          placeholder="选择流程类型"
          size="large"
          allow-clear
        >
          <a-select-option v-for="t in processTypes" :key="t" :value="t">{{ t }}</a-select-option>
        </a-select>
      </a-form-item>
      <a-form-item label="规则内容">
        <a-textarea
          v-model:value="form.rule_content"
          :rows="3"
          placeholder="如：合同金额不得超过预算的110%"
          size="large"
        />
      </a-form-item>
      <a-form-item label="规则级别">
        <a-radio-group v-model:value="form.rule_scope" button-style="solid">
          <a-radio-button v-for="opt in scopeOptions" :key="opt.value" :value="opt.value">
            {{ opt.label }}
          </a-radio-button>
        </a-radio-group>
      </a-form-item>
      <a-form-item label="优先级">
        <a-slider v-model:value="form.priority" :min="0" :max="100" />
        <div style="display: flex; justify-content: space-between; font-size: 12px; color: var(--color-text-tertiary);">
          <span>低</span>
          <span>当前: {{ form.priority }}</span>
          <span>高</span>
        </div>
      </a-form-item>
    </a-form>
  </a-modal>
</template>
