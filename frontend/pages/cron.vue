<script setup lang="ts">
import {
  PlusOutlined,
  DeleteOutlined,
  PlayCircleOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  PauseCircleOutlined,
  StopOutlined,
  EditOutlined,
  LockOutlined,
  MailOutlined,
  ScheduleOutlined,
  UnorderedListOutlined,
  ReloadOutlined,
  GlobalOutlined,
  CalendarOutlined,
  LoadingOutlined,
} from '@ant-design/icons-vue'
import { message, Modal } from 'ant-design-vue'
import { createVNode } from 'vue'
import type { CronTask, CreateCronTaskRequest, UpdateCronTaskRequest, CronLog } from '~/types/cron'
import type { CronTaskConfig } from '~/types/cron'
import type { ProcessAuditConfig } from '~/types/audit-config'
import { useI18n } from '~/composables/useI18n'

definePageMeta({ middleware: 'auth' })

const { t } = useI18n()
const { listTasks, createTask, updateTask, deleteTask, toggleTask, executeTask, abortTask, listConfigs, listTaskLogs } = useCronApi()
const { getCronPrefs, listProcesses, listArchiveConfigs } = useSettingsApi()

// ============================================================
// 数据状态
// ============================================================
const tasks = ref<CronTask[]>([])
const configs = ref<CronTaskConfig[]>([])
const auditWorkflowConfigs = ref<ProcessListItem[]>([])
const archiveWorkflowConfigs = ref<AccessibleArchiveConfig[]>([])
const defaultEmail = ref('')
const loading = ref(false)
const pageError = ref('')

// 按 module 分组
const auditTasks = computed(() => tasks.value.filter(t => t.module === 'audit'))
const archiveTasks = computed(() => tasks.value.filter(t => t.module === 'archive'))

// 仅展示已启用的任务类型
const enabledConfigs = computed(() => configs.value.filter(c => c.is_enabled))

const taskTypeOptions = computed(() =>
  enabledConfigs.value.map(c => ({
    value: c.task_type,
    label: t(`cron.taskType.${c.task_type}` as any) || c.label_zh,
    module: c.module,
    batchLimit: c.batch_limit,
  }))
)

const workflowOptions = computed(() => {
  const taskType = newTask.value.task_type || (editingTask.value?.task_type || '')
  if (taskType.startsWith('archive_')) {
    return archiveWorkflowConfigs.value.map(c => ({
      value: c.process_type,
      label: c.process_type, // 根据用户习惯，process_type 才是名称
    }))
  }
  return auditWorkflowConfigs.value.map(c => ({
    value: c.process_type,
    label: c.process_type, // 根据用户习惯，process_type 才是名称
  }))
})

// 合并所有流程配置，用于回显展示名称
const allWorkflowConfigs = computed(() => {
  return [...auditWorkflowConfigs.value, ...archiveWorkflowConfigs.value]
})

const dateRangeOptions = [
  { label: t('cron.dateRange.30'), value: 30 },
  { label: t('cron.dateRange.90'), value: 90 },
  { label: t('cron.dateRange.365'), value: 365 },
]

// 判断任务类型是否需要推送邮箱（batch 类型不需要）
const taskNeedsEmail = (taskType: string) => !taskType.endsWith('_batch')
// 判断是否为批量任务（需要流程选择和日期范围）
const isBatchTask = (taskType: string) => taskType.endsWith('_batch')

// 获取任务类型的 batch_limit（来自 configs）
const getBatchLimit = (taskType: string): number | null => {
  const cfg = configs.value.find(c => c.task_type === taskType)
  return cfg?.batch_limit ?? null
}

// ============================================================
// 初始化加载
// ============================================================
const fetchData = async () => {
  loading.value = true
  pageError.value = ''
  try {
    const [taskList, configList, auditList, archiveList, prefs] = await Promise.allSettled([
      listTasks(),
      listConfigs(),
      listProcesses(),
      listArchiveConfigs(),
      getCronPrefs(),
    ])
    if (taskList.status === 'fulfilled') tasks.value = taskList.value
    if (configList.status === 'fulfilled') {
      configs.value = configList.value.map(c => ({
        ...c,
        batch_limit: c.batch_limit ?? 10
      }))
    }
    if (auditList.status === 'fulfilled') auditWorkflowConfigs.value = auditList.value
    if (archiveList.status === 'fulfilled') archiveWorkflowConfigs.value = archiveList.value
    if (prefs.status === 'fulfilled' && prefs.value?.default_email) {
      defaultEmail.value = prefs.value.default_email
    }
    if (taskList.status === 'rejected') {
      pageError.value = t('cron.loadFailed')
    }
  } catch {
    pageError.value = t('cron.loadFailed')
  } finally {
    loading.value = false
  }
}

// 页面初始化：加载任务列表、配置及偏好数据
onMounted(fetchData)

