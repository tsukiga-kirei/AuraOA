<script setup lang="ts">
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  DownOutlined,
  UpOutlined,
} from '@ant-design/icons-vue'
import { useI18n } from '~/composables/useI18n'

export interface ResultRuleDetailItem {
  id?: string | number
  name: string
  passed: boolean
  reason: string
}

const props = withDefaults(defineProps<{
  title: string
  rules: ResultRuleDetailItem[]
  emptyText?: string
}>(), {
  emptyText: '',
})

const { t } = useI18n()
const expanded = ref(false)

// 未通过项优先展示；相同状态下保持后端原始顺序。
const sortedRules = computed(() => props.rules
  .map((rule, index) => ({ ...rule, originalIndex: index }))
  .sort((a, b) => Number(a.passed) - Number(b.passed) || a.originalIndex - b.originalIndex))

const totalCount = computed(() => props.rules.length)
const passedCount = computed(() => props.rules.filter(rule => rule.passed).length)
const failedCount = computed(() => totalCount.value - passedCount.value)
const canToggle = computed(() => passedCount.value > 0)
const visibleRules = computed(() => (
  expanded.value ? sortedRules.value : sortedRules.value.filter(rule => !rule.passed)
))

watch(() => props.rules, () => {
  expanded.value = false
})
</script>

<template>
  <section class="result-rule-details">
    <div class="result-rule-details__header">
      <h4 class="result-rule-details__title">{{ title }}</h4>
      <div class="result-rule-details__summary">
        <span class="result-rule-details__count">
          {{ t('resultDetails.total', [totalCount]) }}
        </span>
        <span class="result-rule-details__count result-rule-details__count--pass">
          {{ t('resultDetails.passed', [passedCount]) }}
        </span>
        <span class="result-rule-details__count result-rule-details__count--fail">
          {{ t('resultDetails.failed', [failedCount]) }}
        </span>
        <button
          v-if="canToggle"
          type="button"
          class="result-rule-details__toggle"
          :aria-expanded="expanded"
          @click="expanded = !expanded"
        >
          <span>{{ t(expanded ? 'resultDetails.collapsePassed' : 'resultDetails.expandPassed') }}</span>
          <UpOutlined v-if="expanded" />
          <DownOutlined v-else />
        </button>
      </div>
    </div>

    <div v-if="visibleRules.length" class="result-rule-details__list">
      <div
        v-for="(rule, index) in visibleRules"
        :key="rule.id ?? `${rule.name}-${rule.originalIndex}-${index}`"
        class="result-rule-details__item"
        :class="rule.passed ? 'result-rule-details__item--pass' : 'result-rule-details__item--fail'"
      >
        <div class="result-rule-details__status">
          <CheckCircleOutlined v-if="rule.passed" />
          <CloseCircleOutlined v-else />
        </div>
        <div class="result-rule-details__content">
          <div class="result-rule-details__name">{{ rule.name }}</div>
          <div v-if="rule.reason" class="result-rule-details__reason">{{ rule.reason }}</div>
        </div>
      </div>
    </div>
    <div v-else-if="totalCount === 0" class="result-rule-details__empty">{{ emptyText }}</div>
  </section>
</template>

<style scoped>
.result-rule-details {
  margin-bottom: 24px;
}

.result-rule-details__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 12px;
}

.result-rule-details__title {
  margin: 0;
  color: var(--color-text-primary);
  font-size: 14px;
  font-weight: 600;
  line-height: 1.5;
}

.result-rule-details__summary {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
  flex-wrap: wrap;
}

.result-rule-details__count {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 2px 8px;
  border-radius: var(--radius-full);
  background: var(--color-bg-hover);
  color: var(--color-text-secondary);
  font-size: 12px;
  line-height: 1.4;
  white-space: nowrap;
}

.result-rule-details__count--pass {
  background: var(--color-success-bg);
  color: var(--color-success);
}

.result-rule-details__count--fail {
  background: var(--color-danger-bg);
  color: var(--color-danger);
}

.result-rule-details__toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  min-height: 28px;
  padding: 3px 8px;
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-primary);
  cursor: pointer;
  font: inherit;
  font-size: 12px;
  white-space: nowrap;
}

.result-rule-details__toggle:hover {
  background: var(--color-primary-bg);
}

.result-rule-details__toggle:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}

.result-rule-details__list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.result-rule-details__item {
  display: flex;
  gap: 12px;
  padding: 12px 16px;
  border: 1px solid var(--color-border-light);
  border-left-width: 3px;
  border-radius: var(--radius-md);
}

.result-rule-details__item--pass {
  border-left-color: var(--color-success);
  background: var(--color-success-bg);
}

.result-rule-details__item--fail {
  border-left-color: var(--color-danger);
  background: var(--color-danger-bg);
}

.result-rule-details__status {
  flex-shrink: 0;
  padding-top: 1px;
  font-size: 18px;
}

.result-rule-details__item--pass .result-rule-details__status {
  color: var(--color-success);
}

.result-rule-details__item--fail .result-rule-details__status {
  color: var(--color-danger);
}

.result-rule-details__content {
  flex: 1;
  min-width: 0;
}

.result-rule-details__name {
  color: var(--color-text-primary);
  font-size: 14px;
  font-weight: 600;
  line-height: 1.5;
}

.result-rule-details__reason {
  margin-top: 4px;
  color: var(--color-text-secondary);
  font-size: 13px;
  line-height: 1.6;
  overflow-wrap: anywhere;
}

.result-rule-details__empty {
  padding: 12px;
  color: var(--color-text-tertiary);
  font-size: 13px;
  text-align: center;
}

@media (max-width: 640px) {
  .result-rule-details__header {
    align-items: flex-start;
    flex-direction: column;
    gap: 8px;
  }

  .result-rule-details__summary {
    justify-content: flex-start;
  }

  .result-rule-details__item {
    padding: 11px 12px;
  }
}
</style>
