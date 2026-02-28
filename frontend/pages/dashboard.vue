<script setup lang="ts">
import {
  SearchOutlined,
  ThunderboltOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  EditOutlined,
  ClockCircleOutlined,
  FireOutlined,
  ExportOutlined,
  ReloadOutlined,
  HistoryOutlined,
  EyeOutlined,
  RightOutlined,
  FolderOpenOutlined,
  DownOutlined,
  UpOutlined,
  InfoCircleOutlined,
  LoadingOutlined,
  FilterOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { useI18n } from '~/composables/useI18n'

definePageMeta({ middleware: 'auth' })

const { t } = useI18n()

const {
  mockProcesses, mockApprovedProcesses, mockReturnedProcesses,
  mockHistoricalResults, mockAuditResult, mockDashboardStats, mockSnapshots,
  mockArchivedOAProcesses, mockArchivedAuditChains, mockArchivedHistoricalResults,
  mockBatchAuditResult, mockTodoAuditResults,
  processCascaderOptions,
} = useMockData()

const todoList = ref(mockProcesses)
const approvedList = ref(mockApprovedProcesses)
const returnedList = ref(mockReturnedProcesses)
const archivedList = ref(mockArchivedOAProcesses)
const currentResult = ref<typeof mockAuditResult | null>(null)
const loading = ref(false)
const phase1Done = ref(false)
const selectedProcess = ref<string | null>(null)
const searchText = ref('')
const searchApplicant = ref('')
const stats = ref(mockDashboardStats)
const showFilters = ref(false)

// Per-process audit results cache (for showing score/recommendation in list)
const processAuditCache = ref<Record<string, typeof mockAuditResult>>({ ...mockTodoAuditResults })
// Per-process loading state (for batch audit per-item animation)
const processAuditLoading = ref<Record<string, boolean>>({})

// View mode
const viewMode = ref<'todo' | 'approved' | 'returned' | 'archived'>('todo')
const isHistoryMode = computed(() => viewMode.value !== 'todo')

// Process type filter — cascader: [category, processName]
const filterProcessType = ref<string[][]>([])
// Resolve selected cascader values to process names, handling both
// category-only selections (e.g. ['采购类']) and full paths (e.g. ['采购类','采购审批'])
const filterProcessNames = computed(() => {
  if (filterProcessType.value.length === 0) return []
  const names: string[] = []
  for (const path of filterProcessType.value) {
    if (path.length >= 2) {
      // Full path: last element is the process name
      names.push(path[path.length - 1])
    } else if (path.length === 1) {
      // Category-only: find all children under this category
      const cat = processCascaderOptions.find(o => o.value === path[0])
      if (cat && cat.children) {
        names.push(...cat.children.map((c: any) => c.value))
      }
    }
  }
  return names
})
const departmentOptions = computed(() => {
  const all = [...todoList.value, ...approvedList.value, ...returnedList.value, ...archivedList.value]
  return [...new Set(all.map(p => p.department))]
})
const filterDepartment = ref<string | undefined>(undefined)

// AI audit status filter: 'unaudited' | 'approve' | 'return' | 'review'
const filterAuditStatus = ref<string | undefined>(undefined)

const clearFilters = () => {
  searchText.value = ''
  searchApplicant.value = ''
  filterProcessType.value = []
  filterDepartment.value = undefined
  filterAuditStatus.value = undefined
}
const hasActiveFilters = computed(() => !!searchText.value || !!searchApplicant.value || filterProcessType.value.length > 0 || !!filterDepartment.value || !!filterAuditStatus.value)

// Batch audit
const selectedProcessIds = ref<string[]>([])
const batchAuditing = ref(false)

const toggleSelectProcess = (processId: string) => {
  const idx = selectedProcessIds.value.indexOf(processId)
  if (idx >= 0) selectedProcessIds.value.splice(idx, 1)
  else selectedProcessIds.value.push(processId)
}

const toggleSelectAll = () => {
  if (selectedProcessIds.value.length === filteredList.value.length) {
    selectedProcessIds.value = []
  } else {
    selectedProcessIds.value = filteredList.value.map(p => p.process_id)
  }
}

// Generate a mock audit result for a process that doesn't have one yet
const generateMockResult = (processId: string): typeof mockAuditResult => {
  const hash = processId.split('').reduce((a, c) => a + c.charCodeAt(0), 0)
  const recs: Array<'approve' | 'return' | 'review'> = ['approve', 'return', 'review', 'approve', 'return']
  const rec = recs[hash % recs.length]
  const score = rec === 'approve' ? 80 + (hash % 20) : rec === 'return' ? 50 + (hash % 25) : 55 + (hash % 30)
  return {
    ...mockAuditResult,
    trace_id: `TR-${Date.now().toString(36).toUpperCase()}-${processId.slice(-3)}`,
    process_id: processId,
    recommendation: rec,
    score,
    action_label: rec === 'approve' ? '建议通过' : rec === 'return' ? '建议退回' : '建议复核',
    confidence: 0.7 + (hash % 25) / 100,
    risk_points: rec === 'approve' ? [] : ['存在待确认的合规风险项'],
    suggestions: rec === 'approve' ? ['建议定期复核'] : ['建议补充相关材料后重新提交'],
    ai_summary: rec === 'approve' ? '该流程整体合规，建议通过。' : '该流程存在部分问题，需要关注。',
    duration_ms: 1200 + (hash % 2000),
  }
}

// Batch audit progress tracking
const batchAuditTotal = ref(0)
const batchAuditDone = ref(0)

const handleBatchAudit = async () => {
  if (selectedProcessIds.value.length === 0) return
  batchAuditing.value = true
  const ids = [...selectedProcessIds.value]
  batchAuditTotal.value = ids.length
  batchAuditDone.value = 0

  // Set all selected to loading
  for (const id of ids) {
    processAuditLoading.value[id] = true
  }

  // Process each item with staggered delays
  for (let i = 0; i < ids.length; i++) {
    const id = ids[i]
    await new Promise(r => setTimeout(r, 800 + Math.random() * 1200))
    // Use existing mock result or generate one
    const result = mockTodoAuditResults[id] || generateMockResult(id)
    processAuditCache.value[id] = result
    processAuditLoading.value[id] = false
    batchAuditDone.value = i + 1
    // If this process is currently selected, show its result directly
    if (selectedProcess.value === id) {
      currentResult.value = { ...result }
    }
  }

  batchAuditing.value = false
  selectedProcessIds.value = []
  message.success(t('dashboard.batchDone'))
}

// Audit history chain
const showHistoryChain = ref(false)
const historyChainProcessId = ref<string | null>(null)
const expandedChainNodes = ref<Set<string>>(new Set())

const toggleChainNode = (snapshotId: string) => {
  if (expandedChainNodes.value.has(snapshotId)) expandedChainNodes.value.delete(snapshotId)
  else expandedChainNodes.value.add(snapshotId)
}

const getAuditChain = (processId: string) => {
  const archivedChain = mockArchivedAuditChains[processId]
  if (archivedChain && archivedChain.length > 0) return archivedChain
  const snapshots = mockSnapshots.filter(s => s.process_id === processId)
  if (snapshots.length > 0) return snapshots
  const hist = mockHistoricalResults[processId] || mockArchivedHistoricalResults[processId]
  if (!hist) return []
  return [{
    snapshot_id: `SN-${processId}`,
    process_id: processId,
    title: selectedProcessInfo.value?.title || '',
    applicant: selectedProcessInfo.value?.applicant || '',
    department: selectedProcessInfo.value?.department || '',
    recommendation: hist.recommendation,
    score: hist.score,
    created_at: selectedProcessInfo.value?.submit_time || '',
    adopted: true,
  }]
}

const currentAuditChain = computed(() => {
  if (!historyChainProcessId.value) return []
  return getAuditChain(historyChainProcessId.value)
})

const openHistoryChain = (processId: string) => {
  historyChainProcessId.value = processId
  expandedChainNodes.value = new Set()
  showHistoryChain.value = true
}

const filteredList = computed(() => {
  let list: typeof todoList.value
  switch (viewMode.value) {
    case 'approved': list = approvedList.value; break
    case 'returned': list = returnedList.value; break
    case 'archived': list = archivedList.value; break
    default: list = todoList.value
  }
  if (filterProcessNames.value.length > 0) {
    list = list.filter(p => filterProcessNames.value.includes(p.process_type))
  }
  if (filterDepartment.value) {
    list = list.filter(p => p.department === filterDepartment.value)
  }
  if (searchText.value) {
    const q = searchText.value.toLowerCase()
    list = list.filter(p => p.title.toLowerCase().includes(q))
  }
  if (searchApplicant.value) {
    const q2 = searchApplicant.value.toLowerCase()
    list = list.filter(p => p.applicant.toLowerCase().includes(q2))
  }
  // AI audit status filter
  if (filterAuditStatus.value) {
    if (filterAuditStatus.value === 'unaudited') {
      list = list.filter(p => !processAuditCache.value[p.process_id])
    } else {
      list = list.filter(p => processAuditCache.value[p.process_id]?.recommendation === filterAuditStatus.value)
    }
  }
  return list
})

const { paged: pagedList, current: listPage, pageSize: listPageSize, total: listTotal, onChange: onListPageChange } = usePagination(filteredList, 10)

const handleSelectProcess = (processId: string) => {
  selectedProcess.value = processId
  if (isHistoryMode.value) {
    const hist = mockHistoricalResults[processId] || mockArchivedHistoricalResults[processId]
    currentResult.value = hist ? { ...hist } : null
  } else {
    // If we have a cached audit result, show it directly
    const cached = processAuditCache.value[processId]
    if (cached) {
      currentResult.value = { ...cached, process_id: processId }
      phase1Done.value = true
    } else {
      currentResult.value = null
      phase1Done.value = false
    }
  }
}

const handleAudit = async (processId: string) => {
  loading.value = true
  phase1Done.value = false
  // Phase 1: reasoning
  await new Promise(resolve => setTimeout(resolve, 2200))
  phase1Done.value = true
  // Phase 2: extraction
  await new Promise(resolve => setTimeout(resolve, 1650))
  const result = mockTodoAuditResults[processId] || generateMockResult(processId)
  currentResult.value = { ...result, process_id: processId }
  processAuditCache.value[processId] = currentResult.value
  loading.value = false
}

const handleReAudit = async () => {
  if (!selectedProcess.value) return
  currentResult.value = null
  await handleAudit(selectedProcess.value)
}

const jumpToOA = (processId: string) => {
  message.info(t('dashboard.jumpingToOA', `Jumping to OA: ${processId}...`))
}

const switchView = (mode: 'todo' | 'approved' | 'returned' | 'archived') => {
  viewMode.value = mode
  selectedProcess.value = null
  currentResult.value = null
  listPage.value = 1
  selectedProcessIds.value = []
}

const selectedProcessInfo = computed(() => {
  const all = [...todoList.value, ...approvedList.value, ...returnedList.value, ...archivedList.value]
  return all.find(p => p.process_id === selectedProcess.value)
})

const viewModeLabel = computed(() => {
  switch (viewMode.value) {
    case 'approved': return t('dashboard.viewMode.approved')
    case 'returned': return t('dashboard.viewMode.returned')
    case 'archived': return t('dashboard.viewMode.archived')
    default: return t('dashboard.viewMode.todo')
  }
})

const urgencyConfig = computed<Record<string, { color: string; bg: string; label: string }>>(() => ({
  high: { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)', label: t('dashboard.urgency.high') },
  medium: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', label: t('dashboard.urgency.medium') },
  low: { color: 'var(--color-success)', bg: 'var(--color-success-bg)', label: t('dashboard.urgency.low') },
}))

const recommendationConfig = computed<Record<string, { color: string; bg: string; icon: any; label: string }>>(() => ({
  approve: { color: 'var(--color-success)', bg: 'var(--color-success-bg)', icon: CheckCircleOutlined, label: t('dashboard.rec.approve') },
  return: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', icon: ReloadOutlined, label: t('dashboard.rec.return') },
  review: { color: 'var(--color-info)', bg: 'var(--color-info-bg)', icon: EyeOutlined, label: t('dashboard.rec.review') },
}))

// Helper: get short recommendation label for list display
const getShortRecLabel = (rec: string) => {
  const map: Record<string, string> = {
    approve: t('dashboard.suggestApprove'),
    return: t('dashboard.suggestReturn'),
    review: t('dashboard.suggestReview'),
  }
  return map[rec] || rec
}
</script>

<template>
  <div class="dashboard">
    <!-- Page header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('dashboard.title') }}</h1>
        <p class="page-subtitle">{{ t('dashboard.subtitleWithCount', `${stats.todayAudits}`) }}</p>
      </div>
    </div>

    <!-- Stats row - clickable cards -->
    <div class="stats-row">
      <div
        class="stat-card stat-card--primary"
        :class="{ 'stat-card--selected': viewMode === 'todo' }"
        @click="switchView('todo')"
      >
        <div class="stat-card-icon"><ClockCircleOutlined /></div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ stats.pendingCount }}</span>
          <span class="stat-card-label">{{ t('dashboard.tab.pending') }}</span>
        </div>
      </div>
      <div
        class="stat-card stat-card--success"
        :class="{ 'stat-card--selected': viewMode === 'approved' }"
        @click="switchView('approved')"
      >
        <div class="stat-card-icon"><CheckCircleOutlined /></div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ stats.todayApproved }}</span>
          <span class="stat-card-label">{{ t('dashboard.tab.approved') }}</span>
        </div>
      </div>
      <div
        class="stat-card stat-card--danger"
        :class="{ 'stat-card--selected': viewMode === 'returned' }"
        @click="switchView('returned')"
      >
        <div class="stat-card-icon"><CloseCircleOutlined /></div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ stats.todayReturned }}</span>
          <span class="stat-card-label">{{ t('dashboard.tab.returned') }}</span>
        </div>
      </div>
      <div
        class="stat-card stat-card--archived"
        :class="{ 'stat-card--selected': viewMode === 'archived' }"
        @click="switchView('archived')"
      >
        <div class="stat-card-icon"><FolderOpenOutlined /></div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ archivedList.length }}</span>
          <span class="stat-card-label">{{ t('dashboard.tab.archived') }}</span>
        </div>
      </div>
    </div>

    <!-- Main content area -->
    <div class="dashboard-grid">
      <!-- Left: Process list -->
      <div class="todo-panel">
        <div class="panel-header">
          <div class="panel-header-row">
            <h3 class="panel-title">
              <FireOutlined v-if="viewMode === 'todo'" style="color: var(--color-primary);" />
              <FolderOpenOutlined v-else-if="viewMode === 'archived'" style="color: var(--color-info);" />
              <HistoryOutlined v-else style="color: var(--color-text-tertiary);" />
              {{ viewModeLabel }}
              <a-badge :count="filteredList.length" :number-style="{ backgroundColor: 'var(--color-primary)' }" />
            </h3>
            <a-button size="small" type="default" @click="showFilters = !showFilters" class="filter-toggle-btn" :class="{ 'filter-toggle-btn--active': hasActiveFilters }">
              <FilterOutlined />
              {{ t('dashboard.filter') }}
              <span v-if="hasActiveFilters" class="filter-active-dot" />
            </a-button>
          </div>
          <!-- Collapsible filter bar -->
          <transition name="slide">
            <div v-if="showFilters" class="filter-bar">
              <a-input
                v-model:value="searchText"
                :placeholder="t('dashboard.searchPlaceholder')"
                allow-clear
                style="flex: 2; min-width: 160px;"
              >
                <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
              </a-input>
              <a-input
                v-model:value="searchApplicant"
                :placeholder="t('dashboard.searchApplicant')"
                allow-clear
                style="flex: 1; min-width: 130px;"
              >
                <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
              </a-input>
              <a-cascader
                v-model:value="filterProcessType"
                :options="processCascaderOptions"
                :placeholder="t('dashboard.filterProcessTypePlaceholder')"
                multiple
                :max-tag-count="1"
                allow-clear
                style="flex: 1.5; min-width: 160px;"
                :show-search="{ filter: (inputValue: string, path: any[]) => path.some((o: any) => o.label.toLowerCase().includes(inputValue.toLowerCase())) }"
              />
              <a-select
                v-model:value="filterDepartment"
                :placeholder="t('dashboard.filterDepartment')"
                allow-clear
                style="flex: 1; min-width: 120px;"
              >
                <a-select-option v-for="d in departmentOptions" :key="d" :value="d">{{ d }}</a-select-option>
              </a-select>
              <a-select
                v-model:value="filterAuditStatus"
                :placeholder="t('dashboard.filterAuditStatus')"
                allow-clear
                style="flex: 1; min-width: 130px;"
              >
                <a-select-option value="unaudited">{{ t('dashboard.auditStatus.unaudited') }}</a-select-option>
                <a-select-option value="approve">{{ t('dashboard.auditStatus.approve') }}</a-select-option>
                <a-select-option value="return">{{ t('dashboard.auditStatus.return') }}</a-select-option>
                <a-select-option value="review">{{ t('dashboard.auditStatus.review') }}</a-select-option>
              </a-select>
              <a-button size="small" @click="clearFilters">{{ t('dashboard.filterReset') }}</a-button>
            </div>
          </transition>
          <!-- Batch audit toolbar (todo mode only) -->
          <div v-if="viewMode === 'todo'" class="batch-toolbar">
            <div class="batch-toolbar-left">
              <a-checkbox
                :checked="selectedProcessIds.length === filteredList.length && filteredList.length > 0"
                :indeterminate="selectedProcessIds.length > 0 && selectedProcessIds.length < filteredList.length"
                @change="toggleSelectAll"
              >
                {{ selectedProcessIds.length > 0 ? t('dashboard.selected', `${selectedProcessIds.length}`) : t('dashboard.selectAll') }}
              </a-checkbox>
              <span v-if="batchAuditing" class="batch-progress-hint">
                {{ t('dashboard.auditedProgress', `${batchAuditDone}/${batchAuditTotal}`) }}
              </span>
            </div>
            <a-button
              v-if="selectedProcessIds.length > 0"
              type="primary"
              size="small"
              :disabled="batchAuditing"
              @click="handleBatchAudit"
              class="batch-audit-btn"
            >
              <LoadingOutlined v-if="batchAuditing" />
              <ThunderboltOutlined v-else />
              {{ t('dashboard.batchAudit') }}
            </a-button>
          </div>
        </div>

        <div class="todo-list">
          <div
            v-for="item in pagedList"
            :key="item.process_id"
            class="todo-item"
            :class="{
              'todo-item--selected': selectedProcess === item.process_id,
              'todo-item--audited-approve': viewMode === 'todo' && processAuditCache[item.process_id]?.recommendation === 'approve',
              'todo-item--audited-return': viewMode === 'todo' && processAuditCache[item.process_id]?.recommendation === 'return',
              'todo-item--audited-review': viewMode === 'todo' && processAuditCache[item.process_id]?.recommendation === 'review',
            }"
            @click="handleSelectProcess(item.process_id)"
          >
            <div v-if="viewMode === 'todo'" class="todo-item-checkbox" @click.stop="toggleSelectProcess(item.process_id)">
              <a-checkbox :checked="selectedProcessIds.includes(item.process_id)" />
            </div>
            <div class="todo-item-main">
              <div class="todo-item-title">
                <CheckCircleOutlined
                  v-if="viewMode === 'todo' && processAuditCache[item.process_id]"
                  class="todo-item-audited-icon"
                  :style="{ color: recommendationConfig[processAuditCache[item.process_id].recommendation]?.color }"
                />
                {{ item.title }}
              </div>
              <div class="todo-item-meta">
                <span>{{ item.applicant }}</span>
                <span class="todo-item-dot">·</span>
                <span>{{ item.department }}</span>
                <span class="todo-item-dot">·</span>
                <span>{{ item.submit_time }}</span>
              </div>
              <!-- Node + OA jump left, score badge right -->
              <div class="todo-item-audit-info">
                <div class="todo-item-audit-left">
                  <span
                    class="todo-item-node"
                    :class="{
                      'todo-item-node--success': item.status === 'approved',
                      'todo-item-node--danger': item.status === 'returned',
                      'todo-item-node--info': item.status === 'archived',
                    }"
                  >{{ item.current_node }}</span>
                  <span v-if="isHistoryMode" class="todo-item-process-type">{{ item.process_type }}</span>
                </div>
                <div class="todo-item-audit-right">
                  <!-- Per-item loading animation during batch -->
                  <span v-if="processAuditLoading[item.process_id]" class="todo-item-auditing">
                    <LoadingOutlined style="font-size: 12px;" />
                    <span>{{ t('dashboard.auditingItem') }}</span>
                  </span>
                  <!-- Show score badge when audit is done (todo mode) -->
                  <span
                    v-else-if="processAuditCache[item.process_id] && viewMode === 'todo'"
                    class="todo-item-score-badge"
                    :style="{
                      color: recommendationConfig[processAuditCache[item.process_id].recommendation]?.color,
                      background: recommendationConfig[processAuditCache[item.process_id].recommendation]?.bg,
                    }"
                  >
                    {{ processAuditCache[item.process_id].score }}{{ t('dashboard.points') }}
                    {{ getShortRecLabel(processAuditCache[item.process_id].recommendation) }}
                  </span>
                  <!-- Show historical score for approved/returned/archived -->
                  <span
                    v-else-if="isHistoryMode && (mockHistoricalResults[item.process_id] || mockArchivedHistoricalResults[item.process_id])"
                    class="todo-item-score-badge"
                    :style="{
                      color: recommendationConfig[(mockHistoricalResults[item.process_id] || mockArchivedHistoricalResults[item.process_id]).recommendation]?.color,
                      background: recommendationConfig[(mockHistoricalResults[item.process_id] || mockArchivedHistoricalResults[item.process_id]).recommendation]?.bg,
                    }"
                  >
                    {{ (mockHistoricalResults[item.process_id] || mockArchivedHistoricalResults[item.process_id]).score }}{{ t('dashboard.points') }}
                  </span>
                  <a-tooltip :title="t('dashboard.jumpToOA')" :mouse-enter-delay="0.5">
                    <button class="oa-jump-btn" @click.stop="jumpToOA(item.process_id)">
                      <ExportOutlined />
                    </button>
                  </a-tooltip>
                </div>
              </div>
            </div>
          </div>

          <div v-if="filteredList.length === 0" class="todo-empty">
            <a-empty :description="t('dashboard.noData')" />
          </div>
        </div>

        <!-- Pagination -->
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

      <!-- Right: Audit result / Action panel -->
      <div class="result-panel">
        <div class="panel-header">
          <h3 class="panel-title">
            <ThunderboltOutlined v-if="!isHistoryMode" style="color: var(--color-primary);" />
            <FolderOpenOutlined v-else-if="viewMode === 'archived'" style="color: var(--color-info);" />
            <HistoryOutlined v-else style="color: var(--color-text-tertiary);" />
            {{ viewMode === 'archived' ? t('dashboard.archivedResult') : isHistoryMode ? t('dashboard.historyResult') : t('dashboard.auditResult') }}
          </h3>
        </div>

        <div class="result-content">
          <!-- Loading state: two-phase card style (matches archive.vue audit-progress) -->
          <div v-if="loading" class="result-loading">
            <!-- Process basic info above animation -->
            <div v-if="selectedProcessInfo" class="loading-process-info">
              <div class="loading-process-title">{{ selectedProcessInfo.title }}</div>
              <div class="loading-process-meta">
                {{ selectedProcessInfo.applicant }} · {{ selectedProcessInfo.department }} · {{ selectedProcessInfo.submit_time }}
              </div>
            </div>
            <div class="audit-progress">
              <div class="audit-phase" :class="{ 'audit-phase--done': phase1Done, 'audit-phase--active': !phase1Done }">
                <div class="audit-phase-dot">
                  <LoadingOutlined v-if="!phase1Done" />
                  <CheckCircleOutlined v-else style="color: var(--color-success);" />
                </div>
                <div class="audit-phase-info">
                  <div class="audit-phase-title">{{ t('dashboard.phase1') }}</div>
                  <div class="audit-phase-desc">{{ t('dashboard.phase1Duration') }}</div>
                </div>
              </div>
              <div class="audit-phase" :class="{ 'audit-phase--active': phase1Done, 'audit-phase--pending': !phase1Done }">
                <div class="audit-phase-dot">
                  <LoadingOutlined v-if="phase1Done" />
                  <div v-else class="phase-pending-dot" />
                </div>
                <div class="audit-phase-info">
                  <div class="audit-phase-title">{{ t('dashboard.phase2') }}</div>
                  <div class="audit-phase-desc">{{ t('dashboard.phase2Duration') }}</div>
                </div>
              </div>
            </div>
          </div>

          <!-- TODO mode: Selected but not yet audited - show action prompt -->
          <template v-else-if="!isHistoryMode && selectedProcess && !currentResult">
            <div class="action-prompt">
              <div class="action-prompt-info">
                <h4>{{ selectedProcessInfo?.title }}</h4>
                <p>{{ selectedProcessInfo?.applicant }} · {{ selectedProcessInfo?.department }} · {{ selectedProcessInfo?.submit_time }}</p>
              </div>
              <div class="action-prompt-buttons">
                <a-button type="primary" size="large" @click="handleAudit(selectedProcess!)">
                  <ThunderboltOutlined /> {{ t('dashboard.startAIAudit') }}
                </a-button>
                <a-button size="large" @click="jumpToOA(selectedProcess!)">
                  <ExportOutlined /> {{ t('dashboard.jumpToOASystem') }}
                </a-button>
              </div>
            </div>
          </template>

          <!-- History mode: Selected but no historical result found -->
          <template v-else-if="isHistoryMode && selectedProcess && !currentResult">
            <div class="result-empty">
              <div class="result-empty-icon"><HistoryOutlined /></div>
              <h4>{{ t('dashboard.noHistoryTitle') }}</h4>
              <p>{{ t('dashboard.noHistoryDesc') }}</p>
              <a-button style="margin-top: 16px;" @click="jumpToOA(selectedProcess!)">
                <ExportOutlined /> {{ t('dashboard.jumpToOAView') }}
              </a-button>
            </div>
          </template>

          <!-- Result display (both modes) -->
          <template v-else-if="currentResult">
            <!-- Action bar -->
            <div class="result-action-bar">
              <template v-if="isHistoryMode">
                <div class="history-badge">
                  <FolderOpenOutlined v-if="viewMode === 'archived'" />
                  <HistoryOutlined v-else />
                  {{ viewMode === 'archived' ? t('dashboard.archivedReadonly') : t('dashboard.historyReadonly') }}
                </div>
                <a-button @click="openHistoryChain(currentResult.process_id)">
                  <EyeOutlined /> {{ t('dashboard.auditChain') }}
                </a-button>
                <a-button type="primary" @click="jumpToOA(currentResult.process_id)">
                  <ExportOutlined /> {{ t('dashboard.jumpOA') }}
                </a-button>
              </template>
              <template v-else>
                <a-button @click="jumpToOA(currentResult.process_id)">
                  <ExportOutlined /> {{ t('dashboard.jumpOA') }}
                </a-button>
                <a-button @click="handleReAudit">
                  <ReloadOutlined /> {{ t('dashboard.reAudit') }}
                </a-button>
              </template>
            </div>

            <!-- Recommendation banner -->
            <div
              class="result-banner"
              :style="{
                background: recommendationConfig[currentResult.recommendation].bg,
                borderColor: recommendationConfig[currentResult.recommendation].color,
              }"
            >
              <component
                :is="recommendationConfig[currentResult.recommendation].icon"
                class="result-banner-icon"
                :style="{ color: recommendationConfig[currentResult.recommendation].color }"
              />
              <div class="result-banner-info">
                <div class="result-banner-title" :style="{ color: recommendationConfig[currentResult.recommendation].color }">
                  {{ recommendationConfig[currentResult.recommendation].label }}
                </div>
                <div class="result-banner-meta">
                  {{ t('dashboard.overallScore') }} {{ currentResult.score }}{{ t('dashboard.points') }}
                  · {{ t('dashboard.duration') }} {{ currentResult.duration_ms }}ms
                </div>
              </div>
              <div class="result-score" :style="{ color: recommendationConfig[currentResult.recommendation].color }">
                {{ currentResult.score }}
              </div>
            </div>

            <!-- Rule checks -->
            <div class="result-section">
              <h4 class="result-section-title">{{ t('dashboard.ruleCheckDetail') }}</h4>
              <div class="rule-checks">
                <div
                  v-for="rule in currentResult.details"
                  :key="rule.rule_id"
                  class="rule-check-item"
                  :class="{ 'rule-check-item--pass': rule.passed, 'rule-check-item--fail': !rule.passed }"
                >
                  <div class="rule-check-status">
                    <CheckCircleOutlined v-if="rule.passed" style="color: var(--color-success);" />
                    <CloseCircleOutlined v-else style="color: var(--color-danger);" />
                  </div>
                  <div class="rule-check-content">
                    <div class="rule-check-name">
                      {{ rule.rule_name }}
                      <span v-if="rule.is_locked" class="rule-locked-badge">{{ t('rule.scope.mandatory') }}</span>
                    </div>
                    <div class="rule-check-reasoning">{{ rule.reasoning }}</div>
                  </div>
                </div>
              </div>
            </div>

            <!-- Opt5: Risk points & suggestions as parallel cards below rule checks -->
            <div v-if="currentResult.risk_points?.length || currentResult.suggestions?.length" class="risk-suggest-row">
              <div v-if="currentResult.risk_points?.length" class="insight-card insight-card--risk">
                <div class="insight-card-header">
                  <CloseCircleOutlined style="color: var(--color-danger);" />
                  <span>{{ t('dashboard.riskPoints') }}</span>
                </div>
                <ul class="insight-card-list">
                  <li v-for="(rp, i) in currentResult.risk_points" :key="i">{{ rp }}</li>
                </ul>
              </div>
              <div v-if="currentResult.suggestions?.length" class="insight-card insight-card--suggest">
                <div class="insight-card-header">
                  <InfoCircleOutlined style="color: var(--color-primary);" />
                  <span>{{ t('dashboard.suggestions') }}</span>
                </div>
                <ul class="insight-card-list">
                  <li v-for="(sg, i) in currentResult.suggestions" :key="i">{{ sg }}</li>
                </ul>
              </div>
            </div>

            <!-- AI Reasoning -->
            <div class="result-section">
              <h4 class="result-section-title">{{ t('dashboard.aiReasoning') }}</h4>
              <div class="ai-reasoning">
                <pre>{{ currentResult.ai_reasoning }}</pre>
              </div>
            </div>
          </template>

          <!-- Empty state -->
          <div v-else class="result-empty">
            <div class="result-empty-icon">
              <ThunderboltOutlined v-if="!isHistoryMode" />
              <FolderOpenOutlined v-else-if="viewMode === 'archived'" />
              <HistoryOutlined v-else />
            </div>
            <h4>{{ viewMode === 'archived' ? t('dashboard.emptyArchived') : isHistoryMode ? t('dashboard.emptyHistory') : t('dashboard.emptyTodo') }}</h4>
            <p>{{ viewMode === 'archived' ? t('dashboard.emptyArchivedDesc') : isHistoryMode ? t('dashboard.emptyHistoryDesc') : t('dashboard.emptyTodoDesc') }}</p>
          </div>
        </div>
      </div>
    </div>

    <!-- Audit History Chain Drawer -->
    <Teleport to="body">
      <transition name="drawer">
        <div v-if="showHistoryChain" class="drawer-overlay" @click.self="showHistoryChain = false">
          <div class="drawer-panel">
            <div class="drawer-header">
              <h3>{{ t('dashboard.auditHistoryChain') }}</h3>
              <button class="drawer-close" @click="showHistoryChain = false">✕</button>
            </div>
            <div class="drawer-body">
              <p class="chain-desc">{{ t('dashboard.chainDesc') }}</p>
              <div v-if="currentAuditChain.length === 0" style="padding: 40px; text-align: center;">
                <a-empty :description="t('dashboard.noAuditRecords')" />
              </div>
              <div v-else class="audit-chain">
                <div
                  v-for="(snap, idx) in currentAuditChain"
                  :key="snap.snapshot_id"
                  class="chain-node"
                >
                  <div class="chain-timeline">
                    <div class="chain-dot" :style="{ background: recommendationConfig[snap.recommendation]?.color }" />
                    <div v-if="idx < currentAuditChain.length - 1" class="chain-line" />
                  </div>
                  <div class="chain-card">
                    <div class="chain-card-header" @click="toggleChainNode(snap.snapshot_id)" style="cursor: pointer;">
                      <span
                        class="chain-tag"
                        :style="{ color: recommendationConfig[snap.recommendation]?.color, background: recommendationConfig[snap.recommendation]?.bg }"
                      >
                        <component :is="recommendationConfig[snap.recommendation]?.icon" />
                        {{ recommendationConfig[snap.recommendation]?.label }}
                      </span>
                      <span class="chain-score">{{ snap.score }}{{ t('dashboard.points') }}</span>
                      <span class="chain-expand-btn">
                        <DownOutlined v-if="!expandedChainNodes.has(snap.snapshot_id)" />
                        <UpOutlined v-else />
                      </span>
                    </div>
                    <div class="chain-card-meta">
                      {{ snap.created_at }}
                      <span v-if="snap.adopted !== null" class="chain-adopted" :class="snap.adopted ? 'chain-adopted--yes' : 'chain-adopted--no'">
                        {{ snap.adopted ? t('dashboard.adopted') : t('dashboard.notAdopted') }}
                      </span>
                    </div>
                    <div v-if="expandedChainNodes.has(snap.snapshot_id)" class="chain-detail">
                      <template v-if="mockHistoricalResults[snap.process_id] || mockArchivedHistoricalResults[snap.process_id]">
                        <!-- Rule checks -->
                        <div class="chain-section-title">{{ t('dashboard.ruleCheckDetail') }}</div>
                        <div v-for="rule in (mockHistoricalResults[snap.process_id] || mockArchivedHistoricalResults[snap.process_id])?.details" :key="rule.rule_id" class="chain-rule-item" :class="rule.passed ? 'chain-rule--pass' : 'chain-rule--fail'">
                          <component :is="rule.passed ? CheckCircleOutlined : CloseCircleOutlined" :style="{ color: rule.passed ? 'var(--color-success)' : 'var(--color-danger)' }" />
                          <div>
                            <div class="chain-rule-name">{{ rule.rule_name }}</div>
                            <div class="chain-rule-reasoning">{{ rule.reasoning }}</div>
                          </div>
                        </div>
                        <!-- Risk points & suggestions -->
                        <div
                          v-if="(mockHistoricalResults[snap.process_id] || mockArchivedHistoricalResults[snap.process_id])?.risk_points?.length || (mockHistoricalResults[snap.process_id] || mockArchivedHistoricalResults[snap.process_id])?.suggestions?.length"
                          class="risk-suggest-row"
                          style="margin-top: 10px;"
                        >
                          <div v-if="(mockHistoricalResults[snap.process_id] || mockArchivedHistoricalResults[snap.process_id])?.risk_points?.length" class="insight-card insight-card--risk">
                            <div class="insight-card-header">
                              <CloseCircleOutlined style="color: var(--color-danger);" />
                              <span>{{ t('dashboard.riskPoints') }}</span>
                            </div>
                            <ul class="insight-card-list">
                              <li v-for="(rp, i) in (mockHistoricalResults[snap.process_id] || mockArchivedHistoricalResults[snap.process_id])?.risk_points" :key="i">{{ rp }}</li>
                            </ul>
                          </div>
                          <div v-if="(mockHistoricalResults[snap.process_id] || mockArchivedHistoricalResults[snap.process_id])?.suggestions?.length" class="insight-card insight-card--suggest">
                            <div class="insight-card-header">
                              <InfoCircleOutlined style="color: var(--color-primary);" />
                              <span>{{ t('dashboard.suggestions') }}</span>
                            </div>
                            <ul class="insight-card-list">
                              <li v-for="(sg, i) in (mockHistoricalResults[snap.process_id] || mockArchivedHistoricalResults[snap.process_id])?.suggestions" :key="i">{{ sg }}</li>
                            </ul>
                          </div>
                        </div>
                        <!-- AI Reasoning -->
                        <div class="chain-section-title" style="margin-top: 10px;">{{ t('dashboard.aiReasoning') }}</div>
                        <div class="chain-reasoning">
                          <pre>{{ (mockHistoricalResults[snap.process_id] || mockArchivedHistoricalResults[snap.process_id])?.ai_reasoning }}</pre>
                        </div>
                      </template>
                      <div v-else class="chain-no-detail">{{ t('dashboard.noHistoryDesc') }}</div>
                    </div>
                  </div>
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
.dashboard { animation: fadeIn 0.3s ease-out; }
@keyframes fadeIn { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

.page-header { margin-bottom: 24px; }
.page-title { font-size: 24px; font-weight: 700; color: var(--color-text-primary); margin: 0; letter-spacing: -0.02em; }
.page-subtitle { font-size: 14px; color: var(--color-text-tertiary); margin: 4px 0 0; }

/* Stats row */
.stats-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; margin-bottom: 24px; }
.stat-card {
  background: var(--color-bg-card); border-radius: var(--radius-lg); padding: 20px;
  display: flex; align-items: center; gap: 16px; border: 2px solid var(--color-border-light);
  transition: all var(--transition-base); cursor: pointer; user-select: none;
}
.stat-card:hover { transform: translateY(-2px); box-shadow: var(--shadow-md); }
.stat-card--selected { border-color: var(--color-primary); box-shadow: 0 0 0 1px var(--color-primary); }
.stat-card-icon {
  width: 48px; height: 48px; border-radius: var(--radius-lg);
  display: flex; align-items: center; justify-content: center; font-size: 22px; flex-shrink: 0;
}
.stat-card--primary .stat-card-icon { background: var(--color-primary-bg); color: var(--color-primary); }
.stat-card--success .stat-card-icon { background: var(--color-success-bg); color: var(--color-success); }
.stat-card--danger .stat-card-icon { background: var(--color-danger-bg); color: var(--color-danger); }
.stat-card--archived .stat-card-icon { background: var(--color-info-bg); color: var(--color-info); }
.stat-card-info { display: flex; flex-direction: column; }
.stat-card-value { font-size: 28px; font-weight: 700; color: var(--color-text-primary); line-height: 1.2; }
.stat-card-label { font-size: 13px; color: var(--color-text-tertiary); margin-top: 2px; }

