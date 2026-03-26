export type ArchiveCompliance = 'compliant' | 'partially_compliant' | 'non_compliant'

export type ArchiveRunStatus =
  | 'pending'
  | 'assembling'
  | 'reasoning'
  | 'extracting'
  | 'completed'
  | 'failed'

export interface ArchiveFlowNodeResult {
  node_id: string
  node_name: string
  compliant: boolean
  reasoning: string
}

export interface ArchiveFieldAuditResult {
  field_key?: string
  field_name: string
  passed: boolean
  reasoning: string
}

export interface ArchiveRuleAuditResult {
  rule_id: string
  rule_name: string
  passed: boolean
  reasoning: string
}

export interface ArchiveProgressStep {
  key: string
  label: string
  done?: boolean
  current?: boolean
  failed?: boolean
}

export interface ArchiveProcessSnapshot {
  process_id?: string
  title?: string
  process_type?: string
  process_type_label?: string
  applicant?: string
  department?: string
  current_node?: string
  submit_time?: string
  archive_time?: string
  main_table_name?: string
  flow_snapshot?: {
    is_complete?: boolean
    missing_nodes?: string[]
    nodes?: Array<Record<string, unknown>>
    history_text?: string
    graph_text?: string
  }
}

export interface ArchiveReviewResult {
  id?: string
  trace_id: string
  process_id: string
  title?: string
  process_type?: string
  status?: ArchiveRunStatus
  overall_compliance?: ArchiveCompliance
  overall_score?: number
  confidence?: number
  duration_ms?: number
  ai_reasoning?: string
  ai_summary?: string
  flow_audit?: {
    is_complete: boolean
    missing_nodes: string[]
    node_results: ArchiveFlowNodeResult[]
  }
  field_audit?: ArchiveFieldAuditResult[]
  rule_audit?: ArchiveRuleAuditResult[]
  risk_points?: string[]
  suggestions?: string[]
  created_at?: string
  updated_at?: string
  error_message?: string
  parse_error?: string
  raw_content?: string
  process_snapshot?: ArchiveProcessSnapshot
  progress_steps?: ArchiveProgressStep[]
}

export interface ArchiveProcessItem {
  process_id: string
  title: string
  applicant: string
  department: string
  process_type: string
  process_type_label: string
  current_node: string
  submit_time: string
  archive_time: string
  has_review: boolean
  in_archive: boolean
  archive_status?: ArchiveRunStatus
  archive_result?: ArchiveReviewResult | null
}

export interface ArchiveProcessListResponse {
  items: ArchiveProcessItem[]
  total: number
}

export interface ArchiveReviewStats {
  total_count: number
  compliant_count: number
  partial_count: number
  non_compliant_count: number
  unaudited_count: number
  running_count: number
}

export interface ArchiveReviewExecuteRequest {
  process_id: string
  process_type: string
  title?: string
}

export interface ArchiveReviewSubmitResponse {
  status: ArchiveRunStatus
  id: string
  trace_id: string
  process_id: string
  created_at: string
}

export interface ArchiveBatchExecuteRequest {
  items: ArchiveReviewExecuteRequest[]
}

export interface ArchiveBatchExecuteResponse {
  results: ArchiveReviewSubmitResponse[]
  total: number
  accepted: number
  failed: number
}

export interface ArchiveProcessTypeOption {
  process_type: string
  process_type_label: string
  config_id: string
}

export interface ArchiveReviewHistoryItem {
  id: string
  process_id: string
  title: string
  process_type: string
  status: ArchiveRunStatus
  compliance: ArchiveCompliance
  compliance_score: number
  archive_result: ArchiveReviewResult
  process_snapshot?: ArchiveProcessSnapshot
  duration_ms: number
  ai_reasoning?: string
  confidence: number
  error_message?: string
  created_at: string
  updated_at: string
  user_name?: string
}
