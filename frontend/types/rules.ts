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

// ============================================================
// 定时任务类型配置
// ============================================================

/** 定时任务内容模板 */
export interface CronContentTemplate {
  subject: string
  header: string
  body_template: string
  footer: string
  batch_limit?: number
}

/** 定时任务类型配置（合并预设+租户覆盖，后端返回） */
export interface CronTaskConfig {
  task_type: string                              // 任务类型编码（如 audit_batch / archive_daily）
  module: 'audit' | 'archive'                   // 所属模块
  label_zh: string                               // 中文显示名称
  label_en: string                               // 英文显示名称
  description_zh: string                         // 中文描述
  description_en: string                         // 英文描述
  default_cron: string                           // 预设默认 Cron 表达式（供参考）
  preset_push_format: string                     // 预设推送格式
  preset_content_template: CronContentTemplate   // 预设内容模板（用于"恢复默认"）
  sort_order: number
  // 租户当前状态
  is_enabled: boolean                            // 租户是否已启用该任务类型
  push_format: 'html' | 'markdown' | 'plain'    // 当前生效的推送格式
  content_template: CronContentTemplate          // 当前生效的内容模板
  batch_limit?: number                           // 当前批处理上限（null 表示使用默认）
}

/** 保存 Cron 任务类型配置请求体 */
export interface SaveCronTaskConfigRequest {
  push_format: string
  content_template: CronContentTemplate
  batch_limit?: number | null
}

// ============================================================
// 归档复盘配置
// ============================================================

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
