<script setup lang="ts">
import { SafetyCertificateOutlined, ArrowRightOutlined, DownOutlined } from '@ant-design/icons-vue'
import type { AIRecheckRecord } from '~/types/ai-recheck'
import { renderSafeMarkdown } from '~/utils/markdown'
const props = defineProps<{ record?: AIRecheckRecord | null; taskStatus?: string }>()
const { t } = useI18n()
const show = computed(() => (!props.taskStatus || props.taskStatus === 'completed') && props.record?.status === 'completed' && props.record.changed === true && !props.record.reuse_original && !!props.record.final_result)
const expanded = ref(false)
const reasonHtml = computed(() => renderSafeMarkdown(props.record?.reason || ''))
const originalRules = computed(() => (props.record?.original_result?.rule_results || props.record?.original_result?.rule_audit || []).map((rule: Record<string, any>, index: number) => ({ id: index, name: rule.rule_content || rule.rule_name, passed: !!rule.passed, reason: rule.reason || rule.reasoning })))
function label(result?: Record<string, any>) {
  const value = result?.recommendation || result?.overall_compliance
  return value ? t(`recheck.conclusion.${value}`) : '—'
}

function getConclusionTheme(result?: Record<string, any>): 'success' | 'warning' | 'danger' | 'default' {
  const value = result?.recommendation || result?.overall_compliance
  if (value === 'approve' || value === 'compliant') return 'success'
  if (value === 'review' || value === 'partially_compliant') return 'warning'
  if (value === 'return' || value === 'non_compliant') return 'danger'
  const score = result?.overall_score
  if (typeof score === 'number' && !isNaN(score)) {
    if (score >= 80) return 'success'
    if (score >= 60) return 'warning'
    return 'danger'
  }
  return 'default'
}

function getScoreTheme(result?: Record<string, any>): 'success' | 'warning' | 'danger' | 'default' {
  const score = result?.overall_score
  if (typeof score === 'number' && !isNaN(score)) {
    if (score >= 80) return 'success'
    if (score >= 60) return 'warning'
    return 'danger'
  }
  return getConclusionTheme(result)
}
</script>

<template>
  <section v-if="show" class="recheck-result">
    <header class="recheck-result__header">
      <span class="recheck-result__heading"><span class="recheck-result__icon"><SafetyCertificateOutlined /></span>{{ t('recheck.adjustmentTitle') }}</span>
      <span class="recheck-result__duration">{{ ((record?.duration_ms || 0) / 1000).toFixed(1) }}s</span>
    </header>
    <div class="recheck-result__comparison">
      <span class="recheck-result__metric" :class="`recheck-result__metric--${getConclusionTheme(record?.original_result)}`">
        <span class="recheck-result__caption">{{ t('recheck.initial') }}</span>
        <strong class="recheck-result__label" :class="`recheck-result__label--${getConclusionTheme(record?.original_result)}`">{{ label(record?.original_result) }}</strong>
        <b class="recheck-result__score" :class="`recheck-result__score--${getScoreTheme(record?.original_result)}`">{{ record?.original_result?.overall_score }}</b>
      </span>
      <ArrowRightOutlined class="recheck-result__arrow" />
      <span class="recheck-result__metric" :class="`recheck-result__metric--${getConclusionTheme(record?.final_result)}`">
        <span class="recheck-result__caption">{{ t('recheck.final') }}</span>
        <strong class="recheck-result__label" :class="`recheck-result__label--${getConclusionTheme(record?.final_result)}`">{{ label(record?.final_result) }}</strong>
        <b class="recheck-result__score" :class="`recheck-result__score--${getScoreTheme(record?.final_result)}`">{{ record?.final_result?.overall_score }}</b>
      </span>
    </div>
    <div v-if="record?.reason" class="markdown-body recheck-result__reason" v-html="reasonHtml" />
    <button v-if="record?.original_result" type="button" class="recheck-result__expand" :aria-expanded="expanded" @click="expanded = !expanded">{{ t('recheck.initialDetails') }}<DownOutlined :class="{ 'recheck-result__expanded': expanded }" /></button>
    <div v-if="expanded && record?.original_result" class="recheck-result__original">
      <ResultRuleDetails :title="t('recheck.initial')" :rules="originalRules" />
      <p v-if="record.original_result.ai_summary">{{ record.original_result.ai_summary }}</p>
      <div class="recheck-result__insights">
        <div v-if="record.original_result.risk_points?.length"><strong>{{ t('recheck.initialRisks') }}</strong><ul><li v-for="(risk,index) in record.original_result.risk_points" :key="index">{{ risk }}</li></ul></div>
        <div v-if="record.original_result.suggestions?.length"><strong>{{ t('recheck.initialSuggestions') }}</strong><ul><li v-for="(item,index) in record.original_result.suggestions" :key="index">{{ item }}</li></ul></div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.recheck-result { min-width:0; margin:0 0 14px; padding:16px; border:1px solid var(--color-border-light); border-radius:var(--radius-md); background:var(--color-bg-card); color:var(--color-text-primary); font-size:13px; line-height:1.65; }
.recheck-result__header { display:flex; align-items:center; justify-content:space-between; gap:12px; margin-bottom:12px; }
.recheck-result__heading { display:flex; align-items:center; gap:8px; font-size:13.5px; font-weight:600; }
.recheck-result__icon { display:inline-flex; align-items:center; justify-content:center; width:28px; height:28px; border-radius:8px; color:var(--color-primary); background:var(--color-primary-bg); font-size:16px; }
.recheck-result__duration,.recheck-result__caption { color:var(--color-text-tertiary); font-size:12px; font-weight:400; }
.recheck-result__comparison { display:flex; align-items:center; gap:14px; flex-wrap:wrap; padding:10px 12px; margin-bottom:12px; border-radius:var(--radius-sm); background:var(--color-bg-hover); }
.recheck-result__comparison > span { display:flex; gap:8px; align-items:baseline; flex-wrap:wrap; }
.recheck-result__comparison strong { font-size:13px; font-weight:600; }
.recheck-result__comparison b { font-size:18px; font-weight:600; }
.recheck-result__arrow { color:var(--color-text-tertiary); }
.recheck-result__label--success,
.recheck-result__score--success { color:var(--color-success); }
.recheck-result__label--warning,
.recheck-result__score--warning { color:var(--color-warning); }
.recheck-result__label--danger,
.recheck-result__score--danger { color:var(--color-danger); }
.recheck-result__reason { font-size:13px; line-height:1.75; color:var(--color-text-secondary); overflow-wrap:anywhere; }
.recheck-result__reason :deep(p) { margin:6px 0; }
.recheck-result__expand { display:flex; align-items:center; gap:6px; padding:8px 0 0; margin-top:6px; border:0; background:transparent; color:var(--color-primary); font:inherit; font-size:12px; cursor:pointer; }
.recheck-result__expanded { transform:rotate(180deg); }
.recheck-result__original { margin-top:12px; padding-top:12px; border-top:1px solid var(--color-border-light); }
.recheck-result__insights { display:grid; grid-template-columns:repeat(auto-fit,minmax(min(100%,260px),1fr)); gap:12px; }
.recheck-result__insights ul { padding-left:18px; margin:6px 0 0; color:var(--color-text-secondary); }
.recheck-result__insights strong { font-size:12px; color:var(--color-text-primary); }
.recheck-result__expand:focus-visible { outline:2px solid var(--color-primary); outline-offset:3px; }
@media(max-width:600px) { .recheck-result { padding:12px; } .recheck-result__comparison { gap:8px; } }
</style>
