<script setup lang="ts">
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  DownOutlined,
  FileTextOutlined,
  LoadingOutlined,
  ReloadOutlined,
  ThunderboltOutlined,
  UpOutlined,
  WarningOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import type { EmbedProcessSummary } from '~/types/embed'
import type { SummaryBlockResult, SummaryResult } from '~/types/process-summary'
import type { EmbedSummaryContextResponse } from '~/composables/useEmbedSummaryApi'
import { waitForParentEmbedContext } from '~/composables/useEmbedParent'

definePageMeta({ layout: 'embed' })

const { getSummaryContext, executeSummaryEmbed, waitSummaryJob } = useEmbedSummaryApi()
const { setupEmbedSession } = useEmbedSession()

const processId = ref('')
const pageLoading = ref(true)
const waitingParent = ref(true)
const summarizing = ref(false)
const pageError = ref('')
const context = ref<EmbedSummaryContextResponse | null>(null)
const currentResult = ref<SummaryResult | null>(null)
const streamingBlocks = ref<{ block_id: string; title: string; content: string }[]>([])
const eventSourceStream = ref<EventSource | null>(null)
const streamJobId = ref('')
const collapsedBlockIds = ref<Set<string>>(new Set())

const processInfo = computed<EmbedProcessSummary | null>(() => context.value?.process ?? null)
const isRunning = computed(() => summarizing.value || ['pending', 'assembling', 'reasoning', 'extracting'].includes(currentResult.value?.status || ''))

const SUMMARY_PROGRESS_STEPS = [
  { key: 'pending', label: '排队中' },
  { key: 'assembling', label: '解析流程数据/附件' },
  { key: 'prompt', label: '组装总结提示词' },
  { key: 'reasoning', label: 'AI 生成总结' },
  { key: 'extracting', label: '解析总结结构' },
] as const

const progressStatusOrder: Record<string, number> = {
  pending: 0,
  assembling: 1,
  reasoning: 3,
  extracting: 4,
  completed: 5,
  failed: 5,
}

const summaryProgressSteps = computed(() => {
  const status = currentResult.value?.status || 'pending'
  const order = progressStatusOrder[status] ?? 0
  return SUMMARY_PROGRESS_STEPS.filter(s => s.key !== 'pending').map((step, index) => {
    const stepOrder = index + 1
    return {
      ...step,
      done: order > stepOrder || status === 'completed',
      current: order === stepOrder || (status === 'reasoning' && step.key === 'reasoning'),
      failed: status === 'failed' && order === stepOrder,
    }
  })
})

const headerStatus = computed(() => {
  if (pageLoading.value || waitingParent.value) {
    return { label: '流程总结', color: 'var(--color-text-primary)', bg: 'transparent', icon: FileTextOutlined, spin: false }
  }
  if (isRunning.value) {
    return { label: '正在总结', color: 'var(--color-primary)', bg: 'var(--color-primary-bg)', icon: LoadingOutlined, spin: true }
  }
  if (currentResult.value?.status === 'failed') {
    return { label: '总结失败', color: 'var(--color-danger)', bg: 'var(--color-danger-bg)', icon: CloseCircleOutlined, spin: false }
  }
  if (currentResult.value?.blocks?.length) {
    return { label: '总结完成', color: 'var(--color-success)', bg: 'var(--color-success-bg)', icon: CheckCircleOutlined, spin: false }
  }
  return { label: '待总结', color: 'var(--color-text-tertiary)', bg: 'var(--color-bg-hover)', icon: ThunderboltOutlined, spin: false }
})

const processMetaRows = computed(() => {
  const p = processInfo.value
  if (!p) return []
  return [
    [p.applicant, p.department].filter(Boolean).join(' · '),
    [p.process_type_label || p.process_type, p.current_node].filter(Boolean).join(' · '),
  ].filter(Boolean)
})

const formatLastSummaryAt = (iso?: string) => iso ? formatDateTimeInAppZone(iso) : '-'
const getDurationSec = (ms?: number) => ((ms || 0) / 1000).toFixed(1)

