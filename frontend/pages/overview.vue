<script setup lang="ts">
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  ClockCircleOutlined,
  ThunderboltOutlined,
  RiseOutlined,
  TeamOutlined,
  SafetyCertificateOutlined,
  CloudServerOutlined,
  SettingOutlined,
  EyeOutlined,
  EyeInvisibleOutlined,
  AppstoreOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { OVERVIEW_WIDGETS, OVERVIEW_WIDGET_ID_SET } from '~/constants/overviewWidgets'
import { useI18n } from '~/composables/useI18n'
import type { OverviewWidgetId } from '~/constants/overviewWidgets'
import type { DashboardOverview, DashboardActivityItem, PlatformDashboardOverview } from '~/types/dashboard-overview'
import type { CronTask } from '~/types/cron'
import type { DashboardPref } from '~/types/user-config'

definePageMeta({ middleware: 'auth' })

const EMPTY_OVERVIEW: DashboardOverview = {
  pending_oa_count: 0,
  audit_summary: { total: 0, approved: 0, returned: 0, archived: 0, review: 0, pending_ai: 0 },
  weekly_trend: [],
  recent_activity: [],
  archive_recent: [],
}

const { activeRole, currentUser } = useAuth()
const { t, locale } = useI18n()
const { fetchDashboardOverview, fetchPlatformDashboardOverview } = useDashboardOverviewApi()
const { getDashboardPrefs, updateDashboardPrefs } = useSettingsApi()
const { listTasks } = useCronApi()

const overview = ref<DashboardOverview | null>(null)
const platformOverview = ref<PlatformDashboardOverview | null>(null)
const overviewLoading = ref(false)
const cronTasksList = ref<CronTask[]>([])

const isPlatformAdmin = computed(() => activeRole.value?.role === 'system_admin')

function mergePlatformToDash(p: PlatformDashboardOverview): DashboardOverview {
  return {
    pending_oa_count: p.pending_oa_count,
    audit_summary: p.audit_summary,
    weekly_trend: p.weekly_trend,
    recent_activity: p.recent_activity,
    archive_recent: p.archive_recent,
    ai_performance: p.ai_performance,
    tenant_total: p.tenant_total,
    tenant_active: p.tenant_active,
    tenant_ranking: p.tenant_ranking,
    tenant_usage: p.token_summary
      ? {
          token_used: p.token_summary.total_used,
          token_quota: p.token_summary.total_quota,
          storage_used_mb: 0,
          storage_quota_mb: 0,
          active_users: 0,
          total_users: 0,
        }
      : undefined,
  }
}

const dash = computed(() => {
  if (isPlatformAdmin.value && platformOverview.value)
    return mergePlatformToDash(platformOverview.value)
  return overview.value ?? EMPTY_OVERVIEW
})

const availableWidgets = computed(() => {
  const role = activeRole.value?.role
  if (!role) return []
  return OVERVIEW_WIDGETS.filter(w => w.requiredPermissions.includes(role))
})

const defaultPrefs = computed(() =>
  availableWidgets.value.filter(w => w.defaultEnabled).map(w => w.id))

const enabledWidgets = ref<OverviewWidgetId[]>([])
const widgetSizes = ref<Partial<Record<OverviewWidgetId, 'sm' | 'md' | 'lg'>>>({})

function applyDashboardPrefs(prefs: DashboardPref) {
  const allowed = new Set(availableWidgets.value.map(w => w.id))
  const raw = (prefs.enabled_widgets || []).filter((id): id is OverviewWidgetId =>
    OVERVIEW_WIDGET_ID_SET.has(id as OverviewWidgetId) && allowed.has(id as OverviewWidgetId))
  enabledWidgets.value = raw.length > 0 ? raw : [...defaultPrefs.value]
  widgetSizes.value = { ...(prefs.widget_sizes as Partial<Record<OverviewWidgetId, 'sm' | 'md' | 'lg'>> || {}) }
}

async function loadOverviewPage() {
  if (!activeRole.value?.role) return
  overviewLoading.value = true
  try {
    const prefs = await getDashboardPrefs().catch(() => ({ enabled_widgets: [] as string[], widget_sizes: {} }))
    if (isPlatformAdmin.value) {
      platformOverview.value = await fetchPlatformDashboardOverview()
      overview.value = null
      cronTasksList.value = []
    }
    else {
      platformOverview.value = null
      overview.value = await fetchDashboardOverview()
      cronTasksList.value = await listTasks()
    }
    applyDashboardPrefs(prefs)
  }
  catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e)
    message.error(msg || t('overview.loadFailed'))
  }
  finally {
    overviewLoading.value = false
  }
}

watch(activeRole, () => { void loadOverviewPage() }, { immediate: true, deep: true })

const isEnabled = (id: OverviewWidgetId) => {
  return enabledWidgets.value.includes(id) && availableWidgets.value.some(w => w.id === id)
}

const customizing = ref(false)
const toggleWidget = (id: OverviewWidgetId) => {
  if (!customizing.value) return
  const idx = enabledWidgets.value.indexOf(id)
  if (idx >= 0) enabledWidgets.value.splice(idx, 1)
  else enabledWidgets.value.push(id)
}

const savePrefs = async () => {
  customizing.value = false
  try {
    await updateDashboardPrefs({
      enabled_widgets: [...enabledWidgets.value],
      widget_sizes: { ...widgetSizes.value },
    })
    message.success(t('overview.layoutSaved'))
  }
  catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e)
    message.error(msg || t('overview.saveLayoutFailed'))
  }
}