/* Dashboard grid */
.dashboard-grid { display: grid; grid-template-columns: 420px 1fr; gap: 24px; align-items: start; }
.todo-panel, .result-panel {
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); overflow: hidden;
}
.panel-header {
  padding: 16px 20px; border-bottom: 1px solid var(--color-border-light);
  display: flex; flex-direction: column; gap: 12px;
}
.panel-title {
  font-size: 15px; font-weight: 600; color: var(--color-text-primary);
  margin: 0; display: flex; align-items: center; gap: 8px;
}

/* Panel header row */
.panel-header-row {
  display: flex; align-items: center; justify-content: space-between;
}

/* Collapsible filter bar */
.filter-toggle-btn { position: relative; }
.filter-toggle-btn--active { color: var(--color-primary); border-color: var(--color-primary); }
.filter-active-dot {
  display: inline-block; width: 6px; height: 6px; border-radius: 50%;
  background: var(--color-primary); margin-left: 4px; vertical-align: middle;
}
.filter-bar {
  display: flex; gap: 8px; align-items: center; flex-wrap: wrap;
  padding: 10px 0 0;
}
.slide-enter-active, .slide-leave-active { transition: all 0.2s ease; }
.slide-enter-from, .slide-leave-to { opacity: 0; transform: translateY(-8px); }

