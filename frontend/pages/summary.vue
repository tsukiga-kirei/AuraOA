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
import type { Dayjs } from 'dayjs'
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
const processType = ref<string>()
const summaryStatus = ref<string>()
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

// 与审核、归档一致：首次近 90 天，同一天刷新保留日期选择。
const dateStorageKey = 'auraoa:summary:list-date-range'
const defaultDateRange = (): [Dayjs, Dayjs] => [appDayjs().subtract(90, 'day').startOf('day'), appDayjs().endOf('day')]
const readDateRange = (): [Dayjs, Dayjs] => {
  if (import.meta.client) {
    try {
      const saved = JSON.parse(sessionStorage.getItem(dateStorageKey) || 'null')
      if (saved && appDayjs(saved.savedAt).isSame(appDayjs(), 'day')) {
        const start = appDayjs(saved.start), end = appDayjs(saved.end)
        if (start.isValid() && end.isValid() && !start.isAfter(end) && end.diff(start, 'day') <= 365 * 3) return [start, end]
      }
    } catch {}
  }
  return defaultDateRange()
}
const dateRange = ref<[Dayjs, Dayjs]>(readDateRange())
const statCards = computed(() => [
  { key: undefined, label: t('summary.stats.total'), count: stats.value.total_count, icon: FileTextOutlined, tone: 'primary' },
  { key: 'summarized', label: t('summary.stats.completed'), count: stats.value.summarized_count, icon: CheckCircleOutlined, tone: 'success' },
  { key: 'pending', label: t('summary.stats.pending'), count: stats.value.pending_count, icon: ClockCircleOutlined, tone: 'warning' },
])
const selectStatus = (status?: string) => {
  summaryStatus.value = summaryStatus.value === status ? undefined : status
  applyFilters()
}

