<script setup lang="ts">
import {
  CopyOutlined,
  CheckOutlined,
  LikeOutlined,
  LikeFilled,
  DislikeOutlined,
  DislikeFilled,
  RobotOutlined,
  LoadingOutlined,
} from '@ant-design/icons-vue'
import { message } from 'ant-design-vue'
import type { ChatMessageItem } from '~/types/chat'
import { renderSafeMarkdown } from '~/utils/markdown'
import { safeCopyText } from '~/utils/clipboard'
import { useChatSession } from '~/composables/useChatSession'
import ChatProcessTimeline from './ChatProcessTimeline.vue'

defineProps<{ messages: ChatMessageItem[]; agentEmoji?: string; agentName?: string }>()
const { t } = useI18n()
const { updateMessageFeedback } = useChatSession()
const copied = ref('')

const copy = async (msg: ChatMessageItem) => {
  const ok = await safeCopyText(msg.content)
  if (ok) {
    copied.value = msg.id
    message.success(t('chat.copied', '已复制到剪贴板'))
    setTimeout(() => { copied.value = '' }, 1800)
  } else {
    message.error(t('chat.copyFailed', '复制失败，请手动选择文本复制'))
  }
}

const toggleFeedback = async (msg: ChatMessageItem, type: 'like' | 'dislike') => {
  const next = msg.feedback === type ? null : type
  msg.feedback = next
  try {
    await updateMessageFeedback(msg.id, next)
    if (next === 'like') {
      message.success(t('chat.feedbackLikeSuccess', '感谢您的赞同反馈！'))
    } else if (next === 'dislike') {
      message.info(t('chat.feedbackDislikeSuccess', '已记录您的改进建议'))
    }
  } catch (err: any) {
    message.error(err?.message || '反馈提交失败')
  }
}

function formatMsgTime(isoString?: string): string {
  if (!isoString) return ''
  const d = new Date(isoString)
  if (isNaN(d.getTime())) return ''
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}
</script>

<template>
  <div class="chat-thread">
    <article v-for="msg in messages" :id="msg.id" :key="msg.id" class="message-turn" :class="msg.role">
      <!-- 1. 用户提问：纯净右对齐气泡，无头像，无“你”标识 -->
      <div v-if="msg.role === 'user'" class="user-bubble-wrapper">
        <div class="user-bubble">
          <div class="user-content">{{ msg.content }}</div>
        </div>
      </div>

      <!-- 2. 助手回答：纯净展示，无 OA 标识与头像 -->
      <div v-else-if="msg.role === 'assistant'" class="assistant-content-wrapper">
        <!-- 思考过程与工具调用时间线 -->
        <ChatProcessTimeline :message="msg" />

        <!-- 回答正文 -->
        <div v-if="msg.content" class="answer-document">
          <div v-if="msg.streaming" class="answer-streaming">{{ msg.content }}</div>
          <div v-else v-html="renderSafeMarkdown(msg.content)" />
        </div>

        <div v-if="msg.streaming" class="generation-status" role="status">
          <LoadingOutlined spin /> {{ t('chat.generating') }}
        </div>
        <div v-if="msg.error" class="message-error" role="alert">{{ msg.error }}</div>
        <p v-else-if="msg.status === 'error'" class="message-error" role="alert">{{ t('chat.replyFailed') }}</p>
        <p v-else-if="msg.status === 'interrupted'" class="generation-status">{{ t('chat.interrupted') }}</p>

        <!-- 底部功能栏：复制、点赞、点踩、生成时间、AI生成标签 -->
        <div v-if="!msg.streaming && msg.content" class="answer-actions">
          <div class="actions-left">
            <button class="action-btn" :title="t('chat.copy')" @click="copy(msg)">
              <CheckOutlined v-if="copied === msg.id" style="color: #52c41a;" />
              <CopyOutlined v-else />
              <span>{{ copied === msg.id ? t('chat.copied') : t('chat.copy') }}</span>
            </button>

            <button
              class="action-btn"
              :class="{ 'action-btn--liked': msg.feedback === 'like' }"
              :title="t('chat.like', '赞同')"
              @click="toggleFeedback(msg, 'like')"
            >
              <LikeFilled v-if="msg.feedback === 'like'" style="color: #1890ff;" />
              <LikeOutlined v-else />
              <span>{{ t('chat.like', '赞同') }}</span>
            </button>

            <button
              class="action-btn"
              :class="{ 'action-btn--disliked': msg.feedback === 'dislike' }"
              :title="t('chat.dislike', '改进')"
              @click="toggleFeedback(msg, 'dislike')"
            >
              <DislikeFilled v-if="msg.feedback === 'dislike'" style="color: #ff4d4f;" />
              <DislikeOutlined v-else />
              <span>{{ t('chat.dislike', '改进') }}</span>
            </button>

            <span v-if="msg.token_usage?.total_tokens" class="meta-token-count">
              {{ t('chat.tokenCost', [msg.token_usage.total_tokens]) }}
            </span>
          </div>

          <div class="actions-right">
            <span v-if="msg.created_at" class="meta-time">
              {{ formatMsgTime(msg.created_at) }}
            </span>
            <span class="ai-generated-tag">
              <RobotOutlined /> AI 生成
            </span>
          </div>
        </div>
      </div>
    </article>
  </div>