/* Todo list */
.todo-list { max-height: calc(100vh - 380px); overflow-y: auto; }
.todo-item {
  display: flex; align-items: flex-start;
  padding: 14px 20px; cursor: pointer; transition: all var(--transition-fast);
  border-bottom: 1px solid var(--color-border-light); gap: 12px;
}
.todo-item:last-child { border-bottom: none; }
.todo-item:hover { background: var(--color-bg-hover); }
.todo-item--selected { background: var(--color-primary-bg); border-left: 3px solid var(--color-primary); }
.todo-item--audited-approve { background: rgba(34, 197, 94, 0.03); border-left: 3px solid var(--color-success); }
.todo-item--audited-return { background: rgba(245, 158, 11, 0.03); border-left: 3px solid var(--color-warning); }
.todo-item--audited-review { background: rgba(59, 130, 246, 0.03); border-left: 3px solid var(--color-info); }
.todo-item--audited-approve.todo-item--selected,
.todo-item--audited-return.todo-item--selected,
.todo-item--audited-review.todo-item--selected { background: var(--color-primary-bg); border-left: 3px solid var(--color-primary); }
.todo-item-audited-icon { font-size: 13px; flex-shrink: 0; }
.todo-item-main { flex: 1; min-width: 0; }
.todo-item-title {
  font-size: 14px; font-weight: 500; color: var(--color-text-primary);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; margin-bottom: 4px;
  display: flex; align-items: center; gap: 6px;
}
.todo-item-meta { font-size: 12px; color: var(--color-text-tertiary); display: flex; align-items: center; gap: 4px; flex-wrap: wrap; margin-bottom: 6px; }
.todo-item-dot { color: var(--color-border); }

