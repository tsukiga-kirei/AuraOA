<script setup lang="ts">
import { SearchOutlined, ThunderboltOutlined } from '@ant-design/icons-vue'

definePageMeta({ middleware: 'auth' })

const { todoList, currentResult, loading, getTodoList, executeAudit } = useAudit()
const selectedProcess = ref<string | null>(null)
const searchText = ref('')

onMounted(() => {
  getTodoList()
})

const filteredList = computed(() => {
  if (!searchText.value) return todoList.value
  const q = searchText.value.toLowerCase()
  return todoList.value.filter(
    p => p.title.toLowerCase().includes(q) || p.applicant.toLowerCase().includes(q)
  )
})

const handleAudit = async (processId: string) => {
  selectedProcess.value = processId
  await executeAudit(processId)
}
</script>

<template>
  <div>
    <a-page-header title="审核工作台" sub-title="智能待办审核" />

    <a-row :gutter="16">
      <a-col :span="10">
        <a-card title="待办流程" size="small">
          <template #extra>
            <a-input
              v-model:value="searchText"
              placeholder="搜索流程"
              size="small"
              style="width: 160px;"
              allow-clear
            >
              <template #prefix><SearchOutlined /></template>
            </a-input>
          </template>

          <a-list :data-source="filteredList" size="small" :locale="{ emptyText: '暂无待办流程' }">
            <template #renderItem="{ item }">
              <a-list-item>
                <a-list-item-meta :title="item.title" :description="`${item.applicant} · ${item.submit_time}`" />
                <template #actions>
                  <a-button type="primary" size="small" @click="handleAudit(item.process_id)">
                    <ThunderboltOutlined /> 审核
                  </a-button>
                </template>
              </a-list-item>
            </template>
          </a-list>
        </a-card>
      </a-col>

      <a-col :span="14">
        <a-card title="审核结果" size="small">
          <AuditPanel :result="currentResult" :loading="loading" />
        </a-card>
      </a-col>
    </a-row>
  </div>
</template>
