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
  EyeOutlined,
  StopOutlined,
  DownOutlined,
  UpOutlined,
  InfoCircleOutlined,
  WarningOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import dayjs, { type Dayjs } from 'dayjs'
import { useI18n } from '~/composables/useI18n'
import type {
  ArchiveProcessItem,
  ArchiveProgressStep,
  ArchiveReviewResult,
  ArchiveReviewStats,
  ArchiveReviewHistoryItem,
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

/** 与列表接口返回的 archive_status / archive_result.status 一致；刷新后仍显示「复核中」 */
function isArchiveJobRunning(proc: ArchiveProcessItem): boolean {
  const st = proc.archive_result?.status ?? proc.archive_status
  return !!(st && asyncArchiveStatuses.includes(st))
}

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
const currentResult = ref<ArchiveReviewResult | null>(null)
const batchAborted = ref(false)
const currentInflightProcessId = ref<string | null>(null)

const auditInProgress = computed(
  () =>
    batchAuditing.value
    || Object.values(processAuditLoading.value).some(Boolean)
    || processList.value.some(p => p.archive_status && asyncArchiveStatuses.includes(p.archive_status)),
)

const blockingProcessId = computed((): string | null => {
  if (batchAuditing.value) return currentInflightProcessId.value
  const row = processList.value.find(p => p.archive_status && asyncArchiveStatuses.includes(p.archive_status))
  return row?.process_id ?? null
})

const ARCHIVE_BATCH_KEY = 'oa-smart-audit:archive:batch-queue'

type ArchiveBatchMeta = { process_id: string; process_type: string; title: string }
type ArchiveBatchPersisted = {
  ids: string[]
  queueMeta: ArchiveBatchMeta[]
  nextIndex: number
  tab: ArchiveAuditTab
  /** 当前条异步任务 archive_logs.id，列表未加载或仍为旧快照时用于 POST /api/archive/cancel/:id */
  inflightJobId?: string
}

function saveArchiveBatchState(
  ids: string[],
  queueMeta: ArchiveBatchMeta[],
  nextIndex: number,
  inflightJobId?: string | null,
) {
  try {
    const payload: ArchiveBatchPersisted = { ids, queueMeta, nextIndex, tab: filterAuditStatus.value }
    if (inflightJobId) payload.inflightJobId = inflightJobId
    sessionStorage.setItem(ARCHIVE_BATCH_KEY, JSON.stringify(payload))
  } catch {}
}

function clearArchiveBatchStorage() {
  try {
    sessionStorage.removeItem(ARCHIVE_BATCH_KEY)
  } catch {}
}

function readArchiveBatchState(): ArchiveBatchPersisted | null {
  try {
    const r = sessionStorage.getItem(ARCHIVE_BATCH_KEY)
    if (!r) return null
    return JSON.parse(r) as ArchiveBatchPersisted
  } catch {
    return null
  }
}

/** 中止时：优先列表行上的任务 id，否则用批量持久化里的 inflightJobId（刷新后列表可能尚无 id） */
function resolveArchiveJobIdForCancel(processId: string): string | undefined {
  const item = processList.value.find(p => p.process_id === processId)
  if (item?.archive_result?.id) return item.archive_result.id
  const st = readArchiveBatchState()
  if (st?.inflightJobId && st.ids[st.nextIndex] === processId) return st.inflightJobId
  return undefined
}

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
type ArchiveAuditTab = 'unaudited' | 'compliant' | 'partially_compliant' | 'non_compliant'
const filterAuditStatus = ref<ArchiveAuditTab>('unaudited')

/**
 * 列表日期范围：记住当日选择，避免刷新后回到近 90 天导致批量/长周期任务从列表消失；
 * 跨自然日再打开则丢弃（默认近 90 天），便于次日看到相对「最新」的待办。
 */
const ARCHIVE_DATE_RANGE_KEY = 'oa-smart-audit:archive:list-date-range'

function defaultArchiveDateRange(): [Dayjs, Dayjs] {
  return [dayjs().subtract(90, 'day').startOf('day'), dayjs().endOf('day')]
}

function readArchiveDateRange(): [Dayjs, Dayjs] | null {
  if (typeof window === 'undefined') return null
  try {
    const r = sessionStorage.getItem(ARCHIVE_DATE_RANGE_KEY)
    if (!r) return null
    const o = JSON.parse(r) as { start?: string; end?: string; savedAt?: string }
    if (!o.start || !o.end) return null
    if (o.savedAt) {
      const saved = dayjs(o.savedAt)
      if (!saved.isValid() || !saved.isSame(dayjs(), 'day')) {
        sessionStorage.removeItem(ARCHIVE_DATE_RANGE_KEY)
        return null
      }
    } else {
      sessionStorage.removeItem(ARCHIVE_DATE_RANGE_KEY)
      return null
    }
    const a = dayjs(o.start)
    const b = dayjs(o.end)
    if (!a.isValid() || !b.isValid()) return null
    if (a.isAfter(b)) return null
    const maxSpan = 365 * 3
    if (b.diff(a, 'day') > maxSpan) return null
    return [a.startOf('day'), b.endOf('day')]
  } catch {
    return null
  }
}

function saveArchiveDateRange(range: [Dayjs, Dayjs]) {
  if (typeof window === 'undefined') return
  try {
    const [a, b] = range
    if (!a?.isValid?.() || !b?.isValid?.()) return
    sessionStorage.setItem(
      ARCHIVE_DATE_RANGE_KEY,
      JSON.stringify({
        start: a.format('YYYY-MM-DD'),
        end: b.format('YYYY-MM-DD'),
        savedAt: new Date().toISOString(),
      }),
    )
  } catch {}
}

function clearArchiveDateRangeStorage() {
  try {
    if (typeof window !== 'undefined') sessionStorage.removeItem(ARCHIVE_DATE_RANGE_KEY)
  } catch {}
}

const archiveDateRange = ref<[Dayjs, Dayjs]>(readArchiveDateRange() || defaultArchiveDateRange())

const archiveDateQuery = () => {
  const r = archiveDateRange.value
  if (!r?.[0] || !r?.[1]) return {}
  return {
    start_date: r[0].format('YYYY-MM-DD'),
    end_date: r[1].format('YYYY-MM-DD'),
  }
}

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

const hasActiveFilters = computed(() =>
  !!searchText.value || !!searchApplicant.value || filterProcessType.value.length > 0 || !!filterDepartment.value
)

const computedListTitle = computed(() => {
  const m: Record<ArchiveAuditTab, string> = {
    unaudited: t('archive.statUnaudited'),
    compliant: t('archive.statCompliant'),
    partially_compliant: t('archive.statPartial'),
    non_compliant: t('archive.statNonCompliant'),
  }
  return m[filterAuditStatus.value]
})

const clearFilters = () => {
  if (auditInProgress.value) {
    message.warning(t('archive.auditInProgressNoSwitch'))
    return
  }
  searchText.value = ''
  searchApplicant.value = ''
  filterProcessType.value = []
  filterDepartment.value = undefined
  archiveDateRange.value = defaultArchiveDateRange()
  clearArchiveDateRangeStorage()
  listPage.value = 1
  clearDetailOnFilterChange()
  void Promise.all([loadStats(), loadProcesses()])
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

/** 筛选条件变化时清空左右两侧数据，避免在旧列表上叠加载；分页仅调 loadProcesses 不会走此逻辑 */
const clearDetailOnFilterChange = () => {
  if (batchAuditing.value) return
  disconnectStream()
  processList.value = []
  listTotal.value = 0
  selectedProcess.value = null
  currentResult.value = null
  selectedProcessIds.value = []
}

// 搜索条件变化时重新拉取第一页
const triggerSearch = () => {
  if (searchDebounceTimer) clearTimeout(searchDebounceTimer)
  listLoading.value = true
  searchDebounceTimer = setTimeout(() => {
    listPage.value = 1
    loadProcesses()
  }, 400)
}

//=====选择=====
const toggleSelectProcess = (id: string) => {
  if (auditInProgress.value) return
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
  if (auditInProgress.value) return
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
  if (!archiveLogId) return
  // 已由列表恢复轮询时：仅补挂 SSE（用户随后点开该流程时）
  if (pollProcessId.value === proc.process_id) {
    if (selectedProcess.value?.process_id === proc.process_id) {
      startSSE(archiveLogId, proc.process_id)
    }
    return
  }

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

/** 列表刷新/分页后：恢复对进行中归档任务的轮询与 SSE（不依赖再次点击） */
function resumeArchiveAsyncTrackingFromList() {
  const running = processList.value.filter(
    p => isArchiveJobRunning(p) && p.archive_result?.id,
  )
  if (running.length === 0) return
  const preferred = selectedProcess.value
    ? running.find(p => p.process_id === selectedProcess.value!.process_id)
    : null
  const proc = preferred ?? running[0]
  void trackRunningJob(proc)
}

const selectProcess = (proc: ArchiveProcessItem) => {
  if (auditInProgress.value) {
    const allow = blockingProcessId.value
    if (allow && proc.process_id !== allow) {
      message.warning(t('archive.auditInProgressNoSwitch'))
      return
    }
  }
  selectedProcess.value = proc
  currentResult.value = normalizeArchiveResult(proc.archive_result)
  if (isResultAsyncRunning(currentResult.value)) {
    trackRunningJob(proc)
  } else {
    disconnectStream()
  }
}

const loading = computed(() => isResultAsyncRunning(currentResult.value))

const runArchiveReview = async (proc: ArchiveProcessItem, onStarted?: (id: string) => void) => {
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
      if (status.id && !started) {
        started = true
        if (selectedProcess.value?.process_id === proc.process_id) {
          startSSE(status.id, proc.process_id)
        }
        if (onStarted) onStarted(status.id)
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

function archiveItemFromMeta(meta: ArchiveBatchMeta): ArchiveProcessItem {
  return {
    process_id: meta.process_id,
    title: meta.title,
    process_type: meta.process_type,
    process_type_label: meta.process_type,
    applicant: '',
    department: '',
    current_node: '',
    submit_time: '',
    archive_time: '',
    has_review: false,
    in_archive: true,
  }
}

//=====批量审核（sessionStorage 支持刷新后续跑）=====
async function runArchiveBatchLoop(ids: string[], queueMeta: ArchiveBatchMeta[], startIndex: number) {
  batchAuditing.value = true
  batchAborted.value = false
  const metaById = new Map(queueMeta.map(m => [m.process_id, m]))
  const prev = readArchiveBatchState()
  const carryInflight =
    prev &&
    prev.ids.length === ids.length &&
    prev.nextIndex === startIndex &&
    prev.ids[startIndex] &&
    prev.inflightJobId &&
    prev.queueMeta.length === queueMeta.length &&
    prev.ids.every((id, idx) => id === queueMeta[idx]?.process_id)
      ? prev.inflightJobId
      : undefined
  saveArchiveBatchState(ids, queueMeta, startIndex, carryInflight)

  for (let i = startIndex; i < ids.length; i++) {
    if (batchAborted.value) break
    const id = ids[i]
    currentInflightProcessId.value = id
    const proc = processList.value.find(p => p.process_id === id)
    const meta = proc
      ? { process_id: proc.process_id, process_type: proc.process_type, title: proc.title }
      : metaById.get(id)
    if (!meta) {
      batchAuditDone.value = i + 1
      saveArchiveBatchState(ids, queueMeta, i + 1)
      continue
    }
    const procToRun = proc ?? archiveItemFromMeta(meta)

    processAuditLoading.value = {
      ...processAuditLoading.value,
      [id]: true,
    }

    try {
      await runArchiveReview(procToRun, (jobId) => {
        saveArchiveBatchState(ids, queueMeta, i, jobId)
      })
    } catch {
    }

    processAuditLoading.value = {
      ...processAuditLoading.value,
      [id]: false,
    }
    batchAuditDone.value = i + 1
    saveArchiveBatchState(ids, queueMeta, i + 1)
  }

  currentInflightProcessId.value = null
  batchAuditing.value = false
  selectedProcessIds.value = []
  clearArchiveBatchStorage()
  await Promise.all([loadStats(), loadProcesses()])
  if (batchAborted.value) {
    message.info(t('archive.batchAborted', '批量审核已中止'))
  } else {
    message.success(t('archive.batchDone', `${batchAuditDone.value}`))
  }
}

const handleBatchAudit = async () => {
  if (selectedProcessIds.value.length === 0 || selectedProcessIds.value.length > 10) {
    if (selectedProcessIds.value.length > 10) {
      message.warning(t('archive.batchLimitHint'))
    }
    return
  }
  const ids = [...selectedProcessIds.value]
  const queueMeta: ArchiveBatchMeta[] = ids.map(id => {
    const it = processList.value.find(p => p.process_id === id)
    if (!it) return null
    return { process_id: it.process_id, process_type: it.process_type, title: it.title }
  }).filter(Boolean) as ArchiveBatchMeta[]
  if (queueMeta.length !== ids.length) {
    message.error(t('archive.loadFailed'))
    return
  }
  batchAuditTotal.value = ids.length
  batchAuditDone.value = 0
  await runArchiveBatchLoop(ids, queueMeta, 0)
}

function tryResumeArchiveBatch() {
  const state = readArchiveBatchState()
  if (!state || state.ids.length === 0) return
  if (state.nextIndex >= state.ids.length) {
    clearArchiveBatchStorage()
    return
  }
  const { ids, queueMeta, nextIndex } = state
  batchAuditTotal.value = ids.length
  batchAuditDone.value = nextIndex
  selectedProcessIds.value = [...ids]
  void runArchiveBatchLoop(ids, queueMeta, nextIndex)
}

const handleAbortBatch = async () => {
  batchAborted.value = true
  if (currentInflightProcessId.value) {
    const pid = currentInflightProcessId.value
    await handleCancelAudit(pid).catch(() => {})
    currentInflightProcessId.value = null
  }
  // 中止时，将未运行的 pending 任务置为失败状态，对齐 dashboard 逻辑
  for (const id of selectedProcessIds.value) {
    const item = processList.value.find(p => p.process_id === id)
    if (item && item.archive_status === 'pending') {
      item.archive_status = 'failed'
      item.archive_result = { status: 'failed', error_message: '批量审核已中止' } as any
    }
  }
  clearArchiveBatchStorage()
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

const handleCancelAudit = async (processId: string) => {
  try {
    const jobId = resolveArchiveJobIdForCancel(processId)
    if (!jobId) {
       message.warning(t('archive.noAuditIdFound', '无法中止，任务 ID 缺失'))
       return
    }
    await cancelArchiveJob(jobId)
    message.success(t('archive.cancelSuccess'))
    await Promise.all([loadStats(), loadProcesses()])
    if (selectedProcess.value?.process_id === processId) {
      const next = processList.value.find(proc => proc.process_id === processId)
      if (next) selectProcess(next)
    }
  } catch (error: any) {
    message.error(error?.message || t('archive.cancelFailed'))
  }
}

const handleUnifiedAbortArchive = async () => {
  if (batchAuditing.value) {
    await handleAbortBatch()
    return
  }
  const pid = blockingProcessId.value
  if (pid) await handleCancelAudit(pid)
  else if (selectedProcess.value) {
    await handleCancelAudit(selectedProcess.value.process_id)
  }
}

const loadStats = async () => {
  try {
    stats.value = await getStats(archiveDateQuery())
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
      audit_status: filterAuditStatus.value,
      page: listPage.value,
      page_size: listPageSize.value,
      ...archiveDateQuery(),
    })
    processList.value = Array.isArray(response) ? response : (response?.items ?? [])
    listTotal.value = (response as any)?.total ?? processList.value.length
    if (!batchAuditing.value) {
      selectedProcessIds.value = selectedProcessIds.value.filter(id => selectableIdsComputed.value.includes(id))
    }

    if (selectedProcess.value) {
      const nextSelected = processList.value.find(proc => proc.process_id === selectedProcess.value?.process_id) || null
      selectedProcess.value = nextSelected
      currentResult.value = normalizeArchiveResult(nextSelected?.archive_result)
    }

    resumeArchiveAsyncTrackingFromList()
  } catch {
    processList.value = []
    message.error(t('archive.loadFailed'))
  } finally {
    listLoading.value = false
  }
}

const onArchiveDateRangeChange = () => {
  if (auditInProgress.value) {
    message.warning(t('archive.auditInProgressNoSwitch'))
    return
  }
  saveArchiveDateRange(archiveDateRange.value)
  listPage.value = 1
  clearDetailOnFilterChange()
  void Promise.all([loadStats(), loadProcesses()])
}

watch([searchText, searchApplicant, filterProcessNames, filterDepartment, filterAuditStatus], () => {
  if (auditInProgress.value) return
  clearDetailOnFilterChange()
  triggerSearch()
})

const onStatCardFilterClick = (value: ArchiveAuditTab) => {
  if (auditInProgress.value) {
    message.warning(t('archive.auditInProgressNoSwitch'))
    return
  }
  filterAuditStatus.value = value
}

const filteredProgressSteps = computed(() => {
  if (!currentResult.value) return []
  const steps = currentResult.value.progress_steps?.length
    ? currentResult.value.progress_steps
    : defaultProgressSteps(currentResult.value.status)
  return steps.filter(step => step.key !== 'pending')
})

const showHistoryChain = ref(false)
/** 归档复盘链：来自 /api/archive/history（archive_logs），与 dashboard 的 OA 审核链（audit_logs）数据源不同 */
const archiveHistoryData = ref<ArchiveReviewHistoryItem[]>([])
const auditChainLoading = ref(false)
const expandedChainNodes = ref<Set<string>>(new Set())

const toggleChainNode = (id: string) => {
  if (expandedChainNodes.value.has(id)) expandedChainNodes.value.delete(id)
  else expandedChainNodes.value.add(id)
}

const openAuditChain = async (processId: string) => {
  expandedChainNodes.value = new Set()
  showHistoryChain.value = true
  auditChainLoading.value = true
  try {
    archiveHistoryData.value = await getArchiveHistory(processId)
  } catch {
    archiveHistoryData.value = []
  } finally {
    auditChainLoading.value = false
  }
}

const formatChainDate = (dateStr: string) => {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return isNaN(d.getTime()) ? dateStr : d.toLocaleString('zh-CN', { hour12: false }).replace(/\//g, '-')
}

const getDurationSec = (ms: number | undefined) => {
  if (ms === undefined) return 0
  return (ms / 1000).toFixed(1)
}

const getScoreColorConfig = (score: number | undefined) => {
  if (score === undefined || score === null) return { color: 'var(--color-info)', bg: 'var(--color-info-bg)' }
  if (score < 60) return { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)' }
  if (score > 80) return { color: 'var(--color-success)', bg: 'var(--color-success-bg)' }
  return { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)' }
}

//=====配置助手=====
const complianceConfig = computed((): Record<string, { color: string; bg: string; label: string }> => ({
  compliant: { color: 'var(--color-success)', bg: 'var(--color-success-bg)', label: t('archive.compliant') },
  non_compliant: { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)', label: t('archive.nonCompliant') },
  partially_compliant: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', label: t('archive.partiallyCompliant') },
}))

/** 与后端 archiveItemHasComplianceOutcome 一致：仅 completed 且有三档合规结论才算「已完成复盘」 */
function archiveHasFinishedOutcome(
  proc: ArchiveProcessItem | null | undefined,
  res?: ArchiveReviewResult | null,
): boolean {
  if (!proc) return false
  const r = res ?? proc.archive_result ?? null
  const status = r?.status ?? proc.archive_status
  if (status !== 'completed') return false
  const c = r?.overall_compliance
  return c === 'compliant' || c === 'partially_compliant' || c === 'non_compliant'
}

const archiveDetailShowsComplianceReport = computed(() =>
  archiveHasFinishedOutcome(selectedProcess.value, currentResult.value),
)

const auditedCount = computed(() => processList.value.filter(p => archiveHasFinishedOutcome(p, p.archive_result)).length)

onMounted(async () => {
  // 1. 优先恢复持久化的批量状态（含页签切换状态），确保初始加载使用正确的 filterAuditStatus
  const batchState = readArchiveBatchState()
  if (batchState?.tab) {
    filterAuditStatus.value = batchState.tab
  }
  
  // 2. 加载基础数据（已带上正确的页签过滤条件）
  await Promise.all([loadProcessTypes(), loadStats(), loadProcesses()])
  
  // 3. 尝试恢复执行批量任务
  tryResumeArchiveBatch()
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
    <div class="stats-row" :class="{ 'stats-row--readonly': auditInProgress }">
      <div class="stat-card stat-card--primary" :class="{ 'stat-card--selected': filterAuditStatus === 'unaudited' }" @click="onStatCardFilterClick('unaudited')">
        <div class="stat-card-icon"><FileProtectOutlined /></div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ stats.unaudited_count }}</span>
          <span class="stat-card-label">{{ t('archive.statUnaudited') }}</span>
        </div>
      </div>
      <div class="stat-card stat-card--success" :class="{ 'stat-card--selected': filterAuditStatus === 'compliant' }" @click="onStatCardFilterClick('compliant')">
        <div class="stat-card-icon"><CheckCircleOutlined /></div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ stats.compliant_count }}</span>
          <span class="stat-card-label">{{ t('archive.statCompliant') }}</span>
        </div>
      </div>
      <div class="stat-card stat-card--warning" :class="{ 'stat-card--selected': filterAuditStatus === 'partially_compliant' }" @click="onStatCardFilterClick('partially_compliant')">
        <div class="stat-card-icon"><AlertOutlined /></div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ stats.partial_count }}</span>
          <span class="stat-card-label">{{ t('archive.statPartial') }}</span>
        </div>
      </div>
      <div class="stat-card stat-card--danger" :class="{ 'stat-card--selected': filterAuditStatus === 'non_compliant' }" @click="onStatCardFilterClick('non_compliant')">
        <div class="stat-card-icon"><CloseCircleOutlined /></div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ stats.non_compliant_count }}</span>
          <span class="stat-card-label">{{ t('archive.statNonCompliant') }}</span>
        </div>
      </div>
    </div>

    <!--主要布局（与 dashboard 一致）-->
    <div class="dashboard-grid">
      <!--左：进程列表-->
      <div class="todo-panel">
        <div class="panel-header">
          <div class="panel-header-row">
            <h3 class="panel-title">
              <FireOutlined style="color: var(--color-primary);" />
              {{ computedListTitle }}
              <a-badge :count="listTotal" :number-style="{ backgroundColor: 'var(--color-primary)' }" />
            </h3>
            <div class="panel-header-controls">
              <span class="archive-date-label">{{ t('archive.archiveDateRange') }}</span>
              <a-range-picker
                v-model:value="archiveDateRange"
                :allow-clear="false"
                class="archive-range-picker"
                @change="onArchiveDateRangeChange"
              />
              <a-button size="small" type="default" @click="showFilters = !showFilters" class="filter-toggle-btn" :class="{ 'filter-toggle-btn--active': hasActiveFilters }">
                <FilterOutlined />
                {{ t('archive.filter') }}
                <span v-if="hasActiveFilters" class="filter-active-dot" />
              </a-button>
            </div>
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
                :disabled="auditInProgress"
                :checked="selectedProcessIds.length > 0 && selectedProcessIds.length === Math.min(selectableIdsComputed.length, 10)"
                :indeterminate="selectedProcessIds.length > 0 && selectedProcessIds.length < Math.min(selectableIdsComputed.length, 10)"
                @change="toggleSelectAll"
              >
                {{ selectedProcessIds.length > 0 ? t('archive.selected', `${selectedProcessIds.length}`) : t('archive.selectAll') }}
              </a-checkbox>
              <span class="batch-limit-hint">{{ t('dashboard.batchLimitLabel') }}</span>
              <span v-if="batchAuditing" class="batch-progress-hint">
                {{ t('archive.auditedProgress', `${batchAuditDone}/${batchAuditTotal}`) }}
              </span>
              <span v-else-if="auditInProgress && !batchAuditing" class="batch-progress-hint">
                {{ t('archive.auditingItem') }}
              </span>
              <span v-else-if="auditedCount > 0" class="panel-header-hint">{{ t('archive.reviewed') }} {{ auditedCount }}/{{ listTotal }}</span>
            </div>
            <div class="batch-toolbar-right">
              <a-button
                v-if="auditInProgress"
                size="small"
                danger
                @click="handleUnifiedAbortArchive"
              >
                <StopOutlined /> {{ t('archive.batchAbort', '中止') }}
              </a-button>
              <a-button
                v-if="selectedProcessIds.length > 0"
                type="primary"
                size="small"
                :disabled="auditInProgress"
                @click="handleBatchAudit"
                class="batch-audit-btn"
              >
                <LoadingOutlined v-if="batchAuditing" />
                <ThunderboltOutlined v-else />
                {{ t('archive.batchAudit') }}
              </a-button>
            </div>
          </div>
        </div>

        <!--进程列表-->
        <a-spin :spinning="listLoading">
          <div class="todo-list">
            <div
              v-for="proc in processList"
              :key="proc.process_id"
              class="todo-item"
              :class="{
                'todo-item--selected': selectedProcess?.process_id === proc.process_id,
                'todo-item--audited-approve': archiveHasFinishedOutcome(proc, proc.archive_result) && proc.archive_result?.overall_compliance === 'compliant',
                'todo-item--audited-return': archiveHasFinishedOutcome(proc, proc.archive_result) && proc.archive_result?.overall_compliance === 'partially_compliant',
                'todo-item--archive-noncompliant': archiveHasFinishedOutcome(proc, proc.archive_result) && proc.archive_result?.overall_compliance === 'non_compliant',
              }"
              @click="selectProcess(proc)"
            >
              <div class="todo-item-checkbox" @click.stop="auditInProgress || isArchiveJobRunning(proc) ? null : toggleSelectProcess(proc.process_id)">
                <a-checkbox :checked="selectedProcessIds.includes(proc.process_id)" :disabled="auditInProgress || isArchiveJobRunning(proc)" />
              </div>
              <div class="todo-item-main">
                <div class="todo-item-title">
                  <LoadingOutlined
                    v-if="isArchiveJobRunning(proc)" class="todo-item-audited-icon" spin style="color: var(--color-primary);"
                  />
                  <CheckCircleOutlined
                    v-else-if="archiveHasFinishedOutcome(proc, proc.archive_result) && proc.archive_result?.overall_compliance === 'compliant'"
                    class="todo-item-audited-icon"
                    style="color: var(--color-success);"
                  />
                  <CheckCircleOutlined
                    v-else-if="archiveHasFinishedOutcome(proc, proc.archive_result) && proc.archive_result?.overall_compliance === 'partially_compliant'"
                    class="todo-item-audited-icon"
                    style="color: var(--color-warning);"
                  />
                  <CloseCircleOutlined
                    v-else-if="archiveHasFinishedOutcome(proc, proc.archive_result) && proc.archive_result?.overall_compliance === 'non_compliant'"
                    class="todo-item-audited-icon"
                    style="color: var(--color-danger);"
                  />
                  {{ proc.title }}
                </div>
                <div class="todo-item-meta">
                  <span>{{ proc.applicant }}</span>
                  <span class="todo-item-dot">·</span>
                  <span>{{ proc.department }}</span>
                  <span class="todo-item-dot">·</span>
                  <span>{{ proc.submit_time }}</span>
                </div>
                <div class="todo-item-audit-info">
                  <div class="todo-item-audit-left">
                    <span class="todo-item-node">{{ proc.current_node || '—' }}</span>
                    <span class="todo-item-process-type">{{ proc.process_type_label || proc.process_type }}</span>
                  </div>
                  <div class="todo-item-audit-right">
                    <span
                      v-if="isArchiveJobRunning(proc)"
                      class="todo-item-score-badge"
                      style="color: var(--color-primary); background: var(--color-primary-bg);"
                    >
                      {{ t('archive.auditingItem') }}
                    </span>
                    <span
                      v-else-if="archiveHasFinishedOutcome(proc, proc.archive_result) && proc.archive_result?.overall_compliance"
                      class="todo-item-score-badge"
                      :style="{
                        color: complianceConfig[proc.archive_result.overall_compliance]?.color,
                        background: complianceConfig[proc.archive_result.overall_compliance]?.bg,
                      }"
                    >
                      {{ complianceConfig[proc.archive_result.overall_compliance]?.label }}
                      {{ proc.archive_result.overall_score }}{{ t('archive.score') }}
                    </span>
                    <span
                      v-else
                      class="todo-item-score-badge"
                      style="color: var(--color-text-secondary); background: var(--color-bg-hover);"
                    >
                      {{ t('archive.pendingReview') }}
                    </span>
                    <a-tooltip :title="t('dashboard.auditChain')" :mouse-enter-delay="0.5">
                      <button class="oa-jump-btn" @click.stop="openAuditChain(proc.process_id)">
                        <HistoryOutlined />
                      </button>
                    </a-tooltip>
                    <a-tooltip :title="t('dashboard.jumpToOA')" :mouse-enter-delay="0.5">
                      <button class="oa-jump-btn" @click.stop="jumpToOA(proc.process_id)">
                        <ExportOutlined />
                      </button>
                    </a-tooltip>
                  </div>
                </div>
              </div>
            </div>

            <div v-if="processList.length === 0 && !listLoading" class="todo-empty">
              <a-empty :description="t('archive.noMatch')" />
            </div>
          </div>
        </a-spin>

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

      <!--右：复盘结果（与 dashboard 结果区结构一致）-->
      <div class="result-panel">
        <div class="panel-header">
          <h3 class="panel-title">
            <SafetyCertificateOutlined style="color: var(--color-primary);" />
            {{ t('archive.complianceTitle') }}
          </h3>
        </div>

        <div class="result-content">
          <!--列表加载中且无选中：与左侧一致显示加载态，避免仍显示上一条详情 -->
          <div v-if="listLoading && !selectedProcess" class="result-empty result-empty--loading">
            <a-spin size="large" />
            <p>{{ t('archive.loadingListHint') }}</p>
          </div>
          <div v-else-if="!selectedProcess" class="result-empty">
            <div class="result-empty-icon"><SafetyCertificateOutlined /></div>
            <p>{{ t('archive.selectProcessDesc') }}</p>
          </div>

          <template v-else>
            <!--审核进行中（与 dashboard result-async-panel 一致）-->
            <template v-if="loading && currentResult">
              <div class="result-async-panel">
                <a-spin size="large">
                  <div class="async-progress-steps">
                    <div
                      v-for="s in filteredProgressSteps"
                      :key="s.key"
                      class="async-step-row"
                    >
                      <CheckCircleOutlined v-if="s.done" style="color: var(--color-success);" />
                      <LoadingOutlined v-else-if="s.current" spin style="color: var(--color-primary);" />
                      <CloseCircleOutlined v-else-if="s.failed" style="color: var(--color-danger);" />
                      <span v-else class="async-step-pending-dot" />
                      <span>{{ s.label }}</span>
                    </div>
                  </div>
                </a-spin>
                <div v-if="currentResult.ai_reasoning || loading" class="result-section" style="margin-top: 16px;">
                  <h4 class="result-section-title">{{ t('dashboard.aiReasoning') }}</h4>
                  <div class="ai-reasoning">
                    <div class="markdown-body" v-html="renderMarkdown(currentResult.ai_reasoning || '')" />
                  </div>
                </div>
              </div>
            </template>

            <!--审核结果：仅已形成合规结论时展示完整报告（与审核工作台「已完成」一致）-->
            <template v-else-if="currentResult && !loading && archiveDetailShowsComplianceReport">
              <!--与 dashboard 一致的操作栏 -->
              <div class="result-action-bar">
                <a-button @click="openAuditChain(selectedProcess.process_id)">
                  <EyeOutlined /> {{ t('dashboard.auditChain') }}
                </a-button>
                <a-button @click="jumpToOA(selectedProcess.process_id)">
                  <ExportOutlined /> {{ t('dashboard.jumpOA') }}
                </a-button>
                <a-button @click="handleReAudit">
                  <ReloadOutlined /> {{ t('archive.reAudit') }}
                </a-button>
              </div>

              <!--流程摘要：标题 + 申请人/部门/类别 + 当前节点-->
              <div class="archive-process-meta-line">
                <span class="archive-process-meta-line__title">{{ selectedProcess.title }}</span>
                <span>{{ selectedProcess.applicant }} · {{ selectedProcess.department }} · {{ selectedProcess.process_type_label || selectedProcess.process_type }}</span>
                <span><FieldTimeOutlined /> {{ t('dashboard.currentNode') }}: {{ selectedProcess.current_node || '—' }}</span>
              </div>

              <!--合规横幅（与 dashboard result-banner 一致）-->
              <div
                class="result-banner"
                :style="{
                  background: complianceConfig[currentResult.overall_compliance ?? '']?.bg,
                  borderColor: complianceConfig[currentResult.overall_compliance ?? '']?.color,
                }"
              >
                <SafetyCertificateOutlined
                  class="result-banner-icon"
                  :style="{ color: complianceConfig[currentResult.overall_compliance ?? '']?.color }"
                />
                <div class="result-banner-info">
                  <div class="result-banner-title" :style="{ color: complianceConfig[currentResult.overall_compliance ?? '']?.color }">
                    {{ complianceConfig[currentResult.overall_compliance ?? '']?.label }}
                  </div>
                  <div class="result-banner-meta">
                    {{ t('archive.overallScore') }} {{ currentResult.overall_score }} {{ t('archive.score') }}
                    · {{ t('archive.durationLabel') }} {{ formatDuration(currentResult.duration_ms) }}
                  </div>
                </div>
                <div class="result-score" :style="{ color: complianceConfig[currentResult.overall_compliance ?? '']?.color }">
                  {{ currentResult.overall_score }}
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

            <!--未形成合规结论（含从未复盘、失败、解析失败）：与「开始合规复盘」态同一布局 -->
            <div
              v-else-if="selectedProcess && !loading && !archiveDetailShowsComplianceReport"
              class="action-prompt"
            >
              <div class="action-prompt-info">
                <h4>{{ selectedProcess.title }}</h4>
                <p>{{ selectedProcess.applicant }} · {{ selectedProcess.department }} · {{ selectedProcess.submit_time }}</p>
              </div>
              <div class="action-prompt-buttons">
                <a-button type="primary" size="large" @click="handleAudit()">
                  <ThunderboltOutlined /> {{ t('archive.startAudit') }}
                </a-button>
                <a-button size="large" @click="jumpToOA(selectedProcess.process_id)">
                  <ExportOutlined /> {{ t('dashboard.jumpToOASystem') }}
                </a-button>
              </div>
            </div>

          </template>
        </div>
      </div>
    </div>

    <!-- 归档复盘审核链（数据来自 archive_logs，非 OA 待办 audit_logs） -->
    <Teleport to="body">
      <transition name="drawer">
        <div v-if="showHistoryChain" class="drawer-overlay" @click.self="showHistoryChain = false">
          <div class="drawer-panel">
            <div class="drawer-header">
              <h3>{{ t('archive.archiveReviewChainTitle') }}</h3>
              <button type="button" class="drawer-close" @click="showHistoryChain = false">✕</button>
            </div>
            <div class="drawer-body">
              <p class="chain-desc">{{ t('archive.archiveReviewChainDesc') }}</p>
              <a-spin :spinning="auditChainLoading">
                <div v-if="!auditChainLoading && archiveHistoryData.length === 0" style="padding: 40px; text-align: center;">
                  <a-empty :description="t('archive.noArchiveChainRecords')" />
                </div>
                <div v-else class="audit-chain">
                  <div
                    v-for="(item, idx) in archiveHistoryData"
                    :key="item.id"
                    class="chain-node"
                  >
                    <div class="chain-timeline">
                      <div class="chain-dot" :style="{ background: getScoreColorConfig(item.compliance_score)?.color }" />
                      <div v-if="idx < archiveHistoryData.length - 1" class="chain-line" />
                    </div>
                    <div class="chain-card">
                      <div class="chain-card-header" @click="toggleChainNode(item.id)">
                        <span
                          class="chain-tag"
                          :style="{
                            color: complianceConfig[item.compliance]?.color,
                            background: complianceConfig[item.compliance]?.bg,
                          }"
                        >
                          <SafetyCertificateOutlined />
                          {{ complianceConfig[item.compliance]?.label }}
                        </span>
                        <span class="chain-score">{{ item.compliance_score }}{{ t('archive.score') }}</span>
                        <span class="chain-expand-btn">
                          <DownOutlined v-if="!expandedChainNodes.has(item.id)" />
                          <UpOutlined v-else />
                        </span>
                      </div>
                      <div class="chain-card-meta">
                        {{ formatChainDate(item.created_at) }}
                        <span v-if="item.user_name"> · {{ item.user_name }}</span>
                        · {{ t('dashboard.duration') }} {{ getDurationSec(item.duration_ms) }}s
                      </div>
                      <div v-if="expandedChainNodes.has(item.id)" class="chain-detail">
                        <template v-if="item.archive_result">
                          <template v-if="item.archive_result.rule_audit?.length">
                            <div class="chain-section-title">{{ t('archive.ruleAudit') }}</div>
                            <div
                              v-for="(rule, ri) in item.archive_result.rule_audit"
                              :key="ri"
                              class="chain-rule-item"
                              :class="rule.passed ? 'chain-rule--pass' : 'chain-rule--fail'"
                            >
                              <component :is="rule.passed ? CheckCircleOutlined : CloseCircleOutlined" :style="{ color: rule.passed ? 'var(--color-success)' : 'var(--color-danger)' }" />
                              <div>
                                <div class="chain-rule-name">{{ rule.rule_name }}</div>
                                <div class="chain-rule-reasoning">{{ rule.reasoning }}</div>
                              </div>
                            </div>
                          </template>
                          <div v-if="item.archive_result.risk_points?.length || item.archive_result.suggestions?.length" class="risk-suggest-row" style="margin-top: 10px;">
                            <div v-if="item.archive_result.risk_points?.length" class="insight-card insight-card--risk">
                              <div class="insight-card-header">
                                <CloseCircleOutlined style="color: var(--color-danger);" />
                                <span>{{ t('archive.riskPoints') }}</span>
                              </div>
                              <ul class="insight-card-list">
                                <li v-for="(rp, i) in item.archive_result.risk_points" :key="i">{{ rp }}</li>
                              </ul>
                            </div>
                            <div v-if="item.archive_result.suggestions?.length" class="insight-card insight-card--suggest">
                              <div class="insight-card-header">
                                <InfoCircleOutlined style="color: var(--color-primary);" />
                                <span>{{ t('archive.suggestions') }}</span>
                              </div>
                              <ul class="insight-card-list">
                                <li v-for="(sg, i) in item.archive_result.suggestions" :key="i">{{ sg }}</li>
                              </ul>
                            </div>
                          </div>
                          <div v-if="item.archive_result.ai_summary || item.archive_result.ai_reasoning || item.ai_reasoning" class="chain-section-title" style="margin-top: 10px;">{{ t('archive.aiSummary') }}</div>
                          <div v-if="item.archive_result.ai_summary || item.archive_result.ai_reasoning || item.ai_reasoning" class="chain-reasoning">
                            <div class="markdown-body" v-html="renderMarkdown(item.archive_result.ai_summary || item.archive_result.ai_reasoning || item.ai_reasoning || '')" />
                          </div>
                          <div v-if="item.archive_result.parse_error" class="chain-parse-error">
                            <WarningOutlined style="color: var(--color-danger);" />
                            <span>{{ t('dashboard.parseErrorTitle') }}: {{ item.archive_result.parse_error }}</span>
                          </div>
                        </template>
                        <div v-else class="chain-no-detail">{{ t('archive.noArchiveDetail') }}</div>
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
  </div>
