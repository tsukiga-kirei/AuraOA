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
import FormConfirmCard from './FormConfirmCard.vue'

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
    <template v-if="(toolCode === 'get_my_todo_list' || toolCode === 'list_my_todos' || toolCode === 'list_my_requests') && result && !error">
      <TodoListCard :result="result" />
    </template>

    <template v-else-if="toolCode === 'create_oa_draft' && result && !error">
      <div class="draft-success-card">
        <div class="draft-header">
          <CheckCircleOutlined style="color: #52c41a; font-size: 16px;" />
          <span class="draft-title">{{ t('chat.formConfirm.draftCreated', '流程草稿创建成功') }}</span>
          <a-tag v-if="result.request_id" color="blue">ID: {{ result.request_id }}</a-tag>
        </div>
        <div v-if="result.request_name" class="draft-name">{{ result.request_name }}</div>
        <p class="draft-tip">{{ t('chat.formConfirm.draftTip', '草稿已在泛微 OA 中建立，请前往 OA 核对表单联动计算并最终提交。') }}</p>
        <div class="draft-action" v-if="result.process_url">
          <a :href="result.process_url" target="_blank" rel="noopener noreferrer" class="oa-jump-btn">
            <LinkOutlined /> {{ t('chat.formConfirm.goToOa', '前往 OA 办理/提交') }}
          </a>
        </div>
      </div>
    </template>

    <template v-else-if="toolCode === 'execute_safe_query' && result && !error">
      <FormConfirmCard :data="{ variant: 'data_table', title: result.title, columns: result.columns, rows: result.rows }" />
    </template>

    <template v-else-if="toolCode === 'get_workflow_schema' && result && !error">
      <div class="schema-hint-pill">
        <FileSearchOutlined style="color: #1677ff; font-size: 13px;" />
        <span class="schema-hint-title">已自查流程字典：{{ result.workflow_name || ('ID: ' + result.workflow_id) }}</span>
        <span class="schema-hint-count">({{ result.main_fields?.length || 0 }} 个字段)</span>
      </div>
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

.draft-success-card {
  padding: 16px;
  background: var(--color-bg-subtle, #faf8f5);
  border: 1px solid #d9f7be;
  border-radius: 8px;
  margin: 6px 0;
}
.draft-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.draft-title {
  font-weight: 600;
  font-size: 14px;
  color: #274916;
}
.draft-name {
  font-weight: 500;
  font-size: 13px;
  color: #374151;
  margin-bottom: 6px;
}
.draft-tip {
  font-size: 12px;
  color: #6b7280;
  margin-bottom: 12px;
}
.oa-jump-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--color-primary, #9a7342);
  color: #fff;
  padding: 6px 16px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  text-decoration: none;
  transition: opacity 0.2s;
}
.oa-jump-btn:hover {
  opacity: 0.9;
  color: #fff;
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
.schema-hint-pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: #f0f5ff;
  border: 1px solid #d6e4ff;
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 12px;
  color: #1d39c4;
}
.schema-hint-count {
  color: #8c8c8c;
  font-size: 11px;
}
</style>
