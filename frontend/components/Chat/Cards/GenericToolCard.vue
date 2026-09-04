<script setup lang="ts">
import {
  FileSearchOutlined,
  SafetyCertificateOutlined,
  AuditOutlined,
  SendOutlined,
  LinkOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
} from '@ant-design/icons-vue'
import TodoListCard from './TodoListCard.vue'

const props = defineProps<{
  toolCode: string
  toolName: string
  arguments: Record<string, any>
  result: any
  error?: string
  executionMs: number
}>()

const { t } = useI18n()
const isExpanded = ref(true)
</script>

<template>
  <div class="generic-tool-wrapper">
    <!-- 专属卡片渲染 -->
    <template v-if="toolCode === 'get_my_todo_list' && result && !error">
      <TodoListCard :result="result" />
    </template>

    <!-- 通用/兜底工具卡片 -->
    <template v-else>
      <div class="tool-activity-card" :class="{ 'is-error': Boolean(error) }">
        <div class="tool-activity-header" @click="isExpanded = !isExpanded">
          <span class="tool-icon">
            <CheckCircleOutlined v-if="!error && result" style="color: #52c41a" />
            <CloseCircleOutlined v-else-if="error" style="color: #ff4d4f" />
            <AuditOutlined v-else style="color: #1890ff" />
          </span>
          <span class="tool-name">{{ toolName || toolCode }}</span>
          <span class="tool-cost" v-if="executionMs">{{ executionMs }}ms</span>
          <span class="tool-toggle-arrow">{{ isExpanded ? t('chat.toolCard.collapse') : t('chat.toolCard.expand') }}</span>
        </div>

        <div v-if="isExpanded" class="tool-activity-content">
          <div class="tool-section" v-if="arguments && Object.keys(arguments).length > 0">
            <div class="tool-section-label">{{ t('chat.toolCard.params') }}</div>
            <pre class="tool-json-block">{{ JSON.stringify(arguments, null, 2) }}</pre>
          </div>

          <div class="tool-section" v-if="error">
            <div class="tool-section-label is-error-label">{{ t('chat.toolCard.error') }}</div>
            <div class="tool-error-text">{{ error }}</div>
          </div>

          <div class="tool-section" v-else-if="result">
            <div class="tool-section-label">{{ t('chat.toolCard.result') }}</div>
            <pre class="tool-json-block">{{ JSON.stringify(result, null, 2) }}</pre>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<style scoped>
.generic-tool-wrapper {
  margin: 6px 0;
}
.tool-activity-card {
  border: 1px solid #ebe5df;
  border-radius: 8px;
  background: #faf8f5;
  overflow: hidden;
  font-size: 13px;
}
.tool-activity-card.is-error {
  border-color: #ffccc7;
  background: #fff2f0;
}
.tool-activity-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
  user-select: none;
  background: rgba(0, 0, 0, 0.02);
}
.tool-name {
  font-weight: 500;
  color: #374151;
}
.tool-cost {
  color: #9ca3af;
  font-size: 11px;
}
.tool-toggle-arrow {
  margin-left: auto;
  font-size: 12px;
  color: #8c8c8c;
}
.tool-activity-content {
  padding: 10px 12px;
  border-top: 1px solid rgba(0, 0, 0, 0.05);
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.tool-section-label {
  font-size: 11px;
  font-weight: 600;
  color: #6b7280;
  margin-bottom: 4px;
}
.tool-section-label.is-error-label {
  color: #ef4444;
}
.tool-error-text {
  color: #b91c1c;
  font-size: 12.5px;
}
.tool-json-block {
  background: #f3f4f6;
  padding: 8px;
  border-radius: 4px;
  font-size: 12px;
  max-height: 200px;
  overflow-y: auto;
  margin: 0;
}
</style>
