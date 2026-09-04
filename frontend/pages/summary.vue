<script setup lang="ts">
import {
  FileTextOutlined,
  SearchOutlined,
  ReloadOutlined,
  ThunderboltOutlined,
  EyeOutlined,
  ClockCircleOutlined,
  CheckCircleOutlined,
  ExclamationCircleOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import type { SummaryResult, SummaryWorkbenchProcessItem, SummaryWorkbenchStats } from '~/types/process-summary'
import type { ProcessListItem } from '~/types/user-config'

definePageMeta({ middleware: 'auth' })

const { t, locale } = useI18n()
const { listProcesses, getStats, execute, waitJob } = useSummaryWorkbenchApi()
const settingsApi = useSettingsApi()

const loading = ref(false)
const executing = ref<Record<string, boolean>>({})
const items = ref<SummaryWorkbenchProcessItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keyword = ref('')
const applicant = ref('')
const processType = ref('')
const summaryStatus = ref('')
const summaryConfigs = ref<ProcessListItem[]>([])
const selected = ref<SummaryWorkbenchProcessItem | null>(null)
const currentResult = ref<SummaryResult | null>(null)
const detailOpen = ref(false)
const stats = ref<SummaryWorkbenchStats>({
  total_count: 0,
  summarized_count: 0,
  pending_count: 0,
  running_count: 0,
  failed_count: 0,
})

const query = computed(() => ({
  keyword: keyword.value || undefined,
  applicant: applicant.value || undefined,
  process_type: processType.value || undefined,
  summary_status: summaryStatus.value || undefined,
  page: page.value,
  page_size: pageSize.value,
}))

const processTypeOptions = computed(() => {
  return summaryConfigs.value.map(config => ({
    value: config.process_type,
    label: config.process_type_label || config.process_type,
  }))
})

const visibleBlocks = computed(() => {
  const blocks = currentResult.value?.blocks || []
  const visibleIds = selected.value?.visible_block_ids || []
  if (visibleIds.length === 0) return blocks
  const allowed = new Set(visibleIds)
  return blocks.filter(block => allowed.has(block.block_id))
})

const loadData = async () => {
  loading.value = true
  try {
    const [list, summaryStats] = await Promise.all([listProcesses(query.value), getStats(query.value)])
    items.value = list.items || []
    total.value = list.total || 0
    stats.value = summaryStats
  }
  catch (e: any) {
    message.error(e?.message || t('summary.loadFailed'))
  }
  finally { loading.value = false }
}

const loadProcessTypeOptions = async () => {
  try { summaryConfigs.value = await settingsApi.listSummaryConfigs() }
  catch { summaryConfigs.value = [] }
}

const applyFilters = () => {
  page.value = 1
  void loadData()
}

const resetFilters = () => {
  keyword.value = ''
  applicant.value = ''
  processType.value = ''
  summaryStatus.value = ''
  page.value = 1
  void loadData()
}

const handlePageChange = () => { void loadData() }
const handlePageSizeChange = () => {
  page.value = 1
  void loadData()
}

const openResult = (item: SummaryWorkbenchProcessItem) => {
  selected.value = item
  currentResult.value = item.summary_result || null
  detailOpen.value = true
  if (item.running_job_id) void resumeJob(item)
}

const resumeJob = async (item: SummaryWorkbenchProcessItem) => {
  if (!item.running_job_id || executing.value[item.process_id]) return
  executing.value[item.process_id] = true
  try {
    const result = await waitJob(item.running_job_id, (progress) => {
      if (selected.value?.process_id === item.process_id) currentResult.value = progress
    })
    if (selected.value?.process_id === item.process_id) currentResult.value = result
    if (result.status === 'failed') message.error(result.error_message || t('summary.executeFailed'))
    await loadData()
  }
  finally { executing.value[item.process_id] = false }
}

const runSummary = async (item: SummaryWorkbenchProcessItem, useLatestConfig = false) => {
  if (executing.value[item.process_id]) return
  selected.value = item
  currentResult.value = item.summary_result || null
  detailOpen.value = true
  executing.value[item.process_id] = true
  try {
    const result = await execute({
      process_id: item.process_id,
      process_type: item.process_type,
      title: item.title,
    }, useLatestConfig, (progress) => { currentResult.value = progress })
    currentResult.value = result
    if (result.status === 'completed') message.success(t('summary.executeSuccess'))
    else message.error(result.error_message || t('summary.executeFailed'))
    await loadData()
  }
  catch (e: any) { message.error(e?.message || t('summary.executeFailed')) }
  finally { executing.value[item.process_id] = false }
}

const statusColor = (item: SummaryWorkbenchProcessItem) => {
  if (['pending', 'assembling', 'reasoning', 'extracting'].includes(item.summary_status) && item.running_job_id) return 'processing'
  if (item.summary_status === 'failed') return 'error'
  if (item.has_summary) return 'success'
  return 'default'
}

const statusLabel = (item: SummaryWorkbenchProcessItem) => {
  if (['assembling', 'reasoning', 'extracting'].includes(item.summary_status) || item.running_job_id) return t('summary.status.running')
  if (item.summary_status === 'failed') return t('summary.status.failed')
  if (item.has_summary) return t('summary.status.completed')
  return t('summary.status.pending')
}

const formatTime = (value?: string) => value
  ? formatDateTimeInAppZone(value, locale.value === 'en-US' ? 'en-US' : 'zh-CN')
  : '-'

onMounted(() => {
  void loadData()
  void loadProcessTypeOptions()
})
</script>

<template>
  <div class="summary-page fade-in">
    <div class="page-header">
      <div>
        <h1 class="page-title"><FileTextOutlined /> {{ t('summary.title') }}</h1>
        <p class="page-subtitle">{{ t('summary.subtitle') }}</p>
      </div>
      <a-button :loading="loading" @click="loadData"><ReloadOutlined /> {{ t('common.refresh') }}</a-button>
    </div>

    <div class="stats-grid">
      <div class="stat-card"><span>{{ t('summary.stats.total') }}</span><strong>{{ stats.total_count }}</strong></div>
      <div class="stat-card stat-card--success"><span>{{ t('summary.stats.completed') }}</span><strong>{{ stats.summarized_count }}</strong></div>
      <div class="stat-card stat-card--warning"><span>{{ t('summary.stats.pending') }}</span><strong>{{ stats.pending_count }}</strong></div>
      <div class="stat-card stat-card--primary"><span>{{ t('summary.stats.running') }}</span><strong>{{ stats.running_count }}</strong></div>
    </div>

    <div class="filter-card">
      <a-input v-model:value="keyword" allow-clear :placeholder="t('summary.searchTitle')" @pressEnter="applyFilters">
        <template #prefix><SearchOutlined /></template>
      </a-input>
      <a-input v-model:value="applicant" allow-clear :placeholder="t('summary.searchApplicant')" @pressEnter="applyFilters" />
      <a-select v-model:value="processType" allow-clear :placeholder="t('summary.filterProcess')" :options="processTypeOptions" />
      <a-select v-model:value="summaryStatus" allow-clear :placeholder="t('summary.filterStatus')" :options="[
        { value: 'pending', label: t('summary.status.pending') },
        { value: 'summarized', label: t('summary.status.completed') },
        { value: 'running', label: t('summary.status.running') },
        { value: 'failed', label: t('summary.status.failed') },
      ]" />
      <a-button type="primary" @click="applyFilters">{{ t('common.search') }}</a-button>
      <a-button @click="resetFilters">{{ t('common.reset') }}</a-button>
    </div>

    <div class="process-card">
      <a-spin :spinning="loading">
        <a-empty v-if="!loading && items.length === 0" :description="t('summary.noData')" />
        <div v-else class="process-list">
          <div v-for="item in items" :key="item.process_id" class="process-row">
            <div class="process-main">
              <div class="process-title-row">
                <span class="process-title">{{ item.title }}</span>
                <a-tag :color="item.source === 'todo' ? 'blue' : 'default'">
                  {{ item.source === 'todo' ? t('summary.source.todo') : t('summary.source.archived') }}
                </a-tag>
                <a-badge :status="statusColor(item)" :text="statusLabel(item)" />
              </div>
              <div class="process-meta">
                <span>{{ item.process_id }}</span>
                <span>{{ item.process_type_label || item.process_type }}</span>
                <span>{{ item.applicant || '-' }} · {{ item.department || '-' }}</span>
                <span>{{ formatTime(item.submit_time) }}</span>
              </div>
            </div>
            <div class="process-actions">
              <a-button v-if="item.has_summary || item.running_job_id" @click="openResult(item)"><EyeOutlined /> {{ t('summary.view') }}</a-button>
              <a-button type="primary" :loading="executing[item.process_id]" @click="runSummary(item)">
                <ThunderboltOutlined /> {{ item.has_summary ? t('summary.regenerate') : t('summary.generate') }}
              </a-button>
            </div>
          </div>
        </div>
      </a-spin>
      <div class="pagination-wrapper">
        <a-pagination
          v-model:current="page"
          v-model:page-size="pageSize"
          :total="total"
          :page-size-options="['10', '20', '50']"
          show-size-changer
          @change="handlePageChange"
          @showSizeChange="handlePageSizeChange"
        />
      </div>
    </div>

    <a-drawer v-model:open="detailOpen" :title="selected?.title || t('summary.detailTitle')" width="720">
      <div v-if="currentResult && ['pending', 'assembling', 'reasoning', 'extracting'].includes(currentResult.status || '')" class="result-state">
        <a-spin />
        <div><strong>{{ t('summary.status.running') }}</strong><p>{{ t(`summary.progress.${currentResult.status}`) }}</p></div>
      </div>
      <a-alert v-else-if="currentResult?.status === 'failed'" type="error" show-icon :message="t('summary.status.failed')" :description="currentResult.error_message" />
      <a-empty v-else-if="visibleBlocks.length === 0" :description="t('summary.emptyResult')" />
      <div v-else class="result-blocks">
        <section v-for="block in visibleBlocks" :key="block.block_id" class="result-block">
          <div class="result-block-title">
            <CheckCircleOutlined /> {{ block.title }}
            <span v-if="block.duration_ms" class="duration"><ClockCircleOutlined /> {{ Math.round(block.duration_ms / 100) / 10 }}s</span>
          </div>
          <p class="result-content">{{ block.content }}</p>
          <ul v-if="block.points?.length" class="result-points"><li v-for="point in block.points" :key="point">{{ point }}</li></ul>
          <a-collapse v-if="block.deep_thinking" ghost>
            <a-collapse-panel key="thinking" :header="t('summary.deepThinking')">
              <p class="thinking-content">{{ block.deep_thinking }}</p>
            </a-collapse-panel>
          </a-collapse>
        </section>
      </div>
      <template #extra>
        <a-button v-if="selected" :loading="executing[selected.process_id]" @click="runSummary(selected, true)">
          <ExclamationCircleOutlined /> {{ t('summary.useLatestConfig') }}
        </a-button>
      </template>
    </a-drawer>
  </div>
