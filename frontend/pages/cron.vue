<script setup lang="ts">
import {
  PlusOutlined,
  DeleteOutlined,
  PlayCircleOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  PauseCircleOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'

definePageMeta({ middleware: 'auth' })

const { mockCronTasks } = useMockData()

const tasks = ref([...mockCronTasks])
const loading = ref(false)
const showCreate = ref(false)

const newTask = ref({
  cron_expression: '0 16 * * *',
  task_type: 'batch_audit',
})

const deleteTask = (id: string) => {
  tasks.value = tasks.value.filter(t => t.id !== id)
  message.success('已删除')
}

const executeTask = async (id: string) => {
  message.loading({ content: '执行中...', key: 'exec' })
  await new Promise(r => setTimeout(r, 1000))
  message.success({ content: '执行完成', key: 'exec' })
}

const toggleTask = (id: string) => {
  const task = tasks.value.find(t => t.id === id)
  if (task) {
    task.is_active = !task.is_active
    message.success(task.is_active ? '已启用' : '已暂停')
  }
}

const createTask = () => {
  tasks.value.push({
    id: `CT-${Date.now()}`,
    ...newTask.value,
    is_active: true,
    last_run_at: null,
    next_run_at: '待计算',
    created_at: new Date().toISOString().slice(0, 10),
    success_count: 0,
    fail_count: 0,
  })
  showCreate.value = false
  message.success('任务创建成功')
}

const taskTypeConfig: Record<string, { label: string; color: string; bg: string }> = {
  batch_audit: { label: '批量审核', color: 'var(--color-primary)', bg: 'var(--color-primary-bg)' },
  daily_report: { label: '日报推送', color: 'var(--color-accent)', bg: 'var(--color-info-bg)' },
  weekly_report: { label: '周报推送', color: '#8b5cf6', bg: 'var(--color-primary-bg)' },
}

const taskTypeOptions = [
  { value: 'batch_audit', label: '批量审核' },
  { value: 'daily_report', label: '日报推送' },
  { value: 'weekly_report', label: '周报推送' },
]
</script>

<template>
  <div class="cron-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">定时任务中心</h1>
        <p class="page-subtitle">Cron 批量审核与推送管理</p>
      </div>
      <a-button type="primary" size="large" @click="showCreate = true">
        <PlusOutlined /> 新建任务
      </a-button>
    </div>

    <!-- Task cards -->
    <div class="task-grid">
      <div
        v-for="task in tasks"
        :key="task.id"
        class="task-card"
        :class="{ 'task-card--inactive': !task.is_active }"
      >
        <div class="task-card-header">
          <span
            class="task-type-tag"
            :style="{
              color: taskTypeConfig[task.task_type]?.color,
              background: taskTypeConfig[task.task_type]?.bg,
            }"
          >
            {{ taskTypeConfig[task.task_type]?.label || task.task_type }}
          </span>
          <div class="task-status" :class="task.is_active ? 'task-status--active' : 'task-status--paused'">
            <span class="task-status-dot" />
            {{ task.is_active ? '运行中' : '已暂停' }}
          </div>
        </div>

        <div class="task-cron">
          <ClockCircleOutlined />
          <code>{{ task.cron_expression }}</code>
        </div>

        <div class="task-stats">
          <div class="task-stat">
            <span class="task-stat-value" style="color: var(--color-success);">{{ task.success_count }}</span>
            <span class="task-stat-label">成功</span>
          </div>
          <div class="task-stat">
            <span class="task-stat-value" style="color: var(--color-danger);">{{ task.fail_count }}</span>
            <span class="task-stat-label">失败</span>
          </div>
          <div class="task-stat">
            <span class="task-stat-value">{{ task.last_run_at || '—' }}</span>
            <span class="task-stat-label">上次执行</span>
          </div>
        </div>

        <div class="task-actions">
          <a-tooltip title="立即执行">
            <button class="task-action-btn task-action-btn--run" @click="executeTask(task.id)">
              <PlayCircleOutlined />
            </button>
          </a-tooltip>
          <a-tooltip :title="task.is_active ? '暂停' : '启用'">
            <button class="task-action-btn task-action-btn--toggle" @click="toggleTask(task.id)">
              <PauseCircleOutlined v-if="task.is_active" />
              <CheckCircleOutlined v-else />
            </button>
          </a-tooltip>
          <a-popconfirm title="确认删除此任务？" @confirm="deleteTask(task.id)">
            <a-tooltip title="删除">
              <button class="task-action-btn task-action-btn--delete">
                <DeleteOutlined />
              </button>
            </a-tooltip>
          </a-popconfirm>
        </div>
      </div>
    </div>

    <!-- Create modal -->
    <a-modal
      v-model:open="showCreate"
      title="新建定时任务"
      @ok="createTask"
      :okText="'创建'"
      :cancelText="'取消'"
    >
      <a-form layout="vertical" style="margin-top: 16px;">
        <a-form-item label="Cron 表达式">
          <a-input v-model:value="newTask.cron_expression" placeholder="0 16 * * *" size="large" />
          <div style="color: var(--color-text-tertiary); font-size: 12px; margin-top: 6px;">
            示例：0 9 * * 1-5 = 工作日每天上午9点
          </div>
        </a-form-item>
        <a-form-item label="任务类型">
          <a-select v-model:value="newTask.task_type" :options="taskTypeOptions" size="large" />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.page-title {
  font-size: 24px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin: 0;
}

