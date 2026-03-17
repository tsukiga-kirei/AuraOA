// types/user-config.ts — 用户个人配置相关类型

/** 用户可见的流程列表项 */
export interface ProcessListItem {
  process_type: string
  process_type_label: string
  config_id: string
}

/** 用户自定义规则 */
export interface CustomRule {
  id: string
  content: string
  enabled: boolean
  related_flow: boolean
}

/** 规则开关覆盖 */
export interface RuleToggleOverride {
  rule_id: string
  enabled: boolean
}

/** 用户审核个性化配置项 */
export interface AuditDetailItem {
  config_id: string
  process_type: string
  field_config: {
    field_mode: string
    field_overrides: string[]
  }
  rule_config: {
    custom_rules: CustomRule[]
    rule_toggle_overrides: RuleToggleOverride[]
  }
  ai_config: {
    strictness_override: string
  }
}

/** 仪表板偏好 */
export interface DashboardPref {
  id?: string
  enabled_widgets: string[]
  widget_sizes: Record<string, any>
}

/** 用户权限控制标志 */
export interface UserPermissions {
  allow_custom_fields: boolean
  allow_custom_rules: boolean
  allow_modify_strictness: boolean
}

// ===================== 合并视图类型（工作台/归档完整配置） =====================

/** 租户字段配置项（含用户选中状态） */
export interface TenantField {
  field_key: string
  field_name: string
  field_type: string
  selected: boolean
}

/** 明细表配置（含字段选中状态） */
export interface DetailTable {
  table_name: string
  table_label: string
  fields: TenantField[]
}

/** 租户规则（含用户开关状态） */
export interface TenantRule {
  id: string
  rule_content: string
  rule_scope: string // mandatory | default_on | default_off
  related_flow: boolean
  enabled: boolean
}

/** 审核工作台完整配置响应（租户配置+用户覆盖合并） */
export interface FullAuditProcessConfig {
  process_type: string
  process_type_label: string
  config_id: string
  field_mode: string // 租户设置的字段传输模式
  kb_mode: string
  audit_strictness: string
  user_permissions: UserPermissions
  main_fields: TenantField[]
  detail_tables: DetailTable[]
  tenant_rules: TenantRule[]
  custom_rules: CustomRule[]
}

// ===================== Cron 偏好 =====================

/** 用户定时任务个人偏好 */
export interface CronPrefs {
  default_email: string
}

// ===================== 归档复盘用户端类型 =====================

/** 归档复盘用户权限配置 */
export interface ArchiveUserPermissions {
  allow_custom_fields: boolean
  allow_custom_rules: boolean
  allow_modify_strictness: boolean
}

/** 用户可访问的归档配置列表项 */
export interface AccessibleArchiveConfig {
  process_type: string
  process_type_label: string
  config_id: string
}

/** 归档复盘完整配置响应（租户配置+用户覆盖合并） */
export interface FullArchiveConfig {
  process_type: string
  process_type_label: string
  config_id: string
  field_mode: string
  kb_mode: string
  audit_strictness: string
  user_permissions: ArchiveUserPermissions
  main_fields: TenantField[]
  detail_tables: DetailTable[]
  tenant_rules: TenantRule[]
  custom_rules: CustomRule[]
}

/** 更新审核/归档个人配置请求体 */
export interface UpdatePersonalConfigRequest {
  config_id: string
  field_config: {
    field_mode: string
    field_overrides: string[]
  }
  rule_config: {
    custom_rules: CustomRule[]
    rule_toggle_overrides: RuleToggleOverride[]
  }
  ai_config: {
    strictness_override: string
  }
}

// ===================== 管理端用户配置视图类型 =====================

/** 管理员视图：用户自定义规则 */
export interface AdminCustomRule {
  id: string
  content: string
  enabled: boolean
}

/** 管理员视图：规则开关覆盖项 */
export interface AdminRuleToggleItem {
  rule_id: string
  rule_content: string
  rule_scope: string    // mandatory | default_on | default_off
  admin_enabled: boolean // 管理员当前设置的默认状态
  enabled: boolean       // 用户覆盖后的状态
}

/** 管理员视图：单个流程的用户个性化详情（审核工作台/归档复盘共用） */
export interface AdminProcessDetail {
  process_type: string
  strictness_override: string
  custom_rules: AdminCustomRule[]
  field_overrides: string[]
  rule_toggle_overrides: AdminRuleToggleItem[]
}

/** 管理员视图：用户定时任务偏好 */
export interface AdminCronDetail {
  default_email: string
  email_count: number
}

/** 管理员视图：单个用户的完整配置摘要（含成员信息） */
export interface AdminUserConfigItem {
  user_id: string
  member_id: string
  username: string
  display_name: string
  department: string
  role_names: string[]
  audit_process_count: number
  cron_email_count: number
  archive_process_count: number
  last_modified: string
  audit_details: AdminProcessDetail[]
  cron_details: AdminCronDetail
  archive_details: AdminProcessDetail[]
}
