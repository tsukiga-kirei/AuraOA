<script setup lang="ts">
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  ReloadOutlined,
  InfoCircleOutlined,
  LoadingOutlined,
  WarningOutlined,
  ThunderboltOutlined,
  EyeOutlined,
  ExclamationCircleOutlined,
  UserOutlined,
  ApartmentOutlined,
  BulbOutlined,
  ExperimentOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import type { AuditResult } from '~/types/audit'
import type { EmbedContextResponse, EmbedProcessSummary } from '~/types/embed'
import { notifyParentEmbedStatus, waitForParentEmbedContext } from '~/composables/useEmbedParent'
import { embedAuditPreviewKey } from '~/utils/embedAuditPreview'

const preview = inject(embedAuditPreviewKey, null)
const auditPageId = useId()
useEmbedAuditScrollbars()

const { t } = useI18n()
const { getContext, executeEmbed } = useEmbedApi()
const { setupEmbedSession } = useEmbedSession()

type AuditProgressStep = NonNullable<AuditResult['progress_steps']>[number]

const DEFAULT_PROGRESS_STEPS: AuditProgressStep[] = [
  { key: 'pending', label: t('embed.auditProgress.pending'), done: false, current: true },
  { key: 'assembling', label: t('embed.auditProgress.assembling'), done: false, current: false },
  { key: 'reasoning', label: t('embed.auditProgress.reasoning'), done: false, current: false },
  { key: 'extracting', label: t('embed.auditProgress.extracting'), done: false, current: false },
]

const processId = ref('')
const pageLoading = ref(true)
const auditing = ref(false)
const context = ref<EmbedContextResponse | null>(null)
const currentResult = ref<AuditResult | null>(null)
const pageError = ref('')
const waitingParent = ref(true)
const eventSourceStream = ref<EventSource | null>(null)
const streamJobId = ref('')

// 双模视角控制：'standard' (标准审查) | 'personal' (我的定制视角)
const activePerspective = ref<'standard' | 'personal'>('standard')
const userManuallySwitchedPerspective = ref(false)

const processInfo = computed<EmbedProcessSummary | null>(() => context.value?.process ?? null)

const currentRuleDetails = computed(() => (currentResult.value?.rule_results || []).map((rule, index) => ({
  id: index,
  name: rule.rule_content,
  passed: rule.passed,
  reason: rule.reason,
})))

const ruleOutcomeSummary = computed(() => {
  if (!currentRuleDetails.value.length) return ''
  const failed = currentRuleDetails.value.filter(rule => !rule.passed).length
  return failed ? t('embed.auditFailedRuleSummary', [failed]) : t('embed.auditPassedRuleSummary', [currentRuleDetails.value.length])
})

const recommendationConfig = computed(() => ({
  approve: { color: 'var(--color-success)', bg: 'var(--color-success-bg)', icon: CheckCircleOutlined, label: t('dashboard.rec.approve') },
  review: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', icon: ExclamationCircleOutlined, label: t('dashboard.rec.review') },
  return: { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)', icon: CloseCircleOutlined, label: t('dashboard.rec.return') },
}))