</template>

<style scoped>
.archive-page { animation: fadeIn 0.3s ease-out; }
@keyframes fadeIn { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

.page-header { margin-bottom: 24px; display: flex; justify-content: space-between; align-items: flex-start; flex-wrap: wrap; gap: 16px; }
.page-title { font-size: 24px; font-weight: 700; color: var(--color-text-primary); margin: 0; letter-spacing: -0.02em; }
.page-subtitle { font-size: 14px; color: var(--color-text-tertiary); margin: 4px 0 0; }
.page-header-actions { display: flex; gap: 8px; align-items: center; }

/*统计行（与 dashboard 卡片风格一致，保留 4 列）*/
.stats-row { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; margin-bottom: 24px; }
.stat-card {
  background: var(--color-bg-card); border-radius: var(--radius-lg); padding: 20px;
  display: flex; align-items: center; gap: 16px; border: 2px solid var(--color-border-light);
  transition: all var(--transition-base); cursor: pointer; user-select: none;
}
.stat-card:hover { transform: translateY(-2px); box-shadow: var(--shadow-md); }
.stats-row--readonly .stat-card { cursor: not-allowed; }
.stats-row--readonly .stat-card:hover { transform: none; box-shadow: none; }
.stat-card--selected { border-color: var(--color-primary); box-shadow: 0 0 0 1px var(--color-primary); }
.stat-card-icon {
  width: 48px; height: 48px; border-radius: var(--radius-lg);
  display: flex; align-items: center; justify-content: center; font-size: 22px; flex-shrink: 0;
}
.stat-card--primary .stat-card-icon { background: var(--color-primary-bg); color: var(--color-primary); }
.stat-card--success .stat-card-icon { background: var(--color-success-bg); color: var(--color-success); }
.stat-card--warning .stat-card-icon { background: var(--color-warning-bg); color: var(--color-warning); }
.stat-card--danger .stat-card-icon { background: var(--color-danger-bg); color: var(--color-danger); }
.stat-card-info { display: flex; flex-direction: column; }
.stat-card-value { font-size: 28px; font-weight: 700; color: var(--color-text-primary); line-height: 1.2; }
.stat-card-label { font-size: 13px; color: var(--color-text-tertiary); margin-top: 2px; }

/*主格（与 dashboard 同宽）*/
.dashboard-grid { display: grid; grid-template-columns: 420px 1fr; gap: 24px; align-items: start; }
.todo-panel, .result-panel {
  background: var(--color-bg-card); border-radius: var(--radius-lg);
  border: 1px solid var(--color-border-light); overflow: hidden;
}
.panel-header {
  padding: 16px 20px; border-bottom: 1px solid var(--color-border-light);
  display: flex; flex-direction: column; gap: 12px;
}
.panel-header-row { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 12px; }
.panel-header-controls { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; }
.archive-date-label { font-size: 13px; color: var(--color-text-secondary); white-space: nowrap; }
.archive-range-picker { min-width: 240px; }
.panel-title {
  font-size: 15px; font-weight: 600; color: var(--color-text-primary);
  margin: 0; display: flex; align-items: center; gap: 8px;
}
.panel-header-hint { font-size: 12px; color: var(--color-text-tertiary); }
.filter-toggle-btn { position: relative; }
.filter-toggle-btn--active { color: var(--color-primary); border-color: var(--color-primary); }
.filter-active-dot {
  display: inline-block; width: 6px; height: 6px; border-radius: 50%;
  background: var(--color-primary); margin-left: 4px; vertical-align: middle;
}

.filter-bar { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; padding: 10px 0 0; }
.slide-enter-active, .slide-leave-active { transition: all 0.2s ease; }
.slide-enter-from, .slide-leave-to { opacity: 0; transform: translateY(-8px); }

.batch-toolbar { display: flex; align-items: center; justify-content: space-between; padding: 6px 0; gap: 8px; }
.batch-toolbar-left { display: flex; align-items: center; gap: 12px; }
.batch-toolbar-right { display: flex; align-items: center; gap: 8px; }
.batch-limit-hint { font-size: 11px; color: var(--color-text-quaternary); }
.batch-progress-hint { font-size: 12px; font-weight: 600; color: var(--color-primary); animation: auditPulse 1.5s ease-in-out infinite; }
@keyframes auditPulse { 0%, 100% { opacity: 0.6; } 50% { opacity: 1; } }
.batch-audit-btn { flex-shrink: 0; }

/*列表（与 dashboard todo 一致）*/
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
.todo-item--archive-noncompliant { background: rgba(239, 68, 68, 0.04); border-left: 3px solid var(--color-danger); }
.todo-item--audited-approve.todo-item--selected,
.todo-item--audited-return.todo-item--selected,
.todo-item--archive-noncompliant.todo-item--selected { background: var(--color-primary-bg); border-left: 3px solid var(--color-primary); }
.todo-item-audited-icon { font-size: 13px; flex-shrink: 0; }
.todo-item-main { flex: 1; min-width: 0; }
.todo-item-title {
  font-size: 14px; font-weight: 500; color: var(--color-text-primary);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis; margin-bottom: 4px;
  display: flex; align-items: center; gap: 6px;
}
.todo-item-meta { font-size: 12px; color: var(--color-text-tertiary); display: flex; align-items: center; gap: 4px; flex-wrap: wrap; margin-bottom: 6px; }
.todo-item-dot { color: var(--color-border); }

.todo-item-audit-info { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.todo-item-audit-left { display: flex; align-items: center; gap: 8px; min-width: 0; }
.todo-item-audit-right { display: flex; align-items: center; gap: 8px; flex-shrink: 0; }
.todo-item-node {
  font-size: 11px; font-weight: 500; padding: 2px 8px;
  border-radius: var(--radius-full); background: var(--color-bg-hover);
  color: var(--color-text-secondary); white-space: nowrap;
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
  background: var(--color-primary-bg); transform: scale(1.1);
  box-shadow: 0 2px 8px rgba(79, 70, 229, 0.15);
}
.oa-jump-btn:active { transform: scale(0.95); }
.todo-item-score-badge {
  display: inline-flex; align-items: center; gap: 4px;
  font-size: 11px; font-weight: 600; padding: 2px 8px;
  border-radius: var(--radius-full); white-space: nowrap;
}
.todo-item-checkbox { flex-shrink: 0; padding-top: 2px; }
.todo-empty { padding: 48px 20px; }

.pagination-wrapper { padding: 12px 20px; border-top: 1px solid var(--color-border-light); display: flex; justify-content: center; }

/*结果区（与 dashboard 一致）*/
.result-content { padding: 20px; }
.result-action-bar { display: flex; gap: 8px; margin-bottom: 16px; align-items: center; flex-wrap: wrap; }
.archive-process-meta-line {
  display: flex; flex-direction: column; gap: 6px; margin-bottom: 16px;
  font-size: 12px; color: var(--color-text-tertiary); line-height: 1.5;
}
.archive-process-meta-line__title {
  font-size: 15px; font-weight: 600; color: var(--color-text-primary);
}

.result-async-panel { padding: 8px 0 16px; }
.async-progress-steps { display: flex; flex-direction: column; gap: 10px; margin-top: 12px; }
.async-step-row { display: flex; align-items: center; gap: 10px; font-size: 13px; color: var(--color-text-secondary); }
.async-step-pending-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--color-border); display: inline-block; flex-shrink: 0; }

