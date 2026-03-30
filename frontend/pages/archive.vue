<script setup lang="ts">
import { marked } from 'marked'
import {
  SearchOutlined,
  FilterOutlined,
  DownloadOutlined,
  SafetyCertificateOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ExportOutlined,
  ThunderboltOutlined,
  ReloadOutlined,
  FileProtectOutlined,
  FieldTimeOutlined,
  LoadingOutlined,
  FireOutlined,
  AlertOutlined,
  BulbOutlined,
  RightOutlined,
  HistoryOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { useI18n } from '~/composables/useI18n'
import type {
  ArchiveReviewHistoryItem,
  ArchiveProcessItem,
  ArchiveProgressStep,
  ArchiveReviewResult,
  ArchiveReviewStats,
  ArchiveRuleAuditResult,
  ArchiveFieldAuditResult,
} from '~/types/archive-review'

definePageMeta({ middleware: 'auth' })

const { t } = useI18n()
const { token } = useAuth()
const {
  getStats,
  listProcesses: fetchArchiveProcesses,
  executeReview,
  waitArchiveJob,
  cancelArchiveJob,
  getArchiveResult,
  getArchiveHistory,
  getProcessTypes,
} = useArchiveReviewApi()

const asyncArchiveStatuses = ['pending', 'assembling', 'reasoning', 'extracting']

const stats = ref<ArchiveReviewStats>({
  total_count: 0,
  compliant_count: 0,
  partial_count: 0,
  non_compliant_count: 0,
  unaudited_count: 0,
  running_count: 0,
})
const processList = ref<ArchiveProcessItem[]>([])
const processCascaderOptions = ref<{ label: string; value: string; children: { label: string; value: string }[] }[]>([])
const listLoading = ref(false)

const selectedProcess = ref<ArchiveProcessItem | null>(null)
const searchText = ref('')
const searchApplicant = ref('')
const showFilters = ref(false)
const batchAuditing = ref(false)
const selectedProcessIds = ref<string[]>([])
const processAuditLoading = ref<Record<string, boolean>>({})
const pollProcessId = ref<string | null>(null)
const eventSourceStream = ref<EventSource | null>(null)
const reviewHistory = ref<ArchiveReviewHistoryItem[]>([])
const historyLoading = ref(false)
const selectedHistoryId = ref<string | null>(null)
const currentResult = ref<ArchiveReviewResult | null>(null)

//=====过滤器=====
const filterProcessType = ref<string[][]>([])
const filterProcessNames = computed(() => {
  if (filterProcessType.value.length === 0) return []
  const names: string[] = []
  for (const path of filterProcessType.value) {
    if (path.length >= 2) {
      names.push(path[path.length - 1])
    } else if (path.length === 1) {
      const cat = processCascaderOptions.value.find((o: any) => o.value === path[0])
      if (cat && (cat as any).children) {
        names.push(...(cat as any).children.map((c: any) => c.value))
      }
    }
  }
  return names
})
const filterDepartment = ref<string | undefined>(undefined)
const filterAuditStatus = ref<string | undefined>('unaudited')

const departmentOptions = computed(() => [...new Set(processList.value.map(p => p.department).filter(Boolean))])

const renderMarkdown = (md: string | undefined | null): string => {
  if (!md) return ''
  try {
    return marked.parse(md, { breaks: true, gfm: true }) as string
  } catch {
    return md
  }
}

const formatDuration = (ms: number | undefined | null): string => {
  if (!ms || ms <= 0) return '0ms'
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

const formatDate = (dateStr: string | undefined | null): string => {
  if (!dateStr) return ''
  try {
    const d = new Date(dateStr)
    if (isNaN(d.getTime())) return dateStr
    return d.toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
  } catch {
    return dateStr
  }
}

const hasActiveFilters = computed(() =>
  !!searchText.value || !!searchApplicant.value || filterProcessType.value.length > 0 || !!filterDepartment.value
)

const computedListTitle = computed(() => {
  if (!filterAuditStatus.value) return t('archive.archivedProcesses')
  if (filterAuditStatus.value === 'unaudited') return t('archive.statUnaudited')
  if (filterAuditStatus.value === 'compliant') return t('archive.statCompliant')
  if (filterAuditStatus.value === 'partially_compliant') return t('archive.statPartial')
  if (filterAuditStatus.value === 'non_compliant') return t('archive.statNonCompliant')
  return t('archive.archivedProcesses')
})

const clearFilters = () => {
  searchText.value = ''
  searchApplicant.value = ''
  filterProcessType.value = []
  filterDepartment.value = undefined
}

const isResultAsyncRunning = (result: ArchiveReviewResult | null | undefined) =>
  !!(result?.status && asyncArchiveStatuses.includes(result.status))

const defaultProgressSteps = (status?: string): ArchiveProgressStep[] => {
  const defs = [
    { key: 'pending', label: '排队中' },
    { key: 'assembling', label: '组装复盘提示词' },
    { key: 'reasoning', label: '推理分析' },
    { key: 'extracting', label: '结构化提取' },
  ]
  const phaseIdx: Record<string, number> = {
    pending: 0,
    assembling: 1,
    reasoning: 2,
    extracting: 3,
  }
  let current = phaseIdx[status || 'pending'] ?? 0
  if (status === 'completed') current = 3
  if (status === 'failed') current = 2

  const steps = defs.map((def, index) => {
    const step: ArchiveProgressStep = { key: def.key, label: def.label }
    if (status === 'failed' && index === current) step.failed = true
    else if (index < current) step.done = true
    else if (index === current && status !== 'completed') step.current = true
    return step
  })
  if (status === 'completed') {
    steps.push({ key: 'done', label: '已完成', done: true })
  }
  return steps
}

const normalizeArchiveResult = (input?: Partial<ArchiveReviewResult> | null): ArchiveReviewResult | null => {
  if (!input) return null
  return {
    id: input.id,
    trace_id: input.trace_id || '',
    process_id: input.process_id || '',
    title: input.title,
    process_type: input.process_type,
    status: input.status,
    overall_compliance: input.overall_compliance,
    overall_score: input.overall_score ?? 0,
    confidence: input.confidence ?? 0,
    duration_ms: input.duration_ms ?? 0,
    ai_reasoning: input.ai_reasoning || '',
    ai_summary: input.ai_summary || '',
    flow_audit: {
      is_complete: input.flow_audit?.is_complete ?? true,
      missing_nodes: input.flow_audit?.missing_nodes ?? [],
      node_results: input.flow_audit?.node_results ?? [],
    },
    field_audit: input.field_audit ?? [],
    rule_audit: input.rule_audit ?? [],
    risk_points: input.risk_points ?? [],
    suggestions: input.suggestions ?? [],
    created_at: input.created_at,
    updated_at: input.updated_at,
    error_message: input.error_message,
    parse_error: input.parse_error,
    raw_content: input.raw_content,
    process_snapshot: input.process_snapshot,
    progress_steps: input.progress_steps?.length ? input.progress_steps : defaultProgressSteps(input.status),
  }
}

const syncResultToList = (processId: string, result: ArchiveReviewResult | null) => {
  const item = processList.value.find(proc => proc.process_id === processId)
  if (!item) return
  item.archive_result = result
  item.archive_status = result?.status
  item.has_review = result?.status === 'completed'
  processAuditLoading.value = {
    ...processAuditLoading.value,
    [processId]: isResultAsyncRunning(result),
  }
}

const updateLiveResult = (processId: string, result: Partial<ArchiveReviewResult> | null | undefined) => {
  const normalized = normalizeArchiveResult(result)
  if (!normalized) return
  syncResultToList(processId, normalized)
  if (selectedProcess.value?.process_id === processId) {
    const oldReasoning = currentResult.value?.ai_reasoning || ''
    currentResult.value = normalized
    if (oldReasoning.length > (normalized.ai_reasoning?.length || 0) && currentResult.value) {
      currentResult.value.ai_reasoning = oldReasoning
    }
  }
}

// ===== 后端分页 =====
const listPage = ref(1)
const listPageSize = ref(20)
const listTotal = ref(0)
let searchDebounceTimer: ReturnType<typeof setTimeout> | null = null

const onListPageChange = (page: number, size: number) => {
  listPage.value = page
  listPageSize.value = size
  loadProcesses()
}

// 搜索条件变化时重新拉取第一页
const triggerSearch = () => {
  if (searchDebounceTimer) clearTimeout(searchDebounceTimer)
  searchDebounceTimer = setTimeout(() => {
    listPage.value = 1
    loadProcesses()
  }, 400)
}

watch([searchText, searchApplicant, filterProcessNames, filterDepartment, filterAuditStatus], () => {
  triggerSearch()
})

//=====选择=====
const toggleSelectProcess = (id: string) => {
  if (processAuditLoading.value[id]) return
  const idx = selectedProcessIds.value.indexOf(id)
  if (idx >= 0) selectedProcessIds.value.splice(idx, 1)
  else if (selectedProcessIds.value.length < 10) selectedProcessIds.value.push(id)
  else message.warning(t('archive.batchLimitHint'))
}

const selectableIdsComputed = computed(() =>
  processList.value
    .filter((proc: ArchiveProcessItem) => !proc.archive_status || !asyncArchiveStatuses.includes(proc.archive_status))
    .map((proc: ArchiveProcessItem) => proc.process_id),
)

const toggleSelectAll = () => {
  const selectableIds = selectableIdsComputed.value
  if (selectedProcessIds.value.length === Math.min(selectableIds.length, 10) || selectableIds.length === 0) {
    selectedProcessIds.value = []
  } else {
    selectedProcessIds.value = selectableIds.slice(0, 10)
  }
}

const disconnectStream = () => {
  if (eventSourceStream.value) {
    eventSourceStream.value.close()
    eventSourceStream.value = null
  }
}

const startSSE = (archiveLogId: string, processId: string) => {
  disconnectStream()
  const tokenVal = token.value || localStorage.getItem('token') || ''
  const config = useRuntimeConfig()
  const url = `${String(config.public.apiBase)}/api/archive/stream/${archiveLogId}?token=${encodeURIComponent(tokenVal)}`

  eventSourceStream.value = new EventSource(url)
  eventSourceStream.value.onmessage = (event) => {
    if (selectedProcess.value?.process_id !== processId || !currentResult.value) return
    currentResult.value.ai_reasoning = (currentResult.value.ai_reasoning || '') + event.data
  }
  eventSourceStream.value.onerror = () => {
    disconnectStream()
  }
}

const trackRunningJob = async (proc: ArchiveProcessItem) => {
  const archiveLogId = proc.archive_result?.id
  if (!archiveLogId || pollProcessId.value === proc.process_id) return

  pollProcessId.value = proc.process_id
  processAuditLoading.value = {
    ...processAuditLoading.value,
    [proc.process_id]: true,
  }
  if (selectedProcess.value?.process_id === proc.process_id) {
    startSSE(archiveLogId, proc.process_id)
  }

  try {
    const result = await waitArchiveJob(archiveLogId, (status) => {
      updateLiveResult(proc.process_id, status)
    })
    updateLiveResult(proc.process_id, result)
    await Promise.all([loadStats(), loadProcesses()])
  } catch {
    await Promise.all([loadStats(), loadProcesses()])
  } finally {
    if (pollProcessId.value === proc.process_id) {
      pollProcessId.value = null
    }
    if (selectedProcess.value?.process_id === proc.process_id) {
      disconnectStream()
    }
    processAuditLoading.value = {
      ...processAuditLoading.value,
      [proc.process_id]: false,
    }
  }
}

const selectProcess = (proc: ArchiveProcessItem) => {
  selectedProcess.value = proc
  selectedHistoryId.value = proc.archive_result?.status === 'completed' ? proc.archive_result.id || null : null
  currentResult.value = normalizeArchiveResult(proc.archive_result)
  loadHistory(proc.process_id)
  if (isResultAsyncRunning(currentResult.value)) {
    trackRunningJob(proc)
  } else {
    disconnectStream()
  }
}

const loading = computed(() => isResultAsyncRunning(currentResult.value))

const runArchiveReview = async (proc: ArchiveProcessItem) => {
  selectedHistoryId.value = null
  const pendingResult = normalizeArchiveResult({
    trace_id: '',
    process_id: proc.process_id,
    title: proc.title,
    process_type: proc.process_type,
    status: 'pending',
    ai_reasoning: '',
  })
  syncResultToList(proc.process_id, pendingResult)
  if (selectedProcess.value?.process_id === proc.process_id) {
    currentResult.value = pendingResult
  }

  let started = false
  try {
    const result = await executeReview({
      process_id: proc.process_id,
      process_type: proc.process_type,
      title: proc.title,
    }, (status) => {
      if (!started && status.id && selectedProcess.value?.process_id === proc.process_id) {
        startSSE(status.id, proc.process_id)
        started = true
      }
      updateLiveResult(proc.process_id, status)
    })
    updateLiveResult(proc.process_id, result)
    await Promise.all([loadStats(), loadProcesses()])
    return result
  } finally {
    if (selectedProcess.value?.process_id === proc.process_id && !isResultAsyncRunning(currentResult.value)) {
      disconnectStream()
    }
  }
}

const handleAudit = async () => {
  if (!selectedProcess.value) return
  const processId = selectedProcess.value.process_id
  processAuditLoading.value = {
    ...processAuditLoading.value,
    [processId]: true,
  }
  try {
    await runArchiveReview(selectedProcess.value)
  } catch (error: any) {
    message.error(error?.message || t('archive.auditFailed'))
    await Promise.all([loadStats(), loadProcesses()])
  } finally {
    processAuditLoading.value = {
      ...processAuditLoading.value,
      [processId]: false,
    }
  }
}

const handleReAudit = async () => {
  await handleAudit()
}

const batchAuditTotal = ref(0)
const batchAuditDone = ref(0)

//=====批量审核=====
const handleBatchAudit = async () => {
  if (selectedProcessIds.value.length === 0 || selectedProcessIds.value.length > 10) {
    if (selectedProcessIds.value.length > 10) {
      message.warning(t('archive.batchLimitHint'))
    }
    return
  }
  batchAuditing.value = true
  const ids = [...selectedProcessIds.value]
  batchAuditTotal.value = ids.length
  batchAuditDone.value = 0

  for (let i = 0; i < ids.length; i++) {
    const id = ids[i]
    const proc = processList.value.find(p => p.process_id === id)
    if (!proc) {
      batchAuditDone.value = i + 1
      continue
    }

    processAuditLoading.value = {
      ...processAuditLoading.value,
      [id]: true,
    }

    try {
      await runArchiveReview(proc)
    } catch {
    }

    processAuditLoading.value = {
      ...processAuditLoading.value,
      [id]: false,
    }
    batchAuditDone.value = i + 1
  }

  batchAuditing.value = false
  selectedProcessIds.value = []
  await Promise.all([loadStats(), loadProcesses()])
  message.success(t('archive.batchDone', `${batchAuditDone.value}`))
}

//=====导出（仅在选择流程后显示）=====
const handleExport = async (format: string) => {
  if (!selectedProcess.value || !currentResult.value) {
    message.warning(t('archive.noResultToExport'))
    return
  }
  if (format !== 'json') {
    message.info(t('archive.exportFormatPending', format.toUpperCase()))
    return
  }

  const fullResult = currentResult.value.id
    ? normalizeArchiveResult(await getArchiveResult(currentResult.value.id))
    : currentResult.value
  const payload = {
    process: selectedProcess.value,
    result: fullResult,
  }
  const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `archive-review-${selectedProcess.value.process_id}.json`
  link.click()
  URL.revokeObjectURL(url)
  message.success(t('archive.exportJsonReady'))
}

const jumpToOA = (processId: string) => {
  message.info(t('archive.jumpingToOA', processId))
}

const handleCancelAudit = async () => {
  if (!currentResult.value?.id || !selectedProcess.value) return
  try {
    await cancelArchiveJob(currentResult.value.id)
    message.success(t('archive.cancelSuccess'))
    await Promise.all([loadStats(), loadProcesses()])
    const next = processList.value.find(proc => proc.process_id === selectedProcess.value?.process_id)
    if (next) {
      selectProcess(next)
    }
  } catch (error: any) {
    message.error(error?.message || t('archive.cancelFailed'))
  }
}

const loadHistory = async (processId: string) => {
  historyLoading.value = true
  try {
    reviewHistory.value = await getArchiveHistory(processId)
  } catch {
    reviewHistory.value = []
  } finally {
    historyLoading.value = false
  }
}

const handleSelectHistory = async (item: ArchiveReviewHistoryItem) => {
  selectedHistoryId.value = item.id
  try {
    const result = normalizeArchiveResult(await getArchiveResult(item.id))
    if (result) {
      currentResult.value = result
    }
  } catch (error: any) {
    message.error(error?.message || t('archive.loadFailed'))
  }
}

const loadStats = async () => {
  try {
    stats.value = await getStats()
  } catch {}
}

const loadProcessTypes = async () => {
  try {
    const list = await getProcessTypes()
    const categoryMap = new Map<string, { label: string; value: string; children: { label: string; value: string }[] }>()
    for (const item of list || []) {
      const categoryLabel = item.process_type_label || item.process_type
      if (!categoryMap.has(categoryLabel)) {
        categoryMap.set(categoryLabel, { label: categoryLabel, value: categoryLabel, children: [] })
      }
      const category = categoryMap.get(categoryLabel)!
      if (!category.children.some(child => child.value === item.process_type)) {
        category.children.push({ label: item.process_type, value: item.process_type })
      }
    }
    processCascaderOptions.value = Array.from(categoryMap.values())
  } catch {}
}

const loadProcesses = async () => {
  listLoading.value = true
  try {
    const pt = filterProcessNames.value.length === 1 ? filterProcessNames.value[0] : ''
    const response = await fetchArchiveProcesses({
      keyword: searchText.value || undefined,
      applicant: searchApplicant.value || undefined,
      process_type: pt || undefined,
      department: filterDepartment.value || undefined,
      audit_status: filterAuditStatus.value || undefined,
      page: listPage.value,
      page_size: listPageSize.value,
    })
    processList.value = Array.isArray(response) ? response : (response?.items ?? [])
    listTotal.value = (response as any)?.total ?? processList.value.length
    selectedProcessIds.value = selectedProcessIds.value.filter(id => selectableIdsComputed.value.includes(id))

    if (!selectedProcess.value) return
    const nextSelected = processList.value.find(proc => proc.process_id === selectedProcess.value?.process_id) || null
    selectedProcess.value = nextSelected
    if (selectedHistoryId.value) return
    currentResult.value = normalizeArchiveResult(nextSelected?.archive_result)
    if (nextSelected && isResultAsyncRunning(currentResult.value)) {
      trackRunningJob(nextSelected)
    }
  } catch {
    processList.value = []
    message.error(t('archive.loadFailed'))
  } finally {
    listLoading.value = false
  }
}

const filteredProgressSteps = computed(() => {
  if (!currentResult.value) return []
  const steps = currentResult.value.progress_steps?.length
    ? currentResult.value.progress_steps
    : defaultProgressSteps(currentResult.value.status)
  return steps.filter(step => step.key !== 'pending')
})

//=====配置助手=====
const complianceConfig = computed((): Record<string, { color: string; bg: string; label: string }> => ({
  compliant: { color: 'var(--color-success)', bg: 'var(--color-success-bg)', label: t('archive.compliant') },
  non_compliant: { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)', label: t('archive.nonCompliant') },
  partially_compliant: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', label: t('archive.partiallyCompliant') },
}))

const auditedCount = computed(() => processList.value.filter(p => !!p.archive_result?.overall_compliance).length)

onMounted(async () => {
  await Promise.all([loadProcessTypes(), loadStats(), loadProcesses()])
})

onUnmounted(() => {
  disconnectStream()
})
</script>

<template>
  <div class="archive-page fade-in">
    <!--页眉-->
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('archive.title') }}</h1>
        <p class="page-subtitle">{{ t('archive.subtitle') }}</p>
      </div>
      <div class="page-header-actions">
        <!--导出按钮：仅在选择进程时显示-->
        <a-dropdown v-if="selectedProcess">
          <a-button>
            <DownloadOutlined /> {{ t('archive.exportReport') }}
          </a-button>
          <template #overlay>
            <a-menu>
              <a-menu-item key="json" @click="handleExport('json')">{{ t('archive.exportJSON') }}</a-menu-item>
              <a-menu-item key="csv" @click="handleExport('csv')">{{ t('archive.exportCSV') }}</a-menu-item>
              <a-menu-item key="excel" @click="handleExport('excel')">{{ t('archive.exportExcel') }}</a-menu-item>
            </a-menu>
          </template>
        </a-dropdown>
      </div>
    </div>

    <!--统计行-->
    <div class="stats-row">
      <div class="stat-card stat-card--primary" :class="{ 'stat-card--selected': filterAuditStatus === 'unaudited' }" @click="filterAuditStatus = filterAuditStatus === 'unaudited' ? undefined : 'unaudited'">
        <div class="stat-card-icon"><FileProtectOutlined /></div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ stats.unaudited_count }}</span>
          <span class="stat-card-label">{{ t('archive.statUnaudited') }}</span>
        </div>
      </div>
      <div class="stat-card stat-card--success" :class="{ 'stat-card--selected': filterAuditStatus === 'compliant' }" @click="filterAuditStatus = filterAuditStatus === 'compliant' ? undefined : 'compliant'">
        <div class="stat-card-icon"><CheckCircleOutlined /></div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ stats.compliant_count }}</span>
          <span class="stat-card-label">{{ t('archive.statCompliant') }}</span>
        </div>
      </div>
      <div class="stat-card stat-card--warning" :class="{ 'stat-card--selected': filterAuditStatus === 'partially_compliant' }" @click="filterAuditStatus = filterAuditStatus === 'partially_compliant' ? undefined : 'partially_compliant'">
        <div class="stat-card-icon"><AlertOutlined /></div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ stats.partial_count }}</span>
          <span class="stat-card-label">{{ t('archive.statPartial') }}</span>
        </div>
      </div>
      <div class="stat-card stat-card--danger" :class="{ 'stat-card--selected': filterAuditStatus === 'non_compliant' }" @click="filterAuditStatus = filterAuditStatus === 'non_compliant' ? undefined : 'non_compliant'">
        <div class="stat-card-icon"><CloseCircleOutlined /></div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ stats.non_compliant_count }}</span>
          <span class="stat-card-label">{{ t('archive.statNonCompliant') }}</span>
        </div>
      </div>
    </div>

    <!--主要布局-->
    <div class="archive-grid">
      <!--左：进程列表-->
      <div class="list-panel">
        <div class="panel-header">
          <div class="panel-header-row">
            <h3 class="panel-title">
              <FireOutlined style="color: var(--color-primary);" />
              {{ computedListTitle }}
              <a-badge :count="listTotal" :number-style="{ backgroundColor: 'var(--color-primary)' }" />
            </h3>
            <a-button size="small" @click="showFilters = !showFilters" :class="{ 'filter-toggle-btn--active': hasActiveFilters }">
              <FilterOutlined /> {{ t('archive.filter') }}
              <span v-if="hasActiveFilters" class="filter-active-dot" />
            </a-button>
          </div>

          <!--过滤器-->
          <transition name="slide">
            <div v-if="showFilters" class="filter-bar">
              <a-input v-model:value="searchText" :placeholder="t('archive.searchPlaceholder')" allow-clear style="flex: 2; min-width: 160px;">
                <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
              </a-input>
              <a-input v-model:value="searchApplicant" :placeholder="t('archive.searchApplicant')" allow-clear style="flex: 1; min-width: 130px;">
                <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
              </a-input>
              <a-cascader
                v-model:value="filterProcessType"
                :options="processCascaderOptions"
                :placeholder="t('archive.processType')"
                multiple
                :max-tag-count="1"
                allow-clear
                style="flex: 1.5; min-width: 160px;"
                :show-search="{ filter: (inputValue: string, path: any[]) => path.some((o: any) => o.label.toLowerCase().includes(inputValue.toLowerCase())) }"
              />
              <a-select v-model:value="filterDepartment" :placeholder="t('archive.department')" allow-clear style="flex: 1; min-width: 120px;">
                <a-select-option v-for="d in departmentOptions" :key="d" :value="d">{{ d }}</a-select-option>
              </a-select>

              <a-button size="small" @click="clearFilters">{{ t('archive.reset') }}</a-button>
            </div>
          </transition>

          <!--批处理工具栏-->
          <div class="batch-toolbar">
            <div class="batch-toolbar-left">
              <a-checkbox
                :checked="selectedProcessIds.length === Math.min(selectableIdsComputed.length, 10) && selectableIdsComputed.length > 0"
                :indeterminate="selectedProcessIds.length > 0 && selectedProcessIds.length < Math.min(selectableIdsComputed.length, 10)"
                @change="toggleSelectAll"
              >
                {{ selectedProcessIds.length > 0 ? t('archive.selected', `${selectedProcessIds.length}`) : t('archive.selectAll') }}
              </a-checkbox>
              <span v-if="batchAuditing" class="batch-progress-hint">
                {{ t('archive.auditedProgress', `${batchAuditDone}/${batchAuditTotal}`) }}
              </span>
              <span v-else-if="auditedCount > 0" class="panel-header-hint">{{ t('archive.reviewed') }} {{ auditedCount }}/{{ listTotal }}</span>
            </div>
            <a-button v-if="selectedProcessIds.length > 0" type="primary" size="small" :disabled="batchAuditing" @click="handleBatchAudit" class="batch-audit-btn">
              <LoadingOutlined v-if="batchAuditing" />
              <ThunderboltOutlined v-else />
              {{ t('archive.batchAudit') }}
            </a-button>
          </div>
        </div>

        <!--进程列表-->
        <div class="process-list">
          <div v-if="listLoading" class="list-empty" style="display: flex; justify-content: center; align-items: center; padding: 40px 0;">
            <a-spin />
          </div>
          <template v-else>
            <div
              v-for="proc in processList"
              :key="proc.process_id"
              class="process-item"
              :class="{
                'process-item--selected': selectedProcess?.process_id === proc.process_id,
                'process-item--compliant': proc.archive_result?.overall_compliance === 'compliant',
                'process-item--partial': proc.archive_result?.overall_compliance === 'partially_compliant',
                'process-item--noncompliant': proc.archive_result?.overall_compliance === 'non_compliant',
              }"
              @click="selectProcess(proc)"
            >
              <div class="process-item-checkbox" @click.stop="toggleSelectProcess(proc.process_id)">
                <a-checkbox :checked="selectedProcessIds.includes(proc.process_id)" :disabled="processAuditLoading[proc.process_id]" />
              </div>
              <div class="process-item-main">
                <div class="process-item-title-row">
                  <span class="process-item-title">{{ proc.title }}</span>
                  <span
                    v-if="proc.archive_result?.overall_compliance"
                    class="process-audit-badge"
                    :style="{
                      color: complianceConfig[proc.archive_result.overall_compliance]?.color,
                      background: complianceConfig[proc.archive_result.overall_compliance]?.bg,
                    }"
                  >
                    <SafetyCertificateOutlined />
                    {{ complianceConfig[proc.archive_result.overall_compliance]?.label }}
                    {{ proc.archive_result.overall_score }}{{ t('archive.score') }} · {{ formatDuration(proc.archive_result.duration_ms) }}
                  </span>
                </div>
                <div class="process-item-meta">
                  <span>{{ proc.applicant }}</span>
                  <span class="meta-dot">·</span>
                  <span>{{ proc.department }}</span>
                  <span class="meta-dot">·</span>
                  <span>{{ proc.submit_time }}</span>
                </div>
                <div class="process-item-footer">
                  <span class="process-type-tag">{{ proc.process_type }}</span>
                  <span v-if="processAuditLoading[proc.process_id]" class="process-auditing" style="display: inline-flex; align-items: center; gap: 4px;">
                    <LoadingOutlined style="font-size: 11px;" /> {{ t('archive.auditingItem') }}
                  </span>
                  <a-tooltip :title="t('archive.jumpOA')" :mouse-enter-delay="0.5">
                    <button class="oa-jump-btn" @click.stop="jumpToOA(proc.process_id)">
                      <ExportOutlined />
                    </button>
                  </a-tooltip>
                </div>
              </div>
            </div>
            <div v-if="processList.length === 0" class="list-empty">
              <a-empty :description="t('archive.noMatch')" />
            </div>
          </template>
        </div>

        <!--分页-->
        <div class="pagination-wrapper">
          <a-pagination
            :current="listPage"
            :page-size="listPageSize"
            :total="listTotal"
            size="small"
            show-size-changer
            show-quick-jumper
            :page-size-options="['10', '20', '50']"
            @change="onListPageChange"
            @showSizeChange="onListPageChange"
          />
        </div>
      </div>

      <!--右：细节面板-->
      <div class="detail-panel">
        <div class="panel-header">
          <h3 class="panel-title">
            <SafetyCertificateOutlined style="color: var(--color-primary);" />
            {{ t('archive.complianceTitle') }}
          </h3>
        </div>

        <div class="detail-content">
          <!--空状态-->
          <div v-if="!selectedProcess" class="detail-empty">
            <div class="detail-empty-icon"><SafetyCertificateOutlined /></div>
            <h4>{{ t('archive.selectProcess') }}</h4>
            <p>{{ t('archive.selectProcessDesc') }}</p>
          </div>

          <template v-else>
            <!--流程信息卡-->
            <div class="process-info-card">
              <div class="process-info-header">
                <div>
                  <h4 class="process-info-title">{{ selectedProcess.title }}</h4>
                  <div class="process-info-meta">
                    {{ selectedProcess.applicant }} · {{ selectedProcess.department }} · {{ selectedProcess.process_type }}
                  </div>
                  <div class="process-info-meta" style="margin-top: 4px;">
                    <FieldTimeOutlined /> {{ t('archive.submitLabel') }}: {{ selectedProcess.submit_time }}
                    &nbsp;→&nbsp; {{ t('archive.archiveLabel') }}: {{ selectedProcess.archive_time }}
                  </div>
                </div>
                <div class="process-info-actions">
                  <a-button @click="jumpToOA(selectedProcess.process_id)">
                    <ExportOutlined /> OA
                  </a-button>
                  <a-button v-if="loading && currentResult?.id" danger @click="handleCancelAudit">
                    <CloseCircleOutlined /> {{ t('archive.cancelReview') }}
                  </a-button>
                  <a-button type="primary" :loading="loading" @click="currentResult ? handleReAudit() : handleAudit()" style="display: inline-flex; align-items: center; gap: 6px;">
                    <template v-if="currentResult && !loading">
                      <ReloadOutlined /> {{ t('archive.reAudit') }}
                    </template>
                    <template v-else-if="!loading">
                      <ThunderboltOutlined /> {{ t('archive.startAudit') }}
                    </template>
                  </a-button>
                </div>
              </div>
            </div>

            <!--审核正在进行中-->
            <template v-if="loading && currentResult">
              <div class="audit-progress">
                <div
                  v-for="step in filteredProgressSteps"
                  :key="step.key"
                  class="audit-phase"
                  :class="{
                    'audit-phase--done': step.done,
                    'audit-phase--active': step.current,
                    'audit-phase--failed': step.failed,
                    'audit-phase--pending': !step.done && !step.current && !step.failed,
                  }"
                >
                  <div class="audit-phase-dot">
                    <LoadingOutlined v-if="step.current" />
                    <CloseCircleOutlined v-else-if="step.failed" style="color: var(--color-danger);" />
                    <CheckCircleOutlined v-else-if="step.done" style="color: var(--color-success);" />
                    <div v-else class="phase-pending-dot" />
                  </div>
                  <div class="audit-phase-info">
                    <div class="audit-phase-title">{{ step.label }}</div>
                    <div class="audit-phase-desc">{{ t('archive.aiAuditing') }}</div>
                  </div>
                </div>
                <div v-if="currentResult.ai_reasoning" class="ai-summary markdown-body">
                  <div v-html="renderMarkdown(currentResult.ai_reasoning)" />
                </div>
                <div v-else class="audit-check-empty">
                  {{ t('archive.aiAuditing') }}
                </div>
              </div>
            </template>

            <!--审核结果-->
            <template v-if="currentResult && !loading">
              <!--合规横幅-->
              <div
                class="compliance-banner"
                :style="{
                  background: complianceConfig[currentResult.overall_compliance ?? '']?.bg,
                  borderColor: complianceConfig[currentResult.overall_compliance ?? '']?.color,
                }"
              >
                <SafetyCertificateOutlined
                  class="compliance-banner-icon"
                  :style="{ color: complianceConfig[currentResult.overall_compliance ?? '']?.color }"
                />
                <div class="compliance-banner-info">
                  <div class="compliance-banner-title" :style="{ color: complianceConfig[currentResult.overall_compliance ?? '']?.color }">
                    {{ complianceConfig[currentResult.overall_compliance ?? '']?.label }}
                  </div>
                  <div class="compliance-banner-meta">
                    {{ t('archive.overallScore') }} {{ currentResult.overall_score }} {{ t('archive.score') }}
                    · {{ t('archive.durationLabel') }} {{ formatDuration(currentResult.duration_ms) }}
                  </div>
                </div>
                <div class="compliance-score" :style="{ color: complianceConfig[currentResult.overall_compliance ?? '']?.color }">
                  {{ currentResult.overall_score }}
                </div>
              </div>

              <div v-if="currentResult.error_message" class="section-block">
                <div class="audit-check-item audit-check-item--fail">
                  <div class="audit-check-status">
                    <CloseCircleOutlined style="color: var(--color-danger);" />
                  </div>
                  <div class="audit-check-content">
                    <div class="audit-check-name">{{ t('archive.auditFailed') }}</div>
                    <div class="audit-check-reasoning">{{ currentResult.error_message }}</div>
                  </div>
                </div>
              </div>

              <!--规则检查-->
              <div class="section-block">
                <h4 class="section-title"><SafetyCertificateOutlined /> {{ t('archive.ruleAudit') }}</h4>
                <div class="audit-checks">
                  <div
                    v-for="ra in currentResult.rule_audit"
                    :key="ra.rule_id"
                    class="audit-check-item"
                    :class="ra.passed ? 'audit-check-item--pass' : 'audit-check-item--fail'"
                  >
                    <div class="audit-check-status">
                      <CheckCircleOutlined v-if="ra.passed" style="color: var(--color-success);" />
                      <CloseCircleOutlined v-else style="color: var(--color-danger);" />
                    </div>
                    <div class="audit-check-content">
                      <div class="audit-check-name">{{ ra.rule_name }}</div>
                      <div class="audit-check-reasoning">{{ ra.reasoning }}</div>
                    </div>
                  </div>
                  <div v-if="!currentResult.rule_audit?.length" class="audit-check-empty">
                    {{ t('archive.noRules') }}
                  </div>
                </div>
              </div>

              <!--风险点及建议-->
              <div v-if="currentResult.overall_compliance !== 'compliant'" class="risk-suggestions-row">
                <div class="risk-card">
                  <h4 class="section-title"><AlertOutlined style="color: var(--color-danger);" /> {{ t('archive.riskPoints') }}</h4>
                  <div v-if="(currentResult.risk_points?.length ?? 0) > 0" class="risk-list">
                    <div v-for="(rp, i) in currentResult.risk_points" :key="'rp-'+i" class="risk-item">
                      <CloseCircleOutlined style="color: var(--color-danger); flex-shrink: 0;" />
                      <span>{{ rp }}</span>
                    </div>
                  </div>
                  <div v-else-if="currentResult.flow_audit?.missing_nodes?.length" class="risk-list">
                    <div v-for="node in currentResult.flow_audit?.missing_nodes" :key="node" class="risk-item">
                      <CloseCircleOutlined style="color: var(--color-danger); flex-shrink: 0;" />
                      <span>{{ t('archive.missingNode') }}: {{ node }}</span>
                    </div>
                  </div>
                  <div v-else class="risk-list">
                    <div v-for="(ra, i) in currentResult.rule_audit?.filter((r: ArchiveRuleAuditResult) => !r.passed) ?? []" :key="i" class="risk-item">
                      <CloseCircleOutlined style="color: var(--color-danger); flex-shrink: 0;" />
                      <span>{{ ra.rule_name }}: {{ ra.reasoning }}</span>
                    </div>
                  </div>
                  <div v-if="!currentResult.risk_points?.length && !currentResult.flow_audit?.missing_nodes?.length && !(currentResult.rule_audit?.filter((r: ArchiveRuleAuditResult) => !r.passed)?.length)" class="risk-empty">
                    {{ t('archive.noRiskPoints') }}
                  </div>
                </div>
                <div class="suggestion-card">
                  <h4 class="section-title"><BulbOutlined style="color: var(--color-warning);" /> {{ t('archive.suggestions') }}</h4>
                  <div class="suggestion-list">
                    <template v-if="(currentResult.suggestions?.length ?? 0) > 0">
                      <div v-for="(sg, i) in currentResult.suggestions" :key="'sg-'+i" class="suggestion-item">
                        <RightOutlined style="color: var(--color-warning); flex-shrink: 0;" />
                        <span>{{ sg }}</span>
                      </div>
                    </template>
                    <template v-else>
                      <div v-for="(fa, i) in currentResult.field_audit?.filter((f: ArchiveFieldAuditResult) => !f.passed) ?? []" :key="i" class="suggestion-item">
                        <RightOutlined style="color: var(--color-warning); flex-shrink: 0;" />
                        <span>{{ fa.reasoning }}</span>
                      </div>
                      <div v-if="!currentResult.field_audit?.filter((f: ArchiveFieldAuditResult) => !f.passed)?.length" class="suggestion-item">
                        <RightOutlined style="color: var(--color-warning); flex-shrink: 0;" />
                        <span>{{ t('archive.reviewSuggestion') }}</span>
                      </div>
                    </template>
                  </div>
                </div>
              </div>

              <!--人工智能总结-->
              <div class="section-block">
                <h4 class="section-title"><ThunderboltOutlined /> {{ t('archive.aiSummary') }}</h4>
                <div class="ai-summary markdown-body">
                  <div v-html="renderMarkdown(currentResult.ai_summary || currentResult.ai_reasoning)" />
                </div>
              </div>
            </template>

            <!--还没有结果（未加载）-->
            <div v-if="!currentResult && !loading" class="no-result-hint">
              <HistoryOutlined style="font-size: 32px; color: var(--color-text-tertiary);" />
              <p>{{ t('archive.noResultHint') }}</p>
            </div>

            <div class="section-block">
              <h4 class="section-title"><HistoryOutlined /> {{ t('archive.historyTitle') }}</h4>
              <div v-if="historyLoading" class="audit-check-empty">
                <a-spin size="small" />
              </div>
              <div v-else-if="reviewHistory.length" class="history-list">
                <button
                  v-for="item in reviewHistory"
                  :key="item.id"
                  class="history-item"
                  :class="{ 'history-item--active': selectedHistoryId === item.id }"
                  @click="handleSelectHistory(item)"
                >
                  <span class="history-item-title">
                    <span>{{ item.user_name || item.title }}</span>
                    <span class="history-item-time">{{ formatDate(item.created_at) }}</span>
                  </span>
                  <span class="history-item-meta">
                    <span
                      class="process-audit-badge"
                      :style="{
                        color: complianceConfig[item.compliance]?.color,
                        background: complianceConfig[item.compliance]?.bg,
                      }"
                    >
                      {{ complianceConfig[item.compliance]?.label }}
                      {{ item.compliance_score }}{{ t('archive.score') }}
                    </span>
                  </span>
                </button>
              </div>
              <div v-else class="audit-check-empty">
                {{ t('archive.noHistory') }}
              </div>
            </div>
          </template>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.archive-page { animation: fadeIn 0.3s ease-out; }
