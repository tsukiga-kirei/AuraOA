export interface QueryParameter {
  name: string
  label: string
  type: 'string' | 'number' | 'date' | 'string_list'
  default_binding?: QueryBinding
}
export interface QueryDefinition {
  name: string
  description: string
  sql: string
  dialect: 'mysql' | 'oracle' | 'dm'
  tables: string[]
  parameters: QueryParameter[]
  max_rows: number
  builder?: QueryBuilder
}
export interface AuditDataQuery {
  id: string
  tenant_id: string
  name: string
  category: string
  draft: QueryDefinition
  latest_version: number
  archived: boolean
  tested_hash: string
  tested_at?: string
  updated_at: string
}
export interface QueryVersion {
  id: string
  query_id: string
  version_no: number
  definition: QueryDefinition
  created_at: string
}
export interface QueryBinding {
  source: 'field' | 'system' | 'constant'
  field?: string
  value?: string
  days_offset?: number
}
export interface QueryReference {
  query_id: string
  version_no: number
  enabled: boolean
  bindings: Record<string, QueryBinding>
}
export interface QueryCheck {
  valid: boolean
  issues: { code: string; message: string; position: number }[]
  parameters: string[]
  columns: string[]
}
export interface QueryColumn {
  key: string
  label: string
  data_type: string
}
export interface QueryEvidence {
  rule_id?: string
  rule_content?: string
  query_id?: string
  version_no: number
  name: string
  description: string
  parameters: Record<string, unknown>
  queried_at: string
  status: string
  error?: string
  result?: {
    columns: QueryColumn[]
    rows: Record<string, unknown>[]
    truncated: boolean
    duration_ms: number
  }
  model_text: string
}
export interface QueryPrompts {
  generate: string
  explain: string
  repair: string
}
export interface QueryAssist {
  sql: string
  explanation: string
  parameters: QueryParameter[]
  warnings: string[]
  check: QueryCheck
}

/** 可视化配置只负责生成 SQL，实际执行和安全校验统一以 SQL 为准。 */
export interface QueryBuilder {
  /** 手工 SQL 优先使用；保留此前配置以便切回编辑，不反向解析 SQL。 */
  sql_mode?: boolean
  source_type: 'table' | 'workflow'
  /** 旧流程表配置未设置时沿用仅归档，新配置由独立开关决定。 */
  archived_only?: boolean
  table: string
  columns: {
    field: string
    alias: string
    aggregate: '' | 'SUM' | 'COUNT' | 'MIN' | 'MAX' | 'AVG'
  }[]
  filters: {
    field: string
    operator: '=' | '<>' | '>' | '>=' | '<' | '<=' | 'LIKE' | 'IN' | 'IS NULL' | 'IS NOT NULL'
    parameter: string
    binding?: QueryBinding
    label: string
    type: QueryParameter['type']
  }[]
  order_alias: string
  order_dir: 'ASC' | 'DESC'
}
