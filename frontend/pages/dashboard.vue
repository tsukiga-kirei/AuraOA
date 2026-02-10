<script setup lang="ts">
import {
  SearchOutlined,
  ThunderboltOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  EditOutlined,
  ClockCircleOutlined,
  RiseOutlined,
  FireOutlined,
  ArrowUpOutlined,
} from '@ant-design/icons-vue'

definePageMeta({ middleware: 'auth' })

const { mockProcesses, mockAuditResult, mockDashboardStats } = useMockData()

const todoList = ref(mockProcesses)
const currentResult = ref<typeof mockAuditResult | null>(null)
const loading = ref(false)
const selectedProcess = ref<string | null>(null)
const searchText = ref('')
const stats = ref(mockDashboardStats)

const filteredList = computed(() => {
  if (!searchText.value) return todoList.value
  const q = searchText.value.toLowerCase()
  return todoList.value.filter(
    p => p.title.toLowerCase().includes(q) || p.applicant.toLowerCase().includes(q)
  )
})

const handleAudit = async (processId: string) => {
  selectedProcess.value = processId
  loading.value = true
  // Simulate API delay
  await new Promise(resolve => setTimeout(resolve, 1500))
  currentResult.value = { ...mockAuditResult, process_id: processId }
  loading.value = false
}

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

    <!-- Stats row -->
    <div class="stats-row">
      <div class="stat-card stat-card--primary">
        <div class="stat-card-icon">
          <ClockCircleOutlined />
        </div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ stats.pendingCount }}</span>
          <span class="stat-card-label">待审核</span>
        </div>
      </div>
      <div class="stat-card stat-card--success">
        <div class="stat-card-icon">
          <CheckCircleOutlined />
        </div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ stats.todayApproved }}</span>
          <span class="stat-card-label">已通过</span>
        </div>
      </div>
      <div class="stat-card stat-card--warning">
        <div class="stat-card-icon">
          <EditOutlined />
        </div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ stats.todayRevised }}</span>
          <span class="stat-card-label">需修改</span>
        </div>
      </div>
      <div class="stat-card stat-card--danger">
        <div class="stat-card-icon">
          <CloseCircleOutlined />
        </div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ stats.todayRejected }}</span>
          <span class="stat-card-label">已驳回</span>
        </div>
      </div>
    </div>

    <!-- Main content area -->
    <div class="dashboard-grid">
      <!-- Left: Todo list -->
      <div class="todo-panel">
        <div class="panel-header">
          <h3 class="panel-title">
            <FireOutlined style="color: var(--color-primary);" />
            待办流程
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
            v-for="item in filteredList"
            :key="item.process_id"
            class="todo-item"
            :class="{ 'todo-item--selected': selectedProcess === item.process_id }"
            @click="handleAudit(item.process_id)"
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
              <span
                v-if="item.amount"
                class="todo-item-amount"
              >
                ¥{{ item.amount.toLocaleString() }}
              </span>
              <span
                class="urgency-tag"
                :style="{
                  color: urgencyConfig[item.urgency].color,
                  background: urgencyConfig[item.urgency].bg,
                }"
              >
                {{ urgencyConfig[item.urgency].label }}
              </span>
            </div>
          </div>

          <div v-if="filteredList.length === 0" class="todo-empty">
            <a-empty description="暂无待办流程" />
          </div>
        </div>
      </div>

      <!-- Right: Audit result -->
      <div class="result-panel">
        <div class="panel-header">
          <h3 class="panel-title">
            <ThunderboltOutlined style="color: var(--color-primary);" />
            审核结果
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

          <!-- Result display -->
          <template v-else-if="currentResult">
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
              <ThunderboltOutlined />
            </div>
            <h4>选择待办流程开始审核</h4>
            <p>点击左侧列表中的流程，AI 将自动进行规则校验并给出审核建议</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dashboard {
  animation: fadeIn 0.3s ease-out;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(8px); }
  to { opacity: 1; transform: translateY(0); }
}

/* Page header */
.page-header {
  margin-bottom: 24px;
}

.page-title {
  font-size: 24px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin: 0;
  letter-spacing: -0.02em;
}

.page-subtitle {
  font-size: 14px;
  color: var(--color-text-tertiary);
  margin: 4px 0 0;
}

/* Stats row */
.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 16px;
  margin-bottom: 24px;
}

.stat-card {
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  padding: 20px;
  display: flex;
  align-items: center;
  gap: 16px;
  border: 1px solid var(--color-border-light);
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

.stat-card--primary .stat-card-icon {
  background: var(--color-primary-bg);
  color: var(--color-primary);
}

.stat-card--success .stat-card-icon {
  background: var(--color-success-bg);
  color: var(--color-success);
}

.stat-card--warning .stat-card-icon {
  background: var(--color-warning-bg);
  color: var(--color-warning);
}

.stat-card--danger .stat-card-icon {
  background: var(--color-danger-bg);
  color: var(--color-danger);
}

.stat-card-info {
  display: flex;
  flex-direction: column;
}

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

/* Dashboard grid */
.dashboard-grid {
  display: grid;
  grid-template-columns: 420px 1fr;
  gap: 24px;
  align-items: start;
}

/* Panels */
.todo-panel,
.result-panel {
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light);
  overflow: hidden;
}