@keyframes fadeIn { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

.page-header {
  display: flex; justify-content: space-between; align-items: flex-start;
  margin-bottom: 20px; flex-wrap: wrap; gap: 16px;
}
.page-title { font-size: 24px; font-weight: 700; color: var(--color-text-primary); margin: 0; letter-spacing: -0.02em; }
.page-subtitle { font-size: 14px; color: var(--color-text-tertiary); margin: 4px 0 0; }
.page-header-actions { display: flex; gap: 8px; align-items: center; }

/*统计行*/
.stats-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; margin-bottom: 20px; }
.stat-card {
  display: flex; align-items: center; gap: 12px; padding: 14px 18px;
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); cursor: pointer;
  transition: all var(--transition-fast);
}
.stat-card:hover { box-shadow: var(--shadow-md); transform: translateY(-1px); }
.stat-card--selected { border-width: 2px; }
.stat-card--primary.stat-card--selected { border-color: var(--color-primary); }
.stat-card--success.stat-card--selected { border-color: var(--color-success); }
.stat-card--warning.stat-card--selected { border-color: var(--color-warning); }
.stat-card--danger.stat-card--selected { border-color: var(--color-danger); }
.stat-card-icon { font-size: 22px; }
.stat-card--primary .stat-card-icon { color: var(--color-primary); }
.stat-card--success .stat-card-icon { color: var(--color-success); }
.stat-card--warning .stat-card-icon { color: var(--color-warning); }
.stat-card--danger .stat-card-icon { color: var(--color-danger); }
.stat-card-info { display: flex; flex-direction: column; }
.stat-card-value { font-size: 22px; font-weight: 700; color: var(--color-text-primary); line-height: 1.2; }
.stat-card-label { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }

