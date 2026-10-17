<script setup lang="ts">
import { LikeOutlined, DislikeOutlined, MessageOutlined, EditOutlined, DeleteOutlined, UpOutlined } from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import type { AuditInteractions, AuditComment, Feedback } from '~/types/experience'
import { useExperienceApi } from '~/composables/useExperienceApi'
import { embedAuditPreviewKey } from '~/utils/embedAuditPreview'

const props = defineProps<{ auditId: string }>()
const { t } = useI18n()
const preview = inject(embedAuditPreviewKey, null)
const api = preview?.feedbackApi || useExperienceApi()
const data = ref<AuditInteractions | null>(null)
const loading = ref(false)
const saving = ref(false)
const error = ref(false)
const draft = ref('')
const selectedFeedback = ref<Feedback | null>(null)
const page = ref(1)
const editingId = ref<string | null>(null)
const composerOpen = ref(false)
const composerId = useId()
const inputRef = ref<{ focus: () => void } | null>(null)
let previousComposerOpen = false
let previousDraft = ''
let previousFeedback: Feedback | null = null
let request = 0

async function load() {
  const seq = ++request
  loading.value = true
  error.value = false
  try { const result = await api.interactions(props.auditId, page.value); if (seq === request) data.value = result }
  catch { if (seq === request) error.value = true }
  finally { if (seq === request) loading.value = false }
}
watch(() => props.auditId, () => { data.value = null; draft.value = ''; selectedFeedback.value = null; editingId.value = null; composerOpen.value = false; previousComposerOpen = false; previousDraft = ''; previousFeedback = null; page.value = 1; void load() }, { immediate: true })
onBeforeUnmount(() => { request++ })
async function toggleComposer() {
  if (saving.value) return
  composerOpen.value = !composerOpen.value
  if (composerOpen.value) { await nextTick(); inputRef.value?.focus() }
}
function vote(feedback: Feedback) {
  selectedFeedback.value = selectedFeedback.value === feedback ? null : feedback
}
async function send() {
  if (saving.value || !draft.value.trim() || !data.value?.can_interact) return
  saving.value = true
  try {
    const wasEditing = !!editingId.value
    if (editingId.value) await api.updateComment(props.auditId, editingId.value, draft.value.trim(), selectedFeedback.value)
    else await api.comment(props.auditId, draft.value.trim(), selectedFeedback.value)
    draft.value = ''
    selectedFeedback.value = null
    if (wasEditing) cancelEdit()
    else { page.value = Math.ceil(((data.value?.total || 0) + 1) / 20); composerOpen.value = false }
    await load()
    message.success(t(wasEditing ? 'experience.updated' : 'experience.sent'))
  } catch { message.error(t('experience.saveError')) }
  finally { saving.value = false }
}
async function startEdit(item: AuditComment) {
  if (saving.value || !item.can_manage) return
  if (!editingId.value) { previousDraft = draft.value; previousFeedback = selectedFeedback.value; previousComposerOpen = composerOpen.value }
  editingId.value = item.id; draft.value = item.content; selectedFeedback.value = item.feedback
  composerOpen.value = true
  await nextTick()
  inputRef.value?.focus()
}
function cancelEdit() {
  editingId.value = null; draft.value = previousDraft; selectedFeedback.value = previousFeedback
  composerOpen.value = previousComposerOpen
  previousDraft = ''; previousFeedback = null
}
async function removeComment(item: AuditComment) {
  if (saving.value || !item.can_manage) return
  saving.value = true
  try {
    await api.deleteComment(props.auditId, item.id)
    if (editingId.value === item.id) cancelEdit()
    page.value = Math.min(page.value, Math.max(1, Math.ceil(((data.value?.total || 1) - 1) / 20)))
    await load()
    message.success(t('experience.deleted'))
  } catch { message.error(t('experience.saveError')) }
  finally { saving.value = false }
}
function changePage(value: number) { page.value = value; void load() }
</script>