/* Opt3: Audit info row below meta — left/right layout */
.todo-item-audit-info {
  display: flex; align-items: center; justify-content: space-between; gap: 8px;
}
.todo-item-audit-left {
  display: flex; align-items: center; gap: 8px; min-width: 0;
}
.todo-item-audit-right {
  display: flex; align-items: center; gap: 8px; flex-shrink: 0;
}
.todo-item-node {
  font-size: 11px; font-weight: 500; padding: 2px 8px;
  border-radius: var(--radius-full); background: var(--color-bg-hover);
  color: var(--color-text-secondary); white-space: nowrap;
}
.todo-item-node--success {
  background: var(--color-success-bg); color: var(--color-success); font-weight: 600;
}
.todo-item-node--danger {
  background: var(--color-danger-bg); color: var(--color-danger); font-weight: 600;
}
.todo-item-node--info {
  background: var(--color-info-bg); color: var(--color-info); font-weight: 600;
}
.todo-item-process-type {
  font-size: 11px; padding: 2px 7px; border-radius: var(--radius-full);
  background: var(--color-bg-hover); color: var(--color-text-tertiary);
  border: 1px solid var(--color-border-light); white-space: nowrap;
}
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

/* Opt4: Per-item auditing animation */
.todo-item-auditing {
  display: inline-flex; align-items: center; gap: 4px;
  font-size: 11px; color: var(--color-primary); font-weight: 500;
  animation: auditPulse 1.5s ease-in-out infinite;
}
@keyframes auditPulse { 0%, 100% { opacity: 0.6; } 50% { opacity: 1; } }

