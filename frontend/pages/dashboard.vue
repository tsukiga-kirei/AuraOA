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
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { useI18n } from '~/composables/useI18n'

definePageMeta({ middleware: 'auth' })

const { t } = useI18n()

const {
  mockProcesses, mockApprovedProcesses, mockRejectedProcesses,
  mockHistoricalResults, mockAuditResult, mockDashboardStats, mockSnapshots,
  mockArchivedOAProcesses, mockArchivedAuditChains, mockArchivedHistoricalResults,
  mockBatchAuditResult,
} = useMockData()

const todoList = ref(mockProcesses)
const approvedList = ref(mockApprovedProcesses)
const rejectedList = ref(mockRejectedProcesses)
const archivedList = ref(mockArchivedOAProcesses)
const currentResult = ref<typeof mockAuditResult | null>(null)
const loading = ref(false)
const phase1Done = ref(false)
const selectedProcess = ref<string | null>(null)
const searchText = ref('')
const stats = ref(mockDashboardStats)

// View mode
const viewMode = ref<'todo' | 'approved' | 'rejected' | 'archived'>('todo')
const isHistoryMode = computed(() => viewMode.value !== 'todo')

// Process type filter
const filterProcessType = ref<string[]>([])
const processTypeOptions = computed(() => {
  const all = [...todoList.value, ...approvedList.value, ...rejectedList.value, ...archivedList.value]
  const types = [...new Set(all.map(p => p.process_type))]
  return types.map(t => ({ label: t, value: t }))
})

// Batch audit
const selectedProcessIds = ref<string[]>([])
const batchAuditing = ref(false)
const batchProgress = ref(0)
const batchResult = ref<typeof mockBatchAuditResult | null>(null)
const showBatchResult = ref(false)

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