<template>
  <section class="feedback-panel" :aria-label="t('experience.embedTitle')">
    <div class="feedback-heading">
      <MessageOutlined aria-hidden="true" /><h3>{{ t('experience.embedTitle') }} <span class="comment-count">{{ data?.total || 0 }}</span></h3>
      <button type="button" class="compose-toggle" :aria-expanded="composerOpen" :aria-controls="composerId" :disabled="saving || loading || !data?.can_interact" @click="toggleComposer">
        <UpOutlined v-if="composerOpen" /><EditOutlined v-else />{{ t(composerOpen ? 'experience.collapseComposer' : 'experience.shareThoughts') }}
      </button>
    </div>
    <a-alert v-if="error" type="error" show-icon :message="t('experience.loadError')"><template #action><a-button size="small" @click="load">{{ t('experience.retry') }}</a-button></template></a-alert>
    <a-spin v-else :spinning="loading">
      <p v-if="!data?.items.length" class="comment-empty">{{ t('experience.commentEmpty') }}</p>
      <div v-else class="comment-thread" aria-live="polite">
        <article v-for="item in data.items" :key="item.id" class="comment-item">
          <span class="comment-avatar">{{ (item.username || '').slice(0, 1).toUpperCase() }}</span>
          <div class="comment-body"><div class="comment-meta"><strong>{{ item.username }}</strong><time>{{ formatDateTimeInAppZone(item.created_at) }}</time></div><a-tag v-if="item.feedback" :color="item.feedback === 'like' ? 'success' : 'warning'">{{ t(item.feedback === 'like' ? 'experience.agree' : 'experience.disagree') }}</a-tag><p v-if="item.content">{{ item.content }}</p>
            <div v-if="item.can_manage" class="comment-actions">
              <a-button type="text" size="small" :disabled="saving" @click="startEdit(item)"><EditOutlined />{{ t('experience.edit') }}</a-button>
              <a-popconfirm :title="t('experience.deleteConfirm')" :ok-text="t('experience.delete')" :cancel-text="t('experience.cancel')" :disabled="saving" @confirm="removeComment(item)"><a-button type="text" size="small" :disabled="saving"><DeleteOutlined />{{ t('experience.delete') }}</a-button></a-popconfirm>
            </div>
          </div>
        </article>
      </div>
      <div v-if="data && data.total > 20" class="pagination-wrapper"><a-pagination size="small" :current="page" :page-size="20" :total="data.total" :show-size-changer="false" @change="changePage" /></div>
      <p v-if="data && !data.can_interact" class="feedback-hint">{{ t('experience.identityRequired') }}</p>
      <Transition name="compose-reveal">
      <div v-if="composerOpen" :id="composerId" class="comment-compose">
        <div v-if="editingId" class="editing-notice"><span>{{ t('experience.editing') }}</span><a-button type="link" size="small" :disabled="saving" @click="cancelEdit">{{ t('experience.cancel') }}</a-button></div>
        <div class="satisfaction-row"><span class="satisfaction-label">{{ t('experience.satisfactionOptional') }}</span>
      <div class="feedback-votes">
        <a-button :type="selectedFeedback === 'like' ? 'primary' : 'default'" :aria-pressed="selectedFeedback === 'like'" :disabled="!data?.can_interact || saving || loading" @click="vote('like')"><LikeOutlined />{{ t('experience.agree') }}</a-button>
        <a-button :type="selectedFeedback === 'dislike' ? 'primary' : 'default'" :aria-pressed="selectedFeedback === 'dislike'" :disabled="!data?.can_interact || saving || loading" @click="vote('dislike')"><DislikeOutlined />{{ t('experience.disagree') }}</a-button>
      </div>
        </div>

        <a-textarea ref="inputRef" v-model:value="draft" :aria-label="t('experience.placeholder')" :placeholder="t('experience.placeholder')" :auto-size="{ minRows: 2, maxRows: 7 }" :maxlength="2000" show-count :disabled="!data?.can_interact || saving" />
        <div class="compose-footer"><span>{{ data?.can_interact ? t('experience.commentHint') : t('experience.identityRequired') }}</span><a-button class="comment-submit" type="primary" :loading="saving" :disabled="!data?.can_interact || !draft.trim() || loading" @click="send">{{ t(editingId ? 'experience.save' : 'experience.send') }}</a-button></div>
      </div>
      </Transition>
    </a-spin>
  </section>
</template>

