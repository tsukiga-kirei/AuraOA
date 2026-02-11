<script setup lang="ts">
import {
  PlusOutlined,
  DeleteOutlined,
  PlayCircleOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  PauseCircleOutlined,
  EditOutlined,
  CopyOutlined,
  LockOutlined,
  MailOutlined,
  ScheduleOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import type { CronTask } from '~/composables/useMockData'

definePageMeta({ middleware: 'auth' })

const { mockCronTasks } = useMockData()

const tasks = ref<CronTask[]>(JSON.parse(JSON.stringify(mockCronTasks)))
const loading = ref(false)
const showCreate = ref(false)
const showEdit = ref(false)
const editingTask = ref<CronTask | null>(null)

// ===== Cron expression builder =====
const cronPresets = [
  { label: '工作日每天上午9点', value: '0 9 * * 1-5' },
  { label: '工作日每天下午6点', value: '0 18 * * 1-5' },
  { label: '每天凌晨2点', value: '0 2 * * *' },
  { label: '每周一上午10点', value: '0 10 * * 1' },
  { label: '每月1号上午9点', value: '0 9 1 * *' },
  { label: '每小时', value: '0 * * * *' },
  { label: '每天中午12点', value: '0 12 * * *' },
  { label: '自定义', value: 'custom' },
]

const cronParts = ref({ minute: '0', hour: '9', day: '*', month: '*', weekday: '1-5' })

const weekdayOptions = [
  { label: '周一', value: '1' }, { label: '周二', value: '2' },
  { label: '周三', value: '3' }, { label: '周四', value: '4' },
  { label: '周五', value: '5' }, { label: '周六', value: '6' },
  { label: '周日', value: '0' },
]

// Default push email from personal settings
const defaultPushEmail = 'zhangming@example.com'

const newTask = ref({
  cron_expression: '0 9 * * 1-5',
  cron_mode: '0 9 * * 1-5' as string,
  task_type: 'batch_audit',
  push_email: defaultPushEmail,
})

const buildCronFromParts = () => {
  return `${cronParts.value.minute} ${cronParts.value.hour} ${cronParts.value.day} ${cronParts.value.month} ${cronParts.value.weekday}`
}

// Expand weekday field into a Set of individual day numbers (handles ranges like "1-5" and lists like "1,3,5")
const expandWeekdays = (weekdayStr: string): Set<string> => {
  if (weekdayStr === '*') return new Set(['0','1','2','3','4','5','6'])
  const result = new Set<string>()
  for (const part of weekdayStr.split(',')) {
    const trimmed = part.trim()
    if (trimmed.includes('-')) {
      const [start, end] = trimmed.split('-').map(Number)
      if (!isNaN(start) && !isNaN(end)) {
        for (let i = start; i <= end; i++) result.add(String(i))
      }
    } else {
      result.add(trimmed)
    }
  }
  return result
}

// Check if a weekday chip is active in the current weekday field
const isWeekdayActive = (weekdayStr: string, dayValue: string): boolean => {
  return expandWeekdays(weekdayStr).has(dayValue)
}

// Toggle a weekday chip: rebuild as comma-separated list
const toggleWeekday = (partsRef: typeof cronParts.value, dayValue: string) => {
  const current = expandWeekdays(partsRef.weekday)
  if (current.has(dayValue)) {
    current.delete(dayValue)
  } else {
    current.add(dayValue)
  }
  if (current.size === 0 || current.size === 7) {
    partsRef.weekday = '*'
  } else {
    // Sort numerically and join
    partsRef.weekday = [...current].map(Number).sort((a, b) => a - b).map(String).join(',')
  }
}

watch(cronParts, () => {
  if (newTask.value.cron_mode === 'custom') {
    newTask.value.cron_expression = buildCronFromParts()
  }
}, { deep: true })

watch(() => newTask.value.cron_mode, (val) => {
  if (val !== 'custom') {
    newTask.value.cron_expression = val
  } else {
    newTask.value.cron_expression = buildCronFromParts()
  }
})

// ===== Cron description & next run =====
const describeCron = (expr: string): string => {
  const map: Record<string, string> = {
    '0 9 * * 1-5': '工作日每天上午 9:00',
    '0 18 * * 1-5': '工作日每天下午 6:00',
    '0 2 * * *': '每天凌晨 2:00',
    '0 10 * * 1': '每周一上午 10:00',
    '0 9 1 * *': '每月1号上午 9:00',
    '0 * * * *': '每小时整点',
    '0 12 * * *': '每天中午 12:00',
    '0 16 * * *': '每天下午 4:00',
  }
  return map[expr] || expr
}

const calcNextRuns = (expr: string, count: number = 3): string[] => {
  const now = new Date(2026, 1, 11, 10, 0) // Feb 11, 2026 (Wednesday)
  const parts = expr.split(' ')
  if (parts.length !== 5) return ['表达式格式错误']
  const [minStr, hourStr, dayStr, monthStr, weekdayStr] = parts
  const h = parseInt(hourStr)
  const m = parseInt(minStr)
  if (isNaN(h) || isNaN(m)) return ['待计算']

  const allowedWeekdays = expandWeekdays(weekdayStr)
  const hasMonthFilter = monthStr !== '*'
  const hasDayFilter = dayStr !== '*'
  const allowedMonths = hasMonthFilter ? new Set(monthStr.split(',').map(s => s.trim())) : null
  const allowedDays = hasDayFilter ? new Set(dayStr.split(',').map(s => s.trim())) : null

  const results: string[] = []
  const candidate = new Date(now)
  candidate.setHours(h, m, 0, 0)
  // If today's time already passed, start from tomorrow
  if (candidate <= now) candidate.setDate(candidate.getDate() + 1)

  let safety = 0
  while (results.length < count && safety < 400) {
    safety++
    const dow = candidate.getDay() // 0=Sun
    const dom = candidate.getDate()
    const mon = candidate.getMonth() + 1

    const weekdayOk = allowedWeekdays.has(String(dow))
    const monthOk = !allowedMonths || allowedMonths.has(String(mon))
    const dayOk = !allowedDays || allowedDays.has(String(dom))

    if (weekdayOk && monthOk && dayOk) {
      results.push(
        `${candidate.getFullYear()}-${String(candidate.getMonth() + 1).padStart(2, '0')}-${String(candidate.getDate()).padStart(2, '0')} ${String(h).padStart(2, '0')}:${String(m).padStart(2, '0')}`
      )
    }
    candidate.setDate(candidate.getDate() + 1)
  }
  return results.length ? results : ['无匹配的执行时间']
}

const previewNextRuns = computed(() => calcNextRuns(newTask.value.cron_expression))
const editPreviewNextRuns = computed(() => editingTask.value ? calcNextRuns(editingTask.value.cron_expression) : [])

// ===== Edit cron parts for edit modal =====
const editCronParts = ref({ minute: '0', hour: '9', day: '*', month: '*', weekday: '1-5' })
const editCronMode = ref('0 9 * * 1-5')

watch(editCronParts, () => {
  if (editCronMode.value === 'custom' && editingTask.value) {
    editingTask.value.cron_expression = `${editCronParts.value.minute} ${editCronParts.value.hour} ${editCronParts.value.day} ${editCronParts.value.month} ${editCronParts.value.weekday}`
  }
}, { deep: true })

watch(editCronMode, (val) => {
  if (!editingTask.value) return
  if (val !== 'custom') {
    editingTask.value.cron_expression = val
  } else {
    editingTask.value.cron_expression = `${editCronParts.value.minute} ${editCronParts.value.hour} ${editCronParts.value.day} ${editCronParts.value.month} ${editCronParts.value.weekday}`
  }
})

// ===== Task CRUD =====
const deleteTask = (id: string) => {
  const task = tasks.value.find(t => t.id === id)
  if (task?.is_builtin) {
    message.warning('内置任务不可删除')
    return
  }
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
    cron_expression: newTask.value.cron_expression,
    task_type: newTask.value.task_type,
    push_email: newTask.value.push_email,
    is_active: true,
    last_run_at: null,
    next_run_at: calcNextRuns(newTask.value.cron_expression, 1)[0] || '待计算',
    created_at: new Date().toISOString().slice(0, 10),
    success_count: 0,
    fail_count: 0,
  })
  showCreate.value = false
  newTask.value = { cron_expression: '0 9 * * 1-5', cron_mode: '0 9 * * 1-5', task_type: 'batch_audit', push_email: defaultPushEmail }
  message.success('任务创建成功')
}

