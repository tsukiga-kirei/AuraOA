<script setup lang="ts">
import {
  SearchOutlined,
  DownloadOutlined,
  FilterOutlined,
  EyeOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  EditOutlined,
} from '@ant-design/icons-vue'

definePageMeta({ middleware: 'auth' })

const { mockSnapshots } = useMockData()

const results = ref([...mockSnapshots])
const loading = ref(false)
const showFilters = ref(false)
const selectedSnapshot = ref<any>(null)

const filters = ref({
  department: undefined as string | undefined,
  recommendation: undefined as string | undefined,
  dateRange: null as any,
})

const filteredResults = computed(() => {
  let data = [...results.value]
  if (filters.value.department) {
    data = data.filter(r => r.department === filters.value.department)
  }
  if (filters.value.recommendation) {
    data = data.filter(r => r.recommendation === filters.value.recommendation)
  }
  return data
})

const search = () => {
  loading.value = true
  setTimeout(() => { loading.value = false }, 500)
}

const clearFilters = () => {
  filters.value = { department: undefined, recommendation: undefined, dateRange: null }
}

const recommendationConfig: Record<string, { color: string; bg: string; icon: any; label: string }> = {
  approve: { color: '#10b981', bg: '#ecfdf5', icon: CheckCircleOutlined, label: '通过' },
  reject: { color: '#ef4444', bg: '#fef2f2', icon: CloseCircleOutlined, label: '驳回' },
  revise: { color: '#f59e0b', bg: '#fffbeb', icon: EditOutlined, label: '修改' },
}

const departments = ['IT部', '销售部', '研发部', '行政部', '人力资源部', '市场部']
</script>

<template>
  <div class="archive-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title">归档复盘</h1>
        <p class="page-subtitle">历史审核记录检索与合规复核</p>
      </div>
      <div class="page-header-actions">
        <a-button @click="showFilters = !showFilters">
          <FilterOutlined /> 筛选
        </a-button>
        <a-dropdown>
          <a-button>
            <DownloadOutlined /> 导出
          </a-button>
          <template #overlay>
            <a-menu>
              <a-menu-item key="json">导出 JSON</a-menu-item>
              <a-menu-item key="csv">导出 CSV</a-menu-item>
              <a-menu-item key="excel">导出 Excel</a-menu-item>
            </a-menu>
          </template>
        </a-dropdown>
      </div>
    </div>

    <!-- Filters -->
    <transition name="slide">
      <div v-if="showFilters" class="filter-bar">
        <a-select
          v-model:value="filters.department"
          placeholder="选择部门"
          allow-clear
          style="width: 160px;"
        >
          <a-select-option v-for="d in departments" :key="d" :value="d">{{ d }}</a-select-option>
        </a-select>
        <a-select
          v-model:value="filters.recommendation"
          placeholder="审核建议"
          allow-clear
          style="width: 140px;"
        >
          <a-select-option value="approve">通过</a-select-option>
          <a-select-option value="reject">驳回</a-select-option>
          <a-select-option value="revise">修改</a-select-option>
        </a-select>
        <a-button type="primary" @click="search" :loading="loading">
          <SearchOutlined /> 检索
        </a-button>
        <a-button @click="clearFilters">重置</a-button>
      </div>
    </transition>

    <!-- Results -->
    <div class="archive-list">
      <div
        v-for="item in filteredResults"
        :key="item.snapshot_id"
        class="archive-item"
      >
        <div class="archive-item-left">
          <div
            class="archive-item-indicator"
            :style="{ background: recommendationConfig[item.recommendation]?.color }"
          />
          <div class="archive-item-info">
            <div class="archive-item-title">{{ item.title }}</div>
            <div class="archive-item-meta">
              <span>{{ item.applicant }}</span>
              <span class="meta-dot">·</span>
              <span>{{ item.department }}</span>
              <span class="meta-dot">·</span>
              <span>{{ item.created_at }}</span>
              <span class="meta-dot">·</span>
              <span style="font-family: var(--font-mono); font-size: 11px;">{{ item.process_id }}</span>
            </div>
          </div>
        </div>
        <div class="archive-item-right">
          <div class="archive-item-score">
            <span class="score-value">{{ item.score }}</span>
            <span class="score-label">分</span>
          </div>
          <span
            class="recommendation-tag"
            :style="{
              color: recommendationConfig[item.recommendation]?.color,
              background: recommendationConfig[item.recommendation]?.bg,
            }"
          >
            <component :is="recommendationConfig[item.recommendation]?.icon" />
            {{ recommendationConfig[item.recommendation]?.label }}
          </span>
          <span
            v-if="item.adopted !== null"
            class="adopted-tag"
            :class="item.adopted ? 'adopted-tag--yes' : 'adopted-tag--no'"
          >
            {{ item.adopted ? '已采纳' : '未采纳' }}
          </span>
          <button class="view-btn" @click="selectedSnapshot = item">
            <EyeOutlined /> 详情
          </button>
        </div>
      </div>

      <div v-if="filteredResults.length === 0" style="padding: 48px;">
        <a-empty description="暂无匹配记录" />
      </div>
    </div>

    <!-- Detail drawer -->
    <SnapshotDetail :snapshot="selectedSnapshot" @close="selectedSnapshot = null" />
  </div>
