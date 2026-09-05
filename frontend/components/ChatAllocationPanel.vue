<script setup lang="ts">
import { message } from 'ant-design-vue'
import type { TenantChatAllocationItem, AgentCatalogResponse } from '~/types/chat'
const props = defineProps<{ tenantId: string; models: { id: string; name?: string; model_name?: string }[] }>()
const { authFetch } = useAuth()
const { t } = useI18n()
const allocation = ref<TenantChatAllocationItem | null>(null)
const catalog = ref<AgentCatalogResponse | null>(null)
const loading = ref(false)
const error = ref('')
watch(() => props.tenantId, async id => {
  allocation.value = null; loading.value = true; error.value = ''
  try {
    const [value, items] = await Promise.all([authFetch<TenantChatAllocationItem>(`/api/admin/tenants/${id}/chat-allocation`), authFetch<AgentCatalogResponse>('/api/admin/agent-catalog')])
    if (props.tenantId === id) { allocation.value = value; catalog.value = items }
  } catch (err: any) { error.value = err.message }
  finally { loading.value = false }
}, { immediate: true })
const save = async () => {
  if (!allocation.value) return
  loading.value = true
  try { await authFetch(`/api/admin/tenants/${props.tenantId}/chat-allocation`, { method: 'PUT', body: { ...allocation.value, chat_primary_model_id: allocation.value.chat_primary_model_id || null, chat_fallback_model_id: allocation.value.chat_fallback_model_id || null } }); message.success(t('agentAdmin.saveSuccess')) }
  catch (err: any) { message.error(err.message) }
  finally { loading.value = false }
}
const modelOptions = computed(() => props.models.map(model => ({ value: model.id, label: model.name || model.model_name })))
</script>

<template>
  <a-spin :spinning="loading">
    <a-alert v-if="error" :message="error" type="error" />
    <a-form v-if="allocation && catalog" layout="vertical" class="allocation-form">
      <a-form-item :label="t('chat.allocation.enabled')"><a-switch v-model:checked="allocation.chat_enabled" /></a-form-item>
      <a-form-item :label="t('chat.allocation.retention')"><a-input-number v-model:value="allocation.chat_retention_days" :min="1" :max="3650" /></a-form-item>
      <a-form-item :label="t('chat.allocation.primaryModel')"><a-select :value="allocation.chat_primary_model_id || undefined" @change="value => allocation && (allocation.chat_primary_model_id = value ? String(value) : null)" allow-clear :options="modelOptions" :placeholder="t('chat.allocation.defaultModel')" /></a-form-item>
      <a-form-item :label="t('chat.allocation.fallbackModel')"><a-select :value="allocation.chat_fallback_model_id || undefined" @change="value => allocation && (allocation.chat_fallback_model_id = value ? String(value) : null)" allow-clear :options="modelOptions" :placeholder="t('chat.allocation.defaultModel')" /></a-form-item>
      <a-form-item :label="t('chat.agents')"><a-select v-model:value="allocation.agent_codes" mode="multiple" :options="catalog.agent_catalog.map(item => ({value:item.agent_code,label:item.name}))" /></a-form-item>
      <a-form-item :label="t('agentAdmin.form.bindTools')"><a-select v-model:value="allocation.tool_codes" mode="multiple" :options="catalog.tool_catalog.map(item => ({value:item.tool_code,label:item.name}))" /></a-form-item>
      <a-form-item :label="t('agentAdmin.tab.skills')"><a-select v-model:value="allocation.skill_codes" mode="multiple" :options="catalog.skill_catalog.map(item => ({value:item.skill_code,label:item.name}))" /></a-form-item>
      <a-form-item :label="t('chat.allocation.customSkills')"><a-switch v-model:checked="allocation.allow_custom_skills" /></a-form-item>
      <a-form-item :label="t('chat.allocation.mcp')"><a-switch v-model:checked="allocation.allow_tenant_mcp" /></a-form-item>
      <a-form-item v-if="allocation.allow_tenant_mcp" :label="t('chat.allocation.maxMCP')"><a-input-number v-model:value="allocation.max_mcp_servers" :min="0" :max="100" /></a-form-item>
      <a-button type="primary" :loading="loading" @click="save">{{ t('admin.org.save') }}</a-button>
    </a-form>
  </a-spin>
</template>

<style scoped>.allocation-form { padding:22px 4px; }</style>
