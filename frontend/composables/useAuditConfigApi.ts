/**
 * useAuditConfigApi — 审核配置相关 API 调用封装
 * 对接后端路由组：
 *   /api/tenant/rules/configs          流程审核配置 CRUD
 *   /api/tenant/rules/audit-rules      审核规则 CRUD
 *   /api/tenant/rules/prompt-templates 系统提示词模板查询
 */

import type { ProcessAuditConfig, AuditRule } from '~/types/audit-config'
import type { SystemPromptTemplate, ProcessInfo, ProcessFields } from '~/types/common'

export type { ProcessAuditConfig, AuditRule, SystemPromptTemplate, ProcessInfo, ProcessFields }

export const useAuditConfigApi = () => {
  const { authFetch } = useAuth()

  // ============================================================
  // 流程审核配置
  // ============================================================

  /** 获取当前租户所有流程审核配置列表 */
  async function listConfigs(): Promise<ProcessAuditConfig[]> {
    return await authFetch<ProcessAuditConfig[]>('/api/tenant/rules/configs')
  }

  /**
   * 创建新的流程审核配置。
   * @param config 配置信息（流程类型、OA 连接、AI 模型等）
   */
  async function createConfig(config: Partial<ProcessAuditConfig>): Promise<ProcessAuditConfig> {
    return await authFetch<ProcessAuditConfig>('/api/tenant/rules/configs', { method: 'POST', body: config })
  }

  /**
   * 更新指定流程审核配置。
   * @param id 配置 ID
   * @param config 要更新的字段
   */
  async function updateConfig(id: string, config: Partial<ProcessAuditConfig>): Promise<ProcessAuditConfig> {
    return await authFetch<ProcessAuditConfig>(`/api/tenant/rules/configs/${id}`, { method: 'PUT', body: config })
  }

  /**
   * 删除指定流程审核配置（同时删除关联规则）。
   * @param id 配置 ID
   */
  async function deleteConfig(id: string): Promise<void> {
    await authFetch<null>(`/api/tenant/rules/configs/${id}`, { method: 'DELETE' })
  }

  /**
   * 测试 OA 数据库连接并获取流程基本信息。
   * @param processType 流程类型编码
   * @param mainTableName 主表名（可选）
   * @param processTypeLabel 流程类型显示名（可选）
   * @returns 流程基本信息（表结构、字段列表等）
   */
  async function testConnection(processType: string, mainTableName?: string, processTypeLabel?: string): Promise<ProcessInfo> {
    return await authFetch<ProcessInfo>('/api/tenant/rules/configs/test-connection', {
      method: 'POST',
      body: { process_type: processType, main_table_name: mainTableName || '', process_type_label: processTypeLabel || '' },
    })
  }

  /**
   * 拉取指定配置对应流程的字段列表（用于规则编辑器的字段选择）。
   * @param configId 审核配置 ID
   * @returns 流程字段定义列表
   */
  async function fetchFields(configId: string): Promise<ProcessFields> {
    return await authFetch<ProcessFields>(`/api/tenant/rules/configs/${configId}/fetch-fields`, { method: 'POST' })
  }

  // ============================================================
  // 审核规则
  // ============================================================

  /**
   * 获取指定配置下的审核规则列表，支持按规则范围和启用状态筛选。
   * @param configId 审核配置 ID
   * @param ruleScope 规则范围（可选，如 tenant / user）
   * @param enabled 是否只返回启用的规则（可选）
   */
  async function listRules(configId: string, ruleScope?: string, enabled?: boolean): Promise<AuditRule[]> {
    const params = new URLSearchParams({ config_id: configId })
    if (ruleScope) params.set('rule_scope', ruleScope)
    if (enabled !== undefined) params.set('enabled', String(enabled))
    return await authFetch<AuditRule[]>(`/api/tenant/rules/audit-rules?${params.toString()}`)
  }

  /**
   * 创建新的审核规则。
   * @param rule 规则信息（规则类型、条件、权重等）
   */
  async function createRule(rule: Partial<AuditRule>): Promise<AuditRule> {
    return await authFetch<AuditRule>('/api/tenant/rules/audit-rules', { method: 'POST', body: rule })
  }

  /**
   * 更新指定审核规则。
   * @param id 规则 ID
   * @param rule 要更新的字段
   */
  async function updateRule(id: string, rule: Partial<AuditRule>): Promise<AuditRule> {
    return await authFetch<AuditRule>(`/api/tenant/rules/audit-rules/${id}`, { method: 'PUT', body: rule })
  }

  /**
   * 删除指定审核规则。
   * @param id 规则 ID
   */
  async function deleteRule(id: string): Promise<void> {
    await authFetch<null>(`/api/tenant/rules/audit-rules/${id}`, { method: 'DELETE' })
  }

  // ============================================================
  // 系统提示词模板（审核专用）
  // ============================================================

  /** 获取审核专用的 AI 系统提示词模板列表 */
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