const processStat = computed(() => {
  if (currentResult.value?.duration_ms) {
    const sec = currentResult.value.duration_ms / 1000
    const tone = sec >= 120 ? 'danger' : sec >= 60 ? 'warning' : 'success'
    return { text: `耗时 ${getDurationSec(currentResult.value.duration_ms)} 秒`, tone }
  }
  if (isRunning.value) {
    return { text: '执行中', tone: 'running' }
  }
  return null
})

const visibleStreamingBlocks = computed(() => streamingBlocks.value.filter(block => block.content))

function createPendingResult(): SummaryResult {
  return {
    process_id: processId.value,
    status: 'pending',
    blocks: [],
    duration_ms: 0,
  }
}

function mergeSummaryProgress(st: SummaryResult) {
  currentResult.value = { ...currentResult.value, ...st }
}

function getBlockKey(block: SummaryBlockResult, idx: number) {
  return block.block_id || block.title || `block-${idx}`
}

function isBlockCollapsed(block: SummaryBlockResult, idx: number) {
  return collapsedBlockIds.value.has(getBlockKey(block, idx))
}

function toggleBlock(block: SummaryBlockResult, idx: number) {
  const key = getBlockKey(block, idx)
  const next = new Set(collapsedBlockIds.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  collapsedBlockIds.value = next
}

function resetStreamingBlocks() {
  streamingBlocks.value = []
}

function appendStreamingChunk(raw: string) {
  if (!raw) return
  let payload: { block_id?: string; title?: string; chunk?: string }
  try {
    payload = JSON.parse(raw)
  } catch {
    payload = { block_id: 'legacy', title: '模型回复', chunk: raw }
  }
  const chunk = payload.chunk || ''
  if (!chunk) return
  const key = payload.block_id || payload.title || 'legacy'
  const title = payload.title || '模型回复'
  const next = [...streamingBlocks.value]
  const existing = next.find(block => block.block_id === key)
  if (existing) {
    existing.content += chunk
  } else {
    next.push({ block_id: key, title, content: chunk })
  }
  streamingBlocks.value = next
}

function disconnectStream() {
  if (eventSourceStream.value) {
    eventSourceStream.value.close()
    eventSourceStream.value = null
  }
  streamJobId.value = ''
}

function startSSE(jobId?: string) {
  if (!process.client || !jobId || streamJobId.value === jobId) return
  disconnectStream()
  streamJobId.value = jobId
  resetStreamingBlocks()
  const { appendEmbedTokenQuery } = useEmbedAuth()
  eventSourceStream.value = new EventSource(
    appendEmbedTokenQuery(`/api/embed/summary/stream/${encodeURIComponent(jobId)}`),
  )
  eventSourceStream.value.onmessage = (event) => {
    appendStreamingChunk(event.data || '')
  }
  eventSourceStream.value.onerror = () => disconnectStream()
}

async function runSummary(trigger: 'summary_embed_auto' | 'summary_embed_manual') {
  if (!processId.value || summarizing.value) return
  summarizing.value = true
  currentResult.value = createPendingResult()
  resetStreamingBlocks()
  collapsedBlockIds.value = new Set()
  try {
    const result = await executeSummaryEmbed(
      {
        process_id: processId.value,
        process_type: processInfo.value?.process_type,
        title: processInfo.value?.title,
        trigger_source: trigger,
      },
      (st) => {
        mergeSummaryProgress(st)
        startSSE(st.id)
      },
    )
    mergeSummaryProgress(result)
    await refreshContext(false)
    if (trigger === 'summary_embed_manual') message.success('总结已刷新')
  } catch (e: any) {
    message.error(e?.message || '总结失败')
    currentResult.value = null
    await refreshContext(false)
  } finally {
    summarizing.value = false
    disconnectStream()
  }
}

async function refreshContext(autoRun = true) {
  if (!processId.value) return
  try {
    const resp = await getSummaryContext(processId.value)
    context.value = resp
    if (resp.summary_result && !summarizing.value) {
      currentResult.value = resp.summary_result
    }
    if (autoRun && resp.should_auto_summary && !summarizing.value && !resp.running_job_id) {
      await runSummary('summary_embed_auto')
    } else if (resp.running_job_id && !summarizing.value) {
      summarizing.value = true
      currentResult.value = resp.summary_result ? ({ ...createPendingResult(), ...resp.summary_result }) : createPendingResult()
      startSSE(resp.running_job_id)
      try {
        const result = await waitSummaryJob(resp.running_job_id, mergeSummaryProgress)
        mergeSummaryProgress(result)
        await refreshContext(false)
      } finally {
        summarizing.value = false
        disconnectStream()
      }
    }
  } catch (e: any) {
    pageError.value = e?.message || '加载总结上下文失败'
  }
}

async function bootstrap() {
  pageLoading.value = true
  pageError.value = ''
  context.value = null
  currentResult.value = null
  try {
    if (!processId.value) {
      pageError.value = '未获取到 OA 流程编号'
      return
    }
    await refreshContext(false)
  } finally {
    pageLoading.value = false
    waitingParent.value = false
  }
  const ctx = context.value
  if (ctx?.supported && ctx.should_auto_summary && !summarizing.value && !ctx.running_job_id) {
    await runSummary('summary_embed_auto')
  } else if (ctx?.running_job_id && !summarizing.value) {
    await refreshContext(true)
  }
}

onMounted(async () => {
  waitingParent.value = true
  const parentCtx = await waitForParentEmbedContext()
  processId.value = parentCtx.requestId
  waitingParent.value = false
  if (!parentCtx.embedToken) {
    pageError.value = '未获取到嵌入访问令牌，请检查 OA 父页面配置'
    pageLoading.value = false
    return
  }
  try {
    await setupEmbedSession(parentCtx.embedToken)
  } catch (e: any) {
    pageError.value = e?.message || '未获取到嵌入访问令牌，请检查 OA 父页面配置'
    pageLoading.value = false
    return
  }
  await bootstrap()
})

onBeforeUnmount(() => disconnectStream())
</script>

<template>
  <div class="embed-summary">
    <div class="embed-header">
      <h2 class="embed-title" :style="{ color: headerStatus.color }">
        <span class="embed-title-badge" :style="{ background: headerStatus.bg, color: headerStatus.color }">
          <component :is="headerStatus.icon" :spin="headerStatus.spin" />
        </span>
        {{ headerStatus.label }}
      </h2>
      <div v-if="!isRunning && context?.last_summary_at" class="embed-subline">
        最近总结：{{ formatLastSummaryAt(context.last_summary_at) }}
        <a-tag v-if="context.stale" color="warning" style="margin-left: 8px;">已变化</a-tag>
      </div>
      <p v-else-if="isRunning" class="embed-subline embed-subline--active">AI 正在整理流程内容</p>
    </div>

    <div v-if="pageLoading || waitingParent" class="embed-page-loading">
      <a-spin size="large" :tip="waitingParent ? '等待 OA 流程编号' : undefined" />
    </div>

    <template v-else>
      <a-alert v-if="pageError" type="error" :message="pageError" show-icon style="margin-bottom: 16px;" />

      <a-result
        v-else-if="context && !context.supported"
        status="warning"
        title="暂不支持总结"
        :sub-title="context.message"
      >
        <template #extra>
          <div v-if="context.process" class="unsupported-process">
            <div class="unsupported-process__title">{{ context.process.title }}</div>
            <div class="unsupported-process__meta">{{ context.process.process_type }}</div>
          </div>
        </template>
      </a-result>

      <template v-else-if="context?.supported">
        <div v-if="processInfo" class="process-card">
          <h3 class="process-title" :title="processInfo.title">{{ processInfo.title }}</h3>
          <div class="process-card-main">
            <div class="process-copy">
              <div
                v-for="row in processMetaRows"
                :key="row"
                class="process-meta-row"
              >
                <span
                  class="process-meta-chip"
                  :title="row"
                >
                  {{ row }}
                </span>
              </div>
            </div>
            <div class="process-actions">
              <div
                v-if="processStat"
                class="process-stat"
                :class="`process-stat--${processStat.tone}`"
              >
                {{ processStat.text }}
              </div>
              <a-button
                class="process-action"
                type="text"
                size="small"
                :disabled="summarizing"
                @click="runSummary('summary_embed_manual')"
              >
                <LoadingOutlined v-if="summarizing" spin />
                <ReloadOutlined v-else />
                <span>重新总结</span>
              </a-button>
            </div>
          </div>
        </div>

        <div v-if="isRunning" class="summary-loading">
          <a-spin size="large" />
          <p class="summary-loading-title">正在生成总结</p>
          <div class="async-progress-steps">
            <div
              v-for="s in summaryProgressSteps"
              :key="s.key"
              class="async-step-row"
              :class="{ 'async-step-row--current': s.current }"
            >
              <CheckCircleOutlined v-if="s.done" class="async-step-icon async-step-icon--done" />
              <LoadingOutlined v-else-if="s.current" spin class="async-step-icon async-step-icon--current" />
              <CloseCircleOutlined v-else-if="s.failed" class="async-step-icon async-step-icon--fail" />
              <span v-else class="async-step-pending-dot" />
              <span class="async-step-label">{{ s.label }}</span>
            </div>
          </div>
          <div v-if="visibleStreamingBlocks.length" class="summary-stream-blocks">
            <section v-for="block in visibleStreamingBlocks" :key="block.block_id" class="summary-stream-card">
              <div class="summary-stream-card__header">
                <span>{{ block.title }}</span>
                <em>生成中</em>
              </div>
              <AiMarkdownStream
                :text="block.content"
                max-height="220px"
              />
            </section>
          </div>
        </div>

        <template v-else-if="currentResult">
          <div v-if="currentResult.status === 'failed'" class="result-error">
            <WarningOutlined />
            <div>
              <strong>总结失败</strong>
              <p>{{ currentResult.error_message }}</p>
            </div>
          </div>

            <div v-else>
              <div v-if="currentResult.parse_error" class="summary-meta">
                <a-tag v-if="currentResult.parse_error" color="warning">已使用兜底解析</a-tag>
              </div>

            <div v-if="currentResult.blocks?.length" class="summary-blocks">
              <section v-for="(block, idx) in currentResult.blocks" :key="getBlockKey(block, idx)" class="summary-card">
                <button type="button" class="summary-card-header" @click="toggleBlock(block, idx)">
                  <span class="summary-card-title">{{ block.title }}</span>
                  <span class="summary-card-tools">
                    <span v-if="block.duration_ms" class="summary-card-duration">耗时 {{ getDurationSec(block.duration_ms) }} 秒</span>
                    <DownOutlined v-if="isBlockCollapsed(block, idx)" />
                    <UpOutlined v-else />
                  </span>
                </button>
                <div v-show="!isBlockCollapsed(block, idx)" class="summary-card-body">
                  <AiMarkdownStream
                    :text="block.content"
                    max-height="300px"
                  />
                  <ul v-if="block.points?.length" class="summary-points">
                    <li v-for="(point, pointIdx) in block.points" :key="pointIdx">{{ point }}</li>
                  </ul>
                </div>
              </section>
            </div>
          </div>
        </template>

        <div v-else class="result-empty">
          <div class="result-empty-icon"><ThunderboltOutlined /></div>
          <h4>暂无总结</h4>
          <p>点击下方按钮生成流程总结</p>
          <a-button type="primary" @click="runSummary('summary_embed_manual')">
            <ThunderboltOutlined /> 开始总结
          </a-button>
        </div>
      </template>
    </template>
  </div>
</template>

<style scoped>
.embed-summary { max-width: 720px; margin: 0 auto; min-height: 100vh; padding: 12px 12px 28px; }
.embed-header {
  margin-bottom: 14px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--color-border-light);
}
.embed-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 17px;
  font-weight: 700;
  margin: 0 0 4px;
  letter-spacing: 0;
}
.embed-title-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 8px;
  font-size: 16px;
  flex-shrink: 0;
}
.embed-subline { font-size: 12px; color: var(--color-text-secondary); margin: 0; line-height: 1.4; }
.embed-subline--active { color: var(--color-primary); font-size: 12px; }
.embed-page-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: calc(100vh - 120px);
  padding: 48px 20px;
}
.unsupported-process { text-align: left; margin-top: 8px; }
.unsupported-process__title { font-weight: 600; color: var(--color-text-primary); }
.unsupported-process__meta { font-size: 13px; color: var(--color-text-tertiary); margin-top: 4px; }
.process-card {
  margin-bottom: 14px;
  border-radius: var(--radius-lg);
  background: var(--color-bg-card);
  border: 1px solid var(--color-border-light);
  padding: 14px 16px;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}
