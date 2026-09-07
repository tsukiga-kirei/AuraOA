<script setup lang="ts">
import { ArrowDownOutlined, ArrowRightOutlined, AppstoreOutlined, FileSearchOutlined, UnorderedListOutlined, BookOutlined } from '@ant-design/icons-vue'
import { buildJumpTurns } from '~/utils/chatJump'
import type { ChatMessageItem } from '~/types/chat'
import ChatThread from '~/components/Chat/ChatThread.vue'
import ChatComposer from '~/components/Chat/ChatComposer.vue'
import MessageJumpRail from '~/components/Chat/MessageJumpRail.vue'
definePageMeta({ layout: 'default', middleware: ['auth'] })
const { t } = useI18n()
const route = useRoute()
const { effectiveAgents, currentSessionId, currentDetail, selectedAgentCode, detailLoading, error, initialize, selectSession, createSession, newConversation } = useChatSession()
const messages = ref<ChatMessageItem[]>([])
const canvas = ref<HTMLElement | null>(null)
const canvasBody = ref<HTMLElement | null>(null)
const composer = ref<InstanceType<typeof ChatComposer> | null>(null)
const pinned = ref(true)
const agent = computed(() => {
  const code = currentDetail.value?.session.agent_code || selectedAgentCode.value
  return code ? effectiveAgents.value.find(item => item.agent_code === code) : effectiveAgents.value[0]
})
const { streaming, sendStreamMessage, stopStreaming } = useChatStream()
const BOTTOM_THRESHOLD = 36
let ignoreScroll = false
const distanceToBottom = () => {
  const el = canvas.value
  if (!el) return 0
  return el.scrollHeight - el.scrollTop - el.clientHeight
}
const scrollBottom = () => {
  const el = canvas.value
  if (!el) return
  ignoreScroll = true
  el.scrollTop = el.scrollHeight
  requestAnimationFrame(() => { ignoreScroll = false })
}
const followLatest = () => { if (pinned.value) nextTick(scrollBottom) }
const onCanvasScroll = () => {
  if (!canvas.value || ignoreScroll) return
  pinned.value = distanceToBottom() <= BOTTOM_THRESHOLD
}
watch(currentDetail, detail => { messages.value = [...(detail?.messages || [])]; pinned.value = true; nextTick(scrollBottom) })
watch(currentSessionId, () => stopStreaming())
watch(() => messages.value.map(item => [item.content, item.reasoning_content, item.tool_calls?.length, item.status, item.streaming]), followLatest, { deep: true })
watch(streaming, followLatest)
const selectFromRoute = async () => {
  const id = typeof route.query.session === 'string' ? route.query.session : null
  const agentCode = typeof route.query.agent === 'string' ? route.query.agent : ''
  if (id) {
    if (currentSessionId.value !== id) await selectSession(id)
    return
  }
  if (agentCode && selectedAgentCode.value !== agentCode) newConversation(agentCode)
}
watch(() => [route.query.session, route.query.agent], selectFromRoute)
onMounted(async () => {
  await initialize()
  await selectFromRoute()
  if (!canvasBody.value) return
  const observer = new ResizeObserver(followLatest)
  observer.observe(canvasBody.value)
  onBeforeUnmount(() => observer.disconnect())
})
const changeAgent = (code: string) => { stopStreaming(); newConversation(code); navigateTo({ path: '/chat', query: { agent: code } }) }
const submit = async (content: string) => {
  if (detailLoading.value || streaming.value) return
  if (!currentSessionId.value) {
    const created = await createSession(agent.value?.agent_code)
    if (!created) return
    await nextTick()
  }
  pinned.value = true
  await sendStreamMessage(currentSessionId.value!, content, messages)
}
const suggestions = [
  { icon: UnorderedListOutlined, title: 'chat.quickPill.todos', prompt: 'chat.prompt.todos', detail: 'chat.suggestion.todos' },
  { icon: FileSearchOutlined, title: 'chat.quickPill.summary', prompt: 'chat.prompt.summary', detail: 'chat.suggestion.summary' },
  { icon: BookOutlined, title: 'chat.quickPill.guideline', prompt: 'chat.prompt.guideline', detail: 'chat.suggestion.guideline' },
]
const jumpTurns = computed(() => buildJumpTurns(messages.value))
const jump = (id: string) => {
  pinned.value = false
  document.getElementById(id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}
</script>

<template>
  <div class="chat-workspace">
    <header class="workspace-heading">
      <div class="workspace-agent"><AppstoreOutlined />
        <a-select :value="agent?.agent_code" :bordered="false" :options="effectiveAgents.map(item => ({ value: item.agent_code, label: item.name }))" :aria-label="t('chat.agents')" @change="value => value && changeAgent(String(value))" />
      </div>
      <span class="workspace-context">{{ currentDetail?.session.title || t('chat.workspaceLabel') }}</span>
    </header>
    <div ref="canvas" class="chat-canvas" @scroll.passive="onCanvasScroll">
      <div ref="canvasBody" class="chat-canvas-body">
        <div v-if="error" class="chat-error" role="alert">{{ error }}</div>
        <div v-if="detailLoading" class="chat-loading"><a-spin /></div>
        <div v-else-if="!messages.length" class="chat-welcome">
          <div class="welcome-mark"><img src="/favicon.svg" alt="" width="34" height="34" /></div>
          <p class="welcome-eyebrow">{{ agent?.name || t('chat.assistantName') }}</p>
          <h1>{{ t('chat.welcomeTitle') }}</h1>
          <p class="welcome-description">{{ agent?.description || t('chat.welcomeDescription') }}</p>
          <div class="suggestions">
            <button v-for="item in suggestions" :key="item.title" @click="composer?.appendPrompt(t(item.prompt))">
              <component :is="item.icon" /><strong>{{ t(item.title) }}</strong><span>{{ t(item.detail) }}</span><ArrowRightOutlined class="suggestion-arrow" />
            </button>
          </div>
          <p v-if="!effectiveAgents.length" class="no-agents">{{ t('chat.noAgents') }}</p>
        </div>
        <ChatThread v-else :messages="messages" :agent-name="agent?.name" />
      </div>
    </div>
    <MessageJumpRail v-if="jumpTurns.length > 2" :turns="jumpTurns" @jump="jump" />
    <div class="chat-bottom">
      <button v-if="!pinned && messages.length" class="scroll-bottom" :aria-label="t('chat.scrollBottom')" @click="pinned = true; scrollBottom()"><ArrowDownOutlined /></button>
      <ChatComposer ref="composer" :submitting="streaming" :disabled="detailLoading || !agent" @submit="submit" @stop="stopStreaming" />
      <p class="composer-hint">{{ t('chat.composerHint') }}</p>
    </div>
  </div>
</template>

<style scoped>
.chat-workspace { flex:1; min-height:0; height:100%; display:flex; flex-direction:column; position:relative; background:var(--color-bg-card); color:var(--color-text-primary); }
.workspace-heading { flex-shrink:0; height:62px; padding:0 30px; display:flex; align-items:center; justify-content:space-between; gap:20px; }
.workspace-agent { display:flex; gap:4px; align-items:center; color:var(--color-text-secondary); }
.workspace-agent :deep(.ant-select) { min-width:155px; font-weight:600; }
.workspace-context { font-size:12px; color:var(--color-text-tertiary); overflow:hidden; white-space:nowrap; text-overflow:ellipsis; }
.chat-canvas { flex:1; min-height:0; overflow-y:auto; overflow-x:hidden; scrollbar-gutter:stable; }
.chat-canvas-body { min-height:100%; display:flex; flex-direction:column; }
.chat-welcome { width:min(780px,100%); margin:auto; padding:clamp(45px,10vh,110px) 30px 48px; }
.welcome-mark { margin-bottom:26px; }
.welcome-eyebrow { font-size:12px; color:var(--color-primary); font-weight:600; margin-bottom:12px; letter-spacing:.06em; }
h1 { font-size:clamp(27px,3vw,38px); font-weight:550; line-height:1.45; letter-spacing:-.035em; margin:0 0 14px; }
.welcome-description { font-size:14px; color:var(--color-text-secondary); line-height:1.85; max-width:580px; }
.suggestions { display:grid; grid-template-columns:repeat(3,minmax(0,1fr)); gap:12px; margin-top:34px; }
.suggestions button { position:relative; display:flex; flex-direction:column; align-items:flex-start; text-align:left; padding:20px 18px; border:1px solid var(--color-border-light); background:var(--color-bg-page); border-radius:14px; color:var(--color-text-secondary); cursor:pointer; transition:border-color .2s,transform .2s; }
.suggestions button:hover { border-color:var(--color-primary); transform:translateY(-2px); }
.suggestions strong { font-size:13px; margin:16px 0 6px; color:var(--color-text-primary); font-weight:550; }
.suggestions span { font-size:12px; line-height:1.65; }
.suggestions .suggestion-arrow { position:absolute; right:16px; top:20px; opacity:0; }.suggestions button:hover .suggestion-arrow { opacity:1; }
.chat-bottom { flex-shrink:0; position:relative; background:var(--color-bg-card); padding:8px 24px 10px; }
.composer-hint { text-align:center; font-size:11px; color:var(--color-text-tertiary); margin:6px 0 0; }
.scroll-bottom { position:absolute; top:-44px; left:calc(50% - 17px); width:34px; height:34px; background:var(--color-bg-card); border:1px solid var(--color-border); color:var(--color-text-secondary); border-radius:50%; cursor:pointer; }
.chat-error,.no-agents { color:var(--color-text-secondary); font-size:13px; padding:14px; background:var(--color-bg-page); border-radius:8px; margin:16px auto; max-width:740px; }
.chat-loading { padding:60px; text-align:center; }
@media(max-width:600px) { .workspace-heading { padding:0 14px; height:50px; }.workspace-context { display:none; }.chat-welcome { padding:30px 22px; }.suggestions { grid-template-columns:1fr; gap:8px; margin-top:24px; }.suggestions button { padding:14px; display:grid; grid-template-columns:24px 1fr; gap:4px 8px; }.suggestions strong { margin:0; }.suggestions span:not(.anticon) { grid-column:2; }.chat-bottom { padding:8px 12px 10px; }.welcome-mark { margin-bottom:18px; } }
@media(prefers-reduced-motion:reduce) { .suggestions button { transition:none; } }
</style>
