<script setup lang="ts">
import type { LLMLogDetail } from '~/types/admin-data'
const props = defineProps<{ item: LLMLogDetail }>()
const { t } = useI18n()
const output = computed(() => {
  try {
    const raw = props.item.response_content.trim().replace(/^```(?:json)?\s*/i, '').replace(/\s*```$/, '')
    const value = JSON.parse(raw)
    return value && typeof value === 'object' && !Array.isArray(value) ? value : null
  } catch { return null }
})
const rows = computed<{ condition: string; content: string; refs: string[] }[]>(() => {
  const data = output.value?.items || output.value?.rules
  if (!Array.isArray(data)) return []
  return data.filter(row => row && typeof row === 'object').map(row => ({
    condition: typeof (row.condition ?? row.applicability_condition) === 'string' ? (row.condition ?? row.applicability_condition) : '',
    content: typeof (row.content ?? row.rule_content) === 'string' ? (row.content ?? row.rule_content) : '',
    refs: Array.isArray(row.source_refs) ? row.source_refs.filter((ref: unknown) => typeof ref === 'string') : [],
  }))
})
const warnings = computed(() => Array.isArray(output.value?.warnings) ? output.value.warnings.filter((row: unknown) => typeof row === 'string') : [])
</script>
<template>
  <div class="operation-detail">
    <a-alert type="info" show-icon :message="t('admin.data.llmOperationHint')" />
    <div class="operation-usage"><span>{{ t('admin.data.llmInputTokens') }} <strong>{{ item.input_tokens }}</strong></span><span>{{ t('admin.data.llmOutputTokens') }} <strong>{{ item.output_tokens }}</strong></span></div>
    <h4>{{ t('admin.data.llmOperationResult') }}</h4>
    <CopyableContent v-for="(warning, index) in warnings" :key="index" :text="warning">
      <a-alert type="warning" show-icon :message="warning" class="operation-warning" />
    </CopyableContent>
    <template v-if="typeof output?.sql === 'string'">
      <h5>{{ t('admin.data.llmSQL') }}</h5>
      <CopyableContent :text="output.sql"><pre>{{ output.sql }}</pre></CopyableContent>
      <CopyableContent v-if="typeof output.explanation === 'string'" :text="output.explanation">
        <p>{{ output.explanation }}</p>
      </CopyableContent>
    </template>
    <article v-for="(row, index) in rows" :key="index" class="operation-row">
      <strong>{{ index + 1 }}.</strong>
      <CopyableContent :text="[row.condition && `${t('feedbackMemory.condition')}：${row.condition}`, row.content, row.refs.length && `${t('feedbackMemory.basedOn')}：${row.refs.join(' · ')}`].filter(Boolean).join('\n')">
        <p v-if="row.condition" class="operation-condition">{{ t('feedbackMemory.condition') }}：{{ row.condition }}</p>
        <p>{{ row.content }}</p>
        <small v-if="row.refs.length">{{ t('feedbackMemory.basedOn') }}：{{ row.refs.join(' · ') }}</small>
      </CopyableContent>
    </article>
    <CopyableContent v-if="!rows.length && typeof output?.sql !== 'string'" :text="item.response_content">
      <pre>{{ item.response_content || t('admin.data.llmNoPrompt') }}</pre>
    </CopyableContent>
    <a-collapse ghost>
      <a-collapse-panel key="system" :header="t('admin.data.llmSystemPrompt')">
        <CopyableContent :text="item.system_prompt"><pre>{{ item.system_prompt }}</pre></CopyableContent>
      </a-collapse-panel>
      <a-collapse-panel key="input" :header="t('admin.data.llmUserPrompt')">
        <CopyableContent :text="item.user_prompt"><pre>{{ item.user_prompt }}</pre></CopyableContent>
      </a-collapse-panel>
      <a-collapse-panel v-if="item.reasoning_content" key="reasoning" :header="t('admin.data.llmReasoningContent')">
        <CopyableContent :text="item.reasoning_content"><pre>{{ item.reasoning_content }}</pre></CopyableContent>
      </a-collapse-panel>
      <a-collapse-panel key="response" :header="t('admin.data.llmRawResponse')">
        <CopyableContent :text="item.response_content"><pre>{{ item.response_content }}</pre></CopyableContent>
      </a-collapse-panel>
    </a-collapse>
  </div>
</template>
<style scoped>
.operation-detail { min-width: 0; }
.operation-usage { display: flex; gap: 24px; flex-wrap: wrap; margin: 18px 0; color: var(--color-text-secondary); }
.operation-usage strong { color: var(--color-primary); margin-left: 8px; }
.operation-row { display: flex; align-items: flex-start; gap: 12px; margin: 12px 0; padding: 14px; border: 1px solid var(--color-border-light); border-radius: var(--radius-md); }
.operation-row p { white-space: pre-wrap; overflow-wrap: anywhere; margin: 0 0 6px; line-height: 1.8; }
.operation-condition, small { color: var(--color-text-tertiary); font-size: 12px; overflow-wrap: anywhere; }
.operation-warning { margin-bottom: 12px; }
pre { font: 12px/1.8 var(--font-mono, monospace); padding: 14px; background: var(--color-bg-hover); border: 1px solid var(--color-border-light); border-radius: var(--radius-md); white-space: pre-wrap; overflow-wrap: anywhere; max-height: 480px; overflow: auto; }
</style>
