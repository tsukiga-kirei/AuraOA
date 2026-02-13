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
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'

definePageMeta({ middleware: 'auth' })

const {
  mockProcesses, mockApprovedProcesses, mockRejectedProcesses,
  mockHistoricalResults, mockAuditResult, mockDashboardStats, mockSnapshots,
  mockArchivedOAProcesses, mockArchivedAuditChains, mockArchivedHistoricalResults,
} = useMockData()

const todoList = ref(mockProcesses)
const approvedList = ref(mockApprovedProcesses)
const rejectedList = ref(mockRejectedProcesses)
const archivedList = ref(mockArchivedOAProcesses)
const currentResult = ref<typeof mockAuditResult | null>(null)
const loading = ref(false)
const selectedProcess = ref<string | null>(null)
const searchText = ref('')
const stats = ref(mockDashboardStats)

// View mode: 'todo' | 'approved' | 'rejected' | 'archived'
const viewMode = ref<'todo' | 'approved' | 'rejected' | 'archived'>('todo')

// Whether current view is history-only (no audit actions)
const isHistoryMode = computed(() => viewMode.value !== 'todo')

// Audit history chain: multiple audit snapshots for a single process
const showHistoryChain = ref(false)
const historyChainProcessId = ref<string | null>(null)

// Get audit chain for a process (multi-round audit snapshots)
const getAuditChain = (processId: string) => {
  // Check archived chains first
  const archivedChain = mockArchivedAuditChains[processId]
  if (archivedChain && archivedChain.length > 0) return archivedChain
  // Then check regular snapshots
  const snapshots = mockSnapshots.filter(s => s.process_id === processId)
  if (snapshots.length > 0) return snapshots
  // Fallback: generate from historical result
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
  if (searchText.value) {
    const q = searchText.value.toLowerCase()
    list = list.filter(
      p => p.title.toLowerCase().includes(q) || p.applicant.toLowerCase().includes(q)
    )
  }
  return list
})

// Pagination
const { paged: pagedList, current: listPage, pageSize: listPageSize, total: listTotal, onChange: onListPageChange } = usePagination(filteredList, 10)

const handleSelectProcess = (processId: string) => {
  selectedProcess.value = processId
  if (isHistoryMode.value) {
    // Auto-load historical result for approved/rejected/archived
    const hist = mockHistoricalResults[processId] || mockArchivedHistoricalResults[processId]
    currentResult.value = hist ? { ...hist } : null
  } else {
    currentResult.value = null
  }
}

const handleAudit = async (processId: string) => {
  loading.value = true
  await new Promise(resolve => setTimeout(resolve, 1500))
  currentResult.value = { ...mockAuditResult, process_id: processId }
  loading.value = false
}

const handleReAudit = async () => {
  if (!selectedProcess.value) return
  currentResult.value = null
  await handleAudit(selectedProcess.value)
}

const jumpToOA = (processId: string) => {
  message.info(`正在跳转 OA 系统查看流程 ${processId}...`)
}

const switchView = (mode: 'todo' | 'approved' | 'rejected' | 'archived') => {
  viewMode.value = mode
  selectedProcess.value = null
  currentResult.value = null
  listPage.value = 1
}

const selectedProcessInfo = computed(() => {
  const all = [...todoList.value, ...approvedList.value, ...rejectedList.value, ...archivedList.value]
  return all.find(p => p.process_id === selectedProcess.value)
})

const viewModeLabel = computed(() => {
  switch (viewMode.value) {
    case 'approved': return '已通过流程'
    case 'rejected': return '已驳回流程'
    case 'archived': return '已归档流程'
    default: return '待办流程'
  }
})

const urgencyConfig = {
  high: { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)', label: '紧急' },
  medium: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', label: '一般' },
  low: { color: 'var(--color-success)', bg: 'var(--color-success-bg)', label: '低' },
}

const recommendationConfig = {
  approve: { color: 'var(--color-success)', bg: 'var(--color-success-bg)', icon: CheckCircleOutlined, label: '建议通过' },
  reject: { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)', icon: CloseCircleOutlined, label: '建议驳回' },
  revise: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', icon: EditOutlined, label: '建议修改' },
}
</script>

