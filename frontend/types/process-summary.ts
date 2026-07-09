import type { DetailTableDef, ProcessField } from '~/types/common'

export interface SummaryBlockConfig {
  id: string
  title: string
  user_prompt: string
  /** true：传入全部数据；false：仅传入 enabled_data_variables 中选中的数据 */
  include_meta: boolean
  enabled_data_variables?: string[]
  field_mode: 'all' | 'selected'
  selected_fields: string[]
  enabled: boolean
  sort_order: number
}

export interface ProcessSummaryConfig {
  id: string
  tenant_id?: string
  process_type: string
  process_type_label: string
  main_table_name: string
  main_fields: ProcessField[]
  detail_tables: DetailTableDef[]
  summary_blocks: SummaryBlockConfig[]
  embed_enabled?: boolean
  embed_config?: {
    auto_summary_on_open?: boolean
    auto_summary_on_stale?: boolean
  }
  status: string
  created_at?: string
  updated_at?: string
}

export interface SummaryBlockResult {
  block_id: string
  title: string
  content: string
  points: string[]
  duration_ms?: number
}

export interface SummaryResult {
  status?: string
  id?: string
  trace_id?: string
  process_id?: string
  blocks?: SummaryBlockResult[]
  duration_ms?: number
  created_at?: string
  parse_error?: string
  raw_content?: string
  error_message?: string
}