.result-banner {
  display: flex; align-items: center; padding: 16px 20px;
  border-radius: var(--radius-lg); border-left: 4px solid; margin-bottom: 24px; gap: 14px;
}
.result-banner-icon { font-size: 28px; flex-shrink: 0; }
.result-banner-info { flex: 1; }
.result-banner-title { font-size: 16px; font-weight: 700; }
.result-banner-meta { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }
.result-score { font-size: 36px; font-weight: 800; line-height: 1; }

.action-prompt { text-align: center; padding: 40px 20px; }
.action-prompt-info h4 { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin: 0 0 8px; }
.action-prompt-info p { font-size: 13px; color: var(--color-text-tertiary); margin: 0 0 24px; }
.action-prompt-buttons { display: flex; gap: 12px; justify-content: center; flex-wrap: wrap; }

.result-section { margin-bottom: 24px; }
.result-section-title { font-size: 14px; font-weight: 600; color: var(--color-text-primary); margin: 0 0 12px; }

.section-block { margin-bottom: 16px; }
.section-title {
  font-size: 13px; font-weight: 600; color: var(--color-text-primary);
  margin: 0 0 10px; display: flex; align-items: center; gap: 6px;
}

.audit-checks { display: flex; flex-direction: column; gap: 8px; }
.audit-check-item {
  display: flex; gap: 12px; padding: 12px 16px;
  border-radius: var(--radius-md); border: 1px solid var(--color-border-light);
  transition: background var(--transition-fast);
}
.audit-check-item:hover { background: var(--color-bg-hover); }
.audit-check-item--pass { border-left: 3px solid var(--color-success); }
.audit-check-item--fail { border-left: 3px solid var(--color-danger); background: var(--color-danger-bg); }
.audit-check-status { font-size: 18px; flex-shrink: 0; padding-top: 1px; }
.audit-check-content { flex: 1; min-width: 0; }
.audit-check-name { font-size: 14px; font-weight: 500; color: var(--color-text-primary); margin-bottom: 4px; }
.audit-check-reasoning { font-size: 13px; color: var(--color-text-secondary); line-height: 1.5; }
.audit-check-empty { font-size: 13px; color: var(--color-text-tertiary); padding: 12px; text-align: center; }

