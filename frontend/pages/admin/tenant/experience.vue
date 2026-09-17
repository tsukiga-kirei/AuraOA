<script setup lang="ts">
import {
  SafetyCertificateOutlined,
  RobotOutlined,
  MessageOutlined,
  LikeOutlined,
  DislikeOutlined,
  ReloadOutlined,
  EyeOutlined,
  CheckCircleOutlined,
  CloseCircleOutlined,
  InfoCircleOutlined,
  SearchOutlined,
} from '@ant-design/icons-vue'
import type { AuditExperienceItem, AgentExperienceItem, AuditExperienceDetail, ExperienceQuery } from '~/types/experience'
import type { ChatMessageItem } from '~/types/chat'
import { renderSafeMarkdown } from '~/utils/markdown'
import { useExperienceApi } from '~/composables/useExperienceApi'

definePageMeta({ middleware: 'auth', layout: 'default' })
const { t } = useI18n()
const api = useExperienceApi()
const { getTenantSessionMessages } = useAdminDataApi()

const recommendationConfig = computed<Record<string, { color: string; bg: string; icon: any; label: string }>>(() => ({
  approve: { color: 'var(--color-success)', bg: 'var(--color-success-bg)', icon: CheckCircleOutlined, label: t('dashboard.rec.approve') },
  return: { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)', icon: CloseCircleOutlined, label: t('dashboard.rec.return') },
  review: { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)', icon: EyeOutlined, label: t('dashboard.rec.review') },
}))

const getScoreColorConfig = (score: number | undefined) => {
  if (score === undefined || score === null) return { color: 'var(--color-info)', bg: 'var(--color-info-bg)' }
  if (score < 60) return { color: 'var(--color-danger)', bg: 'var(--color-danger-bg)' }
  if (score > 80) return { color: 'var(--color-success)', bg: 'var(--color-success-bg)' }
  return { color: 'var(--color-warning)', bg: 'var(--color-warning-bg)' }
}
const tab = ref<'audit' | 'agents'>('audit')
const keyword = ref('')
const search = ref('')
const feedback = ref<ExperienceQuery['feedback']>()
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const audits = ref<AuditExperienceItem[]>([])
const agents = ref<AgentExperienceItem[]>([])
const loading = ref(false)
const error = ref(false)
const drawer = ref(false)
const detailLoading = ref(false)
const detailError = ref(false)
const selected = ref<AuditExperienceItem | AgentExperienceItem | null>(null)
const detailKind = ref<'audit' | 'agents'>('audit')
const detail = ref<AuditExperienceDetail | null>(null)
const messages = ref<ChatMessageItem[]>([])
const commentPage = ref(1)
let listRequest = 0
let detailRequest = 0
const filters = computed(() => [
  { value: '', label: t('experience.all') }, { value: 'like', label: t(tab.value === 'audit' ? 'experience.agree' : 'experience.likes') },
  { value: 'dislike', label: t(tab.value === 'audit' ? 'experience.disagree' : 'experience.dislikes') }, ...(tab.value === 'agents' ? [{ value: 'comments', label: t('experience.withComments') }] : []),
])
const columns = computed(() => [
  { title: t(tab.value === 'audit' ? 'experience.process' : 'experience.conversation'), key: 'subject' },
  { title: t('experience.interaction'), key: 'interaction', width: 250 },
  { title: t('experience.updatedAt'), key: 'updated_at', width: 170 },
  { title: t('experience.actions'), key: 'actions', width: 110 },
])
async function load() {
  const seq = ++listRequest
  loading.value = true; error.value = false
  const query = { keyword: search.value, feedback: feedback.value, page: page.value, page_size: pageSize.value }
  try {
    if (tab.value === 'audit') { const result = await api.listAudits(query); if (seq !== listRequest) return; audits.value = result.items; total.value = result.total; page.value = result.page }
    else { const result = await api.listAgents(query); if (seq !== listRequest) return; agents.value = result.items; total.value = result.total; page.value = result.page }
  } catch { if (seq === listRequest) { error.value = true; audits.value = []; agents.value = []; total.value = 0 } }
  finally { if (seq === listRequest) loading.value = false }
}
function filterChanged() { page.value = 1; void load() }
function applySearch() { search.value = keyword.value.trim(); filterChanged() }
watch(tab, () => { feedback.value = undefined; page.value = 1; void load() })
function changePage(next: number, size: number) { page.value = size === pageSize.value ? next : 1; pageSize.value = size; void load() }
async function loadDetail() {
  if (!selected.value) return
  const seq = ++detailRequest
  const item = selected.value
  detailLoading.value = true; detailError.value = false
  try {
    if (detailKind.value === 'audit') { const result = await api.auditDetail(item.id, commentPage.value); if (seq === detailRequest) detail.value = result }
    else { const result = await getTenantSessionMessages((item as AgentExperienceItem).session_id); if (seq === detailRequest) messages.value = result.messages }
  } catch { if (seq === detailRequest) detailError.value = true }
  finally { if (seq === detailRequest) detailLoading.value = false }
}
function openDetail(item: AuditExperienceItem | AgentExperienceItem) {
  selected.value = item; detailKind.value = tab.value; detail.value = null; messages.value = []; commentPage.value = 1; drawer.value = true; void loadDetail()
}
function changeCommentPage(value: number) { commentPage.value = value; void loadDetail() }
onMounted(load)
onBeforeUnmount(() => { listRequest++; detailRequest++ })
</script>