/*主格*/
.archive-grid { display: grid; grid-template-columns: 400px 1fr; gap: 20px; align-items: start; }

/*面板*/
.list-panel, .detail-panel {
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); overflow: hidden;
}
.panel-header {
  padding: 14px 18px; border-bottom: 1px solid var(--color-border-light);
  display: flex; flex-direction: column; gap: 10px;
}
.panel-header-row { display: flex; justify-content: space-between; align-items: center; }
.panel-title {
  font-size: 15px; font-weight: 600; color: var(--color-text-primary);
  margin: 0; display: flex; align-items: center; gap: 8px;
}
.panel-header-hint { font-size: 12px; color: var(--color-text-tertiary); }
.filter-toggle-btn--active { color: var(--color-primary); border-color: var(--color-primary); }
.filter-active-dot {
  display: inline-block; width: 6px; height: 6px; border-radius: 50%;
  background: var(--color-primary); margin-left: 4px; vertical-align: middle;
}

/*过滤栏*/
.filter-bar {
  display: flex; gap: 8px; align-items: center; flex-wrap: wrap;
  padding: 10px 0 2px;
}
.slide-enter-active, .slide-leave-active { transition: all 0.2s ease; }
.slide-enter-from, .slide-leave-to { opacity: 0; transform: translateY(-6px); }

