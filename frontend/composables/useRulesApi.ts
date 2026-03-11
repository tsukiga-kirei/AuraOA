// useRulesApi — 封装规则配置相关 API 调用

import type {
  ProcessAuditConfig,
  AuditRule,
  StrictnessPreset,
  ProcessInfo,
  ProcessFields,
} from '~/types/rules'

export type { ProcessAuditConfig, AuditRule, StrictnessPreset, ProcessInfo, ProcessFields }

export const useRulesApi = () => {
  const { authFetch } = useAuth()

  const configs = ref<ProcessAuditConfig[]>([])
  const rules = ref<AuditRule[]>([])
  const presets = ref<StrictnessPreset[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  // ============================================================
  // 流程审核配置
  // ============================================================

  async function listConfigs(): Promise<ProcessAuditConfig[]> {
    loading.value = true
    error.value = null
    try {
      const data = await authFetch<ProcessAuditConfig[]>('/api/tenant/rules/configs')
      configs.value = data
      return data
    }
    catch (e: any) {
      error.value = e.message || '加载流程配置失败'
      throw e
    }
    finally { loading.value = false }
  }

  async function createConfig(config: Partial<ProcessAuditConfig>): Promise<ProcessAuditConfig> {
    const data = await authFetch<ProcessAuditConfig>('/api/tenant/rules/configs', { method: 'POST', body: config })
    return data
  }

  async function updateConfig(id: string, config: Partial<ProcessAuditConfig>): Promise<ProcessAuditConfig> {
    const data = await authFetch<ProcessAuditConfig>(`/api/tenant/rules/configs/${id}`, { method: 'PUT', body: config })
    return data
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
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams({ config_id: configId })
      if (ruleScope) params.set('rule_scope', ruleScope)
      if (enabled !== undefined) params.set('enabled', String(enabled))
      const data = await authFetch<AuditRule[]>(`/api/tenant/rules/audit-rules?${params.toString()}`)
      rules.value = data
      return data
    }
    catch (e: any) {
      error.value = e.message || '加载审核规则失败'
      throw e
    }
    finally { loading.value = false }
  }

  async function createRule(rule: Partial<AuditRule>): Promise<AuditRule> {
    const data = await authFetch<AuditRule>('/api/tenant/rules/audit-rules', { method: 'POST', body: rule })
    return data
  }

  async function updateRule(id: string, rule: Partial<AuditRule>): Promise<AuditRule> {
    const data = await authFetch<AuditRule>(`/api/tenant/rules/audit-rules/${id}`, { method: 'PUT', body: rule })
    return data
  }

  async function deleteRule(id: string): Promise<void> {
    await authFetch<null>(`/api/tenant/rules/audit-rules/${id}`, { method: 'DELETE' })
  }

  // ============================================================
  // 审核尺度预设
  // ============================================================

  async function listPresets(): Promise<StrictnessPreset[]> {
    loading.value = true
    error.value = null
    try {
      const data = await authFetch<StrictnessPreset[]>('/api/tenant/rules/strictness-presets')
      presets.value = data
      return data
    }
    catch (e: any) {
      error.value = e.message || '加载审核尺度预设失败'
      throw e
    }
    finally { loading.value = false }
  }

  async function updatePreset(strictness: string, body: { reasoning_instruction: string; extraction_instruction: string }): Promise<StrictnessPreset> {
    const data = await authFetch<StrictnessPreset>(`/api/tenant/rules/strictness-presets/${strictness}`, { method: 'PUT', body })
    return data
  }

  return {
    configs, rules, presets, loading, error,
    listConfigs, createConfig, updateConfig, deleteConfig,
    testConnection, fetchFields,
    listRules, createRule, updateRule, deleteRule,
    listPresets, updatePreset,
  }
}
