<script setup lang="ts">
import { SearchOutlined, DownloadOutlined } from '@ant-design/icons-vue'

definePageMeta({ middleware: 'auth' })

const config = useRuntimeConfig()
const { token } = useAuth()

const filters = ref({
  time_from: '',
  time_to: '',
  department: '',
  process_type: '',
})
const results = ref<any[]>([])
const total = ref(0)
const loading = ref(false)
const selectedSnapshot = ref<any>(null)

const search = async () => {
  loading.value = true
  try {
    const params = new URLSearchParams()
    if (filters.value.time_from) params.set('time_from', filters.value.time_from)
    if (filters.value.time_to) params.set('time_to', filters.value.time_to)
    if (filters.value.department) params.set('department', filters.value.department)
    if (filters.value.process_type) params.set('process_type', filters.value.process_type)

    const data = await $fetch<{ total: number; snapshots: any[] }>(
      `${config.public.apiBase}/api/history/search?${params.toString()}`,
      { headers: { Authorization: `Bearer ${token.value}` } }
    )
    results.value = data.snapshots
    total.value = data.total
  } catch {
    results.value = []
  } finally {
    loading.value = false
  }
}

const exportData = (format: string) => {
  window.open(`${config.public.apiBase}/api/history/export?format=${format}`, '_blank')
}

const columns = [
  { title: '流程 ID', dataIndex: 'process_id', key: 'process_id' },
  { title: '建议', dataIndex: ['audit_result', 'recommendation'], key: 'recommendation' },
  { title: '时间', dataIndex: 'created_at', key: 'created_at' },
  { title: '操作', key: 'action' },
]
</script>

<template>
  <div>
    <a-page-header title="归档复盘" sub-title="历史审核记录检索与复核" />

    <a-card size="small" style="margin-bottom: 16px;">
      <a-space wrap>
        <a-input v-model:value="filters.department" placeholder="部门" style="width: 140px;" />
        <a-input v-model:value="filters.process_type" placeholder="流程类型" style="width: 140px;" />
        <a-button type="primary" @click="search" :loading="loading">
          <SearchOutlined /> 检索
        </a-button>
        <a-button @click="exportData('json')"><DownloadOutlined /> 导出 JSON</a-button>
        <a-button @click="exportData('csv')"><DownloadOutlined /> 导出 CSV</a-button>
      </a-space>
    </a-card>

    <a-table :columns="columns" :data-source="results" :loading="loading" row-key="snapshot_id" size="small">
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'recommendation'">
          <a-tag :color="record.audit_result?.recommendation === 'approve' ? 'green' : record.audit_result?.recommendation === 'reject' ? 'red' : 'orange'">
            {{ record.audit_result?.recommendation }}
          </a-tag>
        </template>
        <template v-if="column.key === 'action'">
          <a-button type="link" size="small" @click="selectedSnapshot = record">详情</a-button>
        </template>
      </template>
    </a-table>

    <SnapshotDetail :snapshot="selectedSnapshot" @close="selectedSnapshot = null" />
  </div>
</template>