.risk-suggestions-row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 24px; }
.risk-card, .suggestion-card {
  padding: 16px; background: var(--color-bg-page);
  border-radius: var(--radius-md); border: 1px solid var(--color-border-light);
}
.risk-list, .suggestion-list { display: flex; flex-direction: column; gap: 6px; }
.risk-item, .suggestion-item {
  display: flex; align-items: flex-start; gap: 8px;
  font-size: 13px; color: var(--color-text-secondary); line-height: 1.5;
}
.risk-empty { font-size: 12px; color: var(--color-text-tertiary); }

.ai-summary { background: var(--color-bg-page); border-radius: var(--radius-md); padding: 16px; border: 1px solid var(--color-border-light); }
.ai-reasoning { background: var(--color-bg-page); border-radius: var(--radius-md); padding: 16px; border: 1px solid var(--color-border-light); }
.ai-reasoning pre { white-space: pre-wrap; word-break: break-word; font-family: var(--font-sans); font-size: 13px; line-height: 1.7; color: var(--color-text-secondary); margin: 0; }

.result-empty { text-align: center; padding: 60px 20px; }
.result-empty-icon {
  width: 64px; height: 64px; border-radius: 50%; background: var(--color-primary-bg);
  color: var(--color-primary); font-size: 28px; display: flex; align-items: center;
  justify-content: center; margin: 0 auto 16px;
}
.result-empty p { font-size: 13px; color: var(--color-text-tertiary); margin: 0 auto; max-width: 280px; }
.result-empty--loading {
  display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 16px;
  padding: 48px 20px; min-height: 200px;
}
.result-empty--loading p { margin: 0; }

