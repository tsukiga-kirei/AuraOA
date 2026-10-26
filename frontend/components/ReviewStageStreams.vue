<script setup lang="ts">
import { BulbOutlined, ExperimentOutlined, CodeOutlined, SafetyCertificateOutlined, DownOutlined, LoadingOutlined } from '@ant-design/icons-vue'
const props = withDefaults(defineProps<{
  result: { id?: string; status?: string; deep_thinking?: string; ai_reasoning?: string; structured_output?: string; recheck_output?: string }
  pageFlow?: boolean
  expansion?: Record<string, boolean>
}>(), { pageFlow: false })
const emit = defineEmits<{ 'update:expansion': [value: Record<string, boolean>] }>()
const { t } = useI18n()
const definitions = computed(() => [
  { key: 'thinking', text: props.result.deep_thinking || '', title: t('dashboard.deepThinking'), icon: BulbOutlined, format: 'markdown' as const },
  { key: 'reasoning', text: props.result.ai_reasoning || '', title: t('dashboard.aiReasoning'), icon: ExperimentOutlined, format: 'markdown' as const },
  { key: 'structured', text: props.result.structured_output || '', title: t('embed.auditProgress.extracting'), icon: CodeOutlined, format: 'json' as const },
  { key: 'recheck', text: props.result.recheck_output || '', title: t('recheck.title'), icon: SafetyCertificateOutlined, format: 'json' as const },
])
const latest = computed(() => props.result.status === 'rechecking' ? 'recheck' : props.result.status === 'extracting' ? 'structured' : [...definitions.value].reverse().find(item => item.text)?.key || 'reasoning')
const localExpansion = ref<Record<string, boolean>>({})
const manualExpansion = computed(() => props.expansion ?? localExpansion.value)
// 仅未操作的面板跟随当前阶段；手动展开和收起均持续保留。
function isExpanded(stage: string) {
  return manualExpansion.value[stage] ?? latest.value === stage
}
function toggle(stage: string) {
  const next = { ...manualExpansion.value, [stage]: !isExpanded(stage) }
  localExpansion.value = next
  emit('update:expansion', next)
}
watch(() => props.result.id, (id, previous) => {
  if (id && previous && id !== previous) {
    localExpansion.value = {}
    emit('update:expansion', {})
  }
})
const visibleStages = computed(() => definitions.value.filter(item => item.text || (item.key === 'reasoning' && ['reasoning', 'assembling'].includes(props.result.status || '')) || (item.key === 'structured' && props.result.status === 'extracting') || (item.key === 'recheck' && props.result.status === 'rechecking')))
</script>

<template>
  <div class="review-stage-streams" :class="{ 'review-stage-streams--page-flow': pageFlow }">
    <section v-for="stage in visibleStages" :key="stage.key" class="review-stage" :class="{ 'review-stage--expanded': isExpanded(stage.key) }">
      <button type="button" class="review-stage__toggle" :aria-expanded="isExpanded(stage.key)" @click="toggle(stage.key)">
        <span class="review-stage__title"><component :is="stage.icon" />{{ stage.title }}</span>
        <span class="review-stage__action"><LoadingOutlined v-if="latest === stage.key" spin /><DownOutlined :class="{ 'review-stage__arrow--open': isExpanded(stage.key) }" /></span>
      </button>
      <div v-if="isExpanded(stage.key)" class="review-stage__body">
        <AiMarkdownStream :text="stage.text" :format="stage.format" :page-flow="pageFlow" :max-height="'320px'" />
      </div>
    </section>
  </div>
</template>

<style scoped>
.review-stage-streams { width:100%; min-width:0; margin-top:16px; display:grid; gap:10px; }
.review-stage { min-width:0; border:1px solid var(--color-border-light); border-radius:var(--radius-md); background:var(--color-bg-card); overflow:hidden; }
.review-stage__toggle { width:100%; border:0; padding:12px 14px; background:transparent; color:var(--color-text-primary); display:flex; justify-content:space-between; align-items:center; gap:12px; text-align:left; cursor:pointer; font:inherit; }
.review-stage__title { display:flex; align-items:center; gap:8px; font-size:13.5px; font-weight:600; }
.review-stage__title :deep(.anticon), .review-stage__action { color:var(--color-primary); }
.review-stage__action { display:flex; align-items:center; gap:10px; font-size:12px; }
.review-stage__arrow--open { transform:rotate(180deg); }
.review-stage__body { padding:0 12px 12px; text-align:left; }
.review-stage__body :deep(.ai-stream) { margin:0; border:0; }
.review-stage__toggle:focus-visible { outline:2px solid var(--color-primary); outline-offset:-2px; }
.review-stage-streams--page-flow .review-stage { overflow:visible; border-radius:12px; }
.review-stage-streams--page-flow .review-stage__toggle { position:sticky; top:0; z-index:4; border-radius:12px; background:var(--color-bg-card); }
.review-stage-streams--page-flow .review-stage--expanded .review-stage__toggle { border-radius:12px 12px 0 0; box-shadow:0 3px 8px rgb(15 23 42 / 3%); }
.review-stage-streams--page-flow .review-stage__body { border-radius:0 0 12px 12px; }
</style>
