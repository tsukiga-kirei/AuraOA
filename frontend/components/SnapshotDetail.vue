<script setup lang="ts">
import { CheckCircleFilled, CloseCircleFilled } from '@ant-design/icons-vue'

interface RuleDetail {
  rule_id: string
  passed: boolean
  reasoning: string
}

interface Snapshot {
  snapshot_id: string
  process_id: string
  ai_reasoning: string
  audit_result: {
    recommendation: string
    details: RuleDetail[]
  }
  user_feedback?: {
    adopted: boolean
    action_taken: string
  }
  created_at: string
}

defineProps<{
  snapshot: Snapshot | null
}>()
</script>

<template>
  <a-drawer
    :open="!!snapshot"
    title="审核快照详情"
    width="600"
    @close="$emit('close')"
  >
    <template v-if="snapshot">
      <a-descriptions :column="1" bordered size="small">
        <a-descriptions-item label="快照 ID">{{ snapshot.snapshot_id }}</a-descriptions-item>
        <a-descriptions-item label="流程 ID">{{ snapshot.process_id }}</a-descriptions-item>
        <a-descriptions-item label="审核建议">
          <a-tag :color="snapshot.audit_result.recommendation === 'approve' ? 'green' : snapshot.audit_result.recommendation === 'reject' ? 'red' : 'orange'">
            {{ snapshot.audit_result.recommendation }}
          </a-tag>
        </a-descriptions-item>
        <a-descriptions-item label="用户采纳">
          <template v-if="snapshot.user_feedback">
            <a-tag :color="snapshot.user_feedback.adopted ? 'green' : 'default'">
              {{ snapshot.user_feedback.adopted ? '已采纳' : '未采纳' }}
            </a-tag>
            <span v-if="snapshot.user_feedback.action_taken"> · {{ snapshot.user_feedback.action_taken }}</span>
          </template>
          <span v-else style="color: #999;">暂无反馈</span>
        </a-descriptions-item>
        <a-descriptions-item label="审核时间">{{ snapshot.created_at }}</a-descriptions-item>
      </a-descriptions>

      <a-divider>规则详情</a-divider>
      <a-list :data-source="snapshot.audit_result.details" size="small">
        <template #renderItem="{ item }">
          <a-list-item>
            <a-list-item-meta>
              <template #avatar>
                <CheckCircleFilled v-if="item.passed" style="color: #52c41a;" />
                <CloseCircleFilled v-else style="color: #ff4d4f;" />
              </template>
              <template #title>{{ item.rule_id }}</template>
              <template #description>{{ item.reasoning }}</template>
            </a-list-item-meta>
          </a-list-item>
        </template>
      </a-list>

      <a-divider>AI 推理过程</a-divider>
      <pre style="white-space: pre-wrap; background: #fafafa; padding: 12px; border-radius: 6px; font-size: 13px;">{{ snapshot.ai_reasoning }}</pre>
    </template>
  </a-drawer>
</template>
