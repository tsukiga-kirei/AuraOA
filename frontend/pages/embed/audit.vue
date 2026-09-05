<script setup lang="ts">
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  ReloadOutlined,
  DownOutlined,
  UpOutlined,
  InfoCircleOutlined,
  LoadingOutlined,
  WarningOutlined,
  ThunderboltOutlined,
  EyeOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import type { AuditResult } from '~/types/audit'
import type { EmbedContextResponse, EmbedProcessSummary } from '~/types/embed'
import { waitForParentEmbedContext } from '~/composables/useEmbedParent'

definePageMeta({ layout: 'embed' })

const { t } = useI18n()
const { getContext, executeEmbed } = useEmbedApi()
const { setupEmbedSession } = useEmbedSession()

type AuditProgressStep = NonNullable<AuditResult['progress_steps']>[number]

const DEFAULT_PROGRESS_STEPS: AuditProgressStep[] = [
  { key: 'pending', label: '排队中', done: false, current: true },
  { key: 'assembling', label: '组装提示词', done: false, current: false },
  { key: 'reasoning', label: '推理分析', done: false, current: false },
  { key: 'extracting', label: '结构化提取', done: false, current: false },
]

const processId = ref('')
const pageLoading = ref(true)
const auditing = ref(false)
const context = ref<EmbedContextResponse | null>(null)
const currentResult = ref<AuditResult | null>(null)
const showReasoning = ref(false)
const showDeepThinking = ref(false)
const pageError = ref('')
const waitingParent = ref(true)
const eventSourceStream = ref<EventSource | null>(null)
const streamJobId = ref('')

// 双模视角控制：'standard' (标准审查) | 'personal' (我的定制视角)
const activePerspective = ref<'standard' | 'personal'>('standard')
const userManuallySwitchedPerspective = ref(false)

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

const isAuditingActive = computed(
  () => auditing.value || !!(currentResult.value && isResultAsyncRunning(currentResult.value)),
)

const filteredProgressSteps = computed((): AuditProgressStep[] => {
  const steps = currentResult.value?.progress_steps
  if (!steps?.length) return DEFAULT_PROGRESS_STEPS.map(s => ({ ...s }))
  return steps.filter(s => s.key !== 'pending')
})

const embedHeaderStatus = computed(() => {
  if (pageLoading.value || waitingParent.value) {
    return {
      label: t('embed.title'),
      color: 'var(--color-text-primary)',
      bg: 'transparent',
      icon: ThunderboltOutlined,
      spin: false,
    }
  }
  if (isAuditingActive.value) {
    return {
      label: t('embed.statusAuditing'),
      color: 'var(--color-primary)',
      bg: 'var(--color-primary-bg)',
      icon: LoadingOutlined,
      spin: true,
    }
  }
  const r = currentResult.value
  if (!r) {
    return {
      label: t('embed.statusWaiting'),
      color: 'var(--color-text-tertiary)',
      bg: 'var(--color-bg-hover)',
      icon: ThunderboltOutlined,
      spin: false,
    }
  }
  if (r.status === 'failed' || r.status === 'cancelled' || r.parse_error) {
    return {
      label: t('embed.statusFailed'),
      color: 'var(--color-danger)',
      bg: 'var(--color-danger-bg)',
      icon: CloseCircleOutlined,
      spin: false,
    }
  }
  const rec = r.recommendation || 'review'
  if (rec === 'approve') {
    return {
      label: t('embed.statusPassed'),
      color: 'var(--color-success)',
      bg: 'var(--color-success-bg)',
      icon: CheckCircleOutlined,
      spin: false,
    }
  }
  if (rec === 'return') {
    return {
      label: t('embed.statusReturn'),
      color: 'var(--color-danger)',
      bg: 'var(--color-danger-bg)',
      icon: CloseCircleOutlined,
      spin: false,
    }
  }
  return {
    label: t('embed.statusReview'),
    color: 'var(--color-info)',
    bg: 'var(--color-info-bg)',
    icon: EyeOutlined,
    spin: false,
  }
})

