// types/embed.ts — OA 嵌入审核页类型

import type { AuditResult } from '~/types/audit'

export interface EmbedProcessSummary {
  process_id: string
  title: string
  applicant: string
  department: string
  process_type: string
  process_type_label: string
  current_node: string
  submit_time: string
}

export interface EmbedContextResponse {
  supported: boolean
  reason?: 'not_found_in_oa' | 'no_config' | 'config_inactive' | 'embed_disabled'
  message?: string
  process?: EmbedProcessSummary
  embed_enabled?: boolean
  has_audit?: boolean
  stale?: boolean
  should_auto_audit?: boolean
  last_audit_at?: string
  running_job_id?: string
  audit_result?: AuditResult | null
}

export interface EmbedExecuteRequest {
  process_id: string
  process_type?: string
  title?: string
  trigger_source?: 'embed_auto' | 'embed_manual'
}
