<script setup lang="ts">
import {
  ArrowDownOutlined,
  ArrowRightOutlined,
  AppstoreOutlined,
  FileSearchOutlined,
  UnorderedListOutlined,
  BookOutlined,
  ThunderboltOutlined,
  QuestionCircleOutlined,
  CheckCircleOutlined,
  SearchOutlined,
  HourglassOutlined,
  ClockCircleOutlined,
  EditOutlined,
  BarChartOutlined,
  BulbOutlined,
  FileTextOutlined,
} from '@ant-design/icons-vue'
import { buildJumpTurns } from '~/utils/chatJump'
import type { ChatMessageItem } from '~/types/chat'
import ChatThread from '~/components/Chat/ChatThread.vue'
import ChatComposer from '~/components/Chat/ChatComposer.vue'
import MessageJumpRail from '~/components/Chat/MessageJumpRail.vue'
definePageMeta({ layout: 'default', middleware: ['auth'] })
const { t } = useI18n()
const route = useRoute()
const { effectiveAgents, currentSessionId, currentDetail, selectedAgentCode, detailLoading, error, initialize, selectSession, createSession, newConversation } = useChatSession()
const messages = ref<ChatMessageItem[]>([...(currentDetail.value?.messages || [])])
const canvas = ref<HTMLElement | null>(null)
const canvasBody = ref<HTMLElement | null>(null)
const composer = ref<InstanceType<typeof ChatComposer> | null>(null)
const welcomeComposer = ref<InstanceType<typeof ChatComposer> | null>(null)
const isSubmittingFirst = ref(false)
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
watch(currentDetail, detail => {
  messages.value = [...(detail?.messages || [])]
  pinned.value = true
  nextTick(scrollBottom)
}, { immediate: true })
watch(currentSessionId, () => stopStreaming())
watch(() => messages.value.map(item => [item.content, item.reasoning_content, item.tool_calls?.length, item.status, item.streaming]), followLatest, { deep: true })
watch(streaming, followLatest)
const selectFromRoute = async () => {
  const rawId = route.query.session ?? route.query.session_id
  const id = typeof rawId === 'string' && rawId.trim() ? rawId.trim() : null
  const agentCode = typeof route.query.agent === 'string' ? route.query.agent.trim() : ''
  if (id) {
    if (currentSessionId.value !== id || !currentDetail.value || currentDetail.value.session.id !== id) {
      await selectSession(id)
    }
    return
  }
  if (agentCode && selectedAgentCode.value !== agentCode) newConversation(agentCode)
}
watch(() => [route.query.session, route.query.session_id, route.query.agent], selectFromRoute)
onMounted(async () => {
  await initialize()
  await selectFromRoute()
  if (!canvasBody.value) return
  const observer = new ResizeObserver(followLatest)
  observer.observe(canvasBody.value)
  onBeforeUnmount(() => observer.disconnect())
})
const changeAgent = (code: string) => { stopStreaming(); newConversation(code); navigateTo({ path: '/chat', query: { agent: code } }) }

const handleSelectSuggestion = (prompt: string) => {
  if (welcomeComposer.value) {
    welcomeComposer.value.appendPrompt(prompt)
  } else if (composer.value) {
    composer.value.appendPrompt(prompt)
  }
}

const submit = async (content: string) => {
  if (detailLoading.value || streaming.value) return
  const isFirstMessage = !messages.value.length
  if (isFirstMessage) {
    isSubmittingFirst.value = true
    await new Promise(r => setTimeout(r, 160))
  }
  if (!currentSessionId.value) {
    const created = await createSession(agent.value?.agent_code)
    if (!created) {
      isSubmittingFirst.value = false
      return
    }
    await nextTick()
  }
  pinned.value = true
  isSubmittingFirst.value = false
  await sendStreamMessage(currentSessionId.value!, content, messages)
}
const iconMap: Record<string, any> = {
  // Component names
  UnorderedListOutlined,
  FileSearchOutlined,
  BookOutlined,
  ThunderboltOutlined,
  QuestionCircleOutlined,
  CheckCircleOutlined,
  SearchOutlined,
  HourglassOutlined,
  ClockCircleOutlined,
  EditOutlined,
  BarChartOutlined,
  BulbOutlined,
  FileTextOutlined,

  // Lowercase & short aliases
  clipboard: UnorderedListOutlined,
  todolist: UnorderedListOutlined,
  search: FileSearchOutlined,
  hourglass: HourglassOutlined,
  clock: ClockCircleOutlined,
  edit: EditOutlined,
  barchart: BarChartOutlined,
  chart: BarChartOutlined,
  lightbulb: BulbOutlined,
  bulb: BulbOutlined,
  book: BookOutlined,
  thunderbolt: ThunderboltOutlined,
  question: QuestionCircleOutlined,
  check: CheckCircleOutlined,
  file: FileTextOutlined,
}

