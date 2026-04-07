<script setup lang="ts">
import {
  SearchOutlined,
  ClockCircleOutlined,
  FolderOpenOutlined,
  ExportOutlined,
  EyeOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  SyncOutlined,
  AppstoreOutlined,
  FilterOutlined,
  AlertOutlined,
  SafetyCertificateOutlined,
  InfoCircleOutlined,
  CloseOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import type { Dayjs } from 'dayjs'
import 'dayjs/locale/zh-cn'
import { useI18n } from '~/composables/useI18n'
import { useAdminDataApi } from '~/composables/useAdminDataApi'
import { useAuditApi } from '~/composables/useAuditApi'
import { useArchiveReviewApi } from '~/composables/useArchiveReviewApi'
import type {
  AuditLogItem,
  AuditSnapshotItem,
  AuditSnapshotStats,
  ArchiveLogItem,
  ArchiveSnapshotItem,
  ArchiveSnapshotStats,
  CronLogItem,
  CronLogStats,
} from '~/types/admin-data'

definePageMeta({ middleware: 'auth', layout: 'default' })

type MainTab = 'audit' | 'cron' | 'archive'
type AuditSubTab = 'all' | 'approve' | 'return' | 'review'
type CronSubTab = 'all' | 'success' | 'failed' | 'running'
type ArchiveSubTab = 'all' | 'compliant' | 'partially_compliant' | 'non_compliant'

type AuditRuleItem = {
  rule_id?: string
  rule_name?: string
  rule_content?: string
  passed?: boolean
  reasoning?: string
  reason?: string
}

type AuditDetailPayload = {
  recommendation?: string
  overall_score?: number
  score?: number
  rule_results?: AuditRuleItem[]
  details?: AuditRuleItem[]
  risk_points?: string[]
  suggestions?: string[]
  ai_reasoning?: string
  duration_ms?: number
}

type ArchiveFlowNode = {
  node_id?: string
  node_name?: string
  compliant?: boolean
  reasoning?: string
}

type ArchiveRuleItem = {
  rule_id?: string
  rule_name?: string
  passed?: boolean
  reasoning?: string
}

type ArchiveDetailPayload = {
  overall_compliance?: string
  overall_score?: number
  flow_audit?: {
    is_complete?: boolean
    missing_nodes?: string[]
    node_results?: ArchiveFlowNode[]
  }
  rule_audit?: ArchiveRuleItem[]
  risk_points?: string[]
  suggestions?: string[]
  ai_summary?: string
  duration_ms?: number
}

const { t } = useI18n()
const {
  listAuditSnapshots,
  getAuditSnapshotStats,
  getAuditSnapshotChain,
  exportAuditLogs,
  listArchiveSnapshots,
  getArchiveSnapshotStats,
  getArchiveSnapshotChain,
  exportArchiveLogs,
  listCronLogs,
  getCronLogStats,
  exportCronLogs,
} = useAdminDataApi()
const { getProcessTypes: getAuditProcessTypes } = useAuditApi()
const { getProcessTypes: getArchiveProcessTypes } = useArchiveReviewApi()

const activeTab = ref<MainTab>('audit')
const activeAuditSubTab = ref<AuditSubTab>('all')
const activeCronSubTab = ref<CronSubTab>('all')
const activeArchiveSubTab = ref<ArchiveSubTab>('all')

const auditProcessOptions = ref<{ label: string; value: string }[]>([])
const archiveProcessOptions = ref<{ label: string; value: string }[]>([])

const auditStats = ref<AuditSnapshotStats>({
  total: 0,
  approve_count: 0,
  return_count: 0,
  review_count: 0,
})
const cronStats = ref<CronLogStats>({
  total: 0,
  success: 0,
  failed: 0,
  running: 0,
})
const archiveStats = ref<ArchiveSnapshotStats>({
  total: 0,
  compliant: 0,
  partial: 0,
  non_compliant: 0,
})

const auditSnapshots = ref<AuditSnapshotItem[]>([])
const cronLogs = ref<CronLogItem[]>([])
const archiveSnapshots = ref<ArchiveSnapshotItem[]>([])

const auditLoading = ref(false)
const cronLoading = ref(false)
const archiveLoading = ref(false)

const auditSearch = ref('')
const auditFilterProcessType = ref<string | undefined>(undefined)
const auditFilterOperator = ref('')
const auditFilterDepartment = ref<string | undefined>(undefined)
const auditFilterDateRange = ref<[Dayjs, Dayjs] | undefined>(undefined)
const auditShowFilters = ref(false)
const auditPage = ref(1)
const auditPageSize = ref(10)
const auditTotal = ref(0)

const cronFilterTaskType = ref<string | undefined>(undefined)
const cronFilterTriggerType = ref<string | undefined>(undefined)
const cronFilterDepartment = ref<string | undefined>(undefined)
const cronShowFilters = ref(false)
const cronPage = ref(1)
const cronPageSize = ref(10)
const cronTotal = ref(0)

const archiveSearch = ref('')
const archiveFilterProcessType = ref<string | undefined>(undefined)
const archiveFilterOperator = ref('')
const archiveFilterDepartment = ref<string | undefined>(undefined)
const archiveFilterDateRange = ref<[Dayjs, Dayjs] | undefined>(undefined)
const archiveShowFilters = ref(false)
const archivePage = ref(1)
const archivePageSize = ref(10)
const archiveTotal = ref(0)

const auditDetailVisible = ref(false)
const archiveDetailVisible = ref(false)
const cronDetailVisible = ref(false)
const selectedAuditLog = ref<AuditLogItem | null>(null)
const selectedArchiveLog = ref<ArchiveLogItem | null>(null)
const selectedCronLog = ref<CronLogItem | null>(null)
const auditChainLogs = ref<AuditLogItem[]>([])
const archiveChainLogs = ref<ArchiveLogItem[]>([])
const chainLoading = ref(false)

const recommendationConfig = computed<Record<string, { color: string; bg: string }>>(() => ({
  approve: { color: 'var(--color-success)', bg: 'var(--color-success-bg)' },
  return: { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)' },
  review: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)' },
}))

const complianceConfig = computed<Record<string, { color: string; bg: string }>>(() => ({
  compliant: { color: 'var(--color-success)', bg: 'var(--color-success-bg)' },
  non_compliant: { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)' },
  partially_compliant: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)' },
}))

const auditSubTabs = computed(() => [
  {
    key: 'all' as AuditSubTab,
    icon: AppstoreOutlined,
    count: auditStats.value.total,
    label: t('admin.data.auditTab.all'),
    cssClass: 'stat-card--info',
  },
  {
    key: 'approve' as AuditSubTab,
    icon: CheckCircleOutlined,
    count: auditStats.value.approve_count,
    label: t('admin.data.approved'),
    cssClass: 'stat-card--success',
  },
  {
    key: 'return' as AuditSubTab,
    icon: CloseCircleOutlined,
    count: auditStats.value.return_count,
    label: t('admin.data.returned'),
    cssClass: 'stat-card--danger',
  },
  {
    key: 'review' as AuditSubTab,
    icon: AlertOutlined,
    count: auditStats.value.review_count,
    label: t('admin.data.archived'),
    cssClass: 'stat-card--warning',
  },
])

const auditHasActiveFilters = computed(() =>
    !!auditSearch.value ||
    !!auditFilterProcessType.value ||
    !!auditFilterOperator.value ||
    !!auditFilterDepartment.value ||
    !!auditFilterDateRange.value)

const cronHasActiveFilters = computed(() =>
    !!cronFilterTaskType.value ||
    !!cronFilterTriggerType.value ||
    !!cronFilterDepartment.value)

const archiveHasActiveFilters = computed(() =>
    !!archiveSearch.value ||
    !!archiveFilterProcessType.value ||
    !!archiveFilterOperator.value ||
    !!archiveFilterDepartment.value ||
    !!archiveFilterDateRange.value)