/* Opt3: Score badge in list */
.todo-item-score-badge {
  display: inline-flex; align-items: center; gap: 4px;
  font-size: 11px; font-weight: 600; padding: 2px 8px;
  border-radius: var(--radius-full); white-space: nowrap;
}

.todo-item-checkbox { flex-shrink: 0; padding-top: 2px; }
.todo-empty { padding: 48px 20px; }

/* Result panel */
.result-content { padding: 20px; }

/* Action prompt */
.action-prompt { text-align: center; padding: 40px 20px; }
.action-prompt-info h4 { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin: 0 0 8px; }
.action-prompt-info p { font-size: 13px; color: var(--color-text-tertiary); margin: 0 0 24px; }
.action-prompt-buttons { display: flex; gap: 12px; justify-content: center; }

/* Result action bar */
.result-action-bar { display: flex; gap: 8px; margin-bottom: 16px; align-items: center; }

/* History badge */
.history-badge {
  display: flex; align-items: center; gap: 6px; font-size: 12px; font-weight: 600;
  padding: 4px 12px; border-radius: var(--radius-full);
  background: var(--color-bg-hover); color: var(--color-text-tertiary); margin-right: auto;
}

/* Loading */
.result-loading { display: flex; flex-direction: column; align-items: center; padding: 40px 20px; gap: 20px; }
.loading-process-info { text-align: center; }
.loading-process-title { font-size: 15px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 4px; }
.loading-process-meta { font-size: 13px; color: var(--color-text-tertiary); }