/*批处理工具栏*/
.batch-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 10px; }
.batch-toolbar-left { display: flex; align-items: center; gap: 12px; }
.batch-progress-hint {
  font-size: 12px; font-weight: 600; color: var(--color-primary);
  animation: batchPulse 1.5s ease-in-out infinite;
}
@keyframes batchPulse { 0%, 100% { opacity: 0.6; } 50% { opacity: 1; } }
.batch-audit-btn { flex-shrink: 0; }

/*进程列表*/
.process-list { max-height: calc(100vh - 340px); overflow-y: auto; }
.process-item {
  display: flex; align-items: flex-start; padding: 12px 16px; cursor: pointer;
  transition: all var(--transition-fast); border-bottom: 1px solid var(--color-border-light); gap: 10px;
}
.process-item:last-child { border-bottom: none; }
.process-item:hover { background: var(--color-bg-hover); }
.process-item--selected { background: var(--color-primary-bg); border-left: 3px solid var(--color-primary); }
.process-item--compliant { border-left: 3px solid var(--color-success); }
.process-item--partial { border-left: 3px solid var(--color-warning); }
.process-item--noncompliant { border-left: 3px solid var(--color-danger); }
.process-item-checkbox { padding-top: 2px; flex-shrink: 0; }
.process-item-main { flex: 1; min-width: 0; }
.process-item-title-row { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; flex-wrap: wrap; }
.process-item-title {
  font-size: 13px; font-weight: 500; color: var(--color-text-primary);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 180px;
}
.process-audit-badge {
  display: inline-flex; align-items: center; gap: 3px;
  font-size: 11px; font-weight: 600; padding: 1px 7px; border-radius: var(--radius-full); white-space: nowrap; flex-shrink: 0;
}
.process-item-meta {
  font-size: 12px; color: var(--color-text-tertiary);
  display: flex; align-items: center; gap: 4px; flex-wrap: wrap; margin-bottom: 4px;
}
.meta-dot { color: var(--color-border); }
.process-item-footer { display: flex; align-items: center; gap: 6px; }
.process-type-tag {
  font-size: 11px; padding: 1px 8px; border-radius: var(--radius-full);
  background: var(--color-bg-hover); color: var(--color-text-tertiary); border: 1px solid var(--color-border-light);
}
.process-auditing { font-size: 11px; color: var(--color-primary); display: flex; align-items: center; gap: 4px; }
.oa-jump-btn { margin-left: auto; color: var(--color-text-tertiary); padding: 0 4px; height: 22px; }
.oa-jump-btn:hover { color: var(--color-primary); }
.list-empty { padding: 40px 20px; }