.panel-header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--color-border-light);
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.panel-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0;
  display: flex;
  align-items: center;
  gap: 8px;
}

.search-input {
  height: 36px;
}

/* Todo list */
.todo-list {
  max-height: calc(100vh - 380px);
  overflow-y: auto;
}

.todo-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 20px;
  cursor: pointer;
  transition: all var(--transition-fast);
  border-bottom: 1px solid var(--color-border-light);
  gap: 12px;
}

.todo-item:last-child {
  border-bottom: none;
}

.todo-item:hover {
  background: var(--color-bg-hover);
}

.todo-item--selected {
  background: var(--color-primary-bg);
  border-left: 3px solid var(--color-primary);
}

.todo-item-main {
  flex: 1;
  min-width: 0;
}

.todo-item-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  margin-bottom: 4px;
}

.todo-item-meta {
  font-size: 12px;
  color: var(--color-text-tertiary);
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}

.todo-item-dot {
  color: var(--color-border);
}

.todo-item-right {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
  flex-shrink: 0;
}

.todo-item-amount {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-primary);
}

.urgency-tag {
  font-size: 11px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: var(--radius-full);
}

.todo-empty {
  padding: 48px 20px;
}

/* Result panel */
.result-content {
  padding: 20px;
}

/* Loading */
.result-loading {
  display: flex;
  justify-content: center;
  padding: 60px 0;
}

.loading-animation {
  text-align: center;
}

.loading-pulse {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: var(--color-primary);
  margin: 0 auto 16px;
  animation: pulse 1.5s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { transform: scale(1); opacity: 0.6; }
  50% { transform: scale(1.15); opacity: 1; }
}

.loading-text {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin-bottom: 4px;
}

.loading-subtext {
  font-size: 13px;
  color: var(--color-text-tertiary);
}

/* Result banner */
.result-banner {
  display: flex;
  align-items: center;
  padding: 16px 20px;
  border-radius: var(--radius-lg);
  border-left: 4px solid;
  margin-bottom: 24px;
  gap: 14px;
}

.result-banner-icon {
  font-size: 28px;
  flex-shrink: 0;
}

.result-banner-info {
  flex: 1;
}

.result-banner-title {
  font-size: 16px;
  font-weight: 700;
}

.result-banner-meta {
  font-size: 12px;
  color: var(--color-text-tertiary);
  margin-top: 2px;
}

.result-score {
  font-size: 36px;
  font-weight: 800;
  line-height: 1;
}

/* Rule checks */
.result-section {
  margin-bottom: 24px;
}

.result-section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0 0 12px;
}

.rule-checks {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.rule-check-item {
  display: flex;
  gap: 12px;
  padding: 12px 16px;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border-light);
  transition: background var(--transition-fast);
}

.rule-check-item:hover {
  background: var(--color-bg-hover);
}

.rule-check-item--pass {
  border-left: 3px solid var(--color-success);
}

.rule-check-item--fail {
  border-left: 3px solid var(--color-danger);
  background: var(--color-danger-bg);
}

.rule-check-status {
  font-size: 18px;
  flex-shrink: 0;
  padding-top: 1px;
}

.rule-check-content {
  flex: 1;
  min-width: 0;
}

.rule-check-name {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-primary);
  margin-bottom: 4px;
  display: flex;
  align-items: center;
  gap: 8px;
}

.rule-locked-badge {
  font-size: 10px;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: var(--radius-full);
  background: var(--color-danger-bg);
  color: var(--color-danger);
}

.rule-check-reasoning {
  font-size: 13px;
  color: var(--color-text-secondary);
  line-height: 1.5;
}

/* AI Reasoning */
.ai-reasoning {
  background: var(--color-bg-page);
  border-radius: var(--radius-md);
  padding: 16px;
  border: 1px solid var(--color-border-light);
}

.ai-reasoning pre {
  white-space: pre-wrap;
  word-break: break-word;
  font-family: var(--font-sans);
  font-size: 13px;
  line-height: 1.7;
  color: var(--color-text-secondary);
  margin: 0;
}

/* Empty state */
.result-empty {
  text-align: center;
  padding: 60px 20px;
}

.result-empty-icon {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: var(--color-primary-bg);
  color: var(--color-primary);
  font-size: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 16px;
}

.result-empty h4 {
  font-size: 16px;
  font-weight: 600;
  color: var(--color-text-primary);
  margin: 0 0 8px;
}

.result-empty p {
  font-size: 13px;
  color: var(--color-text-tertiary);
  margin: 0;
  max-width: 280px;
  margin: 0 auto;
}

/* Responsive */
@media (max-width: 1024px) {
  .dashboard-grid {
    grid-template-columns: 1fr;
  }

  .stats-row {
    grid-template-columns: repeat(2, 1fr);
  }
}

@media (max-width: 640px) {
  .stats-row {
    grid-template-columns: 1fr 1fr;
    gap: 12px;
  }

  .stat-card {
    padding: 14px;
  }

  .stat-card-value {
    font-size: 22px;
  }
}
</style>