const cronTaskTypeOptions = computed(() => {
  const seen = new Map<string, string>()
  for (const item of cronLogs.value) {
    if (item.task_type && !seen.has(item.task_type)) {
      seen.set(item.task_type, item.task_label || item.task_type)
    }
  }
  return Array.from(seen.entries()).map(([value, label]) => ({ value, label }))
})

const normalizedAuditDetail = computed<Required<AuditDetailPayload>>(() => {
  const current = selectedAuditLog.value
  const raw = normalizeObject<AuditDetailPayload>(current?.audit_result)

  return {
    recommendation: raw.recommendation || current?.recommendation || '',
    overall_score: Number(raw.overall_score ?? raw.score ?? current?.score ?? 0),
    score: Number(raw.score ?? raw.overall_score ?? current?.score ?? 0),
    rule_results: Array.isArray(raw.rule_results)
        ? raw.rule_results
        : Array.isArray(raw.details)
            ? raw.details
            : [],
    details: Array.isArray(raw.details)
        ? raw.details
        : Array.isArray(raw.rule_results)
            ? raw.rule_results
            : [],
    risk_points: Array.isArray(raw.risk_points) ? raw.risk_points : [],
    suggestions: Array.isArray(raw.suggestions) ? raw.suggestions : [],
    ai_reasoning: raw.ai_reasoning || current?.ai_reasoning || '',
    duration_ms: Number(raw.duration_ms ?? current?.duration_ms ?? 0),
  }
})

const normalizedArchiveDetail = computed<Required<ArchiveDetailPayload>>(() => {
  const current = selectedArchiveLog.value
  const raw = normalizeObject<ArchiveDetailPayload>(current?.archive_result)

  return {
    overall_compliance: raw.overall_compliance || current?.compliance || '',
    overall_score: Number(raw.overall_score ?? current?.compliance_score ?? 0),
    flow_audit: raw.flow_audit || { is_complete: false, missing_nodes: [], node_results: [] },
    rule_audit: Array.isArray(raw.rule_audit) ? raw.rule_audit : [],
    risk_points: Array.isArray(raw.risk_points) ? raw.risk_points : [],
    suggestions: Array.isArray(raw.suggestions) ? raw.suggestions : [],
    ai_summary: raw.ai_summary || current?.ai_reasoning || '',
    duration_ms: Number(raw.duration_ms ?? current?.duration_ms ?? 0),
  }
})

const auditQuery = computed(() => ({
  recommendation: activeAuditSubTab.value === 'all' ? '' : activeAuditSubTab.value,
  keyword: auditSearch.value.trim(),
  process_type: auditFilterProcessType.value || '',
  operator: auditFilterOperator.value.trim(),
  department: auditFilterDepartment.value || '',
  start_date: auditFilterDateRange.value?.[0]?.format('YYYY-MM-DD') || '',
  end_date: auditFilterDateRange.value?.[1]?.format('YYYY-MM-DD') || '',
  page: auditPage.value,
  page_size: auditPageSize.value,
}))

const cronQuery = computed(() => ({
  status: activeCronSubTab.value === 'all' ? '' : activeCronSubTab.value,
  task_type: cronFilterTaskType.value || '',
  trigger_type: cronFilterTriggerType.value || '',
  department: cronFilterDepartment.value || '',
  page: cronPage.value,
  page_size: cronPageSize.value,
}))

const archiveQuery = computed(() => ({
  compliance: activeArchiveSubTab.value === 'all' ? '' : activeArchiveSubTab.value,
  keyword: archiveSearch.value.trim(),
  process_type: archiveFilterProcessType.value || '',
  operator: archiveFilterOperator.value.trim(),
  department: archiveFilterDepartment.value || '',
  start_date: archiveFilterDateRange.value?.[0]?.format('YYYY-MM-DD') || '',
  end_date: archiveFilterDateRange.value?.[1]?.format('YYYY-MM-DD') || '',
  page: archivePage.value,
  page_size: archivePageSize.value,
}))

function normalizeObject<T>(value: unknown): T {
  if (!value) return {} as T
  if (typeof value === 'string') {
    try {
      return JSON.parse(value) as T
    } catch {
      return {} as T
    }
  }
  if (typeof value === 'object') return value as T
  return {} as T
}

function getRecLabel(rec: string) {
  const map: Record<string, string> = {
    approve: t('admin.data.auditApprove'),
    return: t('admin.data.auditReturn'),
    review: t('admin.data.auditReview'),
  }
  return map[rec] || rec || '-'
}

function getComplianceLabel(value: string) {
  const map: Record<string, string> = {
    compliant: t('admin.data.compliant'),
    non_compliant: t('admin.data.nonCompliant'),
    partially_compliant: t('admin.data.partiallyCompliant'),
  }
  return map[value] || value || '-'
}

function getTriggerTypeLabel(value: string) {
  const map: Record<string, string> = {
    manual: t('admin.data.triggerManual'),
    scheduled: t('admin.data.triggerScheduled'),
  }
  return map[value] || value || '-'
}

function getAsyncStatusLabel(status: string) {
  const map: Record<string, string> = {
    pending: t('admin.data.statusPending'),
    assembling: t('admin.data.statusAssembling'),
    reasoning: t('admin.data.statusReasoning'),
    extracting: t('admin.data.statusExtracting'),
    completed: t('admin.data.statusCompleted'),
    failed: t('admin.data.statusFailed'),
  }
  return map[status] || status || '-'
}

async function openAuditDetail(snapshot: AuditSnapshotItem) {
  auditDetailVisible.value = true
  chainLoading.value = true
  try {
    const res = await getAuditSnapshotChain(snapshot.process_id)
    auditChainLogs.value = res.chain || []
    selectedAuditLog.value = auditChainLogs.value[0] || null
  } catch {
    auditChainLogs.value = []
    selectedAuditLog.value = null
  } finally {
    chainLoading.value = false
  }
}

async function openArchiveDetail(snapshot: ArchiveSnapshotItem) {
  archiveDetailVisible.value = true
  chainLoading.value = true
  try {
    const res = await getArchiveSnapshotChain(snapshot.process_id)
    archiveChainLogs.value = res.chain || []
    selectedArchiveLog.value = archiveChainLogs.value[0] || null
  } catch {
    archiveChainLogs.value = []
    selectedArchiveLog.value = null
  } finally {
    chainLoading.value = false
  }
}

function openCronDetail(log: CronLogItem) {
  selectedCronLog.value = log
  cronDetailVisible.value = true
}

function clearAuditFilters() {
  auditSearch.value = ''
  auditFilterProcessType.value = undefined
  auditFilterOperator.value = ''
  auditFilterDepartment.value = undefined
  auditFilterDateRange.value = undefined
  auditPage.value = 1
}

function clearCronFilters() {
  cronFilterTaskType.value = undefined
  cronFilterTriggerType.value = undefined
  cronFilterDepartment.value = undefined
  cronPage.value = 1
}

function clearArchiveFilters() {
  archiveSearch.value = ''
  archiveFilterProcessType.value = undefined
  archiveFilterOperator.value = ''
  archiveFilterDepartment.value = undefined
  archiveFilterDateRange.value = undefined
  archivePage.value = 1
}

function handleAuditPageChange(page: number, pageSize: number) {
  auditPage.value = page
  auditPageSize.value = pageSize
}

function handleCronPageChange(page: number, pageSize: number) {
  cronPage.value = page
  cronPageSize.value = pageSize
}

function handleArchivePageChange(page: number, pageSize: number) {
  archivePage.value = page
  archivePageSize.value = pageSize
}

