<script setup lang="ts">
import { CheckCircleFilled, CloseCircleFilled, LockFilled } from '@ant-design/icons-vue'

interface RuleResult {
  rule_id: string
  passed: boolean
  reasoning: string
  is_locked?: boolean
  content?: string
}

defineProps<{
  rules: RuleResult[]
}>()

const expandedKeys = ref<string[]>([])

const toggle = (ruleId: string) => {
  const idx = expandedKeys.value.indexOf(ruleId)
  if (idx >= 0) {
    expandedKeys.value.splice(idx, 1)
  } else {
    expandedKeys.value.push(ruleId)
  }
}
</script>

<template>
  <a-list :data-source="rules" size="small">
    <template #renderItem="{ item }">
      <a-list-item style="cursor: pointer;" @click="toggle(item.rule_id)">
        <a-list-item-meta>
          <template #avatar>
            <CheckCircleFilled v-if="item.passed" style="color: #52c41a; font-size: 18px;" />
            <CloseCircleFilled v-else style="color: #ff4d4f; font-size: 18px;" />
          </template>
          <template #title>
            <span>{{ item.content || item.rule_id }}</span>
            <LockFilled v-if="item.is_locked" style="margin-left: 8px; color: #999; font-size: 12px;" />
          </template>
          <template #description>
            <div v-if="expandedKeys.includes(item.rule_id)" style="margin-top: 8px; padding: 8px; background: #f5f5f5; border-radius: 4px;">
              {{ item.reasoning }}
            </div>
          </template>
        </a-list-item-meta>
      </a-list-item>
    </template>
  </a-list>
</template>
