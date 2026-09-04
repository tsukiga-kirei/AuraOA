<script setup lang="ts">
import { buildJumpTurns } from '~/utils/chatJump'
import type { ChatMessageItem } from '~/types/chat'
import ChatSidebar from '~/components/Chat/ChatSidebar.vue'
import ChatHeader from '~/components/Chat/ChatHeader.vue'
import ChatThread from '~/components/Chat/ChatThread.vue'
import ChatComposer from '~/components/Chat/ChatComposer.vue'
import MessageJumpRail from '~/components/Chat/MessageJumpRail.vue'

definePageMeta({
  layout: 'default',
})

const {
  sessions,
  effectiveAgents,
  currentSessionId,
  currentDetail,
  loading,
  fetchEffectiveAgents,
  fetchSessions,
  selectSession,
  createSession,
  renameSession,
  deleteSession,
} = useChatSession()

const messages = ref<ChatMessageItem[]>([])
const canvasScrollRef = ref<HTMLElement | null>(null)
const composerRef = ref<any>(null)

const { streaming, sendStreamMessage, stopStreaming } = useChatStream({
  onDone: () => {
    scrollToBottom()
  },
})

// 当切换选中的会话时，同步会话内的消息记录
watch(() => currentDetail.value, (val) => {
  if (val && val.messages) {
    messages.value = [...val.messages]
    nextTick(() => {
      scrollToBottom()
    })
  } else {
    messages.value = []
  }
})

// 计算跳转导航轨道 turns
const jumpTurns = computed(() => {
  return buildJumpTurns(messages.value)
})

const scrollToBottom = () => {
  if (canvasScrollRef.value) {
    canvasScrollRef.value.scrollTop = canvasScrollRef.value.scrollHeight
  }
}

const handleJump = (id: string) => {
  const el = document.getElementById(id)
  if (el) {
    el.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
}

const handleSelectAgent = async (agentCode: string) => {
  // 切换智能体创建新会话
  await createSession(agentCode)
}

const handleCreateSession = async () => {
  const defaultAgent = effectiveAgents.value[0]?.code
  await createSession(defaultAgent)
}

const handleSubmitMessage = async (content: string) => {
  if (!currentSessionId.value) {
    const defaultAgent = effectiveAgents.value[0]?.code
    const created = await createSession(defaultAgent, content.slice(0, 20))
    if (!created) return
  }
  if (currentSessionId.value) {
    sendStreamMessage(currentSessionId.value, content, messages)
    nextTick(() => {
      scrollToBottom()
    })
  }
}

const { t } = useI18n()

onMounted(async () => {
  await Promise.all([
    fetchEffectiveAgents(),
    fetchSessions(),
  ])
  if (sessions.value.length > 0 && !currentSessionId.value) {
    await selectSession(sessions.value[0].id)
  }
})
</script>

<template>
  <div class="chat-workspace">
    <!-- 左侧会话历史侧边栏 -->
    <ChatSidebar
      :sessions="sessions"
      :current-session-id="currentSessionId"
      :loading="loading"
      @select="selectSession"
      @create="handleCreateSession"
      @rename="renameSession"
      @delete="deleteSession"
      @search="fetchSessions"
    />

    <!-- 右侧对话主工作区 -->
    <div class="chat-main">
      <!-- 顶栏：智能体选择与状态 -->
      <ChatHeader
        :agents="effectiveAgents"
        :current-agent-code="currentDetail?.session.agent_code"
        @select="handleSelectAgent"
      />

      <!-- 消息画布 -->
      <div class="chat-canvas" ref="canvasScrollRef">
        <template v-if="messages.length === 0">
          <div class="chat-empty-state">
            <div class="empty-icon">{{ currentDetail?.agent?.avatar_emoji || '🤖' }}</div>
            <div class="empty-title">{{ t('chat.emptyGreeting') }}</div>
            <div class="empty-subtitle">{{ t('chat.emptyDesc') }}</div>
            <div class="quick-pills">
              <button class="pill-btn" @click="composerRef?.appendPrompt(t('chat.prompt.todos'))">
                {{ t('chat.quickPill.todos') }}
              </button>
              <button class="pill-btn" @click="composerRef?.appendPrompt(t('chat.prompt.summary'))">
                {{ t('chat.quickPill.summary') }}
              </button>
              <button class="pill-btn" @click="composerRef?.appendPrompt(t('chat.prompt.guideline'))">
                {{ t('chat.quickPill.guideline') }}
              </button>
            </div>
          </div>
        </template>

        <template v-else>
          <ChatThread
            :messages="messages"
            :agent-emoji="currentDetail?.session.agent_avatar_emoji"
            :agent-name="currentDetail?.session.agent_name"
          />
        </template>
      </div>

      <!-- 右侧悬浮跳转线 -->
      <MessageJumpRail
        :turns="jumpTurns"
        @jump="handleJump"
      />

      <!-- 底部输入框与工具栏 -->
      <div class="chat-bottom-bar">
        <ChatComposer
          ref="composerRef"
          :submitting="streaming"
          @submit="handleSubmitMessage"
          @stop="stopStreaming"
        />
      </div>
    </div>
  </div>
</template>

<style scoped>
.chat-workspace {
  display: flex;
  height: calc(100vh - 64px);
  background: #ffffff;
  position: relative;
  overflow: hidden;
}
.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  position: relative;
  height: 100%;
  background: #fafaf8;
}
.chat-canvas {
  flex: 1;
  overflow-y: auto;
  position: relative;
  padding-bottom: 20px;
}
.chat-bottom-bar {
  background: linear-gradient(180deg, rgba(250, 250, 248, 0) 0%, #fafaf8 20%);
}
.chat-empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 80%;
  text-align: center;
  padding: 40px 20px;
}
.empty-icon {
  font-size: 48px;
  margin-bottom: 16px;
}
.empty-title {
  font-size: 20px;
  font-weight: 600;
  color: #1f2937;
  margin-bottom: 8px;
}
.empty-subtitle {
  font-size: 14px;
  color: #6b7280;
  margin-bottom: 24px;
  max-width: 480px;
}
.quick-pills {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  justify-content: center;
}
.pill-btn {
  background: #ffffff;
  border: 1px solid #e5e7eb;
  border-radius: 20px;
  padding: 8px 16px;
  font-size: 13px;
  color: #374151;
  cursor: pointer;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.04);
  transition: all 0.2s;
}
.pill-btn:hover {
  border-color: #1890ff;
  color: #1890ff;
  transform: translateY(-1px);
}
</style>
