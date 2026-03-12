// useRulesApi — 封装规则配置相关 API 调用

import type {
  ProcessAuditConfig,
  AuditRule,
  SystemPromptTemplate,
  ProcessInfo,
  ProcessFields,
} from '~/types/rules'

export type { ProcessAuditConfig, AuditRule, SystemPromptTemplate, ProcessInfo, ProcessFields }

export const useRulesApi = () => {
  const { authFetch } = useAuth()

  // ============================================================
  // 流程审核配置
  // ============================================================

  async function listConfigs(): Promise<ProcessAuditConfig[]> {
    return await authFetch<ProcessAuditConfig[]>('/api/tenant/rules/configs')
  }

  async function createConfig(config: Partial<ProcessAuditConfig>): Promise<ProcessAuditConfig> {
    return await authFetch<ProcessAuditConfig>('/api/tenant/rules/configs', { method: 'POST', body: config })
  }

  async function updateConfig(id: string, config: Partial<ProcessAuditConfig>): Promise<ProcessAuditConfig> {
    return await authFetch<ProcessAuditConfig>(`/api/tenant/rules/configs/${id}`, { method: 'PUT', body: config })
  }

  async function deleteConfig(id: string): Promise<void> {
    await authFetch<null>(`/api/tenant/rules/configs/${id}`, { method: 'DELETE' })
  }

  async function testConnection(processType: string, mainTableName?: string, processTypeLabel?: string): Promise<ProcessInfo> {
    return await authFetch<ProcessInfo>('/api/tenant/rules/configs/test-connection', {
      method: 'POST',
      body: { process_type: processType, main_table_name: mainTableName || '', process_type_label: processTypeLabel || '' },
    })
  }

  async function fetchFields(configId: string): Promise<ProcessFields> {
    return await authFetch<ProcessFields>(`/api/tenant/rules/configs/${configId}/fetch-fields`, { method: 'POST' })
  }

  // ============================================================
  // 审核规则
  // ============================================================

  async function listRules(configId: string, ruleScope?: string, enabled?: boolean): Promise<AuditRule[]> {
    const params = new URLSearchParams({ config_id: configId })
    if (ruleScope) params.set('rule_scope', ruleScope)
    if (enabled !== undefined) params.set('enabled', String(enabled))
    return await authFetch<AuditRule[]>(`/api/tenant/rules/audit-rules?${params.toString()}`)
  }

  async function createRule(rule: Partial<AuditRule>): Promise<AuditRule> {
    return await authFetch<AuditRule>('/api/tenant/rules/audit-rules', { method: 'POST', body: rule })
  }

  async function updateRule(id: string, rule: Partial<AuditRule>): Promise<AuditRule> {
    return await authFetch<AuditRule>(`/api/tenant/rules/audit-rules/${id}`, { method: 'PUT', body: rule })
  }

  async function deleteRule(id: string): Promise<void> {
    await authFetch<null>(`/api/tenant/rules/audit-rules/${id}`, { method: 'DELETE' })
  }

  // ============================================================
  // 系统提示词模板
  // ============================================================

  async function listPromptTemplates(): Promise<SystemPromptTemplate[]> {
    return await authFetch<SystemPromptTemplate[]>('/api/tenant/rules/prompt-templates')
  }

  return {
    listConfigs, createConfig, updateConfig, deleteConfig,
    testConnection, fetchFields,
    listRules, createRule, updateRule, deleteRule,
    listPromptTemplates,
  }
}
