<script setup lang="ts">
import { SafetyCertificateOutlined, CodeOutlined, InfoCircleOutlined } from '@ant-design/icons-vue'
import type { AIRecheckConfig } from '~/types/ai-recheck'
const props = defineProps<{ modelValue?: AIRecheckConfig; defaultSystem: string; defaultUser: string; outputInstructions?: string; strictness?: string; configIdentity?: string }>()
const emit = defineEmits<{ 'update:modelValue': [AIRecheckConfig] }>()
const { t } = useI18n()
const { systemPromptVariables } = usePromptSystemVariables()
const systemTextareaRef = ref<{ insertAtCursor: (variable: string) => void } | null>(null)
const userTextareaRef = ref<{ insertAtCursor: (variable: string) => void } | null>(null)
const dataVariables = computed(() => [
  { key: '{{rules}}', desc: t('recheck.varRules') },
  { key: '{{initial_result}}', desc: t('recheck.varInitialResult') },
  { key: '{{reasoning_result}}', desc: t('recheck.varReasoningResult') },
  { key: '{{audit_context}}', desc: t('recheck.varAuditContext') },
  { key: '{{original_system_prompt}}', desc: t('recheck.varOriginalSystemPrompt') },
])
const config = computed(() => props.modelValue || { enabled: false, system_prompt: props.defaultSystem, user_prompt: props.defaultUser })
function update(values: Partial<AIRecheckConfig>) {
  emit('update:modelValue', { ...config.value, ...values })
}
watch(() => [props.defaultSystem, props.strictness, props.configIdentity] as const, (next, previous) => {
  if (previous[0] && next[0] && next[1] !== previous[1] && next[2] === previous[2] && props.modelValue?.system_prompt === previous[0]) update({ system_prompt: next[0] })
})
</script>

<template>
  <section class="recheck-config">
    <div class="recheck-heading">
      <div><strong class="recheck-title"><SafetyCertificateOutlined aria-hidden="true" />{{ t('recheck.enable') }}</strong><p>{{ t('recheck.configDescription') }}</p></div>
      <a-switch :checked="config.enabled" :aria-label="t('recheck.enable')" @change="value => update({ enabled: !!value })" />
    </div>
    <div v-if="config.enabled" class="recheck-prompts">
      <div class="recheck-heading"><span>{{ t('recheck.promptDescription') }}</span><a-button size="small" @click="update({ system_prompt: defaultSystem, user_prompt: defaultUser })">{{ t('recheck.reset') }}</a-button></div>
      <label>{{ t('recheck.systemPrompt') }}</label>
      <PromptVariableBar :data-variables="dataVariables" :system-variables="systemPromptVariables" @insert="variable => systemTextareaRef?.insertAtCursor(variable)" />
      <PromptTextarea ref="systemTextareaRef" :value="config.system_prompt" :rows="6" :dialog-title="t('recheck.systemPrompt')" @update:value="value => update({ system_prompt: value })" />
      <div class="recheck-output-format">
        <div class="recheck-output-heading"><strong><CodeOutlined aria-hidden="true" />{{ t('recheck.outputTitle') }}</strong><span class="recheck-readonly">{{ t('recheck.outputReadonly') }}</span></div>
        <p>{{ t('recheck.outputDescription') }}</p>
        <pre v-if="outputInstructions" class="recheck-output-raw" tabindex="0" :aria-label="t('recheck.outputTitle')">{{ outputInstructions }}</pre>
        <p v-else class="recheck-output-unavailable">{{ t('recheck.outputUnavailable') }}</p>
      </div>
      <aside class="recheck-output-help" role="note">
        <strong><InfoCircleOutlined aria-hidden="true" />{{ t('recheck.outputHelpTitle') }}</strong>
        <p>{{ t('recheck.outputValidation') }}</p>
      </aside>
      <label>{{ t('recheck.userPrompt') }}</label>
      <PromptVariableBar :data-variables="dataVariables" :system-variables="systemPromptVariables" @insert="variable => userTextareaRef?.insertAtCursor(variable)" />
      <PromptTextarea ref="userTextareaRef" :value="config.user_prompt" :rows="8" :dialog-title="t('recheck.userPrompt')" @update:value="value => update({ user_prompt: value })" />
      <p>{{ t('recheck.variables') }}</p>
    </div>
  </section>
</template>

<style scoped>
.recheck-config { margin-bottom:24px; padding:16px; border:1px solid var(--color-border-light); border-radius:var(--radius-lg); background:var(--color-bg-card); }
.recheck-heading { display:flex; justify-content:space-between; align-items:center; gap:16px; }
p { margin:6px 0 0; color:var(--color-text-secondary); font-size:13px; }
.recheck-prompts { display:grid; gap:12px; margin-top:18px; }
.recheck-title, .recheck-output-heading, .recheck-output-heading strong, .recheck-output-help strong { display:flex; align-items:center; gap:8px; }
.recheck-title :deep(.anticon) { color:var(--color-primary); }
.recheck-output-format { min-width:0; padding:14px; border:1px solid var(--color-border); border-radius:var(--radius-md); background:var(--color-bg-card); }
.recheck-output-heading { justify-content:space-between; flex-wrap:wrap; }
.recheck-readonly { padding:2px 8px; border-radius:var(--radius-sm); color:var(--color-text-secondary); background:var(--color-bg-hover); font-size:12px; }
.recheck-output-raw { max-height:360px; margin:12px 0 0; padding:12px; overflow:auto; white-space:pre-wrap; overflow-wrap:anywhere; color:var(--color-text-primary); background:var(--color-bg-hover); border:1px solid var(--color-border-light); border-radius:var(--radius-sm); font-family:monospace; font-size:12px; line-height:1.65; }
.recheck-output-help { min-width:0; padding:12px 14px; border-radius:var(--radius-md); background:var(--color-primary-bg); border-left:3px solid var(--color-primary); }
.recheck-output-help :deep(.anticon) { color:var(--color-primary); }
label { font-weight:600; }
@media(max-width:600px) { .recheck-heading { align-items:flex-start; } }
</style>