<template>
  <div class="dashboard">
    <!-- Page header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">审核工作台</h1>
        <p class="page-subtitle">智能待办审核 · 今日已处理 {{ stats.todayAudits }} 条</p>
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
          <span class="stat-card-label">待审核</span>
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
          <span class="stat-card-label">已通过</span>
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
          <span class="stat-card-label">已驳回</span>
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
          <span class="stat-card-label">已归档</span>
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
            placeholder="搜索流程或申请人..."
            allow-clear
            class="search-input"
          >
            <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
          </a-input>
        </div>

        <div class="todo-list">
          <div
            v-for="item in pagedList"
            :key="item.process_id"
            class="todo-item"
            :class="{ 'todo-item--selected': selectedProcess === item.process_id }"
            @click="handleSelectProcess(item.process_id)"
          >
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
              <span v-if="item.amount" class="todo-item-amount">
                ¥{{ item.amount.toLocaleString() }}
              </span>
              <div class="todo-item-tags">
                <span
                  class="urgency-tag"
                  :style="{
                    color: urgencyConfig[item.urgency].color,
                    background: urgencyConfig[item.urgency].bg,
                  }"
                >
                  {{ urgencyConfig[item.urgency].label }}
                </span>
                <a-tooltip title="跳转 OA 系统" :mouse-enter-delay="0.5">
                  <button class="oa-jump-btn" @click.stop="jumpToOA(item.process_id)">
                    <ExportOutlined />
                  </button>
                </a-tooltip>
              </div>
            </div>
          </div>

          <div v-if="filteredList.length === 0" class="todo-empty">
            <a-empty description="暂无流程" />
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
            {{ viewMode === 'archived' ? '归档审核记录' : isHistoryMode ? '历史审核结果' : '审核结果' }}
          </h3>
        </div>

        <div class="result-content">
          <!-- Loading state -->
          <div v-if="loading" class="result-loading">
            <div class="loading-animation">
              <div class="loading-pulse" />
              <div class="loading-text">AI 正在分析审核中...</div>
              <div class="loading-subtext">正在校验规则并生成建议</div>
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
                  <ThunderboltOutlined /> 开始 AI 审核
                </a-button>
                <a-button size="large" @click="jumpToOA(selectedProcess!)">
                  <ExportOutlined /> 跳转 OA 系统
                </a-button>
              </div>
            </div>
          </template>

          <!-- History mode: Selected but no historical result found -->
          <template v-else-if="isHistoryMode && selectedProcess && !currentResult">
            <div class="result-empty">
              <div class="result-empty-icon"><HistoryOutlined /></div>
              <h4>暂无历史审核记录</h4>
              <p>该流程尚未生成 AI 审核结果</p>
              <a-button style="margin-top: 16px;" @click="jumpToOA(selectedProcess!)">
                <ExportOutlined /> 跳转 OA 系统查看
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
                  {{ viewMode === 'archived' ? '归档记录（只读）' : '历史记录（只读）' }}
                </div>
                <a-button @click="openHistoryChain(currentResult.process_id)">
                  <EyeOutlined /> 审核链
                </a-button>
                <a-button type="primary" @click="jumpToOA(currentResult.process_id)">
                  <ExportOutlined /> 跳转 OA
                </a-button>
              </template>
              <!-- Todo mode: OA jump + re-audit -->
              <template v-else>
                <a-button @click="jumpToOA(currentResult.process_id)">
                  <ExportOutlined /> 跳转 OA
                </a-button>
                <a-button @click="handleReAudit">
                  <ReloadOutlined /> 重新审核
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
                <div
                  class="result-banner-title"
                  :style="{ color: recommendationConfig[currentResult.recommendation].color }"
                >
                  {{ recommendationConfig[currentResult.recommendation].label }}
                </div>
                <div class="result-banner-meta">
                  综合评分 {{ currentResult.score }} 分 · 耗时 {{ currentResult.duration_ms }}ms · {{ currentResult.trace_id }}
                </div>
              </div>
              <div class="result-score" :style="{ color: recommendationConfig[currentResult.recommendation].color }">
                {{ currentResult.score }}
              </div>
            </div>

            <!-- Rule checks -->
            <div class="result-section">
              <h4 class="result-section-title">规则校验详情</h4>
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
                      <span v-if="rule.is_locked" class="rule-locked-badge">强制</span>
                    </div>
                    <div class="rule-check-reasoning">{{ rule.reasoning }}</div>
                  </div>
                </div>
              </div>
            </div>

            <!-- AI Reasoning -->
            <div class="result-section">
              <h4 class="result-section-title">AI 推理分析</h4>
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
            <h4>{{ viewMode === 'archived' ? '选择归档流程查看审核链' : isHistoryMode ? '选择流程查看历史审核结果' : '选择待办流程开始审核' }}</h4>
            <p>{{ viewMode === 'archived' ? '点击左侧归档流程，查看完整的多轮 AI 审核记录链' : isHistoryMode ? '点击左侧列表中的流程，查看 AI 历史审核记录' : '点击左侧列表中的流程，AI 将自动进行规则校验并给出审核建议' }}</p>
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
              <h3>审核历史链</h3>
              <button class="drawer-close" @click="showHistoryChain = false">✕</button>
            </div>
            <div class="drawer-body">
              <p class="chain-desc">该流程的所有 AI 审核记录（按时间倒序）</p>
              <div v-if="currentAuditChain.length === 0" style="padding: 40px; text-align: center;">
                <a-empty description="暂无审核记录" />
              </div>
              <div v-else class="audit-chain">
                <div
                  v-for="(snap, idx) in currentAuditChain"
                  :key="snap.snapshot_id"
                  class="chain-node"
                >
                  <div class="chain-timeline">
                    <div
                      class="chain-dot"
                      :style="{ background: recommendationConfig[snap.recommendation]?.color }"
                    />
                    <div v-if="idx < currentAuditChain.length - 1" class="chain-line" />
                  </div>
                  <div class="chain-card">
                    <div class="chain-card-header">
                      <span
                        class="chain-tag"
                        :style="{
                          color: recommendationConfig[snap.recommendation]?.color,
                          background: recommendationConfig[snap.recommendation]?.bg,
                        }"
                      >
                        <component :is="recommendationConfig[snap.recommendation]?.icon" />
                        {{ recommendationConfig[snap.recommendation]?.label }}
                      </span>
                      <span class="chain-score">{{ snap.score }}分</span>
                    </div>
                    <div class="chain-card-meta">
                      {{ snap.created_at }}
                      <span v-if="snap.adopted !== null" class="chain-adopted" :class="snap.adopted ? 'chain-adopted--yes' : 'chain-adopted--no'">
                        {{ snap.adopted ? '已采纳' : '未采纳' }}
                      </span>
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
</style>
