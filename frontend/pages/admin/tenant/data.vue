<script setup lang="ts">
import {
  CalendarOutlined,
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
import * as XLSX from 'xlsx'
import { useI18n } from '~/composables/useI18n'
import { usePagination } from '~/composables/usePagination'
import {
  mockAuditLogs,
  mockCronLogs,
  mockArchiveLogs,
} from '~/composables/useMockData'
import type { AuditLog, CronLog, ArchiveLog, AuditResult, ArchiveAuditResult } from '~/composables/useMockData'
import type { Dayjs } from 'dayjs'
import dayjs from 'dayjs'
import 'dayjs/locale/zh-cn'

definePageMeta({ middleware: 'auth', layout: 'default' })

const { t } = useI18n()
const activeTab = ref<'audit' | 'cron' | 'archive'>('audit')

// Cascader options from mock data
const { processCascaderOptions, archiveProcessCascaderOptions } = useMockData()

// ===== Audit tab =====
const auditLogs = ref<AuditLog[]>(JSON.parse(JSON.stringify(mockAuditLogs)))
const auditSearch = ref('')
const auditSearchOperator = ref('')
const auditShowFilters = ref(false)
const auditFilterDepartment = ref<string | undefined>(undefined)
const auditFilterProcessType = ref<string[][]>([])
const auditFilterProcessNames = computed(() => {
  if (auditFilterProcessType.value.length === 0) return []
  const names: string[] = []
  for (const path of auditFilterProcessType.value) {
    if (path.length >= 2) {
      names.push(path[path.length - 1])
    } else if (path.length === 1) {
      const cat = processCascaderOptions.find((o: any) => o.value === path[0])
      if (cat && (cat as any).children) {
        names.push(...(cat as any).children.map((c: any) => c.value))
      }
    }
  }
  return names
})
const auditFilterStatus = ref<string | undefined>(undefined)
const auditFilterDateRange = ref<[Dayjs, Dayjs] | null>(null)
const auditSelectedIds = ref<string[]>([])
const auditDetailVisible = ref(false)
const auditDetailResult = ref<AuditResult | null>(null)
const auditDetailTitle = ref('')
const auditCardFilter = ref<string | null>(null)

const auditDepartmentOptions = computed(() => [...new Set(auditLogs.value.map(l => l.department))])
const auditHasActiveFilters = computed(() => !!auditSearch.value || !!auditSearchOperator.value || auditFilterDepartment.value !== undefined || auditFilterProcessType.value.length > 0 || auditFilterStatus.value !== undefined || auditFilterDateRange.value !== null)

const clearAuditFilters = () => { auditSearch.value = ''; auditSearchOperator.value = ''; auditFilterDepartment.value = undefined; auditFilterProcessType.value = []; auditFilterStatus.value = undefined; auditFilterDateRange.value = null }

const filteredAuditLogs = computed(() => {
  return auditLogs.value.filter(l => {
    if (auditCardFilter.value) {
      if (auditCardFilter.value === 'all') { /* no filter */ }
      else if (l.recommendation !== auditCardFilter.value) return false
    }
    if (auditSearch.value) { const q = auditSearch.value.toLowerCase(); if (!l.title.toLowerCase().includes(q) && !l.process_id.toLowerCase().includes(q)) return false }
    if (auditSearchOperator.value) { const q = auditSearchOperator.value.toLowerCase(); if (!l.operator.toLowerCase().includes(q)) return false }
    if (auditFilterDepartment.value && l.department !== auditFilterDepartment.value) return false
    if (auditFilterProcessNames.value.length > 0 && !auditFilterProcessNames.value.includes(l.process_type)) return false
    if (auditFilterStatus.value && l.recommendation !== auditFilterStatus.value) return false
    if (auditFilterDateRange.value) {
      const logDate = new Date(l.created_at).getTime()
      const start = auditFilterDateRange.value[0].startOf('day').valueOf()
      const end = auditFilterDateRange.value[1].endOf('day').valueOf()
      if (logDate < start || logDate > end) return false
    }
    return true
  })
})

const toggleAuditCardFilter = (filter: string) => { auditCardFilter.value = auditCardFilter.value === filter ? null : filter }

const auditPagination = usePagination(filteredAuditLogs, 10)

const toggleAuditSelect = (id: string) => { const idx = auditSelectedIds.value.indexOf(id); if (idx >= 0) auditSelectedIds.value.splice(idx, 1); else auditSelectedIds.value.push(id) }
const toggleAuditSelectAll = () => { if (auditSelectedIds.value.length === filteredAuditLogs.value.length) auditSelectedIds.value = []; else auditSelectedIds.value = filteredAuditLogs.value.map(l => l.id) }

const openAuditDetail = (log: AuditLog) => { auditDetailResult.value = log.audit_result; auditDetailTitle.value = log.title; auditDetailVisible.value = true }

// Audit stats
const auditApprovedCount = computed(() => auditLogs.value.filter(l => l.recommendation === 'approve').length)
const auditReturnedCount = computed(() => auditLogs.value.filter(l => l.recommendation === 'return').length)
const auditArchivedCount = computed(() => auditLogs.value.filter(l => l.recommendation === 'review').length)

// ===== Cron tab =====
const cronLogs = ref<CronLog[]>(JSON.parse(JSON.stringify(mockCronLogs)))
const cronSearchTask = ref('')
const cronSearchOperator = ref('')
const cronShowFilters = ref(false)
const cronStatusFilter = ref<string | undefined>(undefined)
const cronFilterTaskType = ref<string[]>([])
const cronFilterDepartment = ref<string | undefined>(undefined)
const cronSelectedIds = ref<string[]>([])
const cronCardFilter = ref<string | null>(null)
const cronHasActiveFilters = computed(() => !!cronSearchTask.value || !!cronSearchOperator.value || cronStatusFilter.value !== undefined || cronFilterTaskType.value.length > 0 || cronFilterDepartment.value !== undefined)
const clearCronFilters = () => { cronSearchTask.value = ''; cronSearchOperator.value = ''; cronStatusFilter.value = undefined; cronFilterTaskType.value = []; cronFilterDepartment.value = undefined }

const cronTaskTypeOptions = computed(() => [...new Set(cronLogs.value.map(l => l.task_label))].map(t => ({ label: t, value: t })))
const cronDepartmentOptions = computed(() => [...new Set(cronLogs.value.map(l => l.department))])

const filteredCronLogs = computed(() => {
  return cronLogs.value.filter(l => {
    if (cronCardFilter.value) {
      if (cronCardFilter.value === 'all') { /* no filter */ }
      else if (l.status !== cronCardFilter.value) return false
    }
    if (cronSearchTask.value && !l.task_label.includes(cronSearchTask.value) && !l.task_id.includes(cronSearchTask.value)) return false
    if (cronSearchOperator.value && !l.operator.includes(cronSearchOperator.value)) return false
    if (cronStatusFilter.value && l.status !== cronStatusFilter.value) return false
    if (cronFilterTaskType.value.length > 0 && !cronFilterTaskType.value.includes(l.task_label)) return false
    if (cronFilterDepartment.value && l.department !== cronFilterDepartment.value) return false
    return true
  })
})

const toggleCronCardFilter = (filter: string) => { cronCardFilter.value = cronCardFilter.value === filter ? null : filter }

// Cron tooltip: success shows "time + task succeeded", failed shows failure reason
const getCronResultTooltip = (l: CronLog) => l.message



const cronPagination = usePagination(filteredCronLogs, 10)
const toggleCronSelect = (id: string) => { const idx = cronSelectedIds.value.indexOf(id); if (idx >= 0) cronSelectedIds.value.splice(idx, 1); else cronSelectedIds.value.push(id) }
const toggleCronSelectAll = () => { if (cronSelectedIds.value.length === filteredCronLogs.value.length) cronSelectedIds.value = []; else cronSelectedIds.value = filteredCronLogs.value.map(l => l.id) }

// ===== Archive tab =====
const archiveLogs = ref<ArchiveLog[]>(JSON.parse(JSON.stringify(mockArchiveLogs)))
const archiveSearch = ref('')
const archiveSearchOperator = ref('')
const archiveShowFilters = ref(false)
const archiveFilterDepartment = ref<string | undefined>(undefined)
const archiveFilterProcessType = ref<string[][]>([])
const archiveFilterProcessNames = computed(() => {
  if (archiveFilterProcessType.value.length === 0) return []
  const names: string[] = []
  for (const path of archiveFilterProcessType.value) {
    if (path.length >= 2) {
      names.push(path[path.length - 1])
    } else if (path.length === 1) {
      const cat = archiveProcessCascaderOptions.find((o: any) => o.value === path[0])
      if (cat && (cat as any).children) {
        names.push(...(cat as any).children.map((c: any) => c.value))
      }
    }
  }
  return names
})
const archiveFilterCompliance = ref<string | undefined>(undefined)
const archiveFilterDateRange = ref<[Dayjs, Dayjs] | null>(null)
const archiveSelectedIds = ref<string[]>([])
const archiveDetailVisible = ref(false)
const archiveDetailResult = ref<ArchiveAuditResult | null>(null)
const archiveDetailTitle = ref('')
const archiveCardFilter = ref<string | null>(null)

const archiveDepartmentOptions = computed(() => [...new Set(archiveLogs.value.map(l => l.department))])
const archiveHasActiveFilters = computed(() => !!archiveSearch.value || !!archiveSearchOperator.value || archiveFilterDepartment.value !== undefined || archiveFilterProcessType.value.length > 0 || archiveFilterCompliance.value !== undefined || archiveFilterDateRange.value !== null)
const clearArchiveFilters = () => { archiveSearch.value = ''; archiveSearchOperator.value = ''; archiveFilterDepartment.value = undefined; archiveFilterProcessType.value = []; archiveFilterCompliance.value = undefined; archiveFilterDateRange.value = null }

const filteredArchiveLogs = computed(() => {
  return archiveLogs.value.filter(l => {
    if (archiveCardFilter.value) {
      if (archiveCardFilter.value === 'all') { /* no filter */ }
      else if (l.compliance !== archiveCardFilter.value) return false
    }
    if (archiveSearch.value) { const q = archiveSearch.value.toLowerCase(); if (!l.title.toLowerCase().includes(q) && !l.process_id.toLowerCase().includes(q)) return false }
    if (archiveSearchOperator.value) { const q = archiveSearchOperator.value.toLowerCase(); if (!l.operator.toLowerCase().includes(q)) return false }
    if (archiveFilterDepartment.value && l.department !== archiveFilterDepartment.value) return false
    if (archiveFilterProcessNames.value.length > 0 && !archiveFilterProcessNames.value.includes(l.process_type)) return false
    if (archiveFilterCompliance.value && l.compliance !== archiveFilterCompliance.value) return false
    if (archiveFilterDateRange.value) {
      const logDate = new Date(l.created_at).getTime()
      const start = archiveFilterDateRange.value[0].startOf('day').valueOf()
      const end = archiveFilterDateRange.value[1].endOf('day').valueOf()
      if (logDate < start || logDate > end) return false
    }
    return true
  })
})

const toggleArchiveCardFilter = (filter: string) => { archiveCardFilter.value = archiveCardFilter.value === filter ? null : filter }

const archivePagination = usePagination(filteredArchiveLogs, 10)
const toggleArchiveSelect = (id: string) => { const idx = archiveSelectedIds.value.indexOf(id); if (idx >= 0) archiveSelectedIds.value.splice(idx, 1); else archiveSelectedIds.value.push(id) }
const toggleArchiveSelectAll = () => { if (archiveSelectedIds.value.length === filteredArchiveLogs.value.length) archiveSelectedIds.value = []; else archiveSelectedIds.value = filteredArchiveLogs.value.map(l => l.id) }

const openArchiveDetail = (log: ArchiveLog) => { archiveDetailResult.value = log.archive_result; archiveDetailTitle.value = log.title; archiveDetailVisible.value = true }

const archiveCompliantCount = computed(() => archiveLogs.value.filter(l => l.compliance === 'compliant').length)
const archivePartialCount = computed(() => archiveLogs.value.filter(l => l.compliance === 'partially_compliant').length)
const archiveNonCompliantCount = computed(() => archiveLogs.value.filter(l => l.compliance === 'non_compliant').length)

// ===== Recommendation / Compliance display configs =====
const recommendationConfig: Record<string, { color: string; bg: string; label: string }> = {
  approve: { color: 'var(--color-success)', bg: 'var(--color-success-bg)', label: '' },
  return: { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)', label: '' },
  review: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', label: '' },
}
// Lazy-init labels (needs t())
const getRecLabel = (rec: string) => {
  const map: Record<string, string> = { approve: t('admin.data.auditApprove'), return: t('admin.data.auditReturn'), review: t('admin.data.auditReview') }
  return map[rec] || rec
}

const complianceConfig: Record<string, { color: string; bg: string }> = {
  compliant: { color: 'var(--color-success)', bg: 'var(--color-success-bg)' },
  non_compliant: { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)' },
  partially_compliant: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)' },
}
const getComplianceLabel = (c: string) => {
  const map: Record<string, string> = { compliant: t('admin.data.compliant'), non_compliant: t('admin.data.nonCompliant'), partially_compliant: t('admin.data.partiallyCompliant') }
  return map[c] || c
}

