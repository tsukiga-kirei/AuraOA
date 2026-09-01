// types/audit-config.ts — 审核配置相关类型

import type { ProcessField, DetailTableDef } from '~/types/common'
import type { AccessControl } from '~/types/archive-config'
import type { ExternalContextMount } from '~/types/external-context'

/** 流程审核配置 */
export interface ProcessAuditConfig {
  id: string
  process_id?: string
  tenant_id?: string
  process_type: string
  process_type_label: string
  main_table_name: string
  main_fields: ProcessField[]
  detail_tables: DetailTableDef[]
  field_mode: string
  kb_mode: string
  ai_config: Record<string, any>
  ai_strictness?: string
  ai_prompt_template?: string
  allow_user_custom_rules?: boolean
  allow_user_fields?: boolean
  allow_user_ai_strictness?: boolean
  enabled?: boolean
  user_permissions: Record<string, any>
  access_control?: AccessControl
  embed_enabled?: boolean
  embed_config?: {
    auto_audit_on_open?: boolean
    auto_audit_on_data_change?: boolean
    auto_audit_on_return_resubmit?: boolean
    auto_audit_on_flow_change?: boolean
    scheduled_refresh_enabled?: boolean
    scheduled_refresh_lookback_days?: number
    scheduled_refresh_interval_minutes?: number
  }
  status: string
  created_at?: string
  updated_at?: string
}

/** 审核规则 */
export interface AuditRule {
  id: string
  tenant_id?: string
  config_id?: string | null
  process_type: string
  rule_content: string
  rule_scope: 'mandatory' | 'default_on' | 'default_off'
  priority: number
  enabled: boolean
  source: string
  related_flow: boolean
  context_enabled?: boolean
  context_mounts?: ExternalContextMount[]
  created_at?: string
  updated_at?: string
}
