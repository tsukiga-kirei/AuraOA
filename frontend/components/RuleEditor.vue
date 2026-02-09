<script setup lang="ts">
defineProps<{
  open: boolean
  rule?: {
    id?: string
    rule_content: string
    rule_scope: string
    priority: number
    process_type: string
  }
}>()

const emit = defineEmits<{
  close: []
  save: [rule: any]
}>()

const form = ref({
  rule_content: '',
  rule_scope: 'default_on',
  priority: 0,
  process_type: '',
})

watch(() => form.value, () => {}, { immediate: true })

const scopeOptions = [
  { value: 'mandatory', label: '强制执行' },
  { value: 'default_on', label: '默认开启' },
  { value: 'default_off', label: '默认关闭' },
]

const handleSave = () => {
  emit('save', { ...form.value })
}
</script>

<template>
  <a-modal :open="open" title="规则编辑器" @cancel="$emit('close')" @ok="handleSave">
    <a-form layout="vertical">
      <a-form-item label="流程类型">
        <a-input v-model:value="form.process_type" placeholder="如：采购审批" />
      </a-form-item>
      <a-form-item label="规则内容">
        <a-textarea v-model:value="form.rule_content" :rows="3" placeholder="如：合同金额不得超过预算的110%" />
      </a-form-item>
      <a-form-item label="规则级别">
        <a-select v-model:value="form.rule_scope" :options="scopeOptions" />
      </a-form-item>
      <a-form-item label="优先级">
        <a-input-number v-model:value="form.priority" :min="0" :max="100" />
      </a-form-item>
    </a-form>
  </a-modal>
</template>
