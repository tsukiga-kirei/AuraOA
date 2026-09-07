<script setup lang="ts">
import { CheckOutlined, LoadingOutlined, CloseOutlined, DownOutlined, ApiOutlined, BookOutlined, SearchOutlined } from '@ant-design/icons-vue'
import type { ChatToolExecution } from '~/types/chat'
import { renderSafeMarkdown } from '~/utils/markdown'
const props = defineProps<{ tool: ChatToolExecution }>()
const { t, te } = useI18n()
const { loadOAJumpConfig, jumpToOA, canJumpToOA } = useOAJump()
const kind = computed(() => {
  const code = props.tool?.tool_code || ''
  return code.startsWith('skill:') ? 'skill' : code.startsWith('mcp:') ? 'mcp' : 'system'
})
const label = computed(() => {
  const code = props.tool?.tool_code || ''
  return te(`chat.tools.${code}`) ? t(`chat.tools.${code}`) : code.replace(/^(skill:|mcp:)/, '').replace(/:/g, ' / ')
})
const statusKey = computed(() => props.tool?.status && ['running', 'success', 'error'].includes(props.tool.status) ? props.tool.status : 'running')
const rows = computed(() => Array.isArray(props.tool.payload?.items) ? props.tool.payload.items : [])
const text = computed(() => {
  const result = props.tool.payload
  if (typeof result === 'string') return result
  if (Array.isArray(result?.content)) return result.content.filter((item: any) => item.type === 'text').map((item: any) => item.text).join('\n\n')
  return result?.summary || result?.content || result?.message || ''
})
onMounted(() => { if (props.tool.tool_code === 'list_my_todos') loadOAJumpConfig() })
</script>

<template>
  <details v-if="tool" class="tool-activity" :class="[kind, tool.status]">
    <summary>
      <span class="activity-icon"><BookOutlined v-if="kind === 'skill'" /><ApiOutlined v-else-if="kind === 'mcp'" /><SearchOutlined v-else /></span>
      <span class="activity-label">{{ label }}<small>{{ t(`chat.toolKind.${kind}`) }}</small></span>
      <span class="activity-state"><LoadingOutlined v-if="statusKey === 'running'" spin /><CloseOutlined v-else-if="statusKey === 'error'" /><CheckOutlined v-else />{{ t(`chat.toolStatus.${statusKey}`) }}</span>
      <DownOutlined class="activity-chevron" />
    </summary>
    <div class="activity-body">
      <p v-if="tool.status === 'error'" role="alert">{{ tool.payload?.error || t('chat.toolInterrupted') }}</p>
      <div v-else-if="rows.length" class="result-list">
        <div v-for="(item, index) in rows" :key="item.process_id || index" class="result-row">
          <div><strong>{{ item.title || item.requestname || item.process_id }}</strong><p>{{ [item.applicant_name || item.creator_name, item.process_type_name || item.workflow_name, item.current_node_name].filter(Boolean).join(' · ') }}</p></div>
          <button v-if="canJumpToOA && item.process_id" @click="jumpToOA(item.process_id)">{{ t('chat.openProcess') }}</button>
        </div>
      </div>
      <div v-else-if="typeof text === 'string' && text" class="activity-markdown" v-html="renderSafeMarkdown(text)" />
      <p v-else-if="tool.tool_code === 'list_my_todos' && tool.status === 'success'">{{ t('chat.todoCard.empty') }}</p>
      <pre v-else-if="tool.payload">{{ JSON.stringify(tool.payload, null, 2) }}</pre>
      <details v-if="tool.arguments" class="activity-arguments"><summary>{{ t('chat.toolArguments') }}</summary><pre>{{ tool.arguments }}</pre></details>
    </div>
  </details>
</template>

<style scoped>
.tool-activity { margin:0; background:transparent; font-size:12px; color:var(--color-text-secondary); }
summary { display:flex; align-items:center; gap:8px; cursor:pointer; list-style:none; min-height:30px; padding:5px 6px; border-radius:6px; }
summary:hover { background:var(--color-bg-hover); }
summary::-webkit-details-marker { display:none; }
.activity-icon { display:grid; place-items:center; width:16px; height:16px; flex-shrink:0; color:var(--color-text-tertiary); }
.activity-label { display:flex; align-items:center; flex:1; flex-wrap:wrap; gap:8px; font-size:12px; font-weight:500; min-width:0; overflow-wrap:anywhere; }
.activity-label small { color:var(--color-text-tertiary); font-weight:400; font-size:10px; }
.activity-state { display:flex; align-items:center; gap:5px; color:var(--color-text-secondary); font-size:11px; white-space:nowrap; }
.activity-chevron { color:var(--color-text-tertiary); font-size:9px; }.tool-activity[open] .activity-chevron { transform:rotate(180deg); }
.activity-body { margin:2px 6px 8px 13px; border-left:1px solid var(--color-border); padding:6px 16px; color:var(--color-text-secondary); max-height:300px; overflow:auto; }
.activity-body p { margin:4px 0; }.error .activity-state { color:var(--color-error, #c45050); }.running .activity-state { color:var(--color-primary); }
.activity-body pre { white-space:pre-wrap; overflow-wrap:anywhere; max-height:300px; overflow:auto; font-size:11px; line-height:1.7; }
.result-row { display:flex; align-items:center; justify-content:space-between; gap:10px; padding:9px 0; border-bottom:1px solid var(--color-border-light); }.result-row:last-child { border:0; }
.result-row strong { font-weight:500; color:var(--color-text-primary); }.result-row p { margin:4px 0 0; font-size:11px; color:var(--color-text-tertiary); }
.result-row button { background:none; border:0; color:var(--color-primary); white-space:nowrap; font-size:11px; cursor:pointer; }
.activity-arguments { margin-top:10px; color:var(--color-text-tertiary); }.activity-arguments summary { padding:0; font-size:11px; }
.activity-markdown { line-height:1.8; overflow-wrap:anywhere; }.activity-markdown :deep(pre) { overflow-x:auto; }
</style>