<template>
  <div class="experience-page">
    <header class="page-header"><h1>{{ t('experience.title') }}</h1><p>{{ t('experience.subtitle') }}</p></header>
    <div class="tab-nav" role="tablist" :aria-label="t('experience.title')">
      <button id="experience-audit-tab" type="button" role="tab" aria-controls="experience-list" :aria-selected="tab === 'audit'" :class="['tab-btn', { active: tab === 'audit' }]" @click="tab = 'audit'"><SafetyCertificateOutlined />{{ t('experience.audit') }}</button>
      <button id="experience-agents-tab" type="button" role="tab" aria-controls="experience-list" :aria-selected="tab === 'agents'" :class="['tab-btn', { active: tab === 'agents' }]" @click="tab = 'agents'"><RobotOutlined />{{ t('experience.agents') }}</button>
    </div>
    <section id="experience-list" class="experience-card" role="tabpanel" :aria-labelledby="`experience-${tab}-tab`">
      <div class="section-intro"><span class="section-icon"><MessageOutlined /></span><div><h2>{{ t(tab === 'audit' ? 'experience.auditTitle' : 'experience.agentTitle') }}</h2><p>{{ t(tab === 'audit' ? 'experience.auditHint' : 'experience.agentHint') }}</p></div></div>
      <div class="filter-bar">
        <a-input
            v-model:value="keyword"
            allow-clear
            :placeholder="t('experience.search')"
            :aria-label="t('experience.search')"
            class="search-input"
            @pressEnter="applySearch"
        >
          <template #prefix>
            <SearchOutlined class="search-prefix-icon" @click="applySearch" />
          </template>
        </a-input>
        <a-select :value="feedback || ''" :options="filters" :aria-label="t('experience.filter')" class="feedback-filter" @change="(value: any) => { feedback = value || undefined; filterChanged() }" />
        <a-button :loading="loading" @click="load"><ReloadOutlined />{{ t('experience.refresh') }}</a-button>
      </div>
      <a-alert v-if="error" type="error" show-icon :message="t('experience.loadError')"><template #action><a-button size="small" @click="load">{{ t('experience.retry') }}</a-button></template></a-alert>
      <a-table v-else :columns="columns" :data-source="tab === 'audit' ? audits : agents" row-key="id" :pagination="false" :loading="loading" :scroll="{ x: 800 }">
        <template #emptyText><a-empty :description="t('experience.empty')" /></template>
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'subject'">
            <button class="subject-title" type="button" @click="openDetail(record as AuditExperienceItem | AgentExperienceItem)">{{ record.title }}</button>
            <p class="subject-meta">{{ tab === 'audit' ? `${record.process_type} · ${record.process_id}` : `${record.agent_name} · ${record.username}` }}</p>
            <p v-if="tab === 'agents' && record.feedback_comment" class="feedback-excerpt">{{ record.feedback_comment }}</p>
          </template>
          <template v-else-if="column.key === 'interaction'">
            <div v-if="tab === 'audit'" class="interaction-counts"><span class="positive"><LikeOutlined />{{ record.like_count }}</span><span class="negative"><DislikeOutlined />{{ record.dislike_count }}</span><span><MessageOutlined />{{ record.comment_count }}</span></div>
            <a-tag v-else :color="record.feedback === 'like' ? 'success' : 'warning'"><LikeOutlined v-if="record.feedback === 'like'" /><DislikeOutlined v-else /> {{ t(record.feedback === 'like' ? 'experience.likes' : 'experience.dislikes') }}</a-tag>
          </template>
          <template v-else-if="column.key === 'updated_at'"><span class="subject-meta">{{ formatDateTimeInAppZone(record.updated_at) }}</span></template>
          <template v-else-if="column.key === 'actions'"><a-button type="link" size="small" @click="openDetail(record as AuditExperienceItem | AgentExperienceItem)"><EyeOutlined />{{ t('experience.view') }}</a-button></template>
        </template>
      </a-table>
      <div v-if="total" class="pagination-wrapper"><a-pagination :current="page" :page-size="pageSize" :total="total" show-size-changer :page-size-options="['10', '20', '50']" :show-total="(count: number) => t('experience.total', count)" @change="changePage" /></div>
    </section>
    <a-drawer v-model:open="drawer" :title="selected?.title" width="min(760px, 100vw)" :destroy-on-close="true">
      <a-spin :spinning="detailLoading">
        <a-alert v-if="detailError" type="error" show-icon :message="t('experience.loadError')"><template #action><a-button @click="loadDetail">{{ t('experience.retry') }}</a-button></template></a-alert>
        <template v-else-if="detailKind === 'audit' && detail">
          <div class="detail-summary">
            <SafetyCertificateOutlined class="summary-icon" />
            <strong>{{ t('experience.originalAudit') }}</strong>
            <span class="process-id-badge">{{ detail.audit_result.process_id }}</span>
          </div>
          <div class="audit-result-container">
            <!-- 审核结论横幅 -->
            <div
              class="result-banner"
              :style="{
                background: getScoreColorConfig(detail.audit_result.overall_score)?.bg,
                borderColor: getScoreColorConfig(detail.audit_result.overall_score)?.color,
              }"
            >
              <component
                :is="recommendationConfig[detail.audit_result.recommendation || 'review']?.icon"
                class="result-banner-icon"
                :style="{ color: getScoreColorConfig(detail.audit_result.overall_score)?.color }"
              />
              <div class="result-banner-info">
                <div
                  class="result-banner-title"
                  :style="{ color: getScoreColorConfig(detail.audit_result.overall_score)?.color }"
                >
                  {{ recommendationConfig[detail.audit_result.recommendation || 'review']?.label }}
                </div>
                <div class="result-banner-meta">
                  {{ t('dashboard.overallScore') }} {{ detail.audit_result.overall_score }}{{ t('dashboard.points') }}
                  <template v-if="detail.audit_result.confidence">
                    · {{ t('dashboard.confidence') }} {{ detail.audit_result.confidence }}%
                  </template>
                  <template v-if="detail.audit_result.duration_ms">
                    · {{ t('dashboard.duration') }} {{ (detail.audit_result.duration_ms / 1000).toFixed(1) }}s
                  </template>
                </div>
              </div>
              <div
                class="result-score"
                :style="{ color: getScoreColorConfig(detail.audit_result.overall_score)?.color }"
              >
                {{ detail.audit_result.overall_score }}
              </div>
            </div>

            <!-- 规则校验列表 -->
            <div v-if="detail.audit_result.rule_results?.length" class="result-section">
              <h4 class="result-section-title">{{ t('dashboard.ruleCheckDetail') }}</h4>
              <div class="rule-checks">
                <div
                  v-for="(rule, idx) in detail.audit_result.rule_results"
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

            <!-- 风险点 & 改进建议 -->
            <div v-if="detail.audit_result.risk_points?.length || detail.audit_result.suggestions?.length" class="risk-suggest-row">
              <div v-if="detail.audit_result.risk_points?.length" class="insight-card insight-card--risk">
                <div class="insight-card-header">
                  <CloseCircleOutlined style="color: var(--color-danger);" />
                  <span>{{ t('dashboard.riskPoints') }}</span>
                </div>
                <ul class="insight-card-list">
                  <li v-for="(rp, i) in detail.audit_result.risk_points" :key="i">{{ rp }}</li>
                </ul>
              </div>
              <div v-if="detail.audit_result.suggestions?.length" class="insight-card insight-card--suggest">
                <div class="insight-card-header">
                  <InfoCircleOutlined style="color: var(--color-primary);" />
                  <span>{{ t('dashboard.suggestions') }}</span>
                </div>
                <ul class="insight-card-list">
                  <li v-for="(sg, i) in detail.audit_result.suggestions" :key="i">{{ sg }}</li>
                </ul>
              </div>
            </div>

            <!-- AI 推理过程 -->
            <AiMarkdownStream
              v-if="detail.audit_result.ai_reasoning"
              :text="detail.audit_result.ai_reasoning"
              :title="t('dashboard.aiReasoning')"
              max-height="320px"
            />
          </div>

          <!-- 评论区域 -->
          <div class="detail-summary">
            <MessageOutlined class="summary-icon" />
            <strong>{{ t('experience.comments') }}</strong>
            <div class="interaction-counts">
              <span class="positive"><LikeOutlined />{{ detail.interactions.like_count }}</span>
              <span class="negative"><DislikeOutlined />{{ detail.interactions.dislike_count }}</span>
            </div>
          </div>
          <a-empty v-if="!detail.interactions.items.length" :description="t('experience.noComments')" />
          <article v-for="item in detail.interactions.items" :key="item.id" class="detail-message">
            <div class="message-meta">
              <span class="user-avatar-badge">{{ (item.username || '').slice(0, 1).toUpperCase() }}</span>
              <strong>{{ item.username }}</strong>
              <time>{{ formatDateTimeInAppZone(item.created_at) }}</time>
              <a-tag v-if="item.feedback" :color="item.feedback === 'like' ? 'success' : 'warning'">
                {{ t(item.feedback === 'like' ? 'experience.agree' : 'experience.disagree') }}
              </a-tag>
            </div>
            <p v-if="item.content" class="plain-content">{{ item.content }}</p>
          </article>
          <div v-if="detail.interactions.total > 20" class="pagination-wrapper">
            <a-pagination :current="commentPage" :page-size="20" :total="detail.interactions.total" :show-size-changer="false" @change="changeCommentPage" />
          </div>
        </template>
        <template v-else-if="detailKind === 'agents'">
          <p class="subject-meta">{{ t('experience.chatContext') }}</p>
          <article v-for="item in messages" :key="item.id" :class="['detail-message', { highlighted: item.id === selected?.id, 'user-message': item.role === 'user' }]">
            <div class="message-meta"><strong>{{ t(item.role === 'assistant' ? 'experience.assistant' : item.role === 'user' ? 'experience.user' : 'experience.system') }}</strong><time>{{ formatDateTimeInAppZone(item.created_at) }}</time><a-tag v-if="item.id === selected?.id" color="blue">{{ t('experience.selectedFeedback') }}</a-tag></div>
            <div class="markdown-content" v-html="renderSafeMarkdown(item.content || '')" />
            <div v-if="item.feedback" class="message-feedback"><a-tag :color="item.feedback === 'like' ? 'success' : 'warning'">{{ t(item.feedback === 'like' ? 'experience.likes' : 'experience.dislikes') }}</a-tag><p v-if="item.feedback_comment" class="plain-content">{{ item.feedback_comment }}</p></div>
          </article>
          <a-empty v-if="!messages.length && !detailLoading" :description="t('experience.empty')" />
        </template>
      </a-spin>
    </a-drawer>
  </div>
