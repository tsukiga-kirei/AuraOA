<script setup lang="ts">
import {
  UnorderedListOutlined,
  CheckCircleOutlined,
  ClockCircleOutlined,
  LinkOutlined,
} from '@ant-design/icons-vue'

const props = defineProps<{
  result: any
}>()

const { t } = useI18n()

const list = computed(() => {
  if (Array.isArray(props.result)) return props.result
  if (props.result?.items && Array.isArray(props.result.items)) return props.result.items
  return []
})
</script>

<template>
  <div class="oa-card todo-list-card">
    <div class="oa-card-header">
      <span class="oa-card-icon"><UnorderedListOutlined /></span>
      <span class="oa-card-title">{{ t('chat.todoCard.title') }}</span>
      <a-tag color="blue" class="oa-card-badge">{{ t('chat.todoCard.count', [list.length]) }}</a-tag>
    </div>

    <div v-if="list.length === 0" class="oa-card-empty">
      {{ t('chat.todoCard.empty') }}
    </div>

    <div v-else class="todo-items">
      <div v-for="(item, idx) in list" :key="idx" class="todo-item">
        <div class="todo-item-main">
          <span class="todo-item-title">{{ item.requestname || item.title || item.request_name || t('chat.todoCard.title') }}</span>
          <div class="todo-item-meta">
            <span v-if="item.creator_name || item.creatorname">{{ t('chat.todoCard.creator', [item.creator_name || item.creatorname]) }}</span>
            <span v-if="item.created_at || item.createdate">{{ t('chat.todoCard.time', [item.created_at || item.createdate]) }}</span>
            <span v-if="item.current_node_name || item.nodename" class="node-tag">{{ t('chat.todoCard.node', [item.current_node_name || item.nodename]) }}</span>
          </div>
        </div>
        <div class="todo-item-action" v-if="item.requestid || item.process_id">
          <a-button type="link" size="small" :href="item.oa_link || item.oa_url || '#'" target="_blank">
            <template #icon><LinkOutlined /></template>
            {{ t('chat.todoCard.handle') }}
          </a-button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.oa-card {
  border: 1px solid #ebe5df;
  border-radius: 8px;
  background: #faf8f5;
  padding: 12px 14px;
  margin: 8px 0;
}
.oa-card-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  color: #2c3e50;
  margin-bottom: 10px;
}
.oa-card-icon {
  color: #1890ff;
}
.oa-card-badge {
  margin-left: auto;
}
.oa-card-empty {
  font-size: 13px;
  color: #8c8c8c;
  text-align: center;
  padding: 12px;
}
.todo-items {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.todo-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 10px;
  background: #ffffff;
  border: 1px solid #e8e8e8;
  border-radius: 6px;
}
.todo-item-title {
  font-weight: 500;
  font-size: 13.5px;
  color: #1f2937;
}
.todo-item-meta {
  display: flex;
  gap: 12px;
  font-size: 12px;
  color: #6b7280;
  margin-top: 4px;
}
.node-tag {
  color: #d97706;
}
</style>