// 轮询正在运行的任务状态（每 5 秒刷新一次）
let pollTimer: any = null
onMounted(() => {
  pollTimer = setInterval(async () => {
    const runningTasks = tasks.value.filter(t => t.current_log_id)
    if (runningTasks.length > 0) {
      const newList = await listTasks()
      // 局部更新，合并状态
      tasks.value = newList
    }
  }, 5000)
})

onUnmounted(() => {
  if (pollTimer) clearInterval(pollTimer)
})

// ============================================================
// Cron 表达式工具
// ============================================================
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

const weekdayOptions = computed(() => [
  { label: t('cron.weekday.mon'), value: '1' },
  { label: t('cron.weekday.tue'), value: '2' },
  { label: t('cron.weekday.wed'), value: '3' },
  { label: t('cron.weekday.thu'), value: '4' },
  { label: t('cron.weekday.fri'), value: '5' },
  { label: t('cron.weekday.sat'), value: '6' },
  { label: t('cron.weekday.sun'), value: '0' },
])

const expandWeekdays = (weekdayStr: string): Set<string> => {
  if (weekdayStr === '*') return new Set(['0', '1', '2', '3', '4', '5', '6'])
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

const isWeekdayActive = (weekdayStr: string, dayValue: string): boolean =>
  expandWeekdays(weekdayStr).has(dayValue)

const toggleWeekday = (partsRef: { weekday: string }, dayValue: string) => {
  const current = expandWeekdays(partsRef.weekday)
  if (current.has(dayValue)) current.delete(dayValue)
  else current.add(dayValue)
  if (current.size === 0 || current.size === 7) {
    partsRef.weekday = '*'
  } else {
    partsRef.weekday = [...current].map(Number).sort((a, b) => a - b).map(String).join(',')
  }
}

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

// 使用当前真实时间计算下次执行
const calcNextRuns = (expr: string, count = 3): string[] => {
  const now = new Date()
  const parts = expr.split(' ')
  if (parts.length !== 5) return [t('cron.describe.exprError')]
  const [minStr, hourStr, dayStr, monthStr, weekdayStr] = parts
  const h = parseInt(hourStr)
  const m = parseInt(minStr)
  if (isNaN(h) || isNaN(m)) return [t('cron.describe.pending')]

  const allowedWeekdays = expandWeekdays(weekdayStr)
  const allowedMonths = monthStr !== '*' ? new Set(monthStr.split(',').map(s => s.trim())) : null
  const allowedDays = dayStr !== '*' ? new Set(dayStr.split(',').map(s => s.trim())) : null

  const results: string[] = []
  const candidate = new Date(now)
  candidate.setHours(h, m, 0, 0)
  if (candidate <= now) candidate.setDate(candidate.getDate() + 1)

  let safety = 0
  while (results.length < count && safety < 400) {
    safety++
    const dow = candidate.getDay()
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

// ============================================================
// 新建任务
// ============================================================
const showCreate = ref(false)
const createLoading = ref(false)
const newTask = ref({
  task_type: '',
  task_label: '',
  cron_expression: '0 9 * * 1-5',
  cron_mode: '0 9 * * 1-5' as string,
  push_email: '',
  workflow_ids: [] as string[],
  date_range: 30,
})
const cronParts = ref({ minute: '0', hour: '9', day: '*', month: '*', weekday: '1-5' })


watch(() => newTask.value.task_type, (type) => {
  // 新类型时自动填入预设 cron 表达式
  const cfg = configs.value.find(c => c.task_type === type)
  if (cfg?.default_cron) {
    newTask.value.cron_expression = cfg.default_cron
    newTask.value.cron_mode = cronPresets.value.find(p => p.value === cfg.default_cron && p.value !== 'custom')
      ? cfg.default_cron
      : 'custom'
  }
})

watch(cronParts, () => {
  if (newTask.value.cron_mode === 'custom') {
    newTask.value.cron_expression = `${cronParts.value.minute} ${cronParts.value.hour} ${cronParts.value.day} ${cronParts.value.month} ${cronParts.value.weekday}`
  }
}, { deep: true })

watch(() => newTask.value.cron_mode, (val) => {
  if (val !== 'custom') newTask.value.cron_expression = val
  else newTask.value.cron_expression = `${cronParts.value.minute} ${cronParts.value.hour} ${cronParts.value.day} ${cronParts.value.month} ${cronParts.value.weekday}`
})

const previewNextRuns = computed(() => calcNextRuns(newTask.value.cron_expression))

const openCreate = () => {
  const firstEnabled = enabledConfigs.value[0]
  newTask.value = {
    task_type: firstEnabled?.task_type ?? '',
    task_label: '',
    cron_expression: firstEnabled?.default_cron || '0 9 * * 1-5',
    cron_mode: firstEnabled?.default_cron || '0 9 * * 1-5',
    push_email: defaultEmail.value,
    workflow_ids: [],
    date_range: 30,
  }
  cronParts.value = { minute: '0', hour: '9', day: '*', month: '*', weekday: '1-5' }
  showCreate.value = true
}

const doCreateTask = async () => {
  if (!newTask.value.task_type) {
    message.warning(t('cron.selectTaskType'))
    return
  }
  createLoading.value = true
  try {
    const req: CreateCronTaskRequest = {
      task_type: newTask.value.task_type,
      cron_expression: newTask.value.cron_expression,
      task_label: newTask.value.task_label || undefined,
      push_email: newTask.value.push_email || undefined,
      workflow_ids: isBatchTask(newTask.value.task_type) ? (newTask.value.workflow_ids.length > 0 ? newTask.value.workflow_ids : undefined) : undefined,
      date_range: isBatchTask(newTask.value.task_type) ? newTask.value.date_range : undefined,
    }
    const created = await createTask(req)
    tasks.value.push(created)
    showCreate.value = false
    message.success(t('cron.taskCreated'))
  } catch (e: any) {
    message.error(e?.data?.message || t('cron.loadFailed'))
  } finally {
    createLoading.value = false
  }
}

// ============================================================
// 编辑任务
// ============================================================
const showEdit = ref(false)
const editLoading = ref(false)
const editingTask = ref<CronTask | null>(null)
const editForm = ref({
  task_label: '',
  cron_expression: '0 9 * * 1-5',
  cron_mode: '0 9 * * 1-5',
  push_email: '',
  workflow_ids: [] as string[],
  date_range: 30,
})
const editCronParts = ref({ minute: '0', hour: '9', day: '*', month: '*', weekday: '1-5' })

const editPreviewNextRuns = computed(() => calcNextRuns(editForm.value.cron_expression))

watch(editCronParts, () => {
  if (editForm.value.cron_mode === 'custom') {
    editForm.value.cron_expression = `${editCronParts.value.minute} ${editCronParts.value.hour} ${editCronParts.value.day} ${editCronParts.value.month} ${editCronParts.value.weekday}`
  }
}, { deep: true })

watch(() => editForm.value.cron_mode, (val) => {
  if (val !== 'custom') editForm.value.cron_expression = val
  else editForm.value.cron_expression = `${editCronParts.value.minute} ${editCronParts.value.hour} ${editCronParts.value.day} ${editCronParts.value.month} ${editCronParts.value.weekday}`
})

const openEdit = (task: CronTask) => {
  editingTask.value = task
  editForm.value = {
    task_label: task.task_label,
    cron_expression: task.cron_expression,
    cron_mode: cronPresets.value.find(p => p.value === task.cron_expression && p.value !== 'custom')
      ? task.cron_expression
      : 'custom',
    push_email: task.push_email || defaultEmail.value,
    workflow_ids: task.workflow_ids || [],
    date_range: task.date_range || 30,
  }
  if (editForm.value.cron_mode === 'custom') {
    const parts = task.cron_expression.split(' ')
    if (parts.length === 5) {
      editCronParts.value = { minute: parts[0], hour: parts[1], day: parts[2], month: parts[3], weekday: parts[4] }
    }
  }
  showEdit.value = true
}

const doSaveEdit = async () => {
  if (!editingTask.value) return
  editLoading.value = true
  try {
    const req: UpdateCronTaskRequest = {
      task_label: editForm.value.task_label || undefined,
      cron_expression: editForm.value.cron_expression,
      push_email: editForm.value.push_email,
      workflow_ids: isBatchTask(editingTask.value.task_type) ? (editForm.value.workflow_ids.length > 0 ? editForm.value.workflow_ids : undefined) : undefined,
      date_range: isBatchTask(editingTask.value.task_type) ? editForm.value.date_range : undefined,
    }
    const updated = await updateTask(editingTask.value.id, req)
    const idx = tasks.value.findIndex(t => t.id === updated.id)
    if (idx >= 0) tasks.value[idx] = updated
    showEdit.value = false
    editingTask.value = null
    message.success(t('cron.taskUpdated'))
  } catch (e: any) {
    message.error(e?.data?.message || t('cron.loadFailed'))
  } finally {
    editLoading.value = false
  }
}

// ============================================================
// 删除任务
// ============================================================
const doDeleteTask = async (id: string) => {
  try {
    await deleteTask(id)
    tasks.value = tasks.value.filter(t => t.id !== id)
    message.success(t('cron.deleteSuccess'))
  } catch (e: any) {
    message.error(e?.data?.message || t('cron.loadFailed'))
  }
}

// ============================================================
// 切换启用/禁用
// ============================================================
const doToggleTask = async (id: string) => {
  try {
    const updated = await toggleTask(id)
    const idx = tasks.value.findIndex(t => t.id === id)
    if (idx >= 0) tasks.value[idx] = updated
    message.success(t('cron.toggleSuccess'))
  } catch (e: any) {
    message.error(e?.data?.message || t('cron.loadFailed'))
  }
}

// ============================================================
// 立即执行与中止
// ============================================================
const doExecuteTask = async (id: string) => {
  try {
    await executeTask(id)
    message.success(t('cron.executeTrigger'))
    // 立即刷新列表获取运行态
    setTimeout(fetchData, 1000)
  } catch (e: any) {
    message.error(e?.data?.message || t('cron.loadFailed'))
  }
}

const doAbortTask = async (id: string) => {
  Modal.confirm({
    title: t('cron.abortConfirm'),
    icon: createVNode(StopOutlined, { style: 'color: var(--color-danger)' }),
    onOk: async () => {
      try {
        await abortTask(id)
        message.success(t('cron.abortSuccess'))
        setTimeout(fetchData, 1000)
      } catch (e: any) {
        message.error(e?.data?.message || t('cron.abortFailed'))
      }
    }
  })
}

// ============================================================
// 执行日志抽屉
// ============================================================
const showLogs = ref(false)
const logsTask = ref<CronTask | null>(null)
const logs = ref<CronLog[]>([])
const logsLoading = ref(false)

const openLogs = async (task: CronTask) => {
  logsTask.value = task
  showLogs.value = true
  logsLoading.value = true
  try {
    logs.value = await listTaskLogs(task.id)
  } catch {
    logs.value = []
  } finally {
    logsLoading.value = false
  }
}

const reloadLogs = async () => {
  if (!logsTask.value) return
  logsLoading.value = true
  try {
    logs.value = await listTaskLogs(logsTask.value.id)
  } finally {
    logsLoading.value = false
  }
}

// 日志状态颜色
const logStatusColor = (status: string) => {
  if (status === 'success') return 'var(--color-success)'
  if (status === 'failed') return 'var(--color-danger)'
  return 'var(--color-warning)'
}

// ============================================================
// 任务类型样式
// ============================================================
const taskTypeStyle = (taskType: string): { color: string; bg: string } => {
  if (taskType.startsWith('audit_')) return { color: 'var(--color-primary)', bg: 'var(--color-primary-bg)' }
  if (taskType.startsWith('archive_')) return { color: '#8b5cf6', bg: 'var(--color-primary-bg-alt, #f5f3ff)' }
  return { color: 'var(--color-accent)', bg: 'var(--color-info-bg)' }
}

const taskTypeLabel = (task: CronTask | null): string => {
  if (!task) return ''
  const key = `cron.taskType.${task.task_type}` as any
  return t(key) || task.task_label || task.task_type
}

// 渲染流程标签列表
const renderWorkflowLabels = (workflowIds: string[] | undefined): string => {
  if (!workflowIds || workflowIds.length === 0) return t('common.all')
  return workflowIds.map(id => {
    const cfg = allWorkflowConfigs.value.find(c => c.process_type === id)
    return cfg?.process_type || id // 使用 process_type 作为展示名称
  }).join('、')
}

</script>

<template>
  <div class="cron-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('cron.pageTitle') }}</h1>
        <p class="page-subtitle">{{ t('cron.pageSubtitle') }}</p>
      </div>
      <a-button type="primary" size="large" @click="openCreate" :disabled="enabledConfigs.length === 0">
        <PlusOutlined /> {{ t('cron.createTask') }}
      </a-button>
    </div>

    <!-- 加载错误提示 -->
    <a-alert v-if="pageError" type="error" :message="pageError" show-icon style="margin-bottom: 20px;" />

    <!-- 无启用类型提示 -->
    <a-alert
      v-if="!loading && !pageError && enabledConfigs.length === 0"
      type="warning"
      :message="t('cron.noEnabledTypes')"
      show-icon
      style="margin-bottom: 20px;"
    />

    <a-spin :spinning="loading">
      <!-- ===== 审核工作台分组 ===== -->
      <template v-if="auditTasks.length > 0">
        <div class="module-header">
          <span class="module-icon audit-icon" />
          <span>{{ t('cron.moduleAudit') }}</span>
        </div>
        <div class="task-grid">
          <div
            v-for="task in auditTasks"
            :key="task.id"
            class="task-card"
            :class="{ 'task-card--inactive': !task.is_active }"
          >
            <div class="task-card-header">
              <div class="task-card-header-left">
                <span
                  class="task-type-tag"
                  :style="{ color: taskTypeStyle(task.task_type).color, background: taskTypeStyle(task.task_type).bg }"
                >{{ taskTypeLabel(task) }}</span>
                <span v-if="task.is_builtin" class="builtin-tag">
                  <LockOutlined /> {{ t('cron.builtin') }}
                </span>
                <span v-if="isBatchTask(task.task_type)" class="batch-badge">
                  <GlobalOutlined /> {{ renderWorkflowLabels(task.workflow_ids) }}
                </span>
                <span v-if="isBatchTask(task.task_type)" class="batch-badge">
                  <CalendarOutlined /> {{ t(`cron.dateRange.${task.date_range || 30}`) }}
                </span>
                <template v-if="task.push_email && taskNeedsEmail(task.task_type)">
                  <span v-for="email in task.push_email.split(',').filter(e => e.trim())" :key="email" class="batch-badge">
                    <MailOutlined /> {{ email.trim() }}
                  </span>
                </template>
              </div>
              <div class="task-status" :class="task.is_active ? 'task-status--active' : 'task-status--paused'">
                <span class="task-status-dot" />
                {{ task.is_active ? t('cron.running') : t('cron.paused') }}
              </div>
            </div>
            <div v-if="task.task_label" class="task-custom-label">{{ task.task_label }}</div>
            <div class="task-cron">
              <ClockCircleOutlined />
              <code>{{ task.cron_expression }}</code>
              <span class="cron-desc">{{ describeCron(task.cron_expression) }}</span>
            </div>

            <!-- 运行中的特殊状态显示 -->
            <div v-if="task.current_log_id" class="task-running-box">
              <div class="running-info">
                <LoadingOutlined spin />
                <span>{{ t('cron.runningDesc') }}</span>
              </div>
              <a-button danger size="small" type="ghost" @click="doAbortTask(task.id)">
                <StopOutlined /> {{ t('cron.abort') }}
              </a-button>
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
                <span class="task-stat-value">{{ task.last_run_at ? task.last_run_at.slice(0, 16).replace('T', ' ') : '—' }}</span>
                <span class="task-stat-label">{{ t('cron.lastExec') }}</span>
              </div>
            </div>
            <div class="task-actions">
              <a-tooltip :title="t('cron.executeNow')">
                <button class="task-action-btn task-action-btn--run" :disabled="!!task.current_log_id" @click="doExecuteTask(task.id)"><PlayCircleOutlined /></button>
              </a-tooltip>
              <a-tooltip :title="task.is_active ? t('cron.pause') : t('cron.enable')">
                <button class="task-action-btn task-action-btn--toggle" @click="doToggleTask(task.id)">
                  <PauseCircleOutlined v-if="task.is_active" /><CheckCircleOutlined v-else />
                </button>
              </a-tooltip>
              <a-tooltip :title="t('cron.edit')">
                <button class="task-action-btn" @click="openEdit(task)"><EditOutlined /></button>
              </a-tooltip>
              <a-tooltip :title="t('cron.viewLogs')">
                <button class="task-action-btn" @click="openLogs(task)"><UnorderedListOutlined /></button>
              </a-tooltip>
              <a-popconfirm v-if="!task.is_builtin" :title="t('cron.deleteConfirm')" @confirm="doDeleteTask(task.id)">
                <a-tooltip :title="t('cron.delete')">
                  <button class="task-action-btn task-action-btn--delete"><DeleteOutlined /></button>
                </a-tooltip>
              </a-popconfirm>
              <a-tooltip v-else :title="t('cron.builtinNoDelete')">
                <button class="task-action-btn task-action-btn--disabled" disabled><DeleteOutlined /></button>
              </a-tooltip>
            </div>
          </div>
        </div>
      </template>

      <!-- ===== 归档复盘分组 ===== -->
      <template v-if="archiveTasks.length > 0">
        <div class="module-header" :style="auditTasks.length > 0 ? 'margin-top: 28px;' : ''">
          <span class="module-icon archive-icon" />
          <span>{{ t('cron.moduleArchive') }}</span>
        </div>
        <div class="task-grid">
          <div
            v-for="task in archiveTasks"
            :key="task.id"
            class="task-card"
            :class="{ 'task-card--inactive': !task.is_active }"
          >
            <div class="task-card-header">
              <div class="task-card-header-left">
                <span
                  class="task-type-tag"
                  :style="{ color: taskTypeStyle(task.task_type).color, background: taskTypeStyle(task.task_type).bg }"
                >{{ taskTypeLabel(task) }}</span>
                <span v-if="task.is_builtin" class="builtin-tag">
                  <LockOutlined /> {{ t('cron.builtin') }}
                </span>
                <span v-if="isBatchTask(task.task_type)" class="batch-badge">
                  <GlobalOutlined /> {{ renderWorkflowLabels(task.workflow_ids) }}
                </span>
                <span v-if="isBatchTask(task.task_type)" class="batch-badge">
                  <CalendarOutlined /> {{ t(`cron.dateRange.${task.date_range || 30}`) }}
                </span>
                <template v-if="task.push_email && taskNeedsEmail(task.task_type)">
                  <span v-for="email in task.push_email.split(',').filter(e => e.trim())" :key="email" class="batch-badge">
                    <MailOutlined /> {{ email.trim() }}
                  </span>
                </template>
              </div>
              <div class="task-status" :class="task.is_active ? 'task-status--active' : 'task-status--paused'">
                <span class="task-status-dot" />
                {{ task.is_active ? t('cron.running') : t('cron.paused') }}
              </div>
            </div>
            <div v-if="task.task_label" class="task-custom-label">{{ task.task_label }}</div>
            <div class="task-cron">
              <ClockCircleOutlined />
              <code>{{ task.cron_expression }}</code>
              <span class="cron-desc">{{ describeCron(task.cron_expression) }}</span>
            </div>

            <!-- 运行中的特殊状态显示 -->
            <div v-if="task.current_log_id" class="task-running-box">
              <div class="running-info">
                <LoadingOutlined spin />
                <span>{{ t('cron.runningDesc') }}</span>
              </div>
              <a-button danger size="small" type="ghost" @click="doAbortTask(task.id)">
                <StopOutlined /> {{ t('cron.abort') }}
              </a-button>
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
                <span class="task-stat-value">{{ task.last_run_at ? task.last_run_at.slice(0, 16).replace('T', ' ') : '—' }}</span>
                <span class="task-stat-label">{{ t('cron.lastExec') }}</span>
              </div>
            </div>
            <div class="task-actions">
              <a-tooltip :title="t('cron.executeNow')">
                <button class="task-action-btn task-action-btn--run" :disabled="!!task.current_log_id" @click="doExecuteTask(task.id)"><PlayCircleOutlined /></button>
              </a-tooltip>
              <a-tooltip :title="task.is_active ? t('cron.pause') : t('cron.enable')">
                <button class="task-action-btn task-action-btn--toggle" @click="doToggleTask(task.id)">
                  <PauseCircleOutlined v-if="task.is_active" /><CheckCircleOutlined v-else />
                </button>
              </a-tooltip>
              <a-tooltip :title="t('cron.edit')">
                <button class="task-action-btn" @click="openEdit(task)"><EditOutlined /></button>
              </a-tooltip>
              <a-tooltip :title="t('cron.viewLogs')">
                <button class="task-action-btn" @click="openLogs(task)"><UnorderedListOutlined /></button>
              </a-tooltip>
              <a-popconfirm v-if="!task.is_builtin" :title="t('cron.deleteConfirm')" @confirm="doDeleteTask(task.id)">
                <a-tooltip :title="t('cron.delete')">
                  <button class="task-action-btn task-action-btn--delete"><DeleteOutlined /></button>
                </a-tooltip>
              </a-popconfirm>
              <a-tooltip v-else :title="t('cron.builtinNoDelete')">
                <button class="task-action-btn task-action-btn--disabled" disabled><DeleteOutlined /></button>
              </a-tooltip>
            </div>
          </div>
        </div>
      </template>

      <!-- 无任务占位 -->
      <div v-if="!loading && !pageError && tasks.length === 0 && enabledConfigs.length > 0" class="empty-state">
        <ScheduleOutlined class="empty-icon" />
        <p>{{ t('cron.noTasks') }}</p>
      </div>
    </a-spin>

    <!-- ===== 新建任务弹窗 ===== -->
    <a-modal
      v-model:open="showCreate"
      :title="t('cron.createTitle')"
      @ok="doCreateTask"
      :okText="t('cron.create')"
      :cancelText="t('cron.cancel')"
      :confirmLoading="createLoading"
      :width="580"
    >
      <a-form layout="vertical" style="margin-top: 16px;">
        <!-- 任务类型 -->
        <a-form-item :label="t('cron.taskType')">
          <a-select
            v-model:value="newTask.task_type"
            :options="taskTypeOptions"
            size="large"
            :placeholder="t('cron.selectTaskType')"
          />
          <template v-if="newTask.task_type">
            <div class="task-type-hint">
              {{ t(`cron.taskType.${newTask.task_type}.desc` as any) }}
            </div>
            <div v-if="newTask.task_type.endsWith('_batch') && getBatchLimit(newTask.task_type)" class="task-type-hint">
              {{ t('cron.batchLimit', getBatchLimit(newTask.task_type)!) }}
            </div>
          </template>
        </a-form-item>

        <!-- 任务标签 -->
        <a-form-item :label="t('cron.taskLabel')">
          <a-input
            v-model:value="newTask.task_label"
            :placeholder="t('cron.taskLabelPlaceholder')"
            size="large"
          />
        </a-form-item>

        <!-- 批量任务特有：流程选择 & 日期范围 -->
        <template v-if="isBatchTask(newTask.task_type)">
          <a-form-item :label="t('cron.selectWorkflows')">
            <a-select
              v-model:value="newTask.workflow_ids"
              mode="multiple"
              :options="workflowOptions"
              :placeholder="t('cron.selectWorkflowsPlaceholder')"
              size="large"
              show-search
              option-filter-prop="label"
            />
          </a-form-item>
          <a-form-item :label="t('cron.dateRangeLabel')">
            <a-radio-group v-model:value="newTask.date_range" button-style="solid">
              <a-radio-button v-for="opt in dateRangeOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </a-radio-button>
            </a-radio-group>
          </a-form-item>
        </template>

        <!-- 执行计划 -->
        <a-form-item :label="t('cron.executePlan')">
          <a-select v-model:value="newTask.cron_mode" size="large" style="width: 100%;">
            <a-select-option v-for="p in cronPresets" :key="p.value" :value="p.value">
              {{ p.label }}
              <span v-if="p.value !== 'custom'" style="color: var(--color-text-tertiary); margin-left: 8px; font-family: monospace; font-size: 12px;">{{ p.value }}</span>
            </a-select-option>
          </a-select>
        </a-form-item>

        <!-- 自定义 cron 构建器 -->
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

        <!-- 下次执行预览 -->
        <div class="next-run-preview">
          <ScheduleOutlined />
          <div>
            <div class="next-run-title">{{ t('cron.nextRunPreview') }}</div>
            <div v-for="(run, i) in previewNextRuns" :key="i" class="next-run-item">{{ run }}</div>
          </div>
        </div>

        <!-- 推送邮箱（仅报告类） -->
        <a-form-item v-if="taskNeedsEmail(newTask.task_type)" :label="t('cron.pushEmail')">
          <a-input v-model:value="newTask.push_email" :placeholder="t('cron.emailPlaceholder')" size="large">
            <template #prefix><MailOutlined style="color: var(--color-text-tertiary);" /></template>
          </a-input>
          <div class="email-hint">{{ t('cron.emailHint') }}</div>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- ===== 编辑任务弹窗 ===== -->
    <a-modal
      v-model:open="showEdit"
      :title="t('cron.editTitle')"
      @ok="doSaveEdit"
      :okText="t('cron.save')"
      :cancelText="t('cron.cancel')"
      :confirmLoading="editLoading"
      :width="580"
    >
      <a-form v-if="editingTask" layout="vertical" style="margin-top: 16px;">
        <!-- 任务类型（只读） -->
        <a-form-item :label="t('cron.taskType')">
          <a-input :value="taskTypeLabel(editingTask)" disabled size="large" />
        </a-form-item>

        <!-- 任务标签 -->
        <a-form-item :label="t('cron.taskLabel')">
          <a-input v-model:value="editForm.task_label" :placeholder="t('cron.taskLabelPlaceholder')" size="large" />
        </a-form-item>

        <!-- 批量任务特有：流程选择 & 日期范围 -->
        <template v-if="isBatchTask(editingTask.task_type)">
          <a-form-item :label="t('cron.selectWorkflows')">
            <a-select
              v-model:value="editForm.workflow_ids"
              mode="multiple"
              :options="workflowOptions"
              :placeholder="t('cron.selectWorkflowsPlaceholder')"
              size="large"
              show-search
              option-filter-prop="label"
            />
          </a-form-item>
          <a-form-item :label="t('cron.dateRangeLabel')">
            <a-radio-group v-model:value="editForm.date_range" button-style="solid">
              <a-radio-button v-for="opt in dateRangeOptions" :key="opt.value" :value="opt.value">
                {{ opt.label }}
              </a-radio-button>
            </a-radio-group>
          </a-form-item>
        </template>

        <!-- 执行计划 -->
        <a-form-item :label="t('cron.executePlan')">
          <a-select v-model:value="editForm.cron_mode" size="large" style="width: 100%;">
            <a-select-option v-for="p in cronPresets" :key="p.value" :value="p.value">
              {{ p.label }}
              <span v-if="p.value !== 'custom'" style="color: var(--color-text-tertiary); margin-left: 8px; font-family: monospace; font-size: 12px;">{{ p.value }}</span>
            </a-select-option>
          </a-select>
        </a-form-item>

        <!-- 自定义构建器 -->
        <div v-if="editForm.cron_mode === 'custom'" class="cron-builder">
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
            <code>{{ editForm.cron_expression }}</code>
          </div>
        </div>

        <!-- 下次执行预览 -->
        <div class="next-run-preview">
          <ScheduleOutlined />
          <div>
            <div class="next-run-title">{{ t('cron.nextRunPreview') }}</div>
            <div v-for="(run, i) in editPreviewNextRuns" :key="i" class="next-run-item">{{ run }}</div>
          </div>
        </div>

        <!-- 推送邮箱（仅报告类） -->
        <a-form-item v-if="taskNeedsEmail(editingTask.task_type)" :label="t('cron.pushEmail')">
          <a-input v-model:value="editForm.push_email" :placeholder="t('cron.emailPlaceholder')" size="large">
            <template #prefix><MailOutlined style="color: var(--color-text-tertiary);" /></template>
          </a-input>
          <div class="email-hint">{{ t('cron.emailHint') }}</div>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- ===== 执行日志抽屉 ===== -->
    <a-drawer
      v-model:open="showLogs"
      :title="logsTask ? `${t('cron.logsTitle')} — ${logsTask.task_label || taskTypeLabel(logsTask)}` : t('cron.logsTitle')"
      placement="right"
      :width="520"
    >
      <template #extra>
        <a-button size="small" @click="reloadLogs" :loading="logsLoading">
          <ReloadOutlined /> {{ t('cron.run') }}
        </a-button>
      </template>
      <a-spin :spinning="logsLoading">
        <div v-if="logs.length === 0 && !logsLoading" class="logs-empty">
          <UnorderedListOutlined style="font-size: 32px; opacity: 0.3;" />
          <p>{{ t('cron.logsEmpty') }}</p>
        </div>
        <div v-for="log in logs" :key="log.id" class="log-item">
          <div class="log-item-header">
            <span class="log-status" :style="{ color: logStatusColor(log.status) }">
              ● {{ t(`cron.logStatus.${log.status}` as any) }}
            </span>
            <span class="log-time">{{ log.started_at?.slice(0, 19).replace('T', ' ') }}</span>
          </div>
          <div v-if="log.message" class="log-message">{{ log.message }}</div>
          <div v-if="log.finished_at" class="log-duration">
            {{ t('cron.lastExec') }}：{{ log.finished_at?.slice(0, 19).replace('T', ' ') }}
          </div>
        </div>
      </a-spin>
    </a-drawer>
  </div>
</template>

<!-- task card styles are scoped in the style block below -->

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


/* 模块分组标题 */
.module-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-secondary);
  margin-bottom: 14px;
}

.module-icon {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  display: inline-block;
}

.audit-icon { background: var(--color-primary); }
.archive-icon { background: #8b5cf6; }

/* 空状态 */
.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: var(--color-text-tertiary);
}

.empty-icon {
  font-size: 48px;
  opacity: 0.25;
  margin-bottom: 12px;
}

/* 任务卡片网格 */
.task-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 20px;
  margin-bottom: 8px;
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

.task-card--inactive { opacity: 0.65; }

/* 任务卡片内部 */
.task-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 12px;
}

