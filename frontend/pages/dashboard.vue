<script setup lang="ts">
import {
  SearchOutlined,
  ThunderboltOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  ClockCircleOutlined,
  FireOutlined,
  ExportOutlined,
  ReloadOutlined,
  HistoryOutlined,
  EyeOutlined,
  FieldTimeOutlined,
  DownOutlined,
  UpOutlined,
  InfoCircleOutlined,
  LoadingOutlined,
  FilterOutlined,
  WarningOutlined,
  StopOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import dayjs, { type Dayjs } from 'dayjs'
import { marked } from 'marked'
import { useI18n } from '~/composables/useI18n'
import type { OAProcessItem, AuditResult, AuditChainItem, AuditTab, AuditStats } from '~/types/audit'

definePageMeta({ middleware: 'auth' })

const { t } = useI18n()
const { token } = useAuth()
const { getStats, listProcesses, executeAudit, getAuditChain: fetchAuditChain, getProcessTypes, cancelAuditJob, waitAuditJob, batchAudit } = useAuditApi()

// ─── 页签 & 列表数据 ───
const activeTab = ref<AuditTab>('pending_ai')
const processList = ref<OAProcessItem[]>([])
const listLoading = ref(false)
const stats = ref<AuditStats>({ pending_ai_count: 0, ai_done_count: 0, completed_count: 0, today_completed_count: 0 })

// ─── 流程类型选项（从后端获取） ───
const processCascaderOptions = ref<{ label: string; value: string; children: { label: string; value: string }[] }[]>([])

const loadProcessTypes = async () => {
  try {
    const list = await getProcessTypes()
    const categoryMap = new Map<string, { label: string; value: string; children: { label: string; value: string }[] }>()
    for (const item of (Array.isArray(list) ? list : [])) {
      const catLabel = item.process_type_label || item.process_type
      if (!categoryMap.has(catLabel)) {
        categoryMap.set(catLabel, { label: catLabel, value: catLabel, children: [] })
      }
      const cat = categoryMap.get(catLabel)!
      if (!cat.children.some(c => c.value === item.process_type)) {
        cat.children.push({ label: item.process_type, value: item.process_type })
      }
    }
    processCascaderOptions.value = Array.from(categoryMap.values())
  } catch {}
}

// ─── 筛选 ───
const searchText = ref('')
const searchApplicant = ref('')
const filterProcessType = ref<string[][]>([])
const filterDepartment = ref<string | undefined>(undefined)
const filterAuditStatus = ref<string | undefined>(undefined)
const showFilters = ref(false)

/**
 * 列表日期范围：记住当日选择，避免刷新后回到近 90 天导致批量/长周期任务从列表消失；
 * 跨自然日再打开则丢弃（默认近 90 天），便于次日看到相对「最新」的待办。
 */
const DASHBOARD_DATE_RANGE_KEY = 'oa-smart-audit:dashboard:list-date-range'

function defaultDashboardDateRange(): [Dayjs, Dayjs] {
  return [dayjs().subtract(90, 'day').startOf('day'), dayjs().endOf('day')]
}

function readDashboardDateRange(): [Dayjs, Dayjs] | null {
  if (typeof window === 'undefined') return null
  try {
    const r = sessionStorage.getItem(DASHBOARD_DATE_RANGE_KEY)
    if (!r) return null
    const o = JSON.parse(r) as { start?: string; end?: string; savedAt?: string }
    if (!o.start || !o.end) return null
    if (o.savedAt) {
      const saved = dayjs(o.savedAt)
      if (!saved.isValid() || !saved.isSame(dayjs(), 'day')) {
        sessionStorage.removeItem(DASHBOARD_DATE_RANGE_KEY)
        return null
      }
    } else {
      sessionStorage.removeItem(DASHBOARD_DATE_RANGE_KEY)
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

function saveDashboardDateRange(range: [Dayjs, Dayjs]) {
  if (typeof window === 'undefined') return
  try {
    const [a, b] = range
    if (!a?.isValid?.() || !b?.isValid?.()) return
    sessionStorage.setItem(
      DASHBOARD_DATE_RANGE_KEY,
      JSON.stringify({
        start: a.format('YYYY-MM-DD'),
        end: b.format('YYYY-MM-DD'),
        savedAt: new Date().toISOString(),
      }),
    )
  } catch {}
}

function clearDashboardDateRangeStorage() {
  try {
    if (typeof window !== 'undefined') sessionStorage.removeItem(DASHBOARD_DATE_RANGE_KEY)
  } catch {}
}

/** 与后端 OA SQL 一致：按流程创建/提交时间筛待办；默认最近 90 天 */
const auditDateRange = ref<[Dayjs, Dayjs]>(defaultDashboardDateRange())

const auditListDateQuery = () => {
  const r = auditDateRange.value
  if (!r?.[0] || !r?.[1]) return {}
  return {
    start_date: r[0].format('YYYY-MM-DD'),
    end_date: r[1].format('YYYY-MM-DD'),
  }
}

const filterProcessNames = computed(() => {
  if (filterProcessType.value.length === 0) return []
  const names: string[] = []
  for (const path of filterProcessType.value) {
    if (path.length >= 2) {
      names.push(path[path.length - 1])
    } else if (path.length === 1) {
      const cat = processCascaderOptions.value.find((o: any) => o.value === path[0])
      if (cat?.children) names.push(...cat.children.map((c: any) => c.value))
    }
  }
  return names
})

const departmentOptions = computed(() => [...new Set(processList.value.map(p => p.department).filter(Boolean))])

const clearFilters = () => {
  searchText.value = ''
  searchApplicant.value = ''
  filterProcessType.value = []
  filterDepartment.value = undefined
  filterAuditStatus.value = undefined
  auditDateRange.value = defaultDashboardDateRange()
  clearDashboardDateRangeStorage()
  listPage.value = 1
  void Promise.all([loadStats(), loadProcesses()])
}
const hasActiveFilters = computed(() => !!searchText.value || !!searchApplicant.value || filterProcessType.value.length > 0 || !!filterDepartment.value || !!filterAuditStatus.value)

// 第一个页签不展示"AI 审核状态"筛选；第二三个页签去掉"未审核"选项
const showAuditStatusFilter = computed(() => activeTab.value !== 'pending_ai')

/** 仅「全部已完成」为只读历史；待 AI / AI 已审核仍处待办流程，允许重新审核 */
const isCompletedHistoryTab = computed(() => activeTab.value === 'completed')

/** 「待 AI」与「AI 已审核」共用批量勾选、批量审核与中止（与待办一致） */
const isBatchAuditTab = computed(() => activeTab.value === 'pending_ai' || activeTab.value === 'ai_done')
const auditStatusOptions = computed(() => {
  const opts = [
    { value: 'approve', label: t('dashboard.auditStatus.approve') },
    { value: 'return', label: t('dashboard.auditStatus.return') },
    { value: 'review', label: t('dashboard.auditStatus.review') },
  ]
  return opts
})

const listPage = ref(1)
const listPageSize = ref(10)
const listTotal = ref(0)

// ─── 选中流程 & 审核结果 ───
const selectedProcess = ref<string | null>(null)
const currentResult = ref<AuditResult | null>(null)
const loading = ref(false)
const phase1Done = ref(false)

const selectedProcessInfo = computed(() => processList.value.find(p => p.process_id === selectedProcess.value))

/** 摘要第一行仅展示流程标题（避免与下方元信息重复） */
const dashboardProcessSummaryTitle = computed(() => selectedProcessInfo.value?.title ?? '')

const dashboardProcessMetaLine2 = computed(() => {
  const p = selectedProcessInfo.value
  if (!p) return ''
  const ap = p.applicant?.trim() || '—'
  const dept = p.department?.trim() || '—'
  const cat = (p.process_type_label || p.process_type || '').trim() || '—'
  return `${ap} · ${dept} · ${cat}`
})

// ─── 批量审核 ───
const selectedProcessIds = ref<string[]>([])
const batchAuditing = ref(false)
const batchAborted = ref(false)
const batchAuditTotal = ref(0)
const batchAuditDone = ref(0)
const currentInflightProcessId = ref<string | null>(null)
const pollProcessId = ref<string | null>(null)

const asyncAuditStatuses = ['pending', 'assembling', 'reasoning', 'extracting'] as const

const auditInProgress = computed(() => {
  if (loading.value || batchAuditing.value) return true
  return processList.value.some(p => p.audit_status && asyncAuditStatuses.includes(p.audit_status as any))
})

const blockingProcessId = computed((): string | null => {
  if (loading.value && selectedProcess.value) return selectedProcess.value
  if (batchAuditing.value) return currentInflightProcessId.value
  const row = processList.value.find(p => p.audit_status && asyncAuditStatuses.includes(p.audit_status as any))
  return row?.process_id ?? null
})

type DashboardBatchMeta = { process_id: string; process_type: string; title: string }
type DashboardBatchPersisted = {
  ids: string[]
  queueMeta: DashboardBatchMeta[]
  nextIndex: number
  /** 发起批量时所在页签，刷新后先恢复，避免待办/已审核列表错位 */
  activeTab?: AuditTab
  /** 当前条异步任务 audit_logs.id，列表未加载或仍为旧快照时用于 POST /api/audit/cancel/:id */
  inflightJobId?: string
}

const DASHBOARD_BATCH_KEY = 'oa-smart-audit:dashboard:batch-queue'

function saveDashboardBatchState(
  ids: string[],
  queueMeta: DashboardBatchMeta[],
  nextIndex: number,
  inflightJobId?: string | null,
) {
  try {
    const payload: DashboardBatchPersisted = {
      ids,
      queueMeta,
      nextIndex,
      activeTab: activeTab.value,
    }
    if (inflightJobId) payload.inflightJobId = inflightJobId
    sessionStorage.setItem(DASHBOARD_BATCH_KEY, JSON.stringify(payload))
  } catch {}
}

function clearDashboardBatchStorage() {
  try {
    sessionStorage.removeItem(DASHBOARD_BATCH_KEY)
  } catch {}
}

function readDashboardBatchState(): DashboardBatchPersisted | null {
  try {
    const r = sessionStorage.getItem(DASHBOARD_BATCH_KEY)
    if (!r) return null
    return JSON.parse(r) as DashboardBatchPersisted
  } catch {
    return null
  }
}

/** 中止时：优先列表行上的任务 id，否则用批量持久化里的 inflightJobId（刷新后列表可能尚无 id） */
function resolveAuditJobIdForCancel(processId: string): string | undefined {
  const item = processList.value.find(p => p.process_id === processId)
  if (item?.audit_result?.id) return item.audit_result.id
  const st = readDashboardBatchState()
  if (st?.inflightJobId && st.ids[st.nextIndex] === processId) return st.inflightJobId
  return undefined
}

// ─── 流式推理 SSE ───
const eventSourceStream = ref<EventSource | null>(null)

const disconnectStream = () => {
  if (eventSourceStream.value) {
    eventSourceStream.value.close()
    eventSourceStream.value = null
  }
}

const startSSE = (auditResultId: string, processId: string) => {
  disconnectStream()
  const tokenVal = token.value || localStorage.getItem('token') || ''
  const config = useRuntimeConfig()
  const url = `${String(config.public.apiBase)}/api/audit/stream/${auditResultId}?token=${encodeURIComponent(tokenVal)}`
  
  eventSourceStream.value = new EventSource(url)
  
  if (!currentResult.value?.ai_reasoning && selectedProcess.value === processId) {
    if (currentResult.value) currentResult.value.ai_reasoning = ''
  }
  
  eventSourceStream.value.onmessage = (e) => {
    if (selectedProcess.value === processId && currentResult.value) {
      currentResult.value.ai_reasoning = (currentResult.value.ai_reasoning || '') + e.data
    }
  }
  
  eventSourceStream.value.onerror = () => {
    disconnectStream()
  }
}

const renderMarkdown = (text: string) => text ? marked.parse(text) : ''

// ─── 数据选择与操作 ───
const toggleSelectProcess = (processId: string) => {
  if (auditInProgress.value) return
  const idx = selectedProcessIds.value.indexOf(processId)
  if (idx >= 0) selectedProcessIds.value.splice(idx, 1)
  else if (selectedProcessIds.value.length < 10) selectedProcessIds.value.push(processId)
  else message.warning(t('dashboard.batchLimitHint'))
}

const selectableIdsComputed = computed(() => {
  return processList.value
    .filter(p => !(p.audit_status && ['pending', 'assembling', 'reasoning', 'extracting'].includes(p.audit_status)))
    .map(p => p.process_id)
})

const toggleSelectAll = () => {
  if (auditInProgress.value) return
  const selectableIds = selectableIdsComputed.value
  
  if (selectedProcessIds.value.length === Math.min(selectableIds.length, 10) || selectableIds.length === 0) {
    selectedProcessIds.value = []
  } else {
    selectedProcessIds.value = selectableIds.slice(0, 10)
  }
}

// ─── 审核链 ───
const showHistoryChain = ref(false)
const historyChainProcessId = ref<string | null>(null)
const auditChainData = ref<AuditChainItem[]>([])
const auditChainLoading = ref(false)
const expandedChainNodes = ref<Set<string>>(new Set())

const toggleChainNode = (id: string) => {
  if (expandedChainNodes.value.has(id)) expandedChainNodes.value.delete(id)
  else expandedChainNodes.value.add(id)
}

// ─── 数据加载 ───
const loadStats = async () => {
  try {
    stats.value = await getStats(auditListDateQuery())
  } catch {}
}

const loadProcesses = async () => {
  listLoading.value = true
  try {
    const pt =
      filterProcessNames.value.length > 0 ? filterProcessNames.value.join(',') : undefined
    const res = await listProcesses(activeTab.value, {
      keyword: searchText.value || undefined,
      applicant: searchApplicant.value || undefined,
      process_type: pt,
      department: filterDepartment.value || undefined,
      audit_status: filterAuditStatus.value || undefined,
      page: listPage.value,
      page_size: listPageSize.value,
      ...auditListDateQuery(),
    })
    processList.value = res?.items ?? []
    listTotal.value = res?.total ?? 0
    if (res?.page) listPage.value = res.page
    if (res?.page_size) listPageSize.value = res.page_size

    // 刷新/分页后：若当前选中行仍在进行中，恢复轮询与 SSE（列表徽标本就依赖 audit_status）
    if (selectedProcess.value) {
      const row = processList.value.find(p => p.process_id === selectedProcess.value)
      if (
        row?.audit_result?.id
        && row.audit_status
        && asyncAuditStatuses.includes(row.audit_status as any)
      ) {
        handleSelectProcess(selectedProcess.value)
      }
    }
  } catch (e: any) {
    message.error(t('dashboard.loadFailed'))
    processList.value = []
    listTotal.value = 0
  } finally {
    listLoading.value = false
  }
}

let filterDebounceTimer: ReturnType<typeof setTimeout> | null = null
const triggerFilterReload = () => {
  if (auditInProgress.value) return
  if (filterDebounceTimer) clearTimeout(filterDebounceTimer)
  filterDebounceTimer = setTimeout(() => {
    listPage.value = 1
    loadProcesses()
  }, 400)
}

watch([searchText, searchApplicant, filterProcessNames, filterDepartment, filterAuditStatus], () => {
  triggerFilterReload()
})

const onAuditDateRangeChange = () => {
  if (auditInProgress.value) {
    message.warning(t('dashboard.auditInProgressNoSwitch'))
    return
  }
  saveDashboardDateRange(auditDateRange.value)
  listPage.value = 1
  void Promise.all([loadStats(), loadProcesses()])
}

const onListPageChange = (page: number, size: number) => {
  listPage.value = page
  listPageSize.value = size
  loadProcesses()
}

const switchTab = (tab: AuditTab) => {
  if (auditInProgress.value) {
    message.warning(t('dashboard.auditInProgressNoSwitch'))
    return
  }
  disconnectStream()
  pollProcessId.value = null
  showHistoryChain.value = false
  loading.value = false
  activeTab.value = tab
  processList.value = []
  selectedProcess.value = null
  currentResult.value = null
  listPage.value = 1
  selectedProcessIds.value = []
  filterAuditStatus.value = undefined
  loadProcesses()
  loadStats()
}

const handleSelectProcess = (processId: string) => {
  if (auditInProgress.value) {
    const allow = blockingProcessId.value
    if (allow && processId !== allow) {
      message.warning(t('dashboard.auditInProgressNoSwitch'))
      return
    }
  }
  selectedProcess.value = processId
  const item = processList.value.find(p => p.process_id === processId)
  if (item?.audit_result) {
    currentResult.value = { ...item.audit_result }
    const st = item.audit_status ?? item.audit_result.status
    if (st && ['pending', 'assembling', 'reasoning', 'extracting'].includes(st)) {
       if (!currentResult.value.progress_steps) {
         currentResult.value.progress_steps = [
           { key: 'pending', label: '排队中', done: st !== 'pending', current: st === 'pending' },
           { key: 'assembling', label: '组装提示词', done: ['reasoning','extracting'].includes(st), current: st === 'assembling' },
           { key: 'reasoning', label: '推理分析', done: st === 'extracting', current: st === 'reasoning' },
           { key: 'extracting', label: '结构化提取', done: false, current: st === 'extracting' },
         ]
       }

       if (item.audit_result.id && pollProcessId.value !== processId) {
          pollProcessId.value = processId
          startSSE(item.audit_result.id, processId)
          waitAuditJob(item.audit_result.id, (st: any) => {
             if (selectedProcess.value === processId) {
                 const oldReasoning = currentResult.value?.ai_reasoning || ''
                 currentResult.value = st as any
                 if (oldReasoning.length > (st.ai_reasoning?.length || 0)) {
                     if (currentResult.value) currentResult.value.ai_reasoning = oldReasoning
                 }
             }
             item.audit_status = st.status as any
             item.audit_result = st as any
          }).then((st: any) => {
             if (pollProcessId.value === processId) pollProcessId.value = null
             if (selectedProcess.value === processId) {
                 const oldReasoning = currentResult.value?.ai_reasoning || ''
                 currentResult.value = st as any
                 if (oldReasoning.length > (st.ai_reasoning?.length || 0)) {
                     if (currentResult.value) currentResult.value.ai_reasoning = oldReasoning
                 }
                 disconnectStream()
             }
             item.audit_status = st.status as any
             item.audit_result = st as any
             item.has_audit = st.status === 'completed'
             loadProcesses()
             loadStats()
          }).catch(() => {
             if (pollProcessId.value === processId) {
               pollProcessId.value = null
               disconnectStream()
             }
          })
       } else if (item.audit_result.id && pollProcessId.value === processId) {
          // 列表已恢复轮询后再次 handleSelectProcess：仅补挂 SSE
          startSSE(item.audit_result.id, processId)
       }
    }
  } else {
    currentResult.value = null
  }
}

// ─── 单条审核 ───
const handleAudit = async (processId: string) => {
  const item = processList.value.find(p => p.process_id === processId)
  if (!item) return
  loading.value = true
  currentResult.value = {
    progress_steps: [
      { key: 'pending', label: '排队中', done: false, current: true },
      { key: 'assembling', label: '组装提示词', done: false, current: false },
      { key: 'reasoning', label: '推理分析', done: false, current: false },
      { key: 'extracting', label: '结构化提取', done: false, current: false },
    ],
    status: 'pending',
    process_id: processId,
    ai_reasoning: '',
    id: undefined,
  } as any
  item.audit_result = { ...currentResult.value } as any
  let started = false

  try {
    const result = await executeAudit({
      process_id: processId,
      process_type: item.process_type,
      title: item.title,
    }, (st: any) => {
      if (!started) {
         started = true
         loadStats()
         if (st?.id) {
           startSSE(st.id, processId)
         }
      }
      if (selectedProcess.value === processId) {
         const oldReasoning = currentResult.value?.ai_reasoning || ''
         currentResult.value = { ...st } as any
         if (oldReasoning.length > (st.ai_reasoning?.length || 0)) {
             if (currentResult.value) currentResult.value.ai_reasoning = oldReasoning
         }
      }
      item.audit_status = st.status as any
      item.audit_result = { ...st } as any
    })
    
    if (selectedProcess.value === processId) {
      const oldReasoning = currentResult.value?.ai_reasoning || ''
      currentResult.value = result as any
      if (oldReasoning.length > (result.ai_reasoning?.length || 0)) {
          if (currentResult.value) currentResult.value.ai_reasoning = oldReasoning
      }
      disconnectStream()
    }
    item.audit_status = result.status as any
    item.audit_result = result
    item.has_audit = result.status === 'completed'
    await loadStats()
    await loadProcesses()
  } catch (e: any) {
    message.error(e?.message || t('dashboard.auditFailed'))
    currentResult.value = null
  } finally {
    loading.value = false
  }
}

const handleReAudit = async () => {
  if (!selectedProcess.value) return
  const pid = selectedProcess.value
  await handleAudit(pid)
  // 如果是已完成页签重新审核，它会回到待审核并从当前列表消失，为了不引发困惑，我们不自动退回，而是通过 loadProcesses() 刷新列表
}

const handleCancelAudit = async (processId: string) => {
  try {
    const jobId = resolveAuditJobIdForCancel(processId)
    if (!jobId) {
      message.warning(t('dashboard.noAuditIdFound', '无法中止，任务 ID 缺失'))
      return
    }
    await cancelAuditJob(jobId)
    message.success(t('dashboard.cancelSuccess', '中止成功'))
    await loadProcesses()
    if (selectedProcess.value === processId) {
      handleSelectProcess(processId)
    }
  } catch (e: any) {
    message.error(t('dashboard.cancelFailed', '中止失败: ') + e.message)
  }
}

// ─── 批量审核（逐条调用，支持中途退出；sessionStorage 支持刷新后续跑） ───
async function runDashboardBatchLoop(ids: string[], queueMeta: DashboardBatchMeta[], startIndex: number) {
  batchAuditing.value = true
  batchAborted.value = false
  const metaById = new Map(queueMeta.map(m => [m.process_id, m]))
  const prev = readDashboardBatchState()
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
  saveDashboardBatchState(ids, queueMeta, startIndex, carryInflight)

  for (let i = startIndex; i < ids.length; i++) {
    if (batchAborted.value) break
    const id = ids[i]
    currentInflightProcessId.value = id
    const item = processList.value.find(p => p.process_id === id)
    const meta = item
      ? { process_id: item.process_id, process_type: item.process_type, title: item.title }
      : metaById.get(id)
    if (!meta) {
      batchAuditDone.value = i + 1
      saveDashboardBatchState(ids, queueMeta, i + 1)
      continue
    }

    let started = false
    let persistedJobId = false
    if (selectedProcess.value === id) {
      currentResult.value = {
        progress_steps: [
          { key: 'pending', label: '排队中', done: false, current: true },
          { key: 'assembling', label: '组装提示词', done: false, current: false },
          { key: 'reasoning', label: '推理分析', done: false, current: false },
          { key: 'extracting', label: '结构化提取', done: false, current: false },
        ],
        status: 'pending',
        process_id: id,
        ai_reasoning: '',
        id: undefined,
      } as any
    }
    if (item) {
      item.audit_result = {
        status: 'pending',
        process_id: id,
        ai_reasoning: '',
        id: undefined,
      } as any
    }

    try {
      const result = await executeAudit({
        process_id: meta.process_id,
        process_type: meta.process_type,
        title: meta.title,
      }, (st: any) => {
        if (st?.id && !persistedJobId) {
          persistedJobId = true
          saveDashboardBatchState(ids, queueMeta, i, st.id)
        }
        if (!started) {
          started = true
          if (st?.id && selectedProcess.value === id) {
            startSSE(st.id, id)
          }
        }
        if (selectedProcess.value === id) {
          const oldReasoning = currentResult.value?.ai_reasoning || ''
          currentResult.value = { ...st } as any
          if (oldReasoning.length > (st.ai_reasoning?.length || 0)) {
            if (currentResult.value) currentResult.value.ai_reasoning = oldReasoning
          }
        }
        if (item) {
          item.audit_status = st.status as any
          item.audit_result = { ...st } as any
        }
      })
      if (item) {
        item.audit_status = result.status as any
        item.audit_result = result as any
        item.has_audit = result.status === 'completed'
      }
      if (selectedProcess.value === id) {
        const oldReasoning = currentResult.value?.ai_reasoning || ''
        currentResult.value = result as any
        if (oldReasoning.length > (result.ai_reasoning?.length || 0)) {
          if (currentResult.value) currentResult.value.ai_reasoning = oldReasoning
        }
        disconnectStream()
      }
    } catch {}
    batchAuditDone.value = i + 1
    saveDashboardBatchState(ids, queueMeta, i + 1)
  }

  currentInflightProcessId.value = null
  batchAuditing.value = false
  selectedProcessIds.value = []
  clearDashboardBatchStorage()
  if (batchAborted.value) message.info(t('dashboard.batchAborted'))
  else message.success(t('dashboard.batchDone'))
  await loadStats()
  await loadProcesses()
}

const handleBatchAudit = async () => {
  if (selectedProcessIds.value.length === 0) return
  if (selectedProcessIds.value.length > 10) {
    message.warning(t('dashboard.batchLimitHint'))
    return
  }
  const ids = [...selectedProcessIds.value]
  const queueMeta: DashboardBatchMeta[] = ids.map(id => {
    const it = processList.value.find(p => p.process_id === id)
    if (!it) return null
    return { process_id: it.process_id, process_type: it.process_type, title: it.title }
  }).filter(Boolean) as DashboardBatchMeta[]
  if (queueMeta.length !== ids.length) {
    message.error(t('dashboard.loadFailed'))
    return
  }
  clearDashboardBatchStorage()
  batchAuditTotal.value = ids.length
  batchAuditDone.value = 0
  await runDashboardBatchLoop(ids, queueMeta, 0)
}

function tryResumeDashboardBatch() {
  const state = readDashboardBatchState()
  if (!state || state.ids.length === 0) return
  if (state.nextIndex >= state.ids.length) {
    clearDashboardBatchStorage()
    return
  }
  const { ids, queueMeta, nextIndex } = state
  batchAuditTotal.value = ids.length
  batchAuditDone.value = nextIndex
  selectedProcessIds.value = [...ids]
  void runDashboardBatchLoop(ids, queueMeta, nextIndex)
}

const handleAbortBatch = async () => {
  batchAborted.value = true
  if (currentInflightProcessId.value) {
    const processId = currentInflightProcessId.value
    await handleCancelAudit(processId).catch(() => {})
    currentInflightProcessId.value = null
  }
  for (const id of selectedProcessIds.value) {
    const item = processList.value.find(p => p.process_id === id)
    if (item && item.audit_status === 'pending') {
      item.audit_status = 'failed'
      item.audit_result = { status: 'failed', error_message: '批量审核已中止', recommendation: 'review' } as any
    }
  }
  clearDashboardBatchStorage()
}

const handleUnifiedAbort = async () => {
  if (batchAuditing.value) {
    await handleAbortBatch()
    return
  }
  const pid = blockingProcessId.value
  if (pid) await handleCancelAudit(pid)
}

// ─── 审核链 ───
const openAuditChain = async (processId: string) => {
  historyChainProcessId.value = processId
  expandedChainNodes.value = new Set()
  showHistoryChain.value = true
  auditChainLoading.value = true
  try {
    auditChainData.value = await fetchAuditChain(processId)
  } catch {
    auditChainData.value = []
  } finally {
    auditChainLoading.value = false
  }
}

// ─── OA 跳转 ───
const jumpToOA = (processId: string) => {
  message.info(t('dashboard.jumpingToOA', `Jumping to OA: ${processId}...`))
}

// ─── 配置常量 ───
const tabConfig = computed(() => [
  { key: 'pending_ai' as AuditTab, icon: ClockCircleOutlined, count: stats.value.pending_ai_count, label: t('dashboard.tab.pendingAI'), cssClass: 'stat-card--primary' },
  { key: 'ai_done' as AuditTab, icon: CheckCircleOutlined, count: stats.value.ai_done_count, label: t('dashboard.tab.aiDone'), cssClass: 'stat-card--success' },
  { key: 'completed' as AuditTab, icon: HistoryOutlined, count: stats.value.completed_count, label: t('dashboard.tab.completed'), cssClass: 'stat-card--info' },
])

const viewModeLabel = computed(() => {
  const map: Record<AuditTab, string> = {
    pending_ai: t('dashboard.viewMode.pendingAI'),
    ai_done: t('dashboard.viewMode.aiDone'),
    completed: t('dashboard.viewMode.completed'),
  }
  return map[activeTab.value]
})

const urgencyConfig = computed<Record<string, { color: string; bg: string; label: string }>>(() => ({
  high: { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)', label: t('dashboard.urgency.high') },
  medium: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', label: t('dashboard.urgency.medium') },
  low: { color: 'var(--color-success)', bg: 'var(--color-success-bg)', label: t('dashboard.urgency.low') },
}))

const isResultAsyncRunning = (r: AuditResult | null) =>
  !!(r?.status && ['pending', 'assembling', 'reasoning', 'extracting'].includes(r.status))

/** 「AI 已审核 / 全部已完成」页签下，已出结果时展示流程摘要（与归档复盘区一致） */
const showDashboardProcessSummary = computed(
  () =>
    (activeTab.value === 'ai_done' || activeTab.value === 'completed')
    && !!selectedProcessInfo.value
    && !!currentResult.value
    && !isResultAsyncRunning(currentResult.value),
)

const filteredProgressSteps = computed(() => {
  if (!currentResult.value?.progress_steps) return []
  if (batchAuditing.value) return currentResult.value.progress_steps
  return currentResult.value.progress_steps.filter(s => s.key !== 'pending')
})

const recommendationConfig = computed<Record<string, { color: string; bg: string; icon: any; label: string }>>(() => ({
  approve: { color: 'var(--color-success)', bg: 'var(--color-success-bg)', icon: CheckCircleOutlined, label: t('dashboard.rec.approve') },
  return: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', icon: ReloadOutlined, label: t('dashboard.rec.return') },
  review: { color: 'var(--color-info)', bg: 'var(--color-info-bg)', icon: EyeOutlined, label: t('dashboard.rec.review') },
}))

const getShortRecLabel = (rec: string) => {
  const map: Record<string, string> = { approve: t('dashboard.suggestApprove'), return: t('dashboard.suggestReturn'), review: t('dashboard.suggestReview') }
  return map[rec] || rec
}

const getScoreColorConfig = (score: number | undefined) => {
  if (score === undefined || score === null) return { color: 'var(--color-info)', bg: 'var(--color-info-bg)' }
  if (score < 60) return { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)' }
  if (score > 80) return { color: 'var(--color-success)', bg: 'var(--color-success-bg)' }
  return { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)' }
}

const formatChainDate = (dateStr: string) => {
  if (!dateStr) return '';
  const d = new Date(dateStr)
  return isNaN(d.getTime()) ? dateStr : d.toLocaleString('zh-CN', { hour12: false }).replace(/\//g, '-')
}

const getDurationSec = (ms: number | undefined) => {
  if (ms === undefined) return 0
  return (ms / 1000).toFixed(1)
}

/** 审核链 GET /api/audit/chain 返回的是 audit_logs 整行：推理在顶层 ai_reasoning；audit_result JSONB 仅存结构化字段，不含推理正文 */
const chainItemAiReasoning = (item: AuditChainItem) =>
  (item.ai_reasoning || item.audit_result?.ai_reasoning || '').trim()

// ─── 初始化 ───
onMounted(async () => {
  const restoredRange = readDashboardDateRange()
  if (restoredRange) auditDateRange.value = restoredRange
  const pending = readDashboardBatchState()
  if (pending && pending.ids.length > 0 && pending.nextIndex < pending.ids.length) {
    const tab = pending.activeTab
    if (tab === 'pending_ai' || tab === 'ai_done' || tab === 'completed') {
      activeTab.value = tab
    }
  }
  loadStats()
  await loadProcesses()
  loadProcessTypes()
  tryResumeDashboardBatch()
})
</script>

<template>
  <div class="dashboard">
    <!--页眉-->
    <div class="page-header">
      <div>
        <h1 class="page-title">{{ t('dashboard.title') }}</h1>
        <p class="page-subtitle">{{ t('dashboard.subtitleWithCount', `${stats.today_completed_count ?? 0}`) }}</p>
      </div>
    </div>

    <!--统计行 - 3 个页签卡片-->
    <div class="stats-row">
      <div
        v-for="tab in tabConfig"
        :key="tab.key"
        class="stat-card"
        :class="[tab.cssClass, { 'stat-card--selected': activeTab === tab.key }]"
        @click="switchTab(tab.key)"
      >
        <div class="stat-card-icon"><component :is="tab.icon" /></div>
        <div class="stat-card-info">
          <span class="stat-card-value">{{ tab.count }}</span>
          <span class="stat-card-label">{{ tab.label }}</span>
        </div>
      </div>
    </div>

    <!--主要内容区-->
    <div class="dashboard-grid">
      <!--左：流程列表-->
      <div class="todo-panel">
        <div class="panel-header">
          <div class="panel-header-row">
            <h3 class="panel-title">
              <FireOutlined v-if="activeTab === 'pending_ai'" style="color: var(--color-primary);" />
              <CheckCircleOutlined v-else-if="activeTab === 'ai_done'" style="color: var(--color-success);" />
              <HistoryOutlined v-else style="color: var(--color-text-tertiary);" />
              {{ viewModeLabel }}
              <a-badge :count="listTotal" :number-style="{ backgroundColor: 'var(--color-primary)' }" />
            </h3>
            <div class="panel-header-controls">
              <span class="dashboard-audit-date-label">{{ t('dashboard.listDateRange') }}</span>
              <a-range-picker
                v-model:value="auditDateRange"
                :allow-clear="false"
                class="dashboard-audit-range-picker"
                @change="onAuditDateRangeChange"
              />
              <a-button size="small" type="default" @click="showFilters = !showFilters" class="filter-toggle-btn" :class="{ 'filter-toggle-btn--active': hasActiveFilters }">
                <FilterOutlined />
                {{ t('dashboard.filter') }}
                <span v-if="hasActiveFilters" class="filter-active-dot" />
              </a-button>
            </div>
          </div>
          <!--可折叠筛选条-->
          <transition name="slide">
            <div v-if="showFilters" class="filter-bar">
              <a-input v-model:value="searchText" :placeholder="t('dashboard.searchPlaceholder')" allow-clear style="flex: 2; min-width: 160px;">
                <template #prefix><SearchOutlined style="color: var(--color-text-tertiary);" /></template>
              </a-input>
              <a-input v-model:value="searchApplicant" :placeholder="t('dashboard.searchApplicant')" allow-clear style="flex: 1; min-width: 130px;">
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
              <a-select v-model:value="filterDepartment" :placeholder="t('dashboard.filterDepartment')" allow-clear style="flex: 1; min-width: 120px;">
                <a-select-option v-for="d in departmentOptions" :key="d" :value="d">{{ d }}</a-select-option>
              </a-select>
              <a-select
                v-if="showAuditStatusFilter"
                v-model:value="filterAuditStatus"
                :placeholder="t('dashboard.filterAuditStatus')"
                allow-clear
                style="flex: 1; min-width: 130px;"
              >
                <a-select-option v-for="opt in auditStatusOptions" :key="opt.value" :value="opt.value">{{ opt.label }}</a-select-option>
              </a-select>
              <a-button size="small" @click="clearFilters">{{ t('dashboard.filterReset') }}</a-button>
            </div>
          </transition>
          <!--批量审核工具栏（待 AI / AI 已审核）-->
          <div v-if="isBatchAuditTab" class="batch-toolbar">
            <div class="batch-toolbar-left">
              <a-checkbox
                :disabled="auditInProgress"
                :checked="selectedProcessIds.length > 0 && selectedProcessIds.length === Math.min(selectableIdsComputed.length, 10)"
                :indeterminate="selectedProcessIds.length > 0 && selectedProcessIds.length < Math.min(selectableIdsComputed.length, 10)"
                @change="toggleSelectAll"
              >
                {{ selectedProcessIds.length > 0 ? t('dashboard.selected', `${selectedProcessIds.length}`) : t('dashboard.selectAll') }}
              </a-checkbox>
              <span class="batch-limit-hint">{{ t('dashboard.batchLimitLabel') }}</span>
              <span v-if="batchAuditing" class="batch-progress-hint">
                {{ t('dashboard.auditedProgress', `${batchAuditDone}/${batchAuditTotal}`) }}
              </span>
              <span v-else-if="auditInProgress && !batchAuditing" class="batch-progress-hint">
                {{ t('dashboard.auditingItem') }}
              </span>
            </div>
            <div class="batch-toolbar-right">
              <a-button
                v-if="auditInProgress"
                size="small"
                danger
                @click="handleUnifiedAbort"
              >
                <StopOutlined /> {{ t('dashboard.batchAbort') }}
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
                {{ t('dashboard.batchAudit') }}
              </a-button>
            </div>
          </div>
        </div>

        <a-spin :spinning="listLoading">
          <div class="todo-list">
            <div
              v-for="item in processList"
              :key="item.process_id"
              class="todo-item"
              :class="{
                'todo-item--selected': selectedProcess === item.process_id,
                'todo-item--audited-approve': item.has_audit && item.audit_result?.recommendation === 'approve',
                'todo-item--audited-return': item.has_audit && item.audit_result?.recommendation === 'return',
                'todo-item--audited-review': item.has_audit && item.audit_result?.recommendation === 'review',
              }"
              @click="handleSelectProcess(item.process_id)"
            >
              <div v-if="isBatchAuditTab" class="todo-item-checkbox" @click.stop="auditInProgress || (item.audit_status && ['pending', 'assembling', 'reasoning', 'extracting'].includes(item.audit_status)) ? null : toggleSelectProcess(item.process_id)">
                <a-checkbox :checked="selectedProcessIds.includes(item.process_id)" :disabled="auditInProgress || (item.audit_status && ['pending', 'assembling', 'reasoning', 'extracting'].includes(item.audit_status))" />
              </div>
              <div class="todo-item-main">
                <div class="todo-item-title">
                  <LoadingOutlined
                    v-if="item.audit_status && ['pending', 'assembling', 'reasoning', 'extracting'].includes(item.audit_status)"
                    class="todo-item-audited-icon"
                    spin
                    style="color: var(--color-primary);"
                  />
                  <CheckCircleOutlined
                    v-else-if="item.has_audit && item.audit_result"
                    class="todo-item-audited-icon"
                    :style="{ color: getScoreColorConfig(item.audit_result.overall_score).color }"
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
                <div class="todo-item-audit-info">
                  <div class="todo-item-audit-left">
                    <span class="todo-item-node">{{ item.current_node }}</span>
                    <span class="todo-item-process-type">{{ item.process_type_label || item.process_type }}</span>
                  </div>
                  <div class="todo-item-audit-right">
                    <span
                      v-if="item.audit_status && ['pending', 'assembling', 'reasoning', 'extracting'].includes(item.audit_status)"
                      class="todo-item-score-badge"
                      style="color: var(--color-primary); background: var(--color-primary-bg);"
                    >
                      {{ t('dashboard.auditingItem') }}
                    </span>
                    <span
                      v-else-if="item.has_audit && item.audit_result"
                      class="todo-item-score-badge"
                      :style="{
                        color: getScoreColorConfig(item.audit_result.overall_score).color,
                        background: getScoreColorConfig(item.audit_result.overall_score).bg,
                      }"
                    >
                      {{ item.audit_result.overall_score }}{{ t('dashboard.points') }}
                      {{ getShortRecLabel(item.audit_result.recommendation || 'review') }}
                    </span>
                    <a-tooltip :title="t('dashboard.auditChain')" :mouse-enter-delay="0.5">
                      <button class="oa-jump-btn" @click.stop="openAuditChain(item.process_id)">
                        <HistoryOutlined />
                      </button>
                    </a-tooltip>
                    <a-tooltip :title="t('dashboard.jumpToOA')" :mouse-enter-delay="0.5">
                      <button class="oa-jump-btn" @click.stop="jumpToOA(item.process_id)">
                        <ExportOutlined />
                      </button>
                    </a-tooltip>
                  </div>
                </div>
              </div>
            </div>

            <div v-if="processList.length === 0 && !listLoading" class="todo-empty">
              <a-empty :description="t('dashboard.noData')" />
            </div>
          </div>
        </a-spin>

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

      <!--右：审核结果面板-->
      <div class="result-panel">
        <div class="panel-header">
          <h3 class="panel-title">
            <ThunderboltOutlined v-if="activeTab === 'pending_ai'" style="color: var(--color-primary);" />
            <CheckCircleOutlined v-else-if="activeTab === 'ai_done'" style="color: var(--color-success);" />
            <HistoryOutlined v-else style="color: var(--color-text-tertiary);" />
            {{ isCompletedHistoryTab ? t('dashboard.historyResult') : t('dashboard.auditResult') }}
          </h3>
        </div>

        <div class="result-content">
          <!--切换页签或列表加载中：不显示上一条详情 -->
          <div v-if="listLoading && !selectedProcess" class="result-empty result-empty--loading">
            <a-spin size="large" />
            <p>{{ t('dashboard.loadingListHint') }}</p>
          </div>
          <!--非 pending_ai 页签：未选中任何记录-->
          <div v-else-if="!selectedProcess" class="result-empty">
            <div class="result-empty-icon"><FileOutlined /></div>
            <p>{{ t('dashboard.selectProcessHint', '请在左侧列表中选择一个流程以查看明细') }}</p>
          </div>

          <!--pending_ai 页签：已选未审核 → 操作提示-->
          <template v-else-if="activeTab === 'pending_ai' && selectedProcess && !isResultAsyncRunning(currentResult)">
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

          <!--非 pending_ai 页签：已选但无结果-->
          <template v-else-if="activeTab !== 'pending_ai' && selectedProcess && !currentResult">
            <div class="result-empty">
              <div class="result-empty-icon"><HistoryOutlined /></div>
              <h4>{{ t('dashboard.noHistoryTitle') }}</h4>
              <p>{{ t('dashboard.noHistoryDesc') }}</p>
            </div>
          </template>

          <!--结果展示-->
          <template v-else-if="currentResult">
            <div v-if="isResultAsyncRunning(currentResult)" class="result-async-panel">
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
              <div v-if="currentResult.ai_reasoning || isResultAsyncRunning(currentResult)" class="result-section" style="margin-top: 16px;">
                <h4 class="result-section-title">{{ t('dashboard.aiReasoning') }}</h4>
                <div class="ai-reasoning">
                  <div class="markdown-body" v-html="renderMarkdown(currentResult.ai_reasoning || '')"></div>
                </div>
              </div>
            </div>

            <template v-else>
            <template v-if="currentResult.status === 'failed'">
              <div class="result-banner result-banner--error">
                <WarningOutlined class="result-banner-icon" style="color: var(--color-danger);" />
                <div class="result-banner-info">
                  <div class="result-banner-title" style="color: var(--color-danger);">{{ t('dashboard.auditFailed') }}</div>
                  <div class="result-banner-meta">{{ currentResult.error_message }}</div>
                </div>
              </div>
            </template>

            <div class="result-action-bar">
              <template v-if="isCompletedHistoryTab">
                <div class="history-badge">
                  <HistoryOutlined />
                  {{ t('dashboard.historyReadonly') }}
                </div>
              </template>
              <a-button @click="openAuditChain(currentResult.process_id)">
                <EyeOutlined /> {{ t('dashboard.auditChain') }}
              </a-button>
              <a-button @click="jumpToOA(currentResult.process_id)">
                <ExportOutlined /> {{ t('dashboard.jumpOA') }}
              </a-button>
              <a-button v-if="!isCompletedHistoryTab" @click="handleReAudit">
                <ReloadOutlined /> {{ t('dashboard.reAudit') }}
              </a-button>
            </div>

            <!--已审核流程摘要：标题 + 申请人/部门/类别 + 当前节点-->
            <div v-if="showDashboardProcessSummary" class="dashboard-process-summary">
              <span class="dashboard-process-summary__title">{{ dashboardProcessSummaryTitle }}</span>
              <span class="dashboard-process-summary__meta">{{ dashboardProcessMetaLine2 }}</span>
              <span class="dashboard-process-summary__node">
                <FieldTimeOutlined />
                {{ t('dashboard.currentNode') }}: {{ selectedProcessInfo?.current_node || '—' }}
              </span>
            </div>

            <!--解析失败降级展示-->
            <template v-if="currentResult.parse_error">
              <div class="result-banner result-banner--error">
                <WarningOutlined class="result-banner-icon" style="color: var(--color-danger);" />
                <div class="result-banner-info">
                  <div class="result-banner-title" style="color: var(--color-danger);">{{ t('dashboard.parseErrorTitle') }}</div>
                  <div class="result-banner-meta">{{ currentResult.parse_error }}</div>
                </div>
              </div>
              <div class="result-section">
                <h4 class="result-section-title">{{ t('dashboard.rawContent') }}</h4>
                <div class="ai-reasoning">
                  <pre>{{ currentResult.raw_content }}</pre>
                </div>
              </div>
              <div v-if="currentResult.ai_reasoning" class="result-section">
                <h4 class="result-section-title">{{ t('dashboard.aiReasoning') }}</h4>
                <div class="ai-reasoning">
                  <div class="markdown-body" v-html="renderMarkdown(currentResult.ai_reasoning || '')"></div>
                </div>
              </div>
            </template>

            <!--正常结果展示（failed 仅展示上方错误条 + 操作栏，不再套结论横幅）-->
            <template v-else-if="currentResult.status !== 'failed'">
              <div
                class="result-banner"
                :style="{
                  background: getScoreColorConfig(currentResult.overall_score)?.bg,
                  borderColor: getScoreColorConfig(currentResult.overall_score)?.color,
                }"
              >
                <component
                  :is="recommendationConfig[currentResult.recommendation || 'review']?.icon"
                  class="result-banner-icon"
                  :style="{ color: getScoreColorConfig(currentResult.overall_score)?.color }"
                />
                <div class="result-banner-info">
                  <div class="result-banner-title" :style="{ color: getScoreColorConfig(currentResult.overall_score)?.color }">
                    {{ recommendationConfig[currentResult.recommendation || 'review']?.label }}
                  </div>
                  <div class="result-banner-meta">
                    {{ t('dashboard.overallScore') }} {{ currentResult.overall_score }}{{ t('dashboard.points') }}
                    · {{ t('dashboard.confidence') }} {{ currentResult.confidence }}%
                    · {{ t('dashboard.duration') }} {{ getDurationSec(currentResult.duration_ms) }}s
                  </div>
                </div>
                <div class="result-score" :style="{ color: getScoreColorConfig(currentResult.overall_score)?.color }">
                  {{ currentResult.overall_score }}
                </div>
              </div>

              <!--规则校验-->
              <div v-if="currentResult.rule_results?.length" class="result-section">
                <h4 class="result-section-title">{{ t('dashboard.ruleCheckDetail') }}</h4>
                <div class="rule-checks">
                  <div
                    v-for="(rule, idx) in currentResult.rule_results"
                    :key="idx"
                    class="rule-check-item"
                    :class="{ 'rule-check-item--pass': rule.passed, 'rule-check-item--fail': !rule.passed }"
                  >
                    <div class="rule-check-status">
                      <CheckCircleOutlined v-if="rule.passed" style="color: var(--color-success);" />
                      <CloseCircleOutlined v-else style="color: var(--color-danger);" />
                    </div>
                    <div class="rule-check-content">
                      <div class="rule-check-name">{{ rule.rule_content }}</div>
                      <div class="rule-check-reasoning">{{ rule.reason }}</div>
                    </div>
                  </div>
                </div>
              </div>

              <!--风险点 & 建议-->
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

              <!--AI 推理-->
              <div v-if="currentResult.ai_reasoning" class="result-section">
                <h4 class="result-section-title">{{ t('dashboard.aiReasoning') }}</h4>
                <div class="ai-reasoning">
                  <div class="markdown-body" v-html="renderMarkdown(currentResult.ai_reasoning || '')"></div>
                </div>
              </div>
            </template>
            </template>
          </template>

          <!--空状态-->
          <div v-else class="result-empty">
            <div class="result-empty-icon">
              <ThunderboltOutlined v-if="activeTab === 'pending_ai'" />
              <HistoryOutlined v-else />
            </div>
            <h4>{{ activeTab === 'pending_ai' ? t('dashboard.emptyTodo') : t('dashboard.emptyHistory') }}</h4>
            <p>{{ activeTab === 'pending_ai' ? t('dashboard.emptyTodoDesc') : t('dashboard.emptyHistoryDesc') }}</p>
          </div>
        </div>
      </div>
    </div>

    <!--审核链抽屉-->
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
              <a-spin :spinning="auditChainLoading">
                <div v-if="!auditChainLoading && auditChainData.length === 0" style="padding: 40px; text-align: center;">
                  <a-empty :description="t('dashboard.noAuditRecords')" />
                </div>
                <div v-else class="audit-chain">
                  <div
                    v-for="(item, idx) in auditChainData"
                    :key="item.id"
                    class="chain-node"
                  >
                    <div class="chain-timeline">
                      <div class="chain-dot" :style="{ background: getScoreColorConfig(item.score)?.color }" />
                      <div v-if="idx < auditChainData.length - 1" class="chain-line" />
                    </div>
                    <div class="chain-card">
                      <div class="chain-card-header" @click="toggleChainNode(item.id)" style="cursor: pointer;">
                        <span
                          class="chain-tag"
                          :style="{ color: getScoreColorConfig(item.score)?.color, background: getScoreColorConfig(item.score)?.bg }"
                        >
                          <component :is="recommendationConfig[item.recommendation || 'review']?.icon" />
                          {{ recommendationConfig[item.recommendation || 'review']?.label }}
                        </span>
                        <span class="chain-score">{{ item.score }}{{ t('dashboard.points') }}</span>
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
                        <template v-if="item.audit_result">
                          <!--规则校验-->
                          <template v-if="item.audit_result.rule_results?.length">
                            <div class="chain-section-title">{{ t('dashboard.ruleCheckDetail') }}</div>
                            <div v-for="(rule, ri) in item.audit_result.rule_results" :key="ri" class="chain-rule-item" :class="rule.passed ? 'chain-rule--pass' : 'chain-rule--fail'">
                              <component :is="rule.passed ? CheckCircleOutlined : CloseCircleOutlined" :style="{ color: rule.passed ? 'var(--color-success)' : 'var(--color-danger)' }" />
                              <div>
                                <div class="chain-rule-name">{{ rule.rule_content }}</div>
                                <div class="chain-rule-reasoning">{{ rule.reason }}</div>
                              </div>
                            </div>
                          </template>
                          <!--风险点 & 建议-->
                          <div v-if="item.audit_result.risk_points?.length || item.audit_result.suggestions?.length" class="risk-suggest-row" style="margin-top: 10px;">
                            <div v-if="item.audit_result.risk_points?.length" class="insight-card insight-card--risk">
                              <div class="insight-card-header">
                                <CloseCircleOutlined style="color: var(--color-danger);" />
                                <span>{{ t('dashboard.riskPoints') }}</span>
                              </div>
                              <ul class="insight-card-list">
                                <li v-for="(rp, i) in item.audit_result.risk_points" :key="i">{{ rp }}</li>
                              </ul>
                            </div>
                            <div v-if="item.audit_result.suggestions?.length" class="insight-card insight-card--suggest">
                              <div class="insight-card-header">
                                <InfoCircleOutlined style="color: var(--color-primary);" />
                                <span>{{ t('dashboard.suggestions') }}</span>
                              </div>
                              <ul class="insight-card-list">
                                <li v-for="(sg, i) in item.audit_result.suggestions" :key="i">{{ sg }}</li>
                              </ul>
                            </div>
                          </div>
                          <!--AI 推理：后端链数据里推理在行字段 ai_reasoning，不在 audit_result JSONB 内 -->
                          <div v-if="chainItemAiReasoning(item)" class="chain-section-title" style="margin-top: 10px;">{{ t('dashboard.aiReasoning') }}</div>
                          <div v-if="chainItemAiReasoning(item)" class="chain-reasoning">
                            <div class="markdown-body" v-html="renderMarkdown(chainItemAiReasoning(item) || '')"></div>
                          </div>
                          <!--解析错误-->
                          <div v-if="item.audit_result.parse_error" class="chain-parse-error">
                            <WarningOutlined style="color: var(--color-danger);" />
                            <span>{{ t('dashboard.parseErrorTitle') }}: {{ item.audit_result.parse_error }}</span>
                          </div>
                        </template>
                        <div v-else class="chain-no-detail">{{ t('dashboard.noHistoryDesc') }}</div>
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
.dashboard { animation: fadeIn 0.3s ease-out; }
@keyframes fadeIn { from { opacity: 0; transform: translateY(8px); } to { opacity: 1; transform: translateY(0); } }

.page-header { margin-bottom: 24px; }
.page-title { font-size: 24px; font-weight: 700; color: var(--color-text-primary); margin: 0; letter-spacing: -0.02em; }
.page-subtitle { font-size: 14px; color: var(--color-text-tertiary); margin: 4px 0 0; }

/*统计行 — 3 列*/
.stats-row { display: grid; grid-template-columns: repeat(3, 1fr); gap: 16px; margin-bottom: 24px; }
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
.stat-card--info .stat-card-icon { background: var(--color-info-bg); color: var(--color-info); }
.stat-card-info { display: flex; flex-direction: column; }
.stat-card-value { font-size: 28px; font-weight: 700; color: var(--color-text-primary); line-height: 1.2; }
.stat-card-label { font-size: 13px; color: var(--color-text-tertiary); margin-top: 2px; }

/*仪表板网格*/
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
.panel-header-row { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 12px; }
.panel-header-controls { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; }
.dashboard-audit-date-label { font-size: 13px; color: var(--color-text-secondary); white-space: nowrap; }
.dashboard-audit-range-picker { min-width: 240px; }

/*筛选条*/
.filter-toggle-btn { position: relative; }
.filter-toggle-btn--active { color: var(--color-primary); border-color: var(--color-primary); }
.filter-active-dot {
  display: inline-block; width: 6px; height: 6px; border-radius: 50%;
  background: var(--color-primary); margin-left: 4px; vertical-align: middle;
}
.filter-bar { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; padding: 10px 0 0; }
.slide-enter-active, .slide-leave-active { transition: all 0.2s ease; }
.slide-enter-from, .slide-leave-to { opacity: 0; transform: translateY(-8px); }

/*列表*/
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

/*批量审核工具栏*/
.batch-toolbar { display: flex; align-items: center; justify-content: space-between; padding: 6px 0; gap: 8px; }
.batch-toolbar-left { display: flex; align-items: center; gap: 12px; }
.batch-toolbar-right { display: flex; align-items: center; gap: 8px; }
.batch-limit-hint { font-size: 11px; color: var(--color-text-quaternary); }
.batch-progress-hint { font-size: 12px; font-weight: 600; color: var(--color-primary); animation: auditPulse 1.5s ease-in-out infinite; }
@keyframes auditPulse { 0%, 100% { opacity: 0.6; } 50% { opacity: 1; } }
.batch-audit-btn { flex-shrink: 0; }

/*分页*/
.pagination-wrapper { padding: 12px 20px; border-top: 1px solid var(--color-border-light); display: flex; justify-content: center; }

/*结果面板*/
.result-content { padding: 20px; }

/*操作提示*/
.action-prompt { text-align: center; padding: 40px 20px; }
.action-prompt-info h4 { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin: 0 0 8px; }
.action-prompt-info p { font-size: 13px; color: var(--color-text-tertiary); margin: 0 0 24px; }
.action-prompt-buttons { display: flex; gap: 12px; justify-content: center; }

/*操作栏*/
.result-action-bar { display: flex; gap: 8px; margin-bottom: 16px; align-items: center; flex-wrap: wrap; }

.dashboard-process-summary {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-bottom: 16px;
  font-size: 12px;
  color: var(--color-text-tertiary);
  line-height: 1.5;
}
.dashboard-process-summary__title {
  font-size: 15px;
  font-weight: 600;
  color: var(--color-text-primary);
  word-break: break-word;
}
.dashboard-process-summary__meta { font-size: 12px; }
.dashboard-process-summary__node {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
}
.dashboard-process-summary__node .anticon {
  margin-right: 2px;
}
.history-badge {
  display: flex; align-items: center; gap: 6px; font-size: 12px; font-weight: 600;
  padding: 4px 12px; border-radius: var(--radius-full);
  background: var(--color-bg-hover); color: var(--color-text-tertiary); margin-right: auto;
}

/*加载中*/
.result-loading { display: flex; flex-direction: column; align-items: center; padding: 40px 20px; gap: 20px; }
.loading-process-info { text-align: center; }
.loading-process-title { font-size: 15px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 4px; }
.loading-process-meta { font-size: 13px; color: var(--color-text-tertiary); }
.audit-progress { display: flex; flex-direction: column; gap: 12px; width: 100%; max-width: 400px; }
.audit-phase {
  display: flex; align-items: flex-start; gap: 14px; padding: 14px 16px;
  border-radius: var(--radius-md); border: 1px solid var(--color-border-light);
  background: var(--color-bg-page); transition: all 0.3s ease;
}
.audit-phase--active { border-color: var(--color-primary); background: var(--color-primary-bg); box-shadow: 0 2px 8px rgba(79, 70, 229, 0.1); }
.audit-phase--done { border-color: var(--color-success); background: var(--color-success-bg); }
.audit-phase--pending { opacity: 0.5; }
.audit-phase-dot {
  width: 28px; height: 28px; border-radius: 50%; display: flex; align-items: center;
  justify-content: center; font-size: 16px; flex-shrink: 0; background: var(--color-bg-hover);
}
.audit-phase--active .audit-phase-dot { background: var(--color-primary-bg); color: var(--color-primary); }
.audit-phase--done .audit-phase-dot { background: var(--color-success-bg); color: var(--color-success); }
.phase-pending-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--color-border); }
.audit-phase-info { flex: 1; }
.audit-phase-title { font-size: 14px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 2px; }
.audit-phase-desc { font-size: 12px; color: var(--color-text-tertiary); }

