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
import dayjs, { type Dayjs } from 'dayjs'
import 'dayjs/locale/zh-cn'
import { marked } from 'marked'
import { ref, computed, watch, onMounted } from 'vue'
import { useI18n } from '~/composables/useI18n'
import { message } from 'ant-design-vue'
import { useAdminDataApi } from '~/composables/useAdminDataApi'
import { useAuditApi } from '~/composables/useAuditApi'
import { useOrgApi } from '~/composables/useOrgApi'
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

const activeTab = ref<MainTab>('audit')
const activeAuditSubTab = ref<AuditSubTab>('all')
const activeCronSubTab = ref<CronSubTab>('all')
const activeArchiveSubTab = ref<ArchiveSubTab>('all')

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
const archiveStats = ref<ArchiveSnapshotStats>({ total: 0, compliant: 0, partial: 0, non_compliant: 0 })

const auditSnapshots = ref<AuditSnapshotItem[]>([])
const cronLogs = ref<CronLogItem[]>([])
const archiveSnapshots = ref<ArchiveSnapshotItem[]>([])

// 抽屉详情相关变量
const auditDetailVisible = ref(false)
const selectedAuditLog = ref<AuditSnapshotItem | null>(null)
const auditChainLogs = ref<AuditLogItem[]>([])
const expandedAuditChainNodes = ref<Set<string>>(new Set())

const archiveDetailVisible = ref(false)
const selectedArchiveLog = ref<ArchiveSnapshotItem | null>(null)
const archiveChainLogs = ref<ArchiveLogItem[]>([])
const expandedArchiveChainNodes = ref<Set<string>>(new Set())

const cronDetailVisible = ref(false)
const selectedCronLog = ref<CronLogItem | null>(null)

const chainLoading = ref(false)

const { listDepartments } = useOrgApi()
const { getProcessTypes } = useAuditApi()

const auditLoading = ref(false)
const cronLoading = ref(false)
const archiveLoading = ref(false)

const departmentOptions = ref<{label: string, value: string}[]>([])
const processCascaderOptions = ref<any[]>([])

const auditSearch = ref('')
const auditFilterProcessPath = ref<string[][]>([])
const auditFilterProcessType = computed(() => auditFilterProcessPath.value.length ? auditFilterProcessPath.value.map((p: any[]) => p[p.length - 1]).join(',') : undefined)
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
const archiveFilterProcessPath = ref<string[][]>([])
const archiveFilterProcessType = computed(() => archiveFilterProcessPath.value.length ? archiveFilterProcessPath.value.map((p: any[]) => p[p.length - 1]).join(',') : undefined)
const archiveFilterOperator = ref('')
const archiveFilterDepartment = ref<string | undefined>(undefined)
const archiveFilterDateRange = ref<[Dayjs, Dayjs] | undefined>(undefined)
const archiveShowFilters = ref(false)
const archivePage = ref(1)
const archivePageSize = ref(10)
const archiveTotal = ref(0)

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



async function openAuditDetail(item: AuditSnapshotItem) {
  selectedAuditLog.value = item
  auditDetailVisible.value = true
  chainLoading.value = true
  expandedAuditChainNodes.value.clear()
  try {
    const res = await getAuditSnapshotChain(item.process_id)
    auditChainLogs.value = res.chain || []
    if (auditChainLogs.value.length > 0) {
      expandedAuditChainNodes.value.add(auditChainLogs.value[0].id)
    }
  } catch (e) {
    message.error(t('admin.data.fetchFailed'))
    auditChainLogs.value = []
  } finally {
    chainLoading.value = false
  }
}

function toggleAuditChainNode(id: string) {
  if (expandedAuditChainNodes.value.has(id)) expandedAuditChainNodes.value.delete(id)
  else expandedAuditChainNodes.value.add(id)
}

async function openArchiveDetail(item: ArchiveSnapshotItem) {
  selectedArchiveLog.value = item
  archiveDetailVisible.value = true
  chainLoading.value = true
  expandedArchiveChainNodes.value.clear()
  try {
    const res = await getArchiveSnapshotChain(item.process_id)
    archiveChainLogs.value = res.chain || []
    if (archiveChainLogs.value.length > 0) {
      expandedArchiveChainNodes.value.add(archiveChainLogs.value[0].id)
    }
  } catch (e) {
    message.error(t('admin.data.fetchFailed'))
    archiveChainLogs.value = []
  } finally {
    chainLoading.value = false
  }
}