/*分页*/
.pagination-wrapper { padding: 12px 16px; border-top: 1px solid var(--color-border-light); display: flex; justify-content: flex-end; }

/*细节面板*/
.detail-content { padding: 18px; max-height: calc(100vh - 220px); overflow-y: auto; }
.detail-empty { text-align: center; padding: 60px 20px; }
.detail-empty-icon {
  width: 64px; height: 64px; border-radius: 50%; background: var(--color-primary-bg);
  color: var(--color-primary); font-size: 28px; display: flex; align-items: center;
  justify-content: center; margin: 0 auto 16px;
}
.detail-empty h4 { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin: 0 0 8px; }
.detail-empty p { font-size: 13px; color: var(--color-text-tertiary); margin: 0 auto; max-width: 300px; }

/*流程信息卡*/
.process-info-card {
  padding: 14px 16px; background: var(--color-bg-page);
  border-radius: var(--radius-lg); border: 1px solid var(--color-border-light); margin-bottom: 16px;
}
.process-info-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; }
.process-info-title { font-size: 15px; font-weight: 600; color: var(--color-text-primary); margin: 0 0 6px; }
.process-info-meta { font-size: 12px; color: var(--color-text-tertiary); }
.process-info-actions { display: flex; gap: 8px; flex-shrink: 0; }