/* Two-phase audit progress cards (matches archive.vue style) */
.audit-progress { display: flex; flex-direction: column; gap: 12px; width: 100%; max-width: 400px; }
.audit-phase {
  display: flex; align-items: flex-start; gap: 14px; padding: 14px 16px;
  border-radius: var(--radius-md); border: 1px solid var(--color-border-light);
  background: var(--color-bg-page); transition: all 0.3s ease;
}
.audit-phase--active {
  border-color: var(--color-primary); background: var(--color-primary-bg);
  box-shadow: 0 2px 8px rgba(79, 70, 229, 0.1);
}
.audit-phase--done { border-color: var(--color-success); background: var(--color-success-bg); }
.audit-phase--pending { opacity: 0.5; }
.audit-phase-dot {
  width: 28px; height: 28px; border-radius: 50%; display: flex; align-items: center;
  justify-content: center; font-size: 16px; flex-shrink: 0;
  background: var(--color-bg-hover);
}
.audit-phase--active .audit-phase-dot { background: var(--color-primary-bg); color: var(--color-primary); }
.audit-phase--done .audit-phase-dot { background: var(--color-success-bg); color: var(--color-success); }
.phase-pending-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--color-border); }
.audit-phase-info { flex: 1; }
.audit-phase-title { font-size: 14px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 2px; }
.audit-phase-desc { font-size: 12px; color: var(--color-text-tertiary); }