const handleBatchAudit = async () => {
  if (selectedProcessIds.value.length === 0) return
  batchAuditing.value = true
  batchProgress.value = 0
  showBatchResult.value = true
  // Simulate progress
  for (let i = 0; i <= 100; i += 10) {
    await new Promise(r => setTimeout(r, 150))
    batchProgress.value = i
  }
  batchResult.value = { ...mockBatchAuditResult, total: selectedProcessIds.value.length, completed: selectedProcessIds.value.length, status: 'done' as any, progress_percent: 100 }
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
    case 'rejected': list = rejectedList.value; break
    case 'archived': list = archivedList.value; break
    default: list = todoList.value
  }
  if (filterProcessType.value.length > 0) {
    list = list.filter(p => filterProcessType.value.includes(p.process_type))
  }
  if (searchText.value) {
    const q = searchText.value.toLowerCase()
    list = list.filter(p => p.title.toLowerCase().includes(q) || p.applicant.toLowerCase().includes(q))
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
    currentResult.value = null
    phase1Done.value = false
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
  currentResult.value = { ...mockAuditResult, process_id: processId }
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

const switchView = (mode: 'todo' | 'approved' | 'rejected' | 'archived') => {
  viewMode.value = mode
  selectedProcess.value = null
  currentResult.value = null
  listPage.value = 1
  selectedProcessIds.value = []
}

const selectedProcessInfo = computed(() => {
  const all = [...todoList.value, ...approvedList.value, ...rejectedList.value, ...archivedList.value]
  return all.find(p => p.process_id === selectedProcess.value)
})

const viewModeLabel = computed(() => {
  switch (viewMode.value) {
    case 'approved': return t('dashboard.viewMode.approved')
    case 'rejected': return t('dashboard.viewMode.rejected')
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
  reject: { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)', icon: CloseCircleOutlined, label: t('dashboard.rec.reject') },
  revise: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', icon: EditOutlined, label: t('dashboard.rec.revise') },
  return: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', icon: ReloadOutlined, label: t('dashboard.rec.return') },
  review: { color: 'var(--color-info)', bg: 'var(--color-info-bg)', icon: EyeOutlined, label: t('dashboard.rec.review') },
}))
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
        :class="{ 'stat-card--selected': viewMode === 'rejected' }"
        @click="switchView('rejected')"
      >
        <div class="stat-card-icon"><CloseCircleOutlined /></div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ stats.todayRejected }}</span>
          <span class="stat-card-label">{{ t('dashboard.tab.rejected') }}</span>
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
          <h3 class="panel-title">
            <FireOutlined v-if="viewMode === 'todo'" style="color: var(--color-primary);" />
            <FolderOpenOutlined v-else-if="viewMode === 'archived'" style="color: var(--color-info);" />
            <HistoryOutlined v-else style="color: var(--color-text-tertiary);" />
            {{ viewModeLabel }}
            <a-badge :count="filteredList.length" :number-style="{ backgroundColor: 'var(--color-primary)' }" />
          </h3>
          <a-input
            v-model:value="searchText"
            :placeholder="t('dashboard.searchPlaceholder')"
            allow-clear
            class="search-input"
          >
            <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
          </a-input>
          <!-- Process type filter -->
          <a-select
            v-model:value="filterProcessType"
            mode="multiple"
            :placeholder="t('dashboard.filterProcessTypePlaceholder')"
            allow-clear
            style="width: 100%;"
            :options="processTypeOptions"
          />
          <!-- Batch audit toolbar (todo mode only) -->
          <div v-if="viewMode === 'todo'" class="batch-toolbar">
            <a-checkbox
              :checked="selectedProcessIds.length === filteredList.length && filteredList.length > 0"
              :indeterminate="selectedProcessIds.length > 0 && selectedProcessIds.length < filteredList.length"
              @change="toggleSelectAll"
            >
              {{ selectedProcessIds.length > 0 ? t('dashboard.selected', `${selectedProcessIds.length}`) : t('dashboard.selectAll') }}
            </a-checkbox>
            <a-button
              v-if="selectedProcessIds.length > 0"
              type="primary"
              size="small"
              :loading="batchAuditing"
              @click="handleBatchAudit"
            >
              <ThunderboltOutlined /> {{ t('dashboard.batchAudit') }}
            </a-button>
          </div>
        </div>

        <!-- Batch progress -->
        <div v-if="showBatchResult" class="batch-progress-bar">
          <div class="batch-progress-header">
            <span>{{ t('dashboard.batchAuditTitle') }}</span>
            <button class="batch-close-btn" @click="showBatchResult = false">✕</button>
          </div>
          <a-progress :percent="batchResult?.progress_percent ?? batchProgress" :status="batchAuditing ? 'active' : 'success'" />
          <div v-if="batchResult && !batchAuditing" class="batch-summary">
            <span>{{ t('dashboard.batchTotal') }}: {{ batchResult.total }}</span>
            <span>{{ t('dashboard.batchCompleted') }}: {{ batchResult.completed }}</span>
            <span v-if="batchResult.failed > 0" style="color: var(--color-danger);">{{ t('dashboard.batchFailed') }}: {{ batchResult.failed }}</span>
          </div>
          <div v-else-if="batchAuditing" class="batch-summary">{{ t('dashboard.batchProcessing') }}</div>
        </div>

        <div class="todo-list">
          <div
            v-for="item in pagedList"
            :key="item.process_id"
            class="todo-item"
            :class="{ 'todo-item--selected': selectedProcess === item.process_id }"
            @click="handleSelectProcess(item.process_id)"
          >
            <div v-if="viewMode === 'todo'" class="todo-item-checkbox" @click.stop="toggleSelectProcess(item.process_id)">
              <a-checkbox :checked="selectedProcessIds.includes(item.process_id)" />
            </div>
            <div class="todo-item-main">
              <div class="todo-item-title">{{ item.title }}</div>
              <div class="todo-item-meta">
                <span>{{ item.applicant }}</span>
                <span class="todo-item-dot">·</span>
                <span>{{ item.department }}</span>
                <span class="todo-item-dot">·</span>
                <span>{{ item.submit_time }}</span>
              </div>
            </div>
            <div class="todo-item-right">
              <span class="todo-item-node">{{ item.current_node }}</span>
              <div class="todo-item-tags">
                <a-tooltip :title="t('dashboard.jumpToOA')" :mouse-enter-delay="0.5">
                  <button class="oa-jump-btn" @click.stop="jumpToOA(item.process_id)">
                    <ExportOutlined />
                  </button>
                </a-tooltip>
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
          <!-- Loading state -->
          <div v-if="loading" class="result-loading">
            <div class="loading-animation">
              <div class="loading-pulse" />
              <div class="loading-text">
                {{ phase1Done ? t('dashboard.phase2') : t('dashboard.phase1') }}
              </div>
              <div class="loading-subtext">{{ t('dashboard.aiAnalyzingSub') }}</div>
              <div class="phase-steps">
                <div class="phase-step" :class="{ 'phase-step--done': phase1Done, 'phase-step--active': !phase1Done }">
                  <span class="phase-step-dot" />
                  <span>{{ t('dashboard.phase1Duration') }}</span>
                </div>
                <div class="phase-step" :class="{ 'phase-step--active': phase1Done }">
                  <span class="phase-step-dot" />
                  <span>{{ t('dashboard.phase2Duration') }}</span>
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
            <!-- Action bar - different for history vs todo -->
            <div class="result-action-bar">
              <!-- History mode: OA jump + audit chain -->
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
              <!-- Todo mode: OA jump + re-audit -->
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

            <!-- v2 meta info -->
            <div v-if="currentResult.confidence != null" class="result-meta-row">
              <div class="result-meta-item">
                <span class="result-meta-label">{{ t('dashboard.confidence') }}</span>
                <a-progress
                  type="circle"
                  :percent="Math.round((currentResult.confidence ?? 0) * 100)"
                  :width="44"
                  :stroke-color="(currentResult.confidence ?? 0) >= 0.8 ? 'var(--color-success)' : 'var(--color-warning)'"
                />
              </div>
              <div class="result-meta-item">
                <span class="result-meta-label">{{ t('dashboard.modelUsed') }}</span>
                <span class="result-meta-value">{{ currentResult.model_used }}</span>
              </div>
              <div class="result-meta-item">
                <span class="result-meta-label">{{ t('dashboard.interactionMode') }}</span>
                <span class="result-meta-value">
                  {{ currentResult.interaction_mode === 'two_phase' ? t('dashboard.twoPhase') : t('dashboard.singlePass') }}
                </span>
              </div>
              <div v-if="currentResult.phase1_duration_ms" class="result-meta-item">
                <span class="result-meta-label">{{ t('dashboard.phase1Duration') }}</span>
                <span class="result-meta-value">{{ currentResult.phase1_duration_ms }}{{ t('dashboard.ms') }}</span>
              </div>
              <div v-if="currentResult.phase2_duration_ms" class="result-meta-item">
                <span class="result-meta-label">{{ t('dashboard.phase2Duration') }}</span>
                <span class="result-meta-value">{{ currentResult.phase2_duration_ms }}{{ t('dashboard.ms') }}</span>
              </div>
              <div class="result-meta-item result-meta-item--full">
                <span class="result-meta-label">{{ t('dashboard.traceId') }}</span>
                <span class="result-meta-value result-meta-mono">{{ currentResult.trace_id }}</span>
              </div>
            </div>

            <!-- Risk points & suggestions -->
            <div v-if="currentResult.risk_points?.length" class="result-section">
              <h4 class="result-section-title" style="color: var(--color-danger);">
                <CloseCircleOutlined /> {{ t('dashboard.riskPoints') }}
              </h4>
              <ul class="result-list result-list--danger">
                <li v-for="(rp, i) in currentResult.risk_points" :key="i">{{ rp }}</li>
              </ul>
            </div>

            <div v-if="currentResult.suggestions?.length" class="result-section">
              <h4 class="result-section-title" style="color: var(--color-primary);">
                <InfoCircleOutlined /> {{ t('dashboard.suggestions') }}
              </h4>
              <ul class="result-list result-list--primary">
                <li v-for="(sg, i) in currentResult.suggestions" :key="i">{{ sg }}</li>
              </ul>
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
                    <!-- Expanded detail -->
                    <div v-if="expandedChainNodes.has(snap.snapshot_id)" class="chain-detail">
                      <template v-if="mockHistoricalResults[snap.process_id] || mockArchivedHistoricalResults[snap.process_id]">
                        <div v-for="rule in (mockHistoricalResults[snap.process_id] || mockArchivedHistoricalResults[snap.process_id])?.details" :key="rule.rule_id" class="chain-rule-item" :class="rule.passed ? 'chain-rule--pass' : 'chain-rule--fail'">
                          <component :is="rule.passed ? CheckCircleOutlined : CloseCircleOutlined" :style="{ color: rule.passed ? 'var(--color-success)' : 'var(--color-danger)' }" />
                          <div>
                            <div class="chain-rule-name">{{ rule.rule_name }}</div>
                            <div class="chain-rule-reasoning">{{ rule.reasoning }}</div>
                          </div>
                        </div>
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
.search-input { height: 36px; }

