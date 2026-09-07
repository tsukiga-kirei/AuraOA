import type { DetailTableDef, ProcessField } from '~/types/common'
import type { ExternalContextMount } from '~/types/external-context'

export interface SummaryBlockConfig {
  id: string
  title: string
  user_prompt: string
  enable_thinking?: boolean
  /** true：传入全部数据；false：仅传入 enabled_data_variables 中选中的数据 */
  include_meta: boolean
  enabled_data_variables?: string[]
  field_mode: 'all' | 'selected'
  selected_fields: string[]
  context_mounts?: ExternalContextMount[]
  enabled: boolean
  sort_order: number
}

export interface ProcessSummaryConfig {
  id: string
  process_id?: string
  enabled?: boolean
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
    auto_summary_on_data_change?: boolean
    auto_summary_on_return_resubmit?: boolean
    auto_summary_on_flow_change?: boolean
    scheduled_refresh_enabled?: boolean
    scheduled_refresh_lookback_days?: number
    scheduled_refresh_interval_minutes?: number
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
  deep_thinking?: string
  duration_ms?: number
}

export interface SummaryResult {
  result_source?: 'personal' | 'embed'
  trigger_source?: string
  user_id?: string
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
  config_version_no?: number
}

export interface SummaryWorkbenchProcessItem {
  process_id: string
  title: string
  applicant: string
  department: string
  process_type: string
  process_type_label: string
  current_node: string
  submit_time: string
  source: 'todo' | 'archived' | 'embed'
  has_summary: boolean
  summary_status: string
  summary_result?: SummaryResult | null
  summary_updated_at?: string
  running_job_id?: string
  visible_block_ids?: string[]
}

export interface SummaryWorkbenchListResponse {
  items: SummaryWorkbenchProcessItem[]
  total: number
  page: number
  page_size: number
}

export interface SummaryWorkbenchStats {
  total_count: number
  summarized_count: number
  pending_count: number
  running_count: number
  failed_count: number
}
