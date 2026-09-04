<script setup lang="ts">
import { marked } from 'marked'
import {
  BulbOutlined,
  DownOutlined,
  UpOutlined,
  CopyOutlined,
  CheckOutlined,
  UserOutlined,
} from '@ant-design/icons-vue'
import type { ChatMessageItem } from '~/types/chat'
import GenericToolCard from './Cards/GenericToolCard.vue'

const props = defineProps<{
  messages: ChatMessageItem[]
  agentEmoji?: string
  agentName?: string
}>()

const { t } = useI18n()
const isThinkingExpanded = ref<Record<string, boolean>>({})
const copiedId = ref<string | null>(null)

const toggleThinking = (id: string) => {
  isThinkingExpanded.value[id] = !isThinkingExpanded.value[id]
}

const renderMarkdown = (content: string) => {
  if (!content) return ''
  return marked.parse(content)
}

const copyMessageText = async (msg: ChatMessageItem) => {
  try {
    await navigator.clipboard.writeText(msg.content)
    copiedId.value = msg.id
    setTimeout(() => {
      copiedId.value = null
    }, 1500)
  } catch (err) {
    console.error('复制失败', err)
  }
}
</script>

<template>
  <div class="chat-thread-wrapper">
    <div
      v-for="msg in messages"
      :key="msg.id"
      :id="msg.id"
      class="message-row"
      :class="`is-${msg.sender_type}`"
    >
      <!-- 用户消息气泡 -->
      <template v-if="msg.sender_type === 'user'">
        <div class="user-bubble">
          {{ msg.content }}
        </div>
      </template>

      <!-- 智能体消息 -->
      <template v-else-if="msg.sender_type === 'assistant'">
        <div class="assistant-container">
          <div class="assistant-header">
            <span class="assistant-avatar">{{ agentEmoji || '🤖' }}</span>
            <span class="assistant-name">{{ agentName || t('chat.assistantName') }}</span>
            <span v-if="msg.streaming" class="streaming-pill">{{ t('chat.generating') }}</span>
          </div>

          <!-- 思考过程折叠块 -->
          <div
            v-if="msg.reasoning_content"
            class="thinking-block"
            :class="{ 'is-expanded': isThinkingExpanded[msg.id] }"
          >
            <div class="thinking-header" @click="toggleThinking(msg.id)">
              <span class="thinking-icon"><BulbOutlined /></span>
              <span class="thinking-title">{{ t('chat.thinking') }}</span>
              <span class="thinking-arrow">
                <UpOutlined v-if="isThinkingExpanded[msg.id]" />
                <DownOutlined v-else />
              </span>
            </div>
            <div v-if="isThinkingExpanded[msg.id]" class="thinking-body">
              {{ msg.reasoning_content }}
            </div>
          </div>

          <!-- 工具执行卡片展示 -->
          <div v-if="msg.tool_executions && msg.tool_executions.length > 0" class="tools-container">
            <GenericToolCard
              v-for="(tool, tIdx) in msg.tool_executions"
              :key="tIdx"
              :tool-code="tool.tool_code"
              :tool-name="tool.tool_name"
              :arguments="tool.arguments"
              :result="tool.result"
              :error="tool.error"
              :execution-ms="tool.execution_ms"
            />
          </div>

          <!-- Markdown 回答正文 -->
          <div
            v-if="msg.content"
            class="markdown-body"
            v-html="renderMarkdown(msg.content)"
          />

          <!-- 底部工具栏：复制等 -->
          <div v-if="!msg.streaming && msg.content" class="assistant-footer">
            <button class="msg-action-btn" @click="copyMessageText(msg)">
              <CheckOutlined v-if="copiedId === msg.id" style="color: #52c41a" />
              <CopyOutlined v-else />
              <span>{{ copiedId === msg.id ? t('chat.copied') : t('chat.copy') }}</span>
            </button>
            <span v-if="msg.token_cost" class="token-cost-badge">{{ t('chat.tokenCost', [msg.token_cost]) }}</span>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.chat-thread-wrapper {
  display: flex;
  flex-direction: column;
  gap: 20px;
  width: min(100%, 820px);
  margin: 0 auto;
  padding: 24px 16px;
}
.message-row {
  display: flex;
  flex-direction: column;
}
.message-row.is-user {
  align-items: flex-end;
}
.user-bubble {
  max-width: 80%;
  background: #293747;
  color: #ffffff;
  padding: 10px 16px;
  border-radius: 14px 14px 2px 14px;
  font-size: 14px;
  line-height: 1.6;
  word-break: break-word;
  white-space: pre-wrap;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}
.assistant-container {
  max-width: 100%;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.assistant-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}
.assistant-avatar {
  font-size: 18px;
}
.assistant-name {
  font-size: 13px;
  font-weight: 600;
  color: #374151;
}
.streaming-pill {
  font-size: 11px;
  color: #1890ff;
  background: #e6f7ff;
  border: 1px solid #91d5ff;
  padding: 1px 6px;
  border-radius: 10px;
}
.thinking-block {
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background: #f9fafb;
  overflow: hidden;
}
.thinking-header {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  cursor: pointer;
  user-select: none;
  font-size: 12.5px;
  color: #6b7280;
}
.thinking-icon {
  color: #8b5cf6;
}
.thinking-arrow {
  margin-left: auto;
  font-size: 11px;
}
.thinking-body {
  padding: 8px 12px;
  font-size: 12px;
  color: #4b5563;
  line-height: 1.6;
  white-space: pre-wrap;
  border-top: 1px dashed #e5e7eb;
  background: #ffffff;
}
.markdown-body {
  font-size: 14px;
  line-height: 1.7;
  color: #1f2937;
  word-break: break-word;
}
.markdown-body :deep(pre) {
  background: #f3f4f6;
  padding: 12px;
  border-radius: 6px;
  overflow-x: auto;
}
.markdown-body :deep(code) {
  background: #f3f4f6;
  padding: 2px 4px;
  border-radius: 4px;
  font-family: monospace;
}
.assistant-footer {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 4px;
}
.msg-action-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  background: transparent;
  border: none;
  color: #8c8c8c;
  font-size: 12px;
  cursor: pointer;
  padding: 2px 6px;
  border-radius: 4px;
}
.msg-action-btn:hover {
  background: rgba(0, 0, 0, 0.05);
  color: #1890ff;
}
.token-cost-badge {
  font-size: 11px;
  color: #9ca3af;
}
</style>
