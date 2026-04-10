// types/common.ts — 公共类型定义（原 rules.ts）

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

/** 系统提示词模板 */
export interface SystemPromptTemplate {
  id: string
  prompt_key: string
  prompt_type: 'system' | 'user'
  phase: 'reasoning' | 'extraction'
  strictness: string | null
  content: string
  description: string
  created_at?: string
  updated_at?: string
}

/** OA 流程基本信息（测试连接返回） */
export interface ProcessInfo {
  process_type: string
  process_name: string
  process_type_label?: string
  main_table: string
  detail_count: number
  table_mismatch?: boolean
  expected_table?: string
  type_label_mismatch?: boolean
  expected_type_label?: string
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
