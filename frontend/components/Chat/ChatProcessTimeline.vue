<script setup lang="ts">
import {
  LoadingOutlined,
  CloseCircleOutlined,
  DownOutlined,
} from '@ant-design/icons-vue'
import type { ChatMessageItem, ChatToolExecution } from '~/types/chat'
import { renderSafeMarkdown } from '~/utils/markdown'
import ToolActivity from './ToolActivity.vue'

const props = defineProps<{
  message: ChatMessageItem
}>()

const { t, te } = useI18n()

interface TimelineStep {
  id: string
  type: 'thinking' | 'tool'
  title: string
  hasPreamble?: boolean
  content?: string
  tool?: ChatToolExecution
}

// 格式化毫秒为人类可读格式，例如 "2m 8s" 或 "15s"
function formatDuration(ms?: number): string {
  if (!ms || ms <= 0) return '0s'
  if (ms < 1000) return `${ms}ms`
  const seconds = Math.floor(ms / 1000)
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  const remSeconds = seconds % 60
  return remSeconds > 0 ? `${minutes}m ${remSeconds}s` : `${minutes}m`
}

// 提取思考内容首行作为预览标题（在无开场白时显示）
function getThinkingPreview(content?: string): string {
  if (!content) return t('chat.thinking')
  const clean = content
    .replace(/^#+\s+/gm, '')
    .replace(/\*\*/g, '')
    .replace(/\*/g, '')
    .replace(/`/g, '')
    .replace(/\n+/g, ' ')
    .trim()
  if (clean.length <= 50) return clean || t('chat.thinking')
  return `${clean.slice(0, 50)}...`
}

// 获取友好的工具展示名称
function getToolDisplayName(tool?: ChatToolExecution): string {
  if (!tool) return ''
  const code = tool.tool_code || ''
  if (te(`chat.tools.${code}`)) {
    return t(`chat.tools.${code}`)
  }
  return code.replace(/^(skill:|mcp:)/, '').replace(/:/g, ' / ')
}

// 构造时间线步骤：支持多轮思考与多轮工具调用顺着排列
const steps = computed<TimelineStep[]>(() => {
  const result: TimelineStep[] = []
  const toolCalls = props.message.tool_calls || []

  // 判断工具是否携带自己的 thought（开场白或当轮思考）
  const hasToolThoughts = toolCalls.some(tool => Boolean(tool.thought))

  if (hasToolThoughts) {
    const seenThoughts = new Set<string>()
    toolCalls.forEach((tool, index) => {
      if (tool.thought && !seenThoughts.has(tool.thought.trim())) {
        seenThoughts.add(tool.thought.trim())
        result.push({
          id: `thought_${tool.tool_call_id || index}`,
          type: 'thinking',
          title: tool.thought,
          hasPreamble: true,
          content: props.message.reasoning_content || tool.thought,
        })
      }
      result.push({
        id: `tool_${tool.tool_call_id || index}`,
        type: 'tool',
        title: tool.tool_code,
        tool,
      })
    })

    // 如果最终还有剩余未归入工具的 reasoning_content（如最终回复前的深度思考）
    if (props.message.reasoning_content) {
      let remainder = props.message.reasoning_content
      for (const pt of seenThoughts) {
        remainder = remainder.replace(pt, '').trim()
      }
      if (remainder && remainder.length > 5) {
        result.push({
          id: `thought_extra_${props.message.id}`,
          type: 'thinking',
          title: getThinkingPreview(remainder),
          hasPreamble: false,
          content: remainder,
        })
      }
    }
  } else {
    // 历史数据或普通单次思考：思考在先，工具顺着排列
    if (props.message.reasoning_content) {
      result.push({
        id: `reasoning_${props.message.id}`,
        type: 'thinking',
        title: getThinkingPreview(props.message.reasoning_content),
        hasPreamble: false,
        content: props.message.reasoning_content,
      })
    }
    toolCalls.forEach((tool, index) => {
      result.push({
        id: `tool_${tool.tool_call_id || index}`,
        type: 'tool',
        title: tool.tool_code,
        tool,
      })
    })
  }

  return result
})

const isStreaming = computed(() => Boolean(props.message.streaming))
const hasError = computed(() => props.message.status === 'error' || props.message.tool_calls?.some(t => t.status === 'error'))
const isBusy = computed(() => isStreaming.value && (props.message.tool_calls?.some(t => t.status === 'running') || !props.message.content))

// 耗时计时器支持
const liveDuration = ref(props.message.duration_ms || 0)
let timer: ReturnType<typeof setInterval> | null = null

watch(
  () => props.message.streaming,
  (streaming) => {
    if (streaming) {
      const start = Date.now() - (liveDuration.value || 0)
      timer = setInterval(() => {
        liveDuration.value = Date.now() - start
      }, 500)
    } else {
      if (timer) {
        clearInterval(timer)
        timer = null
      }
      if (props.message.duration_ms) {
        liveDuration.value = props.message.duration_ms
      }
    }
  },
  { immediate: true }
)

watch(
  () => props.message.duration_ms,
  (ms) => {
    if (ms) liveDuration.value = ms
  }
)

onUnmounted(() => {
  if (timer) clearInterval(timer)
})

// 默认总折叠状态：流式执行且无正文时展开；生成完毕且已有正文时默认收起
const open = ref(true)

watch(
  () => props.message.streaming,
  (streaming, prev) => {
    if (prev && !streaming && props.message.content) {
      open.value = false
    }
  },
  { immediate: true }
)

onMounted(() => {
  if (!isStreaming.value && props.message.content) {
    open.value = false
  }
})

// 单项展开状态集合（每个小项可单独展开或折叠）
const expandedStepIds = ref<Set<string>>(new Set())

function toggleStep(id: string) {
  if (expandedStepIds.value.has(id)) {
    expandedStepIds.value.delete(id)
  } else {
    expandedStepIds.value.add(id)
  }
}

function isStepExpanded(id: string) {
  return expandedStepIds.value.has(id)
}

// 思考轮次与工具调用次数
const reasoningRoundsCount = computed(() => steps.value.filter(s => s.type === 'thinking').length)
const toolCallsCount = computed(() => steps.value.filter(s => s.type === 'tool').length)

// 头部摘要 HTML（如：思考 2 轮 · 调用 2 次工具 · 耗时 2m 8s）
const summaryHtml = computed(() => {
  const parts: string[] = []
  const rounds = reasoningRoundsCount.value
  const tools = toolCallsCount.value
  const elapsed = liveDuration.value

  if (rounds > 0) {
    parts.push(t('chat.reasoningRounds', { rounds: `<strong>${rounds}</strong>` }))
  }
  if (tools > 0) {
    parts.push(t('chat.toolCallsCount', { tools: `<strong>${tools}</strong>` }))
  }
  if (parts.length === 0) {
    parts.push(t('chat.stepCount', [`<strong>${steps.value.length}</strong>`]))
  }
  if (elapsed > 0) {
    parts.push(t('chat.durationSuffix', { duration: `<strong>${formatDuration(elapsed)}</strong>` }))
  }

  return parts.join(t('chat.stepSummarySeparator'))
})
</script>

<template>
  <div v-if="steps.length" class="tree-container">
    <!-- 统一外层树根折叠头部：思考 N 轮 · 调用 N 次工具 · 耗时 N ⌄ -->
    <div class="tree-root" role="button" tabindex="0" @click="open = !open" @keydown.enter="open = !open" @keydown.space.prevent="open = !open">
      <span class="tree-root-summary" v-html="summaryHtml" />
      <DownOutlined class="tree-root-chevron" :class="{ 'is-collapsed': !open }" />
    </div>

    <!-- 过程步骤纵向时间线树 -->
    <div v-show="open" class="tree-children">
      <div
        v-for="step in steps"
        :key="step.id"
        class="tree-child"
      >
        <!-- 思考步骤 -->
        <template v-if="step.type === 'thinking'">
          <div class="tree-node-header" role="button" tabindex="0" @click="toggleStep(step.id)" @keydown.enter="toggleStep(step.id)" @keydown.space.prevent="toggleStep(step.id)">
            <!-- 灯泡图标 -->
            <span class="tree-node-icon icon-bulb">
              <svg viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
                <path d="M15 14c.2-1 .7-1.7 1.5-2.5 1-.9 1.5-2.2 1.5-3.5A6 6 0 0 0 6 8c0 1 .2 2.2 1.5 3.5.7.7 1.3 1.5 1.5 2.5" />
                <path d="M9 18h6" />
                <path d="M10 21h4" />
              </svg>
            </span>
            <span class="tree-node-title" :class="{ 'is-preview': !step.hasPreamble }">
              {{ step.title }}
            </span>
          </div>
          <!-- 展开内容：完整 Markdown 思考过程 -->
          <div v-if="isStepExpanded(step.id) && step.content" class="tree-node-details">
            <div class="thinking-content markdown-content" v-html="renderSafeMarkdown(step.content)" />
          </div>
        </template>

        <!-- 工具调用步骤 -->
        <template v-else-if="step.type === 'tool' && step.tool">
          <div class="tree-node-header" role="button" tabindex="0" @click="toggleStep(step.id)" @keydown.enter="toggleStep(step.id)" @keydown.space.prevent="toggleStep(step.id)">
            <!-- 终端提示符 >_ 图标 -->
            <span class="tree-node-icon icon-terminal">
              <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                <polyline points="4 17 10 12 4 7" />
                <line x1="12" y1="19" x2="20" y2="19" />
              </svg>
            </span>
            <span class="tree-node-title">
              {{ t('chat.toolCall') }}
              <span v-if="getToolDisplayName(step.tool)" class="tool-name-hint"> · {{ getToolDisplayName(step.tool) }}</span>
            </span>
            <span v-if="step.tool.status === 'running'" class="node-state-running">
              <LoadingOutlined spin />
            </span>
            <span v-else-if="step.tool.status === 'error'" class="node-state-error">
              <CloseCircleOutlined />
            </span>
          </div>
          <!-- 展开内容：工具输入与输出结果 -->
          <div v-if="isStepExpanded(step.id)" class="tree-node-details">
            <ToolActivity :tool="step.tool" :expanded-only="true" />
          </div>
        </template>
      </div>

      <!-- 终态节点（完成 / 进行中） -->
      <div class="tree-child tree-child-last">
        <div class="tree-node-header is-static">
          <span class="tree-node-icon icon-done">
            <LoadingOutlined v-if="isBusy" spin class="icon-spinning" />
            <CloseCircleOutlined v-else-if="hasError" class="icon-error" />
            <svg v-else viewBox="0 0 24 24" width="16" height="16" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="10" />
              <polyline points="8 12 11 15 16 9" />
            </svg>
          </span>
          <span class="tree-node-title">
            {{ isBusy ? t('chat.processing') : t('chat.stepFinish') }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tree-container {
  margin: 0 0 16px;
  position: relative;
  max-width: 680px;
}

/* 顶部汇总条（类 Weknora / Claude 极简风格，无边框卡片） */
.tree-root {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  user-select: none;
  font-size: 13.5px;
  color: var(--color-text-secondary, #595959);
  transition: color 0.15s ease;
}

.tree-root:hover {
  color: var(--color-text-primary, #262626);
}

.tree-root-summary :deep(strong) {
  font-weight: 600;
  color: var(--color-text-primary, #262626);
}

.tree-root-chevron {
  font-size: 10px;
  color: var(--color-text-tertiary, #8c8c8c);
  transition: transform 0.2s ease;
}

.tree-root-chevron.is-collapsed {
  transform: rotate(-90deg);
}

/* 时间线树容器 */
.tree-children {
  position: relative;
  margin-top: 14px;
  margin-left: 2px;
}

/* 每个时间线节点与贯穿纵向连线 */
.tree-child {
  position: relative;
  padding-left: 28px;
  margin-bottom: 16px;
}

/* 纵向连线连接相邻节点 */
.tree-child::before {
  content: '';
  position: absolute;
  left: 7.5px;
  top: 20px;
  bottom: -16px;
  width: 0;
  border-left: 1px solid var(--color-border-light, #e8e8e8);
}

.tree-child.tree-child-last {
  margin-bottom: 0;
}

.tree-child.tree-child-last::before {
  content: none;
}

/* 节点头部 */
.tree-node-header {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  cursor: pointer;
  user-select: none;
}

.tree-node-header.is-static {
  cursor: default;
}

/* 节点图标定位与对齐 */
.tree-node-icon {
  position: absolute;
  left: 0;
  top: 2px;
  width: 16px;
  height: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-tertiary, #8c8c8c);
  flex-shrink: 0;
}

.tree-node-icon.icon-done .icon-spinning {
  color: var(--color-primary);
}

.tree-node-icon.icon-done .icon-error {
  color: var(--color-error, #ff4d4f);
}

/* 节点文字 */
.tree-node-title {
  font-size: 13px;
  line-height: 1.55;
  color: var(--color-text-secondary, #595959);
  font-weight: 400;
  flex: 1;
  min-width: 0;
  word-break: break-word;
}

.tree-node-title.is-preview {
  color: var(--color-text-tertiary, #8c8c8c);
}

.tool-name-hint {
  color: var(--color-text-tertiary, #8c8c8c);
  font-size: 12px;
}

.node-state-running {
  font-size: 11px;
  color: var(--color-primary);
}

.node-state-error {
  font-size: 11px;
  color: var(--color-error, #ff4d4f);
}

/* 节点展开详情区域 */
.tree-node-details {
  margin-top: 8px;
  padding-left: 0;
  font-size: 13px;
  line-height: 1.65;
  color: var(--color-text-secondary, #595959);
  word-break: break-word;
}

.thinking-content {
  color: var(--color-text-secondary, #595959);
}

.thinking-content :deep(p) {
  margin: 6px 0;
}

.thinking-content :deep(pre) {
  background: var(--color-bg-page, #f5f5f5);
  border: 1px solid var(--color-border-light, #e8e8e8);
  border-radius: 8px;
  padding: 8px 12px;
  margin: 8px 0;
  font-size: 12px;
  overflow-x: auto;
}

.thinking-content :deep(code) {
  font-family: ui-monospace, SFMono-Regular, Consolas, monospace;
  font-size: 0.9em;
  background: var(--color-bg-hover, #f0f0f0);
  padding: 1px 4px;
  border-radius: 3px;
}

.thinking-content :deep(pre code) {
  background: none;
  padding: 0;
}
</style>
