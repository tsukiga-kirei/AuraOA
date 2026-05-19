<script setup lang="ts">
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  ReloadOutlined,
  FieldTimeOutlined,
  DownOutlined,
  UpOutlined,
  InfoCircleOutlined,
  LoadingOutlined,
  WarningOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import { marked } from 'marked'
import type { AuditResult } from '~/types/audit'
import type { EmbedContextResponse, EmbedProcessSummary } from '~/types/embed'
import { waitForParentRequestId } from '~/composables/useEmbedParent'

definePageMeta({ layout: 'embed' })

const { t } = useI18n()
const { getContext, executeEmbed, waitAuditJob } = useEmbedApi()

const processId = ref('')
const pageLoading = ref(true)
const auditing = ref(false)
const context = ref<EmbedContextResponse | null>(null)
const currentResult = ref<AuditResult | null>(null)
const showReasoning = ref(false)
const pageError = ref('')
const waitingParent = ref(true)

const processInfo = computed<EmbedProcessSummary | null>(() => context.value?.process ?? null)

const recommendationConfig = computed(() => ({
  approve: { color: 'var(--color-success)', bg: 'var(--color-success-bg)', icon: CheckCircleOutlined, label: t('dashboard.rec.approve') },
  return: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', icon: ReloadOutlined, label: t('dashboard.rec.return') },
  review: { color: 'var(--color-info)', bg: 'var(--color-info-bg)', icon: InfoCircleOutlined, label: t('dashboard.rec.review') },
}))

const getScoreColorConfig = (score: number | undefined) => {
  if (score === undefined || score === null) return { color: 'var(--color-info)', bg: 'var(--color-info-bg)' }
  if (score < 60) return { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)' }
  if (score > 80) return { color: 'var(--color-success)', bg: 'var(--color-success-bg)' }
  return { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)' }
}

const isResultAsyncRunning = (r: AuditResult | null) =>
  !!(r?.status && ['pending', 'assembling', 'reasoning', 'extracting'].includes(r.status))

const getDurationSec = (ms?: number) => ((ms || 0) / 1000).toFixed(1)

const renderMarkdown = (text: string) => {
  try {
    return marked.parse(text || '') as string
  } catch {
    return text
  }
}

const formatLastAuditAt = (iso?: string) => {
  if (!iso) return '—'
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString()
}

const processMetaLine = computed(() => {
  const p = processInfo.value
  if (!p) return ''
  const parts = [p.applicant, p.department, p.process_type_label || p.process_type].filter(Boolean)
  return parts.join(' · ')
})

async function runAudit(trigger: 'embed_auto' | 'embed_manual') {
  if (!processId.value || auditing.value) return
  auditing.value = true
  currentResult.value = {
    trace_id: '',
    process_id: processId.value,
    status: 'pending',
    rule_results: [],
    risk_points: [],
    suggestions: [],
    confidence: 0,
    ai_reasoning: '',
    duration_ms: 0,
    progress_steps: [],
  }
  try {
    const result = await executeEmbed(
      {
        process_id: processId.value,
        process_type: processInfo.value?.process_type,
        title: processInfo.value?.title,
        trigger_source: trigger,
      },
      (st) => {
        const oldReasoning = currentResult.value?.ai_reasoning || ''
        currentResult.value = { ...currentResult.value, ...st } as AuditResult
        if (oldReasoning.length > (st.ai_reasoning?.length || 0)) {
          if (currentResult.value) currentResult.value.ai_reasoning = oldReasoning
        }
      },
    )
    currentResult.value = result
    await refreshContext(false)
    if (trigger === 'embed_manual') {
      message.success(t('embed.reAuditDone'))
    }
  } catch (e: any) {
    message.error(e?.message || t('embed.auditFailed'))
    currentResult.value = null
    await refreshContext(false)
  } finally {
    auditing.value = false
  }
}

async function refreshContext(autoRun = true) {
  if (!processId.value) return
  try {
    const resp = await getContext(processId.value)
    context.value = resp
    if (resp.audit_result && !auditing.value) {
      currentResult.value = resp.audit_result
    }
    if (autoRun && resp.should_auto_audit && !auditing.value && !resp.running_job_id) {
      await runAudit('embed_auto')
    } else if (resp.running_job_id && !auditing.value) {
      auditing.value = true
      try {
        currentResult.value = await waitAuditJob(resp.running_job_id, (st) => {
          currentResult.value = { ...currentResult.value, ...st } as AuditResult
        })
        await refreshContext(false)
      } finally {
        auditing.value = false
      }
    }
  } catch (e: any) {
    pageError.value = e?.message || t('embed.loadFailed')
  }
}

async function bootstrap() {
  pageLoading.value = true
  pageError.value = ''
  context.value = null
  currentResult.value = null
  try {
    if (!processId.value) {
      pageError.value = t('embed.noRequestId')
      return
    }
    await refreshContext(true)
  } finally {
    pageLoading.value = false
    waitingParent.value = false
  }
}

const handleReAudit = () => runAudit('embed_manual')

