<script setup lang="ts">
import { CheckCircleOutlined, CloseCircleOutlined, LockOutlined } from '@ant-design/icons-vue'

interface RuleResult {
  rule_id: string
  rule_name?: string
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
  <div class="rule-list">
    <div
      v-for="item in rules"
      :key="item.rule_id"
      class="rule-item"
      :class="{
        'rule-item--pass': item.passed,
        'rule-item--fail': !item.passed,
        'rule-item--expanded': expandedKeys.includes(item.rule_id),
      }"
      @click="toggle(item.rule_id)"
    >
      <div class="rule-item-header">
        <div class="rule-item-status">
          <CheckCircleOutlined v-if="item.passed" style="color: #10b981; font-size: 18px;" />
          <CloseCircleOutlined v-else style="color: #ef4444; font-size: 18px;" />
        </div>
        <div class="rule-item-content">
          <span class="rule-item-name">{{ item.rule_name || item.content || item.rule_id }}</span>
          <span v-if="item.is_locked" class="rule-locked-tag">
            <LockOutlined /> 强制
          </span>
        </div>
      </div>
      <transition name="expand">
        <div v-if="expandedKeys.includes(item.rule_id)" class="rule-item-reasoning">
          {{ item.reasoning }}
        </div>
      </transition>
    </div>
  </div>
</template>

<style scoped>
.rule-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.rule-item {
  padding: 12px 16px;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border-light);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.rule-item:hover {
  background: var(--color-bg-hover);
}

.rule-item--pass {
  border-left: 3px solid #10b981;
}

.rule-item--fail {
  border-left: 3px solid #ef4444;
  background: rgba(239, 68, 68, 0.03);
}

.rule-item-header {
  display: flex;
  align-items: center;
  gap: 12px;
}

.rule-item-status {
  flex-shrink: 0;
}

.rule-item-content {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.rule-item-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-primary);
}

.rule-locked-tag {
  font-size: 10px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: var(--radius-full);
  background: var(--color-danger-bg);
  color: var(--color-danger);
  display: inline-flex;
  align-items: center;
  gap: 3px;
}

.rule-item-reasoning {
  margin-top: 10px;
  padding: 10px 14px;
  background: var(--color-bg-page);
  border-radius: var(--radius-sm);
  font-size: 13px;
  line-height: 1.6;
  color: var(--color-text-secondary);
}

.expand-enter-active,
.expand-leave-active {
  transition: all 0.2s ease;
  overflow: hidden;
}

.expand-enter-from,
.expand-leave-to {
  opacity: 0;
  max-height: 0;
  margin-top: 0;
  padding: 0 14px;
}
</style>
