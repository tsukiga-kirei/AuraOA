<script setup lang="ts">
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
  <a-list :data-source="history" size="small" :locale="{ emptyText: '暂无执行记录' }">
    <template #renderItem="{ item }">
      <a-list-item>
        <a-list-item-meta>
          <template #title>
            <a-badge :status="item.success ? 'success' : 'error'" />
            {{ item.message }}
          </template>
          <template #description>
            {{ item.executed_at }} · 处理 {{ item.item_count }} 条
          </template>
        </a-list-item-meta>
      </a-list-item>
    </template>
  </a-list>
</template>