.result-async-panel { padding: 8px 0 16px; }
.async-progress-steps { display: flex; flex-direction: column; gap: 10px; margin-top: 12px; }
.async-step-row { display: flex; align-items: center; gap: 10px; font-size: 13px; color: var(--color-text-secondary); }
.async-step-pending-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--color-border); display: inline-block; flex-shrink: 0; }

/*结果横幅*/
.result-banner {
  display: flex; align-items: center; padding: 16px 20px;
  border-radius: var(--radius-lg); border-left: 4px solid; margin-bottom: 24px; gap: 14px;
}
.result-banner--error {
  background: var(--color-danger-bg); border-color: var(--color-danger);
}
.result-banner-icon { font-size: 28px; flex-shrink: 0; }
.result-banner-info { flex: 1; }
.result-banner-title { font-size: 16px; font-weight: 700; }
.result-banner-meta { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }
.result-score { font-size: 36px; font-weight: 800; line-height: 1; }

/*规则校验*/
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
.rule-check-name { font-size: 14px; font-weight: 500; color: var(--color-text-primary); margin-bottom: 4px; }
.rule-check-reasoning { font-size: 13px; color: var(--color-text-secondary); line-height: 1.5; }

/*风险 + 建议*/
.risk-suggest-row { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-bottom: 24px; }
.risk-suggest-row:has(.insight-card:only-child) { grid-template-columns: 1fr; }
.insight-card { border-radius: var(--radius-md); padding: 16px; border: 1px solid var(--color-border-light); }
.insight-card--risk { background: linear-gradient(135deg, rgba(239, 68, 68, 0.04), rgba(239, 68, 68, 0.01)); border-color: rgba(239, 68, 68, 0.15); }
.insight-card--suggest { background: linear-gradient(135deg, rgba(79, 70, 229, 0.04), rgba(79, 70, 229, 0.01)); border-color: rgba(79, 70, 229, 0.15); }
.insight-card-header { display: flex; align-items: center; gap: 8px; font-size: 13px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 10px; }
.insight-card-list { margin: 0; padding-left: 18px; display: flex; flex-direction: column; gap: 6px; }
.insight-card-list li { font-size: 13px; line-height: 1.6; color: var(--color-text-secondary); }
.insight-card--risk .insight-card-list li { color: var(--color-danger); }

