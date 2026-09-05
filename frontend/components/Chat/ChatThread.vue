<script setup lang="ts">
import { CopyOutlined, CheckOutlined, LoadingOutlined, BulbOutlined, DownOutlined } from '@ant-design/icons-vue'
import type { ChatMessageItem } from '~/types/chat'
import { renderSafeMarkdown } from '~/utils/markdown'
import ToolActivity from './ToolActivity.vue'
defineProps<{ messages: ChatMessageItem[]; agentEmoji?: string; agentName?: string }>()
const { t } = useI18n()
const copied = ref('')
const copy = async (msg: ChatMessageItem) => {
  try { await navigator.clipboard.writeText(msg.content); copied.value = msg.id; setTimeout(() => { copied.value = '' }, 1600) } catch { copied.value = '' }
}
</script>

<template>
  <div class="chat-thread">
    <article v-for="msg in messages" :id="msg.id" :key="msg.id" class="message" :class="msg.role">
      <template v-if="msg.role === 'user'"><div class="user-caption">{{ t('chat.you') }}</div><div class="user-prompt">{{ msg.content }}</div></template>
      <template v-else-if="msg.role === 'assistant'">
        <div class="assistant-byline"><img src="/favicon.svg" alt="" width="18" height="18" /><span>{{ agentName || t('chat.assistantName') }}</span></div>
        <details v-if="msg.reasoning_content" class="reasoning"><summary><BulbOutlined />{{ t('chat.thinking') }}<DownOutlined /></summary><p>{{ msg.reasoning_content }}</p></details>
        <div v-if="msg.tool_calls?.length" class="tool-timeline"><ToolActivity v-for="tool in msg.tool_calls" :key="tool.tool_call_id" :tool="tool" /></div>
        <div v-if="msg.content" class="answer-document" v-html="renderSafeMarkdown(msg.content)" />
        <div v-if="msg.streaming" class="generation-status" role="status"><LoadingOutlined spin />{{ t('chat.generating') }}</div>
        <div v-if="msg.error" class="message-error" role="alert">{{ msg.error }}</div>
        <p v-else-if="msg.status === 'error'" class="message-error" role="alert">{{ t('chat.replyFailed') }}</p>
        <p v-else-if="msg.status === 'interrupted'" class="generation-status">{{ t('chat.interrupted') }}</p>
        <div v-if="!msg.streaming && msg.content" class="answer-actions"><button :aria-label="t('chat.copy')" @click="copy(msg)"><CheckOutlined v-if="copied === msg.id" /><CopyOutlined v-else />{{ copied === msg.id ? t('chat.copied') : t('chat.copy') }}</button><span v-if="msg.token_usage?.total_tokens">{{ t('chat.tokenCost', [msg.token_usage.total_tokens]) }}</span></div>
      </template>
    </article>
  </div>
</template>

<style scoped>
.chat-thread { width:min(100%,780px); margin:auto; padding:26px 22px 44px; }
.message { scroll-margin-top:24px; min-width:0; }.message.user { margin:20px 0 28px; padding:0 0 20px; border-bottom:1px solid var(--color-border-light); }.message.assistant { margin-bottom:46px; }
.user-caption { color:var(--color-text-tertiary); font-size:11px; margin-bottom:10px; }.user-prompt { font-size:19px; font-weight:550; line-height:1.65; white-space:pre-wrap; overflow-wrap:anywhere; letter-spacing:-.02em; }
.assistant-byline { display:flex; align-items:center; gap:9px; margin-bottom:18px; font-size:12px; color:var(--color-text-secondary); font-weight:500; }
.reasoning { margin:0 0 16px; color:var(--color-text-tertiary); font-size:12px; }.reasoning summary { display:flex; align-items:center; gap:8px; cursor:pointer; list-style:none; }.reasoning summary::-webkit-details-marker { display:none; }.reasoning p { border-left:2px solid var(--color-border); padding:8px 16px; white-space:pre-wrap; line-height:1.85; max-height:260px; overflow:auto; color:var(--color-text-secondary); }
.tool-timeline { margin:0 0 18px; max-width:620px; }
.answer-document { font-size:14.5px; line-height:1.95; color:var(--color-text-primary); overflow-wrap:anywhere; }
.answer-document :deep(p) { margin:0 0 17px; }.answer-document :deep(h1) { font-size:25px; }.answer-document :deep(h2) { font-size:20px; }.answer-document :deep(h3) { font-size:16px; }
.answer-document :deep(h1),.answer-document :deep(h2),.answer-document :deep(h3),.answer-document :deep(h4) { font-weight:600; letter-spacing:-.015em; margin:30px 0 12px; line-height:1.6; }
.answer-document :deep(ul),.answer-document :deep(ol) { padding-left:24px; margin:12px 0 22px; }.answer-document :deep(li) { padding-left:3px; margin:7px 0; }
.answer-document :deep(strong) { font-weight:600; }.answer-document :deep(a) { color:var(--color-primary); text-decoration:underline; text-underline-offset:3px; }
.answer-document :deep(blockquote) { margin:22px 0; padding:12px 20px; border-left:3px solid var(--color-primary); background:var(--color-bg-page); border-radius:0 9px 9px 0; color:var(--color-text-secondary); }.answer-document :deep(blockquote p:last-child) { margin:0; }
.answer-document :deep(pre) { overflow:auto; background:var(--color-bg-page); border:1px solid var(--color-border-light); border-radius:12px; padding:18px 20px; margin:18px 0 24px; line-height:1.7; }.answer-document :deep(code) { font-family:ui-monospace,SFMono-Regular,Consolas,monospace; font-size:.86em; background:var(--color-bg-hover); padding:2px 5px; border-radius:4px; }.answer-document :deep(pre code) { background:none; padding:0; }
.answer-document :deep(table) { display:block; max-width:100%; overflow:auto; border-collapse:collapse; margin:20px 0; font-size:13px; }.answer-document :deep(th),.answer-document :deep(td) { border-bottom:1px solid var(--color-border); padding:11px 15px; text-align:left; min-width:100px; }.answer-document :deep(th) { background:var(--color-bg-page); font-weight:600; }.answer-document :deep(hr) { border:0; border-top:1px solid var(--color-border-light); margin:30px 0; }
.answer-actions { display:flex; align-items:center; gap:16px; margin-top:18px; color:var(--color-text-tertiary); font-size:10px; }.answer-actions button { display:flex; align-items:center; gap:6px; padding:5px 0; background:none; border:0; color:inherit; font-size:11px; cursor:pointer; }.answer-actions button:hover { color:var(--color-primary); }
.generation-status { display:flex; align-items:center; gap:7px; margin-top:14px; font-size:12px; color:var(--color-text-tertiary); }.message-error { background:var(--color-bg-page); border-left:2px solid var(--color-primary); padding:12px; font-size:13px; margin-top:14px; }
@media(max-width:600px) { .chat-thread { padding:18px 22px 28px; }.user-prompt { font-size:17px; }.answer-document { font-size:14px; } }
</style>
