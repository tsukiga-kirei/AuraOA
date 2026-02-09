<script setup lang="ts">
import { CheckCircleFilled, CloseCircleFilled, EditFilled } from '@ant-design/icons-vue'

interface ChecklistResult {
  rule_id: string
  passed: boolean
  reasoning: string
}

interface AuditResult {
  trace_id: string
  process_id: string
  recommendation: 'approve' | 'reject' | 'revise'
  details: ChecklistResult[]
  ai_reasoning: string
}

defineProps<{
  result: AuditResult | null
  loading: boolean
}>()

const recommendationConfig = {
  approve: { color: '#52c41a', icon: CheckCircleFilled, text: '建议通过' },
  reject: { color: '#ff4d4f', icon: CloseCircleFilled, text: '建议驳回' },
  revise: { color: '#faad14', icon: EditFilled, text: '建议修改' },
}
</script>

<template>
  <div>
    <a-skeleton v-if="loading" active :paragraph="{ rows: 6 }" />
    <template v-else-if="result">
      <a-result
        :status="result.recommendation === 'approve' ? 'success' : result.recommendation === 'reject' ? 'error' : 'warning'"
        :title="recommendationConfig[result.recommendation]?.text"
        :sub-title="`Trace ID: ${result.trace_id}`"
      />

      <a-divider>规则审核详情</a-divider>
      <RuleList :rules="result.details" />

      <a-divider>AI 推理过程</a-divider>
      <a-typography-paragraph>
        <pre style="white-space: pre-wrap; background: #fafafa; padding: 12px; border-radius: 6px;">{{ result.ai_reasoning }}</pre>
      </a-typography-paragraph>
    </template>
    <a-empty v-else description="选择一条待办流程开始审核" />
  </div>
</template>