/*AI 推理*/
.ai-reasoning { background: var(--color-bg-page); border-radius: var(--radius-md); padding: 16px; border: 1px solid var(--color-border-light); }
.ai-reasoning pre { white-space: pre-wrap; word-break: break-word; font-family: var(--font-sans); font-size: 13px; line-height: 1.7; color: var(--color-text-secondary); margin: 0; }

/*空状态*/
.result-empty { text-align: center; padding: 60px 20px; }
.result-empty-icon {
  width: 64px; height: 64px; border-radius: 50%; background: var(--color-primary-bg);
  color: var(--color-primary); font-size: 28px; display: flex; align-items: center;
  justify-content: center; margin: 0 auto 16px;
}
.result-empty h4 { font-size: 16px; font-weight: 600; color: var(--color-text-primary); margin: 0 0 8px; }
.result-empty p { font-size: 13px; color: var(--color-text-tertiary); margin: 0 auto; max-width: 280px; }
.result-empty--loading {
  display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 16px;
  padding: 48px 20px; min-height: 200px;
}
.result-empty--loading p { margin: 0; }

/*抽屉*/
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

/*审核链时间线*/
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
.chain-parse-error {
  display: flex; align-items: center; gap: 8px; padding: 8px 12px;
  border-radius: var(--radius-sm); background: var(--color-danger-bg);
  font-size: 12px; color: var(--color-danger);
}

.drawer-enter-active { transition: opacity 0.2s ease; }
.drawer-enter-active .drawer-panel { transition: transform 0.3s cubic-bezier(0.16,1,0.3,1); }
.drawer-leave-active { transition: opacity 0.2s ease 0.1s; }
.drawer-leave-active .drawer-panel { transition: transform 0.2s ease; }
.drawer-enter-from { opacity: 0; }
.drawer-enter-from .drawer-panel { transform: translateX(100%); }
.drawer-leave-to { opacity: 0; }
.drawer-leave-to .drawer-panel { transform: translateX(100%); }

@media (max-width: 1024px) {
  .dashboard-grid { grid-template-columns: 1fr; }
  .stats-row { grid-template-columns: repeat(3, 1fr); }
}
@media (max-width: 768px) {
  .stats-row { grid-template-columns: 1fr; gap: 12px; }
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
  .result-banner { flex-wrap: wrap; padding: 12px 14px; }
  .result-score { font-size: 28px; }
}
</style>