async function loadAuditProcessTypeOptions() {
  try {
    const list = await getAuditProcessTypes()
    auditProcessOptions.value = (Array.isArray(list) ? list : []).map(item => ({
      value: item.process_type,
      label: item.process_type_label || item.process_type,
    }))
  } catch {
    auditProcessOptions.value = []
  }
}

async function loadArchiveProcessTypeOptions() {
  try {
    const list = await getArchiveProcessTypes()
    archiveProcessOptions.value = (Array.isArray(list) ? list : []).map(item => ({
      value: item.process_type,
      label: item.process_type_label || item.process_type,
    }))
  } catch {
    archiveProcessOptions.value = []
  }
}

async function loadAuditStats() {
  try {
    auditStats.value = await getAuditSnapshotStats()
  } catch (e: any) {
    message.error(e?.message || t('admin.data.loadFailed'))
  }
}

async function loadCronStats() {
  try {
    cronStats.value = await getCronLogStats()
  } catch (e: any) {
    message.error(e?.message || t('admin.data.loadFailed'))
  }
}

async function loadArchiveStats() {
  try {
    archiveStats.value = await getArchiveSnapshotStats()
  } catch (e: any) {
    message.error(e?.message || t('admin.data.loadFailed'))
  }
}

async function loadAuditLogs() {
  auditLoading.value = true
  try {
    const res = await listAuditSnapshots(auditQuery.value)
    auditSnapshots.value = res.items || []
    auditTotal.value = res.total || 0
  } catch (e: any) {
    auditSnapshots.value = []
    auditTotal.value = 0
    message.error(e?.message || t('admin.data.loadFailed'))
  } finally {
    auditLoading.value = false
  }
}

async function loadCronLogs() {
  cronLoading.value = true
  try {
    const res = await listCronLogs(cronQuery.value)
    cronLogs.value = res.items || []
    cronTotal.value = res.total || 0
  } catch (e: any) {
    cronLogs.value = []
    cronTotal.value = 0
    message.error(e?.message || t('admin.data.loadFailed'))
  } finally {
    cronLoading.value = false
  }
}

async function loadArchiveLogs() {
  archiveLoading.value = true
  try {
    const res = await listArchiveSnapshots(archiveQuery.value)
    archiveSnapshots.value = res.items || []
    archiveTotal.value = res.total || 0
  } catch (e: any) {
    archiveSnapshots.value = []
    archiveTotal.value = 0
    message.error(e?.message || t('admin.data.loadFailed'))
  } finally {
    archiveLoading.value = false
  }
}

async function handleExport(type: MainTab) {
  const hide = message.loading(
      type === 'audit'
          ? t('admin.data.exportingAudit')
          : type === 'cron'
              ? t('admin.data.exportingCron')
              : t('admin.data.exportingArchive'),
      0,
  )

  try {
    if (type === 'audit') {
      const { page, page_size, ...filters } = auditQuery.value
      await exportAuditLogs(filters)
    } else if (type === 'cron') {
      const { page, page_size, ...filters } = cronQuery.value
      await exportCronLogs(filters)
    } else {
      const { page, page_size, ...filters } = archiveQuery.value
      await exportArchiveLogs(filters)
    }
    hide()
    message.success(t('admin.data.exportSuccess'))
  } catch (e: any) {
    hide()
    message.error(e?.message || t('admin.data.exportFailed'))
  }
}

watch(auditQuery, loadAuditLogs, { immediate: true })
watch(cronQuery, loadCronLogs, { immediate: true })
watch(archiveQuery, loadArchiveLogs, { immediate: true })

watch(activeAuditSubTab, () => {
  auditPage.value = 1
})

onMounted(async () => {
  await Promise.all([
    loadAuditProcessTypeOptions(),
    loadArchiveProcessTypeOptions(),
    loadAuditStats(),
    loadCronStats(),
    loadArchiveStats(),
  ])
})
</script>

