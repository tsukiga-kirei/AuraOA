<script setup lang="ts">
import type { AIRecheckConfig } from '~/types/ai-recheck'
const props = defineProps<{ modelValue?: AIRecheckConfig; defaultSystem: string; defaultUser: string }>()
const emit = defineEmits<{ 'update:modelValue': [AIRecheckConfig] }>()
const { t } = useI18n()
const config = computed(() => props.modelValue || { enabled: false, system_prompt: props.defaultSystem, user_prompt: props.defaultUser })
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
      <PromptTextarea :value="config.system_prompt" :rows="6" @update:value="value => update({ system_prompt: value })" />
      <label>{{ t('recheck.userPrompt') }}</label>
      <PromptTextarea :value="config.user_prompt" :rows="8" @update:value="value => update({ user_prompt: value })" />
      <p>{{ t('recheck.variables') }}</p>
    </div>
  </section>
</template>

<style scoped>
.recheck-config { margin-bottom:24px; padding:16px; border:1px solid var(--color-border-light); border-radius:var(--radius-lg); background:var(--color-bg-card); }
.recheck-heading { display:flex; justify-content:space-between; align-items:center; gap:16px; }
p { margin:6px 0 0; color:var(--color-text-secondary); font-size:13px; }
.recheck-prompts { display:grid; gap:12px; margin-top:18px; }
label { font-weight:600; }
@media(max-width:600px) { .recheck-heading { align-items:flex-start; } }
</style>