/* Result banner */
.result-banner {
  display: flex; align-items: center; padding: 16px 20px;
  border-radius: var(--radius-lg); border-left: 4px solid; margin-bottom: 24px; gap: 14px;
}
.result-banner-icon { font-size: 28px; flex-shrink: 0; }
.result-banner-info { flex: 1; }
.result-banner-title { font-size: 16px; font-weight: 700; }
.result-banner-meta { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }
.result-score { font-size: 36px; font-weight: 800; line-height: 1; }

/* Rule checks */
.result-section { margin-bottom: 24px; }
.result-section-title { font-size: 14px; font-weight: 600; color: var(--color-text-primary); margin: 0 0 12px; }
.rule-checks { display: flex; flex-direction: column; gap: 8px; }
.rule-check-item {
  display: flex; gap: 12px; padding: 12px 16px;
  border-radius: var(--radius-md); border: 1px solid var(--color-border-light);
  transition: background var(--transition-fast);
}
.rule-check-item:hover { background: var(--color-bg-hover); }
.rule-check-item--pass { border-left: 3px solid var(--color-success); }
.rule-check-item--fail { border-left: 3px solid var(--color-danger); background: var(--color-danger-bg); }
.rule-check-status { font-size: 18px; flex-shrink: 0; padding-top: 1px; }
.rule-check-content { flex: 1; min-width: 0; }
.rule-check-name { font-size: 14px; font-weight: 500; color: var(--color-text-primary); margin-bottom: 4px; display: flex; align-items: center; gap: 8px; }
.rule-locked-badge { font-size: 10px; font-weight: 600; padding: 1px 6px; border-radius: var(--radius-full); background: var(--color-danger-bg); color: var(--color-danger); }
.rule-check-reasoning { font-size: 13px; color: var(--color-text-secondary); line-height: 1.5; }

/* Opt5: Risk + Suggestions parallel cards */
.risk-suggest-row {
  display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 24px;
}
.risk-suggest-row:has(.insight-card:only-child) {
  grid-template-columns: 1fr;
}
.insight-card {
  border-radius: var(--radius-md); padding: 16px;
  border: 1px solid var(--color-border-light);
}
.insight-card--risk {
  background: linear-gradient(135deg, rgba(239, 68, 68, 0.04), rgba(239, 68, 68, 0.01));
  border-color: rgba(239, 68, 68, 0.15);
}
.insight-card--suggest {
  background: linear-gradient(135deg, rgba(79, 70, 229, 0.04), rgba(79, 70, 229, 0.01));
  border-color: rgba(79, 70, 229, 0.15);
}
.insight-card-header {
  display: flex; align-items: center; gap: 8px;
  font-size: 13px; font-weight: 600; color: var(--color-text-primary);
  margin-bottom: 10px;
}
.insight-card-list {
  margin: 0; padding-left: 18px; display: flex; flex-direction: column; gap: 6px;
}
.insight-card-list li {
  font-size: 13px; line-height: 1.6; color: var(--color-text-secondary);
}
.insight-card--risk .insight-card-list li { color: var(--color-danger); }