.process-card-main {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 104px;
  gap: 16px;
  align-items: center;
}
.process-copy { min-width: 0; }
.process-title {
  margin: 0 0 10px;
  font-size: 15px;
  font-weight: 700;
  line-height: 1.5;
  color: var(--color-text-primary);
  overflow-wrap: anywhere;
  white-space: normal;
}
.process-meta-row {
  display: flex;
  align-items: center;
  min-width: 0;
}
.process-meta-row + .process-meta-row {
  margin-top: 7px;
}
.process-meta-chip {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  max-width: 100%;
  min-width: 0;
  height: 26px;
  padding: 0 10px;
  border-radius: var(--radius-full);
  border: 1px solid color-mix(in srgb, var(--color-border-light) 82%, var(--color-primary) 18%);
  background: color-mix(in srgb, var(--color-bg-hover) 78%, var(--color-bg-card));
  color: var(--color-text-secondary);
  font-size: 13px;
  font-weight: 600;
  line-height: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.process-meta-row:first-of-type .process-meta-chip:first-child {
  color: var(--color-text-primary);
  background: color-mix(in srgb, var(--color-primary-bg) 54%, var(--color-bg-card));
  border-color: color-mix(in srgb, var(--color-primary) 22%, var(--color-border-light));
}
.process-actions {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  justify-content: center;
  gap: 8px;
  flex-shrink: 0;
  min-width: 0;
  padding-left: 12px;
  border-left: 1px solid var(--color-border-light);
}
.process-stat {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 28px;
  padding: 0 7px;
  border-radius: var(--radius-md);
  background: var(--color-bg-hover);
  border: 1px solid var(--color-border-light);
  text-align: center;
  color: var(--color-text-secondary);
  font-size: 12px;
  font-weight: 700;
  white-space: nowrap;
}
.process-stat--success {
  color: var(--color-success);
  background: var(--color-success-bg);
  border-color: color-mix(in srgb, var(--color-success) 24%, transparent);
}
.process-stat--warning {
  color: var(--color-warning);
  background: var(--color-warning-bg);
  border-color: color-mix(in srgb, var(--color-warning) 28%, transparent);
}
.process-stat--danger {
  color: var(--color-danger);
  background: var(--color-danger-bg);
  border-color: color-mix(in srgb, var(--color-danger) 24%, transparent);
}
.process-stat--running {
  color: var(--color-primary);
  background: var(--color-primary-bg);
  border-color: color-mix(in srgb, var(--color-primary) 18%, transparent);
}
.process-action.ant-btn {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  width: 100%;
  height: 30px !important;
  padding: 0 8px !important;
  margin: 0;
  border-radius: var(--radius-md) !important;
  color: var(--color-primary) !important;
  background: var(--color-primary-bg) !important;
  font-size: 12px !important;
  line-height: 28px !important;
  white-space: nowrap;
}
.process-action.ant-btn:hover:not(:disabled) {
  color: var(--color-primary) !important;
  background: color-mix(in srgb, var(--color-primary-bg) 70%, var(--color-bg-hover)) !important;
}
.process-action.ant-btn:disabled {
  color: color-mix(in srgb, var(--color-primary) 70%, var(--color-text-tertiary)) !important;
  background: var(--color-primary-bg) !important;
}
.process-action :deep(.anticon) {
  width: 14px;
  font-size: 13px;
}
@media (max-width: 420px) {
  .process-actions {
    gap: 6px;
    padding-left: 8px;
  }
  .process-stat {
    padding: 0 5px;
  }
}
.summary-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-start;
  min-height: calc(100vh - 220px);
  padding: 28px 0;
  text-align: center;
}
.summary-loading-title {
  margin: 14px 0 18px;
  color: var(--color-text-secondary);
  font-size: 14px;
  font-weight: 500;
}
.async-progress-steps {
  display: flex;
  flex-direction: column;
  gap: 10px;
  width: 100%;
  max-width: 340px;
  margin-bottom: 18px;
}
.async-step-row {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
  color: var(--color-text-tertiary);
  padding: 8px 12px;
  border-radius: var(--radius-md);
  transition: background 0.2s ease, color 0.2s ease;
}
.async-step-row--current {
  background: var(--color-primary-bg);
  color: var(--color-text-primary);
  font-weight: 500;
}
.async-step-icon { font-size: 16px; flex-shrink: 0; }
.async-step-icon--done { color: var(--color-success); }
.async-step-icon--current { color: var(--color-primary); }
.async-step-icon--fail { color: var(--color-danger); }
.async-step-label { flex: 1; text-align: left; }
.async-step-pending-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-border);
  display: inline-block;
  flex-shrink: 0;
}
.summary-stream-blocks {
  width: 100%;
  text-align: left;
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.summary-stream-card {
  width: 100%;
}
.summary-stream-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  margin-bottom: 6px;
  color: var(--color-text-primary);
  font-size: 13px;
  font-weight: 600;
}
.summary-stream-card__header em {
  font-style: normal;
  padding: 2px 7px;
  border-radius: var(--radius-full);
  background: var(--color-primary-bg);
  color: var(--color-primary);
  font-size: 10px;
  font-weight: 600;
}
.summary-meta {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  margin-bottom: 10px;
  font-size: 12px;
  color: var(--color-text-tertiary);
}
.summary-blocks { display: flex; flex-direction: column; gap: 12px; }
.summary-card {
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-lg);
  background: var(--color-bg-card);
  overflow: hidden;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}