onMounted(async () => {
  waitingParent.value = true
  processId.value = await waitForParentRequestId()
  await bootstrap()
})
</script>

<template>
  <div class="embed-audit">
    <div class="embed-header">
      <h2 class="embed-title">
        <CheckCircleOutlined v-if="context?.supported && currentResult && !isResultAsyncRunning(currentResult)" />
        <ThunderboltOutlined v-else />
        {{ t('embed.title') }}
      </h2>
      <div v-if="context?.last_audit_at" class="embed-last-audit">
        {{ t('embed.lastAuditAt') }}：{{ formatLastAuditAt(context.last_audit_at) }}
        <a-tag v-if="context.stale" color="warning" style="margin-left: 8px;">{{ t('embed.staleTag') }}</a-tag>
      </div>
    </div>

    <a-spin :spinning="pageLoading || waitingParent">
      <div v-if="waitingParent && !pageError" class="result-empty result-empty--loading">
        <LoadingOutlined spin style="font-size: 24px;" />
        <p>{{ t('embed.waitingParent') }}</p>
      </div>

      <template v-else>
      <a-alert
        v-if="pageError"
        type="error"
        :message="pageError"
        show-icon
        style="margin-bottom: 16px;"
      />

      <a-result
        v-else-if="context && !context.supported"
        status="warning"
        :title="t('embed.unsupportedTitle')"
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
        <div v-if="processInfo" class="dashboard-process-summary">
          <span class="dashboard-process-summary__title">{{ processInfo.title }}</span>
          <span class="dashboard-process-summary__meta">{{ processMetaLine }}</span>
          <span class="dashboard-process-summary__node">
            <FieldTimeOutlined />
            {{ t('dashboard.currentNode') }}: {{ processInfo.current_node || '—' }}
          </span>
        </div>

        <div class="result-action-bar">
          <a-button
            type="default"
            :loading="auditing"
            :disabled="!!context.running_job_id && !auditing"
            @click="handleReAudit"
          >
            <ReloadOutlined /> {{ t('dashboard.reAudit') }}
          </a-button>
        </div>

        <div v-if="auditing || (currentResult && isResultAsyncRunning(currentResult))" class="result-async-panel">
          <a-spin size="large">
            <div class="async-progress-steps">
              <div v-for="s in (currentResult?.progress_steps || [])" :key="s.key" class="async-step-row">
                <CheckCircleOutlined v-if="s.done" style="color: var(--color-success);" />
                <LoadingOutlined v-else-if="s.current" spin style="color: var(--color-primary);" />
                <CloseCircleOutlined v-else-if="s.failed" style="color: var(--color-danger);" />
                <span v-else class="async-step-pending-dot" />
                <span>{{ s.label }}</span>
              </div>
            </div>
          </a-spin>
        </div>

        <template v-else-if="currentResult">
          <template v-if="currentResult.status === 'failed'">
            <div class="result-banner result-banner--error">
              <WarningOutlined class="result-banner-icon" style="color: var(--color-danger);" />
              <div class="result-banner-info">
                <div class="result-banner-title" style="color: var(--color-danger);">{{ t('dashboard.auditFailed') }}</div>
                <div class="result-banner-meta">{{ currentResult.error_message }}</div>
              </div>
            </div>
          </template>

          <template v-else-if="currentResult.parse_error">
            <div class="result-banner result-banner--error">
              <WarningOutlined class="result-banner-icon" style="color: var(--color-danger);" />
              <div class="result-banner-info">
                <div class="result-banner-title" style="color: var(--color-danger);">{{ t('dashboard.parseErrorTitle') }}</div>
                <div class="result-banner-meta">{{ currentResult.parse_error }}</div>
              </div>
            </div>
          </template>

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

            <div v-if="currentResult.ai_reasoning" class="result-section">
              <button type="button" class="reasoning-toggle" @click="showReasoning = !showReasoning">
                <span>{{ t('dashboard.aiReasoning') }}</span>
                <DownOutlined v-if="!showReasoning" />
                <UpOutlined v-else />
              </button>
              <div v-show="showReasoning" class="ai-reasoning">
                <div class="markdown-body" v-html="renderMarkdown(currentResult.ai_reasoning || '')" />
              </div>
            </div>
          </template>
        </template>

        <div v-else-if="!auditing" class="result-empty">
          <div class="result-empty-icon"><ThunderboltOutlined /></div>
          <h4>{{ t('embed.emptyTitle') }}</h4>
          <p>{{ t('embed.emptyDesc') }}</p>
          <a-button type="primary" @click="handleReAudit">
            <ThunderboltOutlined /> {{ t('embed.startAudit') }}
          </a-button>
        </div>
      </template>
      </template>
    </a-spin>
  </div>
</template>

<style scoped>
.embed-audit { max-width: 720px; margin: 0 auto; }
.embed-header { margin-bottom: 16px; }
.embed-title {
  display: flex; align-items: center; gap: 8px;
  font-size: 18px; font-weight: 700; margin: 0 0 6px; color: var(--color-text-primary);
}
.embed-last-audit { font-size: 12px; color: var(--color-text-tertiary); }
.unsupported-process { text-align: left; margin-top: 8px; }
.unsupported-process__title { font-weight: 600; color: var(--color-text-primary); }
.unsupported-process__meta { font-size: 13px; color: var(--color-text-tertiary); margin-top: 4px; }