/*审核链抽屉（与 dashboard 完全一致）*/
.drawer-overlay {
  position: fixed; inset: 0; background: rgba(0,0,0,0.4);
  backdrop-filter: blur(4px); z-index: 1000; display: flex; justify-content: flex-end;
}
.drawer-panel {
  width: 520px; max-width: 100vw; background: var(--color-bg-card);
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
.chain-card-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; cursor: pointer; }
.chain-tag {
  display: inline-flex; align-items: center; gap: 4px;
  font-size: 12px; font-weight: 600; padding: 3px 10px; border-radius: var(--radius-full);
}
.chain-score { font-size: 18px; font-weight: 700; color: var(--color-text-primary); }
.chain-card-meta { font-size: 12px; color: var(--color-text-tertiary); display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
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
.chain-no-detail { font-size: 12px; color: var(--color-text-tertiary); text-align: center; padding: 12px; }
.chain-section-title { font-size: 12px; font-weight: 600; color: var(--color-text-secondary); margin-bottom: 6px; }
.chain-parse-error {
  display: flex; align-items: center; gap: 8px; padding: 8px 12px;
  border-radius: var(--radius-sm); background: var(--color-danger-bg);
  font-size: 12px; color: var(--color-danger);
}

.risk-suggest-row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 24px; }
.risk-suggest-row:has(.insight-card:only-child) { grid-template-columns: 1fr; }
.insight-card { border-radius: var(--radius-md); padding: 16px; border: 1px solid var(--color-border-light); }
.insight-card--risk { background: linear-gradient(135deg, rgba(239, 68, 68, 0.04), rgba(239, 68, 68, 0.01)); border-color: rgba(239, 68, 68, 0.15); }
.insight-card--suggest { background: linear-gradient(135deg, rgba(79, 70, 229, 0.04), rgba(79, 70, 229, 0.01)); border-color: rgba(79, 70, 229, 0.15); }
.insight-card-header { display: flex; align-items: center; gap: 8px; font-size: 13px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 10px; }
.insight-card-list { margin: 0; padding-left: 18px; display: flex; flex-direction: column; gap: 6px; }
.insight-card-list li { font-size: 13px; line-height: 1.6; color: var(--color-text-secondary); }
.insight-card--risk .insight-card-list li { color: var(--color-danger); }