/* Todo list */
.todo-list { max-height: calc(100vh - 380px); overflow-y: auto; }
.todo-item {
  display: flex; align-items: center; justify-content: space-between;
  padding: 14px 20px; cursor: pointer; transition: all var(--transition-fast);
  border-bottom: 1px solid var(--color-border-light); gap: 12px;
}
.todo-item:last-child { border-bottom: none; }
.todo-item:hover { background: var(--color-bg-hover); }
.todo-item--selected { background: var(--color-primary-bg); border-left: 3px solid var(--color-primary); }
.todo-item-main { flex: 1; min-width: 0; }
.todo-item-title {
  font-size: 14px; font-weight: 500; color: var(--color-text-primary);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; margin-bottom: 4px;
}
.todo-item-meta { font-size: 12px; color: var(--color-text-tertiary); display: flex; align-items: center; gap: 4px; flex-wrap: wrap; }
.todo-item-dot { color: var(--color-border); }
.todo-item-right { display: flex; flex-direction: column; align-items: flex-end; gap: 6px; flex-shrink: 0; }
.todo-item-amount { font-size: 13px; font-weight: 600; color: var(--color-text-primary); }
.todo-item-tags { display: flex; align-items: center; gap: 6px; }
.urgency-tag { font-size: 11px; font-weight: 600; padding: 2px 8px; border-radius: var(--radius-full); }
.oa-jump-btn {
  width: 24px; height: 24px; border: 1px solid var(--color-border);
  background: transparent; border-radius: var(--radius-sm); cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  font-size: 12px; color: var(--color-text-tertiary);
  transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  outline: none;
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
.oa-jump-btn:active {
  transform: scale(0.95);
}
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
.result-loading { display: flex; justify-content: center; padding: 60px 0; }
.loading-animation { text-align: center; }
.loading-pulse {
  width: 48px; height: 48px; border-radius: 50%; background: var(--color-primary);
  margin: 0 auto 16px; animation: pulse 1.5s ease-in-out infinite;
}
@keyframes pulse { 0%, 100% { transform: scale(1); opacity: 0.6; } 50% { transform: scale(1.15); opacity: 1; } }
.loading-text { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 4px; }
.loading-subtext { font-size: 13px; color: var(--color-text-tertiary); }

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
}
@media (max-width: 480px) {
  .stats-row { grid-template-columns: 1fr 1fr; gap: 8px; }
  .stat-card { padding: 12px; gap: 10px; }
  .stat-card-value { font-size: 20px; }
  .stat-card-label { font-size: 11px; }
  .stat-card-icon { width: 36px; height: 36px; font-size: 16px; }
  .todo-item-right { display: none; }
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

/* Batch toolbar */
.batch-toolbar {
  display: flex; align-items: center; justify-content: space-between;
  padding: 6px 0; gap: 8px;
}
.batch-progress-bar {
  padding: 12px 20px; border-bottom: 1px solid var(--color-border-light);
  background: var(--color-bg-hover);
}
.batch-progress-header {
  display: flex; justify-content: space-between; align-items: center;
  font-size: 13px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 8px;
}
.batch-close-btn {
  border: none; background: transparent; cursor: pointer;
  color: var(--color-text-tertiary); font-size: 14px; padding: 2px 6px;
}
.batch-summary {
  display: flex; gap: 16px; font-size: 12px; color: var(--color-text-tertiary); margin-top: 6px;
}
.todo-item-checkbox { flex-shrink: 0; }

/* Two-phase loading steps */
.phase-steps { display: flex; flex-direction: column; gap: 8px; margin-top: 16px; text-align: left; }
.phase-step {
  display: flex; align-items: center; gap: 8px;
  font-size: 13px; color: var(--color-text-tertiary);
}
.phase-step-dot {
  width: 8px; height: 8px; border-radius: 50%; background: var(--color-border);
  flex-shrink: 0; transition: background 0.3s;
}
.phase-step--active .phase-step-dot { background: var(--color-primary); animation: pulse 1s infinite; }
.phase-step--active { color: var(--color-primary); font-weight: 600; }
.phase-step--done .phase-step-dot { background: var(--color-success); }
.phase-step--done { color: var(--color-success); }

/* v2 result meta */
.result-meta-row {
  display: flex; flex-wrap: wrap; gap: 12px; margin-bottom: 20px;
  padding: 12px 16px; background: var(--color-bg-hover);
  border-radius: var(--radius-md); border: 1px solid var(--color-border-light);
}
.result-meta-item {
  display: flex; flex-direction: column; align-items: center; gap: 4px; min-width: 80px;
}
.result-meta-item--full { flex-direction: row; align-items: center; gap: 8px; width: 100%; min-width: unset; }
.result-meta-label { font-size: 11px; color: var(--color-text-tertiary); }
.result-meta-value { font-size: 13px; font-weight: 600; color: var(--color-text-primary); }
.result-meta-mono { font-family: monospace; font-size: 12px; color: var(--color-text-secondary); }

/* Risk / suggestion lists */
.result-list { margin: 0; padding-left: 20px; display: flex; flex-direction: column; gap: 4px; }
.result-list li { font-size: 13px; line-height: 1.6; }
.result-list--danger li { color: var(--color-danger); }
.result-list--primary li { color: var(--color-text-secondary); }

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

/* Node tag */
.todo-item-node {
  font-size: 11px; font-weight: 500; padding: 2px 8px;
  border-radius: var(--radius-full); background: var(--color-bg-hover);
  color: var(--color-text-secondary); white-space: nowrap;
}
</style>
