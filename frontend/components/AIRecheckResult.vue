<script setup lang="ts">
import type { AIRecheckRecord } from '~/types/ai-recheck'
const props = defineProps<{ record?: AIRecheckRecord | null; taskStatus?: string }>()
const { t } = useI18n()
const state = computed(() => props.record?.status === 'running' && ['failed', 'cancelled'].includes(props.taskStatus || '') ? props.taskStatus : props.record?.status)
const originalRules = computed(() => (props.record?.original_result?.rule_results || props.record?.original_result?.rule_audit || []).map((rule: Record<string, any>, index: number) => ({ id: index, name: rule.rule_content || rule.rule_name, passed: !!rule.passed, reason: rule.reason || rule.reasoning })))
function label(result?: Record<string, any>) {
  const value = result?.recommendation || result?.overall_compliance
  return value ? t(`recheck.conclusion.${value}`) : '—'
}
</script>

<template>
  <section v-if="record?.status" class="recheck-result">
    <div class="recheck-title"><strong>{{ t('recheck.title') }}</strong><a-tag :color="state === 'completed' ? 'blue' : state === 'running' ? 'processing' : 'orange'">{{ t(`recheck.status.${state}`) }}</a-tag></div>
    <p v-if="state === 'completed'">{{ record.changed ? t('recheck.adjusted') : t('recheck.maintained') }} · {{ ((record.duration_ms || 0) / 1000).toFixed(1) }}s</p>
    <p v-if="record.reason" class="recheck-copy">{{ record.reason }}</p>
    <p v-if="state === 'failed' || state === 'cancelled'">{{ record.error || t('recheck.incomplete') }}</p>
    <details v-if="record.original_result">
      <summary>{{ t('recheck.compare') }}</summary>
      <div class="recheck-comparison">
        <div><strong>{{ t('recheck.initial') }}</strong><p>{{ label(record.original_result) }} · {{ record.original_result.overall_score }}</p></div>
        <div v-if="record.final_result"><strong>{{ t('recheck.final') }}</strong><p>{{ label(record.final_result) }} · {{ record.final_result.overall_score }}</p></div>
      </div>
      <details>
        <summary>{{ t('recheck.initialDetails') }}</summary>
        <ResultRuleDetails :title="t('recheck.initial')" :rules="originalRules" />
        <p v-if="record.original_result.ai_summary">{{ record.original_result.ai_summary }}</p>
        <template v-if="record.original_result.risk_points?.length"><strong>{{ t('recheck.initialRisks') }}</strong><ul><li v-for="(risk, index) in record.original_result.risk_points" :key="index">{{ risk }}</li></ul></template>
        <template v-if="record.original_result.suggestions?.length"><strong>{{ t('recheck.initialSuggestions') }}</strong><ul><li v-for="(suggestion, index) in record.original_result.suggestions" :key="index">{{ suggestion }}</li></ul></template>
      </details>
    </details>
  </section>
</template>

<style scoped>
.recheck-result { margin:16px 0; padding:16px; border:1px solid var(--color-border-light); border-radius:var(--radius-lg); background:var(--color-bg-card); }
.recheck-title,.recheck-comparison { display:flex; gap:16px; align-items:center; flex-wrap:wrap; }
.recheck-comparison { align-items:flex-start; margin:12px 0; }
.recheck-copy { white-space:pre-wrap; overflow-wrap:anywhere; }
p { margin:8px 0; } summary { cursor:pointer; color:var(--color-text-secondary); }
</style>
