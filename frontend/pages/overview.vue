<script setup lang="ts">
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  EditOutlined,
  ClockCircleOutlined,
  ThunderboltOutlined,
  RiseOutlined,
  TeamOutlined,
  SafetyCertificateOutlined,
  CloudServerOutlined,
  ApiOutlined,
  SettingOutlined,
  EyeOutlined,
  EyeInvisibleOutlined,
  AlertOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { OVERVIEW_WIDGETS } from '~/composables/useMockData'
import { useI18n } from '~/composables/useI18n'
import type { OverviewWidgetId } from '~/composables/useMockData'

definePageMeta({ middleware: 'auth' })

const { userPermissions, currentUser } = useAuth()
const { mockOverviewData, mockUserDashboardPrefs, mockCronTasks, mockArchiveLogs } = useMockData()
const { t, locale } = useI18n()
const data = ref(mockOverviewData)

// Alert helpers for monitor_alerts widget
const alertLevelConfig: Record<string, { color: string; bg: string }> = {
  warning: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)' },
  error: { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)' },
  info: { color: 'var(--color-info)', bg: 'var(--color-info-bg)' },
}
const getAlertMessage = (alert: typeof data.value.monitorAlerts[0]) => locale.value === 'en-US' ? alert.messageEn : alert.messageZh
const getAlertTime = (alert: typeof data.value.monitorAlerts[0]) => locale.value === 'en-US' ? alert.timeEn : alert.timeZh

const username = computed(() => currentUser.value?.username || '')
const defaultPrefs = computed(() => {
  const perms = userPermissions.value
  return OVERVIEW_WIDGETS
    .filter(w => w.requiredPermissions.some(p => perms.includes(p)) && w.defaultEnabled)
    .map(w => w.id)
})
const enabledWidgets = ref<OverviewWidgetId[]>([])
const widgetSizes = ref<Partial<Record<OverviewWidgetId, 'sm' | 'md' | 'lg'>>>({})

const availableWidgets = computed(() => {
  const perms = userPermissions.value
  return OVERVIEW_WIDGETS.filter(w => w.requiredPermissions.some(p => perms.includes(p)))
})

watch(userPermissions, () => {
  const prefsKey = `${username.value}_${userPermissions.value[0] || 'business'}`
  const saved = mockUserDashboardPrefs[prefsKey]
  
  if (saved) {
    enabledWidgets.value = [...saved.enabledWidgets]
    widgetSizes.value = { ...(saved.widgetSizes || {}) }
  } else {
    const generalSaved = mockUserDashboardPrefs[username.value]
    if (generalSaved) {
       const validGeneral = generalSaved.enabledWidgets.filter(id => availableWidgets.value.some(w => w.id === id))
       const merged = [...new Set([...validGeneral, ...defaultPrefs.value])]
       enabledWidgets.value = merged
       widgetSizes.value = { ...(generalSaved.widgetSizes || {}) }
    } else {
       enabledWidgets.value = [...defaultPrefs.value]
       widgetSizes.value = {}
    }
  }
}, { immediate: true })

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
const savePrefs = () => { 
  customizing.value = false
  const prefsKey = `${username.value}_${userPermissions.value[0] || 'business'}`
  mockUserDashboardPrefs[prefsKey] = { 
    enabledWidgets: [...enabledWidgets.value],
    widgetSizes: { ...widgetSizes.value }
  }
  message.success(t('overview.layoutSaved')) 
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

const activityStyle: Record<string, { color: string; bg: string }> = {
  audit: { color: 'var(--color-primary)', bg: 'var(--color-primary-bg)' },
  cron: { color: 'var(--color-accent)', bg: 'rgba(6,182,212,0.1)' },
  system: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)' },
  config: { color: 'var(--color-success)', bg: 'var(--color-success-bg)' },
}

