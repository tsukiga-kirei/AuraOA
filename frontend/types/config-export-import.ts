// types/config-export-import.ts — 流程配置导出导入类型契约

import type { SummaryBlockConfig } from '~/types/process-summary'

export type ConfigExportModule = 'audit' | 'archive' | 'summary'

/** 单条导出的规则数据结构（解耦数据库特定主键与租户ID） */
export interface ExportedRuleItem {
  rule_content: string
  rule_scope: 'mandatory' | 'default_on' | 'default_off'
  enabled?: boolean
  source?: string
  related_flow?: boolean
  context_enabled?: boolean
  context_mounts?: any[]
}

/** 导出的 AI 配置项 */
export interface ExportedAIConfig {
  audit_strictness?: string
  enable_thinking?: boolean
  system_reasoning_prompt?: string
  system_extraction_prompt?: string
  user_reasoning_prompt?: string
  user_extraction_prompt?: string
}

/** 导出内容载荷 */
export interface ConfigExportContents {
  rules?: ExportedRuleItem[]
  ai?: ExportedAIConfig
  summary_blocks?: SummaryBlockConfig[]
}

/** 标准导出包格式 Schema v1.0 */
export interface ConfigExportBundle {
  schema_version: string
  app: string
  module: ConfigExportModule
  exported_at: string
  source_meta: {
    process_type: string
    process_type_label?: string
    main_table_name?: string
  }
  contents: ConfigExportContents
}

/** 重复规则处理策略 */
export type RuleConflictStrategy = 'skip' | 'overwrite' | 'append'

/** 导入校验与预检报告 */
export interface ImportValidationReport {
  valid: boolean
  detected_module: ConfigExportModule | 'unknown'
  target_module: ConfigExportModule
  module_mismatch: boolean
  errors: string[]
  warnings: string[]
  has_rules: boolean
  has_ai: boolean
  has_summary_blocks: boolean
  rule_stats?: {
    total: number
    valid: number
    duplicate: number
    new_rules: number
    invalid: number
  }
  ai_stats?: {
    strictness?: string
    enable_thinking?: boolean
    unknown_variables: string[]
  }
  summary_stats?: {
    total_blocks: number
    block_titles: string[]
    unknown_variables: string[]
  }
  parsed_data?: ConfigExportContents
}
