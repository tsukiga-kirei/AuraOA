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

export interface EmbedPersonalView {
  available: boolean
  user_id?: string
  username?: string
  display_name?: string
  has_audit?: boolean
  last_audit_at?: string
  running_job_id?: string
  audit_result?: AuditResult | null
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
  auto_retry_blocked?: boolean
  last_audit_at?: string
  running_job_id?: string
  audit_result?: AuditResult | null
  config_version_no?: number
  config_upgrade_available?: boolean
  personal_view?: EmbedPersonalView | null
  default_perspective?: 'personal' | 'standard'
}

export interface EmbedExecuteRequest {
  process_id: string
  process_type?: string
  title?: string
  trigger_source?: 'embed_auto' | 'embed_manual'
  trigger_detail?: 'visible_open' | 'manual'
  use_latest_config?: boolean
  perspective?: 'personal' | 'standard'
  oa_user_id?: string
}