</template>

<style scoped>
.summary-page { display: flex; flex-direction: column; gap: 20px; }
.page-header { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; }
.page-title { display: flex; align-items: center; gap: 10px; margin: 0; font-size: 24px; font-weight: 700; color: var(--color-text-primary); }
.page-subtitle { margin: 4px 0 0; color: var(--color-text-tertiary); }
.stats-grid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 14px; }
.stat-card { display: flex; flex-direction: column; gap: 6px; padding: 18px; background: var(--color-bg-card); border: 1px solid var(--color-border-light); border-radius: var(--radius-lg); color: var(--color-text-secondary); }
.stat-card strong { font-size: 28px; color: var(--color-text-primary); }
.stat-card--success { border-top: 3px solid var(--color-success); }
.stat-card--warning { border-top: 3px solid var(--color-warning); }
.stat-card--primary { border-top: 3px solid var(--color-primary); }
.filter-card { display: grid; grid-template-columns: 1.3fr 1fr 1fr 1fr auto auto; gap: 10px; padding: 16px; background: var(--color-bg-card); border: 1px solid var(--color-border-light); border-radius: var(--radius-lg); }
.process-card { background: var(--color-bg-card); border: 1px solid var(--color-border-light); border-radius: var(--radius-lg); overflow: hidden; }
.process-list { display: flex; flex-direction: column; }
.process-row { display: flex; align-items: center; justify-content: space-between; gap: 20px; padding: 18px 20px; border-bottom: 1px solid var(--color-border-light); }
.process-row:last-child { border-bottom: 0; }
.process-main { min-width: 0; }
.process-title-row { display: flex; align-items: center; flex-wrap: wrap; gap: 9px; }
.process-title { font-size: 15px; font-weight: 600; color: var(--color-text-primary); }
.process-meta { display: flex; flex-wrap: wrap; gap: 8px 18px; margin-top: 7px; color: var(--color-text-tertiary); font-size: 12px; }
.process-actions { display: flex; gap: 8px; flex-shrink: 0; }
.pagination-wrapper { display: flex; justify-content: flex-end; padding: 16px 20px; border-top: 1px solid var(--color-border-light); }
.result-state { display: flex; align-items: center; gap: 16px; padding: 24px; background: var(--color-bg-page); border-radius: var(--radius-lg); }
.result-state p { margin: 4px 0 0; color: var(--color-text-tertiary); }
.result-blocks { display: flex; flex-direction: column; gap: 16px; }
.result-block { padding: 18px; border: 1px solid var(--color-border-light); border-radius: var(--radius-lg); background: var(--color-bg-card); }
.result-block-title { display: flex; align-items: center; gap: 8px; font-size: 16px; font-weight: 600; color: var(--color-text-primary); }
.duration { margin-left: auto; font-size: 12px; font-weight: 400; color: var(--color-text-tertiary); }
.result-content, .thinking-content { white-space: pre-wrap; line-height: 1.75; color: var(--color-text-secondary); }
.result-points { margin: 12px 0 0; padding-left: 20px; color: var(--color-text-secondary); line-height: 1.7; }
@media (max-width: 1000px) {
  .stats-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .filter-card { grid-template-columns: 1fr 1fr; }
}
@media (max-width: 720px) {
  .stats-grid, .filter-card { grid-template-columns: 1fr; }
  .process-row { align-items: flex-start; flex-direction: column; }
  .process-actions { width: 100%; justify-content: flex-end; }
}
</style>