.task-card-header-left {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  flex: 1;
  min-width: 0;
}

.task-type-tag {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 10px;
  border-radius: 6px;
  white-space: nowrap;
  display: inline-flex;
  align-items: center;
  height: 22px;
}

.builtin-tag {
  font-size: 10px;
  font-weight: 600;
  padding: 0 8px;
  border-radius: 6px;
  background: var(--color-warning-bg);
  color: var(--color-warning);
  display: inline-flex;
  align-items: center;
  gap: 3px;
  height: 22px;
}

.task-custom-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 10px;
}

.task-status {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 500;
  white-space: nowrap;
  margin-left: 12px;
  align-self: flex-start;
  padding-top: 2px;
}

.task-status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
}

.task-status--active { color: var(--color-success); }
.task-status--active .task-status-dot {
  background: var(--color-success);
  box-shadow: 0 0 0 3px rgba(16, 185, 129, 0.2);
  animation: blink 2s ease-in-out infinite;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

.task-status--paused { color: var(--color-text-tertiary); }
.task-status--paused .task-status-dot { background: var(--color-text-tertiary); }

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
  padding: 4px 14px;
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

.task-stat { display: flex; flex-direction: column; }
.task-stat-value { font-size: 14px; font-weight: 600; }
.task-stat-label { font-size: 11px; color: var(--color-text-tertiary); margin-top: 2px; }

.task-actions { display: flex; gap: 8px; flex-wrap: wrap; }

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

/* 类型标签 */
.task-type-hint {
  font-size: 12px;
  color: var(--color-text-tertiary);
  margin-top: 6px;
}

/* Cron 表达式构建器 */
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

.weekday-chip:hover { border-color: var(--color-primary); }

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

/* 下次执行预览 */
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

/* 日志抽屉 */
.logs-empty {
  text-align: center;
  padding: 40px;
  color: var(--color-text-tertiary);
}

.log-item {
  padding: 12px 0;
  border-bottom: 1px solid var(--color-border-light);
}

.log-item:last-child { border-bottom: none; }

.log-item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.log-status {
  font-size: 13px;
  font-weight: 600;
}

.log-time {
  font-size: 12px;
  color: var(--color-text-tertiary);
  font-family: var(--font-mono);
}

.log-message {
  font-size: 12px;
  color: var(--color-text-secondary);
  margin-top: 4px;
  white-space: pre-wrap;
  word-break: break-all;
}

.log-duration {
  font-size: 11px;
  color: var(--color-text-tertiary);
  margin-top: 2px;
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }

  .task-grid { grid-template-columns: 1fr; }
  .task-card { padding: 16px; }
  .cron-builder-row { grid-template-columns: repeat(3, 1fr); }
  .cron-builder-weekdays { justify-content: center; }
}

@media (max-width: 480px) {
  .page-title { font-size: 20px; }
  .task-card { padding: 14px; }
  .cron-builder-row { grid-template-columns: repeat(2, 1fr); }
  .weekday-chip { font-size: 11px; padding: 2px 8px; }
}

/* 批量任务专用徽章 */
.batch-badge {
  font-size: 11px;
  font-weight: 500;
  padding: 0 10px;
  border-radius: 6px;
  background: var(--color-bg-page);
  border: 1px solid var(--color-border-light);
  color: var(--color-text-secondary);
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 22px;
  max-width: 240px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* 运行中状态条 */
.task-running-box {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 14px;
  padding: 10px 14px;
  background: var(--color-primary-bg);
  border-radius: var(--radius-md);
  border: 1px solid var(--color-primary-light);
}

.running-info {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: var(--color-primary);
  font-weight: 600;
}
</style>
