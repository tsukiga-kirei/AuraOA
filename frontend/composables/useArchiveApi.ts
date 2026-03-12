// useArchiveApi — 归档复盘配置 API 调用封装

import type {
  ProcessArchiveConfig,
  ArchiveRule,
  SystemPromptTemplate,
  ProcessInfo,
  ProcessFields,
} from '~/types/rules'

export const useArchiveApi = () => {
  const { authFetch } = useAuth()

  // ============================================================
  // 归档复盘配置
  // ============================================================

  async function listConfigs(): Promise<ProcessArchiveConfig[]> {
    return await authFetch<ProcessArchiveConfig[]>('/api/tenant/archive/configs')
  }

  async function createConfig(config: Partial<ProcessArchiveConfig>): Promise<ProcessArchiveConfig> {
    return await authFetch<ProcessArchiveConfig>('/api/tenant/archive/configs', {
      method: 'POST',
      body: config,
    })
  }

  async function getConfigById(id: string): Promise<ProcessArchiveConfig> {
    return await authFetch<ProcessArchiveConfig>(`/api/tenant/archive/configs/${id}`)
  }

  async function updateConfig(id: string, config: Partial<ProcessArchiveConfig>): Promise<ProcessArchiveConfig> {
    return await authFetch<ProcessArchiveConfig>(`/api/tenant/archive/configs/${id}`, {
      method: 'PUT',
      body: config,
    })
  }

  async function deleteConfig(id: string): Promise<void> {
    await authFetch<null>(`/api/tenant/archive/configs/${id}`, { method: 'DELETE' })
  }

  async function testConnection(processType: string, mainTableName?: string, processTypeLabel?: string): Promise<ProcessInfo> {
    return await authFetch<ProcessInfo>('/api/tenant/archive/configs/test-connection', {
      method: 'POST',
      body: {
        process_type: processType,
        main_table_name: mainTableName || '',
        process_type_label: processTypeLabel || '',
      },
    })
  }

  async function fetchFields(configId: string): Promise<ProcessFields> {
    return await authFetch<ProcessFields>(`/api/tenant/archive/configs/${configId}/fetch-fields`, {
      method: 'POST',
    })
  }

  // ============================================================
  // 归档规则
  // ============================================================

  async function listRules(configId: string, ruleScope?: string, enabled?: boolean): Promise<ArchiveRule[]> {
    const params = new URLSearchParams({ config_id: configId })
    if (ruleScope) params.set('rule_scope', ruleScope)
    if (enabled !== undefined) params.set('enabled', String(enabled))
    return await authFetch<ArchiveRule[]>(`/api/tenant/archive/audit-rules?${params.toString()}`)
  }

  async function createRule(rule: Partial<ArchiveRule>): Promise<ArchiveRule> {
    return await authFetch<ArchiveRule>('/api/tenant/archive/audit-rules', {
      method: 'POST',
      body: rule,
    })
  }

  async function updateRule(id: string, rule: Partial<ArchiveRule>): Promise<ArchiveRule> {
    return await authFetch<ArchiveRule>(`/api/tenant/archive/audit-rules/${id}`, {
      method: 'PUT',
      body: rule,
    })
  }

  async function deleteRule(id: string): Promise<void> {
    await authFetch<null>(`/api/tenant/archive/audit-rules/${id}`, { method: 'DELETE' })
  }

  // ============================================================
  // 归档专用系统提示词模板（archive_ 前缀）
  // ============================================================

  async function listPromptTemplates(): Promise<SystemPromptTemplate[]> {
    return await authFetch<SystemPromptTemplate[]>('/api/tenant/archive/prompt-templates')
  }

  return {
    listConfigs,
    createConfig,
    getConfigById,
    updateConfig,
    deleteConfig,
    testConnection,
    fetchFields,
    listRules,
    createRule,
    updateRule,
    deleteRule,
    listPromptTemplates,
  }
}
