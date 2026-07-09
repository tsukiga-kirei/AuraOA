export interface ExternalWorkflowContextConfig {
  include_basic?: boolean
  basic_fields?: string[]
  data_mode?: 'none' | 'all_fields' | 'selected_fields'
  target_process_type?: string
  selected_fields?: string[]
  fallback_strategy?: 'basic_with_notice' | 'all_fields' | 'ignore'
  max_refs?: number
  max_rows?: number
}

export interface ExternalModelContextConfig {
  table_name?: string
  join_field?: string
  mode?: 'exists' | 'count' | 'rows' | 'custom_sql'
  return_fields?: string[]
  max_rows?: number
  order_by?: string
  order_dir?: 'ASC' | 'DESC'
  custom_sql?: string
}

export interface ExternalContextMount {
  type: 'workflow' | 'model'
  enabled: boolean
  name: string
  source_field: string
  source_splitter?: string
  workflow?: ExternalWorkflowContextConfig
  model?: ExternalModelContextConfig
}

export interface ExternalContextTestResponse {
  context_text: string
}