const query = computed(() => ({
  start_date: dateRange.value[0].format('YYYY-MM-DD'),
  end_date: dateRange.value[1].format('YYYY-MM-DD'),
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
  try { sessionStorage.setItem(dateStorageKey, JSON.stringify({ start: query.value.start_date, end: query.value.end_date, savedAt: new Date().toISOString() })) } catch {}
  page.value = 1
  void loadData()
}

const resetFilters = () => {
  keyword.value = ''
  applicant.value = ''
  processType.value = undefined
  summaryStatus.value = undefined
  dateRange.value = defaultDateRange()
  try { sessionStorage.removeItem(dateStorageKey) } catch {}
  page.value = 1
  void loadData()
}

const handlePageChange = (nextPage: number, nextSize: number) => {
  page.value = nextSize !== pageSize.value ? 1 : nextPage
  pageSize.value = nextSize
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
    }, useLatestConfig, (progress) => { if (selected.value?.process_id === item.process_id) currentResult.value = progress })
    if (selected.value?.process_id === item.process_id) currentResult.value = result
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
      <button v-for="card in statCards" :key="card.key || 'all'" type="button" class="stat-card" :class="[`stat-card--${card.tone}`, { 'stat-card--selected': summaryStatus === card.key }]" :aria-pressed="summaryStatus === card.key" @click="selectStatus(card.key)">
        <div class="stat-card-icon"><component :is="card.icon" /></div>
        <div class="stat-card-info"><span class="stat-card-value">{{ card.count }}</span><span class="stat-card-label">{{ card.label }}</span></div>
      </button>
    </div>

    <div class="filter-card">
      <label class="filter-field filter-field--date"><span>{{ t('summary.dateRange') }}</span><a-range-picker v-model:value="dateRange" :allow-clear="false" @change="applyFilters" /></label>
      <a-input v-model:value="keyword" allow-clear :placeholder="t('summary.searchTitle')" @pressEnter="applyFilters">
        <template #prefix><SearchOutlined /></template>
      </a-input>
      <a-input v-model:value="applicant" allow-clear :placeholder="t('summary.searchApplicant')" @pressEnter="applyFilters" />
      <label class="filter-field"><span>{{ t('summary.filterProcess') }}</span><a-select v-model:value="processType" allow-clear :placeholder="t('summary.filterProcess')" :options="processTypeOptions" /></label>
      <label class="filter-field"><span>{{ t('summary.filterStatus') }}</span><a-select v-model:value="summaryStatus" allow-clear :placeholder="t('summary.filterStatus')" :options="[
        { value: 'pending', label: t('summary.status.pending') },
        { value: 'summarized', label: t('summary.status.completed') },
        { value: 'running', label: t('summary.status.running') },
        { value: 'failed', label: t('summary.status.failed') },
      ]" /></label>
      <a-button type="primary" @click="applyFilters">{{ t('common.search') }}</a-button>
      <a-button @click="resetFilters">{{ t('common.reset') }}</a-button>
    </div>

    <div class="process-card">
      <a-spin :spinning="loading" :tip="t('common.loading')">
        <a-empty v-if="!loading && items.length === 0" :description="t('summary.noData')" />
        <div v-else class="process-list" :class="{ 'process-list--loading': loading }">
          <div v-for="item in items" :key="item.process_id" class="process-row">
            <div class="process-main">
              <div class="process-title-row">
                <span class="process-title">{{ item.title }}</span>
                <a-tag :color="item.source === 'todo' ? 'blue' : item.source === 'embed' ? 'purple' : 'default'">
                  {{ item.source === 'todo' ? t('summary.source.todo') : item.source === 'embed' ? t('summary.source.embed') : t('summary.source.archived') }}
                </a-tag>
                <a-tag v-if="item.summary_result?.result_source" color="purple">{{ t(`resultSource.${item.summary_result.result_source}`) }}</a-tag>
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
          :current="page"
          :page-size="pageSize"
          :total="total"
          :page-size-options="['10', '20', '50']"
          show-size-changer
          @change="handlePageChange"
        />
      </div>
    </div>

    <a-drawer v-model:open="detailOpen" :title="selected?.title || t('summary.detailTitle')" width="720">
      <a-alert v-if="currentResult?.result_source" type="info" show-icon :message="t(`resultSource.${currentResult.result_source}`)" :description="t('summary.personalHint')" style="margin-bottom: 16px" />
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
.stats-grid { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 16px; }
.stat-card { background: var(--color-bg-card); border-radius: var(--radius-lg); padding: 20px; display: flex; align-items: center; gap: 16px; border: 2px solid var(--color-border-light); cursor: pointer; text-align: left; font: inherit; transition: all var(--transition-base); }
.stat-card:hover { transform: translateY(-2px); box-shadow: var(--shadow-md); }
.stat-card--selected { border-color: var(--color-primary); box-shadow: 0 0 0 1px var(--color-primary); }
.stat-card-icon { width: 48px; height: 48px; border-radius: var(--radius-lg); display: flex; align-items: center; justify-content: center; font-size: 22px; flex-shrink: 0; }
.stat-card--primary .stat-card-icon { background: var(--color-primary-bg); color: var(--color-primary); }
.stat-card--success .stat-card-icon { background: var(--color-success-bg); color: var(--color-success); }
.stat-card--warning .stat-card-icon { background: var(--color-warning-bg); color: var(--color-warning); }
.stat-card-info { display: flex; flex-direction: column; }
.stat-card-value { font-size: 28px; font-weight: 700; color: var(--color-text-primary); line-height: 1.2; }
.stat-card-label { font-size: 13px; color: var(--color-text-tertiary); margin-top: 2px; }
.filter-field { display: flex; flex-direction: column; gap: 6px; min-width: 0; font-size: 12px; color: var(--color-text-secondary); }
.filter-field--date { grid-column: 1 / -1; }
.filter-field--date :deep(.ant-picker) { max-width: 360px; }
.process-list--loading { min-height: 180px; }
.process-card :deep(.ant-spin-nested-loading), .process-card :deep(.ant-spin-container) { min-height: 180px; }
.filter-card { display: grid; grid-template-columns: 1.3fr 1fr 1fr 1fr auto auto; align-items: end; gap: 10px; padding: 16px; background: var(--color-bg-card); border: 1px solid var(--color-border-light); border-radius: var(--radius-lg); }
.process-card { background: var(--color-bg-card); border: 1px solid var(--color-border-light); border-radius: var(--radius-lg); overflow: hidden; }
.process-list { display: flex; flex-direction: column; }
.process-row { display: flex; align-items: center; justify-content: space-between; gap: 20px; padding: 18px 20px; border-bottom: 1px solid var(--color-border-light); }
.process-row:last-child { border-bottom: 0; }
.process-main { min-width: 0; }
.process-title-row { display: flex; align-items: center; flex-wrap: wrap; gap: 9px; }
.process-title { font-size: 15px; font-weight: 600; color: var(--color-text-primary); }
.process-meta { display: flex; flex-wrap: wrap; gap: 8px 18px; margin-top: 7px; color: var(--color-text-tertiary); font-size: 12px; }
.process-actions { display: flex; gap: 8px; flex-shrink: 0; align-items: center; }
.process-actions :deep(.ant-btn) { white-space: nowrap; min-height: 32px; }
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