const getQuickIcon = (iconName?: string) => {
  if (!iconName) return FileSearchOutlined
  return iconMap[iconName] || iconMap[iconName.toLowerCase()] || FileSearchOutlined
}

const dynamicSuggestions = computed(() => {
  const qq = agent.value?.quick_questions
  if (qq && qq.length > 0) {
    return qq.map(item => ({
      icon: getQuickIcon(item.icon),
      title: item.title,
      prompt: item.prompt,
      detail: item.detail,
    }))
  }
  return [
    { icon: UnorderedListOutlined, title: t('chat.quickPill.todos'), prompt: t('chat.prompt.todos'), detail: t('chat.suggestion.todos') },
    { icon: FileSearchOutlined, title: t('chat.quickPill.summary'), prompt: t('chat.prompt.summary'), detail: t('chat.suggestion.summary') },
    { icon: BookOutlined, title: t('chat.quickPill.guideline'), prompt: t('chat.prompt.guideline'), detail: t('chat.suggestion.guideline') },
  ]
})

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
        <div
          v-else-if="!messages.length"
          class="chat-welcome"
          :class="{ 'is-exiting': isSubmittingFirst }"
        >
          <div class="welcome-mark"><img src="/favicon.svg" alt="" width="36" height="36" /></div>
          <p class="welcome-eyebrow">{{ agent?.name || t('chat.assistantName') }}</p>
          <h1>{{ t('chat.welcomeTitle') }}</h1>
          <p class="welcome-description">{{ agent?.description || t('chat.welcomeDescription') }}</p>

          <!-- 现代 AI 风格：输入框上移至居中 Hero 区 -->
          <div class="welcome-composer-slot">
            <ChatComposer
              ref="welcomeComposer"
              variant="hero"
              :submitting="streaming"
              :disabled="detailLoading || !agent"
              @submit="submit"
              @stop="stopStreaming"
            />
          </div>

          <!-- 初始推荐卡片：与主题契合，无渐变 -->
          <div class="suggestions">
            <button
              v-for="(item, sIdx) in dynamicSuggestions"
              :key="sIdx"
              class="suggestion-card"
              @click="handleSelectSuggestion(item.prompt)"
            >
              <div class="suggestion-icon-wrap">
                <component :is="item.icon" />
              </div>
              <div class="suggestion-content">
                <strong>{{ item.title }}</strong>
                <span v-if="item.detail">{{ item.detail }}</span>
              </div>
              <ArrowRightOutlined class="suggestion-arrow" />
            </button>
          </div>

          <p class="composer-hint composer-hint--welcome">{{ t('chat.composerHint', '内容由 AI 生成，请结合实际业务审慎核验') }}</p>
          <p v-if="!effectiveAgents.length" class="no-agents">{{ t('chat.noAgents') }}</p>
        </div>
        <div v-else class="chat-thread-container">
          <ChatThread :messages="messages" :agent-name="agent?.name" />
        </div>
      </div>
    </div>
    <MessageJumpRail v-if="jumpTurns.length > 2" :turns="jumpTurns" @jump="jump" />
    <transition name="bottom-slide">
      <div v-if="messages.length" class="chat-bottom chat-bottom--active">
        <button v-if="!pinned && messages.length" class="scroll-bottom" :aria-label="t('chat.scrollBottom')" @click="pinned = true; scrollBottom()"><ArrowDownOutlined /></button>
        <ChatComposer ref="composer" :submitting="streaming" :disabled="detailLoading || !agent" @submit="submit" @stop="stopStreaming" />
        <p class="composer-hint">{{ t('chat.composerHint', '内容由 AI 生成，请结合实际业务审慎核验') }}</p>
      </div>
    </transition>
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