</template>

<style scoped>
.chat-thread {
  width: min(100%, 780px);
  margin: 0 auto;
  padding: 26px 20px 48px;
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.message-turn {
  scroll-margin-top: 24px;
  width: 100%;
}

/* 用户提问气泡（ChatGPT / Copilot 纯净风格） */
.user-bubble-wrapper {
  display: flex;
  justify-content: flex-end;
  width: 100%;
}

.user-bubble {
  max-width: min(78%, 560px);
  border: 1px solid rgba(38, 52, 68, 0.08);
  border-radius: 18px 18px 5px 18px;
  padding: 12px 18px;
  background: #293747;
  color: #fffbf3;
  font-size: 14.5px;
  line-height: 1.65;
  box-shadow: 0 4px 16px rgba(33, 45, 58, 0.08);
}

.user-content {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

/* 助手纯净回答 */
.assistant-content-wrapper {
  width: 100%;
  display: flex;
  flex-direction: column;
}

.answer-document {
  font-size: 14.5px;
  line-height: 1.9;
  color: var(--color-text-primary);
  overflow-wrap: anywhere;
}

.answer-streaming {
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.answer-document :deep(p) {
  margin: 0 0 16px;
}
.answer-document :deep(h1) { font-size: 24px; }
.answer-document :deep(h2) { font-size: 20px; }
.answer-document :deep(h3) { font-size: 16px; }
.answer-document :deep(h1),
.answer-document :deep(h2),
.answer-document :deep(h3),
.answer-document :deep(h4) {
  font-weight: 600;
  letter-spacing: -0.015em;
  margin: 24px 0 10px;
  line-height: 1.5;
}

.answer-document :deep(ul),
.answer-document :deep(ol) {
  padding-left: 22px;
  margin: 10px 0 18px;
}
.answer-document :deep(li) {
  padding-left: 2px;
  margin: 6px 0;
}

.answer-document :deep(strong) {
  font-weight: 600;
}
.answer-document :deep(a) {
  color: var(--color-primary);
  text-decoration: underline;
  text-underline-offset: 3px;
}

.answer-document :deep(blockquote) {
  margin: 18px 0;
  padding: 10px 16px;
  border-left: 3px solid var(--color-primary);
  background: var(--color-bg-page);
  border-radius: 0 8px 8px 0;
  color: var(--color-text-secondary);
}
.answer-document :deep(blockquote p:last-child) {
  margin: 0;
}

.answer-document :deep(pre) {
  overflow: auto;
  background: var(--color-bg-page);
  border: 1px solid var(--color-border-light);
  border-radius: 10px;
  padding: 14px 18px;
  margin: 14px 0 20px;
  line-height: 1.65;
}
.answer-document :deep(code) {
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
  font-size: 0.88em;
  background: var(--color-bg-hover);
  padding: 2px 5px;
  border-radius: 4px;
}
.answer-document :deep(pre code) {
  background: none;
  padding: 0;
}

.answer-document :deep(table) {
  display: block;
  max-width: 100%;
  overflow: auto;
  border-collapse: collapse;
  margin: 16px 0;
  font-size: 13px;
}
.answer-document :deep(th),
.answer-document :deep(td) {
  border-bottom: 1px solid var(--color-border);
  padding: 10px 14px;
  text-align: left;
  min-width: 90px;
}
.answer-document :deep(th) {
  background: var(--color-bg-page);
  font-weight: 600;
}

/* 底部操作栏 */
.answer-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 14px;
  padding-top: 8px;
  border-top: 1px solid rgba(0, 0, 0, 0.04);
}

.actions-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.actions-right {
  display: flex;
  align-items: center;
  gap: 10px;
}

.action-btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 8px;
  border-radius: 6px;
  background: transparent;
  border: 1px solid transparent;
  color: var(--color-text-secondary);
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s ease;
}

.action-btn:hover {
  background: var(--color-bg-page);
  border-color: var(--color-border-light);
  color: var(--color-text-primary);
}

.action-btn--liked {
  color: #1890ff;
  background: #e6f7ff;
}

.action-btn--disliked {
  color: #ff4d4f;
  background: #fff1f0;
}

.meta-token-count {
  font-size: 11px;
  color: var(--color-text-tertiary);
}

.meta-time {
  font-size: 11px;
  color: var(--color-text-tertiary);
}

.ai-generated-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  color: var(--color-primary);
  background: var(--color-primary-bg);
  padding: 2px 8px;
  border-radius: 10px;
  font-weight: 500;
}

.generation-status {
  display: flex;
  align-items: center;
  gap: 7px;
  margin-top: 14px;
  font-size: 12px;
  color: var(--color-text-tertiary);
}

.message-error {
  background: var(--color-bg-page);
  border-left: 2px solid var(--color-primary);
  padding: 12px;
  font-size: 13px;
  margin-top: 14px;
}

@media (max-width: 600px) {
  .chat-thread {
    padding: 16px 14px 28px;
  }
  .user-bubble {
    max-width: 90%;
    padding: 10px 14px;
  }
}
</style>
