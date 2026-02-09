<script setup lang="ts">
import { PlusOutlined, DeleteOutlined, PlayCircleOutlined } from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'

definePageMeta({ middleware: 'auth' })

const config = useRuntimeConfig()
const { token } = useAuth()

const tasks = ref<any[]>([])
const loading = ref(false)
const showCreate = ref(false)

const newTask = ref({
  cron_expression: '0 16 * * *',
  task_type: 'batch_audit',
})

const headers = computed(() => ({
  Authorization: `Bearer ${token.value}`,
}))

const fetchTasks = async () => {
  loading.value = true
  try {
    const data = await $fetch<{ tasks: any[] }>(`${config.public.apiBase}/api/cron`, {
      headers: headers.value,
    })
    tasks.value = data.tasks
  } catch {
    tasks.value = []
  } finally {
    loading.value = false
  }
}

const createTask = async () => {
  try {
    await $fetch(`${config.public.apiBase}/api/cron`, {
      method: 'POST',
      headers: headers.value,
      body: newTask.value,
    })
    message.success('任务创建成功')
    showCreate.value = false
    await fetchTasks()
  } catch {
    message.error('创建失败')
  }
}

const deleteTask = async (id: string) => {
  await $fetch(`${config.public.apiBase}/api/cron/${id}`, {
    method: 'DELETE',
    headers: headers.value,
  })
  message.success('已删除')
  await fetchTasks()
}

const executeTask = async (id: string) => {
  try {
    await $fetch(`${config.public.apiBase}/api/cron/${id}/execute`, {
      method: 'POST',
      headers: headers.value,
    })
    message.success('执行完成')
    await fetchTasks()
  } catch {
    message.error('执行失败')
  }
}

onMounted(fetchTasks)

const taskTypeOptions = [
  { value: 'batch_audit', label: '批量审核' },
  { value: 'daily_report', label: '日报推送' },
  { value: 'weekly_report', label: '周报推送' },
]

const columns = [
  { title: 'Cron 表达式', dataIndex: 'cron_expression', key: 'cron' },
  { title: '任务类型', dataIndex: 'task_type', key: 'type' },
  { title: '状态', dataIndex: 'is_active', key: 'status' },
  { title: '上次执行', dataIndex: 'last_run_at', key: 'last_run' },
  { title: '操作', key: 'action' },
]
</script>

<template>
  <div>
    <a-page-header title="定时任务中心" sub-title="Cron 批量审核与推送">
      <template #extra>
        <a-button type="primary" @click="showCreate = true"><PlusOutlined /> 新建任务</a-button>
      </template>
    </a-page-header>

    <a-table :columns="columns" :data-source="tasks" :loading="loading" row-key="id" size="small">
      <template #bodyCell="{ column, record }">
        <template v-if="column.key === 'type'">
          <a-tag>{{ record.task_type }}</a-tag>
        </template>
        <template v-if="column.key === 'status'">
          <a-badge :status="record.is_active ? 'processing' : 'default'" :text="record.is_active ? '运行中' : '已停止'" />
        </template>
        <template v-if="column.key === 'last_run'">
          {{ record.last_run_at || '—' }}
        </template>
        <template v-if="column.key === 'action'">
          <a-space>
            <a-button type="link" size="small" @click="executeTask(record.id)"><PlayCircleOutlined /> 执行</a-button>
            <a-popconfirm title="确认删除？" @confirm="deleteTask(record.id)">
              <a-button type="link" danger size="small"><DeleteOutlined /> 删除</a-button>
            </a-popconfirm>
          </a-space>
        </template>
      </template>
    </a-table>

    <a-modal v-model:open="showCreate" title="新建定时任务" @ok="createTask">
      <a-form layout="vertical">
        <a-form-item label="Cron 表达式">
          <a-input v-model:value="newTask.cron_expression" placeholder="0 16 * * *" />
          <div style="color: #999; font-size: 12px; margin-top: 4px;">示例：0 16 * * * = 每天下午4点</div>
        </a-form-item>
        <a-form-item label="任务类型">
          <a-select v-model:value="newTask.task_type" :options="taskTypeOptions" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>