</template>

<style scoped>
.page-header { margin-bottom: 24px; }
.page-header h1 { margin: 0; font-size: 24px; font-weight: 700; color: var(--color-text-primary); }
.page-header p, .section-intro p { margin: 6px 0 0; font-size: 14px; color: var(--color-text-tertiary); }
.tab-nav { display: flex; gap: 4px; padding: 4px; margin-bottom: 24px; background: var(--color-bg-hover); border-radius: var(--radius-lg); width: fit-content; }
.tab-btn { display: flex; align-items: center; gap: 8px; padding: 10px 24px; border: none; border-radius: var(--radius-md); background: transparent; font-size: 14px; font-weight: 500; color: var(--color-text-secondary); cursor: pointer; transition: all var(--transition-fast); }
.tab-btn.active { background: var(--color-bg-card); color: var(--color-primary); box-shadow: var(--shadow-xs); }
.tab-btn:focus-visible, .subject-title:focus-visible { outline: 2px solid var(--color-primary); outline-offset: 3px; }
.experience-card { padding: 24px; background: var(--color-bg-card); border: 1px solid var(--color-border-light); border-radius: var(--radius-lg); }
.section-intro { display: flex; align-items: center; gap: 12px; margin-bottom: 24px; }
.section-icon { display: grid; place-items: center; width: 42px; height: 42px; border-radius: 12px; background: var(--color-primary-bg); color: var(--color-primary); font-size: 20px; flex-shrink: 0; }
.section-intro h2 { margin: 0; font-size: 16px; color: var(--color-text-primary); }
.section-intro p { font-size: 12px; line-height: 1.6; }
.filter-bar { display: flex; flex-wrap: wrap; gap: 12px; margin-bottom: 20px; }
.search-input { width: 360px; max-width: 100%; }
.search-input.ant-input-affix-wrapper,
.search-input :deep(.ant-input-affix-wrapper) {
  display: inline-flex;
  align-items: center;
  padding-inline: 12px;
}
.search-input :deep(.ant-input-prefix) {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  margin-inline-end: 8px;
}
.search-prefix-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-tertiary);
  font-size: 14px;
  line-height: 1;
  cursor: pointer;
}
.feedback-filter { min-width: 150px; }
.subject-title { display: block; padding: 0; border: 0; background: none; text-align: left; color: var(--color-text-primary); font: inherit; font-weight: 500; cursor: pointer; overflow-wrap: anywhere; }
.subject-title:hover { color: var(--color-primary); }
.subject-meta { margin: 6px 0 0; font-size: 12px; color: var(--color-text-tertiary); }
.feedback-excerpt { margin: 8px 0 0; color: var(--color-text-secondary); font-size: 12px; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; overflow-wrap: anywhere; }
.interaction-counts { display: flex; flex-wrap: wrap; align-items: center; gap: 18px; color: var(--color-text-secondary); }
.interaction-counts span { display: inline-flex; align-items: center; gap: 6px; }
.positive { color: var(--color-success); }.negative { color: var(--color-warning); }
.pagination-wrapper { display: flex; justify-content: flex-end; margin-top: 24px; }
.detail-summary { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; margin: 12px 0 16px; color: var(--color-text-primary); }
.summary-icon { font-size: 16px; color: var(--color-primary); }
.process-id-badge {
  display: inline-flex; align-items: center; padding: 2px 8px;
  background: var(--color-bg-hover); border-radius: 4px;
  font-size: 12px; font-family: monospace; color: var(--color-text-secondary);
}
.detail-summary > span:last-child { font-size: 12px; color: var(--color-text-tertiary); }