</template>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: 24px;
}

.page-title {
  font-size: 24px;
  font-weight: 700;
  color: var(--color-text-primary);
  margin: 0;
}

.page-subtitle {
  font-size: 14px;
  color: var(--color-text-tertiary);
  margin: 4px 0 0;
}

.page-header-actions {
  display: flex;
  gap: 8px;
}

/* Filters */
.filter-bar {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 16px 20px;
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light);
  margin-bottom: 20px;
  flex-wrap: wrap;
}

.slide-enter-active,
.slide-leave-active {
  transition: all 0.2s ease;
}

.slide-enter-from,
.slide-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

/* Archive list */
.archive-list {
  background: var(--color-bg-card);
  border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light);
  overflow: hidden;
}

.archive-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--color-border-light);
  transition: background var(--transition-fast);
  gap: 16px;
}

.archive-item:last-child {
  border-bottom: none;
}

.archive-item:hover {
  background: var(--color-bg-hover);
}

.archive-item-left {
  display: flex;
  align-items: center;
  gap: 14px;
  flex: 1;
  min-width: 0;
}

.archive-item-indicator {
  width: 4px;
  height: 36px;
  border-radius: 2px;
  flex-shrink: 0;
}

.archive-item-title {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-primary);
  margin-bottom: 4px;
}

.archive-item-meta {
  font-size: 12px;
  color: var(--color-text-tertiary);
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}

.meta-dot {
  color: var(--color-border);
}

.archive-item-right {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.archive-item-score {
  text-align: center;
}

.score-value {
  font-size: 20px;
  font-weight: 700;
  color: var(--color-text-primary);
}

.score-label {
  font-size: 11px;
  color: var(--color-text-tertiary);
  margin-left: 2px;
}

.recommendation-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  font-weight: 600;
  padding: 4px 12px;
  border-radius: var(--radius-full);
}

.adopted-tag {
  font-size: 11px;
  font-weight: 500;
  padding: 3px 10px;
  border-radius: var(--radius-full);
}

.adopted-tag--yes {
  background: var(--color-success-bg);
  color: var(--color-success);
}

.adopted-tag--no {
  background: var(--color-bg-hover);
  color: var(--color-text-tertiary);
}

.view-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 6px 14px;
  border: 1px solid var(--color-border);
  background: var(--color-bg-card);
  border-radius: var(--radius-md);
  font-size: 13px;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
}

.view-btn:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
  background: var(--color-primary-bg);
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    gap: 16px;
  }

  .archive-item {
    flex-direction: column;
    align-items: flex-start;
  }

  .archive-item-right {
    flex-wrap: wrap;
    margin-top: 8px;
  }
}
</style>