const getWidgetSize = (id: OverviewWidgetId) => {
  if (widgetSizes.value[id]) return widgetSizes.value[id]
  const w = OVERVIEW_WIDGETS.find(x => x.id === id)
  return w?.size || 'md'
}

const cycleWidgetSize = (id: OverviewWidgetId) => {
  const current = getWidgetSize(id)
  const nextSize = current === 'sm' ? 'md' : current === 'md' ? 'lg' : 'sm'
  widgetSizes.value[id] = nextSize
}

const greeting = computed(() => {
  const h = new Date().getHours()
  return h < 6 ? t('overview.greeting.lateNight') : h < 12 ? t('overview.greeting.morning') : h < 14 ? t('overview.greeting.noon') : h < 18 ? t('overview.greeting.afternoon') : t('overview.greeting.evening')
})

const formatNum = (n: number) => n >= 10000 ? (n / 1000).toFixed(1) + 'K' : n.toLocaleString()

const DEPT_COLORS = ['#4f46e5', '#06b6d4', '#f59e0b', '#10b981', '#ef4444', '#8b5cf6', '#ec4899', '#6366f1', '#14b8a6', '#f97316']

const deptRows = computed(() => {
  const rows = dash.value.dept_distribution ?? []
  return rows.map((d, i) => ({
    department: d.department === '__unassigned__' ? t('overview.deptUnassigned') : d.department,
    count: d.count,
    color: DEPT_COLORS[i % DEPT_COLORS.length],
  }))
})

const activityKindStyle: Record<string, { color: string; bg: string }> = {
  audit_completed: { color: 'var(--color-primary)', bg: 'var(--color-primary-bg)' },
  audit_failed: { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)' },
  cron_log: { color: 'var(--color-accent)', bg: 'rgba(6,182,212,0.1)' },
  archive_reviewed: { color: 'var(--color-success)', bg: 'var(--color-success-bg)' },
}

function activityActionLabel(a: DashboardActivityItem) {
  switch (a.kind) {
    case 'audit_completed': return t('overview.activity.auditCompleted')
    case 'audit_failed': return t('overview.activity.auditFailed')
    case 'cron_log': return t('overview.activity.cronLog')
    case 'archive_reviewed': return t('overview.activity.archiveReviewed')
    default: return a.kind
  }
}

function formatActivityTime(iso: string) {
  try {
    const d = new Date(iso)
    return new Intl.DateTimeFormat(locale.value === 'en-US' ? 'en-US' : 'zh-CN', {
      month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit',
    }).format(d)
  }
  catch {
    return iso
  }
}

function formatUserLastActive(iso: string) {
  try {
    const d = new Date(iso)
    return new Intl.DateTimeFormat(locale.value === 'en-US' ? 'en-US' : 'zh-CN', {
      year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit',
    }).format(d)
  }
  catch {
    return iso
  }
}

function archiveComplianceLabel(compliance: string) {
  if (compliance === 'compliant') return t('overview.archiveCompliance.compliant')
  if (compliance === 'non_compliant') return t('overview.archiveCompliance.nonCompliant')
  if (compliance === 'partially_compliant') return t('overview.archiveCompliance.partial')
  return compliance
}

function cronTaskLabel(task: CronTask) {
  if (task.task_label?.trim()) return task.task_label
  const key = `cron.taskType.${task.task_type}` as const
  const tr = t(key as 'cron.taskType.audit_batch')
  return tr && tr !== key ? tr : task.task_type
}

const cronPreviewTasks = computed(() => {
  return [...cronTasksList.value].sort((a, b) =>
    new Date(b.updated_at).getTime() - new Date(a.updated_at).getTime()).slice(0, 5)
})

const trendMax = computed(() => Math.max(...dash.value.weekly_trend.map(x => x.count), 1))
const deptMax = computed(() => Math.max(...deptRows.value.map(d => d.count), 1))

const aiBarMaxMs = computed(() => {
  const stats = dash.value.ai_performance?.daily_stats ?? []
  return Math.max(...stats.map(s => s.avg_ms), 1)
})

const storagePct = computed(() => {
  const u = dash.value.tenant_usage
  if (!u || u.storage_quota_mb <= 0) return 0
  return Math.min(100, (u.storage_used_mb / u.storage_quota_mb) * 100)
})

const getWidgetOrder = (id: OverviewWidgetId) => {
  const index = enabledWidgets.value.indexOf(id)
  return index >= 0 ? index : 999
}

const draggedWidget = ref<OverviewWidgetId | null>(null)

const onDragStart = (e: DragEvent, id: OverviewWidgetId) => {
  if (!customizing.value) return
  draggedWidget.value = id
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.dropEffect = 'move'
    e.dataTransfer.setData('text/plain', id)
  }
}

const onDrop = (e: DragEvent, targetId: OverviewWidgetId) => {
  if (!customizing.value || !draggedWidget.value) return
  if (draggedWidget.value === targetId) return

  const dragIndex = enabledWidgets.value.indexOf(draggedWidget.value)
  const targetIndex = enabledWidgets.value.indexOf(targetId)

  if (dragIndex >= 0 && targetIndex >= 0) {
    const [item] = enabledWidgets.value.splice(dragIndex, 1)
    enabledWidgets.value.splice(targetIndex, 0, item)
  }
  draggedWidget.value = null
}
</script>