.page-subtitle {
  font-size: 14px;
  color: var(--color-text-tertiary);
  margin: 4px 0 0;
}

.task-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 20px;
}

.task-card {
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light);
  padding: 20px;
  transition: all var(--transition-base);
}

.task-card:hover {
  box-shadow: var(--shadow-md);
  transform: translateY(-2px);
}

.task-card--inactive {
  opacity: 0.65;
}

.task-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}

.task-type-tag {
  font-size: 12px;
  font-weight: 600;
  padding: 4px 12px;
  border-radius: var(--radius-full);
}

.task-status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 500;
}

.task-status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
}

.task-status--active {
  color: var(--color-success);
}

.task-status--active .task-status-dot {
  background: var(--color-success);
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.2);
  animation: blink 2s ease-in-out infinite;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.task-status--paused {
  color: var(--color-text-tertiary);
}

.task-status--paused .task-status-dot {
  background: var(--color-text-tertiary);
}

.task-cron {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  background: var(--color-bg-page);
  border-radius: var(--radius-md);
  margin-bottom: 16px;
  color: var(--color-text-secondary);
  font-size: 13px;
}

.task-cron code {
  font-family: var(--font-mono);
  font-weight: 600;
  color: var(--color-text-primary);
}

.task-stats {
  display: grid;
  grid-template-columns: auto auto 1fr;
  gap: 16px;
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--color-border-light);
}

.task-stat {
  display: flex;
  flex-direction: column;
}

.task-stat-value {
  font-size: 14px;
  font-weight: 600;
}

.task-stat-label {
  font-size: 11px;
  color: var(--color-text-tertiary);
  margin-top: 2px;
}

.task-actions {
  display: flex;
  gap: 8px;
}

.task-action-btn {
  width: 36px;
  height: 36px;
  border: 1px solid var(--color-border);
  background: var(--color-bg-card);
  border-radius: var(--radius-md);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  transition: all var(--transition-fast);
  color: var(--color-text-secondary);
}

.task-action-btn:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
  background: var(--color-primary-bg);
}

.task-action-btn--delete:hover {
  border-color: var(--color-danger);
  color: var(--color-danger);
  background: var(--color-danger-bg);
}

@media (max-width: 640px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
  }

  .task-grid {
    grid-template-columns: 1fr;
  }
}
</style>
