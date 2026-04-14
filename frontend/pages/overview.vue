<script setup lang="ts">
import {
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
  DatabaseOutlined,
  BarChartOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { OVERVIEW_WIDGETS, OVERVIEW_WIDGET_ID_SET, WIDGET_PAGE_PERMISSION_MAP } from '~/constants/overviewWidgets'
import { useI18n } from '~/composables/useI18n'
import type { OverviewWidgetId } from '~/constants/overviewWidgets'
import type { PermissionGroup } from '~/types/auth'
import type {
  DashboardOverview,
  PlatformDashboardOverview,
  ActivityItemEnriched,
} from '~/types/dashboard-overview'
import type { DashboardPref } from '~/types/user-config'
import StackedBarChart from '~/components/charts/StackedBarChart.vue'
import DeptDistributionChart from '~/components/charts/DeptDistributionChart.vue'

definePageMeta({ middleware: 'auth' })

// 概览页空数据占位，避免模板渲染时出现 undefined 错误
const EMPTY_OVERVIEW: DashboardOverview = {
  weekly_overview: { total: 0, audit_count: 0, archive_count: 0, cron_count: 0 },
  weekly_trend: [],
  recent_activity: [],
}

// 鉴权、国际化、数据接口
const { effectiveActiveRoleForApi, currentUser, menus } = useAuth()
const { t, locale } = useI18n()
const { fetchDashboardOverview, fetchPlatformDashboardOverview } = useDashboardOverviewApi()
const { getDashboardPrefs, updateDashboardPrefs } = useSettingsApi()

// 租户概览数据（业务/租户管理员角色使用）
const overview = ref<DashboardOverview | null>(null)
// 平台概览数据（系统管理员角色使用）
const platformOverview = ref<PlatformDashboardOverview | null>(null)
// 概览数据加载状态
const overviewLoading = ref(false)

// 是否为平台管理员（系统管理员）
const isPlatformAdmin = computed(() => effectiveActiveRoleForApi.value === 'system_admin')

// 当前有效的概览数据，未加载时使用空占位
const dash = computed(() => overview.value ?? EMPTY_OVERVIEW)

// 根据当前角色和页面权限过滤可用的仪表盘组件
const availableWidgets = computed(() => {
  const role = effectiveActiveRoleForApi.value
  if (!role) return []

  let widgets = OVERVIEW_WIDGETS.filter(w =>
    w.requiredPermissions.includes(role as PermissionGroup),
  )

  // business 角色额外按 page_permissions 过滤，确保只展示有权限访问的页面对应组件
  if (role === 'business') {
    const allowedPaths = new Set(menus.value.map((m: any) => m.path).filter(Boolean))
    widgets = widgets.filter((w) => {
      const requiredPerm = WIDGET_PAGE_PERMISSION_MAP[w.id]
      // 空字符串或 undefined 表示始终可见
      return !requiredPerm || allowedPaths.has(requiredPerm)
    })
  }

  return widgets
})

// 默认启用的组件 ID 列表（用于首次加载时的初始状态）
const defaultPrefs = computed(() =>
  availableWidgets.value.filter(w => w.defaultEnabled).map(w => w.id))

// 当前已启用的组件 ID 列表（用户可自定义）
const enabledWidgets = ref<OverviewWidgetId[]>([])
// 各组件的尺寸偏好（sm/md/lg）
const widgetSizes = ref<Partial<Record<OverviewWidgetId, 'sm' | 'md' | 'lg'>>>({})

// 将后端返回的仪表盘偏好应用到本地状态
function applyDashboardPrefs(prefs: DashboardPref) {
  const allowed = new Set(availableWidgets.value.map(w => w.id))
  const raw = (prefs.enabled_widgets || []).filter((id): id is OverviewWidgetId =>
    OVERVIEW_WIDGET_ID_SET.has(id as OverviewWidgetId) && allowed.has(id as OverviewWidgetId))
  enabledWidgets.value = raw.length > 0 ? raw : [...defaultPrefs.value]
  widgetSizes.value = { ...(prefs.widget_sizes as Partial<Record<OverviewWidgetId, 'sm' | 'md' | 'lg'>> || {}) }
}

// 空偏好占位，用于接口异常时的降级处理
const EMPTY_DASH_PREFS: DashboardPref = { enabled_widgets: [], widget_sizes: {} }

// 加载概览页数据：先获取仪表盘偏好，再根据角色拉取对应的概览数据
async function loadOverviewPage() {
  const role = effectiveActiveRoleForApi.value
  if (!role) return

  if (role === 'system_admin') {
    overview.value = null
  }
  else {
    platformOverview.value = null
  }

  overviewLoading.value = true
  try {
    const prefs = await getDashboardPrefs().catch(() => ({ ...EMPTY_DASH_PREFS, enabled_widgets: [] as string[] }))
    applyDashboardPrefs(prefs)

    if (role === 'system_admin') {
      platformOverview.value = await fetchPlatformDashboardOverview()
    }
    else {
      overview.value = await fetchDashboardOverview()
    }
  }
  catch (e: unknown) {
    const msg = e instanceof Error ? e.message : String(e)
    message.error(msg || t('overview.loadFailed'))
    if (role === 'system_admin') {
      platformOverview.value = null
    }
    else {
      overview.value = null
    }
  }
  finally {
    overviewLoading.value = false
  }
}

// 监听角色变化，切换角色时重新加载概览数据
watch(effectiveActiveRoleForApi, () => { void loadOverviewPage() }, { immediate: true })

// 判断指定组件是否已启用
const isEnabled = (id: OverviewWidgetId) => {
  return enabledWidgets.value.includes(id) && availableWidgets.value.some(w => w.id === id)
}

// 是否处于自定义布局模式
const customizing = ref(false)
// 切换组件的启用/禁用状态（仅在自定义模式下有效）
const toggleWidget = (id: OverviewWidgetId) => {
  if (!customizing.value) return
  const idx = enabledWidgets.value.indexOf(id)
  if (idx >= 0) enabledWidgets.value.splice(idx, 1)
  else enabledWidgets.value.push(id)
}

// 保存仪表盘布局偏好到后端
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

// 获取指定组件的当前尺寸，未设置时使用默认值
const getWidgetSize = (id: OverviewWidgetId) => {
  if (widgetSizes.value[id]) return widgetSizes.value[id]
  const w = OVERVIEW_WIDGETS.find(x => x.id === id)
  return w?.size || 'md'
}

// 循环切换组件尺寸：sm → md → lg → sm
const cycleWidgetSize = (id: OverviewWidgetId) => {
  const current = getWidgetSize(id)
  const nextSize = current === 'sm' ? 'md' : current === 'md' ? 'lg' : 'sm'
  widgetSizes.value[id] = nextSize
}

// 根据当前小时生成问候语
const greeting = computed(() => {
  const h = new Date().getHours()
  return h < 6 ? t('overview.greeting.lateNight') : h < 12 ? t('overview.greeting.morning') : h < 14 ? t('overview.greeting.noon') : h < 18 ? t('overview.greeting.afternoon') : t('overview.greeting.evening')
})

// 格式化大数字，超过 1 万时显示为 K 单位
const formatNum = (n: number) => n >= 10000 ? (n / 1000).toFixed(1) + 'K' : n.toLocaleString()

// 最近动态各类型对应的颜色
const activityKindColor: Record<string, string> = {
  audit: 'var(--color-primary)',
  archive: 'var(--color-success)',
  cron: 'var(--color-accent)',
}

// 将动态类型 key 转换为可读标签
function kindLabel(kind: string) {
  switch (kind) {
    case 'audit': return t('overview.activity.audit')
    case 'archive': return t('overview.activity.archive')
    case 'cron': return t('overview.activity.cron')
    default: return kind
  }
}

// 格式化动态时间为本地化短格式
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

// 获取定时任务的描述文本，优先使用自定义描述，其次使用国际化翻译
function cronTaskDescriptionLabel(task: { description?: string; task_type?: string }) {
  if (task.description?.trim()) return task.description
  const key = `cron.taskType.${task.task_type}` as const
  const tr = t(key as string)
  return tr && tr !== key ? tr : task.task_type ?? ''
}

// 趋势图的日期分类数据
const trendCategories = computed(() => dash.value.weekly_trend.map(d => d.date))
// 趋势图的系列数据（审核、定时任务、归档）
const trendSeries = computed(() => [
  { name: t('overview.auditWorkbench'), data: dash.value.weekly_trend.map(d => d.audit_count), color: '#4f46e5' },
  { name: t('overview.cronTasks'), data: dash.value.weekly_trend.map(d => d.cron_count), color: '#06b6d4' },
  { name: t('overview.archiveReview'), data: dash.value.weekly_trend.map(d => d.archive_count), color: '#10b981' },
])

// 部门分布图的标签配置
const deptChartLabels = computed(() => ({
  audit: t('overview.auditWorkbench'),
  cron: t('overview.cronTasks'),
  archive: t('overview.archiveReview'),
}))

// 获取组件在已启用列表中的排序位置，未启用时排到末尾
const getWidgetOrder = (id: OverviewWidgetId) => {
  const index = enabledWidgets.value.indexOf(id)
  return index >= 0 ? index : 999
}

// 当前正在拖拽的组件 ID
const draggedWidget = ref<OverviewWidgetId | null>(null)

// 拖拽开始：记录被拖拽的组件 ID
const onDragStart = (e: DragEvent, id: OverviewWidgetId) => {
  if (!customizing.value) return
  draggedWidget.value = id
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.dropEffect = 'move'
    e.dataTransfer.setData('text/plain', id)
  }
}