<template>
  <div class="overview-page fade-in">
    <a-spin :spinning="overviewLoading" :tip="t('overview.loading')">
    <div class="ov-header">
      <div>
        <h1 class="ov-title">{{ greeting }}，{{ currentUser?.display_name || t('sidebar.defaultUser') }}</h1>
        <p class="ov-subtitle">{{ isPlatformAdmin ? t('overview.subtitlePlatform') : t('overview.subtitle') }}</p>
      </div>
      <a-button :type="customizing ? 'primary' : 'default'" @click="customizing ? savePrefs() : (customizing = true)">
        <SettingOutlined /> {{ customizing ? t('overview.saveLayout') : t('overview.customizeDashboard') }}
      </a-button>
    </div>

    <!--自定义面板-->
    <transition name="slide-down">
      <div v-if="customizing" class="customize-panel">
        <p class="customize-hint">{{ t('overview.customizeHint') }}</p>
        <div class="customize-grid">
          <div v-for="w in availableWidgets" :key="w.id" class="customize-chip" :class="{ 'customize-chip--active': isEnabled(w.id) }" @click="toggleWidget(w.id)">
            <component :is="isEnabled(w.id) ? EyeOutlined : EyeInvisibleOutlined" />
            <span>{{ t(w.titleKey) }}</span>
          </div>
        </div>
      </div>
    </transition>

    <div class="widget-grid">
      <!--===== 平台：租户规模（system_admin）=====-->
      <div v-if="isEnabled('platform_tenant_stats')"
       :class="['widget', `widget--${getWidgetSize('platform_tenant_stats')}`, { 'widget--editing': customizing }]"
       :style="{ order: getWidgetOrder('platform_tenant_stats') }"
       :draggable="customizing"
       @dragstart="onDragStart($event, 'platform_tenant_stats')"
       @dragover.prevent
       @dragenter.prevent
       @drop="onDrop($event, 'platform_tenant_stats')">
        <div class="widget-title">
          <div class="widget-title-left"><TeamOutlined /> {{ t('overview.widgetTitle.platform_tenant_stats') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('platform_tenant_stats')" :title="t('overview.resizeWidget')" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="summary-cards" style="grid-template-columns: repeat(2, 1fr);">
          <div class="summary-card summary-card--total">
            <div class="summary-num">{{ dash.tenant_total ?? 0 }}</div>
            <div class="summary-label">{{ t('overview.platformTenantsTotal') }}</div>
          </div>
          <div class="summary-card summary-card--approved">
            <div class="summary-num">{{ dash.tenant_active ?? 0 }}</div>
            <div class="summary-label">{{ t('overview.platformTenantsActive') }}</div>
          </div>
        </div>
      </div>

      <!--=====审计摘要（业务）=====-->
      <div v-if="isEnabled('audit_summary')"
       :class="['widget', `widget--${getWidgetSize('audit_summary')}`, { 'widget--editing': customizing }]"
       :style="{ order: getWidgetOrder('audit_summary') }"
       :draggable="customizing"
       @dragstart="onDragStart($event, 'audit_summary')"
       @dragover.prevent
       @dragenter.prevent
       @drop="onDrop($event, 'audit_summary')">
        <div class="widget-title">
          <div class="widget-title-left"><ThunderboltOutlined /> {{ t('overview.auditOverview') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('audit_summary')" :title="t('overview.resizeWidget')" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="summary-cards">
          <div class="summary-card summary-card--total">
            <div class="summary-num">{{ dash.audit_summary.total }}</div>
            <div class="summary-label">{{ t('overview.totalAudits') }}</div>
          </div>
          <div class="summary-card summary-card--approved">
            <CheckCircleOutlined class="summary-icon" />
            <div class="summary-num">{{ dash.audit_summary.approved }}</div>
            <div class="summary-label">{{ t('overview.approved') }}</div>
          </div>
          <div class="summary-card summary-card--returned">
            <CloseCircleOutlined class="summary-icon" />
            <div class="summary-num">{{ dash.audit_summary.returned }}</div>
            <div class="summary-label">{{ t('overview.rejected') }}</div>
          </div>
          <div class="summary-card summary-card--archived">
            <CheckCircleOutlined class="summary-icon" />
            <div class="summary-num">{{ dash.audit_summary.archived }}</div>
            <div class="summary-label">{{ t('dashboard.tab.archived') }}</div>
          </div>
        </div>
      </div>

      <!--=====待处理任务（业务）=====-->
      <div v-if="isEnabled('pending_tasks')"
       :class="['widget', `widget--${getWidgetSize('pending_tasks')}`, { 'widget--editing': customizing }]" 
       :style="{ order: getWidgetOrder('pending_tasks') }" 
       :draggable="customizing" 
       @dragstart="onDragStart($event, 'pending_tasks')" 
       @dragover.prevent 
       @dragenter.prevent 
       @drop="onDrop($event, 'pending_tasks')">
        <div class="widget-title">
          <div class="widget-title-left"><ClockCircleOutlined /> {{ t('overview.pendingTasks') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('pending_tasks')" :title="t('overview.resizeWidget')" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="pending-big">
          <div class="pending-num">{{ dash.pending_oa_count }}</div>
          <div class="pending-label">{{ t('overview.itemsPending') }}</div>
        </div>
        <a-button type="link" size="small" @click="navigateTo('/dashboard')">{{ t('overview.goToWorkbench') }} →</a-button>
      </div>

      <!--=====每周趋势（商业）=====-->
      <div v-if="isEnabled('weekly_trend')"
       :class="['widget', `widget--${getWidgetSize('weekly_trend')}`, { 'widget--editing': customizing }]" 
       :style="{ order: getWidgetOrder('weekly_trend') }" 
       :draggable="customizing" 
       @dragstart="onDragStart($event, 'weekly_trend')" 
       @dragover.prevent 
       @dragenter.prevent 
       @drop="onDrop($event, 'weekly_trend')">
        <div class="widget-title">
          <div class="widget-title-left"><RiseOutlined /> {{ t('overview.auditTrend7d') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('weekly_trend')" :title="t('overview.resizeWidget')" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="bar-chart">
          <div v-for="row in dash.weekly_trend" :key="row.date" class="bar-col">
            <div class="bar-value">{{ row.count }}</div>
            <div class="bar" :style="{ height: (row.count / trendMax * 120) + 'px' }" />
            <div class="bar-label">{{ row.date }}</div>
          </div>
        </div>
      </div>

      <!--=====部门分布（业务）=====-->
      <div v-if="isEnabled('dept_distribution')"
       :class="['widget', `widget--${getWidgetSize('dept_distribution')}`, { 'widget--editing': customizing }]" 
       :style="{ order: getWidgetOrder('dept_distribution') }" 
       :draggable="customizing" 
       @dragstart="onDragStart($event, 'dept_distribution')" 
       @dragover.prevent 
       @dragenter.prevent 
       @drop="onDrop($event, 'dept_distribution')">
        <div class="widget-title">
          <div class="widget-title-left"><TeamOutlined /> {{ t('overview.deptDistribution') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('dept_distribution')" :title="t('overview.resizeWidget')" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="dept-list">
          <div v-for="d in deptRows" :key="d.department" class="dept-row">
            <span class="dept-name">{{ d.department }}</span>
            <div class="dept-bar-wrap">
              <div class="dept-bar" :style="{ width: (d.count / deptMax * 100) + '%', background: d.color }" />
            </div>
            <span class="dept-count">{{ d.count }}</span>
          </div>
        </div>
      </div>

      <!--===== Cron 任务（业务） =====-->
      <div v-if="isEnabled('cron_tasks')"
       :class="['widget', `widget--${getWidgetSize('cron_tasks')}`, { 'widget--editing': customizing }]" 
       :style="{ order: getWidgetOrder('cron_tasks') }" 
       :draggable="customizing" 
       @dragstart="onDragStart($event, 'cron_tasks')" 
       @dragover.prevent 
       @dragenter.prevent 
       @drop="onDrop($event, 'cron_tasks')">
        <div class="widget-title">
          <div class="widget-title-left"><ClockCircleOutlined /> {{ t('overview.widgetTitle.cron_tasks') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('cron_tasks')" :title="t('overview.resizeWidget')" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="rank-list" style="margin-top: 10px;">
          <div v-for="c in cronPreviewTasks" :key="c.id" class="rank-item">
            <div class="rank-info">
              <span class="rank-name">{{ cronTaskLabel(c) }}</span>
              <span class="rank-dept">{{ c.cron_expression }}</span>
            </div>
            <span class="rank-count" :style="{ color: c.is_active ? 'var(--color-success)' : 'var(--color-text-tertiary)' }">{{ c.is_active ? t('overview.cronActive') : t('overview.cronInactive') }}</span>
          </div>
        </div>
      </div>

      <!--=====档案审查（商业）=====-->
      <div v-if="isEnabled('archive_review')"
       :class="['widget', `widget--${getWidgetSize('archive_review')}`, { 'widget--editing': customizing }]" 
       :style="{ order: getWidgetOrder('archive_review') }" 
       :draggable="customizing" 
       @dragstart="onDragStart($event, 'archive_review')" 
       @dragover.prevent 
       @dragenter.prevent 
       @drop="onDrop($event, 'archive_review')">
        <div class="widget-title">
          <div class="widget-title-left"><SafetyCertificateOutlined /> {{ t('overview.widgetTitle.archive_review') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('archive_review')" :title="t('overview.resizeWidget')" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="activity-list" style="margin-top: 10px;">
          <div v-for="a in dash.archive_recent.slice(0, 4)" :key="a.id" class="activity-item">
            <div class="activity-dot" :style="{ background: a.compliance === 'compliant' ? 'var(--color-success)' : a.compliance === 'non_compliant' ? 'var(--color-danger)' : 'var(--color-warning)' }" />
            <div class="activity-body">
              <span class="activity-action">{{ archiveComplianceLabel(a.compliance) }}</span>
              <span class="activity-target">{{ a.title }}</span>
            </div>
            <div class="activity-meta">
              <span class="activity-user">{{ a.user_name }}</span>
              <span class="activity-time">{{ formatActivityTime(a.created_at) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!--=====最近活动（全部）=====-->
      <div v-if="isEnabled('recent_activity')"
       :class="['widget', `widget--${getWidgetSize('recent_activity')}`, { 'widget--editing': customizing }]" 
       :style="{ order: getWidgetOrder('recent_activity') }" 
       :draggable="customizing" 
       @dragstart="onDragStart($event, 'recent_activity')" 
       @dragover.prevent 
       @dragenter.prevent 
       @drop="onDrop($event, 'recent_activity')">
        <div class="widget-title">
          <div class="widget-title-left"><ClockCircleOutlined /> {{ t('overview.recentActivity') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('recent_activity')" :title="t('overview.resizeWidget')" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="activity-list">
          <div v-for="a in dash.recent_activity" :key="a.id" class="activity-item">
            <div class="activity-dot" :style="{ background: activityKindStyle[a.kind]?.color }" />
            <div class="activity-body">
              <span class="activity-action">{{ activityActionLabel(a) }}</span>
              <span class="activity-target">{{ a.title }}</span>
            </div>
            <div class="activity-meta">
              <span class="activity-user">{{ a.user_name }}</span>
              <span class="activity-time">{{ formatActivityTime(a.created_at) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!--===== AI性能（业务+租户） =====-->
      <div v-if="isEnabled('ai_performance')"
       :class="['widget', `widget--${getWidgetSize('ai_performance')}`, { 'widget--editing': customizing }]" 
       :style="{ order: getWidgetOrder('ai_performance') }" 
       :draggable="customizing" 
       @dragstart="onDragStart($event, 'ai_performance')" 
       @dragover.prevent 
       @dragenter.prevent 
       @drop="onDrop($event, 'ai_performance')">
        <div class="widget-title">
          <div class="widget-title-left"><ThunderboltOutlined /> {{ t('overview.aiPerformance') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('ai_performance')" :title="t('overview.resizeWidget')" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="ai-stats">
          <div class="ai-stat">
            <div class="ai-stat-num">{{ dash.ai_performance?.avg_response_ms ?? 0 }}ms</div>
            <div class="ai-stat-label">{{ t('overview.avgResponse') }}</div>
          </div>
          <div class="ai-stat">
            <div class="ai-stat-num">{{ (dash.ai_performance?.success_rate ?? 0).toFixed(1) }}%</div>
            <div class="ai-stat-label">{{ t('overview.successRate') }}</div>
          </div>
          <div class="ai-stat">
            <div class="ai-stat-num">{{ formatNum(dash.ai_performance?.total_calls ?? 0) }}</div>
            <div class="ai-stat-label">{{ t('overview.totalCalls') }}</div>
          </div>
        </div>
        <div class="bar-chart bar-chart--small">
          <div v-for="s in (dash.ai_performance?.daily_stats ?? [])" :key="s.date" class="bar-col">
            <div class="bar-value">{{ s.avg_ms }}</div>
            <div class="bar bar--accent" :style="{ height: (s.avg_ms / aiBarMaxMs * 80) + 'px' }" />
            <div class="bar-label">{{ s.date }}</div>
          </div>
        </div>
      </div>

      <!--===== 租户使用情况 (tenant_admin) / 全平台 Token 合计 (system_admin) =====-->
      <div v-if="isEnabled('tenant_usage')"
       :class="['widget', `widget--${getWidgetSize('tenant_usage')}`, { 'widget--editing': customizing }]" 
       :style="{ order: getWidgetOrder('tenant_usage') }" 
       :draggable="customizing" 
       @dragstart="onDragStart($event, 'tenant_usage')" 
       @dragover.prevent 
       @dragenter.prevent 
       @drop="onDrop($event, 'tenant_usage')">
        <div class="widget-title">
          <div class="widget-title-left"><CloudServerOutlined /> {{ t('overview.tenantUsage') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('tenant_usage')" :title="t('overview.resizeWidget')" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="usage-rows">
          <div class="usage-row">
            <span class="usage-label">{{ isPlatformAdmin ? t('overview.platformTokenAllTenants') : t('overview.tokenUsage') }}</span>
            <div class="usage-bar-wrap">
              <div class="usage-bar" :style="{ width: ((dash.tenant_usage?.token_quota ?? 0) > 0 ? (dash.tenant_usage!.token_used / dash.tenant_usage!.token_quota * 100) : 0) + '%' }" />
            </div>
            <span class="usage-text">{{ formatNum(dash.tenant_usage?.token_used ?? 0) }} / {{ formatNum(dash.tenant_usage?.token_quota ?? 0) }}</span>
          </div>
          <template v-if="!isPlatformAdmin">
            <div v-if="(dash.tenant_usage?.storage_quota_mb ?? 0) > 0" class="usage-row">
              <span class="usage-label">{{ t('overview.storageUsage') }}</span>
              <div class="usage-bar-wrap">
                <div class="usage-bar usage-bar--info" :style="{ width: storagePct + '%' }" />
              </div>
              <span class="usage-text">{{ dash.tenant_usage?.storage_used_mb ?? 0 }}MB / {{ dash.tenant_usage?.storage_quota_mb ?? 0 }}MB</span>
            </div>
            <div v-else class="usage-row">
              <span class="usage-label">{{ t('overview.storageUsage') }}</span>
              <span class="usage-text" style="flex:1;text-align:right;color:var(--color-text-tertiary);">{{ t('overview.storageNotTracked') }}</span>
            </div>
            <div class="usage-row">
              <span class="usage-label">{{ t('overview.activeUsers') }}</span>
              <div class="usage-bar-wrap">
                <div class="usage-bar usage-bar--success" :style="{ width: ((dash.tenant_usage?.total_users ?? 0) > 0 ? (dash.tenant_usage!.active_users / dash.tenant_usage!.total_users * 100) : 0) + '%' }" />
              </div>
              <span class="usage-text">{{ dash.tenant_usage?.active_users ?? 0 }} / {{ dash.tenant_usage?.total_users ?? 0 }}</span>
            </div>
          </template>
        </div>
      </div>

      <!--===== 用户活动 (tenant_admin) =====-->
      <div v-if="isEnabled('user_activity')"
       :class="['widget', `widget--${getWidgetSize('user_activity')}`, { 'widget--editing': customizing }]" 
       :style="{ order: getWidgetOrder('user_activity') }" 
       :draggable="customizing" 
       @dragstart="onDragStart($event, 'user_activity')" 
       @dragover.prevent 
       @dragenter.prevent 
       @drop="onDrop($event, 'user_activity')">
        <div class="widget-title">
          <div class="widget-title-left"><TeamOutlined /> {{ t('overview.userActivityRank') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('user_activity')" :title="t('overview.resizeWidget')" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="rank-list">
          <div v-for="(u, i) in (dash.user_activity ?? [])" :key="u.username" class="rank-item">
            <span class="rank-num" :class="{ 'rank-num--top': i < 3 }">{{ i + 1 }}</span>
            <div class="rank-info">
              <span class="rank-name">{{ u.display_name }}</span>
              <span class="rank-dept">{{ u.department }}</span>
            </div>
            <span class="rank-count">{{ u.audit_count }} {{ t('overview.times') }}</span>
          </div>
        </div>
        <div v-if="!(dash.user_activity?.length)" style="padding: 16px; color: var(--color-text-tertiary); font-size: 13px;">{{ t('overview.emptyUserActivity') }}</div>
      </div>

      <!--===== 平台：租户审核量排行（system_admin）=====-->
      <div v-if="isEnabled('platform_tenant_ranking')"
       :class="['widget', `widget--${getWidgetSize('platform_tenant_ranking')}`, { 'widget--editing': customizing }]"
       :style="{ order: getWidgetOrder('platform_tenant_ranking') }"
       :draggable="customizing"
       @dragstart="onDragStart($event, 'platform_tenant_ranking')"
       @dragover.prevent
       @dragenter.prevent
       @drop="onDrop($event, 'platform_tenant_ranking')">
        <div class="widget-title">
          <div class="widget-title-left"><TeamOutlined /> {{ t('overview.platformTenantRank') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('platform_tenant_ranking')" :title="t('overview.resizeWidget')" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="rank-list">
          <div v-for="(row, i) in (dash.tenant_ranking ?? [])" :key="row.tenant_id" class="rank-item">
            <span class="rank-num" :class="{ 'rank-num--top': i < 3 }">{{ i + 1 }}</span>
            <div class="rank-info">
              <span class="rank-name">{{ row.tenant_name }}</span>
              <span class="rank-dept">{{ row.tenant_code }}</span>
            </div>
            <span class="rank-count">{{ row.audit_count }} {{ t('overview.times') }}</span>
          </div>
        </div>
        <div v-if="!(dash.tenant_ranking?.length)" style="padding: 16px; color: var(--color-text-tertiary); font-size: 13px;">{{ t('overview.emptyUserActivity') }}</div>
      </div>

    </div>
    </a-spin>
  </div>
</template>

<style scoped>
.overview-page { max-width: 1400px; }

.ov-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 20px; gap: 16px; flex-wrap: wrap; }
.ov-title { font-size: 24px; font-weight: 700; color: var(--color-text-primary); margin: 0; }
.ov-subtitle { font-size: 14px; color: var(--color-text-tertiary); margin-top: 4px; }

/*自定义面板*/
.customize-panel {
  background: var(--color-bg-card); border: 1px solid var(--color-border);
  border-radius: var(--radius-lg); padding: 16px 20px; margin-bottom: 20px;
}
.customize-hint { font-size: 13px; color: var(--color-text-secondary); margin-bottom: 12px; }
.customize-grid { display: flex; flex-wrap: wrap; gap: 8px; }
.customize-chip {
  display: flex; align-items: center; gap: 6px;
  padding: 6px 14px; border-radius: var(--radius-full);
  border: 1px solid var(--color-border); background: var(--color-bg-page);
  font-size: 13px; color: var(--color-text-secondary); cursor: pointer;
  transition: all var(--transition-fast);
}
.customize-chip:hover { border-color: var(--color-primary); color: var(--color-primary); }
.customize-chip--active { background: var(--color-primary-bg); border-color: var(--color-primary); color: var(--color-primary); font-weight: 500; }

.slide-down-enter-active { transition: all 0.25s ease; }
.slide-down-leave-active { transition: all 0.2s ease; }
.slide-down-enter-from, .slide-down-leave-to { opacity: 0; transform: translateY(-8px); }

/*小部件网格*/
.widget-title { display: flex; justify-content: space-between; align-items: center; width: 100%; }
.widget-title-left { flex: 1; display: flex; align-items: center; gap: 8px; }
.widget-actions { flex-shrink: 0; padding: 4px; border-radius: 4px; transition: background 0.2s; }
.widget-actions:hover { background: rgba(0,0,0,0.05); }
.widget-grid { display: grid; grid-template-columns: repeat(12, 1fr); gap: 16px; }
.widget {
  background: var(--color-bg-card); border: 1px solid var(--color-border-light);
  border-radius: var(--radius-lg); padding: 20px;
  box-shadow: var(--shadow-xs); transition: box-shadow var(--transition-base);
}
.widget:hover { box-shadow: var(--shadow-sm); }

.widget--editing {
  border: 1px dashed var(--color-primary);
  cursor: grab;
  transform: scale(0.99);
}
.widget--editing:active {
  cursor: grabbing;
}

.widget--sm { grid-column: span 4; }
.widget--md { grid-column: span 6; }
.widget--lg { grid-column: span 12; }
.widget-title {
  font-size: 14px; font-weight: 600; color: var(--color-text-primary);
  margin-bottom: 16px; display: flex; align-items: center; gap: 8px;
}

/*审计总结*/
.summary-cards { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; }
.summary-card {
  text-align: center; padding: 16px 8px; border-radius: var(--radius-md);
  background: var(--color-bg-page);
}
.summary-card--total { background: var(--color-primary-bg); }
.summary-card--approved .summary-icon { color: var(--color-success); font-size: 20px; }
.summary-card--returned .summary-icon { color: var(--color-danger); font-size: 20px; }
.summary-card--archived .summary-icon { color: var(--color-warning); font-size: 20px; }
.summary-num { font-size: 28px; font-weight: 700; color: var(--color-text-primary); line-height: 1.2; }
.summary-label { font-size: 12px; color: var(--color-text-tertiary); margin-top: 4px; }
.summary-card--total .summary-num { color: var(--color-primary); }

/*待办的*/
.pending-big { text-align: center; padding: 20px 0 12px; }
.pending-num { font-size: 48px; font-weight: 700; color: var(--color-primary); line-height: 1; }
.pending-label { font-size: 14px; color: var(--color-text-tertiary); margin-top: 8px; }

/*条形图*/
.bar-chart { display: flex; align-items: flex-end; gap: 8px; justify-content: space-between; padding-top: 8px; }
.bar-chart--small { margin-top: 16px; }
.bar-col { display: flex; flex-direction: column; align-items: center; flex: 1; }
.bar-value { font-size: 11px; color: var(--color-text-tertiary); margin-bottom: 4px; }
.bar {
  width: 100%; max-width: 40px; min-height: 4px;
  background: var(--color-primary); border-radius: 4px 4px 0 0;
  transition: height 0.5s ease;
}
.bar--accent { background: var(--color-accent); }
.bar-label { font-size: 11px; color: var(--color-text-tertiary); margin-top: 6px; }

/*部门分布*/
.dept-list { display: flex; flex-direction: column; gap: 10px; }
.dept-row { display: flex; align-items: center; gap: 10px; }
.dept-name { font-size: 13px; color: var(--color-text-secondary); width: 72px; flex-shrink: 0; text-align: right; }
.dept-bar-wrap { flex: 1; height: 20px; background: var(--color-bg-page); border-radius: 4px; overflow: hidden; }
.dept-bar { height: 100%; border-radius: 4px; transition: width 0.5s ease; }
.dept-count { font-size: 13px; font-weight: 600; color: var(--color-text-primary); width: 28px; text-align: right; }

/*活动*/
.activity-list { display: flex; flex-direction: column; gap: 0; }
.activity-item {
  display: flex; align-items: center; gap: 10px;
  padding: 8px 0; border-bottom: 1px solid var(--color-border-light);
}
.activity-item:last-child { border-bottom: none; }
.activity-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.activity-body { flex: 1; min-width: 0; }
.activity-action { font-size: 13px; font-weight: 500; color: var(--color-text-primary); }
.activity-target { font-size: 13px; color: var(--color-text-secondary); margin-left: 6px; }
.activity-meta { display: flex; flex-direction: column; align-items: flex-end; flex-shrink: 0; }
.activity-user { font-size: 12px; color: var(--color-text-tertiary); }
.activity-time { font-size: 11px; color: var(--color-text-tertiary); }

/*人工智能统计*/
.ai-stats { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; }
.ai-stat { text-align: center; padding: 12px 0; background: var(--color-bg-page); border-radius: var(--radius-md); }
.ai-stat-num { font-size: 20px; font-weight: 700; color: var(--color-text-primary); }
.ai-stat-label { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }

/*使用栏*/
.usage-rows { display: flex; flex-direction: column; gap: 16px; }
.usage-row { display: flex; align-items: center; gap: 12px; }
.usage-label { font-size: 13px; color: var(--color-text-secondary); width: 72px; flex-shrink: 0; }
.usage-bar-wrap { flex: 1; height: 12px; background: var(--color-bg-page); border-radius: 6px; overflow: hidden; }
.usage-bar { height: 100%; background: var(--color-primary); border-radius: 6px; transition: width 0.5s ease; }
.usage-bar--info { background: var(--color-info); }
.usage-bar--success { background: var(--color-success); }
.usage-text { font-size: 12px; color: var(--color-text-tertiary); width: 120px; text-align: right; flex-shrink: 0; }

/*覆盖范围*/
.coverage-list { display: flex; flex-direction: column; gap: 14px; }
.coverage-row { display: flex; align-items: center; gap: 10px; }
.coverage-info { width: 100px; flex-shrink: 0; }
.coverage-type { font-size: 13px; font-weight: 500; color: var(--color-text-primary); display: block; }
.coverage-count { font-size: 11px; color: var(--color-text-tertiary); }
.coverage-bar-wrap { flex: 1; height: 10px; background: var(--color-bg-page); border-radius: 5px; overflow: hidden; }
.coverage-bar { height: 100%; background: var(--color-success); border-radius: 5px; transition: width 0.5s ease; }
.coverage-pct { font-size: 13px; font-weight: 600; color: var(--color-text-primary); width: 40px; text-align: right; }

/*秩*/
.rank-list { display: flex; flex-direction: column; gap: 0; }
.rank-item { display: flex; align-items: center; gap: 12px; padding: 8px 0; border-bottom: 1px solid var(--color-border-light); }
.rank-item:last-child { border-bottom: none; }
.rank-num {
  width: 24px; height: 24px; border-radius: 50%; display: flex; align-items: center; justify-content: center;
  font-size: 12px; font-weight: 600; background: var(--color-bg-page); color: var(--color-text-tertiary); flex-shrink: 0;
}
.rank-num--top { background: var(--color-primary-bg); color: var(--color-primary); }
.rank-info { flex: 1; min-width: 0; }
.rank-name { font-size: 13px; font-weight: 500; color: var(--color-text-primary); }
.rank-dept { font-size: 12px; color: var(--color-text-tertiary); margin-left: 8px; }
.rank-count { font-size: 13px; font-weight: 600; color: var(--color-primary); flex-shrink: 0; }

/*健康网格*/
.health-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); gap: 12px; }
.health-card { background: var(--color-bg-page); border-radius: var(--radius-md); padding: 14px; }
.health-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.health-name { font-size: 13px; font-weight: 600; color: var(--color-text-primary); }
.health-badge { font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: var(--radius-full); }
.health-metrics { display: flex; flex-direction: column; gap: 8px; }
.health-metric { display: flex; align-items: center; gap: 8px; }
.health-metric-label { font-size: 12px; color: var(--color-text-tertiary); width: 32px; }
.health-bar-wrap { flex: 1; height: 6px; background: var(--color-bg-card); border-radius: 3px; overflow: hidden; }
.health-bar { height: 100%; border-radius: 3px; transition: width 0.5s ease; }
.health-metric-val { font-size: 12px; color: var(--color-text-secondary); width: 36px; text-align: right; }
.health-uptime { font-size: 11px; color: var(--color-text-tertiary); margin-top: 8px; }

/*租户表*/
.tenant-table, .api-table { display: flex; flex-direction: column; }
.tenant-row, .api-row {
  display: grid; grid-template-columns: 2fr 1fr 1fr 1fr; gap: 8px;
  padding: 8px 0; border-bottom: 1px solid var(--color-border-light);
  font-size: 13px; color: var(--color-text-secondary); align-items: center;
}
.tenant-row:last-child, .api-row:last-child { border-bottom: none; }
.tenant-row--header, .api-row--header { font-weight: 600; color: var(--color-text-tertiary); font-size: 12px; }
.tenant-name { font-weight: 500; color: var(--color-text-primary); }
.tenant-status { font-weight: 500; }
.api-endpoint { font-family: var(--font-mono); font-size: 12px; color: var(--color-text-primary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

/*监控指标网格*/
.monitor-metrics-grid {
  display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px;
}
.monitor-metric-card {
  display: flex; align-items: center; gap: 12px;
  padding: 14px 16px;
  background: var(--color-bg-page);
  border-radius: var(--radius-md);
  transition: all 0.2s ease;
}
.monitor-metric-card:hover { transform: translateY(-1px); box-shadow: var(--shadow-xs); }
.monitor-metric-icon {
  width: 40px; height: 40px; border-radius: var(--radius-md);
  display: flex; align-items: center; justify-content: center;
  font-size: 18px; flex-shrink: 0;
}
.monitor-metric-icon--primary { background: var(--color-primary-bg); color: var(--color-primary); }
.monitor-metric-icon--success { background: var(--color-success-bg); color: var(--color-success); }
.monitor-metric-icon--warning { background: var(--color-warning-bg); color: var(--color-warning); }
.monitor-metric-icon--info { background: var(--color-info-bg); color: var(--color-info); }
.monitor-metric-value { font-size: 20px; font-weight: 700; color: var(--color-text-primary); line-height: 1.2; }
.monitor-metric-unit { font-size: 12px; font-weight: 500; color: var(--color-text-tertiary); margin-left: 2px; }
.monitor-metric-label { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }

/*监控警报*/
.monitor-alerts-list { display: flex; flex-direction: column; gap: 10px; }
.monitor-alert-item {
  display: flex; align-items: flex-start; gap: 12px;
  padding: 12px 14px; border-radius: var(--radius-md);
  background: var(--color-bg-page); border-left: 3px solid;
  transition: background 0.2s ease;
}
.monitor-alert-item:hover { background: var(--color-bg-hover); }
.monitor-alert-dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; margin-top: 5px; }
.monitor-alert-message { font-size: 13px; color: var(--color-text-primary); line-height: 1.4; }
.monitor-alert-time { font-size: 11px; color: var(--color-text-tertiary); margin-top: 4px; }

@media (max-width: 1024px) {
  .widget--sm, .widget--md { grid-column: span 6; }
  .summary-cards { grid-template-columns: repeat(2, 1fr); }
  .monitor-metrics-grid { grid-template-columns: repeat(2, 1fr); }
}
@media (max-width: 768px) {
  .widget--sm, .widget--md, .widget--lg { grid-column: span 12; }
  .summary-cards { grid-template-columns: repeat(2, 1fr); }
  .health-grid { grid-template-columns: 1fr; }
  .monitor-metrics-grid { grid-template-columns: 1fr; }
  .ov-header { flex-direction: column; }
}
</style>