const openEdit = (task: CronTask) => {
  editingTask.value = JSON.parse(JSON.stringify(task))
  // Default push email from personal settings if empty
  if (!editingTask.value!.push_email) {
    editingTask.value!.push_email = defaultPushEmail
  }
  // Determine cron mode
  const isPreset = cronPresets.find(p => p.value === task.cron_expression && p.value !== 'custom')
  editCronMode.value = isPreset ? task.cron_expression : 'custom'
  if (!isPreset) {
    const parts = task.cron_expression.split(' ')
    if (parts.length === 5) {
      editCronParts.value = { minute: parts[0], hour: parts[1], day: parts[2], month: parts[3], weekday: parts[4] }
    }
  }
  showEdit.value = true
}

const saveEdit = () => {
  if (!editingTask.value) return
  const idx = tasks.value.findIndex(t => t.id === editingTask.value!.id)
  if (idx >= 0) {
    tasks.value[idx] = { ...editingTask.value, next_run_at: calcNextRuns(editingTask.value.cron_expression, 1)[0] || '待计算' }
  }
  showEdit.value = false
  editingTask.value = null
  message.success('任务已更新')
}

const copyTask = (task: CronTask) => {
  const copied: CronTask = {
    ...JSON.parse(JSON.stringify(task)),
    id: `CT-${Date.now()}`,
    is_builtin: false,
    is_active: false,
    success_count: 0,
    fail_count: 0,
    last_run_at: null,
    created_at: new Date().toISOString().slice(0, 10),
  }
  tasks.value.push(copied)
  message.success('任务已复制')
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
          <div class="task-card-header-left">
            <span
              class="task-type-tag"
              :style="{
                color: taskTypeConfig[task.task_type]?.color,
                background: taskTypeConfig[task.task_type]?.bg,
              }"
            >
              {{ taskTypeConfig[task.task_type]?.label || task.task_type }}
            </span>
            <span v-if="task.is_builtin" class="builtin-tag">
              <LockOutlined /> 内置
            </span>
          </div>
          <div class="task-status" :class="task.is_active ? 'task-status--active' : 'task-status--paused'">
            <span class="task-status-dot" />
            {{ task.is_active ? '运行中' : '已暂停' }}
          </div>
        </div>

        <div class="task-cron">
          <ClockCircleOutlined />
          <code>{{ task.cron_expression }}</code>
          <span class="cron-desc">{{ describeCron(task.cron_expression) }}</span>
        </div>

        <div v-if="task.push_email" class="task-email">
          <MailOutlined />
          <span>{{ task.push_email }}</span>
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
          <a-tooltip title="编辑">
            <button class="task-action-btn" @click="openEdit(task)">
              <EditOutlined />
            </button>
          </a-tooltip>
          <a-tooltip title="复制">
            <button class="task-action-btn" @click="copyTask(task)">
              <CopyOutlined />
            </button>
          </a-tooltip>
          <a-popconfirm
            v-if="!task.is_builtin"
            title="确认删除此任务？"
            @confirm="deleteTask(task.id)"
          >
            <a-tooltip title="删除">
              <button class="task-action-btn task-action-btn--delete">
                <DeleteOutlined />
              </button>
            </a-tooltip>
          </a-popconfirm>
          <a-tooltip v-else title="内置任务不可删除">
            <button class="task-action-btn task-action-btn--disabled" disabled>
              <DeleteOutlined />
            </button>
          </a-tooltip>
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
      :width="560"
    >
      <a-form layout="vertical" style="margin-top: 16px;">
        <a-form-item label="任务类型">
          <a-select v-model:value="newTask.task_type" :options="taskTypeOptions" size="large" />
        </a-form-item>
        <a-form-item label="执行计划">
          <a-select v-model:value="newTask.cron_mode" size="large" style="width: 100%;">
            <a-select-option v-for="p in cronPresets" :key="p.value" :value="p.value">
              {{ p.label }}
              <span v-if="p.value !== 'custom'" style="color: var(--color-text-tertiary); margin-left: 8px; font-family: monospace; font-size: 12px;">{{ p.value }}</span>
            </a-select-option>
          </a-select>
        </a-form-item>
        <div v-if="newTask.cron_mode === 'custom'" class="cron-builder">
          <div class="cron-builder-row">
            <div class="cron-builder-field">
              <label>分钟</label>
              <a-input v-model:value="cronParts.minute" placeholder="0-59 或 *" size="small" />
            </div>
            <div class="cron-builder-field">
              <label>小时</label>
              <a-input v-model:value="cronParts.hour" placeholder="0-23 或 *" size="small" />
            </div>
            <div class="cron-builder-field">
              <label>日</label>
              <a-input v-model:value="cronParts.day" placeholder="1-31 或 *" size="small" />
            </div>
            <div class="cron-builder-field">
              <label>月</label>
              <a-input v-model:value="cronParts.month" placeholder="1-12 或 *" size="small" />
            </div>
            <div class="cron-builder-field">
              <label>星期</label>
              <a-input v-model:value="cronParts.weekday" placeholder="0-6 或 *" size="small" />
            </div>
          </div>
          <div class="cron-builder-weekdays">
            <span
              v-for="wd in weekdayOptions"
              :key="wd.value"
              class="weekday-chip"
              :class="{ 'weekday-chip--active': isWeekdayActive(cronParts.weekday, wd.value) }"
              @click="toggleWeekday(cronParts, wd.value)"
            >{{ wd.label }}</span>
          </div>
          <div class="cron-expression-preview">
            <code>{{ newTask.cron_expression }}</code>
          </div>
        </div>
        <div class="next-run-preview">
          <ScheduleOutlined />
          <div>
            <div class="next-run-title">下次执行时间预览</div>
            <div v-for="(run, i) in previewNextRuns" :key="i" class="next-run-item">{{ run }}</div>
          </div>
        </div>
        <a-form-item label="推送邮箱">
          <a-input v-model:value="newTask.push_email" placeholder="接收推送结果的邮箱地址，多个邮箱使用英文逗号分隔" size="large">
            <template #prefix><MailOutlined style="color: var(--color-text-tertiary);" /></template>
          </a-input>
          <div class="email-hint">多个邮箱请使用英文逗号（,）分隔</div>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Edit modal -->
    <a-modal
      v-model:open="showEdit"
      title="编辑定时任务"
      @ok="saveEdit"
      :okText="'保存'"
      :cancelText="'取消'"
      :width="560"
    >
      <a-form v-if="editingTask" layout="vertical" style="margin-top: 16px;">
        <a-form-item label="任务类型">
          <a-select v-model:value="editingTask.task_type" :options="taskTypeOptions" size="large" />
        </a-form-item>
        <a-form-item label="执行计划">
          <a-select v-model:value="editCronMode" size="large" style="width: 100%;">
            <a-select-option v-for="p in cronPresets" :key="p.value" :value="p.value">
              {{ p.label }}
              <span v-if="p.value !== 'custom'" style="color: var(--color-text-tertiary); margin-left: 8px; font-family: monospace; font-size: 12px;">{{ p.value }}</span>
            </a-select-option>
          </a-select>
        </a-form-item>
        <div v-if="editCronMode === 'custom'" class="cron-builder">
          <div class="cron-builder-row">
            <div class="cron-builder-field">
              <label>分钟</label>
              <a-input v-model:value="editCronParts.minute" placeholder="0-59 或 *" size="small" />
            </div>
            <div class="cron-builder-field">
              <label>小时</label>
              <a-input v-model:value="editCronParts.hour" placeholder="0-23 或 *" size="small" />
            </div>
            <div class="cron-builder-field">
              <label>日</label>
              <a-input v-model:value="editCronParts.day" placeholder="1-31 或 *" size="small" />
            </div>
            <div class="cron-builder-field">
              <label>月</label>
              <a-input v-model:value="editCronParts.month" placeholder="1-12 或 *" size="small" />
            </div>
            <div class="cron-builder-field">
              <label>星期</label>
              <a-input v-model:value="editCronParts.weekday" placeholder="0-6 或 *" size="small" />
            </div>
          </div>
          <div class="cron-builder-weekdays">
            <span
              v-for="wd in weekdayOptions"
              :key="wd.value"
              class="weekday-chip"
              :class="{ 'weekday-chip--active': isWeekdayActive(editCronParts.weekday, wd.value) }"
              @click="toggleWeekday(editCronParts, wd.value)"
            >{{ wd.label }}</span>
          </div>
          <div class="cron-expression-preview">
            <code>{{ editingTask.cron_expression }}</code>
          </div>
        </div>
        <div class="next-run-preview">
          <ScheduleOutlined />
          <div>
            <div class="next-run-title">下次执行时间预览</div>
            <div v-for="(run, i) in editPreviewNextRuns" :key="i" class="next-run-item">{{ run }}</div>
          </div>
        </div>
        <a-form-item label="推送邮箱">
          <a-input v-model:value="editingTask.push_email" placeholder="接收推送结果的邮箱地址，多个邮箱使用英文逗号分隔" size="large">
            <template #prefix><MailOutlined style="color: var(--color-text-tertiary);" /></template>
          </a-input>
          <div class="email-hint">多个邮箱请使用英文逗号（,）分隔</div>
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
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
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