const healthColor = (s: string) => s === 'healthy' ? 'var(--color-success)' : s === 'degraded' ? 'var(--color-warning)' : 'var(--color-danger)'
const healthLabel = (s: string) => s === 'healthy' ? t('overview.health.healthy') : s === 'degraded' ? t('overview.health.degraded') : t('overview.health.error')
const trendMax = computed(() => Math.max(...data.value.weeklyTrend.map(t => t.count), 1))
const deptMax = computed(() => Math.max(...data.value.deptDistribution.map(d => d.count), 1))
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
    <div class="ov-header">
      <div>
        <h1 class="ov-title">{{ greeting }}，{{ currentUser?.display_name || t('sidebar.defaultUser') }}</h1>
        <p class="ov-subtitle">{{ t('overview.subtitle') }}</p>
      </div>
      <a-button :type="customizing ? 'primary' : 'default'" @click="customizing ? savePrefs() : (customizing = true)">
        <SettingOutlined /> {{ customizing ? t('overview.saveLayout') : t('overview.customizeDashboard') }}
      </a-button>
    </div>

    <!-- Customize panel -->
    <transition name="slide-down">
      <div v-if="customizing" class="customize-panel">
        <p class="customize-hint">{{ t('overview.customizeHint') }}</p>
        <div class="customize-grid">
          <div v-for="w in availableWidgets" :key="w.id" class="customize-chip" :class="{ 'customize-chip--active': isEnabled(w.id) }" @click="toggleWidget(w.id)">
            <component :is="isEnabled(w.id) ? EyeOutlined : EyeInvisibleOutlined" />
            <span>{{ w.title }}</span>
          </div>
        </div>
      </div>
    </transition>

    <div class="widget-grid">
      <!-- ===== Audit Summary (business) ===== -->
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
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('audit_summary')" title="点击调整组件大小" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="summary-cards">
          <div class="summary-card summary-card--total">
            <div class="summary-num">{{ data.auditSummary.total }}</div>
            <div class="summary-label">{{ t('overview.totalAudits') }}</div>
          </div>
          <div class="summary-card summary-card--approved">
            <CheckCircleOutlined class="summary-icon" />
            <div class="summary-num">{{ data.auditSummary.approved }}</div>
            <div class="summary-label">{{ t('overview.approved') }}</div>
          </div>
          <div class="summary-card summary-card--rejected">
            <CloseCircleOutlined class="summary-icon" />
            <div class="summary-num">{{ data.auditSummary.rejected }}</div>
            <div class="summary-label">{{ t('overview.rejected') }}</div>
          </div>
          <div class="summary-card summary-card--archived">
            <CheckCircleOutlined class="summary-icon" />
            <div class="summary-num">{{ data.auditSummary.archived }}</div>
            <div class="summary-label">{{ t('dashboard.tab.archived') }}</div>
          </div>
        </div>
      </div>

      <!-- ===== Pending Tasks (business) ===== -->
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
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('pending_tasks')" title="点击调整组件大小" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="pending-big">
          <div class="pending-num">{{ data.pendingCount }}</div>
          <div class="pending-label">{{ t('overview.itemsPending') }}</div>
        </div>
        <a-button type="link" size="small" @click="navigateTo('/dashboard')">{{ t('overview.goToWorkbench') }} →</a-button>
      </div>

      <!-- ===== Weekly Trend (business) ===== -->
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
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('weekly_trend')" title="点击调整组件大小" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="bar-chart">
          <div v-for="t in data.weeklyTrend" :key="t.date" class="bar-col">
            <div class="bar-value">{{ t.count }}</div>
            <div class="bar" :style="{ height: (t.count / trendMax * 120) + 'px' }" />
            <div class="bar-label">{{ t.date }}</div>
          </div>
        </div>
      </div>

      <!-- ===== Dept Distribution (business) ===== -->
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
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('dept_distribution')" title="点击调整组件大小" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="dept-list">
          <div v-for="d in data.deptDistribution" :key="d.department" class="dept-row">
            <span class="dept-name">{{ d.department }}</span>
            <div class="dept-bar-wrap">
              <div class="dept-bar" :style="{ width: (d.count / deptMax * 100) + '%', background: d.color }" />
            </div>
            <span class="dept-count">{{ d.count }}</span>
          </div>
        </div>
      </div>

      <!-- ===== Cron Tasks (business) ===== -->
      <div v-if="isEnabled('cron_tasks')"
       :class="['widget', `widget--${getWidgetSize('cron_tasks')}`, { 'widget--editing': customizing }]" 
       :style="{ order: getWidgetOrder('cron_tasks') }" 
       :draggable="customizing" 
       @dragstart="onDragStart($event, 'cron_tasks')" 
       @dragover.prevent 
       @dragenter.prevent 
       @drop="onDrop($event, 'cron_tasks')">
        <div class="widget-title">
          <div class="widget-title-left"><ClockCircleOutlined /> 定时任务任务列表</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('cron_tasks')" title="点击调整组件大小" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="rank-list" style="margin-top: 10px;">
          <div v-for="c in mockCronTasks.slice(0, 5)" :key="c.id" class="rank-item">
            <div class="rank-info">
              <span class="rank-name">{{ c.task_type === 'batch_audit' ? '批量审核' : c.task_type === 'daily_report' ? '日报推送' : '周报推送' }}</span>
              <span class="rank-dept">{{ c.cron_expression }}</span>
            </div>
            <span class="rank-count" :style="{ color: c.is_active ? 'var(--color-success)' : 'var(--color-text-tertiary)' }">{{ c.is_active ? '运行中' : '已停用' }}</span>
          </div>
        </div>
      </div>

      <!-- ===== Archive Review (business) ===== -->
      <div v-if="isEnabled('archive_review')"
       :class="['widget', `widget--${getWidgetSize('archive_review')}`, { 'widget--editing': customizing }]" 
       :style="{ order: getWidgetOrder('archive_review') }" 
       :draggable="customizing" 
       @dragstart="onDragStart($event, 'archive_review')" 
       @dragover.prevent 
       @dragenter.prevent 
       @drop="onDrop($event, 'archive_review')">
        <div class="widget-title">
          <div class="widget-title-left"><SafetyCertificateOutlined /> 归档复盘最新记录</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('archive_review')" title="点击调整组件大小" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="activity-list" style="margin-top: 10px;">
          <div v-for="a in mockArchiveLogs.slice(0, 4)" :key="a.id" class="activity-item">
            <div class="activity-dot" :style="{ background: 'var(--color-primary)' }" />
            <div class="activity-body">
              <span class="activity-action">{{ a.action_label }}</span>
              <span class="activity-target">{{ a.title }}</span>
            </div>
            <div class="activity-meta">
              <span class="activity-user">{{ a.operator }}</span>
              <span class="activity-time">{{ a.created_at }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- ===== Recent Activity (all) ===== -->
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
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('recent_activity')" title="点击调整组件大小" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="activity-list">
          <div v-for="a in data.recentActivity" :key="a.id" class="activity-item">
            <div class="activity-dot" :style="{ background: activityStyle[a.type]?.color }" />
            <div class="activity-body">
              <span class="activity-action">{{ a.action }}</span>
              <span class="activity-target">{{ a.target }}</span>
            </div>
            <div class="activity-meta">
              <span class="activity-user">{{ a.user }}</span>
              <span class="activity-time">{{ a.time }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- ===== AI Performance (business+tenant) ===== -->
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
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('ai_performance')" title="点击调整组件大小" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="ai-stats">
          <div class="ai-stat">
            <div class="ai-stat-num">{{ data.aiPerformance.avgResponseMs }}ms</div>
            <div class="ai-stat-label">{{ t('overview.avgResponse') }}</div>
          </div>
          <div class="ai-stat">
            <div class="ai-stat-num">{{ data.aiPerformance.successRate }}%</div>
            <div class="ai-stat-label">{{ t('overview.successRate') }}</div>
          </div>
          <div class="ai-stat">
            <div class="ai-stat-num">{{ formatNum(data.aiPerformance.totalCalls) }}</div>
            <div class="ai-stat-label">{{ t('overview.totalCalls') }}</div>
          </div>
        </div>
        <div class="bar-chart bar-chart--small">
          <div v-for="s in data.aiPerformance.dailyStats" :key="s.date" class="bar-col">
            <div class="bar-value">{{ s.avgMs }}</div>
            <div class="bar bar--accent" :style="{ height: (s.avgMs / 2500 * 80) + 'px' }" />
            <div class="bar-label">{{ s.date }}</div>
          </div>
        </div>
      </div>

      <!-- ===== Tenant Usage (tenant_admin) ===== -->
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
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('tenant_usage')" title="点击调整组件大小" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="usage-rows">
          <div class="usage-row">
            <span class="usage-label">{{ t('overview.tokenUsage') }}</span>
            <div class="usage-bar-wrap">
              <div class="usage-bar" :style="{ width: (data.tenantUsage.tokenUsed / data.tenantUsage.tokenQuota * 100) + '%' }" />
            </div>
            <span class="usage-text">{{ formatNum(data.tenantUsage.tokenUsed) }} / {{ formatNum(data.tenantUsage.tokenQuota) }}</span>
          </div>
          <div class="usage-row">
            <span class="usage-label">{{ t('overview.storageUsage') }}</span>
            <div class="usage-bar-wrap">
              <div class="usage-bar usage-bar--info" :style="{ width: (data.tenantUsage.storageUsedMB / data.tenantUsage.storageQuotaMB * 100) + '%' }" />
            </div>
            <span class="usage-text">{{ data.tenantUsage.storageUsedMB }}MB / {{ data.tenantUsage.storageQuotaMB }}MB</span>
          </div>
          <div class="usage-row">
            <span class="usage-label">{{ t('overview.activeUsers') }}</span>
            <div class="usage-bar-wrap">
              <div class="usage-bar usage-bar--success" :style="{ width: (data.tenantUsage.activeUsers / data.tenantUsage.totalUsers * 100) + '%' }" />
            </div>
            <span class="usage-text">{{ data.tenantUsage.activeUsers }} / {{ data.tenantUsage.totalUsers }}</span>
          </div>
        </div>
      </div>

      <!-- ===== User Activity (tenant_admin) ===== -->
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
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('user_activity')" title="点击调整组件大小" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="rank-list">
          <div v-for="(u, i) in data.userActivity" :key="u.username" class="rank-item">
            <span class="rank-num" :class="{ 'rank-num--top': i < 3 }">{{ i + 1 }}</span>
            <div class="rank-info">
              <span class="rank-name">{{ u.displayName }}</span>
              <span class="rank-dept">{{ u.department }}</span>
            </div>
            <span class="rank-count">{{ u.auditCount }} {{ t('overview.times') }}</span>
          </div>
        </div>
      </div>

      <!-- ===== System Health (system_admin) ===== -->
      <div v-if="isEnabled('system_health')"
       :class="['widget', `widget--${getWidgetSize('system_health')}`, { 'widget--editing': customizing }]"
       :style="{ order: getWidgetOrder('system_health') }"
       :draggable="customizing"
       @dragstart="onDragStart($event, 'system_health')"
       @dragover.prevent
       @dragenter.prevent
       @drop="onDrop($event, 'system_health')">
        <div class="widget-title">
          <div class="widget-title-left"><CloudServerOutlined /> {{ t('overview.systemHealth') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('system_health')" title="点击调整组件大小" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="health-grid">
          <div v-for="s in data.systemHealth" :key="s.service" class="health-card">
            <div class="health-header">
              <span class="health-name">{{ s.service }}</span>
              <span class="health-badge" :style="{ color: healthColor(s.status), background: s.status === 'healthy' ? 'var(--color-success-bg)' : s.status === 'degraded' ? 'var(--color-warning-bg)' : 'var(--color-danger-bg)' }">
                {{ healthLabel(s.status) }}
              </span>
            </div>
            <div class="health-metrics">
              <div class="health-metric">
                <span class="health-metric-label">CPU</span>
                <div class="health-bar-wrap"><div class="health-bar" :style="{ width: s.cpu + '%', background: s.cpu > 80 ? 'var(--color-danger)' : s.cpu > 60 ? 'var(--color-warning)' : 'var(--color-success)' }" /></div>
                <span class="health-metric-val">{{ s.cpu }}%</span>
              </div>
              <div class="health-metric">
                <span class="health-metric-label">{{ t('overview.memory') }}</span>
                <div class="health-bar-wrap"><div class="health-bar" :style="{ width: s.memory + '%', background: s.memory > 80 ? 'var(--color-danger)' : s.memory > 60 ? 'var(--color-warning)' : 'var(--color-success)' }" /></div>
                <span class="health-metric-val">{{ s.memory }}%</span>
              </div>
            </div>
            <div class="health-uptime">{{ t('overview.uptime') }} {{ s.uptime }}</div>
          </div>
        </div>
      </div>

      <!-- ===== Tenant Overview (system_admin) ===== -->
      <div v-if="isEnabled('tenant_overview')"
       :class="['widget', `widget--${getWidgetSize('tenant_overview')}`, { 'widget--editing': customizing }]" 
       :style="{ order: getWidgetOrder('tenant_overview') }" 
       :draggable="customizing" 
       @dragstart="onDragStart($event, 'tenant_overview')" 
       @dragover.prevent 
       @dragenter.prevent 
       @drop="onDrop($event, 'tenant_overview')">
        <div class="widget-title">
          <div class="widget-title-left"><TeamOutlined /> {{ t('overview.tenantOverview') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('tenant_overview')" title="点击调整组件大小" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="tenant-table">
          <div class="tenant-row tenant-row--header">
            <span>{{ t('overview.tenant') }}</span><span>{{ t('overview.users') }}</span><span>{{ t('overview.auditVolume') }}</span><span>{{ t('common.status') }}</span>
          </div>
          <div v-for="tenant in data.tenantOverview" :key="tenant.tenantId" class="tenant-row">
            <span class="tenant-name">{{ tenant.tenantName }}</span>
            <span>{{ tenant.userCount }}</span>
            <span>{{ formatNum(tenant.auditCount) }}</span>
            <span class="tenant-status" :style="{ color: tenant.status === 'active' ? 'var(--color-success)' : 'var(--color-warning)' }">
              {{ tenant.status === 'active' ? t('overview.active') : t('overview.suspended') }}
            </span>
          </div>
        </div>
      </div>

      <!-- ===== API Metrics (system_admin) ===== -->
      <div v-if="isEnabled('api_metrics')"
       :class="['widget', `widget--${getWidgetSize('api_metrics')}`, { 'widget--editing': customizing }]" 
       :style="{ order: getWidgetOrder('api_metrics') }" 
       :draggable="customizing" 
       @dragstart="onDragStart($event, 'api_metrics')" 
       @dragover.prevent 
       @dragenter.prevent 
       @drop="onDrop($event, 'api_metrics')">
        <div class="widget-title">
          <div class="widget-title-left"><ApiOutlined /> {{ t('overview.apiMetrics') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('api_metrics')" title="点击调整组件大小" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="api-table">
          <div class="api-row api-row--header">
            <span>{{ t('overview.endpoint') }}</span><span>{{ t('overview.calls') }}</span><span>{{ t('overview.latency') }}</span><span>{{ t('overview.successRate') }}</span>
          </div>
          <div v-for="a in data.apiMetrics" :key="a.endpoint" class="api-row">
            <span class="api-endpoint">{{ a.endpoint }}</span>
            <span>{{ formatNum(a.calls) }}</span>
            <span>{{ a.avgMs }}ms</span>
            <span :style="{ color: a.successRate >= 99 ? 'var(--color-success)' : a.successRate >= 95 ? 'var(--color-warning)' : 'var(--color-danger)' }">{{ a.successRate }}%</span>
          </div>
        </div>
      </div>

      <!-- ===== Monitor Metrics (system_admin) ===== -->
      <div v-if="isEnabled('monitor_metrics')"
       :class="['widget', `widget--${getWidgetSize('monitor_metrics')}`, { 'widget--editing': customizing }]"
       :style="{ order: getWidgetOrder('monitor_metrics') }"
       :draggable="customizing"
       @dragstart="onDragStart($event, 'monitor_metrics')"
       @dragover.prevent
       @dragenter.prevent
       @drop="onDrop($event, 'monitor_metrics')">
        <div class="widget-title">
          <div class="widget-title-left"><ThunderboltOutlined /> {{ t('overview.monitorMetrics') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('monitor_metrics')" :title="t('overview.resizeWidget')" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="monitor-metrics-grid">
          <div class="monitor-metric-card">
            <div class="monitor-metric-icon monitor-metric-icon--success"><ApiOutlined /></div>
            <div class="monitor-metric-info">
              <div class="monitor-metric-value">{{ data.monitorMetrics.apiSuccessRate }}<span class="monitor-metric-unit">%</span></div>
              <div class="monitor-metric-label">{{ t('overview.monitor.apiSuccessRate') }}</div>
            </div>
          </div>
          <div class="monitor-metric-card">
            <div class="monitor-metric-icon monitor-metric-icon--primary"><ClockCircleOutlined /></div>
            <div class="monitor-metric-info">
              <div class="monitor-metric-value">{{ data.monitorMetrics.avgModelResponseMs }}<span class="monitor-metric-unit">ms</span></div>
              <div class="monitor-metric-label">{{ t('overview.monitor.avgModelResponse') }}</div>
            </div>
          </div>
          <div class="monitor-metric-card">
            <div class="monitor-metric-icon monitor-metric-icon--warning"><ThunderboltOutlined /></div>
            <div class="monitor-metric-info">
              <div class="monitor-metric-value">{{ data.monitorMetrics.p95Latency }}<span class="monitor-metric-unit">ms</span></div>
              <div class="monitor-metric-label">{{ t('overview.monitor.p95Latency') }}</div>
            </div>
          </div>
          <div class="monitor-metric-card">
            <div class="monitor-metric-icon monitor-metric-icon--info"><RiseOutlined /></div>
            <div class="monitor-metric-info">
              <div class="monitor-metric-value">{{ formatNum(data.monitorMetrics.totalRequests24h) }}</div>
              <div class="monitor-metric-label">{{ t('overview.monitor.requests24h') }}</div>
            </div>
          </div>
          <div class="monitor-metric-card">
            <div class="monitor-metric-icon monitor-metric-icon--success"><TeamOutlined /></div>
            <div class="monitor-metric-info">
              <div class="monitor-metric-value">{{ data.monitorMetrics.activeTenants }}</div>
              <div class="monitor-metric-label">{{ t('overview.monitor.activeTenants') }}</div>
            </div>
          </div>
          <div class="monitor-metric-card">
            <div class="monitor-metric-icon monitor-metric-icon--primary"><CheckCircleOutlined /></div>
            <div class="monitor-metric-info">
              <div class="monitor-metric-value">{{ data.monitorMetrics.uptime }}</div>
              <div class="monitor-metric-label">{{ t('overview.monitor.uptime') }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- ===== Monitor Alerts (system_admin) ===== -->
      <div v-if="isEnabled('monitor_alerts')"
       :class="['widget', `widget--${getWidgetSize('monitor_alerts')}`, { 'widget--editing': customizing }]"
       :style="{ order: getWidgetOrder('monitor_alerts') }"
       :draggable="customizing"
       @dragstart="onDragStart($event, 'monitor_alerts')"
       @dragover.prevent
       @dragenter.prevent
       @drop="onDrop($event, 'monitor_alerts')">
        <div class="widget-title">
          <div class="widget-title-left"><AlertOutlined style="color: var(--color-warning);" /> {{ t('overview.monitor.recentAlerts') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('monitor_alerts')" :title="t('overview.resizeWidget')" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="monitor-alerts-list">
          <div
            v-for="alert in data.monitorAlerts"
            :key="alert.id"
            class="monitor-alert-item"
            :style="{ borderLeftColor: alertLevelConfig[alert.level]?.color }"
          >
            <div class="monitor-alert-dot" :style="{ background: alertLevelConfig[alert.level]?.color }" />
            <div class="monitor-alert-content">
              <div class="monitor-alert-message">{{ getAlertMessage(alert) }}</div>
              <div class="monitor-alert-time">{{ getAlertTime(alert) }}</div>
            </div>
          </div>
        </div>
        <div v-if="data.monitorAlerts.length === 0" style="padding: 32px; text-align: center;">
          <a-empty :description="t('overview.monitor.noAlerts')" />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.overview-page { max-width: 1400px; }

.ov-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 20px; gap: 16px; flex-wrap: wrap; }
.ov-title { font-size: 24px; font-weight: 700; color: var(--color-text-primary); margin: 0; }
.ov-subtitle { font-size: 14px; color: var(--color-text-tertiary); margin-top: 4px; }

/* Customize panel */
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

/* Widget grid */
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

/* Audit summary */
.summary-cards { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; }
.summary-card {
  text-align: center; padding: 16px 8px; border-radius: var(--radius-md);
  background: var(--color-bg-page);
}
.summary-card--total { background: var(--color-primary-bg); }
.summary-card--approved .summary-icon { color: var(--color-success); font-size: 20px; }
.summary-card--rejected .summary-icon { color: var(--color-danger); font-size: 20px; }
.summary-card--archived .summary-icon { color: var(--color-warning); font-size: 20px; }
.summary-num { font-size: 28px; font-weight: 700; color: var(--color-text-primary); line-height: 1.2; }
.summary-label { font-size: 12px; color: var(--color-text-tertiary); margin-top: 4px; }
.summary-card--total .summary-num { color: var(--color-primary); }

/* Pending */
.pending-big { text-align: center; padding: 20px 0 12px; }
.pending-num { font-size: 48px; font-weight: 700; color: var(--color-primary); line-height: 1; }
.pending-label { font-size: 14px; color: var(--color-text-tertiary); margin-top: 8px; }

/* Bar chart */
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

/* Dept distribution */
.dept-list { display: flex; flex-direction: column; gap: 10px; }
.dept-row { display: flex; align-items: center; gap: 10px; }
.dept-name { font-size: 13px; color: var(--color-text-secondary); width: 72px; flex-shrink: 0; text-align: right; }
.dept-bar-wrap { flex: 1; height: 20px; background: var(--color-bg-page); border-radius: 4px; overflow: hidden; }
.dept-bar { height: 100%; border-radius: 4px; transition: width 0.5s ease; }
.dept-count { font-size: 13px; font-weight: 600; color: var(--color-text-primary); width: 28px; text-align: right; }

/* Activity */
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

/* AI stats */
.ai-stats { display: grid; grid-template-columns: repeat(3, 1fr); gap: 12px; }
.ai-stat { text-align: center; padding: 12px 0; background: var(--color-bg-page); border-radius: var(--radius-md); }
.ai-stat-num { font-size: 20px; font-weight: 700; color: var(--color-text-primary); }
.ai-stat-label { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }

/* Usage bars */
.usage-rows { display: flex; flex-direction: column; gap: 16px; }
.usage-row { display: flex; align-items: center; gap: 12px; }
.usage-label { font-size: 13px; color: var(--color-text-secondary); width: 72px; flex-shrink: 0; }
.usage-bar-wrap { flex: 1; height: 12px; background: var(--color-bg-page); border-radius: 6px; overflow: hidden; }
.usage-bar { height: 100%; background: var(--color-primary); border-radius: 6px; transition: width 0.5s ease; }
.usage-bar--info { background: var(--color-info); }
.usage-bar--success { background: var(--color-success); }
.usage-text { font-size: 12px; color: var(--color-text-tertiary); width: 120px; text-align: right; flex-shrink: 0; }

/* Coverage */
.coverage-list { display: flex; flex-direction: column; gap: 14px; }
.coverage-row { display: flex; align-items: center; gap: 10px; }
.coverage-info { width: 100px; flex-shrink: 0; }
.coverage-type { font-size: 13px; font-weight: 500; color: var(--color-text-primary); display: block; }
.coverage-count { font-size: 11px; color: var(--color-text-tertiary); }
.coverage-bar-wrap { flex: 1; height: 10px; background: var(--color-bg-page); border-radius: 5px; overflow: hidden; }
.coverage-bar { height: 100%; background: var(--color-success); border-radius: 5px; transition: width 0.5s ease; }
.coverage-pct { font-size: 13px; font-weight: 600; color: var(--color-text-primary); width: 40px; text-align: right; }

/* Rank */
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

/* Health grid */
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

/* Tenant table */
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

/* Monitor metrics grid */
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

/* Monitor alerts */
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
