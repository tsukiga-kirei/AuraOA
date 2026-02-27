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
import { useI18n } from '~/composables/useI18n'

definePageMeta({ middleware: 'auth' })

const { t } = useI18n()
const { mockCronTasks, mockCronTaskTypeConfigs } = useMockData()

const tasks = ref<CronTask[]>(JSON.parse(JSON.stringify(mockCronTasks)))

// batch_audit config from tenant settings
const batchAuditConfig = computed(() =>
  mockCronTaskTypeConfigs.find(c => c.task_type === 'batch_audit')
)
const batchLimit = computed(() => batchAuditConfig.value?.batch_limit ?? 0)
const loading = ref(false)
const showCreate = ref(false)
const showEdit = ref(false)
const editingTask = ref<CronTask | null>(null)

// ===== Cron expression builder =====
const cronPresets = computed(() => [
  { label: t('cron.preset.weekday9'), value: '0 9 * * 1-5' },
  { label: t('cron.preset.weekday18'), value: '0 18 * * 1-5' },
  { label: t('cron.preset.daily2'), value: '0 2 * * *' },
  { label: t('cron.preset.monday10'), value: '0 10 * * 1' },
  { label: t('cron.preset.monthly1_9'), value: '0 9 1 * *' },
  { label: t('cron.preset.hourly'), value: '0 * * * *' },
  { label: t('cron.preset.daily12'), value: '0 12 * * *' },
  { label: t('cron.preset.custom'), value: 'custom' },
])

const cronParts = ref({ minute: '0', hour: '9', day: '*', month: '*', weekday: '1-5' })

const weekdayOptions = computed(() => [
  { label: t('cron.weekday.mon'), value: '1' }, { label: t('cron.weekday.tue'), value: '2' },
  { label: t('cron.weekday.wed'), value: '3' }, { label: t('cron.weekday.thu'), value: '4' },
  { label: t('cron.weekday.fri'), value: '5' }, { label: t('cron.weekday.sat'), value: '6' },
  { label: t('cron.weekday.sun'), value: '0' },
])

// Default push email from personal settings
const defaultPushEmail = 'zhangming@example.com'

const newTask = ref({
  cron_expression: '0 9 * * 1-5',
  cron_mode: '0 9 * * 1-5' as string,
  task_type: 'batch_audit',
  push_email: defaultPushEmail,
})

// Whether the current new task type needs email push
const newTaskNeedsEmail = computed(() => newTask.value.task_type !== 'batch_audit')

// Whether a given task type needs email push
const taskNeedsEmail = (taskType: string) => taskType !== 'batch_audit'

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
    '0 9 * * 1-5': t('cron.describe.weekday9'),
    '0 18 * * 1-5': t('cron.describe.weekday18'),
    '0 2 * * *': t('cron.describe.daily2'),
    '0 10 * * 1': t('cron.describe.monday10'),
    '0 9 1 * *': t('cron.describe.monthly1_9'),
    '0 * * * *': t('cron.describe.hourly'),
    '0 12 * * *': t('cron.describe.daily12'),
    '0 16 * * *': t('cron.describe.daily16'),
  }
  return map[expr] || expr
}

const calcNextRuns = (expr: string, count: number = 3): string[] => {
  const now = new Date(2026, 1, 11, 10, 0) // Feb 11, 2026 (Wednesday)
  const parts = expr.split(' ')
  if (parts.length !== 5) return [t('cron.describe.exprError')]
  const [minStr, hourStr, dayStr, monthStr, weekdayStr] = parts
  const h = parseInt(hourStr)
  const m = parseInt(minStr)
  if (isNaN(h) || isNaN(m)) return [t('cron.describe.pending')]

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
  return results.length ? results : [t('cron.describe.noMatch')]
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
    message.warning(t('cron.builtinDeleteWarn'))
    return
  }
  tasks.value = tasks.value.filter(t => t.id !== id)
  message.success(t('cron.deleted'))
}

const executeTask = async (id: string) => {
  message.loading({ content: t('cron.executing'), key: 'exec' })
  await new Promise(r => setTimeout(r, 1000))
  message.success({ content: t('cron.executeDone'), key: 'exec' })
}