.drawer-enter-active { transition: opacity 0.2s ease; }
.drawer-enter-active .drawer-panel { transition: transform 0.3s cubic-bezier(0.16,1,0.3,1); }
.drawer-leave-active { transition: opacity 0.2s ease 0.1s; }
.drawer-leave-active .drawer-panel { transition: transform 0.2s ease; }
.drawer-enter-from { opacity: 0; }
.drawer-enter-from .drawer-panel { transform: translateX(100%); }
.drawer-leave-to { opacity: 0; }
.drawer-leave-to .drawer-panel { transform: translateX(100%); }

@media (max-width: 1200px) {
  .dashboard-grid { grid-template-columns: 380px 1fr; }
}
@media (max-width: 1024px) {
  .dashboard-grid { grid-template-columns: 1fr; }
  .stats-row { grid-template-columns: repeat(2, 1fr); }
}
@media (max-width: 768px) {
  .stats-row { grid-template-columns: repeat(2, 1fr); }
  .risk-suggestions-row { grid-template-columns: 1fr; }
  .panel-header { padding: 12px 16px; }
  .todo-item { padding: 12px 16px; }
  .result-content { padding: 16px; }
  .action-prompt-buttons { flex-direction: column; }
  .filter-bar { flex-direction: column; align-items: stretch; }
}
@media (max-width: 480px) {
  .stats-row { grid-template-columns: 1fr 1fr; }
  .page-title { font-size: 20px; }
  .result-banner { flex-wrap: wrap; padding: 12px 14px; }
  .result-score { font-size: 28px; }
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