/* 审核结论容器 */
.audit-result-container {
  border: 1px solid var(--color-border-light);
  border-radius: var(--radius-lg);
  padding: 16px;
  margin-bottom: 24px;
  background: var(--color-bg-card);
}

/* 结果横幅 */
.result-banner {
  display: flex; align-items: center; padding: 14px 18px;
  border-radius: var(--radius-md); border-left: 4px solid; margin-bottom: 20px; gap: 12px;
}
.result-banner-icon { font-size: 26px; flex-shrink: 0; }
.result-banner-info { flex: 1; min-width: 0; }
.result-banner-title { font-size: 15px; font-weight: 700; line-height: 1.4; }
.result-banner-meta { font-size: 12px; color: var(--color-text-tertiary); margin-top: 2px; }
.result-score { font-size: 32px; font-weight: 800; line-height: 1; flex-shrink: 0; }

/* 规则校验 */
.result-section { margin-bottom: 20px; }
.result-section-title { font-size: 13px; font-weight: 600; color: var(--color-text-primary); margin: 0 0 10px; }
.rule-checks { display: flex; flex-direction: column; gap: 8px; }
.rule-check-item {
  display: flex; gap: 10px; padding: 10px 14px;
  border-radius: var(--radius-md); border: 1px solid var(--color-border-light);
  background: var(--color-bg-card); transition: background var(--transition-fast);
}
.rule-check-item:hover { background: var(--color-bg-hover); }
.rule-check-item--pass { border-left: 3px solid var(--color-success); }
.rule-check-item--fail { border-left: 3px solid var(--color-danger); background: var(--color-danger-bg); }
.rule-check-status { font-size: 16px; flex-shrink: 0; padding-top: 1px; }
.rule-check-content { flex: 1; min-width: 0; }
.rule-check-name { font-size: 13px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 4px; line-height: 1.4; }
.rule-check-reasoning { font-size: 12px; color: var(--color-text-secondary); line-height: 1.6; }