/*审核进度*/
.audit-progress {
  display: flex; flex-direction: column; gap: 12px; padding: 20px;
  background: var(--color-bg-page); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); margin-bottom: 16px;
}
.audit-phase { display: flex; align-items: flex-start; gap: 12px; }
.audit-phase-dot { font-size: 18px; flex-shrink: 0; padding-top: 2px; }
.audit-phase--active .audit-phase-dot { color: var(--color-primary); animation: spin 1s linear infinite; }
.audit-phase--pending .audit-phase-dot { color: var(--color-text-tertiary); }
.phase-pending-dot { width: 18px; height: 18px; border-radius: 50%; border: 2px solid var(--color-border); }
.audit-phase-title { font-size: 14px; font-weight: 500; color: var(--color-text-primary); }
.audit-phase-desc { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }
@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }

/*合规横幅*/
.compliance-banner {
  display: flex; align-items: center; padding: 14px 18px;
  border-radius: var(--radius-lg); border-left: 4px solid; margin-bottom: 16px; gap: 12px;
}
.compliance-banner-icon { font-size: 26px; flex-shrink: 0; }
.compliance-banner-info { flex: 1; }
.compliance-banner-title { font-size: 15px; font-weight: 700; }
.compliance-banner-meta { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }
.compliance-score { font-size: 34px; font-weight: 800; line-height: 1; }