/* 欢迎区 Hero 布局与淡出过渡 */
.chat-welcome {
  width: min(800px, 100%);
  margin: auto;
  padding: clamp(32px, 6vh, 60px) 24px 36px;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  transition: opacity 0.32s cubic-bezier(0.2, 0.8, 0.2, 1), transform 0.32s cubic-bezier(0.2, 0.8, 0.2, 1);
}
.chat-welcome.is-exiting {
  opacity: 0;
  transform: translateY(-24px) scale(0.98);
  pointer-events: none;
}
.welcome-mark {
  margin-bottom: 18px;
  display: inline-flex;
  padding: 8px;
  background: var(--color-primary-bg);
  border-radius: 14px;
  box-shadow: 0 2px 8px var(--color-primary-ring);
}
.welcome-eyebrow {
  font-size: 13px;
  color: var(--color-primary);
  font-weight: 600;
  margin-bottom: 8px;
  letter-spacing: .04em;
}
h1 {
  font-size: clamp(26px, 2.8vw, 36px);
  font-weight: 700;
  line-height: 1.35;
  letter-spacing: -.03em;
  margin: 0 0 10px;
  color: var(--color-text-primary);
}
.welcome-description {
  font-size: 14px;
  color: var(--color-text-secondary);
  line-height: 1.65;
  max-width: 600px;
  margin: 0 auto;
}

/* 对话框上移至居中偏上 Hero 区 */
.welcome-composer-slot {
  width: 100%;
  margin: 28px 0 24px;
  text-align: left;
}

/* 推荐问题卡片 — 去除渐变，符合企业级主题 */
.suggestions {
  width: 100%;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
  margin-bottom: 24px;
}
.suggestion-card {
  position: relative;
  display: flex;
  align-items: flex-start;
  gap: 12px;
  text-align: left;
  padding: 14px 16px;
  border: 1px solid var(--color-border-light);
  background: var(--color-bg-card);
  border-radius: 12px;
  color: var(--color-text-secondary);
  cursor: pointer;
  box-shadow: var(--shadow-xs);
  transition: all .22s cubic-bezier(0.2, 0.8, 0.2, 1);
}
.suggestion-card:hover {
  border-color: var(--color-primary-lighter);
  background: var(--color-bg-hover);
  transform: translateY(-2px);
  box-shadow: var(--shadow-sm);
}
.suggestion-icon-wrap {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  background: var(--color-primary-bg);
  color: var(--color-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 15px;
  flex-shrink: 0;
}
.suggestion-content {
  flex: 1;
  min-width: 0;
}
.suggestion-card strong {
  font-size: 13.5px;
  color: var(--color-text-primary);
  font-weight: 600;
  margin-bottom: 3px;
  display: block;
}
.suggestion-card span {
  font-size: 12px;
  line-height: 1.5;
  color: var(--color-text-tertiary);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.suggestion-arrow {
  font-size: 11px;
  color: var(--color-primary);
  opacity: 0;
  transform: translateX(-4px);
  transition: all .2s ease;
  margin-top: 4px;
  flex-shrink: 0;
}
.suggestion-card:hover .suggestion-arrow {
  opacity: 0.85;
  transform: translateX(0);
}

.composer-hint--welcome {
  margin: 0;
  font-size: 12px;
  color: var(--color-text-tertiary);
}

/* 聊天线程淡入上升动画 */
.chat-thread-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  animation: threadFadeIn 0.38s cubic-bezier(0.2, 0.8, 0.2, 1) forwards;
}
@keyframes threadFadeIn {
  from {
    opacity: 0;
    transform: translateY(16px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* 底部输入框过渡动画 */
.chat-bottom { flex-shrink:0; position:relative; background:var(--color-bg-card); padding:8px 24px 10px; }
.bottom-slide-enter-active {
  transition: all 0.35s cubic-bezier(0.2, 0.8, 0.2, 1);
}
.bottom-slide-enter-from {
  opacity: 0;
  transform: translateY(20px);
}
.composer-hint { text-align:center; font-size:11px; color:var(--color-text-tertiary); margin:6px 0 0; }
.scroll-bottom { position:absolute; top:-44px; left:calc(50% - 17px); width:34px; height:34px; background:var(--color-bg-card); border:1px solid var(--color-border); color:var(--color-text-secondary); border-radius:50%; cursor:pointer; }
.chat-error,.no-agents { color:var(--color-text-secondary); font-size:13px; padding:14px; background:var(--color-bg-page); border-radius:8px; margin:16px auto; max-width:740px; }
.chat-loading { padding:60px; text-align:center; }

@media(max-width:768px) {
  .workspace-heading { padding:0 14px; height:50px; }
  .workspace-context { display:none; }
  .chat-welcome { padding:24px 16px; }
  .suggestions { grid-template-columns:1fr; gap:10px; margin-top:20px; }
  .chat-bottom { padding:8px 12px 10px; }
  .welcome-mark { margin-bottom:14px; }
}
@media(prefers-reduced-motion:reduce) {
  .suggestion-card, .chat-welcome, .chat-thread-container, .bottom-slide-enter-active { transition:none !important; animation:none !important; }
}
</style>
