// types/archive-config.ts — 归档配置相关类型

import type { ProcessField, DetailTableDef } from '~/types/common'

/** 访问控制配置（归档复盘专用） */
export interface AccessControl {
  allowed_roles: string[]
  allowed_members: string[]
  allowed_departments: string[]
}

/** 归档复盘流程配置（参考 ProcessAuditConfig，增加 access_control） */
export interface ProcessArchiveConfig {
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
  access_control: AccessControl              // 访问控制权限
  status: string
  created_at?: string
  updated_at?: string
}

/** 归档规则 */
export interface ArchiveRule {
  id: string
  tenant_id?: string
  config_id?: string | null
  process_type: string
  rule_content: string
  rule_scope: 'mandatory' | 'default_on' | 'default_off'
  enabled: boolean
  source: string
  related_flow: boolean
  created_at?: string
  updated_at?: string
}
