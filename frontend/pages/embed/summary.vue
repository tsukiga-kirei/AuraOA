<script setup lang="ts">
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  DownOutlined,
  FieldTimeOutlined,
  FileTextOutlined,
  LoadingOutlined,
  ReloadOutlined,
  ThunderboltOutlined,
  UpOutlined,
  WarningOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { marked } from 'marked'
import type { EmbedProcessSummary } from '~/types/embed'
import type { SummaryResult } from '~/types/process-summary'
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
const streamingText = ref('')
const showRaw = ref(false)
const eventSourceStream = ref<EventSource | null>(null)
const streamJobId = ref('')

const processInfo = computed<EmbedProcessSummary | null>(() => context.value?.process ?? null)
const isRunning = computed(() => summarizing.value || ['pending', 'assembling', 'reasoning', 'extracting'].includes(currentResult.value?.status || ''))

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

const processMetaLine = computed(() => {
  const p = processInfo.value
  if (!p) return ''
  return [p.applicant, p.department, p.process_type_label || p.process_type].filter(Boolean).join(' · ')
})

const formatLastSummaryAt = (iso?: string) => iso ? formatDateTimeInAppZone(iso) : '-'
const getDurationSec = (ms?: number) => ((ms || 0) / 1000).toFixed(1)

const renderMarkdown = (text: string) => {
  try {
    return marked.parse(text || '') as string
  } catch {
    return text
  }
}

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
  streamingText.value = ''
  eventSourceStream.value = new EventSource(`/api/embed/summary/stream/${encodeURIComponent(jobId)}`)
  eventSourceStream.value.onmessage = (event) => {
    streamingText.value += event.data || ''
  }
  eventSourceStream.value.onerror = () => disconnectStream()
}

async function runSummary(trigger: 'summary_embed_auto' | 'summary_embed_manual') {
  if (!processId.value || summarizing.value) return
  summarizing.value = true
  currentResult.value = createPendingResult()
  streamingText.value = ''
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
          <p v-if="processMetaLine" class="process-meta">{{ processMetaLine }}</p>
          <div class="process-footer">
            <div v-if="processInfo.current_node" class="process-node">
              <FieldTimeOutlined />
              <span>{{ processInfo.current_node }}</span>
            </div>
            <a-button type="text" size="small" :loading="summarizing" @click="runSummary('summary_embed_manual')">
              <ReloadOutlined />
              <span>重新总结</span>
            </a-button>
          </div>
        </div>

        <div v-if="isRunning" class="summary-loading">
          <a-spin size="large" />
          <p>正在生成总结</p>
          <div v-if="streamingText" class="raw-stream">
            <button type="button" class="raw-toggle" @click="showRaw = !showRaw">
              <span>模型输出</span>
              <DownOutlined v-if="!showRaw" />
              <UpOutlined v-else />
            </button>
            <pre v-show="showRaw">{{ streamingText }}</pre>
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
            <div class="summary-meta">
              <span v-if="currentResult.duration_ms">耗时 {{ getDurationSec(currentResult.duration_ms) }}s</span>
              <a-tag v-if="currentResult.parse_error" color="warning">已使用兜底解析</a-tag>
            </div>

            <div v-if="currentResult.blocks?.length" class="summary-blocks">
              <section v-for="block in currentResult.blocks" :key="block.block_id || block.title" class="summary-card">
                <h4>{{ block.title }}</h4>
                <div class="markdown-body summary-content" v-html="renderMarkdown(block.content)" />
                <ul v-if="block.points?.length" class="summary-points">
                  <li v-for="(point, idx) in block.points" :key="idx">{{ point }}</li>
                </ul>
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
.embed-summary { max-width: 720px; margin: 0 auto; min-height: 100vh; }
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
.embed-subline { font-size: 11px; color: var(--color-text-quaternary); margin: 0; line-height: 1.4; }
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
  border-radius: 12px;
  background: var(--color-bg-card);
  border: 1px solid var(--color-border-light);
  padding: 12px 14px;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}
.process-title {
  margin: 0 0 6px;
  font-size: 14px;
  font-weight: 600;
  line-height: 1.5;
  color: var(--color-text-primary);
  overflow-wrap: anywhere;
}
.process-meta { margin: 0 0 8px; font-size: 12px; color: var(--color-text-secondary); }
.process-footer { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.process-node {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-width: 0;
  padding: 3px 10px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 500;
  color: var(--color-text-secondary);
  background: var(--color-bg-hover);
}
.process-node span { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.summary-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: calc(100vh - 220px);
  padding: 32px 16px;
  text-align: center;
}
.summary-loading p { margin: 16px 0 0; color: var(--color-text-secondary); }
.raw-stream { width: 100%; margin-top: 20px; text-align: left; }
.raw-toggle {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-md);
  background: var(--color-bg-card);
  color: var(--color-text-primary);
}
.raw-stream pre {
  margin: 8px 0 0;
  max-height: 240px;
  overflow: auto;
  white-space: pre-wrap;
  padding: 12px;
  border-radius: var(--radius-md);
  background: var(--color-bg-page);
  border: 1px solid var(--color-border-light);
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
  border-radius: 12px;
  background: var(--color-bg-card);
  padding: 14px 16px;
}
.summary-card h4 {
  margin: 0 0 8px;
  font-size: 15px;
  font-weight: 700;
  color: var(--color-text-primary);
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