function toggleArchiveChainNode(id: string) {
  if (expandedArchiveChainNodes.value.has(id)) expandedArchiveChainNodes.value.delete(id)
  else expandedArchiveChainNodes.value.add(id)
}

function openCronDetail(log: CronLogItem) {
  selectedCronLog.value = log
  cronDetailVisible.value = true
}

function clearAuditFilters() {
  auditSearch.value = ''
  auditFilterProcessPath.value = []
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
  archiveFilterProcessPath.value = []
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

async function loadProcessCascaderOptions() {
  try {
    const list = await getProcessTypes()
    const categoryMap = new Map<string, any>()
    for (const item of (Array.isArray(list) ? list : [])) {
      const catLabel = item.process_type_label || item.process_type
      if (!categoryMap.has(catLabel)) {
        categoryMap.set(catLabel, { label: catLabel, value: catLabel, children: [] })
      }
      const cat = categoryMap.get(catLabel)!
      if (!cat.children.some((c: any) => c.value === item.process_type)) {
        cat.children.push({ label: item.process_type, value: item.process_type })
      }
    }
    processCascaderOptions.value = Array.from(categoryMap.values())
  } catch (e) {
    console.warn('加载流程类型失败', e)
    processCascaderOptions.value = []
  }
}

const renderMarkdown = (text: string) => text ? marked.parse(text) : ''
const formatDate = (dateStr: string | null | undefined) => dateStr ? dayjs(dateStr).format('YYYY/MM/DD HH:mm') : '-'
const getAuditCount = (validLogIds: any) => {
  if (!validLogIds) return 0
  if (Array.isArray(validLogIds)) return validLogIds.length
  if (typeof validLogIds === 'string') {
    if (validLogIds.startsWith('[') && validLogIds.endsWith(']')) {
      try {
        const parsed = JSON.parse(validLogIds)
        return Array.isArray(parsed) ? parsed.length : 1
      } catch { /* 降级处理 */ }
    }
    if (validLogIds.includes(',')) return validLogIds.split(',').filter(Boolean).length
    if (validLogIds.length > 20) return 1 // 长度超过 20 视为单个 ID
    return isNaN(Number(validLogIds)) ? 1 : Number(validLogIds)
  }
  return 1
}


// 加载审核快照统计数据（各推荐结果的数量）
async function loadAuditStats() {
  try {
    auditStats.value = await getAuditSnapshotStats()
  } catch (e: any) {
    message.error(e?.message || t('admin.data.loadFailed'))
  }
}

// 加载定时任务执行日志统计数据
async function loadCronStats() {
  try {
    cronStats.value = await getCronLogStats()
  } catch (e: any) {
    message.error(e?.message || t('admin.data.loadFailed'))
  }
}

// 加载归档复盘快照统计数据（各合规状态的数量）
async function loadArchiveStats() {
  try {
    archiveStats.value = await getArchiveSnapshotStats()
  } catch (e: any) {
    message.error(e?.message || t('admin.data.loadFailed'))
  }
}

// 加载审核快照列表（分页，支持多维度筛选）
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

// 加载定时任务执行日志列表（分页，支持状态和类型筛选）
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

// 加载归档复盘快照列表（分页，支持多维度筛选）
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

// 切换审核子标签时重置分页到第一页
watch(activeAuditSubTab, () => {
  auditPage.value = 1
})

// 页面初始化：并行加载流程类型、部门列表及各模块统计数据
onMounted(async () => {
  await Promise.all([
    loadProcessCascaderOptions(),
    listDepartments().then(deps => departmentOptions.value = deps.map(d => ({ label: d.name, value: d.name }))).catch(() => {}),
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
      <div class="stats-row">
        <div
            v-for="tab in auditSubTabs"
            :key="tab.key"
            class="stat-card"
            :class="[tab.cssClass, { 'stat-card--selected': activeAuditSubTab === tab.key }]"
            @click="activeAuditSubTab = tab.key; auditPage = 1"
        >
          <div class="stat-card-icon"><component :is="tab.icon" /></div>
          <div class="stat-card-info">
            <span class="stat-card-value">{{ tab.count }}</span>
            <span class="stat-card-label">{{ tab.label }}</span>
          </div>
        </div>
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

          <a-cascader
              v-model:value="auditFilterProcessPath"
              :options="processCascaderOptions"
              :placeholder="t('admin.data.filterProcessType')"
              multiple
              :max-tag-count="1"
              allow-clear
              style="flex: 1.5; min-width: 160px;"
              :show-search="{ filter: (inputValue: string, path: any[]) => path.some((o: any) => o.label.toLowerCase().includes(inputValue.toLowerCase())) }"
              @change="auditPage = 1"
          />

          <a-select
              v-model:value="auditFilterDepartment"
              :placeholder="t('admin.data.filterDepartment')"
              allow-clear
              style="flex: 1; min-width: 140px;"
              :options="departmentOptions"
              @change="auditPage = 1"
          />

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
            <th>{{ t('admin.data.thAuditCount') }}</th>
            <th>{{ t('admin.data.thTime') }}</th>
            <th>{{ t('admin.data.thAction') }}</th>
          </tr>
          </thead>
          <tbody>
          <tr v-if="auditLoading">
            <td colspan="10" class="empty-cell">{{ t('admin.data.loading') }}</td>
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
                  <span class="conf-pill">AI {{ item.confidence }}%</span>
                </span>
            </td>
            <td>{{ getAuditCount(item.valid_log_ids) }}</td>
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
            <td colspan="10" class="empty-cell">{{ t('admin.data.noData') }}</td>
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
      <div class="stats-row">
        <div
            v-for="tab in [
            { key: 'all', icon: AppstoreOutlined, count: cronStats.total, label: t('admin.data.auditTab.all'), cssClass: 'stat-card--info' },
            { key: 'success', icon: CheckCircleOutlined, count: cronStats.success, label: t('admin.data.success'), cssClass: 'stat-card--success' },
            { key: 'failed', icon: CloseCircleOutlined, count: cronStats.failed, label: t('admin.data.failed'), cssClass: 'stat-card--danger' },
            { key: 'running', icon: SyncOutlined, count: cronStats.running, label: t('admin.data.running'), cssClass: 'stat-card--primary' },
          ]"
            :key="tab.key"
            class="stat-card"
            :class="[tab.cssClass, { 'stat-card--selected': activeCronSubTab === tab.key }]"
            @click="activeCronSubTab = tab.key as CronSubTab; cronPage = 1"
        >
          <div class="stat-card-icon"><component :is="tab.icon" /></div>
          <div class="stat-card-info">
            <span class="stat-card-value">{{ tab.count }}</span>
            <span class="stat-card-label">{{ tab.label }}</span>
          </div>
        </div>
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
              :options="departmentOptions"
              @change="cronPage = 1"
          />

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
            <td class="text-secondary">{{ item.task_type_label || item.task_type }}</td>
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
            <td class="text-secondary">{{ formatDate(item.started_at) }}</td>
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
      <div class="stats-row">
        <div
            v-for="tab in [
            { key: 'all', icon: AppstoreOutlined, count: archiveStats.total, label: t('admin.data.auditTab.all'), cssClass: 'stat-card--info' },
            { key: 'compliant', icon: SafetyCertificateOutlined, count: archiveStats.compliant, label: t('admin.data.compliant'), cssClass: 'stat-card--success' },
            { key: 'partially_compliant', icon: AlertOutlined, count: archiveStats.partial, label: t('admin.data.partiallyCompliant'), cssClass: 'stat-card--warning' },
            { key: 'non_compliant', icon: CloseCircleOutlined, count: archiveStats.non_compliant, label: t('admin.data.nonCompliant'), cssClass: 'stat-card--danger' },
          ]"
            :key="tab.key"
            class="stat-card"
            :class="[tab.cssClass, { 'stat-card--selected': activeArchiveSubTab === tab.key }]"
            @click="activeArchiveSubTab = tab.key as ArchiveSubTab; archivePage = 1"
        >
          <div class="stat-card-icon"><component :is="tab.icon" /></div>
          <div class="stat-card-info">
            <span class="stat-card-value">{{ tab.count }}</span>
            <span class="stat-card-label">{{ tab.label }}</span>
          </div>
        </div>
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

          <a-cascader
              v-model:value="archiveFilterProcessPath"
              :options="processCascaderOptions"
              :placeholder="t('admin.data.filterProcessType')"
              multiple
              :max-tag-count="1"
              allow-clear
              style="flex: 1.5; min-width: 160px;"
              :show-search="{ filter: (inputValue: string, path: any[]) => path.some((o: any) => o.label.toLowerCase().includes(inputValue.toLowerCase())) }"
              @change="archivePage = 1"
          />

          <a-select
              v-model:value="archiveFilterDepartment"
              :placeholder="t('admin.data.filterDepartment')"
              allow-clear
              style="flex: 1; min-width: 140px;"
              :options="departmentOptions"
              @change="archivePage = 1"
          />

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
            <th>{{ t('admin.data.thAuditCount') }}</th>
            <th>{{ t('admin.data.thTime') }}</th>
            <th>{{ t('admin.data.thAction') }}</th>
          </tr>
          </thead>
          <tbody>
          <tr v-if="archiveLoading">
            <td colspan="9" class="empty-cell">{{ t('admin.data.loading') }}</td>
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
                  <span class="conf-pill">AI {{ item.confidence }}%</span>
                </span>
              <span v-else class="text-secondary">-</span>
            </td>
            <td>{{ getAuditCount(item.valid_archive_log_ids) }}</td>
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
            <td colspan="9" class="empty-cell">{{ t('admin.data.noData') }}</td>
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

              <a-spin :spinning="chainLoading">
                <div v-if="!chainLoading && auditChainLogs.length === 0" style="padding: 40px; text-align: center;">
                  <a-empty :description="t('admin.data.noData')" />
                </div>
                <div v-else class="audit-chain" style="margin-top: 16px;">
                  <div
                      v-for="(logItem, idx) in auditChainLogs"
                      :key="logItem.id"
                      class="chain-node"
                  >
                    <div class="chain-timeline">
                      <div class="chain-dot" :style="{ background: recommendationConfig[logItem.recommendation]?.color || 'var(--color-primary)' }" />
                      <div v-if="idx < auditChainLogs.length - 1" class="chain-line" />
                    </div>
                    <div class="chain-card">
                      <div class="chain-card-header" @click="toggleAuditChainNode(logItem.id)" style="cursor: pointer;">
                        <span
                            class="chain-tag"
                            :style="{ color: recommendationConfig[logItem.recommendation]?.color || 'var(--color-text-primary)', background: recommendationConfig[logItem.recommendation]?.bg || 'var(--color-bg-page)' }"
                        >
                          <CheckCircleOutlined v-if="logItem.recommendation === 'approve'" />
                          <CloseCircleOutlined v-else-if="logItem.recommendation === 'return'" />
                          <AlertOutlined v-else />
                          {{ getRecLabel(logItem.recommendation) }}
                        </span>
                        <span class="chain-score">{{ logItem.score }}{{ t('admin.data.points') }}</span>
                        <span class="chain-conf-tag">{{ logItem.confidence }}%</span>
                        <span class="chain-expand-btn">
                          <DownOutlined v-if="!expandedAuditChainNodes.has(logItem.id)" />
                          <UpOutlined v-else />
                        </span>
                      </div>
                      <div class="chain-card-meta">
                        {{ formatDate(logItem.created_at) }}
                        <span v-if="logItem.user_name"> · {{ logItem.user_name }}</span>
                        · {{ t('admin.data.duration') }} {{ (logItem.duration_ms / 1000).toFixed(1) }}s
                      </div>

                      <div v-if="expandedAuditChainNodes.has(logItem.id)" class="chain-detail">
                        <template v-if="logItem.audit_result && typeof logItem.audit_result === 'object'">
                          
                          <!-- 规则细节 -->
                          <div class="detail-section" style="margin-top: 0; padding-top: 0; border: none;">
                            <h4 class="chain-section-title">{{ t('admin.data.ruleCheckDetail') }}</h4>
                            <div class="rule-checks">
                               <div
                                  v-for="(rule, index) in (logItem.audit_result.rule_results || [])"
                                  :key="index"
                                  class="chain-rule-item"
                                  :class="{ 'chain-rule--fail': rule.passed === false }"
                              >
                                <div class="chain-rule-name">{{ rule.rule_name || rule.rule_content || '-' }}</div>
                                <div class="chain-rule-reasoning">{{ rule.reasoning || rule.reason || '-' }}</div>
                              </div>
                              <div v-if="!logItem.audit_result.rule_results?.length" class="chain-no-detail">
                                {{ t('admin.data.noData') }}
                              </div>
                            </div>
                          </div>

                          <!-- 推理过程 -->
                          <div class="detail-section">
                            <!-- 风险点 & 优化建议 -->
                            <div v-if="logItem.audit_result?.risk_points?.length || logItem.audit_result?.suggestions?.length" class="risk-suggest-row" style="margin-bottom: 12px;">
                              <div v-if="logItem.audit_result?.risk_points?.length" class="insight-card insight-card--risk" style="padding: 10px;">
                                <div class="insight-card-header" style="margin-bottom: 4px;"><CloseCircleOutlined /> 风险点</div>
                                <ul class="insight-card-list" style="gap: 2px;">
                                  <li v-for="(p, i) in logItem.audit_result.risk_points" :key="i" style="font-size: 12px;">{{ p }}</li>
                                </ul>
                              </div>
                              <div v-if="logItem.audit_result?.suggestions?.length" class="insight-card insight-card--suggest" style="padding: 10px;">
                                <div class="insight-card-header" style="margin-bottom: 4px;"><InfoCircleOutlined /> 优化建议</div>
                                <ul class="insight-card-list" style="gap: 2px;">
                                  <li v-for="(s, i) in logItem.audit_result.suggestions" :key="i" style="font-size: 12px;">{{ s }}</li>
                                </ul>
                              </div>
                            </div>

                            <div v-if="logItem.ai_reasoning" class="chain-section-title">AI推理过程</div>
                            <div v-if="logItem.ai_reasoning" class="chain-reasoning">
                              <div class="markdown-body" v-html="renderMarkdown(logItem.ai_reasoning)"></div>
                            </div>
                          </div>
                        </template>
                        <div v-else class="chain-parse-error">
                          <CloseCircleOutlined />
                          {{ logItem.parse_error || t('admin.data.noData') }}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </a-spin>
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

              <a-spin :spinning="chainLoading">
                <div v-if="!chainLoading && archiveChainLogs.length === 0" style="padding: 40px; text-align: center;">
                  <a-empty :description="t('admin.data.noData')" />
                </div>
                <div v-else class="audit-chain" style="margin-top: 16px;">
                  <div
                      v-for="(logItem, idx) in archiveChainLogs"
                      :key="logItem.id"
                      class="chain-node"
                  >
                    <div class="chain-timeline">
                      <div class="chain-dot" :style="{ background: complianceConfig[logItem.archive_result?.overall_compliance || 'non_compliant']?.color || 'var(--color-danger)' }" />
                      <div v-if="idx < archiveChainLogs.length - 1" class="chain-line" />
                    </div>
                    <div class="chain-card">
                      <div class="chain-card-header" @click="toggleArchiveChainNode(logItem.id)" style="cursor: pointer;">
                        <span
                            class="chain-tag"
                            :style="{ color: complianceConfig[logItem.archive_result?.overall_compliance || 'non_compliant']?.color || 'var(--color-danger)', background: complianceConfig[logItem.archive_result?.overall_compliance || 'non_compliant']?.bg || 'var(--color-bg-page)' }"
                        >
                          <SafetyCertificateOutlined v-if="logItem.archive_result?.overall_compliance === 'compliant'" />
                          <AlertOutlined v-else-if="logItem.archive_result?.overall_compliance === 'partially_compliant'" />
                          <CloseCircleOutlined v-else />
                          {{ getComplianceLabel(logItem.archive_result?.overall_compliance) }}
                        </span>
                        <span class="chain-score">{{ logItem.archive_result?.overall_score || 0 }}{{ t('admin.data.points') }}</span>
                        <span class="chain-conf-tag">{{ logItem.confidence }}%</span>
                        <span class="chain-expand-btn">
                          <DownOutlined v-if="!expandedArchiveChainNodes.has(logItem.id)" />
                          <UpOutlined v-else />
                        </span>
                      </div>
                      <div class="chain-card-meta">
                        {{ formatDate(logItem.created_at) }}
                        <span v-if="logItem.user_name"> · {{ logItem.user_name }}</span>
                        · {{ t('admin.data.duration') }} {{ (logItem.duration_ms / 1000).toFixed(1) }}s
                      </div>

                      <div v-if="expandedArchiveChainNodes.has(logItem.id)" class="chain-detail">
                        <template v-if="logItem.archive_result && typeof logItem.archive_result === 'object'">
                          
                          <!-- 规则细节 -->
                          <div class="detail-section" style="margin-top: 0; padding-top: 0; border: none;">
                            <h4 class="chain-section-title">{{ t('admin.data.ruleAudit') }}</h4>
                            <div class="rule-checks">
                               <div
                                  v-for="(rule, index) in (logItem.archive_result.rule_audit || [])"
                                  :key="index"
                                  class="chain-rule-item"
                                  :class="{ 'chain-rule--fail': rule.passed === false }"
                              >
                                <div class="chain-rule-name">{{ rule.rule_name || '-' }}</div>
                                <div class="chain-rule-reasoning">{{ rule.reasoning || '-' }}</div>
                              </div>
                              <div v-if="!logItem.archive_result.rule_audit?.length" class="chain-no-detail">
                                {{ t('admin.data.noData') }}
                              </div>
                            </div>
                          </div>

                          <!-- 流程分析 (Archive) -->
                          <div v-if="logItem.archive_result?.flow_audit?.node_results?.length" class="chain-section-title">流程分析</div>
                          <div v-if="logItem.archive_result?.flow_audit?.node_results?.length" class="rule-checks" style="margin-bottom: 12px;">
                             <div v-for="(node, ni) in logItem.archive_result.flow_audit.node_results" :key="ni" class="chain-rule-item" :class="{ 'chain-rule--fail': !node.compliant }">
                               <div class="chain-rule-name">
                                 <CheckCircleOutlined v-if="node.compliant" style="color: var(--color-success); margin-right: 4px;" />
                                 <CloseCircleOutlined v-else style="color: var(--color-danger); margin-right: 4px;" />
                                 {{ node.node_name }}
                               </div>
                               <div class="chain-rule-reasoning">{{ node.reasoning }}</div>
                             </div>
                          </div>

                          <!-- 风险点 & 改进建议 -->
                          <div v-if="logItem.archive_result?.risk_points?.length || logItem.archive_result?.suggestions?.length" class="risk-suggest-row" style="margin-bottom: 12px;">
                            <div v-if="logItem.archive_result?.risk_points?.length" class="insight-card insight-card--risk" style="padding: 10px;">
                              <div class="insight-card-header" style="margin-bottom: 4px;"><CloseCircleOutlined /> 风险点</div>
                              <ul class="insight-card-list" style="gap: 2px;">
                                <li v-for="(p, i) in logItem.archive_result.risk_points" :key="i" style="font-size: 12px;">{{ p }}</li>
                              </ul>
                            </div>
                            <div v-if="logItem.archive_result?.suggestions?.length" class="insight-card insight-card--suggest" style="padding: 10px;">
                              <div class="insight-card-header" style="margin-bottom: 4px;"><InfoCircleOutlined /> 改进建议</div>
                              <ul class="insight-card-list" style="gap: 2px;">
                                <li v-for="(s, i) in logItem.archive_result.suggestions" :key="i" style="font-size: 12px;">{{ s }}</li>
                              </ul>
                            </div>
                          </div>

                          <!-- 推理过程 (Archive) -->
                          <div class="detail-section" v-if="logItem.ai_reasoning">
                            <h4 class="chain-section-title">{{ t('admin.data.aiSummary') }}</h4>
                            <div class="chain-reasoning">
                              <div class="markdown-body" v-html="renderMarkdown(logItem.ai_reasoning)"></div>
                            </div>
                          </div>
                        </template>
                        <div v-else class="chain-parse-error">
                          <CloseCircleOutlined />
                          {{ logItem.parse_error || t('admin.data.noData') }}
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </a-spin>
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
                  <span class="detail-meta-value">{{ selectedCronLog.task_type_label || selectedCronLog.task_type }}</span>
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
                  <span class="detail-meta-value">{{ formatDate(selectedCronLog.started_at) }}</span>
                </div>
                <div class="detail-meta-item">
                  <span class="detail-meta-label">{{ t('admin.data.cronEndTime') }}</span>
                  <span class="detail-meta-value">{{ formatDate(selectedCronLog.finished_at) }}</span>
                </div>
                <div class="detail-meta-item" v-if="selectedCronLog.push_email">
                  <span class="detail-meta-label">{{ t('admin.data.cronPushEmail') }}</span>
                  <span class="detail-meta-value">{{ selectedCronLog.push_email }}</span>
                </div>
                <div class="detail-meta-item" v-if="selectedCronLog.workflow_ids && selectedCronLog.workflow_ids !== '[]'">
                  <span class="detail-meta-label">{{ t('admin.data.cronWorkflows') }}</span>
                  <div class="detail-meta-value" style="display: flex; flex-wrap: wrap; gap: 4px;">
                    <span 
                      v-for="wf in (typeof selectedCronLog.workflow_ids === 'string' ? JSON.parse(selectedCronLog.workflow_ids) : selectedCronLog.workflow_ids)" 
                      :key="wf"
                      class="conf-pill"
                      style="margin: 0; padding: 2px 8px; font-size: 11px; border-radius: 4px;"
                    >
                      {{ wf }}
                    </span>
                  </div>
                </div>
                <div class="detail-meta-item" v-if="selectedCronLog.date_range">
                  <span class="detail-meta-label">{{ t('admin.data.cronDateRange') }}</span>
                  <span class="detail-meta-value">{{ selectedCronLog.date_range }} {{ t('admin.data.days') }}</span>
                </div>
              </div>

              <div class="detail-section" v-if="selectedCronLog.message">
                <h4 class="detail-section-title">{{ t('admin.data.cronMessage') }}</h4>
                <div class="ai-reasoning">
                  <div class="markdown-body" v-html="renderMarkdown(selectedCronLog.message || '')"></div>
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
/* Markdown 样式覆盖 */
.markdown-body { font-size: 13px; line-height: 1.7; color: var(--color-text-secondary); word-break: break-word; }
.markdown-body :deep(h1), .markdown-body :deep(h2), .markdown-body :deep(h3),
.markdown-body :deep(h4), .markdown-body :deep(h5), .markdown-body :deep(h6) { margin: 12px 0 6px; font-weight: 600; color: var(--color-text-primary); }
.markdown-body :deep(h1) { font-size: 18px; }
.markdown-body :deep(h2) { font-size: 16px; }
.markdown-body :deep(h3) { font-size: 14px; }
.markdown-body :deep(p) { margin: 6px 0; }
.markdown-body :deep(ul), .markdown-body :deep(ol) { padding-left: 20px; margin: 6px 0; }
.markdown-body :deep(li) { margin: 3px 0; }
.markdown-body :deep(code) { background: var(--color-bg-elevated); padding: 1px 5px; border-radius: 4px; font-size: 12px; }
.markdown-body :deep(pre) { background: var(--color-bg-elevated); padding: 12px; border-radius: 8px; overflow-x: auto; margin: 8px 0; }
.markdown-body :deep(pre code) { background: none; padding: 0; }
.markdown-body :deep(blockquote) { border-left: 3px solid var(--color-primary); padding: 4px 12px; margin: 8px 0; color: var(--color-text-tertiary); background: var(--color-bg-elevated); border-radius: 0 6px 6px 0; }
.markdown-body :deep(strong) { color: var(--color-text-primary); font-weight: 600; }
.markdown-body :deep(table) { width: 100%; border-collapse: collapse; margin: 8px 0; }
.markdown-body :deep(th), .markdown-body :deep(td) { border: 1px solid var(--color-border-light); padding: 6px 10px; font-size: 12px; }
.markdown-body :deep(th) { background: var(--color-bg-elevated); font-weight: 600; }
 
.conf-pill {
  display: inline-flex;
  align-items: center;
  padding: 1px 6px;
  background: var(--color-bg-page);
  border: 1px solid var(--color-border-light);
  border-radius: 10px;
  font-size: 10px;
  color: var(--color-text-tertiary);
  margin-left: 4px;
}

.chain-conf-tag {
  font-size: 11px;
  padding: 1px 6px;
  border-radius: 4px;
  background: var(--color-bg-hover);
  color: var(--color-text-secondary);
  font-weight: 500;
  margin-left: 4px;
}

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
  cursor: pointer;
  user-select: none;
}

.stat-card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}

.stat-card--selected {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 1px var(--color-primary);
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
  line-height: 1.7;
  color: var(--color-text-secondary);
  font-family: var(--font-sans);
}

/* 审核链样式 */
.audit-chain { display: flex; flex-direction: column; }
.chain-node { display: flex; gap: 16px; min-height: 40px; }
.chain-timeline { display: flex; flex-direction: column; align-items: center; width: 20px; flex-shrink: 0; }
.chain-dot { width: 12px; height: 12px; border-radius: 50%; flex-shrink: 0; margin-top: 20px; border: 2px solid var(--color-bg-card); }
.chain-line { width: 2px; flex: 1; background: var(--color-border-light); }
.chain-card {
  flex: 1; padding: 14px 16px; border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md); margin-bottom: 12px; transition: all var(--transition-fast);
  background: var(--color-bg-card);
}
.chain-card:hover { background: var(--color-bg-hover); border-color: var(--color-primary-light); }
.chain-card-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; }
.chain-tag {
  display: inline-flex; align-items: center; gap: 6px;
  font-size: 12px; font-weight: 600; padding: 3px 10px; border-radius: var(--radius-full);
}
.chain-score { font-size: 18px; font-weight: 700; color: var(--color-text-primary); }
.chain-card-meta { font-size: 12px; color: var(--color-text-tertiary); display: flex; align-items: center; gap: 8px; }
.chain-expand-btn { margin-left: auto; font-size: 12px; color: var(--color-text-tertiary); }
.chain-detail {
  margin-top: 12px; padding-top: 12px; border-top: 1px solid var(--color-border-light);
  display: flex; flex-direction: column; gap: 8px;
}
.chain-rule-item {
  display: flex; flex-direction: column; gap: 4px; padding: 8px 12px;
  border-radius: var(--radius-sm); border: 1px solid var(--color-border-light);
  transition: background var(--transition-fast);
}
.chain-rule-item:hover { background: var(--color-bg-hover); }
.chain-rule--fail { background: var(--color-danger-bg); border-color: rgba(239, 68, 68, 0.2); }
.chain-rule-name { font-size: 13px; font-weight: 600; color: var(--color-text-primary); }
.chain-rule-reasoning { font-size: 12px; color: var(--color-text-secondary); line-height: 1.5; }
.chain-reasoning { background: var(--color-bg-page); border-radius: var(--radius-sm); padding: 12px; border: 1px solid var(--color-border-light); }
.chain-no-detail { font-size: 12px; color: var(--color-text-tertiary); text-align: center; padding: 12px; }
.chain-section-title { font-size: 12px; font-weight: 600; color: var(--color-text-secondary); margin-bottom: 8px; margin-top: 12px; }
.chain-parse-error {
  display: flex; align-items: center; gap: 8px; padding: 10px 14px;
  border-radius: var(--radius-sm); background: var(--color-danger-bg);
  font-size: 12px; color: var(--color-danger); border: 1px solid rgba(239, 68, 68, 0.2);
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