/*剖面块*/
.section-block { margin-bottom: 16px; }
.section-title {
  font-size: 13px; font-weight: 600; color: var(--color-text-primary);
  margin: 0 0 10px; display: flex; align-items: center; gap: 6px;
}

/*审计检查*/
.audit-checks { display: flex; flex-direction: column; gap: 6px; }
.audit-check-item {
  display: flex; gap: 10px; padding: 10px 14px;
  border-radius: var(--radius-md); border: 1px solid var(--color-border-light);
}
.audit-check-item--pass { border-left: 3px solid var(--color-success); }
.audit-check-item--fail { border-left: 3px solid var(--color-danger); background: var(--color-danger-bg); }
.audit-check-status { font-size: 16px; flex-shrink: 0; padding-top: 1px; }
.audit-check-content { flex: 1; min-width: 0; }
.audit-check-name { font-size: 13px; font-weight: 500; color: var(--color-text-primary); margin-bottom: 3px; }
.audit-check-reasoning { font-size: 12px; color: var(--color-text-secondary); line-height: 1.5; }
.audit-check-empty { font-size: 13px; color: var(--color-text-tertiary); padding: 12px; text-align: center; }

/*风险与建议*/
.risk-suggestions-row { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; margin-bottom: 16px; }
.risk-card, .suggestion-card {
  padding: 14px; background: var(--color-bg-page);
  border-radius: var(--radius-lg); border: 1px solid var(--color-border-light);
}
.risk-list, .suggestion-list { display: flex; flex-direction: column; gap: 6px; }
.risk-item, .suggestion-item {
  display: flex; align-items: flex-start; gap: 8px;
  font-size: 12px; color: var(--color-text-secondary); line-height: 1.5;
}
.risk-empty { font-size: 12px; color: var(--color-text-tertiary); }