const toggleTask = (id: string) => {
  const task = tasks.value.find(t => t.id === id)
  if (task) {
    task.is_active = !task.is_active
    message.success(task.is_active ? t('cron.enabled') : t('cron.paused'))
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
    next_run_at: calcNextRuns(newTask.value.cron_expression, 1)[0] || t('cron.describe.pending'),
    created_at: new Date().toISOString().slice(0, 10),
    success_count: 0,
    fail_count: 0,
  })
  showCreate.value = false
  newTask.value = { cron_expression: '0 9 * * 1-5', cron_mode: '0 9 * * 1-5', task_type: 'batch_audit', push_email: defaultPushEmail }
  message.success(t('cron.taskCreated'))
}

const openEdit = (task: CronTask) => {
  editingTask.value = JSON.parse(JSON.stringify(task))
  // Default push email from personal settings if empty
  if (!editingTask.value!.push_email) {
    editingTask.value!.push_email = defaultPushEmail
  }
  // Determine cron mode
  const isPreset = cronPresets.value.find(p => p.value === task.cron_expression && p.value !== 'custom')
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
    tasks.value[idx] = { ...editingTask.value, next_run_at: calcNextRuns(editingTask.value.cron_expression, 1)[0] || t('cron.describe.pending') }
  }
  showEdit.value = false
  editingTask.value = null
  message.success(t('cron.taskUpdated'))
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
  message.success(t('cron.taskCopied'))
}

const taskTypeConfig = computed<Record<string, { label: string; color: string; bg: string; }>>(() => ({
  batch_audit: { label: t('cron.batchAudit'), color: 'var(--color-primary)', bg: 'var(--color-primary-bg)' },
  daily_report: { label: t('cron.dailyReport'), color: 'var(--color-accent)', bg: 'var(--color-info-bg)' },
  weekly_report: { label: t('cron.weeklyReport'), color: '#8b5cf6', bg: 'var(--color-primary-bg)' },
}))

const taskTypeOptions = computed(() => [
  { value: 'batch_audit', label: t('cron.batchAudit') },
  { value: 'daily_report', label: t('cron.dailyReport') },
  { value: 'weekly_report', label: t('cron.weeklyReport') },
])
</script>

