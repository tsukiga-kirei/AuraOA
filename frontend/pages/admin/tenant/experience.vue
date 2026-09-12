<script setup lang="ts">
import { SafetyCertificateOutlined, RobotOutlined, MessageOutlined, LikeOutlined, DislikeOutlined, ReloadOutlined, EyeOutlined } from '@ant-design/icons-vue'
import type { AuditExperienceItem, AgentExperienceItem, AuditExperienceDetail, ExperienceQuery } from '~/types/experience'
import type { ChatMessageItem } from '~/types/chat'
import { renderSafeMarkdown } from '~/utils/markdown'
import { useExperienceApi } from '~/composables/useExperienceApi'

definePageMeta({ middleware: 'auth', layout: 'default' })
const { t } = useI18n()
const api = useExperienceApi()
const { getTenantSessionMessages } = useAdminDataApi()
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
        <a-input-search v-model:value="keyword" allow-clear :placeholder="t('experience.search')" :aria-label="t('experience.search')" class="search-input" @search="applySearch" />
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
          <div class="detail-summary"><SafetyCertificateOutlined /><strong>{{ t('experience.originalAudit') }}</strong><span>{{ detail.audit_result.process_id }}</span></div>
          <div class="audit-result">
            <a-tag>{{ t(`dashboard.rec.${detail.audit_result.recommendation || 'review'}`) }}</a-tag>
            <span>{{ t('dashboard.overallScore') }} {{ detail.audit_result.overall_score }}</span>
            <div v-for="(rule, index) in detail.audit_result.rule_results" :key="index" class="audit-rule"><strong>{{ rule.rule_content }}</strong><p>{{ rule.reason }}</p></div>
            <div v-if="detail.audit_result.risk_points?.length"><strong>{{ t('dashboard.riskPoints') }}</strong><ul><li v-for="(risk, index) in detail.audit_result.risk_points" :key="index">{{ risk }}</li></ul></div>
            <div v-if="detail.audit_result.suggestions?.length"><strong>{{ t('dashboard.suggestions') }}</strong><ul><li v-for="(suggestion, index) in detail.audit_result.suggestions" :key="index">{{ suggestion }}</li></ul></div>
            <AiMarkdownStream v-if="detail.audit_result.ai_reasoning" :text="detail.audit_result.ai_reasoning" :title="t('dashboard.aiReasoning')" max-height="320px" />
          </div>
          <div class="detail-summary"><MessageOutlined /><strong>{{ t('experience.comments') }}</strong><div class="interaction-counts"><span class="positive"><LikeOutlined />{{ detail.interactions.like_count }}</span><span class="negative"><DislikeOutlined />{{ detail.interactions.dislike_count }}</span></div></div>
          <a-empty v-if="!detail.interactions.items.length" :description="t('experience.noComments')" />
          <article v-for="item in detail.interactions.items" :key="item.id" class="detail-message"><div class="message-meta"><strong>{{ item.username }}</strong><time>{{ formatDateTimeInAppZone(item.created_at) }}</time></div><a-tag v-if="item.feedback" :color="item.feedback === 'like' ? 'success' : 'warning'">{{ t(item.feedback === 'like' ? 'experience.agree' : 'experience.disagree') }}</a-tag><p v-if="item.content" class="plain-content">{{ item.content }}</p></article>
          <div v-if="detail.interactions.total > 20" class="pagination-wrapper"><a-pagination :current="commentPage" :page-size="20" :total="detail.interactions.total" :show-size-changer="false" @change="changeCommentPage" /></div>
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
.feedback-filter { min-width: 150px; }
.subject-title { display: block; padding: 0; border: 0; background: none; text-align: left; color: var(--color-text-primary); font: inherit; font-weight: 500; cursor: pointer; overflow-wrap: anywhere; }
.subject-title:hover { color: var(--color-primary); }
.subject-meta { margin: 6px 0 0; font-size: 12px; color: var(--color-text-tertiary); }
.feedback-excerpt { margin: 8px 0 0; color: var(--color-text-secondary); font-size: 12px; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; overflow-wrap: anywhere; }
.interaction-counts { display: flex; flex-wrap: wrap; align-items: center; gap: 18px; color: var(--color-text-secondary); }
.interaction-counts span { display: inline-flex; align-items: center; gap: 6px; }
.positive { color: var(--color-success); }.negative { color: var(--color-warning); }
.pagination-wrapper { display: flex; justify-content: flex-end; margin-top: 24px; }
.detail-summary { display: flex; flex-wrap: wrap; align-items: center; gap: 10px; margin: 12px 0 18px; color: var(--color-text-primary); }
.detail-summary > span:last-child { font-size: 12px; color: var(--color-text-tertiary); }
.audit-result, .detail-message { border: 1px solid var(--color-border-light); border-radius: var(--radius-lg); padding: 16px; margin-bottom: 18px; background: var(--color-bg-card); color: var(--color-text-primary); }
.audit-rule { padding-top: 12px; margin-top: 12px; border-top: 1px solid var(--color-border-light); }.audit-rule p { margin: 6px 0 0; color: var(--color-text-secondary); }
.detail-message.highlighted { border-color: var(--color-primary); background: var(--color-primary-bg); }.user-message { background: var(--color-bg-hover); }
.message-meta { display: flex; align-items: center; flex-wrap: wrap; gap: 8px 12px; font-size: 12px; }.message-meta time { color: var(--color-text-tertiary); }
.plain-content { white-space: pre-wrap; overflow-wrap: anywhere; line-height: 1.8; margin: 10px 0 0; }.message-feedback { border-top: 1px solid var(--color-border-light); padding-top: 12px; margin-top: 14px; }
.markdown-content { line-height: 1.8; overflow-wrap: anywhere; }.markdown-content :deep(pre) { overflow-x: auto; }.markdown-content :deep(img) { max-width: 100%; }
@media (max-width: 640px) { .experience-card { padding: 16px; }.filter-bar > * { width: 100%; }.pagination-wrapper { overflow-x: auto; justify-content: flex-start; } }
</style>