/* 风险 + 建议 */
.risk-suggest-row { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; margin-bottom: 20px; }
.risk-suggest-row:has(.insight-card:only-child) { grid-template-columns: 1fr; }
.insight-card { border-radius: var(--radius-md); padding: 14px; border: 1px solid var(--color-border-light); }
.insight-card--risk { background: linear-gradient(135deg, rgba(239, 68, 68, 0.04), rgba(239, 68, 68, 0.01)); border-color: rgba(239, 68, 68, 0.18); }
.insight-card--suggest { background: linear-gradient(135deg, color-mix(in srgb, var(--color-primary) 4%, transparent), color-mix(in srgb, var(--color-primary) 1%, transparent)); border-color: color-mix(in srgb, var(--color-primary) 18%, transparent); }
.insight-card-header { display: flex; align-items: center; gap: 6px; font-size: 13px; font-weight: 600; color: var(--color-text-primary); margin-bottom: 8px; }
.insight-card-list { margin: 0; padding-left: 18px; display: flex; flex-direction: column; gap: 4px; }
.insight-card-list li { font-size: 12px; line-height: 1.6; color: var(--color-text-secondary); }
.insight-card--risk .insight-card-list li { color: var(--color-danger); }

/* 评论卡片与头像 */
.detail-message { border: 1px solid var(--color-border-light); border-radius: var(--radius-lg); padding: 16px; margin-bottom: 18px; background: var(--color-bg-card); color: var(--color-text-primary); }
.user-avatar-badge {
  display: inline-flex; align-items: center; justify-content: center;
  width: 24px; height: 24px; border-radius: 6px;
  background: var(--color-primary-bg); color: var(--color-primary);
  font-size: 12px; font-weight: 600; flex-shrink: 0;
}
.detail-message.highlighted { border-color: var(--color-primary); background: var(--color-primary-bg); }.user-message { background: var(--color-bg-hover); }
.message-meta { display: flex; align-items: center; flex-wrap: wrap; gap: 8px 12px; font-size: 12px; }.message-meta time { color: var(--color-text-tertiary); }
.plain-content { white-space: pre-wrap; overflow-wrap: anywhere; line-height: 1.8; margin: 10px 0 0; }.message-feedback { border-top: 1px solid var(--color-border-light); padding-top: 12px; margin-top: 14px; }
.markdown-content { line-height: 1.8; overflow-wrap: anywhere; }.markdown-content :deep(pre) { overflow-x: auto; }.markdown-content :deep(img) { max-width: 100%; }
@media (max-width: 640px) { .experience-card { padding: 16px; }.filter-bar > * { width: 100%; }.pagination-wrapper { overflow-x: auto; justify-content: flex-start; }.risk-suggest-row { grid-template-columns: 1fr; } }
</style>