const getDurationSec = (ms?: number) => ((ms || 0) / 1000).toFixed(1)

const formatLastAuditAt = (iso?: string) => {
  return iso ? formatDateTimeInAppZone(iso) : '—'
}

const processMetaRows = computed(() => {
  const p = processInfo.value
  if (!p) return []
  return [
    [p.applicant, p.department].filter(Boolean).join(' · '),
    [p.process_type_label || p.process_type, p.current_node].filter(Boolean).join(' · '),
  ].filter(Boolean)
})

const mergeAuditProgress = (st: Partial<AuditResult>) => {
  const oldReasoning = currentResult.value?.ai_reasoning || ''
  currentResult.value = { ...currentResult.value, ...st } as AuditResult
  const nextReasoning = currentResult.value?.ai_reasoning || ''
  if (oldReasoning.length > nextReasoning.length && currentResult.value) {
    currentResult.value.ai_reasoning = oldReasoning
  }
}

const disconnectStream = () => {
  if (eventSourceStream.value) {
    eventSourceStream.value.close()
    eventSourceStream.value = null
  }
  streamJobId.value = ''
}

const startSSE = (auditResultId?: string) => {
  if (!process.client || !auditResultId || streamJobId.value === auditResultId) return
  disconnectStream()
  streamJobId.value = auditResultId
  eventSourceStream.value = new EventSource(
    useEmbedAuth().appendEmbedTokenQuery(`/api/embed/stream/${encodeURIComponent(auditResultId)}`),
  )
  eventSourceStream.value.onmessage = (event) => {
    if (!currentResult.value) return
    const chunk = event.data || ''
    const existing = currentResult.value.ai_reasoning || ''
    if (!existing || chunk.startsWith(existing)) {
      currentResult.value.ai_reasoning = chunk
    } else {
      currentResult.value.ai_reasoning = existing + chunk
    }
  }
  eventSourceStream.value.onerror = () => {
    disconnectStream()
  }
}

function createPendingResult(): AuditResult {
  return {
    trace_id: '',
    process_id: processId.value,
    status: 'pending',
    rule_results: [],
    risk_points: [],
    suggestions: [],
    confidence: 0,
    ai_reasoning: '',
    duration_ms: 0,
    progress_steps: DEFAULT_PROGRESS_STEPS.map(s => ({ ...s })),
  }
}

function updateCurrentResultForPerspective() {
  if (auditing.value) return
  if (activePerspective.value === 'personal') {
    currentResult.value = (context.value?.personal_view?.audit_result as AuditResult | null) || null
  } else {
    currentResult.value = (context.value?.audit_result as AuditResult | null) || null
  }
}

const handleSwitchPerspective = (p: 'standard' | 'personal') => {
  if (activePerspective.value === p) return
  userManuallySwitchedPerspective.value = true
  activePerspective.value = p
  disconnectStream()
  updateCurrentResultForPerspective()
}

const activeViewLastAuditAt = computed(() => {
  if (activePerspective.value === 'personal') {
    return context.value?.personal_view?.last_audit_at
  }
  return context.value?.last_audit_at
})

const activeViewStale = computed(() => {
  if (activePerspective.value === 'personal') {
    return false
  }
  return !!context.value?.stale
})

async function runAudit(trigger: 'embed_auto' | 'embed_manual', useLatestConfig = false) {
  if (!processId.value || auditing.value) return
  disconnectStream()
  auditing.value = true
  currentResult.value = createPendingResult()
  try {
    const result = await executeEmbed(
      {
        process_id: processId.value,
        process_type: processInfo.value?.process_type,
        title: processInfo.value?.title,
        trigger_source: trigger,
        trigger_detail: trigger === 'embed_manual' ? 'manual' : 'visible_open',
        use_latest_config: useLatestConfig,
        perspective: activePerspective.value,
      },
      (st) => {
        mergeAuditProgress(st)
        startSSE(st.id)
      },
    )
    mergeAuditProgress(result)
    await refreshContext(false, true)
    if (trigger === 'embed_manual') {
      message.success(t('embed.reAuditDone'))
    }
  } catch (e: any) {
    message.error(e?.message || t('embed.auditFailed'))
    currentResult.value = null
    await refreshContext(false, true)
  } finally {
    auditing.value = false
    disconnectStream()
  }
}