<style scoped>
.feedback-panel { margin-top: 16px; padding: 16px; background: var(--color-bg-card); border: 1px solid var(--color-border-light); border-radius: 12px; }
.feedback-heading { display: flex; align-items: center; gap: 8px; color: var(--color-primary); }
.feedback-heading h3 { margin: 0; flex: 1; font-size: 14px; color: var(--color-text-primary); line-height: 1.5; }
.comment-count { display: inline-grid; place-items: center; min-width: 21px; height: 21px; margin-left: 5px; padding: 0 5px; border-radius: 6px; background: var(--color-bg-hover); color: var(--color-text-tertiary); font-size: 11px; font-weight: 500; vertical-align: middle; }
.compose-toggle { display: inline-flex; align-items: center; justify-content: center; flex-shrink: 0; gap: 5px; min-height: 32px; padding: 4px 8px; border: 0; border-radius: 7px; background: var(--color-primary-bg); color: var(--color-primary); cursor: pointer; font: inherit; font-size: 12px; transition: background 160ms ease; }
.compose-toggle:hover:not(:disabled) { background: color-mix(in srgb, var(--color-primary) 13%, var(--color-bg-card)); }
.compose-toggle:disabled { opacity: .5; cursor: default; }
.compose-toggle:focus-visible { outline: 2px solid var(--color-primary); outline-offset: 3px; }
.feedback-hint, .comment-empty { font-size: 12px; line-height: 1.7; color: var(--color-text-tertiary); }
.satisfaction-label { font-size: 12px; color: var(--color-text-secondary); }
.satisfaction-row { display: flex; align-items: center; flex-wrap: wrap; gap: 8px 12px; margin-bottom: 12px; }
.feedback-votes { display: flex; flex-wrap: wrap; gap: 6px; }
.feedback-votes :deep(button) { min-height: 30px !important; height: 30px; padding: 3px 10px !important; border-radius: 7px; font-size: 12px; box-shadow: none; }
.comment-heading { font-weight: 600; font-size: 13px; color: var(--color-text-secondary); }
.comment-heading span { margin-left: 6px; color: var(--color-text-tertiary); }
.comment-thread { display: flex; flex-direction: column; gap: 14px; margin-top: 14px; }
.comment-item { display: flex; gap: 10px; }
.comment-avatar { flex-shrink: 0; width: 30px; height: 30px; display: grid; place-items: center; border-radius: 10px; background: var(--color-primary-bg); color: var(--color-primary); }
.comment-body { flex: 1; min-width: 0; }
.comment-meta { display: flex; flex-wrap: wrap; gap: 6px 12px; font-size: 12px; color: var(--color-text-secondary); }
.comment-meta time { color: var(--color-text-tertiary); font-size: 11px; }
.comment-body p { margin: 4px 0 0; color: var(--color-text-primary); white-space: pre-wrap; overflow-wrap: anywhere; line-height: 1.75; font-size: 13px; }
.comment-compose { border-top: 1px solid var(--color-border-light); padding-top: 16px; margin-top: 18px; }
.compose-footer { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 8px; margin-top: 24px; }
.comment-submit.ant-btn { flex: 0 0 auto; width: auto; min-width: 88px; min-height: 34px !important; margin-left: auto; height: 34px; padding: 0 14px !important; border-radius: 8px; box-shadow: none; font-size: 13px; }
.comment-submit :deep(span) { flex: none; min-width: 0; }
.comment-actions { display: flex; gap: 4px; margin-top: 4px; }
.comment-actions :deep(.ant-btn) { color: var(--color-text-tertiary); font-size: 12px; min-height: 28px !important; padding: 2px 6px !important; }
.editing-notice { display: flex; align-items: center; justify-content: space-between; color: var(--color-primary); font-size: 12px; margin-bottom: 8px; }
.compose-footer > span { flex: 1; min-width: 140px; font-size: 11px; color: var(--color-text-tertiary); }
.pagination-wrapper { display: flex; justify-content: flex-end; margin-top: 16px; }
@media (max-width: 480px) { .feedback-panel { padding: 14px; } }
.comment-empty { margin: 12px 0 0; }
.compose-reveal-enter-active { transition: opacity 180ms ease, transform 180ms ease; }
.compose-reveal-enter-from { opacity: 0; transform: translateY(-5px); }
@media (prefers-reduced-motion: reduce) { .compose-reveal-enter-active { transition: none; } }
@media (max-width: 360px) { .feedback-heading { flex-wrap: wrap; } .feedback-heading h3 { font-size: 13px; } }
</style>