.task-card-header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.task-type-tag {
  font-size: 12px;
  font-weight: 600;
  padding: 4px 12px;
  border-radius: var(--radius-full);
}

.builtin-tag {
  font-size: 10px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: var(--radius-full);
  background: var(--color-warning-bg);
  color: var(--color-warning);
  display: inline-flex;
  align-items: center;
  gap: 3px;
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
  margin-bottom: 10px;
  color: var(--color-text-secondary);
  font-size: 13px;
}

.task-cron code {
  font-family: var(--font-mono);
  font-weight: 600;
  color: var(--color-text-primary);
}

.cron-desc {
  font-size: 12px;
  color: var(--color-text-tertiary);
  margin-left: auto;
}

.task-email {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 14px;
  font-size: 12px;
  color: var(--color-text-tertiary);
  margin-bottom: 10px;
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

.task-action-btn--disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.task-action-btn--disabled:hover {
  border-color: var(--color-border);
  color: var(--color-text-tertiary);
  background: var(--color-bg-card);
}

/* Cron builder */
.cron-builder {
  background: var(--color-bg-page);
  border-radius: var(--radius-md);
  padding: 14px;
  margin-bottom: 4px;
}

.cron-builder-row {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 8px;
  margin-bottom: 10px;
}

.cron-builder-field label {
  display: block;
  font-size: 11px;
  font-weight: 600;
  color: var(--color-text-tertiary);
  margin-bottom: 4px;
}

.cron-builder-weekdays {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  margin-bottom: 10px;
}

.weekday-chip {
  font-size: 12px;
  padding: 3px 10px;
  border-radius: var(--radius-full);
  border: 1px solid var(--color-border);
  cursor: pointer;
  transition: all var(--transition-fast);
  color: var(--color-text-secondary);
}

.weekday-chip:hover {
  border-color: var(--color-primary);
}

.weekday-chip--active {
  background: var(--color-primary);
  color: #fff;
  border-color: var(--color-primary);
}

.cron-expression-preview {
  text-align: center;
  padding: 6px;
  background: var(--color-bg-card);
  border-radius: var(--radius-sm);
}

.cron-expression-preview code {
  font-family: var(--font-mono);
  font-size: 14px;
  font-weight: 600;
  color: var(--color-primary);
}

/* Next run preview */
.next-run-preview {
  display: flex;
  gap: 10px;
  padding: 12px 14px;
  background: var(--color-info-bg);
  border-radius: var(--radius-md);
  margin-bottom: 16px;
  color: var(--color-info);
  font-size: 13px;
}

.next-run-title {
  font-weight: 600;
  margin-bottom: 4px;
}

.next-run-item {
  font-size: 12px;
  font-family: var(--font-mono);
  color: var(--color-text-secondary);
}

.email-hint {
  font-size: 12px;
  color: var(--color-text-tertiary);
  margin-top: 4px;
}

@media (max-width: 640px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
  }

  .task-grid {
    grid-template-columns: 1fr;
  }

  .cron-builder-row {
    grid-template-columns: repeat(3, 1fr);
  }
}
</style>