/*人工智能总结*/
.ai-summary {
  background: var(--color-bg-page); border-radius: var(--radius-md);
  padding: 14px; border: 1px solid var(--color-border-light);
}
.ai-summary pre {
  white-space: pre-wrap; word-break: break-word; font-family: var(--font-sans);
  font-size: 13px; line-height: 1.7; color: var(--color-text-secondary); margin: 0;
}

.history-list { display: flex; flex-direction: column; gap: 8px; }
.history-item {
  width: 100%; text-align: left; padding: 12px 14px; border-radius: var(--radius-md);
  border: 1px solid var(--color-border-light); background: var(--color-bg-page); cursor: pointer;
  transition: all var(--transition-fast);
}
.history-item:hover {
  border-color: var(--color-primary); background: var(--color-primary-bg);
}
.history-item--active {
  border-color: var(--color-primary); background: var(--color-primary-bg);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--color-primary) 20%, transparent);
}
.history-item-title {
  display: flex; align-items: center; justify-content: space-between; gap: 8px;
  font-size: 13px; font-weight: 600; color: var(--color-text-primary);
}
.history-item-time {
  font-size: 12px; font-weight: 400; color: var(--color-text-tertiary);
}
.history-item-meta { margin-top: 8px; display: flex; align-items: center; gap: 8px; }

.oa-jump-btn {
  width: 24px; height: 24px; border: 1px solid var(--color-border);
  background: transparent; border-radius: var(--radius-sm); cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  font-size: 12px; color: var(--color-text-tertiary);
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  outline: none; flex-shrink: 0;
}
.oa-jump-btn:hover {
  border-color: var(--color-primary); color: var(--color-primary);
  background: var(--color-primary-bg);
  transform: scale(1.1);
  box-shadow: 0 2px 8px rgba(79, 70, 229, 0.15);
}
.oa-jump-btn:focus-visible {
  border-color: var(--color-primary); color: var(--color-primary);
  box-shadow: 0 0 0 3px rgba(79, 70, 229, 0.2);
  background: var(--color-primary-bg);
}
.oa-jump-btn:active { transform: scale(0.95); }
.no-result-hint {
  text-align: center; padding: 40px 20px;
  color: var(--color-text-tertiary); display: flex; flex-direction: column; align-items: center; gap: 12px;
}
.no-result-hint p { font-size: 13px; margin: 0; }

/*反应灵敏*/
@media (max-width: 1200px) {
  .archive-grid { grid-template-columns: 360px 1fr; }
}
@media (max-width: 1024px) {
  .archive-grid { grid-template-columns: 1fr; }
  .stats-row { grid-template-columns: repeat(2, 1fr); }
}
@media (max-width: 768px) {
  .stats-row { grid-template-columns: repeat(2, 1fr); }
  .risk-suggestions-row { grid-template-columns: 1fr; }
  .process-info-header { flex-direction: column; }
  .process-info-actions { width: 100%; }
}
@media (max-width: 480px) {
  .stats-row { grid-template-columns: 1fr 1fr; }
  .page-title { font-size: 20px; }
}
/*markdown*/
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
</style>
