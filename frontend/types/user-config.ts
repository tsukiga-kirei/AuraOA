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
}

/** 规则开关覆盖 */
export interface RuleToggleOverride {
  rule_id: string
  enabled: boolean
}

/** 用户审核个性化配置项 */
export interface AuditDetailItem {
  process_type: string
  custom_rules: CustomRule[]
  field_overrides: string[]
  field_mode: string
  strictness_override: string
  rule_toggle_overrides: RuleToggleOverride[]
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