// ===== Export (requires selection) =====
const handleExport = (type: 'audit' | 'cron' | 'archive') => {
  const selectedIds = type === 'audit' ? auditSelectedIds.value : type === 'cron' ? cronSelectedIds.value : archiveSelectedIds.value
  if (selectedIds.length === 0) { message.warning(t('admin.data.selectToExport')); return }
  const msgKey = type === 'audit' ? 'admin.data.exportingAudit' : type === 'cron' ? 'admin.data.exportingCron' : 'admin.data.exportingArchive'
  message.loading(t(msgKey), 1)
  setTimeout(() => {
    let data: any[]
    if (type === 'audit') data = auditLogs.value.filter(l => selectedIds.includes(l.id))
    else if (type === 'cron') data = cronLogs.value.filter(l => selectedIds.includes(l.id))
    else data = archiveLogs.value.filter(l => selectedIds.includes(l.id))
    const ws = XLSX.utils.json_to_sheet(data)
    const wb = XLSX.utils.book_new()
    XLSX.utils.book_append_sheet(wb, ws, type)
    XLSX.writeFile(wb, `${type}_data_${new Date().getTime()}.xlsx`)
  }, 1000)
}
</script>

<template>
  <div class="data-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('admin.data.title') }}</h1>
        <p class="page-subtitle">{{ t('admin.data.subtitle') }}</p>
      </div>
    </div>

    <!-- Top tabs -->
    <div class="tab-nav">
      <button v-for="tab in [
        { key: 'audit', label: t('admin.data.tabAudit'), icon: AppstoreOutlined },
        { key: 'cron', label: t('admin.data.tabCron'), icon: ClockCircleOutlined },
        { key: 'archive', label: t('admin.data.tabArchive'), icon: FolderOpenOutlined },
      ]" :key="tab.key" class="tab-btn" :class="{ 'tab-btn--active': activeTab === tab.key }" @click="activeTab = tab.key as any">
        <component :is="tab.icon" style="font-size: 14px;" />
        {{ tab.label }}
      </button>
    </div>

    <!-- ===== Audit Tab ===== -->
    <div v-if="activeTab === 'audit'" class="tab-content fade-in">
      <!-- Stats cards -->
      <div class="stats-row">
        <div class="stat-card stat-card--primary" :class="{ 'stat-card--selected': auditCardFilter === 'all' }" @click="toggleAuditCardFilter('all')" style="cursor: pointer;">
          <div class="stat-card-icon"><AppstoreOutlined /></div>
          <div class="stat-card-info">
            <span class="stat-card-value">{{ auditLogs.length }}</span>
            <span class="stat-card-label">{{ t('admin.data.totalRecords') }}</span>
          </div>
        </div>
        <div class="stat-card stat-card--success" :class="{ 'stat-card--selected': auditCardFilter === 'approve' }" @click="toggleAuditCardFilter('approve')" style="cursor: pointer;">
          <div class="stat-card-icon"><CheckCircleOutlined /></div>
          <div class="stat-card-info">
            <span class="stat-card-value">{{ auditApprovedCount }}</span>
            <span class="stat-card-label">{{ t('admin.data.approved') }}</span>
          </div>
        </div>
        <div class="stat-card stat-card--danger" :class="{ 'stat-card--selected': auditCardFilter === 'return' }" @click="toggleAuditCardFilter('return')" style="cursor: pointer;">
          <div class="stat-card-icon"><CloseCircleOutlined /></div>
          <div class="stat-card-info">
            <span class="stat-card-value">{{ auditReturnedCount }}</span>
            <span class="stat-card-label">{{ t('admin.data.returned') }}</span>
          </div>
        </div>
        <div class="stat-card stat-card--warning" :class="{ 'stat-card--selected': auditCardFilter === 'review' }" @click="toggleAuditCardFilter('review')" style="cursor: pointer;">
          <div class="stat-card-icon"><AlertOutlined /></div>
          <div class="stat-card-info">
            <span class="stat-card-value">{{ auditArchivedCount }}</span>
            <span class="stat-card-label">{{ t('admin.data.archived') }}</span>
          </div>
        </div>
      </div>

      <!-- Toolbar: filter toggle + export -->
      <div class="toolbar">
        <div class="toolbar-left">
          <a-button size="small" @click="auditShowFilters = !auditShowFilters" :class="{ 'filter-toggle-btn--active': auditHasActiveFilters }">
            <FilterOutlined /> {{ t('admin.data.filter') }}
            <span v-if="auditHasActiveFilters" class="filter-active-dot" />
          </a-button>
          <span v-if="auditSelectedIds.length > 0" class="batch-selected-hint">{{ t('admin.data.selected', `${auditSelectedIds.length}`) }}</span>
        </div>
        <div class="toolbar-right">
          <a-button @click="handleExport('audit')">
            <ExportOutlined /> {{ t('admin.data.export') }}
          </a-button>
        </div>
      </div>

      <!-- Collapsible filters -->
      <transition name="slide">
        <div v-if="auditShowFilters" class="filter-bar">
          <a-input v-model:value="auditSearch" :placeholder="t('admin.data.searchAudit')" allow-clear style="flex: 2; min-width: 160px;">
            <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
          </a-input>
          <a-input v-model:value="auditSearchOperator" :placeholder="t('admin.data.searchOperator')" allow-clear style="flex: 1; min-width: 120px;">
            <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
          </a-input>
          <a-cascader
            v-model:value="auditFilterProcessType"
            :options="processCascaderOptions"
            :placeholder="t('admin.data.filterProcessType')"
            multiple
            :max-tag-count="1"
            allow-clear
            style="flex: 1; min-width: 140px;"
            :show-search="{ filter: (inputValue: string, path: any[]) => path.some((o: any) => o.label.toLowerCase().includes(inputValue.toLowerCase())) }"
          />
          <a-select v-model:value="auditFilterDepartment" :placeholder="t('admin.data.filterDepartment')" allow-clear style="flex: 1; min-width: 100px;">
            <a-select-option v-for="d in auditDepartmentOptions" :key="d" :value="d">{{ d }}</a-select-option>
          </a-select>
          <a-select v-model:value="auditFilterStatus" :placeholder="t('admin.data.filterAuditStatus')" allow-clear style="flex: 1; min-width: 120px;">
            <a-select-option value="approve">{{ t('admin.data.auditApprove') }}</a-select-option>
            <a-select-option value="return">{{ t('admin.data.auditReturn') }}</a-select-option>
            <a-select-option value="review">{{ t('admin.data.auditReview') }}</a-select-option>
          </a-select>
          <a-range-picker v-model:value="auditFilterDateRange" :placeholder="[t('admin.data.filterDateRange'), t('admin.data.filterDateRange')]" allow-clear style="flex: 1; min-width: 200px;" />
          <a-button size="small" @click="clearAuditFilters">{{ t('admin.data.filterReset') }}</a-button>
        </div>
      </transition>

      <!-- Audit data table -->
      <div class="data-table-card">
        <table class="data-table">
          <thead>
            <tr>
              <th style="width: 40px;">
                <a-checkbox
                  :checked="auditSelectedIds.length === filteredAuditLogs.length && filteredAuditLogs.length > 0"
                  :indeterminate="auditSelectedIds.length > 0 && auditSelectedIds.length < filteredAuditLogs.length"
                  @change="toggleAuditSelectAll"
                />
              </th>
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
            <tr v-for="l in auditPagination.paged.value" :key="l.id">
              <td @click.stop="toggleAuditSelect(l.id)" style="cursor: pointer;">
                <a-checkbox :checked="auditSelectedIds.includes(l.id)" />
              </td>
              <td class="text-mono">{{ l.process_id }}</td>
              <td>{{ l.title }}</td>
              <td>{{ l.operator }}</td>
              <td class="text-secondary">{{ l.department }}</td>
              <td class="text-secondary">{{ l.process_type }}</td>
              <td>
                <span class="result-tag" :style="{ color: recommendationConfig[l.recommendation]?.color, background: recommendationConfig[l.recommendation]?.bg }">
                  <CheckCircleOutlined v-if="l.recommendation === 'approve'" />
                  <CloseCircleOutlined v-else-if="l.recommendation === 'return'" />
                  <AlertOutlined v-else />
                  {{ getRecLabel(l.recommendation) }} {{ l.score }}{{ t('admin.data.points') }}
                </span>
              </td>
              <td class="text-secondary">{{ l.created_at }}</td>
              <td>
                <div class="action-btns">
                  <button class="icon-btn" :title="t('admin.data.viewDetail')" @click="openAuditDetail(l)"><EyeOutlined /></button>
                </div>
              </td>
            </tr>
            <tr v-if="auditPagination.paged.value.length === 0">
              <td colspan="9" class="empty-cell">{{ t('admin.data.noData') }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="pagination-wrapper">
        <a-pagination v-model:current="auditPagination.current.value" :page-size="auditPagination.pageSize.value" :total="auditPagination.total.value" size="small" show-size-changer show-quick-jumper :page-size-options="['10', '20', '50']" @change="auditPagination.onChange" @showSizeChange="auditPagination.onChange" />
      </div>
    </div>

    <!-- ===== Cron Tab ===== -->
    <div v-if="activeTab === 'cron'" class="tab-content fade-in">
      <div class="stats-row">
        <div class="stat-card stat-card--primary" :class="{ 'stat-card--selected': cronCardFilter === 'all' }" @click="toggleCronCardFilter('all')" style="cursor: pointer;">
          <div class="stat-card-icon"><ClockCircleOutlined /></div>
          <div class="stat-card-info">
            <span class="stat-card-value">{{ cronLogs.length }}</span>
            <span class="stat-card-label">{{ t('admin.data.totalExec') }}</span>
          </div>
        </div>
        <div class="stat-card stat-card--success" :class="{ 'stat-card--selected': cronCardFilter === 'success' }" @click="toggleCronCardFilter('success')" style="cursor: pointer;">
          <div class="stat-card-icon"><CheckCircleOutlined /></div>
          <div class="stat-card-info">
            <span class="stat-card-value">{{ cronLogs.filter(l => l.status === 'success').length }}</span>
            <span class="stat-card-label">{{ t('admin.data.success') }}</span>
          </div>
        </div>
        <div class="stat-card stat-card--danger" :class="{ 'stat-card--selected': cronCardFilter === 'failed' }" @click="toggleCronCardFilter('failed')" style="cursor: pointer;">
          <div class="stat-card-icon"><CloseCircleOutlined /></div>
          <div class="stat-card-info">
            <span class="stat-card-value">{{ cronLogs.filter(l => l.status === 'failed').length }}</span>
            <span class="stat-card-label">{{ t('admin.data.failed') }}</span>
          </div>
        </div>
      </div>

      <div class="toolbar">
        <div class="toolbar-left">
          <a-button size="small" @click="cronShowFilters = !cronShowFilters" :class="{ 'filter-toggle-btn--active': cronHasActiveFilters }">
            <FilterOutlined /> {{ t('admin.data.filter') }}
            <span v-if="cronHasActiveFilters" class="filter-active-dot" />
          </a-button>
          <span v-if="cronSelectedIds.length > 0" class="batch-selected-hint">{{ t('admin.data.selected', `${cronSelectedIds.length}`) }}</span>
        </div>
        <div class="toolbar-right">
          <a-button @click="handleExport('cron')">
            <ExportOutlined /> {{ t('admin.data.export') }}
          </a-button>
        </div>
      </div>

      <transition name="slide">
        <div v-if="cronShowFilters" class="filter-bar">
          <a-input v-model:value="cronSearchTask" :placeholder="t('admin.data.searchCronTask')" allow-clear style="flex: 1; min-width: 140px;">
            <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
          </a-input>
          <a-input v-model:value="cronSearchOperator" :placeholder="t('admin.data.searchCronOperator')" allow-clear style="flex: 1; min-width: 120px;">
            <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
          </a-input>
          <a-select v-model:value="cronFilterDepartment" :placeholder="t('admin.data.filterDepartment')" allow-clear style="flex: 1; min-width: 100px;">
            <a-select-option v-for="d in cronDepartmentOptions" :key="d" :value="d">{{ d }}</a-select-option>
          </a-select>
          <a-select v-model:value="cronFilterTaskType" mode="multiple" :placeholder="t('admin.data.thTaskType')" allow-clear style="flex: 1; min-width: 120px;" :options="cronTaskTypeOptions" :max-tag-count="1" />
          <a-select v-model:value="cronStatusFilter" :placeholder="t('admin.data.execStatus')" allow-clear style="flex: 1; min-width: 120px;">
            <a-select-option value="success">{{ t('admin.data.success') }}</a-select-option>
            <a-select-option value="failed">{{ t('admin.data.failed') }}</a-select-option>
            <a-select-option value="running">{{ t('admin.data.running') }}</a-select-option>
          </a-select>
          <a-button size="small" @click="clearCronFilters">{{ t('admin.data.filterReset') }}</a-button>
        </div>
      </transition>

      <div class="data-table-card">
        <table class="data-table">
          <thead>
            <tr>
              <th style="width: 40px;">
                <a-checkbox
                  :checked="cronSelectedIds.length === filteredCronLogs.length && filteredCronLogs.length > 0"
                  :indeterminate="cronSelectedIds.length > 0 && cronSelectedIds.length < filteredCronLogs.length"
                  @change="toggleCronSelectAll"
                />
              </th>
              <th>{{ t('admin.data.thTaskId') }}</th>
              <th>{{ t('admin.data.thTaskType') }}</th>
              <th>{{ t('admin.data.thOperator') }}</th>
              <th>{{ t('admin.data.thDepartment') }}</th>
              <th>{{ t('admin.data.thStartTime') }}</th>
              <th>{{ t('admin.data.thEndTime') }}</th>
              <th>{{ t('admin.data.thMessage') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="l in cronPagination.paged.value" :key="l.id">
              <td @click.stop="toggleCronSelect(l.id)" style="cursor: pointer;">
                <a-checkbox :checked="cronSelectedIds.includes(l.id)" />
              </td>
              <td class="text-mono">{{ l.task_id }}</td>
              <td>{{ l.task_label }}</td>
              <td>{{ l.operator }}</td>
              <td class="text-secondary">{{ l.department }}</td>
              <td class="text-secondary">{{ l.started_at }}</td>
              <td class="text-secondary">{{ l.finished_at || '-' }}</td>
              <td>
                <a-tooltip :title="getCronResultTooltip(l)" placement="topLeft">
                  <span class="status-tag" :class="'status-tag--' + l.status" style="cursor: default;">
                    <CheckCircleOutlined v-if="l.status === 'success'" />
                    <CloseCircleOutlined v-else-if="l.status === 'failed'" />
                    <SyncOutlined v-else spin />
                    {{ l.status === 'success' ? t('admin.data.success') : l.status === 'failed' ? t('admin.data.failed') : t('admin.data.running') }}
                  </span>
                </a-tooltip>
              </td>
            </tr>
            <tr v-if="cronPagination.paged.value.length === 0">
              <td colspan="8" class="empty-cell">{{ t('admin.data.noData') }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="pagination-wrapper">
        <a-pagination v-model:current="cronPagination.current.value" :page-size="cronPagination.pageSize.value" :total="cronPagination.total.value" size="small" show-size-changer show-quick-jumper :page-size-options="['10', '20', '50']" @change="cronPagination.onChange" @showSizeChange="cronPagination.onChange" />
      </div>
    </div>

    <!-- ===== Archive Tab ===== -->
    <div v-if="activeTab === 'archive'" class="tab-content fade-in">
      <div class="stats-row">
        <div class="stat-card stat-card--primary" :class="{ 'stat-card--selected': archiveCardFilter === 'all' }" @click="toggleArchiveCardFilter('all')" style="cursor: pointer;">
          <div class="stat-card-icon"><FolderOpenOutlined /></div>
          <div class="stat-card-info">
            <span class="stat-card-value">{{ archiveLogs.length }}</span>
            <span class="stat-card-label">{{ t('admin.data.totalRecords') }}</span>
          </div>
        </div>
        <div class="stat-card stat-card--success" :class="{ 'stat-card--selected': archiveCardFilter === 'compliant' }" @click="toggleArchiveCardFilter('compliant')" style="cursor: pointer;">
          <div class="stat-card-icon"><CheckCircleOutlined /></div>
          <div class="stat-card-info">
            <span class="stat-card-value">{{ archiveCompliantCount }}</span>
            <span class="stat-card-label">{{ t('admin.data.compliant') }}</span>
          </div>
        </div>
        <div class="stat-card stat-card--warning" :class="{ 'stat-card--selected': archiveCardFilter === 'partially_compliant' }" @click="toggleArchiveCardFilter('partially_compliant')" style="cursor: pointer;">
          <div class="stat-card-icon"><AlertOutlined /></div>
          <div class="stat-card-info">
            <span class="stat-card-value">{{ archivePartialCount }}</span>
            <span class="stat-card-label">{{ t('admin.data.partiallyCompliant') }}</span>
          </div>
        </div>
        <div class="stat-card stat-card--danger" :class="{ 'stat-card--selected': archiveCardFilter === 'non_compliant' }" @click="toggleArchiveCardFilter('non_compliant')" style="cursor: pointer;">
          <div class="stat-card-icon"><CloseCircleOutlined /></div>
          <div class="stat-card-info">
            <span class="stat-card-value">{{ archiveNonCompliantCount }}</span>
            <span class="stat-card-label">{{ t('admin.data.nonCompliant') }}</span>
          </div>
        </div>
      </div>

      <div class="toolbar">
        <div class="toolbar-left">
          <a-button size="small" @click="archiveShowFilters = !archiveShowFilters" :class="{ 'filter-toggle-btn--active': archiveHasActiveFilters }">
            <FilterOutlined /> {{ t('admin.data.filter') }}
            <span v-if="archiveHasActiveFilters" class="filter-active-dot" />
          </a-button>
          <span v-if="archiveSelectedIds.length > 0" class="batch-selected-hint">{{ t('admin.data.selected', `${archiveSelectedIds.length}`) }}</span>
        </div>
        <div class="toolbar-right">
          <a-button @click="handleExport('archive')">
            <ExportOutlined /> {{ t('admin.data.export') }}
          </a-button>
        </div>
      </div>

      <transition name="slide">
        <div v-if="archiveShowFilters" class="filter-bar">
          <a-input v-model:value="archiveSearch" :placeholder="t('admin.data.searchArchive')" allow-clear style="flex: 2; min-width: 160px;">
            <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
          </a-input>
          <a-input v-model:value="archiveSearchOperator" :placeholder="t('admin.data.searchArchiveOperator')" allow-clear style="flex: 1; min-width: 120px;">
            <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
          </a-input>
          <a-cascader
            v-model:value="archiveFilterProcessType"
            :options="archiveProcessCascaderOptions"
            :placeholder="t('admin.data.filterProcessType')"
            multiple
            :max-tag-count="1"
            allow-clear
            style="flex: 1; min-width: 140px;"
            :show-search="{ filter: (inputValue: string, path: any[]) => path.some((o: any) => o.label.toLowerCase().includes(inputValue.toLowerCase())) }"
          />
          <a-select v-model:value="archiveFilterDepartment" :placeholder="t('admin.data.filterDepartment')" allow-clear style="flex: 1; min-width: 100px;">
            <a-select-option v-for="d in archiveDepartmentOptions" :key="d" :value="d">{{ d }}</a-select-option>
          </a-select>
          <a-select v-model:value="archiveFilterCompliance" :placeholder="t('admin.data.filterAuditStatus')" allow-clear style="flex: 1; min-width: 120px;">
            <a-select-option value="compliant">{{ t('admin.data.compliant') }}</a-select-option>
            <a-select-option value="partially_compliant">{{ t('admin.data.partiallyCompliant') }}</a-select-option>
            <a-select-option value="non_compliant">{{ t('admin.data.nonCompliant') }}</a-select-option>
          </a-select>
          <a-range-picker v-model:value="archiveFilterDateRange" :placeholder="[t('admin.data.filterDateRange'), t('admin.data.filterDateRange')]" allow-clear style="flex: 1; min-width: 200px;" />
          <a-button size="small" @click="clearArchiveFilters">{{ t('admin.data.filterReset') }}</a-button>
        </div>
      </transition>

      <div class="data-table-card">
        <table class="data-table">
          <thead>
            <tr>
              <th style="width: 40px;">
                <a-checkbox
                  :checked="archiveSelectedIds.length === filteredArchiveLogs.length && filteredArchiveLogs.length > 0"
                  :indeterminate="archiveSelectedIds.length > 0 && archiveSelectedIds.length < filteredArchiveLogs.length"
                  @change="toggleArchiveSelectAll"
                />
              </th>
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
            <tr v-for="l in archivePagination.paged.value" :key="l.id">
              <td @click.stop="toggleArchiveSelect(l.id)" style="cursor: pointer;">
                <a-checkbox :checked="archiveSelectedIds.includes(l.id)" />
              </td>
              <td class="text-mono">{{ l.process_id }}</td>
              <td>{{ l.title }}</td>
              <td>{{ l.operator }}</td>
              <td class="text-secondary">{{ l.department }}</td>
              <td class="text-secondary">{{ l.process_type }}</td>
              <td>
                <span class="result-tag" :style="{ color: complianceConfig[l.compliance]?.color, background: complianceConfig[l.compliance]?.bg }">
                  <CheckCircleOutlined v-if="l.compliance === 'compliant'" />
                  <AlertOutlined v-else-if="l.compliance === 'partially_compliant'" />
                  <CloseCircleOutlined v-else />
                  {{ getComplianceLabel(l.compliance) }} {{ l.compliance_score }}{{ t('admin.data.points') }}
                </span>
              </td>
              <td class="text-secondary">{{ l.created_at }}</td>
              <td>
                <div class="action-btns">
                  <button class="icon-btn" :title="t('admin.data.viewDetail')" @click="openArchiveDetail(l)"><EyeOutlined /></button>
                </div>
              </td>
            </tr>
            <tr v-if="archivePagination.paged.value.length === 0">
              <td colspan="9" class="empty-cell">{{ t('admin.data.noData') }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="pagination-wrapper">
        <a-pagination v-model:current="archivePagination.current.value" :page-size="archivePagination.pageSize.value" :total="archivePagination.total.value" size="small" show-size-changer show-quick-jumper :page-size-options="['10', '20', '50']" @change="archivePagination.onChange" @showSizeChange="archivePagination.onChange" />
      </div>
    </div>

    <!-- ===== Audit Detail Drawer ===== -->
    <Teleport to="body">
      <transition name="drawer">
        <div v-if="auditDetailVisible" class="drawer-overlay" @click.self="auditDetailVisible = false">
          <div class="drawer-panel">
            <div class="drawer-header">
              <h3>{{ t('admin.data.detailTitle') }}</h3>
              <button class="drawer-close" @click="auditDetailVisible = false"><CloseOutlined /></button>
            </div>
            <div class="drawer-body" v-if="auditDetailResult">
              <div class="detail-process-title">{{ auditDetailTitle }}</div>
              <!-- Recommendation banner -->
              <div class="detail-banner" :style="{ background: recommendationConfig[auditDetailResult.recommendation]?.bg, borderColor: recommendationConfig[auditDetailResult.recommendation]?.color }">
                <CheckCircleOutlined v-if="auditDetailResult.recommendation === 'approve'" :style="{ color: recommendationConfig[auditDetailResult.recommendation]?.color, fontSize: '24px' }" />
                <CloseCircleOutlined v-else-if="auditDetailResult.recommendation === 'return'" :style="{ color: recommendationConfig[auditDetailResult.recommendation]?.color, fontSize: '24px' }" />
                <AlertOutlined v-else :style="{ color: recommendationConfig[auditDetailResult.recommendation]?.color, fontSize: '24px' }" />
                <div class="detail-banner-info">
                  <div class="detail-banner-title" :style="{ color: recommendationConfig[auditDetailResult.recommendation]?.color }">{{ getRecLabel(auditDetailResult.recommendation) }}</div>
                  <div class="detail-banner-meta">{{ t('admin.data.overallScore') }} {{ auditDetailResult.score }}{{ t('admin.data.points') }} · {{ t('admin.data.duration') }} {{ auditDetailResult.duration_ms }}ms</div>
                </div>
                <div class="detail-score" :style="{ color: recommendationConfig[auditDetailResult.recommendation]?.color }">{{ auditDetailResult.score }}</div>
              </div>
              <!-- Rule checks -->
              <div class="detail-section">
                <h4 class="detail-section-title">{{ t('admin.data.ruleCheckDetail') }}</h4>
                <div class="rule-checks">
                  <div v-for="rule in auditDetailResult.details" :key="rule.rule_id" class="rule-check-item" :class="{ 'rule-check-item--pass': rule.passed, 'rule-check-item--fail': !rule.passed }">
                    <div class="rule-check-status">
                      <CheckCircleOutlined v-if="rule.passed" style="color: var(--color-success);" />
                      <CloseCircleOutlined v-else style="color: var(--color-danger);" />
                    </div>
                    <div class="rule-check-content">
                      <div class="rule-check-name">{{ rule.rule_name }}</div>
                      <div class="rule-check-reasoning">{{ rule.reasoning }}</div>
                    </div>
                  </div>
                </div>
              </div>
              <!-- Risk & Suggestions -->
              <div v-if="auditDetailResult.risk_points?.length || auditDetailResult.suggestions?.length" class="risk-suggest-row">
                <div v-if="auditDetailResult.risk_points?.length" class="insight-card insight-card--risk">
                  <div class="insight-card-header"><CloseCircleOutlined style="color: var(--color-danger);" /> <span>{{ t('admin.data.riskPoints') }}</span></div>
                  <ul class="insight-card-list"><li v-for="(rp, i) in auditDetailResult.risk_points" :key="i">{{ rp }}</li></ul>
                </div>
                <div v-if="auditDetailResult.suggestions?.length" class="insight-card insight-card--suggest">
                  <div class="insight-card-header"><InfoCircleOutlined style="color: var(--color-primary);" /> <span>{{ t('admin.data.suggestions') }}</span></div>
                  <ul class="insight-card-list"><li v-for="(sg, i) in auditDetailResult.suggestions" :key="i">{{ sg }}</li></ul>
                </div>
              </div>
              <!-- AI Reasoning -->
              <div class="detail-section">
                <h4 class="detail-section-title">{{ t('admin.data.aiReasoning') }}</h4>
                <div class="ai-reasoning"><pre>{{ auditDetailResult.ai_reasoning }}</pre></div>
              </div>
            </div>
          </div>
        </div>
      </transition>
    </Teleport>

    <!-- ===== Archive Detail Drawer ===== -->
    <Teleport to="body">
      <transition name="drawer">
        <div v-if="archiveDetailVisible" class="drawer-overlay" @click.self="archiveDetailVisible = false">
          <div class="drawer-panel">
            <div class="drawer-header">
              <h3>{{ t('admin.data.archiveDetailTitle') }}</h3>
              <button class="drawer-close" @click="archiveDetailVisible = false"><CloseOutlined /></button>
            </div>
            <div class="drawer-body" v-if="archiveDetailResult">
              <div class="detail-process-title">{{ archiveDetailTitle }}</div>
              <!-- Compliance banner -->
              <div class="detail-banner" :style="{ background: complianceConfig[archiveDetailResult.overall_compliance]?.bg, borderColor: complianceConfig[archiveDetailResult.overall_compliance]?.color }">
                <SafetyCertificateOutlined :style="{ color: complianceConfig[archiveDetailResult.overall_compliance]?.color, fontSize: '24px' }" />
                <div class="detail-banner-info">
                  <div class="detail-banner-title" :style="{ color: complianceConfig[archiveDetailResult.overall_compliance]?.color }">{{ getComplianceLabel(archiveDetailResult.overall_compliance) }}</div>
                  <div class="detail-banner-meta">{{ t('admin.data.overallScore') }} {{ archiveDetailResult.overall_score }}{{ t('admin.data.points') }} · {{ t('admin.data.duration') }} {{ archiveDetailResult.duration_ms }}ms</div>
                </div>
                <div class="detail-score" :style="{ color: complianceConfig[archiveDetailResult.overall_compliance]?.color }">{{ archiveDetailResult.overall_score }}</div>
              </div>
              <!-- Flow audit -->
              <div class="detail-section">
                <h4 class="detail-section-title">{{ t('admin.data.flowAudit') }}</h4>
                <div class="flow-status" :class="archiveDetailResult.flow_audit.is_complete ? 'flow-status--complete' : 'flow-status--incomplete'">
                  <CheckCircleOutlined v-if="archiveDetailResult.flow_audit.is_complete" style="color: var(--color-success);" />
                  <CloseCircleOutlined v-else style="color: var(--color-danger);" />
                  {{ archiveDetailResult.flow_audit.is_complete ? t('admin.data.flowComplete') : t('admin.data.flowIncomplete') }}
                  <span v-if="archiveDetailResult.flow_audit.missing_nodes.length" class="flow-missing">
                    · {{ t('admin.data.missingNodes') }}: {{ archiveDetailResult.flow_audit.missing_nodes.join(', ') }}
                  </span>
                </div>
                <div class="rule-checks">
                  <div v-for="node in archiveDetailResult.flow_audit.node_results" :key="node.node_id" class="rule-check-item" :class="{ 'rule-check-item--pass': node.compliant, 'rule-check-item--fail': !node.compliant }">
                    <div class="rule-check-status">
                      <CheckCircleOutlined v-if="node.compliant" style="color: var(--color-success);" />
                      <CloseCircleOutlined v-else style="color: var(--color-danger);" />
                    </div>
                    <div class="rule-check-content">
                      <div class="rule-check-name">{{ node.node_name }}</div>
                      <div class="rule-check-reasoning">{{ node.reasoning }}</div>
                    </div>
                  </div>
                </div>
              </div>
              <!-- Rule audit -->
              <div class="detail-section">
                <h4 class="detail-section-title">{{ t('admin.data.ruleAudit') }}</h4>
                <div class="rule-checks">
                  <div v-for="rule in archiveDetailResult.rule_audit" :key="rule.rule_id" class="rule-check-item" :class="{ 'rule-check-item--pass': rule.passed, 'rule-check-item--fail': !rule.passed }">
                    <div class="rule-check-status">
                      <CheckCircleOutlined v-if="rule.passed" style="color: var(--color-success);" />
                      <CloseCircleOutlined v-else style="color: var(--color-danger);" />
                    </div>
                    <div class="rule-check-content">
                      <div class="rule-check-name">{{ rule.rule_name }}</div>
                      <div class="rule-check-reasoning">{{ rule.reasoning }}</div>
                    </div>
                  </div>
                </div>
              </div>
              <!-- AI Summary -->
              <div class="detail-section">
                <h4 class="detail-section-title">{{ t('admin.data.aiSummary') }}</h4>
                <div class="ai-reasoning"><pre>{{ archiveDetailResult.ai_summary }}</pre></div>
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
  display: flex; gap: 4px; background: var(--color-bg-hover); padding: 4px;
  border-radius: var(--radius-lg); margin-bottom: 24px; width: fit-content;
}
.tab-btn {
  padding: 8px 20px; border: none; background: transparent; border-radius: var(--radius-md);
  font-size: 14px; font-weight: 500; color: var(--color-text-secondary); cursor: pointer;
  transition: all var(--transition-fast); display: flex; align-items: center; gap: 6px;
}
.tab-btn:hover { color: var(--color-text-primary); }
.tab-btn--active { background: var(--color-bg-card); color: var(--color-primary); box-shadow: var(--shadow-xs); }

/* Stats row - matching dashboard/archive pattern */
.stats-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; margin-bottom: 20px; }
.stat-card {
  background: var(--color-bg-card); border-radius: var(--radius-lg); padding: 20px;
  display: flex; align-items: center; gap: 16px; border: 2px solid var(--color-border-light);
  transition: all var(--transition-base);
}
.stat-card:hover { transform: translateY(-2px); box-shadow: var(--shadow-md); }
.stat-card--selected { box-shadow: var(--shadow-md); transform: translateY(-2px); }
.stat-card--selected.stat-card--primary { border-color: var(--color-primary); }
.stat-card--selected.stat-card--success { border-color: var(--color-success); }
.stat-card--selected.stat-card--danger { border-color: var(--color-danger); }
.stat-card--selected.stat-card--warning { border-color: var(--color-warning); }
.stat-card-icon {
  width: 48px; height: 48px; border-radius: var(--radius-lg);
  display: flex; align-items: center; justify-content: center; font-size: 22px; flex-shrink: 0;
}
.stat-card--primary .stat-card-icon { background: var(--color-primary-bg); color: var(--color-primary); }
.stat-card--success .stat-card-icon { background: var(--color-success-bg); color: var(--color-success); }
.stat-card--danger .stat-card-icon { background: var(--color-danger-bg); color: var(--color-danger); }
.stat-card--warning .stat-card-icon { background: var(--color-warning-bg); color: var(--color-warning); }
.stat-card-info { display: flex; flex-direction: column; }
.stat-card-value { font-size: 28px; font-weight: 700; color: var(--color-text-primary); line-height: 1.2; }
.stat-card-label { font-size: 13px; color: var(--color-text-tertiary); margin-top: 2px; }

.toolbar { display: flex; align-items: center; justify-content: space-between; margin-bottom: 12px; gap: 12px; flex-wrap: wrap; }
.toolbar-left { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.toolbar-right { display: flex; align-items: center; gap: 8px; }

/* Filter bar */
.filter-bar {
  display: flex; gap: 8px; padding: 12px 16px; background: var(--color-bg-page);
  border-radius: var(--radius-md); margin-bottom: 12px; flex-wrap: wrap; align-items: center;
}
.filter-toggle-btn--active { border-color: var(--color-primary) !important; color: var(--color-primary) !important; }
.filter-active-dot {
  width: 6px; height: 6px; border-radius: 50%; background: var(--color-primary);
  display: inline-block; margin-left: 4px;
}

/* Batch selected hint (inline in toolbar) */
.batch-selected-hint {
  font-size: 12px; font-weight: 500; color: var(--color-primary);
  padding: 2px 10px; border-radius: var(--radius-full);
  background: var(--color-primary-bg);
}

/* Data table */
.data-table-card {
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); overflow: hidden;
}
.data-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.data-table th {
  padding: 12px 16px; text-align: left; font-weight: 600; color: var(--color-text-secondary);
  background: var(--color-bg-page); border-bottom: 1px solid var(--color-border-light);
  font-size: 12px; text-transform: uppercase; letter-spacing: 0.04em; white-space: nowrap;
}
.data-table td {
  padding: 12px 16px; border-bottom: 1px solid var(--color-border-light);
  color: var(--color-text-primary);
}
.data-table tbody tr:hover { background: var(--color-bg-hover); }
.data-table tbody tr:last-child td { border-bottom: none; }
.text-secondary { color: var(--color-text-tertiary); }
.text-mono { font-family: monospace; font-size: 12px; color: var(--color-text-secondary); }
.empty-cell { text-align: center; padding: 32px 16px !important; color: var(--color-text-tertiary); }

/* Result tag (audit recommendation / compliance) */
.result-tag {
  font-size: 11px; font-weight: 600; padding: 3px 10px; border-radius: var(--radius-full);
  white-space: nowrap; display: inline-flex; align-items: center; gap: 4px;
}

/* Status tag (cron) */
.status-tag {
  font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: var(--radius-full);
  display: inline-flex; align-items: center; gap: 4px;
}
.status-tag--success { background: var(--color-success-bg); color: var(--color-success); }
.status-tag--failed { background: var(--color-danger-bg); color: var(--color-danger); }
.status-tag--running { background: var(--color-primary-bg); color: var(--color-primary); }

.action-btns { display: flex; gap: 4px; }
.icon-btn {
  width: 28px; height: 28px; border: 1px solid var(--color-border); background: transparent;
  border-radius: var(--radius-sm); cursor: pointer; display: flex; align-items: center;
  justify-content: center; color: var(--color-text-tertiary); transition: all var(--transition-fast);
}
.icon-btn:hover { border-color: var(--color-primary); color: var(--color-primary); }

.pagination-wrapper { padding: 16px 0; display: flex; justify-content: flex-end; }

/* Drawer (matching dashboard/archive pattern) */
.drawer-overlay {
  position: fixed; inset: 0; background: rgba(0, 0, 0, 0.45); z-index: 1000;
  display: flex; justify-content: flex-end;
}
.drawer-panel {
  width: 560px; max-width: 90vw; background: var(--color-bg-card);
  box-shadow: var(--shadow-xl); display: flex; flex-direction: column; height: 100%;
}
.drawer-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 16px 24px; border-bottom: 1px solid var(--color-border-light);
}
.drawer-header h3 { margin: 0; font-size: 16px; font-weight: 600; color: var(--color-text-primary); }
.drawer-close {
  width: 32px; height: 32px; border: none; background: transparent; border-radius: var(--radius-sm);
  cursor: pointer; display: flex; align-items: center; justify-content: center;
  color: var(--color-text-tertiary); transition: all var(--transition-fast);
}
.drawer-close:hover { background: var(--color-bg-hover); color: var(--color-text-primary); }
.drawer-body { flex: 1; overflow-y: auto; padding: 24px; }

.detail-process-title { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 16px; }

/* Detail banner (matching dashboard result-banner) */
.detail-banner {
  display: flex; align-items: center; gap: 16px; padding: 16px 20px;
  border-radius: var(--radius-lg); border: 1px solid; margin-bottom: 20px;
}
.detail-banner-info { flex: 1; }
.detail-banner-title { font-size: 16px; font-weight: 700; }
.detail-banner-meta { font-size: 12px; color: var(--color-text-tertiary); margin-top: 4px; }
.detail-score { font-size: 36px; font-weight: 800; line-height: 1; }

.detail-section { margin-bottom: 20px; }
.detail-section-title { font-size: 14px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 10px; }

/* Rule checks (matching dashboard pattern) */
.rule-checks { display: flex; flex-direction: column; gap: 8px; }
.rule-check-item {
  display: flex; gap: 10px; padding: 10px 14px; border-radius: var(--radius-md);
  border: 1px solid var(--color-border-light);
}
.rule-check-item--pass { background: var(--color-success-bg); border-color: rgba(16, 185, 129, 0.2); }
.rule-check-item--fail { background: var(--color-danger-bg); border-color: rgba(239, 68, 68, 0.2); }
.rule-check-status { font-size: 16px; flex-shrink: 0; padding-top: 1px; }
.rule-check-content { flex: 1; }
.rule-check-name { font-size: 13px; font-weight: 600; color: var(--color-text-primary); }
.rule-check-reasoning { font-size: 12px; color: var(--color-text-secondary); margin-top: 2px; }

/* Flow status */
.flow-status {
  display: flex; align-items: center; gap: 8px; padding: 10px 14px;
  border-radius: var(--radius-md); font-size: 13px; font-weight: 500; margin-bottom: 10px;
}
.flow-status--complete { background: var(--color-success-bg); color: var(--color-success); }
.flow-status--incomplete { background: var(--color-danger-bg); color: var(--color-danger); }
.flow-missing { font-weight: 400; }

/* Risk & Suggestions (matching dashboard pattern) */
.risk-suggest-row { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin-bottom: 20px; }
.insight-card { padding: 14px; border-radius: var(--radius-md); }
.insight-card--risk { background: var(--color-danger-bg); }
.insight-card--suggest { background: var(--color-primary-bg); }
.insight-card-header { display: flex; align-items: center; gap: 6px; font-size: 13px; font-weight: 600; margin-bottom: 8px; }
.insight-card-list { margin: 0; padding-left: 18px; font-size: 12px; color: var(--color-text-secondary); }
.insight-card-list li { margin-bottom: 4px; }

/* AI Reasoning */
.ai-reasoning {
  background: var(--color-bg-page); border-radius: var(--radius-md); padding: 14px;
  border: 1px solid var(--color-border-light);
}
.ai-reasoning pre {
  margin: 0; white-space: pre-wrap; word-break: break-word;
  font-size: 13px; line-height: 1.6; color: var(--color-text-secondary); font-family: var(--font-sans);
}

/* Transitions */
.slide-enter-active, .slide-leave-active { transition: all 0.2s ease; }
.slide-enter-from, .slide-leave-to { opacity: 0; max-height: 0; overflow: hidden; margin-bottom: 0; padding-top: 0; padding-bottom: 0; }
.slide-enter-to, .slide-leave-from { opacity: 1; max-height: 200px; }

.drawer-enter-active, .drawer-leave-active { transition: opacity 0.3s ease; }
.drawer-enter-active .drawer-panel, .drawer-leave-active .drawer-panel { transition: transform 0.3s ease; }
.drawer-enter-from { opacity: 0; }
.drawer-enter-from .drawer-panel { transform: translateX(100%); }
.drawer-leave-to { opacity: 0; }
.drawer-leave-to .drawer-panel { transform: translateX(100%); }

.fade-in { animation: fadeIn 0.3s ease-out; }

@media (max-width: 768px) {
  .stats-row { grid-template-columns: repeat(2, 1fr); }
  .data-table-card { overflow-x: auto; -webkit-overflow-scrolling: touch; }
  .data-table { min-width: 700px; }
  .toolbar { flex-direction: column; align-items: stretch; }
  .filter-bar { flex-direction: column; }
  .page-title { font-size: 20px; }
  .tab-nav { width: 100%; overflow-x: auto; -webkit-overflow-scrolling: touch; }
  .tab-btn { flex-shrink: 0; padding: 8px 14px; font-size: 13px; }
  .risk-suggest-row { grid-template-columns: 1fr; }
  .drawer-panel { width: 100%; max-width: 100vw; }
}
</style>
