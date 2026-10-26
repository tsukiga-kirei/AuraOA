<script setup lang="ts">
import type { AIRecheckConfig } from '~/types/ai-recheck'
const props = defineProps<{ modelValue?: AIRecheckConfig; defaultSystem: string; defaultUser: string }>()
const emit = defineEmits<{ 'update:modelValue': [AIRecheckConfig] }>()
const { t } = useI18n()
const { systemPromptVariables } = usePromptSystemVariables()
const systemTextareaRef = ref<{ insertAtCursor: (variable: string) => void } | null>(null)
const userTextareaRef = ref<{ insertAtCursor: (variable: string) => void } | null>(null)
const dataVariables = computed(() => [
  { key: '{{initial_result}}', desc: t('recheck.varInitialResult') },
  { key: '{{reasoning_result}}', desc: t('recheck.varReasoningResult') },
  { key: '{{audit_context}}', desc: t('recheck.varAuditContext') },
  { key: '{{original_system_prompt}}', desc: t('recheck.varOriginalSystemPrompt') },
])
const config = computed(() => props.modelValue || { enabled: false, system_prompt: props.defaultSystem, user_prompt: props.defaultUser })
const outputFormat = computed(() => `{\n  "reason": ${JSON.stringify(t('recheck.outputReasonExample'))},\n  "final_result": {{initial_result}}\n}`)
function update(values: Partial<AIRecheckConfig>) {
  emit('update:modelValue', { ...config.value, ...values })
}
</script>

<template>
  <section class="recheck-config">
    <div class="recheck-heading">
      <div><strong>{{ t('recheck.enable') }}</strong><p>{{ t('recheck.configDescription') }}</p></div>
      <a-switch :checked="config.enabled" :aria-label="t('recheck.enable')" @change="value => update({ enabled: !!value })" />
    </div>
    <div v-if="config.enabled" class="recheck-prompts">
      <div class="recheck-heading"><span>{{ t('recheck.promptDescription') }}</span><a-button size="small" @click="update({ system_prompt: defaultSystem, user_prompt: defaultUser })">{{ t('recheck.reset') }}</a-button></div>
      <label>{{ t('recheck.systemPrompt') }}</label>
      <PromptVariableBar :data-variables="dataVariables" :system-variables="systemPromptVariables" @insert="variable => systemTextareaRef?.insertAtCursor(variable)" />
      <PromptTextarea ref="systemTextareaRef" :value="config.system_prompt" :rows="6" :dialog-title="t('recheck.systemPrompt')" @update:value="value => update({ system_prompt: value })" />
      <div class="recheck-output-format" role="note">
        <strong>{{ t('recheck.outputTitle') }}</strong>
        <p>{{ t('recheck.outputDescription') }}</p>
        <pre>{{ outputFormat }}</pre>
        <p>{{ t('recheck.outputFields') }}</p>
        <p>{{ t('recheck.outputValidation') }}</p>
      </div>
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
.recheck-output-format { min-width:0; padding:12px; border:1px solid var(--color-border-light); border-radius:var(--radius-md); background:var(--color-bg-hover); }
.recheck-output-format pre { margin:10px 0; white-space:pre-wrap; overflow-wrap:anywhere; color:var(--color-text-primary); }
label { font-weight:600; }
@media(max-width:600px) { .recheck-heading { align-items:flex-start; } }
</style>
