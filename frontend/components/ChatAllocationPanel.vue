<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TenantChatAllocationItem } from '~/types/chat'
const props = defineProps<{ tenantId: string; models: { id: string; name?: string; model_name?: string }[] }>()
const { authFetch } = useAuth()
const { t } = useI18n()
const allocation = ref<TenantChatAllocationItem | null>(null)
const loading = ref(false)
const error = ref('')
watch(() => props.tenantId, async id => {
  if (!id) return
  allocation.value = null; loading.value = true; error.value = ''
  try {
    const value = await authFetch<TenantChatAllocationItem>(`/api/admin/tenants/${id}/chat-allocation`)
    if (props.tenantId === id) allocation.value = value
  } catch (err: any) { error.value = err.message }
  finally { loading.value = false }
}, { immediate: true })
const save = async (silent = false) => {
  if (!allocation.value) return
  loading.value = true
  try {
    await authFetch(`/api/admin/tenants/${props.tenantId}/chat-allocation`, {
      method: 'PUT',
      body: {
        chat_enabled: allocation.value.chat_enabled,
        chat_retention_days: allocation.value.chat_retention_days,
        chat_primary_model_id: allocation.value.chat_primary_model_id || null,
        chat_fallback_model_id: allocation.value.chat_fallback_model_id || null,
        allow_custom_skills: allocation.value.allow_custom_skills,
        allow_tenant_mcp: allocation.value.allow_tenant_mcp,
        max_mcp_servers: allocation.value.max_mcp_servers,
      },
    })
    if (!silent) message.success(t('agentAdmin.saveSuccess'))
  } catch (err: any) {
    message.error(err.message)
    throw err
  } finally { loading.value = false }
}
const modelOptions = computed(() => props.models.map(model => ({ value: model.id, label: model.name || model.model_name })))
defineExpose({ save })
</script>

<template>
  <a-spin :spinning="loading">
    <a-alert v-if="error" :message="error" type="error" />
    <a-form v-if="allocation" layout="vertical" class="allocation-form">
      <a-form-item :label="t('chat.allocation.enabled')"><a-switch v-model:checked="allocation.chat_enabled" /></a-form-item>
      <a-form-item :label="t('chat.allocation.retention')"><a-input-number v-model:value="allocation.chat_retention_days" :min="1" :max="3650" /></a-form-item>
      <a-form-item :label="t('chat.allocation.primaryModel')"><a-select :value="allocation.chat_primary_model_id || undefined" @change="value => allocation && (allocation.chat_primary_model_id = value ? String(value) : null)" allow-clear :options="modelOptions" :placeholder="t('chat.allocation.defaultModel')" /></a-form-item>
      <a-form-item :label="t('chat.allocation.fallbackModel')"><a-select :value="allocation.chat_fallback_model_id || undefined" @change="value => allocation && (allocation.chat_fallback_model_id = value ? String(value) : null)" allow-clear :options="modelOptions" :placeholder="t('chat.allocation.defaultModel')" /></a-form-item>
      <a-form-item :label="t('chat.allocation.customSkills')"><a-switch v-model:checked="allocation.allow_custom_skills" /></a-form-item>
      <a-form-item :label="t('chat.allocation.mcp')"><a-switch v-model:checked="allocation.allow_tenant_mcp" /></a-form-item>
      <a-form-item v-if="allocation.allow_tenant_mcp" :label="t('chat.allocation.maxMCP')"><a-input-number v-model:value="allocation.max_mcp_servers" :min="0" :max="100" /></a-form-item>
      <p class="perm-hint">{{ t('chat.allocation.hint') }}</p>
    </a-form>
  </a-spin>
</template>

<style scoped>
.allocation-form { padding: 22px 4px; }
.perm-hint { font-size: 12px; color: var(--color-text-tertiary); margin: 0; }
</style>
