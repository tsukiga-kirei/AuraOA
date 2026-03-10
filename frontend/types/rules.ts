// types/rules.ts — 规则配置相关类型

/** 流程字段（带选中状态，用于字段选择器） */
export interface ProcessField {
  field_key: string
  field_name: string
  field_type: string
  selected?: boolean
}

/** 明细表定义（带选中状态） */
export interface DetailTableDef {
  table_name: string
  table_label: string
  fields: ProcessField[]
}

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

/** 审核尺度预设 */
export interface StrictnessPreset {
  id: string
  tenant_id?: string
  strictness: string
  reasoning_instruction: string
  extraction_instruction: string
  created_at?: string
  updated_at?: string
}

/** OA 流程基本信息（测试连接返回） */
export interface ProcessInfo {
  process_type: string
  process_name: string
  main_table: string
  detail_count: number
}

/** 字段定义（OA 拉取的原始字段，无 selected） */
export interface FieldDef {
  field_key: string
  field_name: string
  field_type: string
}

/** OA 流程字段集合（拉取字段返回） */
export interface ProcessFields {
  main_fields: FieldDef[]
  detail_tables: DetailTableDef[]
}