.summary-card-header {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 46px;
  padding: 11px 14px;
  border: 0;
  border-bottom: 1px solid var(--color-border-light);
  background: var(--color-bg-card);
  color: var(--color-text-primary);
  cursor: pointer;
  text-align: left;
}
.summary-card-header:hover {
  background: color-mix(in srgb, var(--color-bg-hover) 45%, var(--color-bg-card));
}
.summary-card-title {
  min-width: 0;
  flex: 1;
  font-size: 14px;
  font-weight: 700;
  line-height: 1.45;
  overflow-wrap: anywhere;
}
.summary-card-tools {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  color: var(--color-text-tertiary);
}
.summary-card-duration {
  padding: 2px 8px;
  border-radius: var(--radius-full);
  background: var(--color-bg-hover);
  color: var(--color-text-secondary);
  font-size: 11px;
  font-weight: 500;
  white-space: nowrap;
}
.summary-card-body {
  padding: 12px 14px 14px;
}
.summary-content { font-size: 13px; line-height: 1.7; color: var(--color-text-secondary); }
.summary-points {
  margin: 10px 0 0;
  padding-left: 18px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.summary-points li { font-size: 13px; line-height: 1.6; color: var(--color-text-secondary); }
.result-error {
  display: flex;
  gap: 12px;
  padding: 14px;
  border-radius: 12px;
  border: 1px solid var(--color-danger);
  background: var(--color-danger-bg);
  color: var(--color-danger);
}
.result-error p { margin: 4px 0 0; color: var(--color-text-secondary); }
.result-empty { text-align: center; padding: 40px 20px; }
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
.result-empty h4 { font-size: 16px; font-weight: 600; margin: 0 0 8px; }
.result-empty p { font-size: 13px; color: var(--color-text-tertiary); margin: 0 auto 16px; max-width: 280px; }
</style>