/* AI Reasoning */
.ai-reasoning { background: var(--color-bg-page); border-radius: var(--radius-md); padding: 16px; border: 1px solid var(--color-border-light); }
.ai-reasoning pre { white-space: pre-wrap; word-break: break-word; font-family: var(--font-sans); font-size: 13px; line-height: 1.7; color: var(--color-text-secondary); margin: 0; }

/* Empty state */
.result-empty { text-align: center; padding: 60px 20px; }
.result-empty-icon {
  width: 64px; height: 64px; border-radius: 50%; background: var(--color-primary-bg);
  color: var(--color-primary); font-size: 28px; display: flex; align-items: center;
  justify-content: center; margin: 0 auto 16px;
}
.result-empty h4 { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin: 0 0 8px; }
.result-empty p { font-size: 13px; color: var(--color-text-tertiary); margin: 0 auto; max-width: 280px; }

/* Batch toolbar */
.batch-toolbar {
  display: flex; align-items: center; justify-content: space-between;
  padding: 6px 0; gap: 8px;
}
.batch-toolbar-left {
  display: flex; align-items: center; gap: 12px;
}
.batch-progress-hint {
  font-size: 12px; font-weight: 600; color: var(--color-primary);
  animation: auditPulse 1.5s ease-in-out infinite;
}
.batch-audit-btn {
  flex-shrink: 0;
}

/* Pagination */
.pagination-wrapper { padding: 12px 20px; border-top: 1px solid var(--color-border-light); display: flex; justify-content: center; }

@media (max-width: 1024px) {
  .dashboard-grid { grid-template-columns: 1fr; }
  .stats-row { grid-template-columns: repeat(4, 1fr); }
}
@media (max-width: 768px) {
  .stats-row { grid-template-columns: repeat(2, 1fr); gap: 12px; }
  .stat-card { padding: 14px; }
  .stat-card-value { font-size: 22px; }
  .stat-card-icon { width: 40px; height: 40px; font-size: 18px; }
  .page-title { font-size: 20px; }
  .dashboard-grid { gap: 16px; }
  .panel-header { padding: 12px 16px; }
  .todo-item { padding: 12px 16px; }
  .result-content { padding: 16px; }
  .action-prompt-buttons { flex-direction: column; }
  .risk-suggest-row { grid-template-columns: 1fr; }
  .filter-bar { flex-direction: column; align-items: stretch; }
}
@media (max-width: 480px) {
  .stats-row { grid-template-columns: 1fr 1fr; gap: 8px; }
  .stat-card { padding: 12px; gap: 10px; }
  .stat-card-value { font-size: 20px; }
  .stat-card-label { font-size: 11px; }
  .stat-card-icon { width: 36px; height: 36px; font-size: 16px; }
  .result-banner { flex-wrap: wrap; padding: 12px 14px; }
  .result-score { font-size: 28px; }
}

/* Drawer */
.drawer-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,0.4);
  backdrop-filter: blur(4px); z-index: 1000; display: flex; justify-content: flex-end;
}
.drawer-panel {
  width: 480px; max-width: 100vw; background: var(--color-bg-card);
  height: 100%; display: flex; flex-direction: column; box-shadow: -8px 0 30px rgba(0,0,0,0.12);
}
.drawer-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 20px 24px; border-bottom: 1px solid var(--color-border-light); flex-shrink: 0;
}
.drawer-header h3 { font-size: 16px; font-weight: 600; margin: 0; }
.drawer-close {
  width: 32px; height: 32px; border: none; background: transparent;
  border-radius: var(--radius-md); cursor: pointer; display: flex;
  align-items: center; justify-content: center; color: var(--color-text-tertiary);
  font-size: 16px; transition: all var(--transition-fast);
}
.drawer-close:hover { background: var(--color-bg-hover); color: var(--color-text-primary); }
.drawer-body { flex: 1; overflow-y: auto; padding: 24px; }
.chain-desc { font-size: 13px; color: var(--color-text-tertiary); margin: 0 0 20px; }

/* Audit chain timeline */
.audit-chain { display: flex; flex-direction: column; }
.chain-node { display: flex; gap: 16px; }
.chain-timeline { display: flex; flex-direction: column; align-items: center; width: 20px; flex-shrink: 0; }
.chain-dot { width: 12px; height: 12px; border-radius: 50%; flex-shrink: 0; }
.chain-line { width: 2px; flex: 1; background: var(--color-border-light); min-height: 20px; }
.chain-card {
  flex: 1; padding: 14px 16px; border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md); margin-bottom: 12px; transition: background var(--transition-fast);
}
.chain-card:hover { background: var(--color-bg-hover); }
.chain-card-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; }
.chain-tag {
  display: inline-flex; align-items: center; gap: 4px;
  font-size: 12px; font-weight: 600; padding: 3px 10px; border-radius: var(--radius-full);
}
.chain-score { font-size: 18px; font-weight: 700; color: var(--color-text-primary); }
.chain-card-meta { font-size: 12px; color: var(--color-text-tertiary); display: flex; align-items: center; gap: 8px; }
.chain-adopted { font-size: 11px; font-weight: 500; padding: 2px 8px; border-radius: var(--radius-full); }
.chain-adopted--yes { background: var(--color-success-bg); color: var(--color-success); }
.chain-adopted--no { background: var(--color-bg-hover); color: var(--color-text-tertiary); }

.drawer-enter-active { transition: opacity 0.2s ease; }
.drawer-enter-active .drawer-panel { transition: transform 0.3s cubic-bezier(0.16,1,0.3,1); }
.drawer-leave-active { transition: opacity 0.2s ease 0.1s; }
.drawer-leave-active .drawer-panel { transition: transform 0.2s ease; }
.drawer-enter-from { opacity: 0; }
.drawer-enter-from .drawer-panel { transform: translateX(100%); }
.drawer-leave-to { opacity: 0; }
.drawer-leave-to .drawer-panel { transform: translateX(100%); }

/* Chain expand */
.chain-expand-btn { margin-left: auto; font-size: 12px; color: var(--color-text-tertiary); }
.chain-detail {
  margin-top: 12px; padding-top: 12px; border-top: 1px solid var(--color-border-light);
  display: flex; flex-direction: column; gap: 8px;
}
.chain-rule-item {
  display: flex; gap: 8px; font-size: 12px; padding: 6px 8px;
  border-radius: var(--radius-sm); border: 1px solid var(--color-border-light);
}
.chain-rule--fail { background: var(--color-danger-bg); }
.chain-rule-name { font-weight: 600; color: var(--color-text-primary); margin-bottom: 2px; }
.chain-rule-reasoning { color: var(--color-text-secondary); }
.chain-reasoning { background: var(--color-bg-page); border-radius: var(--radius-sm); padding: 10px; }
.chain-reasoning pre { font-size: 12px; line-height: 1.6; color: var(--color-text-secondary); margin: 0; white-space: pre-wrap; word-break: break-word; font-family: var(--font-sans); }
.chain-no-detail { font-size: 12px; color: var(--color-text-tertiary); text-align: center; padding: 12px; }
.chain-section-title { font-size: 12px; font-weight: 600; color: var(--color-text-secondary); margin-bottom: 6px; }
</style>