async function refreshContext(autoRun = true, preferCached = false) {
  if (!processId.value) return
  try {
    const resp = await getContext(processId.value, preferCached)
    context.value = resp
    if (!userManuallySwitchedPerspective.value && resp.default_perspective) {
      activePerspective.value = resp.default_perspective as 'standard' | 'personal'
    }
    if (!auditing.value || !autoRun) {
      updateCurrentResultForPerspective()
    }
    if (autoRun && (resp.should_auto_audit || resp.running_job_id) && !auditing.value) {
      await runAudit('embed_auto')
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
    // 首次进入先做轻量指纹检查；不识别附件正文、不调用 AI。
    await refreshContext(false, false)
  } finally {
    pageLoading.value = false
    waitingParent.value = false
  }
  const embedContext = context.value as EmbedContextResponse | null
  if (embedContext?.supported && (embedContext.should_auto_audit || embedContext.running_job_id) && !auditing.value) {
    // 发现变化或已有后台任务时进入交互队列，待执行任务会切换到可见页队列。
    await runAudit('embed_auto')
  }
}

const handleReAudit = () => runAudit('embed_manual')
const handleReAuditWithLatestConfig = () => runAudit('embed_manual', true)

const processStat = computed(() => {
  const ms = currentResult.value?.duration_ms
  if (ms) {
    const sec = ms / 1000
    const tone = sec >= 120 ? 'danger' : sec >= 60 ? 'warning' : 'success'
    return { text: `耗时 ${getDurationSec(ms)} 秒`, tone }
  }
  if (isAuditingActive.value) {
    return { text: '执行中', tone: 'running' }
  }
  return null
})

onMounted(async () => {
  waitingParent.value = true
  const parentCtx = await waitForParentEmbedContext()
  processId.value = parentCtx.requestId
  waitingParent.value = false
  if (!parentCtx.embedToken) {
    pageError.value = t('embed.missingToken')
    pageLoading.value = false
    return
  }
  try {
    await setupEmbedSession(parentCtx.embedToken)
  } catch (e: any) {
    pageError.value = e?.message || t('embed.missingToken')
    pageLoading.value = false
    return
  }
  await bootstrap()
})

onBeforeUnmount(() => {
  disconnectStream()
})
</script>

<template>
  <div class="embed-audit">
    <div class="embed-header">
      <h2
        class="embed-title"
        :style="{ color: embedHeaderStatus.color }"
      >
        <span
          class="embed-title-badge"
          :style="{ background: embedHeaderStatus.bg, color: embedHeaderStatus.color }"
        >
          <component
            :is="embedHeaderStatus.icon"
            :spin="embedHeaderStatus.spin"
          />
        </span>
        {{ embedHeaderStatus.label }}
      </h2>
      <div
        v-if="!isAuditingActive && activeViewLastAuditAt"
        class="embed-last-audit"
      >
        {{ t('embed.lastAuditAt') }}：{{ formatLastAuditAt(activeViewLastAuditAt) }}
        <a-tag v-if="activeViewStale" color="warning" style="margin-left: 8px;">{{ t('embed.staleTag') }}</a-tag>
		<a-tag v-if="context?.config_version_no" color="blue" style="margin-left: 8px;">
		  {{ t('executionConfig.version', [context.config_version_no]) }}
		</a-tag>
      </div>
      <p v-else-if="isAuditingActive" class="embed-last-audit embed-last-audit--active">
        {{ t('dashboard.aiAnalyzingSub') }}
      </p>
    </div>

    <div
      v-if="pageLoading || waitingParent"
      class="embed-page-loading"
    >
      <a-spin size="large" :tip="waitingParent ? t('embed.waitingParent') : undefined" />
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
        <!-- 双模视角切换器（当且仅当当前 OA 人员在 AuraOA 存在个人定制能力时呈现） -->
        <div v-if="context?.personal_view?.available" class="perspective-nav-wrap">
          <div class="perspective-nav">
            <button
              type="button"
              class="perspective-nav-item"
              :class="{ 'perspective-nav-item--active': activePerspective === 'standard' }"
              @click="handleSwitchPerspective('standard')"
            >
              <span class="perspective-title">{{ t('embed.perspective.standard') }}</span>
              <span v-if="context.has_audit" class="perspective-dot" :title="t('embed.perspective.standardHasAudit')" />
            </button>
            <button
              type="button"
              class="perspective-nav-item"
              :class="{ 'perspective-nav-item--active': activePerspective === 'personal' }"
              @click="handleSwitchPerspective('personal')"
            >
              <span class="perspective-title">{{ t('embed.perspective.personal') }}</span>
              <span v-if="context?.personal_view?.has_audit" class="perspective-dot perspective-dot--personal" :title="t('embed.perspective.personalHasAudit')" />
              <span v-else class="perspective-unexecuted-tag">{{ t('embed.perspective.unexecuted') }}</span>
            </button>
          </div>
        </div>

        <div v-if="processInfo" class="embed-process-card">

          <div class="embed-process-card__body">
            <h3 class="embed-process-card__title" :title="processInfo.title">
              {{ processInfo.title }}
            </h3>
            <div class="embed-process-card__main">
              <div class="embed-process-card__copy">
                <div
                  v-for="row in processMetaRows"
                  :key="row"
                  class="embed-process-card__meta-row"
                >
                  <span
                    class="embed-process-card__meta-chip"
                    :title="row"
                  >
                    {{ row }}
                  </span>
                </div>
              </div>
              <div class="embed-process-card__actions">
                <div
                  v-if="processStat"
                  class="embed-process-card__stat"
                  :class="`embed-process-card__stat--${processStat.tone}`"
                >
                  {{ processStat.text }}
                </div>
                <a-button
                  class="embed-process-card__action"
                  type="text"
                  size="small"
                  :disabled="auditing || (!!context.running_job_id && !auditing)"
                  @click="handleReAudit"
                >
                  <LoadingOutlined v-if="auditing" spin />
                  <ReloadOutlined v-else />
                  <span>{{ t('dashboard.reAudit') }}</span>
                </a-button>
				<!--
				  OA 嵌入块暂不开放“使用最新配置重新执行”。
				  仅注释入口，保留完整执行逻辑，后续需要时可直接恢复。
				<a-button
				  v-if="context.config_upgrade_available"
				  class="embed-process-card__action"
				  type="text"
				  size="small"
				  :disabled="auditing"
				  @click="handleReAuditWithLatestConfig"
				>
				  <ThunderboltOutlined />
				  <span>{{ t('executionConfig.useLatest') }}</span>
				</a-button>
				-->
              </div>
            </div>
          </div>
        </div>

        <div v-if="isAuditingActive" class="embed-auditing-center">
          <a-spin size="large" />
          <p class="embed-auditing-title">{{ t('dashboard.aiAnalyzing') }}</p>
          <div class="async-progress-steps">
            <div
              v-for="s in filteredProgressSteps"
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
          <div v-if="currentResult?.ai_reasoning" class="result-section embed-reasoning-live">
            <AiMarkdownStream
              :title="t('dashboard.aiReasoning')"
              :text="currentResult.ai_reasoning || ''"
              max-height="260px"
            />
          </div>
        </div>

        <template v-else-if="currentResult">
          <template v-if="currentResult.status === 'failed' || currentResult.status === 'cancelled'">
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

          <template v-else>
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
                <div
                  class="result-banner-title"
                  :style="{ color: getScoreColorConfig(currentResult.overall_score)?.color }"
                >
                  {{ recommendationConfig[currentResult.recommendation || 'review']?.label }}
                </div>
                <div class="result-banner-meta">
                  {{ t('dashboard.overallScore') }} {{ currentResult.overall_score }}{{ t('dashboard.points') }}
                  · {{ t('dashboard.confidence') }} {{ currentResult.confidence }}%
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

            <div v-if="currentResult.deep_thinking" class="result-section">
              <button type="button" class="reasoning-toggle" @click="showDeepThinking = !showDeepThinking">
                <span style="display: flex; align-items: center; gap: 6px;">
                  <ThunderboltOutlined style="color: var(--color-primary);" />
                  {{ t('dashboard.deepThinking', '深度思考过程') }}
                </span>
                <DownOutlined v-if="!showDeepThinking" />
                <UpOutlined v-else />
              </button>
              <AiMarkdownStream
                v-show="showDeepThinking"
                :text="currentResult.deep_thinking || ''"
                max-height="320px"
              />
            </div>

            <div v-if="currentResult.ai_reasoning" class="result-section">
              <button type="button" class="reasoning-toggle" @click="showReasoning = !showReasoning">
                <span>{{ t('dashboard.aiReasoning') }}</span>
                <DownOutlined v-if="!showReasoning" />
                <UpOutlined v-else />
              </button>
              <AiMarkdownStream
                v-show="showReasoning"
                :text="currentResult.ai_reasoning || ''"
                max-height="320px"
              />
            </div>
          </template>
        </template>

        <div v-else-if="!auditing" class="result-empty">
          <div class="result-empty-icon"><ThunderboltOutlined /></div>
          <h4>
            {{ activePerspective === 'personal' ? t('embed.perspective.personalEmptyTitle') : t('embed.emptyTitle') }}
          </h4>
          <p>
            {{ activePerspective === 'personal' ? t('embed.perspective.personalEmptyDesc') : t('embed.emptyDesc') }}
          </p>
          <a-button type="primary" @click="handleReAudit">
            <ThunderboltOutlined />
            {{ activePerspective === 'personal' ? t('embed.perspective.startPersonalAudit') : t('embed.startAudit') }}
          </a-button>
        </div>

      </template>
    </template>
  </div>
</template>

<style scoped>
.embed-audit { max-width: 720px; margin: 0 auto; min-height: 100vh; padding: 12px 12px 28px; }

.embed-header {
  margin-bottom: 14px; padding-bottom: 12px;
  border-bottom: 1px solid var(--color-border-light);
}
.embed-title {
  display: flex; align-items: center; gap: 10px;
  font-size: 17px; font-weight: 700; margin: 0 0 4px;
  transition: color 0.25s ease; letter-spacing: 0;
}
.embed-title-badge {
  display: inline-flex; align-items: center; justify-content: center;
  width: 30px; height: 30px; border-radius: 8px; font-size: 16px; flex-shrink: 0;
}
.embed-last-audit {
  font-size: 12px; color: var(--color-text-secondary); margin: 0; line-height: 1.4;
}
.embed-last-audit--active { color: var(--color-primary); font-size: 12px; }

.embed-page-loading {
  display: flex; align-items: center; justify-content: center;
  min-height: calc(100vh - 120px); padding: 48px 20px;
}

.perspective-nav-wrap {
  margin-bottom: 12px;
}
.perspective-nav {
  display: flex;
  background: var(--color-bg-hover, #f1f5f9);
  padding: 3px;
  border-radius: var(--radius-md, 8px);
  gap: 4px;
}
.perspective-nav-item {
  flex: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 6px 12px;
  border: none;
  background: transparent;
  border-radius: var(--radius-sm, 6px);
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast, 0.2s);
}
.perspective-nav-item:hover {
  color: var(--color-text-primary);
}
.perspective-nav-item--active {
  background: var(--color-bg-card, #ffffff);
  color: var(--color-primary, #1890ff);
  font-weight: 600;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
}
.perspective-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--color-success, #52c41a);
  display: inline-block;
}
.perspective-dot--personal {
  background: var(--color-primary, #1890ff);
}
.perspective-unexecuted-tag {
  font-size: 10px;
  font-weight: normal;
  padding: 0 4px;
  border-radius: 4px;
  background: rgba(0, 0, 0, 0.06);
  color: var(--color-text-tertiary);
}

.embed-process-card {
  margin-bottom: 14px;
  border-radius: var(--radius-lg);
  background: var(--color-bg-card);
  border: 1px solid var(--color-border-light);
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
  overflow: hidden;
}
.embed-process-card__body { padding: 14px 16px; }
.embed-process-card__main {
  display: grid; grid-template-columns: minmax(0, 1fr) 104px;
  gap: 16px; align-items: center;
}
.embed-process-card__copy { min-width: 0; }
.embed-process-card__title {
  min-width: 0; margin: 0 0 10px;
  font-size: 15px; font-weight: 700; line-height: 1.5;
  color: var(--color-text-primary);
  word-break: keep-all; overflow-wrap: anywhere;
  white-space: normal;
}
.embed-process-card__actions {
  display: flex; flex-direction: column; align-items: stretch; justify-content: center; gap: 8px;
  flex-shrink: 0;
  min-width: 0;
  padding-left: 12px;
  border-left: 1px solid var(--color-border-light);
}
.embed-process-card__stat {
  display: inline-flex; align-items: center; justify-content: center;
  width: 100%; height: 28px; padding: 0 7px; border-radius: var(--radius-md);
  background: var(--color-bg-hover);
  border: 1px solid var(--color-border-light);
  text-align: center;
  color: var(--color-text-secondary); font-size: 12px; font-weight: 700;
  white-space: nowrap;
}
.embed-process-card__stat--success {
  color: var(--color-success);
  background: var(--color-success-bg);
  border-color: color-mix(in srgb, var(--color-success) 24%, transparent);
}
.embed-process-card__stat--warning {
  color: var(--color-warning);
  background: var(--color-warning-bg);
  border-color: color-mix(in srgb, var(--color-warning) 28%, transparent);
}
.embed-process-card__stat--danger {
  color: var(--color-danger);
  background: var(--color-danger-bg);
  border-color: color-mix(in srgb, var(--color-danger) 24%, transparent);
}
.embed-process-card__stat--running {
  color: var(--color-primary);
  background: var(--color-primary-bg);
  border-color: color-mix(in srgb, var(--color-primary) 18%, transparent);
}
.embed-process-card__action.ant-btn {
  flex-shrink: 0; display: inline-flex; align-items: center; justify-content: center; gap: 4px;
  width: 100%; height: 30px !important; padding: 0 8px !important; margin: 0;
  font-size: 12px !important; line-height: 28px !important;
  color: var(--color-primary) !important;
  border-radius: var(--radius-md) !important; white-space: nowrap;
  background: var(--color-primary-bg) !important;
}
.embed-process-card__action.ant-btn:hover:not(:disabled) {
  background: color-mix(in srgb, var(--color-primary-bg) 70%, var(--color-bg-hover)) !important;
  color: var(--color-primary) !important;
}
.embed-process-card__action.ant-btn:disabled {
  color: color-mix(in srgb, var(--color-primary) 70%, var(--color-text-tertiary)) !important;
  background: var(--color-primary-bg) !important;
}
.embed-process-card__action :deep(.anticon) {
  width: 14px; font-size: 13px;
}
@media (max-width: 420px) {
  .embed-process-card__actions {
    gap: 6px;
    padding-left: 8px;
  }
  .embed-process-card__stat {
    padding: 0 5px;
  }
}
.embed-process-card__meta-row {
  display: flex;
  align-items: center;
  min-width: 0;
}
.embed-process-card__meta-row + .embed-process-card__meta-row {
  margin-top: 7px;
}
.embed-process-card__meta-chip {
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
.embed-process-card__meta-row:first-of-type .embed-process-card__meta-chip:first-child {
  color: var(--color-text-primary);
  background: color-mix(in srgb, var(--color-primary-bg) 54%, var(--color-bg-card));
  border-color: color-mix(in srgb, var(--color-primary) 22%, var(--color-border-light));
}

.embed-auditing-center {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  min-height: calc(100vh - 200px); padding: 32px 16px; text-align: center;
}
.embed-auditing-title {
  margin: 16px 0 20px; font-size: 14px; font-weight: 500;
  color: var(--color-text-secondary);
}
.embed-reasoning-live {
  width: 100%; margin-top: 24px; text-align: left;
}

.unsupported-process { text-align: left; margin-top: 8px; }
.unsupported-process__title { font-weight: 600; color: var(--color-text-primary); }
.unsupported-process__meta { font-size: 13px; color: var(--color-text-tertiary); margin-top: 4px; }

.async-progress-steps {
  display: flex; flex-direction: column; gap: 10px; width: 100%; max-width: 320px;
}
.async-step-row {
  display: flex; align-items: center; gap: 10px; font-size: 13px;
  color: var(--color-text-tertiary); padding: 8px 12px; border-radius: var(--radius-md);
  transition: background 0.2s ease, color 0.2s ease;
}
.async-step-row--current {
  background: var(--color-primary-bg); color: var(--color-text-primary); font-weight: 500;
}
.async-step-icon { font-size: 16px; flex-shrink: 0; }
.async-step-icon--done { color: var(--color-success); }
.async-step-icon--current { color: var(--color-primary); }
.async-step-icon--fail { color: var(--color-danger); }
.async-step-label { flex: 1; text-align: left; }
.async-step-pending-dot {
  width: 8px; height: 8px; border-radius: 50%; background: var(--color-border);
  display: inline-block; flex-shrink: 0;
}

.result-banner {
  display: flex; align-items: center; padding: 16px 20px;
  border-radius: var(--radius-lg); border-left: 4px solid; margin-bottom: 24px; gap: 14px;
}
.result-banner--error { background: var(--color-danger-bg); border-color: var(--color-danger); }
.result-banner-icon { font-size: 28px; flex-shrink: 0; }
.result-banner-info { flex: 1; min-width: 0; }
.result-banner-title { font-size: 16px; font-weight: 700; }
.result-banner-meta {
  margin-top: 2px;
  color: var(--color-text-tertiary);
  font-size: 12px;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  word-break: break-word;
}
.result-score { font-size: 36px; font-weight: 800; line-height: 1; }

.result-section { margin-bottom: 24px; }
.result-section-title { font-size: 14px; font-weight: 600; color: var(--color-text-primary); margin: 0 0 12px; }
.rule-checks { display: flex; flex-direction: column; gap: 8px; }
.rule-check-item {
  display: flex; gap: 12px; padding: 12px 16px;
  border-radius: var(--radius-md); border: 1px solid var(--color-border-light);
}
.rule-check-item--pass {
  border-left: 3px solid var(--color-success);
  background: var(--color-success-bg);
}
.rule-check-item--fail {
  border-left: 3px solid var(--color-danger);
  background: var(--color-danger-bg);
}
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
.insight-card--suggest { background: linear-gradient(135deg, color-mix(in srgb, var(--color-primary) 4%, transparent), color-mix(in srgb, var(--color-primary) 1%, transparent)); border-color: color-mix(in srgb, var(--color-primary) 15%, transparent); }
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
.result-empty { text-align: center; padding: 40px 20px; }
.result-empty-icon {
  width: 64px; height: 64px; border-radius: 50%; background: var(--color-primary-bg);
  color: var(--color-primary); font-size: 28px; display: flex; align-items: center;
  justify-content: center; margin: 0 auto 16px;
}
.result-empty h4 { font-size: 16px; font-weight: 600; margin: 0 0 8px; }
.result-empty p { font-size: 13px; color: var(--color-text-tertiary); margin: 0 auto 16px; max-width: 280px; }
</style>