.dashboard-process-summary {
  display: flex; flex-direction: column; gap: 4px; margin-bottom: 16px;
  padding: 12px 14px; border-radius: var(--radius-md);
  background: var(--color-bg-card); border: 1px solid var(--color-border-light);
}
.dashboard-process-summary__title { font-size: 15px; font-weight: 600; color: var(--color-text-primary); }
.dashboard-process-summary__meta { font-size: 13px; color: var(--color-text-secondary); }
.dashboard-process-summary__node {
  display: inline-flex; align-items: center; gap: 4px;
  font-size: 12px; color: var(--color-text-tertiary);
}

.result-action-bar { display: flex; justify-content: flex-end; margin-bottom: 16px; gap: 8px; }

.result-async-panel { padding: 8px 0 16px; }
.async-progress-steps { display: flex; flex-direction: column; gap: 10px; margin-top: 12px; }
.async-step-row { display: flex; align-items: center; gap: 10px; font-size: 13px; color: var(--color-text-secondary); }
.async-step-pending-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--color-border); display: inline-block; flex-shrink: 0; }

.result-banner {
  display: flex; align-items: center; padding: 16px 20px;
  border-radius: var(--radius-lg); border-left: 4px solid; margin-bottom: 24px; gap: 14px;
}
.result-banner--error { background: var(--color-danger-bg); border-color: var(--color-danger); }
.result-banner-icon { font-size: 28px; flex-shrink: 0; }
.result-banner-info { flex: 1; }
.result-banner-title { font-size: 16px; font-weight: 700; }
.result-banner-meta { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }
.result-score { font-size: 36px; font-weight: 800; line-height: 1; }

.result-section { margin-bottom: 24px; }
.result-section-title { font-size: 14px; font-weight: 600; color: var(--color-text-primary); margin: 0 0 12px; }
.rule-checks { display: flex; flex-direction: column; gap: 8px; }
.rule-check-item {
  display: flex; gap: 12px; padding: 12px 16px;
  border-radius: var(--radius-md); border: 1px solid var(--color-border-light);
}
.rule-check-item--pass { border-left: 3px solid var(--color-success); }
.rule-check-item--fail { border-left: 3px solid var(--color-danger); background: var(--color-danger-bg); }
.rule-check-status { font-size: 18px; flex-shrink: 0; padding-top: 1px; }
.rule-check-content { flex: 1; min-width: 0; }
.rule-check-name { font-size: 14px; font-weight: 500; color: var(--color-text-primary); margin-bottom: 4px; }
.rule-check-reasoning { font-size: 13px; color: var(--color-text-secondary); line-height: 1.5; }

.risk-suggest-row { display: grid; grid-template-columns: 1fr; gap: 16px; margin-bottom: 24px; }
@media (min-width: 640px) {
  .risk-suggest-row { grid-template-columns: 1fr 1fr; }
}
.insight-card { border-radius: var(--radius-md); padding: 16px; border: 1px solid var(--color-border-light); }
.insight-card--risk { background: linear-gradient(135deg, rgba(239, 68, 68, 0.04), rgba(239, 68, 68, 0.01)); border-color: rgba(239, 68, 68, 0.15); }
.insight-card--suggest { background: linear-gradient(135deg, rgba(79, 70, 229, 0.04), rgba(79, 70, 229, 0.01)); border-color: rgba(79, 70, 229, 0.15); }
.insight-card-header { display: flex; align-items: center; gap: 8px; font-size: 13px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 10px; }
.insight-card-list { margin: 0; padding-left: 18px; display: flex; flex-direction: column; gap: 6px; }
.insight-card-list li { font-size: 13px; line-height: 1.6; color: var(--color-text-secondary); }

.reasoning-toggle {
  width: 100%; display: flex; align-items: center; justify-content: space-between;
  padding: 10px 14px; border: 1px solid var(--color-border-light); border-radius: var(--radius-md);
  background: var(--color-bg-card); cursor: pointer; font-size: 14px; font-weight: 600;
  color: var(--color-text-primary); margin-bottom: 8px;
}
.reasoning-toggle:hover { background: var(--color-bg-hover); }
.ai-reasoning { background: var(--color-bg-page); border-radius: var(--radius-md); padding: 16px; border: 1px solid var(--color-border-light); }

.result-empty { text-align: center; padding: 40px 20px; }
.result-empty-icon {
  width: 64px; height: 64px; border-radius: 50%; background: var(--color-primary-bg);
  color: var(--color-primary); font-size: 28px; display: flex; align-items: center;
  justify-content: center; margin: 0 auto 16px;
}
.result-empty h4 { font-size: 16px; font-weight: 600; margin: 0 0 8px; }
.result-empty p { font-size: 13px; color: var(--color-text-tertiary); margin: 0 auto 16px; max-width: 280px; }
</style>