// 拖拽放置：交换两个组件的排列顺序
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

// 计算 Token 使用量占配额的百分比，最大 100%
function tokenPct(used: number, quota: number) {
  if (quota <= 0) return 0
  return Math.min(100, (used / quota) * 100)
}</script>

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

      <!--===== 本周概览（weekly_overview）=====-->
      <div v-if="isEnabled('weekly_overview')"
       :class="['widget', `widget--${getWidgetSize('weekly_overview')}`, { 'widget--editing': customizing }]"
       :style="{ order: getWidgetOrder('weekly_overview') }"
       :draggable="customizing"
       @dragstart="onDragStart($event, 'weekly_overview')"
       @dragover.prevent
       @dragenter.prevent
       @drop="onDrop($event, 'weekly_overview')">
        <div class="widget-title">
          <div class="widget-title-left"><ThunderboltOutlined /> {{ t('overview.widgetTitle.weekly_overview') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('weekly_overview')" :title="t('overview.resizeWidget')" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="weekly-overview">
          <div class="wo-total">
            <div class="wo-total-num">{{ dash.weekly_overview.total }}</div>
            <div class="wo-total-label">{{ t('overview.weeklyTotal') }}</div>
          </div>
          <div class="wo-items">
            <div class="wo-item">
              <span class="wo-num" style="color: var(--color-primary);">{{ dash.weekly_overview.audit_count }}</span>
              <span class="wo-label">{{ t('overview.auditWorkbench') }}</span>
            </div>
            <div class="wo-item">
              <span class="wo-num" style="color: var(--color-success);">{{ dash.weekly_overview.archive_count }}</span>
              <span class="wo-label">{{ t('overview.archiveReview') }}</span>
            </div>
            <div class="wo-item">
              <span class="wo-num" style="color: var(--color-accent);">{{ dash.weekly_overview.cron_count }}</span>
              <span class="wo-label">{{ t('overview.cronTasks') }}</span>
            </div>
          </div>
        </div>
      </div>

      <!--===== 待办任务（pending_tasks）=====-->
      <div v-if="isEnabled('pending_tasks')"
       :class="['widget', `widget--${getWidgetSize('pending_tasks')}`, { 'widget--editing': customizing }]"
       :style="{ order: getWidgetOrder('pending_tasks') }"
       :draggable="customizing"
       @dragstart="onDragStart($event, 'pending_tasks')"
       @dragover.prevent
       @dragenter.prevent
       @drop="onDrop($event, 'pending_tasks')">
        <div class="widget-title">
          <div class="widget-title-left"><ClockCircleOutlined /> {{ t('overview.widgetTitle.pending_tasks') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('pending_tasks')" :title="t('overview.resizeWidget')" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="pending-split">
          <div class="pending-item">
            <div class="pending-num">{{ dash.pending_tasks?.audit_pending ?? 0 }}</div>
            <div class="pending-label">{{ t('overview.auditPending') }}</div>
          </div>
          <div class="pending-item">
            <div class="pending-num">{{ dash.pending_tasks?.archive_pending ?? 0 }}</div>
            <div class="pending-label">{{ t('overview.archivePending') }}</div>
          </div>
        </div>
        <div class="pending-total">
          {{ t('overview.totalPending') }}: {{ dash.pending_tasks?.total ?? 0 }}
        </div>
        <a-button type="link" size="small" @click="navigateTo('/dashboard')">{{ t('overview.goToWorkbench') }} →</a-button>
      </div>

      <!--===== 审核趋势（weekly_trend）=====-->
      <div v-if="isEnabled('weekly_trend')"
       :class="['widget', `widget--${getWidgetSize('weekly_trend')}`, { 'widget--editing': customizing }]"
       :style="{ order: getWidgetOrder('weekly_trend') }"
       :draggable="customizing"
       @dragstart="onDragStart($event, 'weekly_trend')"
       @dragover.prevent
       @dragenter.prevent
       @drop="onDrop($event, 'weekly_trend')">
        <div class="widget-title">
          <div class="widget-title-left"><RiseOutlined /> {{ t('overview.widgetTitle.weekly_trend') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('weekly_trend')" :title="t('overview.resizeWidget')" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <StackedBarChart
          v-if="dash.weekly_trend.length > 0"
          :categories="trendCategories"
          :series="trendSeries"
          height="240px"
        />
        <div v-else class="widget-empty">{{ t('overview.noData') }}</div>
      </div>

      <!--===== 定时任务（cron_tasks）=====-->
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
        <div class="cron-table" v-if="dash.cron_tasks?.length">
          <div class="cron-row cron-row--header">
            <span class="cron-cell cron-cell--name">{{ t('overview.cronTaskName') }}</span>
            <span class="cron-cell cron-cell--desc">{{ t('overview.cronTaskDesc') }}</span>
            <span class="cron-cell cron-cell--expr">{{ t('overview.cronExpression') }}</span>
            <span class="cron-cell cron-cell--status">{{ t('overview.cronStatusLabel') }}</span>
          </div>
          <div v-for="c in dash.cron_tasks" :key="c.id" class="cron-row">
            <span class="cron-cell cron-cell--name">{{ c.task_label }}</span>
            <span class="cron-cell cron-cell--desc">{{ cronTaskDescriptionLabel(c) }}</span>
            <span class="cron-cell cron-cell--expr">{{ c.cron_expression }}</span>
            <span class="cron-cell cron-cell--status" :style="{ color: c.is_active ? 'var(--color-success)' : 'var(--color-text-tertiary)' }">
              {{ c.is_active ? t('overview.cronActive') : t('overview.cronInactive') }}
            </span>
          </div>
        </div>
        <div v-else class="widget-empty">{{ t('overview.noData') }}</div>
      </div>

      <!--===== 最近动态（recent_activity）=====-->
      <div v-if="isEnabled('recent_activity')"
       :class="['widget', `widget--${getWidgetSize('recent_activity')}`, { 'widget--editing': customizing }]"
       :style="{ order: getWidgetOrder('recent_activity') }"
       :draggable="customizing"
       @dragstart="onDragStart($event, 'recent_activity')"
       @dragover.prevent
       @dragenter.prevent
       @drop="onDrop($event, 'recent_activity')">
        <div class="widget-title">
          <div class="widget-title-left"><ClockCircleOutlined /> {{ t('overview.widgetTitle.recent_activity') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('recent_activity')" :title="t('overview.resizeWidget')" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="activity-list" v-if="dash.recent_activity.length > 0">
          <div v-for="a in dash.recent_activity.slice(0, 10)" :key="a.id" class="activity-item">
            <div class="activity-dot" :style="{ background: (a.kind === 'cron' && a.cron_status === 'failed') ? 'var(--color-danger)' : (activityKindColor[a.kind] || 'var(--color-text-tertiary)') }" />
            <div class="activity-body">
              <span class="activity-action">{{ kindLabel(a.kind) }}</span>
              <span class="activity-target">{{ a.title }}</span>
              <span v-if="a.kind === 'audit' && a.recommendation" class="activity-tag" :class="{
                'activity-tag--approve': a.recommendation === 'approve',
                'activity-tag--return': a.recommendation === 'return',
                'activity-tag--review': a.recommendation === 'review',
              }">
                {{ t(`overview.recommendation.${a.recommendation}`) }} · {{ a.score }}{{ t('overview.scoreUnit') }}
              </span>
              <span v-if="a.kind === 'archive' && a.compliance" class="activity-tag" :class="{
                'activity-tag--compliant': a.compliance === 'compliant',
                'activity-tag--non-compliant': a.compliance === 'non_compliant',
                'activity-tag--partial': a.compliance === 'partially_compliant',
              }">
                {{ t(`overview.compliance.${a.compliance}`) }} · {{ a.compliance_score }}{{ t('overview.scoreUnit') }}
              </span>
              <span v-if="a.kind === 'cron' && a.cron_status" class="activity-tag" :class="a.cron_status === 'failed' ? 'activity-tag--cron-fail' : 'activity-tag--cron'">
                {{ t(`overview.cronStatus.${a.cron_status}`) }} · {{ a.task_label }}
              </span>
            </div>
            <div class="activity-meta">
              <span class="activity-user">{{ a.user_name }}</span>
              <span class="activity-time">{{ formatActivityTime(a.created_at) }}</span>
            </div>
          </div>
        </div>
        <div v-else class="widget-empty">{{ t('overview.noData') }}</div>
      </div>

      <!--===== 部门分布（dept_distribution）=====-->
      <div v-if="isEnabled('dept_distribution')"
       :class="['widget', `widget--${getWidgetSize('dept_distribution')}`, { 'widget--editing': customizing }]"
       :style="{ order: getWidgetOrder('dept_distribution') }"
       :draggable="customizing"
       @dragstart="onDragStart($event, 'dept_distribution')"
       @dragover.prevent
       @dragenter.prevent
       @drop="onDrop($event, 'dept_distribution')">
        <div class="widget-title">
          <div class="widget-title-left"><TeamOutlined /> {{ t('overview.widgetTitle.dept_distribution') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('dept_distribution')" :title="t('overview.resizeWidget')" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <DeptDistributionChart
          v-if="dash.dept_distribution?.length"
          :data="dash.dept_distribution"
          :labels="deptChartLabels"
          height="300px"
        />
        <div v-else class="widget-empty">{{ t('overview.noData') }}</div>
      </div>

      <!--===== 用户活跃排名（user_activity）=====-->
      <div v-if="isEnabled('user_activity')"
       :class="['widget', `widget--${getWidgetSize('user_activity')}`, { 'widget--editing': customizing }]"
       :style="{ order: getWidgetOrder('user_activity') }"
       :draggable="customizing"
       @dragstart="onDragStart($event, 'user_activity')"
       @dragover.prevent
       @dragenter.prevent
       @drop="onDrop($event, 'user_activity')">
        <div class="widget-title">
          <div class="widget-title-left"><TeamOutlined /> {{ t('overview.widgetTitle.user_activity') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('user_activity')" :title="t('overview.resizeWidget')" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="rank-list" v-if="dash.user_activity?.length">
          <div v-for="(u, i) in dash.user_activity" :key="u.username" class="rank-item">
            <span class="rank-num" :class="{ 'rank-num--top': i < 3 }">{{ i + 1 }}</span>
            <div class="rank-info">
              <span class="rank-name">{{ u.display_name }}</span>
              <span class="rank-dept">{{ u.department }}</span>
            </div>
            <span class="rank-count">{{ u.audit_count }} {{ t('overview.times') }}</span>
          </div>
        </div>
        <div v-else class="widget-empty">{{ t('overview.emptyUserActivity') }}</div>
      </div>

      <!--===== 平台：租户规模（platform_tenant_stats）=====-->
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
            <div class="summary-num">{{ platformOverview?.tenant_stats?.tenant_total ?? 0 }}</div>
            <div class="summary-label">{{ t('overview.platformTenantsTotal') }}</div>
          </div>
          <div class="summary-card summary-card--approved">
            <div class="summary-num">{{ platformOverview?.tenant_stats?.tenant_active ?? 0 }}</div>
            <div class="summary-label">{{ t('overview.platformTenantsActive') }}</div>
          </div>
        </div>
        <p class="active-criteria-text">{{ platformOverview?.tenant_stats?.active_criteria || t('overview.activeCriteria') }}</p>
        <div class="tenant-table" v-if="platformOverview?.tenant_stats?.tenants?.length">
          <div class="tenant-row tenant-row--header">
            <span>{{ t('overview.tenantName') }}</span>
            <span>{{ t('overview.tenantUserCount') }}</span>
            <span>{{ t('overview.tenantStatusLabel') }}</span>
          </div>
          <div v-for="row in platformOverview.tenant_stats.tenants" :key="row.tenant_id" class="tenant-row">
            <span class="tenant-name">{{ row.tenant_name }}</span>
            <span>{{ row.user_count }}</span>
            <span class="tenant-status" :style="{ color: row.is_active ? 'var(--color-success)' : 'var(--color-text-tertiary)' }">
              {{ row.is_active ? t('overview.tenantActive') : t('overview.tenantInactive') }}
            </span>
          </div>
        </div>
      </div>

      <!--===== 平台：AI 模型表现（ai_performance）=====-->
      <div v-if="isEnabled('ai_performance')"
       :class="['widget', `widget--${getWidgetSize('ai_performance')}`, { 'widget--editing': customizing }]"
       :style="{ order: getWidgetOrder('ai_performance') }"
       :draggable="customizing"
       @dragstart="onDragStart($event, 'ai_performance')"
       @dragover.prevent
       @dragenter.prevent
       @drop="onDrop($event, 'ai_performance')">
        <div class="widget-title">
          <div class="widget-title-left"><ThunderboltOutlined /> {{ t('overview.widgetTitle.ai_performance') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('ai_performance')" :title="t('overview.resizeWidget')" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="ai-model-list" v-if="platformOverview?.ai_performance?.models?.length">
          <div v-for="m in platformOverview.ai_performance.models" :key="m.model_config_id" class="ai-model-card">
            <div class="ai-model-header">
              <span class="ai-model-name">{{ m.display_name || m.model_name }}</span>
              <span class="ai-model-provider">{{ m.provider }}</span>
            </div>
            <div class="ai-model-stats">
              <div class="ai-model-stat-group">
                <div class="ai-model-stat-title">{{ t('overview.reasoningCalls') }}</div>
                <div class="ai-model-stat-row">
                  <span>{{ t('overview.callCount') }}: {{ formatNum(m.reasoning_stats.calls) }}</span>
                  <span>{{ t('overview.successRate') }}: {{ m.reasoning_stats.success_rate.toFixed(1) }}%</span>
                  <span>{{ t('overview.avgResponseTime') }}: {{ m.reasoning_stats.avg_ms }}ms</span>
                </div>
              </div>
              <div class="ai-model-stat-group">
                <div class="ai-model-stat-title">{{ t('overview.structuredCalls') }}</div>
                <div class="ai-model-stat-row">
                  <span>{{ t('overview.callCount') }}: {{ formatNum(m.structured_stats.calls) }}</span>
                  <span>{{ t('overview.successRate') }}: {{ m.structured_stats.success_rate.toFixed(1) }}%</span>
                  <span>{{ t('overview.avgResponseTime') }}: {{ m.structured_stats.avg_ms }}ms</span>
                </div>
              </div>
            </div>
            <div class="ai-model-footer">
              <span>{{ t('overview.totalCalls') }}: {{ formatNum(m.total_calls) }}</span>
              <span>{{ t('overview.overallSuccessRate') }}: {{ m.overall_success_rate.toFixed(1) }}%</span>
            </div>
          </div>
        </div>
        <div v-else class="widget-empty">{{ t('overview.noData') }}</div>
      </div>

      <!--===== 平台：租户资源用量（tenant_usage）=====-->
      <div v-if="isEnabled('tenant_usage')"
       :class="['widget', `widget--${getWidgetSize('tenant_usage')}`, { 'widget--editing': customizing }]"
       :style="{ order: getWidgetOrder('tenant_usage') }"
       :draggable="customizing"
       @dragstart="onDragStart($event, 'tenant_usage')"
       @dragover.prevent
       @dragenter.prevent
       @drop="onDrop($event, 'tenant_usage')">
        <div class="widget-title">
          <div class="widget-title-left"><CloudServerOutlined /> {{ t('overview.widgetTitle.tenant_usage') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('tenant_usage')" :title="t('overview.resizeWidget')" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="usage-rows" v-if="platformOverview?.tenant_usage_list?.length">
          <div v-for="row in platformOverview.tenant_usage_list" :key="row.tenant_id" class="usage-row">
            <span class="usage-label" :title="row.tenant_code">{{ row.tenant_name }}</span>
            <div class="usage-bar-wrap">
              <div class="usage-bar" :style="{ width: tokenPct(row.token_used, row.token_quota) + '%' }" />
            </div>
            <span class="usage-text">{{ formatNum(row.token_used) }} / {{ formatNum(row.token_quota) }}</span>
          </div>
        </div>
        <div v-else class="widget-empty">{{ t('overview.noData') }}</div>
      </div>

      <!--===== 平台：租户审核排名（platform_tenant_ranking）=====-->
      <div v-if="isEnabled('platform_tenant_ranking')"
       :class="['widget', `widget--${getWidgetSize('platform_tenant_ranking')}`, { 'widget--editing': customizing }]"
       :style="{ order: getWidgetOrder('platform_tenant_ranking') }"
       :draggable="customizing"
       @dragstart="onDragStart($event, 'platform_tenant_ranking')"
       @dragover.prevent
       @dragenter.prevent
       @drop="onDrop($event, 'platform_tenant_ranking')">
        <div class="widget-title">
          <div class="widget-title-left"><BarChartOutlined /> {{ t('overview.widgetTitle.platform_tenant_ranking') }}</div>
          <div class="widget-actions" v-if="customizing" @click.stop="cycleWidgetSize('platform_tenant_ranking')" :title="t('overview.resizeWidget')" style="cursor: pointer; color: var(--color-primary);"><AppstoreOutlined /></div>
        </div>
        <div class="tenant-ranking-table" v-if="platformOverview?.tenant_ranking?.length">
          <div class="tr-row tr-row--header">
            <span class="tr-cell tr-cell--rank">#</span>
            <span class="tr-cell tr-cell--name">{{ t('overview.tenantName') }}</span>
            <span class="tr-cell">{{ t('overview.auditSnapshots') }}</span>
            <span class="tr-cell">{{ t('overview.archiveSnapshots') }}</span>
            <span class="tr-cell">{{ t('overview.cronExecutions') }}</span>
            <span class="tr-cell tr-cell--fail">{{ t('overview.auditFailures') }}</span>
            <span class="tr-cell tr-cell--fail">{{ t('overview.archiveFailures') }}</span>
          </div>
          <div v-for="(row, i) in platformOverview.tenant_ranking" :key="row.tenant_id" class="tr-row">
            <span class="tr-cell tr-cell--rank">
              <span class="rank-num" :class="{ 'rank-num--top': i < 3 }">{{ i + 1 }}</span>
            </span>
            <span class="tr-cell tr-cell--name tenant-name">{{ row.tenant_name }}</span>
            <span class="tr-cell">{{ row.audit_count }}</span>
            <span class="tr-cell">{{ row.archive_count }}</span>
            <span class="tr-cell">{{ row.cron_count }}</span>
            <span class="tr-cell tr-cell--fail" :style="{ color: row.audit_failed > 0 ? 'var(--color-danger)' : undefined }">{{ row.audit_failed }}</span>
            <span class="tr-cell tr-cell--fail" :style="{ color: row.archive_failed > 0 ? 'var(--color-danger)' : undefined }">{{ row.archive_failed }}</span>
          </div>
        </div>
        <div v-else class="widget-empty">{{ t('overview.noData') }}</div>
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


/*小部件网格*/
.widget-grid { display: grid; grid-template-columns: repeat(12, 1fr); gap: 16px; }
.widget {
  background: var(--color-bg-card); border: 1px solid var(--color-border-light);
  border-radius: var(--radius-lg); padding: 20px;
  box-shadow: var(--shadow-xs); transition: box-shadow var(--transition-base);
}
.widget:hover { box-shadow: var(--shadow-sm); }
.widget--editing { border: 1px dashed var(--color-primary); cursor: grab; transform: scale(0.99); }
.widget--editing:active { cursor: grabbing; }
.widget--sm { grid-column: span 4; }
.widget--md { grid-column: span 6; }
.widget--lg { grid-column: span 12; }

.widget-title {
  font-size: 14px; font-weight: 600; color: var(--color-text-primary);
  margin-bottom: 16px; display: flex; align-items: center; gap: 8px;
  justify-content: space-between; width: 100%;
}
.widget-title-left { flex: 1; display: flex; align-items: center; gap: 8px; }
.widget-actions { flex-shrink: 0; padding: 4px; border-radius: 4px; transition: background 0.2s; }
.widget-actions:hover { background: rgba(0,0,0,0.05); }

.widget-empty { padding: 24px; text-align: center; color: var(--color-text-tertiary); font-size: 13px; }

/*本周概览*/
.weekly-overview { display: flex; align-items: center; gap: 32px; flex-wrap: wrap; }
.wo-total { text-align: center; padding: 16px 24px; background: var(--color-primary-bg); border-radius: var(--radius-md); }
.wo-total-num { font-size: 36px; font-weight: 700; color: var(--color-primary); line-height: 1.2; }
.wo-total-label { font-size: 13px; color: var(--color-text-tertiary); margin-top: 4px; }
.wo-items { display: flex; gap: 24px; flex: 1; }
.wo-item { text-align: center; flex: 1; padding: 12px 0; }
.wo-num { font-size: 24px; font-weight: 700; display: block; line-height: 1.2; }
.wo-label { font-size: 12px; color: var(--color-text-tertiary); margin-top: 4px; display: block; }

/*待办任务*/
.pending-split { display: flex; gap: 16px; margin-bottom: 12px; }
.pending-item { flex: 1; text-align: center; padding: 16px 8px; background: var(--color-bg-page); border-radius: var(--radius-md); }
.pending-num { font-size: 32px; font-weight: 700; color: var(--color-primary); line-height: 1; }
.pending-label { font-size: 12px; color: var(--color-text-tertiary); margin-top: 6px; }
.pending-total { font-size: 13px; color: var(--color-text-secondary); text-align: center; margin-bottom: 8px; }

/*审计总结*/
.summary-cards { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; }
.summary-card {
  text-align: center; padding: 16px 8px; border-radius: var(--radius-md);
  background: var(--color-bg-page);
}
.summary-card--total { background: var(--color-primary-bg); }
.summary-num { font-size: 28px; font-weight: 700; color: var(--color-text-primary); line-height: 1.2; }
.summary-label { font-size: 12px; color: var(--color-text-tertiary); margin-top: 4px; }
.summary-card--total .summary-num { color: var(--color-primary); }

/*定时任务表格*/
.cron-table { display: flex; flex-direction: column; }
.cron-row { display: grid; grid-template-columns: 2fr 3fr 1.5fr 1fr; gap: 8px; padding: 8px 0; border-bottom: 1px solid var(--color-border-light); font-size: 13px; color: var(--color-text-secondary); align-items: center; }
.cron-row:last-child { border-bottom: none; }
.cron-row--header { font-weight: 600; color: var(--color-text-tertiary); font-size: 12px; }
.cron-cell--name { font-weight: 500; color: var(--color-text-primary); }
.cron-cell--desc { color: var(--color-text-tertiary); font-size: 12px; }
.cron-cell--expr { font-family: var(--font-mono, monospace); font-size: 12px; }

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
.activity-tag { font-size: 11px; color: var(--color-primary); background: var(--color-primary-bg); padding: 1px 6px; border-radius: var(--radius-full); margin-left: 6px; white-space: nowrap; }
.activity-tag--approve { color: var(--color-success); background: var(--color-success-bg); }
.activity-tag--return { color: var(--color-warning); background: var(--color-warning-bg); }
.activity-tag--review { color: var(--color-info); background: var(--color-info-bg); }
.activity-tag--compliant { color: var(--color-success); background: var(--color-success-bg); }
.activity-tag--non-compliant { color: var(--color-danger); background: var(--color-danger-bg); }
.activity-tag--partial { color: var(--color-warning); background: var(--color-warning-bg); }
.activity-tag--cron { color: var(--color-accent); background: rgba(6,182,212,0.1); }
.activity-tag--cron-fail { color: var(--color-danger); background: var(--color-danger-bg); }
.activity-meta { display: flex; flex-direction: column; align-items: flex-end; flex-shrink: 0; }
.activity-user { font-size: 12px; color: var(--color-text-tertiary); }
.activity-time { font-size: 11px; color: var(--color-text-tertiary); }

/*排名*/
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

/*租户规模*/
.active-criteria-text { font-size: 12px; color: var(--color-text-tertiary); margin: 12px 0 16px; }
.tenant-table { display: flex; flex-direction: column; }
.tenant-row {
  display: grid; grid-template-columns: 2fr 1fr 1fr; gap: 8px;
  padding: 8px 0; border-bottom: 1px solid var(--color-border-light);
  font-size: 13px; color: var(--color-text-secondary); align-items: center;
}
.tenant-row:last-child { border-bottom: none; }
.tenant-row--header { font-weight: 600; color: var(--color-text-tertiary); font-size: 12px; }
.tenant-name { font-weight: 500; color: var(--color-text-primary); }
.tenant-status { font-weight: 500; }

/*AI 模型表现*/
.ai-model-list { display: flex; flex-direction: column; gap: 16px; }
.ai-model-card { background: var(--color-bg-page); border-radius: var(--radius-md); padding: 16px; }
.ai-model-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.ai-model-name { font-size: 14px; font-weight: 600; color: var(--color-text-primary); }
.ai-model-provider { font-size: 12px; color: var(--color-text-tertiary); background: var(--color-bg-card); padding: 2px 8px; border-radius: var(--radius-full); }
.ai-model-stats { display: flex; gap: 16px; flex-wrap: wrap; }
.ai-model-stat-group { flex: 1; min-width: 200px; }
.ai-model-stat-title { font-size: 12px; font-weight: 600; color: var(--color-text-secondary); margin-bottom: 6px; }
.ai-model-stat-row { display: flex; flex-wrap: wrap; gap: 12px; font-size: 12px; color: var(--color-text-tertiary); }
.ai-model-footer { display: flex; gap: 16px; margin-top: 12px; padding-top: 10px; border-top: 1px solid var(--color-border-light); font-size: 13px; font-weight: 500; color: var(--color-text-secondary); }

/*使用栏*/
.usage-rows { display: flex; flex-direction: column; gap: 16px; }
.usage-row { display: flex; align-items: center; gap: 12px; }
.usage-label { font-size: 13px; color: var(--color-text-secondary); width: 100px; flex-shrink: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.usage-bar-wrap { flex: 1; height: 12px; background: var(--color-bg-page); border-radius: 6px; overflow: hidden; }
.usage-bar { height: 100%; background: var(--color-primary); border-radius: 6px; transition: width 0.5s ease; }
.usage-text { font-size: 12px; color: var(--color-text-tertiary); width: 120px; text-align: right; flex-shrink: 0; }

/*租户审核排名表格*/
.tenant-ranking-table { display: flex; flex-direction: column; overflow-x: auto; }
.tr-row { display: grid; grid-template-columns: 40px 2fr 1fr 1fr 1fr 1fr 1fr; gap: 8px; padding: 8px 0; border-bottom: 1px solid var(--color-border-light); font-size: 13px; color: var(--color-text-secondary); align-items: center; }
.tr-row:last-child { border-bottom: none; }
.tr-row--header { font-weight: 600; color: var(--color-text-tertiary); font-size: 12px; }
.tr-cell { text-align: center; }
.tr-cell--rank { text-align: center; }
.tr-cell--name { text-align: left; }
.tr-cell--fail { font-size: 12px; }

@media (max-width: 1024px) {
  .widget--sm, .widget--md { grid-column: span 6; }
  .summary-cards { grid-template-columns: repeat(2, 1fr); }
  .weekly-overview { flex-direction: column; gap: 16px; }
  .wo-items { width: 100%; }
}
@media (max-width: 768px) {
  .widget--sm, .widget--md, .widget--lg { grid-column: span 12; }
  .summary-cards { grid-template-columns: repeat(2, 1fr); }
  .ov-header { flex-direction: column; }
  .pending-split { flex-direction: column; }
  .tr-row { grid-template-columns: 30px 1.5fr repeat(5, 1fr); font-size: 12px; }
}
</style>