<template>
  <div class="cron-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('cron.pageTitle') }}</h1>
        <p class="page-subtitle">{{ t('cron.pageSubtitle') }}</p>
      </div>
      <a-button type="primary" size="large" @click="showCreate = true">
        <PlusOutlined /> {{ t('cron.createTask') }}
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
              <LockOutlined /> {{ t('cron.builtin') }}
            </span>
          </div>
          <div class="task-status" :class="task.is_active ? 'task-status--active' : 'task-status--paused'">
            <span class="task-status-dot" />
            {{ task.is_active ? t('cron.running') : t('cron.paused') }}
          </div>
        </div>

        <div class="task-cron">
          <ClockCircleOutlined />
          <code>{{ task.cron_expression }}</code>
          <span class="cron-desc">{{ describeCron(task.cron_expression) }}</span>
        </div>

        <div v-if="task.push_email && taskNeedsEmail(task.task_type)" class="task-email">
          <MailOutlined />
          <span>{{ task.push_email }}</span>
        </div>

        <div class="task-stats">
          <div class="task-stat">
            <span class="task-stat-value" style="color: var(--color-success);">{{ task.success_count }}</span>
            <span class="task-stat-label">{{ t('cron.success') }}</span>
          </div>
          <div class="task-stat">
            <span class="task-stat-value" style="color: var(--color-danger);">{{ task.fail_count }}</span>
            <span class="task-stat-label">{{ t('cron.fail') }}</span>
          </div>
          <div class="task-stat">
            <span class="task-stat-value">{{ task.last_run_at || '—' }}</span>
            <span class="task-stat-label">{{ t('cron.lastExec') }}</span>
          </div>
        </div>

        <div class="task-actions">
          <a-tooltip :title="t('cron.executeNow')">
            <button class="task-action-btn task-action-btn--run" @click="executeTask(task.id)">
              <PlayCircleOutlined />
            </button>
          </a-tooltip>
          <a-tooltip :title="task.is_active ? t('cron.pause') : t('cron.enable')">
            <button class="task-action-btn task-action-btn--toggle" @click="toggleTask(task.id)">
              <PauseCircleOutlined v-if="task.is_active" />
              <CheckCircleOutlined v-else />
            </button>
          </a-tooltip>
          <a-tooltip :title="t('cron.edit')">
            <button class="task-action-btn" @click="openEdit(task)">
              <EditOutlined />
            </button>
          </a-tooltip>
          <a-tooltip :title="t('cron.copy')">
            <button class="task-action-btn" @click="copyTask(task)">
              <CopyOutlined />
            </button>
          </a-tooltip>
          <a-popconfirm
            v-if="!task.is_builtin"
            :title="t('cron.deleteConfirm')"
            @confirm="deleteTask(task.id)"
          >
            <a-tooltip :title="t('cron.delete')">
              <button class="task-action-btn task-action-btn--delete">
                <DeleteOutlined />
              </button>
            </a-tooltip>
          </a-popconfirm>
          <a-tooltip v-else :title="t('cron.builtinNoDelete')">
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
      :title="t('cron.createTitle')"
      @ok="createTask"
      :okText="t('cron.create')"
      :cancelText="t('cron.cancel')"
      :width="560"
    >
      <a-form layout="vertical" style="margin-top: 16px;">
        <a-form-item :label="t('cron.taskType')">
          <a-select v-model:value="newTask.task_type" :options="taskTypeOptions" size="large" :placeholder="t('cron.selectTaskType')" />
          <div v-if="newTask.task_type === 'batch_audit'" class="email-hint" style="margin-top: 8px;">
            {{ t('cron.batchAuditDesc') }}
            <div style="margin-top: 6px; color: var(--color-text-secondary);">
              {{ t('cron.batchLimitHint', batchLimit) }}
            </div>
          </div>
          <div v-if="newTask.task_type === 'daily_report'" class="email-hint" style="margin-top: 8px;">
            {{ t('cron.dailyReportDesc') }}
          </div>
          <div v-if="newTask.task_type === 'weekly_report'" class="email-hint" style="margin-top: 8px;">
            {{ t('cron.weeklyReportDesc') }}
          </div>
        </a-form-item>
        <a-form-item :label="t('cron.executePlan')">
          <a-select v-model:value="newTask.cron_mode" size="large" style="width: 100%;" :placeholder="t('cron.selectOrCustom')">
            <a-select-option v-for="p in cronPresets" :key="p.value" :value="p.value">
              {{ p.label }}
              <span v-if="p.value !== 'custom'" style="color: var(--color-text-tertiary); margin-left: 8px; font-family: monospace; font-size: 12px;">{{ p.value }}</span>
            </a-select-option>
          </a-select>
        </a-form-item>
        <div v-if="newTask.cron_mode === 'custom'" class="cron-builder">
          <div class="cron-builder-row">
            <div class="cron-builder-field">
              <label>{{ t('cron.minute') }}</label>
              <a-input v-model:value="cronParts.minute" placeholder="0-59 / *" size="small" />
            </div>
            <div class="cron-builder-field">
              <label>{{ t('cron.hour') }}</label>
              <a-input v-model:value="cronParts.hour" placeholder="0-23 / *" size="small" />
            </div>
            <div class="cron-builder-field">
              <label>{{ t('cron.day') }}</label>
              <a-input v-model:value="cronParts.day" placeholder="1-31 / *" size="small" />
            </div>
            <div class="cron-builder-field">
              <label>{{ t('cron.month') }}</label>
              <a-input v-model:value="cronParts.month" placeholder="1-12 / *" size="small" />
            </div>
            <div class="cron-builder-field">
              <label>{{ t('cron.weekday') }}</label>
              <a-input v-model:value="cronParts.weekday" placeholder="0-6 / *" size="small" />
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
            <div class="next-run-title">{{ t('cron.nextRunPreview') }}</div>
            <div v-for="(run, i) in previewNextRuns" :key="i" class="next-run-item">{{ run }}</div>
          </div>
        </div>
        <a-form-item v-if="newTaskNeedsEmail" :label="t('cron.pushEmail')">
          <a-input v-model:value="newTask.push_email" :placeholder="t('cron.emailPlaceholder')" size="large">
            <template #prefix><MailOutlined style="color: var(--color-text-tertiary);" /></template>
          </a-input>
          <div class="email-hint">{{ t('cron.emailHint') }}</div>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- Edit modal -->
    <a-modal
      v-model:open="showEdit"
      :title="t('cron.editTitle')"
      @ok="saveEdit"
      :okText="t('cron.save')"
      :cancelText="t('cron.cancel')"
      :width="560"
    >
      <a-form v-if="editingTask" layout="vertical" style="margin-top: 16px;">
        <a-form-item :label="t('cron.taskType')">
          <a-select v-model:value="editingTask.task_type" :options="taskTypeOptions" size="large" :placeholder="t('cron.selectTaskType')" />
        </a-form-item>
        <a-form-item :label="t('cron.executePlan')">
          <a-select v-model:value="editCronMode" size="large" style="width: 100%;" :placeholder="t('cron.selectOrCustom')">
            <a-select-option v-for="p in cronPresets" :key="p.value" :value="p.value">
              {{ p.label }}
              <span v-if="p.value !== 'custom'" style="color: var(--color-text-tertiary); margin-left: 8px; font-family: monospace; font-size: 12px;">{{ p.value }}</span>
            </a-select-option>
          </a-select>
        </a-form-item>
        <div v-if="editCronMode === 'custom'" class="cron-builder">
          <div class="cron-builder-row">
            <div class="cron-builder-field">
              <label>{{ t('cron.minute') }}</label>
              <a-input v-model:value="editCronParts.minute" placeholder="0-59 / *" size="small" />
            </div>
            <div class="cron-builder-field">
              <label>{{ t('cron.hour') }}</label>
              <a-input v-model:value="editCronParts.hour" placeholder="0-23 / *" size="small" />
            </div>
            <div class="cron-builder-field">
              <label>{{ t('cron.day') }}</label>
              <a-input v-model:value="editCronParts.day" placeholder="1-31 / *" size="small" />
            </div>
            <div class="cron-builder-field">
              <label>{{ t('cron.month') }}</label>
              <a-input v-model:value="editCronParts.month" placeholder="1-12 / *" size="small" />
            </div>
            <div class="cron-builder-field">
              <label>{{ t('cron.weekday') }}</label>
              <a-input v-model:value="editCronParts.weekday" placeholder="0-6 / *" size="small" />
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
            <div class="next-run-title">{{ t('cron.nextRunPreview') }}</div>
            <div v-for="(run, i) in editPreviewNextRuns" :key="i" class="next-run-item">{{ run }}</div>
          </div>
        </div>
        <a-form-item v-if="editingTask && taskNeedsEmail(editingTask.task_type)" :label="t('cron.pushEmail')">
          <a-input v-model:value="editingTask.push_email" :placeholder="t('cron.emailPlaceholder')" size="large">
            <template #prefix><MailOutlined style="color: var(--color-text-tertiary);" /></template>
          </a-input>
          <div class="email-hint">{{ t('cron.emailHint') }}</div>
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

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }

  .task-grid {
    grid-template-columns: 1fr;
  }

  .task-card {
    padding: 16px;
  }

  .task-stats {
    grid-template-columns: 1fr 1fr 1fr;
    gap: 10px;
  }

  .task-actions {
    flex-wrap: wrap;
  }

  .cron-builder-row {
    grid-template-columns: repeat(3, 1fr);
  }

  .cron-builder-weekdays {
    justify-content: center;
  }
}

@media (max-width: 480px) {
  .page-title { font-size: 20px; }

  .task-card { padding: 14px; }

  .task-cron {
    flex-wrap: wrap;
    gap: 4px;
  }

  .cron-desc {
    width: 100%;
    margin-left: 0;
  }

  .task-action-btn {
    width: 32px;
    height: 32px;
    font-size: 14px;
  }

  .cron-builder-row {
    grid-template-columns: repeat(2, 1fr);
  }

  .weekday-chip {
    font-size: 11px;
    padding: 2px 8px;
  }
}
</style>
