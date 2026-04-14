/**
 * useArchiveConfigApi — 归档复盘配置 API 调用封装
 * 对接后端路由组：
 *   /api/tenant/archive/configs    归档复盘流程配置 CRUD
 *   /api/tenant/archive/rules      归档规则 CRUD
 *   /api/tenant/archive/prompt-templates  归档专用提示词模板
 */

import type { ProcessArchiveConfig, ArchiveRule } from '~/types/archive-config'
import type { SystemPromptTemplate, ProcessInfo, ProcessFields } from '~/types/common'

export const useArchiveConfigApi = () => {
  const { authFetch } = useAuth()

  // ============================================================
  // 归档复盘配置
  // ============================================================

  /** 获取当前租户所有归档复盘流程配置列表 */
  async function listConfigs(): Promise<ProcessArchiveConfig[]> {
    return await authFetch<ProcessArchiveConfig[]>('/api/tenant/archive/configs')
  }

  /**
   * 创建新的归档复盘流程配置。
   * @param config 配置信息（流程类型、OA 连接、AI 模型等）
   */
  async function createConfig(config: Partial<ProcessArchiveConfig>): Promise<ProcessArchiveConfig> {
    return await authFetch<ProcessArchiveConfig>('/api/tenant/archive/configs', {
      method: 'POST',
      body: config,
    })
  }

  /**
   * 更新指定归档复盘流程配置。
   * @param id 配置 ID
   * @param config 要更新的字段
   */
  async function updateConfig(id: string, config: Partial<ProcessArchiveConfig>): Promise<ProcessArchiveConfig> {
    return await authFetch<ProcessArchiveConfig>(`/api/tenant/archive/configs/${id}`, {
      method: 'PUT',
      body: config,
    })
  }

  /**
   * 删除指定归档复盘流程配置（同时删除关联规则）。
   * @param id 配置 ID
   */
  async function deleteConfig(id: string): Promise<void> {
    await authFetch<null>(`/api/tenant/archive/configs/${id}`, { method: 'DELETE' })
  }

  /**
   * 测试 OA 数据库连接并获取流程基本信息。
   * @param processType 流程类型编码
   * @param mainTableName 主表名（可选）
   * @param processTypeLabel 流程类型显示名（可选）
   * @returns 流程基本信息（表结构、字段列表等）
   */
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

  /**
   * 拉取指定配置对应流程的字段列表（用于规则编辑器的字段选择）。
   * @param configId 归档配置 ID
   * @returns 流程字段定义列表
   */
  async function fetchFields(configId: string): Promise<ProcessFields> {
    return await authFetch<ProcessFields>(`/api/tenant/archive/configs/${configId}/fetch-fields`, {
      method: 'POST',
    })
  }

  // ============================================================
  // 归档规则
  // ============================================================

  /**
   * 获取指定配置下的归档规则列表，支持按规则范围和启用状态筛选。
   * @param configId 归档配置 ID
   * @param ruleScope 规则范围（可选，如 tenant / user）
   * @param enabled 是否只返回启用的规则（可选）
   */
  async function listRules(configId: string, ruleScope?: string, enabled?: boolean): Promise<ArchiveRule[]> {
    const params = new URLSearchParams({ config_id: configId })
    if (ruleScope) params.set('rule_scope', ruleScope)
    if (enabled !== undefined) params.set('enabled', String(enabled))
    return await authFetch<ArchiveRule[]>(`/api/tenant/archive/rules?${params.toString()}`)
  }

  /**
   * 创建新的归档规则。
   * @param rule 规则信息（规则类型、条件、权重等）
   */
  async function createRule(rule: Partial<ArchiveRule>): Promise<ArchiveRule> {
    return await authFetch<ArchiveRule>('/api/tenant/archive/rules', {
      method: 'POST',
      body: rule,
    })
  }

  /**
   * 更新指定归档规则。
   * @param id 规则 ID
   * @param rule 要更新的字段
   */
  async function updateRule(id: string, rule: Partial<ArchiveRule>): Promise<ArchiveRule> {
    return await authFetch<ArchiveRule>(`/api/tenant/archive/rules/${id}`, {
      method: 'PUT',
      body: rule,
    })
  }

  /**
   * 删除指定归档规则。
   * @param id 规则 ID
   */
  async function deleteRule(id: string): Promise<void> {
    await authFetch<null>(`/api/tenant/archive/rules/${id}`, { method: 'DELETE' })
  }

  // ============================================================
  // 归档专用系统提示词模板（archive_ 前缀）
  // ============================================================

  /** 获取归档复盘专用的 AI 系统提示词模板列表 */
  async function listPromptTemplates(): Promise<SystemPromptTemplate[]> {
    return await authFetch<SystemPromptTemplate[]>('/api/tenant/archive/prompt-templates')
  }

  return {
    listConfigs,
    createConfig,
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