<template>
  <div class="data-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('admin.data.title') }}</h1>
        <p class="page-subtitle">{{ t('admin.data.subtitle') }}</p>
      </div>
    </div>

    <div class="tab-nav">
      <button
          v-for="tab in [
          { key: 'audit', label: t('admin.data.tabAudit'), icon: AppstoreOutlined },
          { key: 'cron', label: t('admin.data.tabCron'), icon: ClockCircleOutlined },
          { key: 'archive', label: t('admin.data.tabArchive'), icon: FolderOpenOutlined },
        ]"
          :key="tab.key"
          class="tab-btn"
          :class="{ 'tab-btn--active': activeTab === tab.key }"
          @click="activeTab = tab.key as MainTab"
      >
        <component :is="tab.icon" style="font-size: 14px;" />
        {{ tab.label }}
      </button>
    </div>

    <div v-if="activeTab === 'audit'" class="tab-content fade-in">
      <div class="subtab-nav">
        <button
            v-for="tab in auditSubTabs"
            :key="tab.key"
            class="subtab-btn"
            :class="[tab.cssClass, { 'subtab-btn--active': activeAuditSubTab === tab.key }]"
            @click="activeAuditSubTab = tab.key"
        >
          <component :is="tab.icon" />
          <span>{{ tab.label }}</span>
          <span class="subtab-badge">{{ tab.count }}</span>
        </button>
      </div>

      <div class="toolbar">
        <div class="toolbar-left">
          <a-button
              size="small"
              @click="auditShowFilters = !auditShowFilters"
              :class="{ 'filter-toggle-btn--active': auditHasActiveFilters }"
          >
            <FilterOutlined /> {{ t('admin.data.filter') }}
            <span v-if="auditHasActiveFilters" class="filter-active-dot" />
          </a-button>
        </div>
        <div class="toolbar-right">
          <a-button @click="handleExport('audit')">
            <ExportOutlined /> {{ t('admin.data.export') }}
          </a-button>
        </div>
      </div>

      <transition name="slide">
        <div v-if="auditShowFilters" class="filter-bar">
          <a-input
              v-model:value="auditSearch"
              :placeholder="t('admin.data.searchAudit')"
              allow-clear
              style="flex: 2; min-width: 180px;"
              @update:value="auditPage = 1"
          >
            <template #prefix>
              <SearchOutlined style="color: var(--color-text-tertiary);" />
            </template>
          </a-input>

          <a-input
              v-model:value="auditFilterOperator"
              :placeholder="t('admin.data.filterOperator')"
              allow-clear
              style="flex: 1; min-width: 140px;"
              @update:value="auditPage = 1"
          >
            <template #prefix>
              <SearchOutlined style="color: var(--color-text-tertiary);" />
            </template>
          </a-input>

          <a-select
              v-model:value="auditFilterProcessType"
              :placeholder="t('admin.data.filterProcessType')"
              allow-clear
              style="flex: 1; min-width: 160px;"
              :options="auditProcessOptions"
              @change="auditPage = 1"
          />

          <a-select
              v-model:value="auditFilterDepartment"
              :placeholder="t('admin.data.filterDepartment')"
              allow-clear
              style="flex: 1; min-width: 140px;"
              @change="auditPage = 1"
          >
          </a-select>

          <a-range-picker
              v-model:value="auditFilterDateRange"
              :placeholder="[t('admin.data.filterDateRange'), t('admin.data.filterDateRange')]"
              allow-clear
              style="flex: 1; min-width: 220px;"
              @change="auditPage = 1"
          />

          <a-button size="small" @click="clearAuditFilters">
            {{ t('admin.data.filterReset') }}
          </a-button>
        </div>
      </transition>

      <div class="data-table-card">
        <table class="data-table">
          <thead>
          <tr>
            <th>{{ t('admin.data.thProcessId') }}</th>
            <th>{{ t('admin.data.thProcessTitle') }}</th>
            <th>{{ t('admin.data.thOperator') }}</th>
            <th>{{ t('admin.data.thDepartment') }}</th>
            <th>{{ t('admin.data.thProcessType') }}</th>
            <th>{{ t('admin.data.thResult') }}</th>
            <th>{{ t('admin.data.thTime') }}</th>
            <th>{{ t('admin.data.thAction') }}</th>
          </tr>
          </thead>
          <tbody>
          <tr v-if="auditLoading">
            <td colspan="8" class="empty-cell">{{ t('admin.data.loading') }}</td>
          </tr>
          <tr v-else v-for="item in auditSnapshots" :key="item.id">
            <td class="text-mono">{{ item.process_id }}</td>
            <td>{{ item.title }}</td>
            <td>{{ item.operator || '-' }}</td>
            <td>{{ item.department || '-' }}</td>
            <td class="text-secondary">{{ item.process_type }}</td>
            <td>
                <span
                    v-if="item.recommendation"
                    class="result-tag"
                    :style="{
                    color: recommendationConfig[item.recommendation]?.color,
                    background: recommendationConfig[item.recommendation]?.bg,
                  }"
                >
                  <CheckCircleOutlined v-if="item.recommendation === 'approve'" />
                  <CloseCircleOutlined v-else-if="item.recommendation === 'return'" />
                  <AlertOutlined v-else />
                  {{ getRecLabel(item.recommendation) }} {{ item.score }}{{ t('admin.data.points') }}
                </span>
              <span v-else class="text-secondary">-</span>
            </td>
            <td class="text-secondary">{{ item.updated_at_fmt }}</td>
            <td>
              <div class="action-btns">
                <button
                    class="icon-btn"
                    :title="t('admin.data.viewDetail')"
                    @click="openAuditDetail(item)"
                >
                  <EyeOutlined />
                </button>
              </div>
            </td>
          </tr>
          <tr v-if="!auditLoading && auditSnapshots.length === 0">
            <td colspan="8" class="empty-cell">{{ t('admin.data.noData') }}</td>
          </tr>
          </tbody>
        </table>
      </div>

      <div class="pagination-wrapper">
        <a-pagination
            :current="auditPage"
            :page-size="auditPageSize"
            :total="auditTotal"
            size="small"
            show-size-changer
            show-quick-jumper
            :page-size-options="['10', '20', '50']"
            @change="handleAuditPageChange"
            @showSizeChange="handleAuditPageChange"
        />
      </div>
    </div>

    <div v-if="activeTab === 'cron'" class="tab-content fade-in">
      <div class="subtab-nav">
        <button
            v-for="tab in [
            { key: 'all', icon: AppstoreOutlined, count: cronStats.total, label: t('admin.data.auditTab.all'), cssClass: 'stat-card--info' },
            { key: 'success', icon: CheckCircleOutlined, count: cronStats.success, label: t('admin.data.success'), cssClass: 'stat-card--success' },
            { key: 'failed', icon: CloseCircleOutlined, count: cronStats.failed, label: t('admin.data.failed'), cssClass: 'stat-card--danger' },
            { key: 'running', icon: SyncOutlined, count: cronStats.running, label: t('admin.data.running'), cssClass: 'stat-card--warning' },
          ]"
            :key="tab.key"
            class="subtab-btn"
            :class="[tab.cssClass, { 'subtab-btn--active': activeCronSubTab === tab.key }]"
            @click="activeCronSubTab = tab.key as CronSubTab; cronPage = 1"
        >
          <component :is="tab.icon" />
          <span>{{ tab.label }}</span>
          <span class="subtab-badge">{{ tab.count }}</span>
        </button>
      </div>

      <div class="toolbar">
        <div class="toolbar-left">
          <a-button
              size="small"
              @click="cronShowFilters = !cronShowFilters"
              :class="{ 'filter-toggle-btn--active': cronHasActiveFilters }"
          >
            <FilterOutlined /> {{ t('admin.data.filter') }}
            <span v-if="cronHasActiveFilters" class="filter-active-dot" />
          </a-button>
        </div>
        <div class="toolbar-right">
          <a-button @click="handleExport('cron')">
            <ExportOutlined /> {{ t('admin.data.export') }}
          </a-button>
        </div>
      </div>

      <transition name="slide">
        <div v-if="cronShowFilters" class="filter-bar">
          <a-select
              v-model:value="cronFilterTaskType"
              :placeholder="t('admin.data.thTaskType')"
              allow-clear
              style="flex: 1; min-width: 140px;"
              :options="cronTaskTypeOptions"
              @change="cronPage = 1"
          />

          <a-select
              v-model:value="cronFilterTriggerType"
              :placeholder="t('admin.data.filterTriggerType')"
              allow-clear
              style="flex: 1; min-width: 140px;"
              @change="cronPage = 1"
          >
            <a-select-option value="manual">{{ t('admin.data.triggerManual') }}</a-select-option>
            <a-select-option value="scheduled">{{ t('admin.data.triggerScheduled') }}</a-select-option>
          </a-select>

          <a-select
              v-model:value="cronFilterDepartment"
              :placeholder="t('admin.data.filterDepartment')"
              allow-clear
              style="flex: 1; min-width: 140px;"
              @change="cronPage = 1"
          >
          </a-select>

          <a-button size="small" @click="clearCronFilters">
            {{ t('admin.data.filterReset') }}
          </a-button>
        </div>
      </transition>

      <div class="data-table-card">
        <table class="data-table">
          <thead>
          <tr>
            <th>{{ t('admin.data.thTaskName') }}</th>
            <th>{{ t('admin.data.thTaskType') }}</th>
            <th>{{ t('admin.data.thTriggerType') }}</th>
            <th>{{ t('admin.data.thCreatedBy') }}</th>
            <th>{{ t('admin.data.thTaskOwner') }}</th>
            <th>{{ t('admin.data.thDepartment') }}</th>
            <th>{{ t('admin.data.thStatus') }}</th>
            <th>{{ t('admin.data.thTime') }}</th>
            <th>{{ t('admin.data.thAction') }}</th>
          </tr>
          </thead>
          <tbody>
          <tr v-if="cronLoading">
            <td colspan="9" class="empty-cell">{{ t('admin.data.loading') }}</td>
          </tr>
          <tr v-else v-for="item in cronLogs" :key="item.id">
            <td>{{ item.task_label }}</td>
            <td class="text-secondary text-mono">{{ item.task_type }}</td>
            <td>{{ getTriggerTypeLabel(item.trigger_type) }}</td>
            <td>{{ item.created_by || '-' }}</td>
            <td>{{ item.task_owner_display_name || '-' }}</td>
            <td>{{ item.department || '-' }}</td>
            <td>
                <span
                    class="status-tag"
                    :class="`status-tag--${
                    item.status === 'success'
                      ? 'success'
                      : item.status === 'failed'
                        ? 'failed'
                        : 'running'
                  }`"
                >
                  <CheckCircleOutlined v-if="item.status === 'success'" />
                  <CloseCircleOutlined v-else-if="item.status === 'failed'" />
                  <SyncOutlined v-else spin />
                  {{
                    item.status === 'success'
                        ? t('admin.data.success')
                        : item.status === 'failed'
                            ? t('admin.data.failed')
                            : t('admin.data.running')
                  }}
                </span>
            </td>
            <td class="text-secondary">{{ item.started_at }}</td>
            <td>
              <div class="action-btns">
                <button
                    class="icon-btn"
                    :title="t('admin.data.viewDetail')"
                    @click="openCronDetail(item)"
                >
                  <EyeOutlined />
                </button>
              </div>
            </td>
          </tr>
          <tr v-if="!cronLoading && cronLogs.length === 0">
            <td colspan="9" class="empty-cell">{{ t('admin.data.noData') }}</td>
          </tr>
          </tbody>
        </table>
      </div>

      <div class="pagination-wrapper">
        <a-pagination
            :current="cronPage"
            :page-size="cronPageSize"
            :total="cronTotal"
            size="small"
            show-size-changer
            show-quick-jumper
            :page-size-options="['10', '20', '50']"
            @change="handleCronPageChange"
            @showSizeChange="handleCronPageChange"
        />
      </div>
    </div>

    <div v-if="activeTab === 'archive'" class="tab-content fade-in">
      <div class="subtab-nav">
        <button
            v-for="tab in [
            { key: 'all', icon: AppstoreOutlined, count: archiveStats.total, label: t('admin.data.auditTab.all'), cssClass: 'stat-card--info' },
            { key: 'compliant', icon: SafetyCertificateOutlined, count: archiveStats.compliant, label: t('admin.data.compliant'), cssClass: 'stat-card--success' },
            { key: 'partially_compliant', icon: AlertOutlined, count: archiveStats.partial, label: t('admin.data.partiallyCompliant'), cssClass: 'stat-card--warning' },
            { key: 'non_compliant', icon: CloseCircleOutlined, count: archiveStats.non_compliant, label: t('admin.data.nonCompliant'), cssClass: 'stat-card--danger' },
          ]"
            :key="tab.key"
            class="subtab-btn"
            :class="[tab.cssClass, { 'subtab-btn--active': activeArchiveSubTab === tab.key }]"
            @click="activeArchiveSubTab = tab.key as ArchiveSubTab; archivePage = 1"
        >
          <component :is="tab.icon" />
          <span>{{ tab.label }}</span>
          <span class="subtab-badge">{{ tab.count }}</span>
        </button>
      </div>

      <div class="toolbar">
        <div class="toolbar-left">
          <a-button
              size="small"
              @click="archiveShowFilters = !archiveShowFilters"
              :class="{ 'filter-toggle-btn--active': archiveHasActiveFilters }"
          >
            <FilterOutlined /> {{ t('admin.data.filter') }}
            <span v-if="archiveHasActiveFilters" class="filter-active-dot" />
          </a-button>
        </div>
        <div class="toolbar-right">
          <a-button @click="handleExport('archive')">
            <ExportOutlined /> {{ t('admin.data.export') }}
          </a-button>
        </div>
      </div>

      <transition name="slide">
        <div v-if="archiveShowFilters" class="filter-bar">
          <a-input
              v-model:value="archiveSearch"
              :placeholder="t('admin.data.searchArchive')"
              allow-clear
              style="flex: 2; min-width: 180px;"
              @update:value="archivePage = 1"
          >
            <template #prefix>
              <SearchOutlined style="color: var(--color-text-tertiary);" />
            </template>
          </a-input>

          <a-input
              v-model:value="archiveFilterOperator"
              :placeholder="t('admin.data.filterOperator')"
              allow-clear
              style="flex: 1; min-width: 140px;"
              @update:value="archivePage = 1"
          >
            <template #prefix>
              <SearchOutlined style="color: var(--color-text-tertiary);" />
            </template>
          </a-input>

          <a-select
              v-model:value="archiveFilterProcessType"
              :placeholder="t('admin.data.filterProcessType')"
              allow-clear
              style="flex: 1; min-width: 160px;"
              :options="archiveProcessOptions"
              @change="archivePage = 1"
          />

          <a-select
              v-model:value="archiveFilterDepartment"
              :placeholder="t('admin.data.filterDepartment')"
              allow-clear
              style="flex: 1; min-width: 140px;"
              @change="archivePage = 1"
          >
          </a-select>

          <a-range-picker
              v-model:value="archiveFilterDateRange"
              :placeholder="[t('admin.data.filterDateRange'), t('admin.data.filterDateRange')]"
              allow-clear
              style="flex: 1; min-width: 220px;"
              @change="archivePage = 1"
          />

          <a-button size="small" @click="clearArchiveFilters">
            {{ t('admin.data.filterReset') }}
          </a-button>
        </div>
      </transition>

      <div class="data-table-card">
        <table class="data-table">
          <thead>
          <tr>
            <th>{{ t('admin.data.thProcessId') }}</th>
            <th>{{ t('admin.data.thProcessTitle') }}</th>
            <th>{{ t('admin.data.thOperator') }}</th>
            <th>{{ t('admin.data.thDepartment') }}</th>
            <th>{{ t('admin.data.thProcessType') }}</th>
            <th>{{ t('admin.data.thCompliance') }}</th>
            <th>{{ t('admin.data.thTime') }}</th>
            <th>{{ t('admin.data.thAction') }}</th>
          </tr>
          </thead>
          <tbody>
          <tr v-if="archiveLoading">
            <td colspan="8" class="empty-cell">{{ t('admin.data.loading') }}</td>
          </tr>
          <tr v-else v-for="item in archiveSnapshots" :key="item.id">
            <td class="text-mono">{{ item.process_id }}</td>
            <td>{{ item.title }}</td>
            <td>{{ item.operator || '-' }}</td>
            <td>{{ item.department || '-' }}</td>
            <td class="text-secondary">{{ item.process_type }}</td>
            <td>
                <span
                    v-if="item.compliance"
                    class="result-tag"
                    :style="{
                    color: complianceConfig[item.compliance]?.color,
                    background: complianceConfig[item.compliance]?.bg,
                  }"
                >
                  <CheckCircleOutlined v-if="item.compliance === 'compliant'" />
                  <AlertOutlined v-else-if="item.compliance === 'partially_compliant'" />
                  <CloseCircleOutlined v-else />
                  {{ getComplianceLabel(item.compliance) }} {{ item.compliance_score }}{{ t('admin.data.points') }}
                </span>
              <span v-else class="text-secondary">-</span>
            </td>
            <td class="text-secondary">{{ item.updated_at_fmt }}</td>
            <td>
              <div class="action-btns">
                <button
                    class="icon-btn"
                    :title="t('admin.data.viewDetail')"
                    @click="openArchiveDetail(item)"
                >
                  <EyeOutlined />
                </button>
              </div>
            </td>
          </tr>
          <tr v-if="!archiveLoading && archiveSnapshots.length === 0">
            <td colspan="8" class="empty-cell">{{ t('admin.data.noData') }}</td>
          </tr>
          </tbody>
        </table>
      </div>

      <div class="pagination-wrapper">
        <a-pagination
            :current="archivePage"
            :page-size="archivePageSize"
            :total="archiveTotal"
            size="small"
            show-size-changer
            show-quick-jumper
            :page-size-options="['10', '20', '50']"
            @change="handleArchivePageChange"
            @showSizeChange="handleArchivePageChange"
        />
      </div>
    </div>

    <Teleport to="body">
      <transition name="drawer">
        <div v-if="auditDetailVisible" class="drawer-overlay" @click.self="auditDetailVisible = false">
          <div class="drawer-panel">
            <div class="drawer-header">
              <h3>{{ t('admin.data.detailTitle') }}</h3>
              <button class="drawer-close" @click="auditDetailVisible = false">
                <CloseOutlined />
              </button>
            </div>

            <div class="drawer-body" v-if="selectedAuditLog">
              <div class="detail-process-title">{{ selectedAuditLog.title }}</div>

              <div
                  class="detail-banner"
                  :style="{
                  background: recommendationConfig[normalizedAuditDetail.recommendation]?.bg || 'var(--color-bg-page)',
                  borderColor: recommendationConfig[normalizedAuditDetail.recommendation]?.color || 'var(--color-border-light)',
                }"
              >
                <CheckCircleOutlined
                    v-if="normalizedAuditDetail.recommendation === 'approve'"
                    :style="{ color: recommendationConfig[normalizedAuditDetail.recommendation]?.color, fontSize: '24px' }"
                />
                <CloseCircleOutlined
                    v-else-if="normalizedAuditDetail.recommendation === 'return'"
                    :style="{ color: recommendationConfig[normalizedAuditDetail.recommendation]?.color, fontSize: '24px' }"
                />
                <AlertOutlined
                    v-else
                    :style="{ color: recommendationConfig[normalizedAuditDetail.recommendation]?.color || 'var(--color-text-tertiary)', fontSize: '24px' }"
                />
                <div class="detail-banner-info">
                  <div
                      class="detail-banner-title"
                      :style="{ color: recommendationConfig[normalizedAuditDetail.recommendation]?.color || 'var(--color-text-primary)' }"
                  >
                    {{ getRecLabel(normalizedAuditDetail.recommendation) }}
                  </div>
                  <div class="detail-banner-meta">
                    {{ t('admin.data.overallScore') }} {{ normalizedAuditDetail.overall_score }}{{ t('admin.data.points') }}
                    ·
                    {{ t('admin.data.duration') }} {{ normalizedAuditDetail.duration_ms }}ms
                  </div>
                </div>
                <div
                    class="detail-score"
                    :style="{ color: recommendationConfig[normalizedAuditDetail.recommendation]?.color || 'var(--color-text-primary)' }"
                >
                  {{ normalizedAuditDetail.overall_score }}
                </div>
              </div>

              <div class="detail-section">
                <h4 class="detail-section-title">{{ t('admin.data.ruleCheckDetail') }}</h4>
                <div class="rule-checks">
                  <div
                      v-for="(rule, index) in normalizedAuditDetail.rule_results"
                      :key="rule.rule_id || rule.rule_name || rule.rule_content || index"
                      class="rule-check-item"
                      :class="{ 'rule-check-item--pass': !!rule.passed, 'rule-check-item--fail': rule.passed === false }"
                  >
                    <div class="rule-check-status">
                      <CheckCircleOutlined v-if="rule.passed" style="color: var(--color-success);" />
                      <CloseCircleOutlined v-else style="color: var(--color-danger);" />
                    </div>
                    <div class="rule-check-content">
                      <div class="rule-check-name">{{ rule.rule_name || rule.rule_content || '-' }}</div>
                      <div class="rule-check-reasoning">{{ rule.reasoning || rule.reason || '-' }}</div>
                    </div>
                  </div>
                  <div v-if="normalizedAuditDetail.rule_results.length === 0" class="empty-state-inline">
                    {{ t('admin.data.noData') }}
                  </div>
                </div>
              </div>

              <div
                  v-if="normalizedAuditDetail.risk_points.length || normalizedAuditDetail.suggestions.length"
                  class="risk-suggest-row"
              >
                <div v-if="normalizedAuditDetail.risk_points.length" class="insight-card insight-card--risk">
                  <div class="insight-card-header">
                    <CloseCircleOutlined style="color: var(--color-danger);" />
                    <span>{{ t('admin.data.riskPoints') }}</span>
                  </div>
                  <ul class="insight-card-list">
                    <li v-for="(rp, i) in normalizedAuditDetail.risk_points" :key="i">{{ rp }}</li>
                  </ul>
                </div>

                <div v-if="normalizedAuditDetail.suggestions.length" class="insight-card insight-card--suggest">
                  <div class="insight-card-header">
                    <InfoCircleOutlined style="color: var(--color-primary);" />
                    <span>{{ t('admin.data.suggestions') }}</span>
                  </div>
                  <ul class="insight-card-list">
                    <li v-for="(sg, i) in normalizedAuditDetail.suggestions" :key="i">{{ sg }}</li>
                  </ul>
                </div>
              </div>

              <div class="detail-section">
                <h4 class="detail-section-title">{{ t('admin.data.aiReasoning') }}</h4>
                <div class="ai-reasoning">
                  <pre>{{ normalizedAuditDetail.ai_reasoning || '-' }}</pre>
                </div>
              </div>
            </div>
          </div>
        </div>
      </transition>
    </Teleport>

    <Teleport to="body">
      <transition name="drawer">
        <div v-if="archiveDetailVisible" class="drawer-overlay" @click.self="archiveDetailVisible = false">
          <div class="drawer-panel">
            <div class="drawer-header">
              <h3>{{ t('admin.data.archiveDetailTitle') }}</h3>
              <button class="drawer-close" @click="archiveDetailVisible = false">
                <CloseOutlined />
              </button>
            </div>

            <div class="drawer-body" v-if="selectedArchiveLog">
              <div class="detail-process-title">{{ selectedArchiveLog.title }}</div>

              <div
                  class="detail-banner"
                  :style="{
                  background: complianceConfig[normalizedArchiveDetail.overall_compliance]?.bg || 'var(--color-bg-page)',
                  borderColor: complianceConfig[normalizedArchiveDetail.overall_compliance]?.color || 'var(--color-border-light)',
                }"
              >
                <SafetyCertificateOutlined
                    :style="{ color: complianceConfig[normalizedArchiveDetail.overall_compliance]?.color || 'var(--color-text-primary)', fontSize: '24px' }"
                />
                <div class="detail-banner-info">
                  <div
                      class="detail-banner-title"
                      :style="{ color: complianceConfig[normalizedArchiveDetail.overall_compliance]?.color || 'var(--color-text-primary)' }"
                  >
                    {{ getComplianceLabel(normalizedArchiveDetail.overall_compliance) }}
                  </div>
                  <div class="detail-banner-meta">
                    {{ t('admin.data.overallScore') }} {{ normalizedArchiveDetail.overall_score }}{{ t('admin.data.points') }}
                    ·
                    {{ t('admin.data.duration') }} {{ normalizedArchiveDetail.duration_ms }}ms
                  </div>
                </div>
                <div
                    class="detail-score"
                    :style="{ color: complianceConfig[normalizedArchiveDetail.overall_compliance]?.color || 'var(--color-text-primary)' }"
                >
                  {{ normalizedArchiveDetail.overall_score }}
                </div>
              </div>

              <div class="detail-section">
                <h4 class="detail-section-title">{{ t('admin.data.flowAudit') }}</h4>
                <div
                    class="flow-status"
                    :class="normalizedArchiveDetail.flow_audit?.is_complete ? 'flow-status--complete' : 'flow-status--incomplete'"
                >
                  <CheckCircleOutlined v-if="normalizedArchiveDetail.flow_audit?.is_complete" style="color: var(--color-success);" />
                  <CloseCircleOutlined v-else style="color: var(--color-danger);" />
                  {{
                    normalizedArchiveDetail.flow_audit?.is_complete
                        ? t('admin.data.flowComplete')
                        : t('admin.data.flowIncomplete')
                  }}
                  <span v-if="normalizedArchiveDetail.flow_audit?.missing_nodes?.length" class="flow-missing">
                    · {{ t('admin.data.missingNodes') }}:
                    {{ normalizedArchiveDetail.flow_audit?.missing_nodes?.join(', ') }}
                  </span>
                </div>

                <div class="rule-checks">
                  <div
                      v-for="(node, index) in normalizedArchiveDetail.flow_audit?.node_results || []"
                      :key="node.node_id || node.node_name || index"
                      class="rule-check-item"
                      :class="{ 'rule-check-item--pass': !!node.compliant, 'rule-check-item--fail': node.compliant === false }"
                  >
                    <div class="rule-check-status">
                      <CheckCircleOutlined v-if="node.compliant" style="color: var(--color-success);" />
                      <CloseCircleOutlined v-else style="color: var(--color-danger);" />
                    </div>
                    <div class="rule-check-content">
                      <div class="rule-check-name">{{ node.node_name || '-' }}</div>
                      <div class="rule-check-reasoning">{{ node.reasoning || '-' }}</div>
                    </div>
                  </div>
                </div>
              </div>

              <div class="detail-section">
                <h4 class="detail-section-title">{{ t('admin.data.ruleAudit') }}</h4>
                <div class="rule-checks">
                  <div
                      v-for="(rule, index) in normalizedArchiveDetail.rule_audit"
                      :key="rule.rule_id || rule.rule_name || index"
                      class="rule-check-item"
                      :class="{ 'rule-check-item--pass': !!rule.passed, 'rule-check-item--fail': rule.passed === false }"
                  >
                    <div class="rule-check-status">
                      <CheckCircleOutlined v-if="rule.passed" style="color: var(--color-success);" />
                      <CloseCircleOutlined v-else style="color: var(--color-danger);" />
                    </div>
                    <div class="rule-check-content">
                      <div class="rule-check-name">{{ rule.rule_name || '-' }}</div>
                      <div class="rule-check-reasoning">{{ rule.reasoning || '-' }}</div>
                    </div>
                  </div>
                  <div v-if="normalizedArchiveDetail.rule_audit.length === 0" class="empty-state-inline">
                    {{ t('admin.data.noData') }}
                  </div>
                </div>
              </div>

              <div class="detail-section">
                <h4 class="detail-section-title">{{ t('admin.data.aiSummary') }}</h4>
                <div class="ai-reasoning">
                  <pre>{{ normalizedArchiveDetail.ai_summary || '-' }}</pre>
                </div>
              </div>
            </div>
          </div>
        </div>
      </transition>
    </Teleport>

    <Teleport to="body">
      <transition name="drawer">
        <div v-if="cronDetailVisible" class="drawer-overlay" @click.self="cronDetailVisible = false">
          <div class="drawer-panel">
            <div class="drawer-header">
              <h3>{{ t('admin.data.cronDetailTitle') }}</h3>
              <button class="drawer-close" @click="cronDetailVisible = false">
                <CloseOutlined />
              </button>
            </div>

            <div class="drawer-body" v-if="selectedCronLog">
              <div class="detail-process-title">{{ selectedCronLog.task_label }}</div>

              <div class="detail-meta-grid">
                <div class="detail-meta-item">
                  <span class="detail-meta-label">{{ t('admin.data.thTaskType') }}</span>
                  <span class="detail-meta-value text-mono">{{ selectedCronLog.task_type }}</span>
                </div>
                <div class="detail-meta-item">
                  <span class="detail-meta-label">{{ t('admin.data.thTriggerType') }}</span>
                  <span class="detail-meta-value">{{ getTriggerTypeLabel(selectedCronLog.trigger_type) }}</span>
                </div>
                <div class="detail-meta-item">
                  <span class="detail-meta-label">{{ t('admin.data.cronExecStatus') }}</span>
                  <span class="detail-meta-value">
                    <span
                        class="status-tag"
                        :class="`status-tag--${
                        selectedCronLog.status === 'success'
                          ? 'success'
                          : selectedCronLog.status === 'failed'
                            ? 'failed'
                            : 'running'
                      }`"
                    >
                      <CheckCircleOutlined v-if="selectedCronLog.status === 'success'" />
                      <CloseCircleOutlined v-else-if="selectedCronLog.status === 'failed'" />
                      <SyncOutlined v-else spin />
                      {{
                        selectedCronLog.status === 'success'
                            ? t('admin.data.success')
                            : selectedCronLog.status === 'failed'
                                ? t('admin.data.failed')
                                : t('admin.data.running')
                      }}
                    </span>
                  </span>
                </div>
                <div class="detail-meta-item">
                  <span class="detail-meta-label">{{ t('admin.data.thCreatedBy') }}</span>
                  <span class="detail-meta-value">{{ selectedCronLog.created_by || '-' }}</span>
                </div>
                <div class="detail-meta-item">
                  <span class="detail-meta-label">{{ t('admin.data.cronOwner') }}</span>
                  <span class="detail-meta-value">{{ selectedCronLog.task_owner_display_name || '-' }}</span>
                </div>
                <div class="detail-meta-item">
                  <span class="detail-meta-label">{{ t('admin.data.thDepartment') }}</span>
                  <span class="detail-meta-value">{{ selectedCronLog.department || '-' }}</span>
                </div>
                <div class="detail-meta-item">
                  <span class="detail-meta-label">{{ t('admin.data.cronStartTime') }}</span>
                  <span class="detail-meta-value">{{ selectedCronLog.started_at }}</span>
                </div>
                <div class="detail-meta-item">
                  <span class="detail-meta-label">{{ t('admin.data.cronEndTime') }}</span>
                  <span class="detail-meta-value">{{ selectedCronLog.finished_at || '-' }}</span>
                </div>
              </div>

              <div class="detail-section" v-if="selectedCronLog.message">
                <h4 class="detail-section-title">{{ t('admin.data.cronMessage') }}</h4>
                <div class="ai-reasoning">
                  <pre>{{ selectedCronLog.message }}</pre>
                </div>
              </div>
            </div>
          </div>
        </div>
      </transition>
    </Teleport>
  </div>
</template>

<style scoped>
.data-page { animation: fadeIn 0.3s ease-out; }
@keyframes fadeIn { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

.page-header { margin-bottom: 24px; }
.page-title { font-size: 24px; font-weight: 700; color: var(--color-text-primary); margin: 0; }
.page-subtitle { font-size: 14px; color: var(--color-text-tertiary); margin: 4px 0 0; }

.tab-nav {
  display: flex;
  gap: 4px;
  background: var(--color-bg-hover);
  padding: 4px;
  border-radius: var(--radius-lg);
  margin-bottom: 24px;
  width: fit-content;
}

.tab-btn {
  padding: 8px 20px;
  border: none;
  background: transparent;
  border-radius: var(--radius-md);
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
  display: flex;
  align-items: center;
  gap: 6px;
}

.tab-btn:hover { color: var(--color-text-primary); }
.tab-btn--active {
  background: var(--color-bg-card);
  color: var(--color-primary);
  box-shadow: var(--shadow-xs);
}

.subtab-nav {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
  margin-bottom: 20px;
}

.subtab-btn {
  border: 2px solid var(--color-border-light);
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  padding: 16px 18px;
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  transition: all var(--transition-base);
  color: var(--color-text-primary);
  font-weight: 600;
}

.subtab-btn:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.subtab-btn--active {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.subtab-btn.stat-card--primary.subtab-btn--active { border-color: var(--color-primary); }
.subtab-btn.stat-card--success.subtab-btn--active { border-color: var(--color-success); }
.subtab-btn.stat-card--info.subtab-btn--active { border-color: var(--color-info, var(--color-primary)); }

.subtab-badge {
  margin-left: auto;
  min-width: 28px;
  height: 28px;
  border-radius: 999px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0 8px;
  font-size: 12px;
  font-weight: 700;
  background: var(--color-bg-page);
  color: var(--color-text-primary);
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 20px;
}

.stat-card {
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  border: 2px solid var(--color-border-light);
  transition: all var(--transition-base);
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.stat-card-icon {
  width: 48px;
  height: 48px;
  border-radius: var(--radius-lg);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 22px;
  flex-shrink: 0;
}

.stat-card--primary .stat-card-icon { background: var(--color-primary-bg); color: var(--color-primary); }
.stat-card--success .stat-card-icon { background: var(--color-success-bg); color: var(--color-success); }
.stat-card--danger .stat-card-icon { background: var(--color-danger-bg); color: var(--color-danger); }
.stat-card--warning .stat-card-icon { background: var(--color-warning-bg); color: var(--color-warning); }
.stat-card--info .stat-card-icon {
  background: var(--color-info-bg, var(--color-primary-bg));
  color: var(--color-info, var(--color-primary));
}

.stat-card-info { display: flex; flex-direction: column; }
.stat-card-value {
  font-size: 28px;
  font-weight: 700;
  color: var(--color-text-primary);
  line-height: 1.2;
}
.stat-card-label {
  font-size: 13px;
  color: var(--color-text-tertiary);
  margin-top: 2px;
}

.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
  gap: 12px;
  flex-wrap: wrap;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.filter-bar {
  display: flex;
  gap: 8px;
  padding: 12px 16px;
  background: var(--color-bg-page);
  border-radius: var(--radius-md);
  margin-bottom: 12px;
  flex-wrap: wrap;
  align-items: center;
}

.filter-toggle-btn--active {
  border-color: var(--color-primary) !important;
  color: var(--color-primary) !important;
}

.filter-active-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--color-primary);
  display: inline-block;
  margin-left: 4px;
}

.data-table-card {
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light);
  overflow: hidden;
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 13px;
}

.data-table th {
  padding: 12px 16px;
  text-align: left;
  font-weight: 600;
  color: var(--color-text-secondary);
  background: var(--color-bg-page);
  border-bottom: 1px solid var(--color-border-light);
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  white-space: nowrap;
}

.data-table td {
  padding: 12px 16px;
  border-bottom: 1px solid var(--color-border-light);
  color: var(--color-text-primary);
}

.data-table tbody tr:hover { background: var(--color-bg-hover); }
.data-table tbody tr:last-child td { border-bottom: none; }

.text-secondary { color: var(--color-text-tertiary); }
.text-mono { font-family: monospace; font-size: 12px; color: var(--color-text-secondary); }

.empty-cell {
  text-align: center;
  padding: 32px 16px !important;
  color: var(--color-text-tertiary);
}

.empty-state-inline {
  text-align: center;
  padding: 12px 16px;
  border: 1px dashed var(--color-border-light);
  border-radius: var(--radius-md);
  color: var(--color-text-tertiary);
  background: var(--color-bg-page);
}

.result-tag {
  font-size: 11px;
  font-weight: 600;
  padding: 3px 10px;
  border-radius: var(--radius-full);
  white-space: nowrap;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.status-tag {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: var(--radius-full);
  display: inline-flex;
  align-items: center;
  gap: 4px;
}

.status-tag--success { background: var(--color-success-bg); color: var(--color-success); }
.status-tag--failed { background: var(--color-danger-bg); color: var(--color-danger); }
.status-tag--running { background: var(--color-primary-bg); color: var(--color-primary); }

.action-btns { display: flex; gap: 4px; }

.icon-btn {
  width: 28px;
  height: 28px;
  border: 1px solid var(--color-border);
  background: transparent;
  border-radius: var(--radius-sm);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-tertiary);
  transition: all var(--transition-fast);
}

.icon-btn:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.pagination-wrapper {
  padding: 16px 0;
  display: flex;
  justify-content: flex-end;
}

.drawer-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.45);
  z-index: 1000;
  display: flex;
  justify-content: flex-end;
}

.drawer-panel {
  width: 560px;
  max-width: 90vw;
  background: var(--color-bg-card);
  box-shadow: var(--shadow-xl);
  display: flex;
  flex-direction: column;
  height: 100%;
}

.drawer-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 24px;
  border-bottom: 1px solid var(--color-border-light);
}

.drawer-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.drawer-close {
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  border-radius: var(--radius-sm);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-tertiary);
  transition: all var(--transition-fast);
}

.drawer-close:hover {
  background: var(--color-bg-hover);
  color: var(--color-text-primary);
}

.drawer-body {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
}

.detail-process-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 16px;
}

.detail-banner {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 16px 20px;
  border-radius: var(--radius-lg);
  border: 1px solid;
  margin-bottom: 20px;
}

.detail-banner-info { flex: 1; }
.detail-banner-title { font-size: 16px; font-weight: 700; }
.detail-banner-meta {
  font-size: 12px;
  color: var(--color-text-tertiary);
  margin-top: 4px;
}
.detail-score {
  font-size: 36px;
  font-weight: 800;
  line-height: 1;
}

.detail-section { margin-bottom: 20px; }
.detail-section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 10px;
}

.rule-checks {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.rule-check-item {
  display: flex;
  gap: 10px;
  padding: 10px 14px;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border-light);
}

.rule-check-item--pass {
  background: var(--color-success-bg);
  border-color: rgba(16, 185, 129, 0.2);
}

.rule-check-item--fail {
  background: var(--color-danger-bg);
  border-color: rgba(239, 68, 68, 0.2);
}

.rule-check-status {
  font-size: 16px;
  flex-shrink: 0;
  padding-top: 1px;
}

.rule-check-content { flex: 1; }
.rule-check-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-primary);
}
.rule-check-reasoning {
  font-size: 12px;
  color: var(--color-text-secondary);
  margin-top: 2px;
}

.flow-status {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-radius: var(--radius-md);
  font-size: 13px;
  font-weight: 500;
  margin-bottom: 10px;
}

.flow-status--complete { background: var(--color-success-bg); color: var(--color-success); }
.flow-status--incomplete { background: var(--color-danger-bg); color: var(--color-danger); }
.flow-missing { font-weight: 400; }

.risk-suggest-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-bottom: 20px;
}

.insight-card {
  padding: 14px;
  border-radius: var(--radius-md);
}

.insight-card--risk { background: var(--color-danger-bg); }
.insight-card--suggest { background: var(--color-primary-bg); }

.insight-card-header {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 8px;
}

.insight-card-list {
  margin: 0;
  padding-left: 18px;
  font-size: 12px;
  color: var(--color-text-secondary);
}

.insight-card-list li { margin-bottom: 4px; }

.ai-reasoning {
  background: var(--color-bg-page);
  border-radius: var(--radius-md);
  padding: 14px;
  border: 1px solid var(--color-border-light);
}

.ai-reasoning pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 13px;
  line-height: 1.6;
  color: var(--color-text-secondary);
  font-family: var(--font-sans);
}

.slide-enter-active,
.slide-leave-active { transition: all 0.2s ease; }

.slide-enter-from,
.slide-leave-to {
  opacity: 0;
  max-height: 0;
  overflow: hidden;
  margin-bottom: 0;
  padding-top: 0;
  padding-bottom: 0;
}

.slide-enter-to,
.slide-leave-from { opacity: 1; max-height: 240px; }

.drawer-enter-active,
.drawer-leave-active { transition: opacity 0.3s ease; }

.drawer-enter-active .drawer-panel,
.drawer-leave-active .drawer-panel { transition: transform 0.3s ease; }

.drawer-enter-from { opacity: 0; }
.drawer-enter-from .drawer-panel { transform: translateX(100%); }
.drawer-leave-to { opacity: 0; }
.drawer-leave-to .drawer-panel { transform: translateX(100%); }

.fade-in { animation: fadeIn 0.3s ease-out; }

.detail-meta-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
  margin-bottom: 24px;
  padding: 16px;
  background: var(--color-bg-page);
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light);
}

.detail-meta-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.detail-meta-label {
  font-size: 12px;
  color: var(--color-text-tertiary);
  font-weight: 500;
}

.detail-meta-value {
  font-size: 14px;
  color: var(--color-text-primary);
}

@media (max-width: 768px) {
  .stats-row { grid-template-columns: repeat(2, 1fr); }
  .subtab-nav { grid-template-columns: repeat(2, 1fr); }
  .data-table-card { overflow-x: auto; -webkit-overflow-scrolling: touch; }
  .data-table { min-width: 760px; }
  .toolbar { flex-direction: column; align-items: stretch; }
  .filter-bar { flex-direction: column; }
  .page-title { font-size: 20px; }
  .tab-nav { width: 100%; overflow-x: auto; -webkit-overflow-scrolling: touch; }
  .tab-btn { flex-shrink: 0; padding: 8px 14px; font-size: 13px; }
  .risk-suggest-row { grid-template-columns: 1fr; }
  .drawer-panel { width: 100%; max-width: 100vw; }
  .detail-meta-grid { grid-template-columns: 1fr; }
}
</style>