const getScoreColorConfig = (score: number | undefined) => {
  if (score === undefined || score === null) return { color: 'var(--color-info)', bg: 'var(--color-info-bg)' }
  if (score >= 80) return { color: 'var(--color-success)', bg: 'var(--color-success-bg)' }
  if (score >= 60) return { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)' }
  return { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)' }
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
    color: 'var(--color-warning)',
    bg: 'var(--color-warning-bg)',
    icon: ExclamationCircleOutlined,
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
  const oldThinking = currentResult.value?.deep_thinking || ''
  currentResult.value = { ...currentResult.value, ...st } as AuditResult
  const nextReasoning = currentResult.value?.ai_reasoning || ''
  const nextThinking = currentResult.value?.deep_thinking || ''
  if (oldReasoning.length > nextReasoning.length && currentResult.value) {
    currentResult.value.ai_reasoning = oldReasoning
  }
  if (oldThinking.length > nextThinking.length && currentResult.value) {
    currentResult.value.deep_thinking = oldThinking
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
  if (preview) return
  if (!process.client || !auditResultId || streamJobId.value === auditResultId) return
  disconnectStream()
  streamJobId.value = auditResultId
  eventSourceStream.value = new EventSource(
    useEmbedAuth().appendEmbedTokenQuery(`/api/embed/stream/${encodeURIComponent(auditResultId)}`),
  )
  eventSourceStream.value.addEventListener('thinking', (event: any) => {
    if (!currentResult.value) return
    const chunk = event.data || ''
    const existing = currentResult.value.deep_thinking || ''
    if (!existing || chunk.startsWith(existing)) {
      currentResult.value.deep_thinking = chunk
    } else {
      currentResult.value.deep_thinking = existing + chunk
    }
  })
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
    deep_thinking: '',
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

const canUpgradeConfig = computed(() => {
  if (activePerspective.value === 'personal') {
    return false
  }
  return !!(context.value?.allow_latest_config_rerun && context.value?.config_upgrade_available)
})

// 规则发布与审核相关变化分开表达；后端 stale 不提供具体变化类型，不将其归因为内容修改。
const auditUpdateNotice = computed(() => {
  const rulesUpdated = activePerspective.value === 'standard' && !!context.value?.config_upgrade_available
  if (rulesUpdated) {
    return {
      label: t('embed.auditRulesUpdated'),
      hint: [t('embed.auditRulesUpdatedHint'), activeViewStale.value ? t('embed.auditInformationChangedHint') : '', canUpgradeConfig.value ? t('embed.auditLatestRulesActionHint') : ''].filter(Boolean).join(' '),
    }
  }
  return activeViewStale.value ? { label: t('embed.auditInformationChanged'), hint: t('embed.auditInformationChangedHint') } : null
})

function publishStatusToParent() {
  if (preview) return
  if (!process.client || !processId.value) return
  const r = currentResult.value
  const running = isAuditingActive.value
  notifyParentEmbedStatus({
    embed_type: 'audit',
    requestid: processId.value,
    running,
    has_result: !!(r && !running),
    status: r?.status,
    recommendation: r?.recommendation,
    overall_score: r?.overall_score,
    parse_error: !!r?.parse_error,
  })
}

async function runAudit(trigger: 'embed_auto' | 'embed_manual', useLatestConfig = false) {
  if (!processId.value || auditing.value) return
  disconnectStream()
  auditing.value = true
  currentResult.value = createPendingResult()
  publishStatusToParent()
  try {
    const result = preview ? await preview.execute(mergeAuditProgress, activePerspective.value === 'personal', useLatestConfig) : await executeEmbed(
      {
        process_id: processId.value,
        process_type: processInfo.value?.process_type,
        title: processInfo.value?.title,
        trigger_source: trigger,
        trigger_detail: trigger === 'embed_manual' ? 'manual' : 'visible_open',
        use_latest_config: useLatestConfig,
        perspective: activePerspective.value,
        oa_user_id: useEmbedAuth().getOAUserId() || undefined,
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
    message.error(extractEmbedErrorMessage(e, t('embed.auditFailed')))
    currentResult.value = null
    await refreshContext(false, true)
  } finally {
    auditing.value = false
    publishStatusToParent()
    disconnectStream()
  }
}

async function refreshContext(autoRun = true, preferCached = false) {
  if (!processId.value) return
  try {
    const resp = preview ? preview.context.value : await getContext(processId.value, preferCached)
    context.value = resp
    if (!userManuallySwitchedPerspective.value && resp.default_perspective) {
      activePerspective.value = resp.default_perspective as 'standard' | 'personal'
    }
    if (!auditing.value || !autoRun) {
      updateCurrentResultForPerspective()
    }
    if (autoRun && (resp.should_auto_audit || resp.running_job_id) && !auditing.value) {
      await runAudit('embed_auto')
    } else {
      publishStatusToParent()
    }
  } catch (e: any) {
    pageError.value = extractEmbedErrorMessage(e, t('embed.loadFailed'))
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
  const ms = isAuditingActive.value ? 0 : currentResult.value?.duration_ms
  if (ms) {
    const sec = ms / 1000
    const tone = sec >= 120 ? 'danger' : sec >= 60 ? 'warning' : 'success'
    return { text: t('embed.summary.duration', [getDurationSec(ms)]), tone }
  }
  if (isAuditingActive.value) {
    return { text: t('embed.auditRunning'), tone: 'running' }
  }
  return null
})

onMounted(async () => {
  if (preview) {
    processId.value = preview.context.value.process?.process_id || 'preview'
    context.value = preview.context.value
    updateCurrentResultForPerspective()
    pageLoading.value = false
    waitingParent.value = false
    return
  }
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
    await setupEmbedSession(parentCtx.embedToken, parentCtx.oaUserId)
  } catch (e: any) {
    pageError.value = extractEmbedErrorMessage(e, t('embed.missingToken'))
    pageLoading.value = false
    return
  }
  await bootstrap()
})

if (preview) {
  watch(preview.context, value => {
    context.value = value
    processId.value = value.process?.process_id || 'preview'
    if (!auditing.value) updateCurrentResultForPerspective()
  })
}

onBeforeUnmount(() => {
  disconnectStream()
})
</script>

<template>
  <div :id="auditPageId" class="embed-audit">
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
        <span>{{ t('embed.lastAuditAt') }}：{{ formatLastAuditAt(activeViewLastAuditAt) }}</span>
        <a-tag v-if="context?.config_version_no" color="blue" class="embed-config-version">{{ t('executionConfig.version', [context.config_version_no]) }}</a-tag>
        <a-tooltip v-if="auditUpdateNotice" :title="auditUpdateNotice.hint" :trigger="['hover', 'focus']">
          <span class="embed-update-notice" tabindex="0"><ThunderboltOutlined aria-hidden="true" />{{ auditUpdateNotice.label }}</span>
        </a-tooltip>
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
              :title="t('embed.perspective.standard')"
              :aria-label="t('embed.perspective.standard')"
              :aria-pressed="activePerspective === 'standard'"
              @click="handleSwitchPerspective('standard')"
            >
              <span class="perspective-title">{{ t('embed.perspective.standard') }}</span>
              <span v-if="context.has_audit" class="perspective-dot" :title="t('embed.perspective.standardHasAudit')" />
            </button>
            <button
              type="button"
              class="perspective-nav-item"
              :class="{ 'perspective-nav-item--active': activePerspective === 'personal' }"
              :aria-pressed="activePerspective === 'personal'"
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
                    <UserOutlined v-if="row === processMetaRows[0]" aria-hidden="true" />
                    <ApartmentOutlined v-else aria-hidden="true" />
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
                <template v-if="canUpgradeConfig">
                  <a-dropdown :trigger="['click']" placement="bottomRight" :disabled="auditing || (!!context.running_job_id && !auditing)">
                    <a-button
                      class="embed-process-card__action"
                      type="text"
                      size="small"
                      :disabled="auditing || (!!context.running_job_id && !auditing)"
                    >
                      <LoadingOutlined v-if="auditing" spin />
                      <ReloadOutlined v-else />
                      <span>{{ t('dashboard.reAudit') }}</span>
                    </a-button>
                    <template #overlay>
                      <a-menu class="embed-action-dropdown-menu">
                        <a-menu-item key="latest" @click="handleReAuditWithLatestConfig">
                          <div class="embed-menu-simple-item">
                            <ThunderboltOutlined class="embed-menu-simple-icon embed-menu-simple-icon--primary" />
                            <span class="embed-menu-simple-text">{{ t('executionConfig.ruleLatestAudit') }}</span>
                          </div>
                        </a-menu-item>
                        <a-menu-item key="current" @click="handleReAudit">
                          <div class="embed-menu-simple-item">
                            <ReloadOutlined class="embed-menu-simple-icon" />
                            <span class="embed-menu-simple-text">
                              {{ context?.config_version_no ? t('executionConfig.ruleCurrentAuditWithVer', ['v' + context.config_version_no]) : t('executionConfig.ruleCurrentAudit') }}
                            </span>
                          </div>
                        </a-menu-item>
                      </a-menu>
                    </template>
                  </a-dropdown>
                </template>
                <a-button
                  v-else
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
          <div v-if="currentResult?.deep_thinking" class="embed-reasoning-live">
            <AiMarkdownStream
              page-flow
              :loading="isAuditingActive"
              :title="t('dashboard.deepThinking', '深度思考过程')"
              :text="currentResult.deep_thinking || ''"
              max-height="260px"
            ><template #icon><BulbOutlined /></template></AiMarkdownStream>
          </div>
          <div v-if="currentResult?.ai_reasoning" class="embed-reasoning-live">
            <AiMarkdownStream
              page-flow
              :loading="isAuditingActive"
              :title="t('dashboard.aiReasoning')"
              :text="currentResult.ai_reasoning || ''"
              max-height="260px"
            ><template #icon><ExperimentOutlined /></template></AiMarkdownStream>
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

          <div v-else :key="`result-${currentResult.id || currentResult.process_id}-${activePerspective}`" class="audit-result-content">
            <div
              class="result-banner"
              :style="{
                background: recommendationConfig[currentResult.recommendation || 'review']?.bg,
                borderColor: recommendationConfig[currentResult.recommendation || 'review']?.color,
              }"
            >
              <component
                :is="recommendationConfig[currentResult.recommendation || 'review']?.icon"
                class="result-banner-icon"
                aria-hidden="true"
                :style="{ color: recommendationConfig[currentResult.recommendation || 'review']?.color }"
              />
              <div class="result-banner-info">
                <div
                  class="result-banner-title"
                  :style="{ color: recommendationConfig[currentResult.recommendation || 'review']?.color }"
                >
                  {{ recommendationConfig[currentResult.recommendation || 'review']?.label }}
                </div>
                <div class="result-banner-meta result-banner-meta--summary">
                  <span v-if="ruleOutcomeSummary">{{ ruleOutcomeSummary }} · </span>{{ t('dashboard.confidence') }} {{ currentResult.confidence }}%
                </div>
              </div>
              <div class="result-score-block">
                <div class="result-score" :style="{ color: getScoreColorConfig(currentResult.overall_score)?.color }">{{ currentResult.overall_score ?? '—' }}</div>
                <div class="result-score-label">{{ t('dashboard.overallScore') }}</div>
              </div>
            </div>

            <ResultRuleDetails
              embedded
              v-if="currentRuleDetails.length"
              :key="`rules-${currentResult.id || currentResult.process_id}-${activePerspective}`"
              :title="t('dashboard.ruleCheckDetail')"
              :rules="currentRuleDetails"
            />

            <div v-if="currentResult.risk_points?.length || currentResult.suggestions?.length" class="risk-suggest-row">
              <div v-if="currentResult.risk_points?.length" class="insight-card insight-card--risk">
                <div class="insight-card-header">
                  <span class="insight-card-marker"><WarningOutlined aria-hidden="true" /></span>
                  <h4>{{ t('dashboard.riskPoints') }}</h4>
                </div>
                <ul class="insight-card-list">
                  <li v-for="(rp, i) in currentResult.risk_points" :key="i">{{ rp }}</li>
                </ul>
              </div>
              <div v-if="currentResult.suggestions?.length" class="insight-card insight-card--suggest">
                <div class="insight-card-header">
                  <span class="insight-card-marker"><InfoCircleOutlined aria-hidden="true" /></span>
                  <h4>{{ t('dashboard.suggestions') }}</h4>
                </div>
                <ul class="insight-card-list">
                  <li v-for="(sg, i) in currentResult.suggestions" :key="i">{{ sg }}</li>
                </ul>
              </div>
            </div>

            <ResultReasoningPanel
              page-flow
              v-if="currentResult.deep_thinking"
              :key="`thinking-${currentResult.id || currentResult.process_id}-${activePerspective}`"
              :title="t('dashboard.deepThinking', '深度思考过程')"
              :text="currentResult.deep_thinking || ''"
              max-height="320px"
            ><template #icon><BulbOutlined /></template></ResultReasoningPanel>

            <ResultReasoningPanel
              page-flow
              v-if="currentResult.ai_reasoning"
              :key="`reasoning-${currentResult.id || currentResult.process_id}-${activePerspective}`"
              :title="t('dashboard.aiReasoning')"
              :text="currentResult.ai_reasoning || ''"
              max-height="320px"
            ><template #icon><ExperimentOutlined /></template></ResultReasoningPanel>
          </div>
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

        <AuditFeedbackPanel v-if="currentResult?.id && currentResult.status === 'completed' && !isAuditingActive" :key="currentResult.id" :audit-id="currentResult.id" />
      </template>
    </template>
    <EmbedAuditScrollbar :content-id="auditPageId" />
  </div>
</template>

<style scoped>
.embed-audit { width: 100%; max-width: 100%; margin: 0; box-sizing: border-box; min-height: 100vh; padding: 14px 18px 24px; }

.embed-header {
  margin-bottom: 14px; padding-bottom: 12px;
  border-bottom: 1px solid var(--color-border-light);
  display: flex; flex-wrap: wrap; align-items: center; justify-content: space-between; gap: 6px 16px;
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
  display: flex; flex-wrap: wrap; align-items: center; gap: 6px 10px;
}
.embed-config-version { margin: 0; white-space: nowrap; }
.embed-update-notice { display: inline-flex; align-items: center; gap: 5px; padding: 3px 8px; border: 1px solid color-mix(in srgb, var(--color-primary) 12%, transparent); border-radius: 6px; background: var(--color-primary-bg); color: var(--color-primary); font-size: 11px; line-height: 1.5; }
.embed-update-notice:focus-visible { outline: 2px solid var(--color-primary); outline-offset: 2px; }
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
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: flex-start;
  gap: 4px;
  width: fit-content;
  max-width: 100%;
  padding: 3px;
  border-radius: var(--radius-md, 8px);
  background: var(--color-bg-hover, #f1f5f9);
}
.perspective-nav-item {
  flex: 0 1 auto;
  min-width: 0;
  max-width: 100%;
  min-height: 34px;
  display: inline-flex;
  align-items: center;
  justify-content: flex-start;
  gap: 6px;
  padding: 6px 10px;
  border: 0;
  background: transparent;
  border-radius: var(--radius-sm, 6px);
  text-align: left;
  font-family: inherit;
  line-height: 1.5;
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: background 160ms ease, border-color 160ms ease, color 160ms ease;
}
.perspective-nav-item:hover:not(.perspective-nav-item--active) {
  color: var(--color-text-primary);
  background: var(--color-bg-hover);
}
.perspective-nav-item--active {
  background: var(--color-bg-card, #ffffff);
  color: var(--color-primary, #1890ff);
  font-weight: 600;
  box-shadow: 0 1px 3px rgb(0 0 0 / 8%);
}
.perspective-title { min-width: 0; overflow-wrap: anywhere; }
.perspective-dot {
  flex-shrink: 0;
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
  flex-shrink: 0;
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
  display: flex; flex-wrap: wrap;
  gap: 12px 16px; align-items: center;
}
.embed-process-card__copy { min-width: min(100%, 280px); flex: 1; display: flex; flex-wrap: wrap; gap: 7px 8px; }
.embed-process-card__title {
  min-width: 0; margin: 0 0 10px;
  font-size: 15px; font-weight: 700; line-height: 1.5;
  color: var(--color-text-primary);
  word-break: keep-all; overflow-wrap: anywhere;
  white-space: normal;
}
.embed-process-card__actions {
  display: flex; flex-wrap: wrap; align-items: center; justify-content: flex-end; gap: 8px;
  flex-shrink: 0;
  min-width: 0;
  margin-left: auto;
}
.embed-process-card__stat {
  display: inline-flex; align-items: center; justify-content: center;
  width: auto; height: 28px; padding: 0 7px; border-radius: var(--radius-md);
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
  min-height: 32px !important; font-size: 12px !important; line-height: 28px !important;
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
:global(.embed-action-dropdown-menu) {
  min-width: 148px;
  padding: 4px;
  border-radius: var(--radius-md);
  box-shadow: 0 4px 14px 0 rgba(0, 0, 0, 0.08), 0 2px 6px -2px rgba(0, 0, 0, 0.06);
}
:global(.embed-menu-simple-item) {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 4px;
}
:global(.embed-menu-simple-icon) {
  font-size: 14px;
  flex-shrink: 0;
}
:global(.embed-menu-simple-icon--primary) {
  color: var(--color-primary);
}
:global(.embed-menu-simple-text) {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text-primary);
  white-space: nowrap;
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
  max-width: 100%;
}
.embed-process-card__meta-chip {
  display: inline-flex;
  align-items: center;
  width: fit-content;
  max-width: 100%;
  min-width: 0;
  min-height: 27px;
  gap: 6px;
  padding: 4px 9px;
  border-radius: 7px;
  border: 1px solid color-mix(in srgb, var(--color-border-light) 82%, var(--color-primary) 18%);
  background: color-mix(in srgb, var(--color-bg-hover) 78%, var(--color-bg-card));
  color: var(--color-text-secondary);
  font-size: 13px;
  font-weight: 500;
  line-height: 1.5;
  overflow-wrap: anywhere;
}
.embed-process-card__meta-row:first-of-type .embed-process-card__meta-chip:first-child {
  color: var(--color-text-primary);
  background: color-mix(in srgb, var(--color-primary-bg) 54%, var(--color-bg-card));
  border-color: color-mix(in srgb, var(--color-primary) 22%, var(--color-border-light));
}

.embed-auditing-center {
  display: flex; flex-direction: column; align-items: center; justify-content: flex-start;
  min-height: calc(100vh - 200px); padding: 20px 0; text-align: center; width: 100%; box-sizing: border-box;
}
.embed-auditing-title {
  margin: 16px 0 20px; font-size: 14px; font-weight: 500;
  color: var(--color-text-secondary);
}
.embed-reasoning-live {
  width: 100%; margin-top: 10px; text-align: left;
}
.embed-reasoning-live:first-of-type {
  margin-top: 18px;
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
  border-radius: 12px; border-left: 3px solid; margin-bottom: 20px; gap: 12px;
}
.result-banner--error { background: var(--color-danger-bg); border-color: var(--color-danger); }
.result-banner-icon { font-size: 28px; flex-shrink: 0; }
.result-banner-info { flex: 1; min-width: 0; }
.result-banner-title { font-size: 16px; font-weight: 600; line-height: 1.5; }
.result-banner-meta {
  margin-top: 2px;
  color: var(--color-text-tertiary);
  font-size: 12px;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
  word-break: break-word;
}
.result-banner-meta--summary { margin-top: 4px; color: var(--color-text-secondary); line-height: 1.7; }
.result-score-block { flex-shrink: 0; text-align: right; padding-left: 8px; }
.result-score { font-size: 36px; font-weight: 700; line-height: 1; font-variant-numeric: tabular-nums; letter-spacing: -.03em; }
.result-score-label { margin-top: 6px; color: var(--color-text-secondary); font-size: 10px; line-height: 1.4; }
.audit-result-content > .result-banner { animation: audit-result-appear 180ms cubic-bezier(.2,.8,.2,1); }
@keyframes audit-result-appear { from { opacity: 0; transform: translateY(4px); } to { opacity: 1; transform: translateY(0); } }

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

.risk-suggest-row { display: grid; grid-template-columns: minmax(0, 1fr); gap: 12px; margin-bottom: 18px; }
@media (min-width: 640px) {
  .risk-suggest-row { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .risk-suggest-row:has(.insight-card:only-child) { grid-template-columns: 1fr; }
  .risk-suggest-row > .insight-card:only-child { grid-column: 1 / -1; }
}
.insight-card { min-width: 0; border-radius: 10px; padding: 15px 16px; border: 1px solid var(--color-border); background: var(--color-bg-card); transition: border-color 160ms ease; }
.insight-card--risk { --insight-accent: var(--color-danger); --insight-tint: var(--color-danger-bg); }
.insight-card--suggest { --insight-accent: var(--color-primary); --insight-tint: var(--color-primary-bg); }
.insight-card-header { display: flex; align-items: center; gap: 8px; color: var(--color-text-primary); min-height: 26px; margin-bottom: 10px; }
.insight-card-header h4 { margin: 0; font-family: inherit; font-size: 14px; font-weight: 600; line-height: 1.5; }
.insight-card-marker { display: inline-flex; flex-shrink: 0; align-items: center; justify-content: center; width: 26px; height: 26px; border-radius: 7px; color: var(--insight-accent); background: var(--insight-tint); font-size: 18px; }
.insight-card-list { margin: 0; padding-left: 18px; display: flex; flex-direction: column; gap: 6px; }
.insight-card-list li { font-size: 13px; line-height: 1.8; color: var(--color-text-secondary); overflow-wrap: anywhere; }
.insight-card-list li::marker { color: var(--color-text-tertiary); }
@media (hover: hover) and (pointer: fine) { .insight-card:hover { border-color: color-mix(in srgb, var(--insight-accent) 22%, var(--color-border)); } }

.result-empty { text-align: center; padding: 40px 20px; }
.result-empty-icon {
  width: 64px; height: 64px; border-radius: 50%; background: var(--color-primary-bg);
  color: var(--color-primary); font-size: 28px; display: flex; align-items: center;
  justify-content: center; margin: 0 auto 16px;
}
.result-empty h4 { font-size: 16px; font-weight: 600; margin: 0 0 8px; }
.result-empty p { font-size: 13px; color: var(--color-text-tertiary); margin: 0 auto 16px; max-width: 280px; }

.embed-process-card__meta-chip :deep(.anticon) { flex-shrink: 0; color: var(--color-text-tertiary); font-size: 12px; }
.embed-process-card__actions .embed-process-card__action.ant-btn { width: auto; }
.embed-audit :deep(.result-rule-details__header) { flex-wrap: wrap; }
.embed-audit :deep(.result-rule-details__summary) { gap: 4px; }
.embed-audit :deep(.result-rule-details__item) { border-radius: 10px; }
.audit-result-content :deep(.result-rule-details) { margin-bottom: 18px; }
.audit-result-content :deep(.result-rule-details__title) { font-family: inherit; font-weight: 600; }
.embed-audit :deep(.result-reasoning-panel__toggle) { min-height: 50px; }
.embed-audit :deep(.markdown-body table) { display: block; max-width: 100%; overflow-x: auto; }
.embed-audit :deep(.markdown-body pre) { max-width: 100%; }
.embed-audit :deep(button:focus-visible) { outline: 2px solid var(--color-primary); outline-offset: 2px; }
.embed-audit :deep(.result-reasoning-panel__toggle:focus-visible) { outline-offset: -3px; }
@media (max-width: 640px) {
  .embed-audit { padding: 12px 12px 24px; }
  .embed-header { align-items: flex-start; flex-direction: column; gap: 5px; }
  .embed-title { font-size: 16px; }
  .perspective-nav-item { min-height: 40px; }
  .embed-process-card__body { padding: 13px; }
  .embed-process-card__actions { width: 100%; padding: 10px 0 0; border-top: 1px solid var(--color-border-light); gap: 8px; }
  .embed-process-card__stat { margin-right: auto; }
  .result-banner { padding: 14px; }
  .result-score { font-size: 32px; }
  .insight-card { padding: 14px; }
  .embed-auditing-center { padding: 14px 0; min-height: 280px; }
  .embed-audit :deep(.result-rule-details__title) { font-size: 14px; }
}
@media (prefers-reduced-motion: reduce) {
  .embed-audit, .embed-audit :deep(*) { animation: none !important; transition: none !important; }
}

/* 限定到审核文档，避免影响共用 embed 布局的流程总结页或 OA 父页。 */
:global(html.embed-audit-document) { scroll-behavior: auto; scrollbar-gutter: auto; overflow-y: auto; background: var(--color-bg-page, #f4f6fb); scrollbar-width: none; }
:global(html.embed-audit-document body) { overflow: visible; background: var(--color-bg-page, #f4f6fb); }
:global(html.embed-audit-document #__nuxt) { background: var(--color-bg-page, #f4f6fb); }
:global(html.embed-audit-document::-webkit-scrollbar) { display: none; width: 0; height: 0; }
:global(html.embed-audit-document body),
:global(html.embed-audit-document *) { scrollbar-width: thin; scrollbar-color: transparent transparent; }
:global(html.embed-audit-document [data-audit-scrolling]) { scrollbar-color: color-mix(in srgb, var(--color-text-tertiary) 65%, transparent) transparent; }
:global(html.embed-audit-document *::-webkit-scrollbar) { width: 6px; height: 6px; }
:global(html.embed-audit-document *::-webkit-scrollbar-track) { background: transparent; }
:global(html.embed-audit-document *::-webkit-scrollbar-thumb) { background: transparent; border-radius: 999px; }
:global(html.embed-audit-document [data-audit-scrolling]::-webkit-scrollbar-thumb) { background: color-mix(in srgb, var(--color-text-tertiary) 65%, transparent); }
</style>
