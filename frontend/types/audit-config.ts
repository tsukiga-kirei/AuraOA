// types/audit-config.ts — 审核配置相关类型

import type { ProcessField, DetailTableDef } from '~/types/common'
import type { AccessControl } from '~/types/archive-config'

/** 流程审核配置 */
export interface ProcessAuditConfig {
  id: string
  tenant_id?: string
  process_type: string
  process_type_label: string
  main_table_name: string
  main_fields: ProcessField[]
  detail_tables: DetailTableDef[]
  field_mode: string
  kb_mode: string
  ai_config: Record<string, any>
  user_permissions: Record<string, any>
  access_control?: AccessControl
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
  created_at?: string
  updated_at?: string
}
