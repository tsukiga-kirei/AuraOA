<script setup lang="ts">
import { CheckCircleOutlined, CloseCircleOutlined } from '@ant-design/icons-vue'
import { useI18n } from '~/composables/useI18n'

const { t } = useI18n()

// props：定时任务执行历史记录列表
defineProps<{
  history: Array<{
    task_id: string
    success: boolean
    message: string
    item_count: number
    executed_at: string
  }>
}>()
</script>

<template>
  <div class="cron-history">
    <div v-if="history.length === 0" class="history-empty">
      {{ t('cron.history.noRecords') }}
    </div>
    <div v-for="item in history" :key="item.task_id + item.executed_at" class="history-item">
      <div class="history-status">
        <CheckCircleOutlined v-if="item.success" style="color: var(--color-success);" />
        <CloseCircleOutlined v-else style="color: var(--color-danger);" />
      </div>
      <div class="history-content">
        <div class="history-message">{{ item.message }}</div>
        <div class="history-meta">
          {{ item.executed_at }} · {{ t('cron.history.processed', [item.item_count]) }}
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.cron-history {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.history-empty {
  text-align: center;
  padding: 24px;
  color: var(--color-text-tertiary);
  font-size: 13px;
}

.history-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 14px;
  border-radius: var(--radius-md);
  background: var(--color-bg-page);
}

.history-status {
  font-size: 16px;
  flex-shrink: 0;
  padding-top: 1px;
}

.history-message {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-primary);
}

.history-meta {
  font-size: 12px;
  color: var(--color-text-tertiary);
  margin-top: 2px;
}
</style>
