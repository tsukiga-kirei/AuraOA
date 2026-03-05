/**
 * 用于开发的模拟数据 - 模拟 API 响应
 * 所有模拟/虚拟数据都存放在这里。业务代码只引用该文件。*/
import type {PermissionGroup} from "~/types/auth";


// ============================================================
//业务模拟数据
// ============================================================
export interface OAProcess {
  process_id: string
  title: string
  applicant: string
  department: string
  submit_time: string
  process_type: string
  status: string
  current_node: string  //当前审批节点（替换金额显示）
  amount?: number       //已弃用，保留用于向后兼容
  urgency?: 'high' | 'medium' | 'low'  //已弃用
  oa_url?: string
}

export interface ChecklistResult {
  rule_id: string
  rule_name: string
  passed: boolean
  reasoning: string
  is_locked?: boolean
  related_flow?: boolean
}



/** 旧版 AuditResult - 保留用于向后兼容*/
export interface AuditResult {
  trace_id: string
  process_id: string
  recommendation: 'approve' | 'return' | 'review'
  score: number
  details: ChecklistResult[]
  ai_reasoning: string
  duration_ms: number
  //新字段 (v2)
  action_label?: string
  confidence?: number
  risk_points?: string[]
  suggestions?: string[]
  ai_summary?: string
  model_used?: string
  interaction_mode?: 'two_phase' | 'single_pass'
  phase1_duration_ms?: number
  phase2_duration_ms?: number
}

export interface CronTask {
  id: string
  cron_expression: string
  task_type: string
  is_active: boolean
  last_run_at: string | null
  next_run_at: string
  created_at: string
  success_count: number
  fail_count: number
  is_builtin?: boolean
  push_email?: string
}

// ============================================================
//Cron 任务配置类型 (定时任务配置 - 机场管理)
// ============================================================
export interface CronTaskTypeConfig {
  task_type: 'batch_audit' | 'daily_report' | 'weekly_report'
  label: string
  enabled: boolean
  /** 仅batch_audit：每次执行的最大待处理项目数*/
  batch_limit?: number
  push_format: 'html' | 'markdown' | 'plain'
  content_template: {
    subject: string
    header: string
    body_template: string
    footer: string
  }
}

export interface AuditSnapshot {
  snapshot_id: string
  process_id: string
  title: string
  applicant: string
  department: string
  recommendation: string
  score: number
  created_at: string
  adopted: boolean | null
}

export interface TenantJdbcConfig {
  driver: 'mysql' | 'postgresql' | 'oracle' | 'sqlserver'
  host: string
  port: number
  database: string
  username: string
  password: string
  pool_size: number
  connection_timeout: number  //秒
  test_on_borrow: boolean
}

export interface TenantAIConfig {
  default_provider: string
  default_model: string
  fallback_provider: string
  fallback_model: string
  max_tokens_per_request: number
  temperature: number
  timeout_seconds: number
  retry_count: number
}

export interface TenantInfo {
  id: string
  name: string
  code: string                //租户识别码
  oa_type: string
  oa_db_connection_id: string //参考系统级OA数据库连接
  token_quota: number
  token_used: number
  max_concurrency: number
  status: 'active' | 'inactive'
  created_at: string
  contact_name: string
  contact_email: string
  contact_phone: string
  description: string
  ai_config: TenantAIConfig
  log_retention_days: number  //日志保留多少天
  data_retention_days: number //审计数据保留多少天
  allow_custom_model: boolean //租户用户是否可以覆盖AI模型
  sso_enabled: boolean
  sso_endpoint: string
  tenant_admin_id?: string    //引用租户管理员的 MOCK_USERS 用户名
}

// ============================================================
//系统设置类型（系统设置）
// ============================================================
export interface OASystemConfig {
  id: string
  name: string
  type: 'weaver_e9' | 'weaver_ebridge' | 'zhiyuan_a8' | 'landray_ekp' | 'custom'
  type_label: string
  version: string
  status: 'connected' | 'disconnected' | 'testing'
  description: string
  adapter_version: string
  last_sync: string
  sync_interval: number  //秒
  enabled: boolean
}

/** OA数据库连接-系统级，跨租户共享*/
export interface OADatabaseConnection {
  id: string
  name: string                //用户定义的显示名称
  oa_type: 'weaver_e9' | 'weaver_ebridge' | 'zhiyuan_a8' | 'landray_ekp' | 'custom'
  oa_type_label: string
  jdbc_config: TenantJdbcConfig
  status: 'connected' | 'disconnected' | 'testing'
  last_sync: string
  sync_interval: number
  enabled: boolean
  created_at: string
  description: string
}

export interface AIModelConfig {
  id: string
  provider: string
  model_name: string
  display_name: string
  type: 'local' | 'cloud'
  endpoint: string
  api_key_configured: boolean
  max_tokens: number
  context_window: number
  cost_per_1k_tokens: number  //成本人民币
  status: 'online' | 'offline' | 'maintenance'
  enabled: boolean
  description: string
  capabilities: string[]  //例如['文本'、'代码'、'推理']
}

export interface SystemGeneralConfig {
  platform_name: string
  platform_version: string
  default_language: string
  session_timeout: number  //分钟
  max_upload_size: number  //MB
  enable_audit_trail: boolean
  enable_data_encryption: boolean
  backup_enabled: boolean
  backup_cron: string
  backup_retention_days: number
  notification_email: string
  smtp_host: string
  smtp_port: number
  smtp_username: string
  smtp_ssl: boolean
}


export interface AuditRule {
  id: string
  process_type: string
  rule_content: string
  rule_scope: 'mandatory' | 'default_on' | 'default_off'
  priority: number
  enabled: boolean
  related_flow?: boolean
}

export interface FlowNode {
  node_id: string
  node_name: string
  approver: string
  action: 'approve' | 'return'
  action_time: string
  opinion: string
}

export interface ArchivedProcess {
  process_id: string
  title: string
  applicant: string
  department: string
  process_type: string
  amount?: number
  submit_time: string
  archive_time: string
  status: 'archived'
  flow_nodes: FlowNode[]
  fields: Record<string, string>
}

export interface FlowNodeAuditResult {
  node_id: string
  node_name: string
  compliant: boolean
  reasoning: string
}

export interface ArchiveAuditResult {
  trace_id: string
  process_id: string
  overall_compliance: 'compliant' | 'non_compliant' | 'partially_compliant'
  overall_score: number
  duration_ms: number
  flow_audit: {
    is_complete: boolean
    missing_nodes: string[]
    node_results: FlowNodeAuditResult[]
  }
  field_audit: { field_name: string; passed: boolean; reasoning: string }[]
  rule_audit: { rule_id: string; rule_name: string; passed: boolean; reasoning: string }[]
  ai_summary: string
}

export interface DashboardStats {
  todayAudits: number
  todayApproved: number
  todayReturned: number
  pendingCount: number
  avgResponseMs: number
  successRate: number
  weeklyTrend: { date: string; count: number }[]
}

// ============================================================
//概述 仪表板类型（仪表盘）
// ============================================================
export type OverviewWidgetId =
  | 'audit_summary'       //业务：今日审计统计数据
  | 'pending_tasks'       //业务：待处理任务数
  | 'weekly_trend'        //业务：每周审计趋势图
  | 'dept_distribution'   //tenant_admin：按部门审计分布
  | 'recent_activity'     //全部：最近的活动提要
  | 'ai_performance'      //tenant_admin：AI模型性能
  | 'tenant_usage'        //tenant_admin：租户资源使用情况
  | 'user_activity'       //tenant_admin：用户活跃度排名
  | 'system_health'       //system_admin：系统健康状况概述
  | 'tenant_overview'     //system_admin：所有租户概览
  | 'api_metrics'         //system_admin：API 调用指标
  | 'monitor_metrics'     //system_admin：关键运营指标（来自全局监视器）
  | 'monitor_alerts'      //system_admin：最近的警报（来自全局监视器）
  | 'cron_tasks'          //业务：计划任务
  | 'archive_review'      //业务：档案审查

export interface OverviewWidget {
  id: OverviewWidgetId
  title: string
  description: string
  /** 哪些权限组可以看到这个小部件*/
  requiredPermissions: PermissionGroup[]
  /** 默认启用状态*/
  defaultEnabled: boolean
  /** 小部件尺寸：'sm' = 1/3，'md' = 1/2，'lg' = 全宽*/
  size: 'sm' | 'md' | 'lg'
}

export const OVERVIEW_WIDGETS: OverviewWidget[] = [
  { id: 'audit_summary', title: '审核概览', description: '审核通过/退回/已归档数量统计', requiredPermissions: ['business'], defaultEnabled: true, size: 'lg' },
  { id: 'pending_tasks', title: '待办任务', description: '当前待处理的审核流程数量', requiredPermissions: ['business'], defaultEnabled: true, size: 'sm' },
  { id: 'weekly_trend', title: '审核趋势', description: '个人的使用智能审核进行审批的流程数', requiredPermissions: ['business'], defaultEnabled: true, size: 'md' },
  { id: 'cron_tasks', title: '定时任务', description: '定时任务执行情况概览', requiredPermissions: ['business'], defaultEnabled: true, size: 'md' },
  { id: 'archive_review', title: '归档复盘', description: '归档流程合规复核情况', requiredPermissions: ['business'], defaultEnabled: true, size: 'md' },
  { id: 'dept_distribution', title: '部门分布使用情况', description: '各部门审核流程数量与使用分布', requiredPermissions: ['tenant_admin'], defaultEnabled: true, size: 'md' },
  { id: 'recent_activity', title: '最近动态', description: '最近的审核操作与系统事件', requiredPermissions: ['business', 'tenant_admin', 'system_admin'], defaultEnabled: true, size: 'md' },
  { id: 'ai_performance', title: 'AI 模型表现', description: 'AI 审核响应时间与准确率', requiredPermissions: ['tenant_admin'], defaultEnabled: true, size: 'md' },
  { id: 'tenant_usage', title: '租户资源用量', description: 'Token 消耗、存储用量等', requiredPermissions: ['tenant_admin'], defaultEnabled: true, size: 'md' },
  { id: 'user_activity', title: '用户活跃排行', description: '租户内用户审核活跃度排名', requiredPermissions: ['tenant_admin'], defaultEnabled: true, size: 'md' },
  { id: 'system_health', title: '系统健康', description: '各服务运行状态与资源占用', requiredPermissions: ['system_admin'], defaultEnabled: true, size: 'lg' },
  { id: 'tenant_overview', title: '租户总览', description: '所有租户的使用情况汇总', requiredPermissions: ['system_admin'], defaultEnabled: true, size: 'md' },
  { id: 'api_metrics', title: 'API 调用指标', description: 'API 调用量、成功率、延迟分布', requiredPermissions: ['system_admin'], defaultEnabled: true, size: 'md' },
  { id: 'monitor_metrics', title: '运行指标', description: '系统关键运行指标概览（API 成功率、模型响应、延迟等）', requiredPermissions: ['system_admin'], defaultEnabled: true, size: 'lg' },
  { id: 'monitor_alerts', title: '最近告警', description: '系统告警与异常事件', requiredPermissions: ['system_admin'], defaultEnabled: true, size: 'md' },
]

export interface OverviewDashboardData {
  auditSummary: { approved: number; returned: number; archived: number; total: number }
  pendingCount: number
  weeklyTrend: { date: string; count: number }[]
  deptDistribution: { department: string; count: number; color: string }[]
  recentActivity: { id: string; action: string; target: string; user: string; time: string; type: 'audit' | 'cron' | 'system' | 'config' }[]
  aiPerformance: { avgResponseMs: number; successRate: number; totalCalls: number; dailyStats: { date: string; avgMs: number; calls: number }[] }
  tenantUsage: { tokenUsed: number; tokenQuota: number; storageUsedMB: number; storageQuotaMB: number; activeUsers: number; totalUsers: number }
  userActivity: { username: string; displayName: string; department: string; auditCount: number; lastActive: string }[]
  systemHealth: { service: string; status: 'healthy' | 'degraded' | 'down'; cpu: number; memory: number; uptime: string }[]
  tenantOverview: { tenantId: string; tenantName: string; userCount: number; auditCount: number; tokenUsed: number; status: 'active' | 'suspended' }[]
  apiMetrics: { endpoint: string; calls: number; avgMs: number; successRate: number }[]
  monitorMetrics: { apiSuccessRate: number; avgModelResponseMs: number; p95Latency: number; totalRequests24h: number; activeTenants: number; uptime: string }
  monitorAlerts: { id: number; level: string; messageZh: string; messageEn: string; timeZh: string; timeEn: string }[]
}

/** 用户的仪表板小部件首选项（按用户存储）*/
export interface UserDashboardPrefs {
  /** 用户已启用的小组件 ID（布局的顺序很重要）*/
  enabledWidgets: OverviewWidgetId[]
  /** 小部件可选的自定义尺寸*/
  widgetSizes?: Partial<Record<OverviewWidgetId, 'sm' | 'md' | 'lg'>>
}

// ============================================================
//归档复核类型 (归档复盘 - 全流程合规复核)
// ============================================================
//FlowNode、ArchiveProcess、FlowNodeAuditResult、ArchiveAuditResult
//在上面的业务模拟数据部分中定义。

export interface FieldAuditResult {
  field_name: string
  passed: boolean
  reasoning: string
}

// ============================================================
//以流程为中心的审核配置类型（审核工作台配置）
// ============================================================
export interface ProcessField {
  field_key: string
  field_name: string
  field_type: 'text' | 'number' | 'date' | 'select' | 'textarea' | 'file'
  selected: boolean
}

export interface ProcessAIConfig {
  audit_strictness: 'strict' | 'standard' | 'loose'
  reasoning_prompt: string
  extraction_prompt: string
  //保留旧字段以向后兼容其他模块
  ai_provider?: string
  model_name?: string
  system_prompt?: string
  context_window?: number
  temperature?: number
}

//严格提示预设 - 每个租户可配置，从 API 获取
export interface StrictnessPromptPreset {
  strictness: 'strict' | 'standard' | 'loose'
  reasoning_instruction: string
  extraction_instruction: string
}

export interface UserPermissions {
  allow_custom_fields: boolean
  allow_custom_rules: boolean
  allow_modify_strictness: boolean
}

// ============================================================
//存档审核配置类型 (归档复盘配置)
// ============================================================
export interface FlowRuleConfig {
  id: string
  rule_content: string
  rule_scope: 'mandatory' | 'default_on' | 'default_off'
  priority: number
  enabled: boolean
  source: 'manual' | 'file_import'
}

export interface ArchiveReviewConfig {
  id: string
  process_type: string
  /** 流程类型标签，如"采购类"、"费用类"，用于分类展示 */
  process_type_label?: string
  main_table_name?: string
  main_fields?: ProcessField[]
  detail_tables?: DetailTable[]
  fields: ProcessField[]
  field_mode: 'all' | 'selected'
  rules: (AuditRule & { source: 'manual' | 'file_import' })[]
  flow_rules: FlowRuleConfig[]
  kb_mode: 'rules_only' | 'rag_only' | 'hybrid'
  ai_config: ProcessAIConfig
  user_permissions: {
    allow_custom_fields: boolean
    allow_custom_rules: boolean
    allow_custom_flow_rules: boolean
    allow_modify_strictness: boolean
  }
  /** 允许访问此存档过程的角色/成员/部门*/
  allowed_roles: string[]
  /** 明确允许的成员 ID（除了角色之外）*/
  allowed_members: string[]
  /** 明确允许的部门 ID*/
  allowed_departments?: string[]
}

export interface ArchiveUserPermissions {
  allow_custom_fields: boolean
  allow_custom_rules: boolean
  allow_modify_strictness: boolean
}

export interface DetailTable {
  table_name: string
  table_label: string
  fields: ProcessField[]
}

export interface ProcessAuditConfig {
  id: string
  process_type: string
  /** 流程类型标签，如"采购类"、"费用类"，用于分类展示 */
  process_type_label?: string
  main_table_name?: string
  main_fields?: ProcessField[]
  detail_tables?: DetailTable[]
  fields: ProcessField[]
  field_mode: 'all' | 'selected'
  rules: (AuditRule & { source: 'manual' | 'file_import'; related_flow?: boolean })[]
  kb_mode: 'rules_only' | 'rag_only' | 'hybrid'
  ai_config: ProcessAIConfig
  user_permissions: UserPermissions
}

// ============================================================
//组织和人员类型（组织人员）
// ============================================================
//组织类型 - 从 ~/types/org 导入并重新导出以实现向后兼容性
export type { Department, OrgRole, OrgMember } from '~/types/org'

//组织模拟数据已删除 - 现在由 Go 后端通过 useOrgApi 提供服务

// ============================================================
//用户个人配置类型（用户偏好分析 - 机场管理）
// ============================================================

/** 审核工作台 - 单个流程的用户自定义配置 */
export interface UserAuditProcessDetail {
  process_type: string
  custom_rules: { id: string; content: string; enabled: boolean }[]
  field_overrides: string[]  //用户切换的字段名称
  strictness_override: 'strict' | 'standard' | 'loose' | null  //null = 不覆盖
  rule_toggle_overrides: { rule_id: string; rule_content: string; enabled: boolean }[]
}

/** 归档复盘 - 单个流程的用户自定义配置 */
export interface UserArchiveProcessDetail {
  process_type: string
  custom_rules: { id: string; content: string; enabled: boolean }[]
  custom_flow_rules: { id: string; content: string; enabled: boolean }[]
  field_overrides: string[]
  strictness_override: 'strict' | 'standard' | 'loose' | null
}

/** 定时任务 - 用户自定义/修改的定时任务记录 */
export interface UserCronConfigDetail {
  task_id: string
  task_type: string
  task_label: string
  cron_expression: string
  /** 'modified' = 修改了系统默认任务, 'custom' = 用户添加的自定义任务*/
  source: 'modified' | 'custom'
  is_active: boolean
  push_email?: string
}

export interface UserPersonalConfig {
  id: string
  user_id: string
  username: string
  display_name: string
  department: string
  /** 角色名称列表（从组织人员获取） */
  role_names: string[]
  /** 审核工作台：按流程统计的修改数 */
  audit_process_count: number
  /** 定时任务：用户自定义/修改的定时任务数 */
  cron_config_count: number
  /** 归档复盘：按流程统计的修改数 */
  archive_process_count: number
  /** 最后修改时间 */
  last_modified: string
  /** 审核工作台详细配置（按流程） */
  audit_details: UserAuditProcessDetail[]
  /** 定时任务详细配置 */
  cron_config_details: UserCronConfigDetail[]
  /** 归档复盘详细配置（按流程） */
  archive_details: UserArchiveProcessDetail[]
}

export const mockUserPersonalConfigs: UserPersonalConfig[] = [
  {
    id: 'UPC-001', user_id: 'M-001', username: 'zhangming', display_name: '张明', department: '研发部',
    role_names: ['业务用户', '审计管理员'],
    audit_process_count: 1, cron_config_count: 1, archive_process_count: 1,
    last_modified: '2025-06-10 14:30',
    audit_details: [
      {
        process_type: '采购审批',
        custom_rules: [{ id: 'UCR-001', content: '供应商必须在合格名录中', enabled: true }],
        field_overrides: [],
        strictness_override: 'strict',
        rule_toggle_overrides: [],
      },
    ],
    cron_config_details: [
      { task_id: 'CT-USER-ZM-001', task_type: 'batch_audit', task_label: '批量审核', cron_expression: '0 8 * * 1-5', source: 'modified', is_active: true },
    ],
    archive_details: [
      {
        process_type: '采购审批',
        custom_rules: [{ id: 'UACR-001', content: '付款条件须与公司标准一致', enabled: true }],
        custom_flow_rules: [],
        field_overrides: [],
        strictness_override: null,
      },
    ],
  },
  {
    id: 'UPC-002', user_id: 'M-002', username: 'lifang', display_name: '李芳', department: '销售部',
    role_names: ['业务用户'],
    audit_process_count: 1, cron_config_count: 2, archive_process_count: 0,
    last_modified: '2025-06-09 16:20',
    audit_details: [
      {
        process_type: '费用报销',
        custom_rules: [],
        field_overrides: ['出差日期', '发票附件'],
        strictness_override: null,
        rule_toggle_overrides: [{ rule_id: 'R006', rule_content: '差旅住宿标准不超过城市限额', enabled: true }],
      },
    ],
    cron_config_details: [
      { task_id: 'CT-USER-LF-001', task_type: 'daily_report', task_label: '日报推送', cron_expression: '0 18 * * 1-5', source: 'modified', is_active: true, push_email: 'lifang-personal@example.com' },
      { task_id: 'CT-USER-LF-002', task_type: 'weekly_report', task_label: '销售部周报', cron_expression: '0 9 * * 1', source: 'custom', is_active: true, push_email: 'lifang@example.com' },
    ],
    archive_details: [],
  },
  {
    id: 'UPC-003', user_id: 'M-003', username: 'wangqiang', display_name: '王强', department: 'IT部',
    role_names: ['业务用户', '审计管理员'],
    audit_process_count: 2, cron_config_count: 2, archive_process_count: 1,
    last_modified: '2025-06-10 09:45',
    audit_details: [
      {
        process_type: '采购审批',
        custom_rules: [
          { id: 'UCR-W01', content: 'IT设备采购须附技术评估报告', enabled: true },
          { id: 'UCR-W02', content: '服务器采购须经IT架构评审', enabled: true },
        ],
        field_overrides: ['合同编号'],
        strictness_override: 'strict',
        rule_toggle_overrides: [],
      },
      {
        process_type: '合同审批',
        custom_rules: [{ id: 'UCR-W03', content: 'SLA条款须明确响应时间', enabled: true }],
        field_overrides: [],
        strictness_override: 'strict',
        rule_toggle_overrides: [],
      },
    ],
    cron_config_details: [
      { task_id: 'CT-BUILTIN-001', task_type: 'batch_audit', task_label: '批量审核', cron_expression: '0 9 * * 1-5', source: 'modified', is_active: true },
      { task_id: 'CT-USER-WQ-001', task_type: 'batch_audit', task_label: 'IT部专项批量审核', cron_expression: '0 14 * * 3', source: 'custom', is_active: true },
    ],
    archive_details: [
      {
        process_type: '采购审批',
        custom_rules: [
          { id: 'UACR-W01', content: '供应商交付记录须完整', enabled: true },
          { id: 'UACR-W02', content: '验收报告须附测试数据', enabled: true },
        ],
        custom_flow_rules: [{ id: 'UAFR-W01', content: 'IT部门须参与验收节点', enabled: true }],
        field_overrides: [],
        strictness_override: null,
      },
    ],
  },
  {
    id: 'UPC-004', user_id: 'M-004', username: 'zhaoli', display_name: '赵丽', department: '人力资源部',
    role_names: ['业务用户'],
    audit_process_count: 0, cron_config_count: 1, archive_process_count: 1,
    last_modified: '2025-06-08 11:00',
    audit_details: [],
    cron_config_details: [
      { task_id: 'CT-USER-ZL-001', task_type: 'daily_report', task_label: '日报推送', cron_expression: '0 17 * * 1-5', source: 'modified', is_active: true, push_email: 'zhaoli-hr@example.com' },
    ],
    archive_details: [
      {
        process_type: '人事审批',
        custom_rules: [],
        custom_flow_rules: [{ id: 'UAFR-Z01', content: '入职审批须在招聘计划审批之后', enabled: true }],
        field_overrides: [],
        strictness_override: null,
      },
    ],
  },
  {
    id: 'UPC-005', user_id: 'M-005', username: 'chenwei', display_name: '陈伟', department: '市场部',
    role_names: ['业务用户'],
    audit_process_count: 2, cron_config_count: 1, archive_process_count: 0,
    last_modified: '2025-06-10 17:10',
    audit_details: [
      {
        process_type: '采购审批',
        custom_rules: [{ id: 'UCR-C01', content: '市场推广物料采购须附活动方案', enabled: true }],
        field_overrides: [],
        strictness_override: 'loose',
        rule_toggle_overrides: [],
      },
      {
        process_type: '费用报销',
        custom_rules: [{ id: 'UCR-C02', content: '活动费用须附参会人员名单', enabled: true }],
        field_overrides: [],
        strictness_override: null,
        rule_toggle_overrides: [],
      },
    ],
    cron_config_details: [
      { task_id: 'CT-USER-CW-001', task_type: 'daily_report', task_label: '市场部日报', cron_expression: '0 19 * * 1-5', source: 'custom', is_active: true, push_email: 'chenwei@example.com' },
    ],
    archive_details: [],
  },
  {
    id: 'UPC-006', user_id: 'M-007', username: 'zhanghua', display_name: '张华', department: '财务部',
    role_names: ['业务用户', '审计管理员'],
    audit_process_count: 2, cron_config_count: 1, archive_process_count: 1,
    last_modified: '2025-06-09 10:30',
    audit_details: [
      {
        process_type: '费用报销',
        custom_rules: [{ id: 'UCR-ZH01', content: '大额报销须附审批截图', enabled: true }],
        field_overrides: ['出差日期', '发票附件'],
        strictness_override: null,
        rule_toggle_overrides: [{ rule_id: 'R006', rule_content: '差旅住宿标准不超过城市限额', enabled: true }],
      },
      {
        process_type: '采购审批',
        custom_rules: [],
        field_overrides: ['交付日期'],
        strictness_override: null,
        rule_toggle_overrides: [],
      },
    ],
    cron_config_details: [
      { task_id: 'CT-USER-ZH-001', task_type: 'batch_audit', task_label: '批量审核', cron_expression: '0 9 * * 1-5', source: 'modified', is_active: true, push_email: 'zhanghua-finance@example.com' },
    ],
    archive_details: [
      {
        process_type: '费用报销',
        custom_rules: [{ id: 'UACR-ZH01', content: '发票金额须与报销金额一致', enabled: true }],
        custom_flow_rules: [],
        field_overrides: [],
        strictness_override: null,
      },
    ],
  },
  {
    id: 'UPC-007', user_id: 'M-009', username: 'zhoulei', display_name: '周磊', department: '销售部',
    role_names: ['业务用户'],
    audit_process_count: 0, cron_config_count: 0, archive_process_count: 0,
    last_modified: '',
    audit_details: [],
    cron_config_details: [],
    archive_details: [],
  },
  {
    id: 'UPC-008', user_id: 'M-006', username: 'liuyang', display_name: '刘洋', department: '行政部',
    role_names: ['业务用户'],
    audit_process_count: 1, cron_config_count: 2, archive_process_count: 0,
    last_modified: '2025-06-07 15:40',
    audit_details: [
      {
        process_type: '采购审批',
        custom_rules: [],
        field_overrides: ['附件材料'],
        strictness_override: 'loose',
        rule_toggle_overrides: [],
      },
    ],
    cron_config_details: [
      { task_id: 'CT-BUILTIN-001', task_type: 'batch_audit', task_label: '批量审核', cron_expression: '0 9 * * 1-5', source: 'modified', is_active: false },
      { task_id: 'CT-USER-LY-001', task_type: 'weekly_report', task_label: '行政部周报', cron_expression: '0 10 * * 1', source: 'custom', is_active: true },
    ],
    archive_details: [],
  },
]

// ============================================================
//数据管理类型（数据信息）
// ============================================================
export interface AuditLog {
  id: string
  process_id: string
  title: string
  operator: string
  department: string
  process_type: string
  /** AI审核推荐*/
  recommendation: 'approve' | 'return' | 'review'
  score: number
  created_at: string
  /** 完整的人工智能审核结果可详细查看*/
  audit_result: AuditResult
}

export interface CronLog {
  id: string
  task_id: string
  task_type: string
  task_label: string
  operator: string
  department: string
  status: 'success' | 'failed' | 'running'
  started_at: string
  finished_at: string | null
  message: string
}

export interface ArchiveLog {
  id: string
  process_id: string
  title: string
  operator: string
  department: string
  process_type: string
  /** AI合规结果*/
  compliance: 'compliant' | 'non_compliant' | 'partially_compliant'
  compliance_score: number
  created_at: string
  /** 完整存档审核结果以供详细查看*/
  archive_result: ArchiveAuditResult
}

export const mockAuditLogs: AuditLog[] = [
  {
    id: 'AL-001', process_id: 'WF-2025-001', title: '办公设备采购申请', operator: '张明', department: '行政部', process_type: '采购审批', recommendation: 'return', score: 72, created_at: '2025-06-10 09:35',
    audit_result: { trace_id: 'TR-20250610-A3F8', process_id: 'WF-2025-001', recommendation: 'return', score: 72, duration_ms: 3850, details: [{ rule_id: 'R001', rule_name: '预算额度校验', passed: true, reasoning: '采购金额 ¥156,000 未超过部门季度预算上限 ¥200,000', is_locked: true }, { rule_id: 'R003', rule_name: '供应商资质校验', passed: false, reasoning: '供应商未在合格供应商名录中', is_locked: true }, { rule_id: 'R004', rule_name: '采购比价要求', passed: false, reasoning: '单笔采购超过 ¥100,000 需提供至少 3 家供应商报价' }], ai_reasoning: '该采购申请存在供应商资质和比价流程问题，建议退回修改。', action_label: '建议退回', confidence: 0.85, risk_points: ['供应商未在合格名录中', '缺少竞争性比价材料'], suggestions: ['补充供应商资质证明', '提供至少3家供应商报价'], ai_summary: '该采购申请存在两个关键问题需要修正。' }
  },
  {
    id: 'AL-002', process_id: 'WF-2025-002', title: '差旅费报销', operator: '李芳', department: '市场部', process_type: '费用报销', recommendation: 'approve', score: 88, created_at: '2025-06-10 10:20',
    audit_result: { trace_id: 'TR-20250610-B2D4', process_id: 'WF-2025-002', recommendation: 'approve', score: 88, duration_ms: 1280, details: [{ rule_id: 'R006', rule_name: '差旅标准校验', passed: true, reasoning: '差旅费用在公司标准范围内', is_locked: true }, { rule_id: 'R007', rule_name: '发票合规性', passed: true, reasoning: '发票信息完整，日期与行程匹配' }], ai_reasoning: '差旅费报销合规，材料齐全。建议通过。', action_label: '建议通过', confidence: 0.92, risk_points: [], suggestions: ['建议后续出差提前提交预算申请'], ai_summary: '差旅费报销合规，材料齐全。' }
  },
  {
    id: 'AL-003', process_id: 'WF-2025-003', title: '年度服务器租赁合同续签', operator: '王强', department: 'IT部', process_type: '合同审批', recommendation: 'return', score: 45, created_at: '2025-06-10 11:10',
    audit_result: { trace_id: 'TR-20250610-C3E5', process_id: 'WF-2025-003', recommendation: 'return', score: 45, duration_ms: 2100, details: [{ rule_id: 'R001', rule_name: '预算额度校验', passed: true, reasoning: '合同金额在年度IT预算范围内' }, { rule_id: 'R004', rule_name: '合同条款完整性', passed: false, reasoning: 'SLA条款缺少故障响应时间约定', is_locked: true }], ai_reasoning: '合同续签存在SLA条款不完整和价格涨幅较大的问题。', action_label: '建议退回', confidence: 0.78, risk_points: ['SLA条款缺少故障响应时间', '合同金额较上年增长15%'], suggestions: ['补充SLA故障响应时间条款'], ai_summary: '合同续签需关注SLA和价格问题。' }
  },
  {
    id: 'AL-004', process_id: 'WF-2025-004', title: '新员工入职审批', operator: '赵丽', department: '人力资源部', process_type: '人事审批', recommendation: 'approve', score: 91, created_at: '2025-06-10 14:30',
    audit_result: { trace_id: 'TR-20250610-D4F6', process_id: 'WF-2025-004', recommendation: 'approve', score: 91, duration_ms: 1050, details: [{ rule_id: 'R011', rule_name: '入职材料完整性', passed: true, reasoning: '入职材料齐全，身份证明、学历证明均已提供' }, { rule_id: 'R012', rule_name: '审批层级校验', passed: true, reasoning: '审批链完整' }], ai_reasoning: '新员工入职审批完全合规，材料齐全。建议通过。', action_label: '建议通过', confidence: 0.95, risk_points: [], suggestions: ['建议定期复核'], ai_summary: '入职审批完全合规。' }
  },
  {
    id: 'AL-005', process_id: 'WF-2025-005', title: '市场推广活动预算申请', operator: '陈伟', department: '市场部', process_type: '采购审批', recommendation: 'review', score: 65, created_at: '2025-06-10 16:00',
    audit_result: { trace_id: 'TR-20250610-E5G7', process_id: 'WF-2025-005', recommendation: 'review', score: 65, duration_ms: 1800, details: [{ rule_id: 'R001', rule_name: '预算额度校验', passed: true, reasoning: '预算金额在市场部年度预算范围内', is_locked: true }, { rule_id: 'R013', rule_name: '活动方案完整性', passed: false, reasoning: '推广方案缺少预期ROI分析' }], ai_reasoning: '市场推广预算申请部分合规，缺少ROI分析。建议复核。', action_label: '建议复核', confidence: 0.72, risk_points: ['缺少预期ROI分析'], suggestions: ['补充预期ROI分析报告'], ai_summary: '推广预算申请需补充ROI分析。' }
  },
  {
    id: 'AL-006', process_id: 'WF-2025-006', title: '办公室装修工程审批', operator: '张华', department: '行政部', process_type: '工程审批', recommendation: 'approve', score: 85, created_at: '2025-06-09 15:20',
    audit_result: { trace_id: 'TR-20250609-F6H8', process_id: 'WF-2025-006', recommendation: 'approve', score: 85, duration_ms: 2200, details: [{ rule_id: 'R001', rule_name: '预算额度校验', passed: true, reasoning: '工程预算在年度行政预算范围内' }, { rule_id: 'R014', rule_name: '工程资质校验', passed: true, reasoning: '施工方具备相应资质' }], ai_reasoning: '办公室装修工程审批合规，施工方资质齐全。建议通过。', action_label: '建议通过', confidence: 0.88, risk_points: [], suggestions: ['建议施工期间安排专人监督'], ai_summary: '装修工程审批合规。' }
  },
  {
    id: 'AL-007', process_id: 'WF-2025-007', title: '客户招待费报销', operator: '王强', department: '销售部', process_type: '费用报销', recommendation: 'return', score: 52, created_at: '2025-06-09 11:45',
    audit_result: { trace_id: 'TR-20250609-G7I9', process_id: 'WF-2025-007', recommendation: 'return', score: 52, duration_ms: 1500, details: [{ rule_id: 'R006', rule_name: '费用标准校验', passed: false, reasoning: '招待费用超出公司标准上限' }, { rule_id: 'R007', rule_name: '发票合规性', passed: false, reasoning: '部分发票日期与招待记录不匹配' }], ai_reasoning: '客户招待费报销存在费用超标和发票不匹配问题。建议退回。', action_label: '建议退回', confidence: 0.82, risk_points: ['费用超出标准上限', '发票日期不匹配'], suggestions: ['核实招待费用明细', '补充正确日期的发票'], ai_summary: '招待费报销存在多项问题。' }
  },
  {
    id: 'AL-008', process_id: 'WF-2025-008', title: '年度培训计划审批', operator: '李芳', department: '人力资源部', process_type: '人事审批', recommendation: 'approve', score: 93, created_at: '2025-06-08 09:30',
    audit_result: { trace_id: 'TR-20250608-H8J0', process_id: 'WF-2025-008', recommendation: 'approve', score: 93, duration_ms: 1100, details: [{ rule_id: 'R015', rule_name: '培训预算校验', passed: true, reasoning: '培训预算在年度人力资源预算范围内' }, { rule_id: 'R016', rule_name: '培训方案完整性', passed: true, reasoning: '培训计划包含目标、内容、时间安排等完整信息' }], ai_reasoning: '年度培训计划审批完全合规。建议通过。', action_label: '建议通过', confidence: 0.94, risk_points: [], suggestions: ['建议培训结束后收集反馈'], ai_summary: '培训计划审批完全合规。' }
  },
]

export const mockCronLogs: CronLog[] = [
  { id: 'CL-001', task_id: 'CT-BUILTIN-001', task_type: 'batch_audit', task_label: '批量审核', operator: '张明', department: '研发部', status: 'success', started_at: '2025-06-10 09:00', finished_at: '2025-06-10 09:05', message: '2025-06-10 09:00 批量审核任务执行成功，共审核 12 条流程' },
  { id: 'CL-002', task_id: 'CT-002', task_type: 'daily_report', task_label: '日报推送', operator: '李芳', department: '销售部', status: 'success', started_at: '2025-06-09 18:00', finished_at: '2025-06-09 18:02', message: '2025-06-09 18:00 日报推送任务执行成功' },
  { id: 'CL-003', task_id: 'CT-003', task_type: 'weekly_report', task_label: '周报推送', operator: '王强', department: 'IT部', status: 'success', started_at: '2025-06-09 10:00', finished_at: '2025-06-09 10:08', message: '2025-06-09 10:00 周报推送任务执行成功，已推送至 15 人' },
  { id: 'CL-004', task_id: 'CT-BUILTIN-001', task_type: 'batch_audit', task_label: '批量审核', operator: '赵丽', department: '人力资源部', status: 'failed', started_at: '2025-06-08 09:00', finished_at: '2025-06-08 09:01', message: 'AI 服务连接超时，请检查 AI 服务状态' },
  { id: 'CL-005', task_id: 'CT-002', task_type: 'daily_report', task_label: '日报推送', operator: '陈伟', department: '市场部', status: 'success', started_at: '2025-06-08 18:00', finished_at: '2025-06-08 18:03', message: '2025-06-08 18:00 日报推送任务执行成功' },
  { id: 'CL-006', task_id: 'CT-004', task_type: 'batch_audit', task_label: '批量审核', operator: '张华', department: '财务部', status: 'success', started_at: '2025-06-08 02:00', finished_at: '2025-06-08 02:10', message: '2025-06-08 02:00 批量审核任务执行成功，共审核 8 条流程' },
  { id: 'CL-007', task_id: 'CT-002', task_type: 'daily_report', task_label: '日报推送', operator: '王强', department: 'IT部', status: 'failed', started_at: '2025-06-07 18:00', finished_at: '2025-06-07 18:01', message: 'SMTP 邮件服务器连接被拒绝' },
]

export const mockArchiveLogs: ArchiveLog[] = [
  {
    id: 'ARL-001', process_id: 'WF-2025-050', title: '2025年度服务器集群采购', operator: '张华', department: 'IT部', process_type: '采购审批', compliance: 'compliant', compliance_score: 92, created_at: '2025-06-10 10:30',
    archive_result: { trace_id: 'ATR-20250610-001', process_id: 'WF-2025-050', overall_compliance: 'compliant', overall_score: 92, duration_ms: 2500, flow_audit: { is_complete: true, missing_nodes: [], node_results: [{ node_id: 'N1', node_name: '部门经理审批', compliant: true, reasoning: '审批节点完整' }] }, field_audit: [], rule_audit: [{ rule_id: 'R001', rule_name: '预算额度校验', passed: true, reasoning: '采购金额在预算范围内' }, { rule_id: 'R003', rule_name: '供应商资质校验', passed: true, reasoning: '供应商资质齐全' }], ai_summary: '该采购流程整体合规，审批链完整，规则校验全部通过。' }
  },
  {
    id: 'ARL-002', process_id: 'WF-2025-038', title: '华东区域市场推广费用报销', operator: '陈伟', department: '市场部', process_type: '费用报销', compliance: 'partially_compliant', compliance_score: 78, created_at: '2025-06-10 09:15',
    archive_result: { trace_id: 'ATR-20250610-002', process_id: 'WF-2025-038', overall_compliance: 'partially_compliant', overall_score: 78, duration_ms: 2100, flow_audit: { is_complete: true, missing_nodes: [], node_results: [{ node_id: 'N1', node_name: '部门经理审批', compliant: true, reasoning: '审批节点完整' }] }, field_audit: [], rule_audit: [{ rule_id: 'R006', rule_name: '费用标准校验', passed: true, reasoning: '费用在标准范围内' }, { rule_id: 'R007', rule_name: '发票合规性', passed: false, reasoning: '部分发票缺少明细' }], ai_summary: '该费用报销流程存在部分合规问题，发票明细不完整。' }
  },
  {
    id: 'ARL-003', process_id: 'WF-2025-025', title: '外包开发合同签署', operator: '张华', department: 'IT部', process_type: '合同审批', compliance: 'non_compliant', compliance_score: 45, created_at: '2025-06-09 15:00',
    archive_result: { trace_id: 'ATR-20250609-003', process_id: 'WF-2025-025', overall_compliance: 'non_compliant', overall_score: 45, duration_ms: 3200, flow_audit: { is_complete: false, missing_nodes: ['法务审批'], node_results: [{ node_id: 'N1', node_name: '部门经理审批', compliant: true, reasoning: '审批节点完整' }, { node_id: 'N2', node_name: '法务审批', compliant: false, reasoning: '缺少法务审批节点' }] }, field_audit: [], rule_audit: [{ rule_id: 'R004', rule_name: '合同条款完整性', passed: false, reasoning: '合同缺少违约责任条款' }, { rule_id: 'R017', rule_name: '法务审核要求', passed: false, reasoning: '外包合同需经法务审核' }], ai_summary: '该合同签署流程存在较多合规问题，缺少法务审批和违约责任条款。' }
  },
  {
    id: 'ARL-004', process_id: 'WF-2025-012', title: '新员工批量入职审批', operator: '赵丽', department: '人力资源部', process_type: '人事审批', compliance: 'compliant', compliance_score: 95, created_at: '2025-06-09 11:00',
    archive_result: { trace_id: 'ATR-20250609-004', process_id: 'WF-2025-012', overall_compliance: 'compliant', overall_score: 95, duration_ms: 1800, flow_audit: { is_complete: true, missing_nodes: [], node_results: [{ node_id: 'N1', node_name: '部门经理审批', compliant: true, reasoning: '审批节点完整' }] }, field_audit: [], rule_audit: [{ rule_id: 'R011', rule_name: '入职材料完整性', passed: true, reasoning: '入职材料齐全' }], ai_summary: '新员工入职审批完全合规。' }
  },
  {
    id: 'ARL-005', process_id: 'WF-2025-060', title: '年度办公用品集中采购', operator: '王强', department: '行政部', process_type: '采购审批', compliance: 'compliant', compliance_score: 88, created_at: '2025-06-08 16:30',
    archive_result: { trace_id: 'ATR-20250608-005', process_id: 'WF-2025-060', overall_compliance: 'compliant', overall_score: 88, duration_ms: 2000, flow_audit: { is_complete: true, missing_nodes: [], node_results: [{ node_id: 'N1', node_name: '部门经理审批', compliant: true, reasoning: '审批节点完整' }] }, field_audit: [], rule_audit: [{ rule_id: 'R001', rule_name: '预算额度校验', passed: true, reasoning: '采购金额在预算范围内' }], ai_summary: '办公用品采购审批合规。' }
  },
  {
    id: 'ARL-006', process_id: 'WF-2025-055', title: '销售部差旅费季度报销', operator: '李芳', department: '销售部', process_type: '费用报销', compliance: 'partially_compliant', compliance_score: 72, created_at: '2025-06-08 10:00',
    archive_result: { trace_id: 'ATR-20250608-006', process_id: 'WF-2025-055', overall_compliance: 'partially_compliant', overall_score: 72, duration_ms: 2300, flow_audit: { is_complete: true, missing_nodes: [], node_results: [{ node_id: 'N1', node_name: '部门经理审批', compliant: true, reasoning: '审批节点完整' }] }, field_audit: [], rule_audit: [{ rule_id: 'R006', rule_name: '费用标准校验', passed: false, reasoning: '部分差旅费用超出标准' }, { rule_id: 'R007', rule_name: '发票合规性', passed: true, reasoning: '发票信息完整' }], ai_summary: '差旅费报销存在部分费用超标问题。' }
  },
]

export const mockProcessAuditConfigs: ProcessAuditConfig[] = [
  {
    id: 'PAC-001',
    process_type: '采购审批',
    process_type_label: '采购类',
    main_table_name: 'formtable_main_001',
    main_fields: [
      { field_key: 'amount', field_name: '采购金额', field_type: 'number', selected: true },
      { field_key: 'supplier', field_name: '供应商名称', field_type: 'text', selected: true },
      { field_key: 'category', field_name: '采购类别', field_type: 'select', selected: true },
      { field_key: 'reason', field_name: '采购事由', field_type: 'textarea', selected: true },
      { field_key: 'delivery_date', field_name: '交付日期', field_type: 'date', selected: false },
      { field_key: 'contract_no', field_name: '合同编号', field_type: 'text', selected: false },
      { field_key: 'attachment', field_name: '附件材料', field_type: 'file', selected: false },
    ],
    detail_tables: [
      {
        table_name: 'formtable_main_001_dt1',
        table_label: '采购明细',
        fields: [
          { field_key: 'item_name', field_name: '物品名称', field_type: 'text', selected: true },
          { field_key: 'item_qty', field_name: '数量', field_type: 'number', selected: true },
          { field_key: 'unit_price', field_name: '单价', field_type: 'number', selected: true },
          { field_key: 'item_spec', field_name: '规格型号', field_type: 'text', selected: false },
        ],
      },
    ],
    field_mode: 'selected',
    fields: [
      { field_key: 'amount', field_name: '采购金额', field_type: 'number', selected: true },
      { field_key: 'supplier', field_name: '供应商名称', field_type: 'text', selected: true },
      { field_key: 'category', field_name: '采购类别', field_type: 'select', selected: true },
      { field_key: 'reason', field_name: '采购事由', field_type: 'textarea', selected: true },
      { field_key: 'delivery_date', field_name: '交付日期', field_type: 'date', selected: false },
      { field_key: 'contract_no', field_name: '合同编号', field_type: 'text', selected: false },
      { field_key: 'attachment', field_name: '附件材料', field_type: 'file', selected: false },
    ],
    rules: [
      { id: 'R001', process_type: '采购审批', rule_content: '单笔采购金额不得超过部门季度预算上限', rule_scope: 'mandatory', priority: 100, enabled: true, source: 'manual', related_flow: false },
      { id: 'R002', process_type: '采购审批', rule_content: '超过10万元需提供至少3家供应商比价', rule_scope: 'mandatory', priority: 95, enabled: true, source: 'manual', related_flow: false },
      { id: 'R013', process_type: '采购审批', rule_content: '供应商须在合格供应商名录中', rule_scope: 'default_on', priority: 85, enabled: true, source: 'file_import', related_flow: false },
      { id: 'R014', process_type: '采购审批', rule_content: '合同条款须包含付款条件、交付时间、售后条款', rule_scope: 'default_on', priority: 80, enabled: true, source: 'manual', related_flow: false },
      { id: 'R019', process_type: '采购审批', rule_content: '金额超过50万需总经理审批节点', rule_scope: 'mandatory', priority: 90, enabled: true, source: 'manual', related_flow: true },
    ],
    kb_mode: 'rules_only',
    ai_config: {
      audit_strictness: 'standard',
      reasoning_prompt: '你是一个专业的采购审核助手。请根据以下规则对采购申请进行合规性审核，逐条检查并给出判断理由。对于不合规项，请明确指出问题并给出修改建议。\n\n主表数据：{{main_table}}\n明细表数据：{{detail_tables}}\n审核规则：{{rules}}\n审批流历史：{{flow_history}}\n流程图：{{flow_graph}}\n当前节点：{{current_node}}',
      extraction_prompt: '请根据以上推理分析结果，严格按照 JSON Schema 输出结构化审核结论。\n\n需要输出：\n1. recommendation：建议操作（approve/return/review）及置信度\n2. rule_checks：逐条规则校验结果（rule_id、是否通过、判断理由）\n3. risk_points：发现的风险点列表\n4. suggestions：改进建议列表\n\n原始规则列表：{{rules}}',
    },
    user_permissions: {
      allow_custom_fields: false,
      allow_custom_rules: true,
      allow_modify_strictness: true,
    },
  },
  {
    id: 'PAC-002',
    process_type: '费用报销',
    process_type_label: '费用类',
    main_table_name: 'formtable_main_002',
    main_fields: [
      { field_key: 'amount', field_name: '报销金额', field_type: 'number', selected: true },
      { field_key: 'expense_type', field_name: '费用类型', field_type: 'select', selected: true },
      { field_key: 'invoice_count', field_name: '发票数量', field_type: 'number', selected: true },
      { field_key: 'reason', field_name: '报销事由', field_type: 'textarea', selected: true },
      { field_key: 'trip_date', field_name: '出差日期', field_type: 'date', selected: false },
    ],
    detail_tables: [
      {
        table_name: 'formtable_main_002_dt1',
        table_label: '发票明细',
        fields: [
          { field_key: 'invoice_no', field_name: '发票号码', field_type: 'text', selected: true },
          { field_key: 'invoice_amount', field_name: '发票金额', field_type: 'number', selected: true },
          { field_key: 'invoice_date', field_name: '发票日期', field_type: 'date', selected: true },
          { field_key: 'invoice_file', field_name: '发票附件', field_type: 'file', selected: false },
        ],
      },
    ],
    field_mode: 'selected',
    fields: [
      { field_key: 'amount', field_name: '报销金额', field_type: 'number', selected: true },
      { field_key: 'expense_type', field_name: '费用类型', field_type: 'select', selected: true },
      { field_key: 'invoice_count', field_name: '发票数量', field_type: 'number', selected: true },
      { field_key: 'reason', field_name: '报销事由', field_type: 'textarea', selected: true },
      { field_key: 'trip_date', field_name: '出差日期', field_type: 'date', selected: false },
      { field_key: 'invoice_file', field_name: '发票附件', field_type: 'file', selected: false },
    ],
    rules: [
      { id: 'R003', process_type: '费用报销', rule_content: '单次报销金额超过5000元需附发票原件', rule_scope: 'default_on', priority: 80, enabled: true, source: 'manual', related_flow: false },
      { id: 'R006', process_type: '费用报销', rule_content: '差旅住宿标准不超过城市限额', rule_scope: 'default_off', priority: 60, enabled: false, source: 'manual', related_flow: false },
      { id: 'R015', process_type: '费用报销', rule_content: '发票日期须在报销申请日期前90天内', rule_scope: 'mandatory', priority: 90, enabled: true, source: 'file_import', related_flow: false },
    ],
    kb_mode: 'rules_only',
    ai_config: {
      audit_strictness: 'standard',
      reasoning_prompt: '你是一个专业的费用报销审核助手。请根据以下规则对报销申请进行合规性审核，重点关注金额合理性、发票合规性和审批材料完整性。\n\n主表数据：{{main_table}}\n明细表数据：{{detail_tables}}\n审核规则：{{rules}}\n流程图：{{flow_graph}}',
      extraction_prompt: '请根据以上推理分析结果，严格按照 JSON Schema 输出结构化审核结论。\n\n需要输出：\n1. recommendation：建议操作及置信度\n2. rule_checks：逐条规则校验结果\n3. risk_points：风险点\n4. suggestions：改进建议\n\n原始规则列表：{{rules}}',
    },
    user_permissions: {
      allow_custom_fields: true,
      allow_custom_rules: true,
      allow_modify_strictness: false,
    },
  },
  {
    id: 'PAC-003',
    process_type: '合同审批',
    process_type_label: '合同类',
    main_table_name: 'formtable_main_003',
    main_fields: [
      { field_key: 'contract_amount', field_name: '合同金额', field_type: 'number', selected: true },
      { field_key: 'vendor', field_name: '合作方', field_type: 'text', selected: true },
      { field_key: 'contract_period', field_name: '合同期限', field_type: 'text', selected: true },
      { field_key: 'contract_type', field_name: '合同类型', field_type: 'select', selected: true },
      { field_key: 'deliverables', field_name: '交付物', field_type: 'textarea', selected: true },
      { field_key: 'contract_file', field_name: '合同文件', field_type: 'file', selected: true },
    ],
    field_mode: 'all',
    fields: [
      { field_key: 'contract_amount', field_name: '合同金额', field_type: 'number', selected: true },
      { field_key: 'vendor', field_name: '合作方', field_type: 'text', selected: true },
      { field_key: 'contract_period', field_name: '合同期限', field_type: 'text', selected: true },
      { field_key: 'contract_type', field_name: '合同类型', field_type: 'select', selected: true },
      { field_key: 'deliverables', field_name: '交付物', field_type: 'textarea', selected: true },
      { field_key: 'contract_file', field_name: '合同文件', field_type: 'file', selected: true },
    ],
    rules: [
      { id: 'R004', process_type: '合同审批', rule_content: '合同金额超过50万需法务部会签', rule_scope: 'mandatory', priority: 100, enabled: true, source: 'manual', related_flow: true },
      { id: 'R016', process_type: '合同审批', rule_content: '合同须包含知识产权归属条款', rule_scope: 'default_on', priority: 85, enabled: true, source: 'manual', related_flow: false },
      { id: 'R017', process_type: '合同审批', rule_content: '合作方须通过准入评审', rule_scope: 'mandatory', priority: 95, enabled: true, source: 'file_import', related_flow: false },
    ],
    kb_mode: 'rules_only',
    ai_config: {
      audit_strictness: 'strict',
      reasoning_prompt: '你是一个专业的合同审核助手。请根据以下规则对合同进行全面审核，重点关注法律条款完整性、金额合理性和合作方资质。对于高风险条款请特别标注。\n\n主表数据：{{main_table}}\n审核规则：{{rules}}\n审批流历史：{{flow_history}}\n流程图：{{flow_graph}}',
      extraction_prompt: '请根据以上推理分析结果，严格按照 JSON Schema 输出结构化审核结论。\n\n需要输出：\n1. recommendation：建议操作（approve/return/review）及置信度\n2. rule_checks：逐条规则校验结果\n3. risk_points：风险点\n4. suggestions：改进建议\n\n原始规则列表：{{rules}}',
    },
    user_permissions: {
      allow_custom_fields: false,
      allow_custom_rules: false,
      allow_modify_strictness: false,
    },
  },
  {
    id: 'PAC-004',
    process_type: '人事审批',
    process_type_label: '人事类',
    main_table_name: 'formtable_main_004',
    main_fields: [
      { field_key: 'position', field_name: '岗位名称', field_type: 'text', selected: true },
      { field_key: 'headcount', field_name: '招聘人数', field_type: 'number', selected: true },
      { field_key: 'department', field_name: '用人部门', field_type: 'text', selected: true },
      { field_key: 'onboard_date', field_name: '入职日期', field_type: 'date', selected: true },
      { field_key: 'salary_range', field_name: '薪资范围', field_type: 'text', selected: false },
      { field_key: 'job_desc', field_name: '岗位职责', field_type: 'textarea', selected: false },
    ],
    field_mode: 'selected',
    fields: [
      { field_key: 'position', field_name: '岗位名称', field_type: 'text', selected: true },
      { field_key: 'headcount', field_name: '招聘人数', field_type: 'number', selected: true },
      { field_key: 'department', field_name: '用人部门', field_type: 'text', selected: true },
      { field_key: 'onboard_date', field_name: '入职日期', field_type: 'date', selected: true },
      { field_key: 'salary_range', field_name: '薪资范围', field_type: 'text', selected: false },
      { field_key: 'job_desc', field_name: '岗位职责', field_type: 'textarea', selected: false },
    ],
    rules: [
      { id: 'R005', process_type: '人事审批', rule_content: '新增HC需部门负责人和HR总监双签', rule_scope: 'default_on', priority: 75, enabled: true, source: 'manual', related_flow: true },
      { id: 'R018', process_type: '人事审批', rule_content: '招聘人数须在年度HC计划范围内', rule_scope: 'mandatory', priority: 90, enabled: true, source: 'manual', related_flow: false },
    ],
    kb_mode: 'rules_only',
    ai_config: {
      audit_strictness: 'loose',
      reasoning_prompt: '你是一个专业的人事审批审核助手。请根据以下规则对人事申请进行审核，关注HC计划匹配度、审批链完整性和岗位合理性。\n\n主表数据：{{main_table}}\n审核规则：{{rules}}\n流程图：{{flow_graph}}\n当前节点：{{current_node}}',
      extraction_prompt: '请根据以上推理分析结果，严格按照 JSON Schema 输出结构化审核结论。\n\n需要输出：\n1. recommendation：建议操作及置信度\n2. rule_checks：逐条规则校验结果\n3. risk_points：风险点\n4. suggestions：改进建议\n\n原始规则列表：{{rules}}',
    },
    user_permissions: {
      allow_custom_fields: true,
      allow_custom_rules: true,
      allow_modify_strictness: true,
    },
  },
  {
    id: 'PAC-005',
    process_type: '工程审批',
    process_type_label: '工程类',
    main_table_name: 'formtable_main_005',
    main_fields: [
      { field_key: 'project_name', field_name: '工程名称', field_type: 'text', selected: true },
      { field_key: 'budget', field_name: '工程预算', field_type: 'number', selected: true },
      { field_key: 'contractor', field_name: '施工方', field_type: 'text', selected: true },
      { field_key: 'start_date', field_name: '开工日期', field_type: 'date', selected: true },
    ],
    field_mode: 'all',
    fields: [
      { field_key: 'project_name', field_name: '工程名称', field_type: 'text', selected: true },
      { field_key: 'budget', field_name: '工程预算', field_type: 'number', selected: true },
      { field_key: 'contractor', field_name: '施工方', field_type: 'text', selected: true },
      { field_key: 'start_date', field_name: '开工日期', field_type: 'date', selected: true },
    ],
    rules: [
      { id: 'R020', process_type: '工程审批', rule_content: '工程预算须在年度行政预算范围内', rule_scope: 'mandatory', priority: 90, enabled: true, source: 'manual', related_flow: false },
      { id: 'R021', process_type: '工程审批', rule_content: '施工方须具备相应资质', rule_scope: 'mandatory', priority: 95, enabled: true, source: 'manual', related_flow: false },
    ],
    kb_mode: 'rules_only',
    ai_config: {
      audit_strictness: 'standard',
      reasoning_prompt: '你是一个工程审批审核助手。请根据规则对工程项目进行审核。\n\n主表数据：{{main_table}}\n审核规则：{{rules}}\n流程图：{{flow_graph}}',
      extraction_prompt: '请输出结构化审核结论。\n\n原始规则列表：{{rules}}',
    },
    user_permissions: {
      allow_custom_fields: true,
      allow_custom_rules: true,
      allow_modify_strictness: false,
    },
  },
  {
    id: 'PAC-006',
    process_type: '项目审批',
    process_type_label: '项目类',
    main_table_name: 'formtable_main_006',
    main_fields: [
      { field_key: 'project_name', field_name: '项目名称', field_type: 'text', selected: true },
      { field_key: 'project_budget', field_name: '项目预算', field_type: 'number', selected: true },
      { field_key: 'team_size', field_name: '团队规模', field_type: 'number', selected: true },
      { field_key: 'duration', field_name: '预计周期', field_type: 'text', selected: true },
    ],
    field_mode: 'all',
    fields: [
      { field_key: 'project_name', field_name: '项目名称', field_type: 'text', selected: true },
      { field_key: 'project_budget', field_name: '项目预算', field_type: 'number', selected: true },
      { field_key: 'team_size', field_name: '团队规模', field_type: 'number', selected: true },
      { field_key: 'duration', field_name: '预计周期', field_type: 'text', selected: true },
    ],
    rules: [
      { id: 'R022', process_type: '项目审批', rule_content: '项目预算须在年度规划范围内', rule_scope: 'mandatory', priority: 90, enabled: true, source: 'manual', related_flow: false },
      { id: 'R023', process_type: '项目审批', rule_content: '团队配置须包含必要角色', rule_scope: 'default_on', priority: 80, enabled: true, source: 'manual', related_flow: false },
    ],
    kb_mode: 'rules_only',
    ai_config: {
      audit_strictness: 'standard',
      reasoning_prompt: '你是一个项目审批审核助手。请根据规则对项目立项进行审核。\n\n主表数据：{{main_table}}\n审核规则：{{rules}}\n流程图：{{flow_graph}}',
      extraction_prompt: '请输出结构化审核结论。\n\n原始规则列表：{{rules}}',
    },
    user_permissions: {
      allow_custom_fields: true,
      allow_custom_rules: true,
      allow_modify_strictness: true,
    },
  },
  {
    id: 'PAC-007',
    process_type: '预算审批',
    process_type_label: '预算类',
    main_table_name: 'formtable_main_007',
    main_fields: [
      { field_key: 'budget_name', field_name: '预算名称', field_type: 'text', selected: true },
      { field_key: 'budget_amount', field_name: '预算金额', field_type: 'number', selected: true },
      { field_key: 'budget_period', field_name: '预算周期', field_type: 'text', selected: true },
    ],
    field_mode: 'all',
    fields: [
      { field_key: 'budget_name', field_name: '预算名称', field_type: 'text', selected: true },
      { field_key: 'budget_amount', field_name: '预算金额', field_type: 'number', selected: true },
      { field_key: 'budget_period', field_name: '预算周期', field_type: 'text', selected: true },
    ],
    rules: [
      { id: 'R024', process_type: '预算审批', rule_content: '预算金额须在年度财务规划范围内', rule_scope: 'mandatory', priority: 90, enabled: true, source: 'manual', related_flow: false },
    ],
    kb_mode: 'rules_only',
    ai_config: {
      audit_strictness: 'standard',
      reasoning_prompt: '你是一个预算审批审核助手。请根据规则对预算申请进行审核。\n\n主表数据：{{main_table}}\n审核规则：{{rules}}\n流程图：{{flow_graph}}',
      extraction_prompt: '请输出结构化审核结论。\n\n原始规则列表：{{rules}}',
    },
    user_permissions: {
      allow_custom_fields: true,
      allow_custom_rules: true,
      allow_modify_strictness: false,
    },
  },
]

// ============================================================
//存档审核配置 (归档复盘配置 - 机场管理)
// ============================================================
export const mockArchiveReviewConfigs: ArchiveReviewConfig[] = [
  {
    id: 'ARC-001',
    process_type: '采购审批',
    process_type_label: '采购类',
    main_table_name: 'formtable_main_001',
    main_fields: [
      { field_key: 'amount', field_name: '采购金额', field_type: 'number', selected: true },
      { field_key: 'supplier', field_name: '供应商名称', field_type: 'text', selected: true },
      { field_key: 'contract_no', field_name: '合同编号', field_type: 'text', selected: true },
      { field_key: 'delivery_date', field_name: '交付日期', field_type: 'date', selected: false },
      { field_key: 'payment_terms', field_name: '付款条件', field_type: 'text', selected: true },
      { field_key: 'category', field_name: '采购类别', field_type: 'select', selected: false },
    ],
    detail_tables: [
      {
        table_name: 'formtable_main_001_dt1',
        table_label: '采购明细',
        fields: [
          { field_key: 'item_name', field_name: '物品名称', field_type: 'text', selected: true },
          { field_key: 'item_qty', field_name: '数量', field_type: 'number', selected: true },
          { field_key: 'unit_price', field_name: '单价', field_type: 'number', selected: true },
          { field_key: 'item_spec', field_name: '规格型号', field_type: 'text', selected: false },
        ],
      },
    ],
    field_mode: 'selected',
    fields: [
      { field_key: 'amount', field_name: '采购金额', field_type: 'number', selected: true },
      { field_key: 'supplier', field_name: '供应商名称', field_type: 'text', selected: true },
      { field_key: 'contract_no', field_name: '合同编号', field_type: 'text', selected: true },
      { field_key: 'delivery_date', field_name: '交付日期', field_type: 'date', selected: false },
      { field_key: 'payment_terms', field_name: '付款条件', field_type: 'text', selected: true },
    ],
    rules: [
      { id: 'AR001', process_type: '采购审批', rule_content: '采购金额须在年度预算范围内', rule_scope: 'mandatory', priority: 100, enabled: true, source: 'manual' },
      { id: 'AR002', process_type: '采购审批', rule_content: '超过10万元须提供至少3家供应商比价', rule_scope: 'mandatory', priority: 95, enabled: true, source: 'manual' },
      { id: 'AR003', process_type: '采购审批', rule_content: '供应商须在合格供应商名录中', rule_scope: 'default_on', priority: 85, enabled: true, source: 'file_import' },
      { id: 'AR004', process_type: '采购审批', rule_content: '付款条件须符合公司标准比例', rule_scope: 'default_on', priority: 80, enabled: true, source: 'manual' },
    ],
    flow_rules: [],
    kb_mode: 'rules_only',
    ai_config: {
      audit_strictness: 'standard',
      reasoning_prompt: '你是一个专业的归档合规复核助手。请对已归档的采购审批流程进行全流程合规复核，逐条检查规则并给出判断理由。\n\n主表数据：{{main_table}}\n明细表数据：{{detail_tables}}\n审核规则：{{rules}}',
      extraction_prompt: '请根据以上推理分析结果，严格按照 JSON Schema 输出结构化复核结论。\n\n需要输出：\n1. recommendation：合规性判断\n2. rule_checks：逐条规则校验结果\n3. risk_points：风险点\n4. suggestions：改进建议\n\n原始规则列表：{{rules}}',
    },
    user_permissions: {
      allow_custom_fields: false,
      allow_custom_rules: true,
      allow_custom_flow_rules: false,
      allow_modify_strictness: true,
    },
    allowed_roles: ['d0000000-0000-0000-0000-000000000002', 'd0000000-0000-0000-0000-000000000003'],
    allowed_members: ['e0000000-0000-0000-0000-000000000004', 'e0000000-0000-0000-0000-000000000005'],
    allowed_departments: ['c0000000-0000-0000-0000-000000000001', 'c0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000003'],
  },
  {
    id: 'ARC-002',
    process_type: '费用报销',
    process_type_label: '费用类',
    main_table_name: 'formtable_main_002',
    main_fields: [
      { field_key: 'amount', field_name: '报销金额', field_type: 'number', selected: true },
      { field_key: 'expense_type', field_name: '费用类型', field_type: 'select', selected: true },
      { field_key: 'invoice_count', field_name: '发票数量', field_type: 'number', selected: true },
      { field_key: 'reason', field_name: '报销事由', field_type: 'textarea', selected: false },
    ],
    detail_tables: [
      {
        table_name: 'formtable_main_002_dt1',
        table_label: '发票明细',
        fields: [
          { field_key: 'invoice_no', field_name: '发票号码', field_type: 'text', selected: true },
          { field_key: 'invoice_amount', field_name: '发票金额', field_type: 'number', selected: true },
          { field_key: 'invoice_date', field_name: '发票日期', field_type: 'date', selected: true },
          { field_key: 'invoice_file', field_name: '发票附件', field_type: 'file', selected: false },
        ],
      },
    ],
    field_mode: 'selected',
    fields: [
      { field_key: 'amount', field_name: '报销金额', field_type: 'number', selected: true },
      { field_key: 'expense_type', field_name: '费用类型', field_type: 'select', selected: true },
      { field_key: 'invoice_count', field_name: '发票数量', field_type: 'number', selected: true },
      { field_key: 'reason', field_name: '报销事由', field_type: 'textarea', selected: false },
    ],
    rules: [
      { id: 'AR005', process_type: '费用报销', rule_content: '报销金额超过5000元须附发票原件', rule_scope: 'mandatory', priority: 90, enabled: true, source: 'manual' },
      { id: 'AR006', process_type: '费用报销', rule_content: '发票日期须在报销申请日期前90天内', rule_scope: 'mandatory', priority: 85, enabled: true, source: 'file_import' },
    ],
    flow_rules: [],
    kb_mode: 'rules_only',
    ai_config: {
      audit_strictness: 'standard',
      reasoning_prompt: '你是一个专业的归档合规复核助手。请对已归档的费用报销流程进行全流程合规复核。\n\n主表数据：{{main_table}}\n明细表数据：{{detail_tables}}\n审核规则：{{rules}}',
      extraction_prompt: '请根据以上推理分析结果，严格按照 JSON Schema 输出结构化复核结论。\n\n原始规则列表：{{rules}}',
    },
    user_permissions: {
      allow_custom_fields: true,
      allow_custom_rules: true,
      allow_custom_flow_rules: false,
      allow_modify_strictness: false,
    },
    allowed_roles: ['d0000000-0000-0000-0000-000000000002', 'd0000000-0000-0000-0000-000000000003'],
    allowed_members: ['e0000000-0000-0000-0000-000000000005'],
    allowed_departments: ['c0000000-0000-0000-0000-000000000002', 'c0000000-0000-0000-0000-000000000003'],
  },
  {
    id: 'ARC-003',
    process_type: '合同审批',
    process_type_label: '合同类',
    main_table_name: 'formtable_main_003',
    main_fields: [
      { field_key: 'contract_amount', field_name: '合同金额', field_type: 'number', selected: true },
      { field_key: 'vendor', field_name: '合作方', field_type: 'text', selected: true },
      { field_key: 'contract_period', field_name: '合同期限', field_type: 'text', selected: true },
      { field_key: 'deliverables', field_name: '交付物', field_type: 'textarea', selected: true },
      { field_key: 'contract_file', field_name: '合同文件', field_type: 'file', selected: true },
    ],
    field_mode: 'all',
    fields: [
      { field_key: 'contract_amount', field_name: '合同金额', field_type: 'number', selected: true },
      { field_key: 'vendor', field_name: '合作方', field_type: 'text', selected: true },
      { field_key: 'contract_period', field_name: '合同期限', field_type: 'text', selected: true },
      { field_key: 'deliverables', field_name: '交付物', field_type: 'textarea', selected: true },
    ],
    rules: [
      { id: 'AR007', process_type: '合同审批', rule_content: '合同金额超过50万须法务部会签', rule_scope: 'mandatory', priority: 100, enabled: true, source: 'manual' },
      { id: 'AR008', process_type: '合同审批', rule_content: '合作方须通过准入评审', rule_scope: 'mandatory', priority: 95, enabled: true, source: 'file_import' },
    ],
    flow_rules: [],
    kb_mode: 'rules_only',
    ai_config: {
      audit_strictness: 'strict',
      reasoning_prompt: '你是一个专业的归档合规复核助手。请对已归档的合同审批流程进行全流程合规复核，重点关注法务审核完整性和合同条款合规性。\n\n主表数据：{{main_table}}\n审核规则：{{rules}}',
      extraction_prompt: '请根据以上推理分析结果，严格按照 JSON Schema 输出结构化复核结论。\n\n原始规则列表：{{rules}}',
    },
    user_permissions: {
      allow_custom_fields: false,
      allow_custom_rules: false,
      allow_custom_flow_rules: false,
      allow_modify_strictness: false,
    },
    allowed_roles: ['d0000000-0000-0000-0000-000000000003'],
    allowed_members: ['e0000000-0000-0000-0000-000000000005', 'e0000000-0000-0000-0000-000000000001'],
    allowed_departments: ['c0000000-0000-0000-0000-000000000001'],
  },
  {
    id: 'ARC-004',
    process_type: '人事审批',
    process_type_label: '人事类',
    main_table_name: 'formtable_main_004',
    main_fields: [
      { field_key: 'position', field_name: '岗位名称', field_type: 'text', selected: true },
      { field_key: 'headcount', field_name: '招聘人数', field_type: 'number', selected: true },
      { field_key: 'department', field_name: '用人部门', field_type: 'text', selected: true },
      { field_key: 'onboard_date', field_name: '入职日期', field_type: 'date', selected: false },
    ],
    field_mode: 'selected',
    fields: [
      { field_key: 'position', field_name: '岗位名称', field_type: 'text', selected: true },
      { field_key: 'headcount', field_name: '招聘人数', field_type: 'number', selected: true },
      { field_key: 'department', field_name: '用人部门', field_type: 'text', selected: true },
      { field_key: 'onboard_date', field_name: '入职日期', field_type: 'date', selected: false },
    ],
    rules: [
      { id: 'AR009', process_type: '人事审批', rule_content: '招聘人数须在年度HC计划范围内', rule_scope: 'mandatory', priority: 90, enabled: true, source: 'manual' },
    ],
    flow_rules: [],
    kb_mode: 'rules_only',
    ai_config: {
      audit_strictness: 'loose',
      reasoning_prompt: '你是一个专业的归档合规复核助手。请对已归档的人事审批流程进行全流程合规复核。\n\n主表数据：{{main_table}}\n审核规则：{{rules}}',
      extraction_prompt: '请根据以上推理分析结果，严格按照 JSON Schema 输出结构化复核结论。\n\n原始规则列表：{{rules}}',
    },
    user_permissions: {
      allow_custom_fields: true,
      allow_custom_rules: true,
      allow_custom_flow_rules: true,
      allow_modify_strictness: true,
    },
    allowed_roles: ['d0000000-0000-0000-0000-000000000002', 'd0000000-0000-0000-0000-000000000003'],
    allowed_members: ['e0000000-0000-0000-0000-000000000004'],
    allowed_departments: ['c0000000-0000-0000-0000-000000000002'],
  },
]

// ============================================================
//严格性提示预设（读出读数默认提示词 - 显示宽度）
// ============================================================
export const mockStrictnessPresets: StrictnessPromptPreset[] = [
  {
    strictness: 'strict',
    reasoning_instruction: '请以最严格的标准审核，任何疑点均应标记为不合规。宁可误判也不放过，对所有可疑项逐一深入分析，给出明确的退回建议。',
    extraction_instruction: '请以最严格标准提取结论：任何存疑项均应判定为不通过，建议操作倾向于 return，仅在完全合规时才建议 approve。',
  },
  {
    strictness: 'standard',
    reasoning_instruction: '请以常规标准审核，明确违规项标记为不合规并说明理由，存疑项需详细说明疑点并给出改进建议。整体评估应客观公正。',
    extraction_instruction: '请以常规标准提取结论：明确违规项判定为不通过并建议 return，合规项判定为通过，存疑项说明理由。建议操作应综合考虑整体合规情况。',
  },
  {
    strictness: 'loose',
    reasoning_instruction: '请以宽松标准审核，仅明显违规项标记为不合规，轻微问题可标注但不作为不合规依据。重点关注重大风险，对小问题保持宽容。',
    extraction_instruction: '请以宽松标准提取结论：仅明显违规项判定为不通过，轻微问题标注但不影响最终建议。建议操作倾向于 approve，仅严重问题才建议 return。',
  },
]

//模拟 API：获取租户的严格预设
export const fetchStrictnessPresets = async (_tenantId?: string): Promise<StrictnessPromptPreset[]> => {
  await new Promise(r => setTimeout(r, 300))
  return JSON.parse(JSON.stringify(mockStrictnessPresets))
}

//模拟 API：为租户保存严格预设
export const saveStrictnessPresets = async (_tenantId: string, presets: StrictnessPromptPreset[]): Promise<boolean> => {
  await new Promise(r => setTimeout(r, 500))
  //在实际实现中，这将持续到后端
  mockStrictnessPresets.splice(0, mockStrictnessPresets.length, ...presets)
  return true
}

export const useMockData = () => {
  const mockProcesses: OAProcess[] = [
    {
      process_id: 'WF-2025-001',
      title: '办公设备采购申请 - 研发部笔记本电脑',
      applicant: '张明',
      department: '研发部',
      submit_time: '2025-06-10 09:30',
      process_type: '采购审批',
      status: 'pending',
      current_node: '财务总监审批',
      amount: 156000,
      urgency: 'medium',
    },
    {
      process_id: 'WF-2025-002',
      title: '差旅费报销 - 上海客户拜访',
      applicant: '李芳',
      department: '销售部',
      submit_time: '2025-06-10 10:15',
      process_type: '费用报销',
      status: 'pending',
      current_node: '部门经理审批',
      amount: 8500,
      urgency: 'low',
    },
    {
      process_id: 'WF-2025-003',
      title: '年度服务器租赁合同续签',
      applicant: '王强',
      department: 'IT部',
      submit_time: '2025-06-10 11:00',
      process_type: '合同审批',
      status: 'pending',
      current_node: '法务审核',
      amount: 480000,
      urgency: 'high',
    },
    {
      process_id: 'WF-2025-004',
      title: '新员工入职审批 - 产品经理',
      applicant: '赵丽',
      department: '人力资源部',
      submit_time: '2025-06-10 14:20',
      process_type: '人事审批',
      status: 'pending',
      current_node: 'HR经理审批',
      urgency: 'medium',
    },
    {
      process_id: 'WF-2025-005',
      title: '市场推广活动预算申请',
      applicant: '陈伟',
      department: '市场部',
      submit_time: '2025-06-10 15:45',
      process_type: '预算审批',
      status: 'pending',
      current_node: '财务总监审批',
      amount: 250000,
      urgency: 'high',
    },
    {
      process_id: 'WF-2025-006',
      title: '办公室装修改造方案',
      applicant: '刘洋',
      department: '行政部',
      submit_time: '2025-06-09 16:30',
      process_type: '工程审批',
      status: 'pending',
      current_node: '部门经理审批',
      amount: 320000,
      urgency: 'low',
    },
    {
      process_id: 'WF-2025-007',
      title: '客户接待费用预支申请',
      applicant: '李芳',
      department: '销售部',
      submit_time: '2025-06-09 14:00',
      process_type: '费用报销',
      status: 'pending',
      current_node: '财务审核',
      amount: 35000,
      urgency: 'medium',
    },
    {
      process_id: 'WF-2025-009',
      title: '研发部门年度培训计划',
      applicant: '张明',
      department: '研发部',
      submit_time: '2025-06-09 11:20',
      process_type: '预算审批',
      status: 'pending',
      current_node: '总经理审批',
      amount: 120000,
      urgency: 'low',
    },
    {
      process_id: 'WF-2025-010',
      title: '数据中心UPS电源更换',
      applicant: '王强',
      department: 'IT部',
      submit_time: '2025-06-09 09:45',
      process_type: '采购审批',
      status: 'pending',
      current_node: '财务总监审批',
      amount: 280000,
      urgency: 'high',
    },
    {
      process_id: 'WF-2025-011',
      title: '2025年校园招聘方案审批',
      applicant: '赵丽',
      department: '人力资源部',
      submit_time: '2025-06-08 16:30',
      process_type: '人事审批',
      status: 'pending',
      current_node: 'HR总监审批',
      urgency: 'medium',
    },
    {
      process_id: 'WF-2025-013',
      title: '华南区域代理商合同签署',
      applicant: '陈伟',
      department: '市场部',
      submit_time: '2025-06-08 14:10',
      process_type: '合同审批',
      status: 'pending',
      current_node: '总经理审批',
      amount: 560000,
      urgency: 'high',
    },
    {
      process_id: 'WF-2025-014',
      title: '办公区安防系统升级',
      applicant: '刘洋',
      department: '行政部',
      submit_time: '2025-06-08 10:00',
      process_type: '工程审批',
      status: 'pending',
      current_node: '部门经理审批',
      amount: 185000,
      urgency: 'low',
    },
  ]

  const mockAuditResult: AuditResult = {
    trace_id: 'TR-20250610-A3F8',
    process_id: 'WF-2025-001',
    recommendation: 'return',
    score: 72,
    duration_ms: 3850,
    details: [
      { rule_id: 'R001', rule_name: '预算额度校验', passed: true, reasoning: '采购金额 ¥156,000 未超过部门季度预算上限 ¥200,000', is_locked: true },
      { rule_id: 'R002', rule_name: '审批层级校验', passed: true, reasoning: '金额在 10-20 万区间，需部门经理 + 财务总监双签，审批链完整' },
      { rule_id: 'R003', rule_name: '供应商资质校验', passed: false, reasoning: '供应商"XX科技"未在合格供应商名录中，建议补充资质证明或更换供应商', is_locked: true },
      { rule_id: 'R004', rule_name: '采购比价要求', passed: false, reasoning: '单笔采购超过 ¥100,000 需提供至少 3 家供应商报价，当前仅提供 1 家' },
      { rule_id: 'R005', rule_name: '合同条款完整性', passed: true, reasoning: '合同包含付款条件、交付时间、售后条款等必要条款' },
    ],
    ai_reasoning: '该采购申请整体合规性尚可，但存在两个关键问题需要修正：\n\n1. 供应商资质问题：所选供应商未在企业合格供应商名录中登记，存在合规风险。建议申请人补充供应商资质材料或从已认证供应商中选择。\n\n2. 比价流程缺失：根据公司采购管理制度第 12 条，单笔采购金额超过 10 万元需进行竞争性比价（至少 3 家），当前申请仅提供了单一报价。\n\n建议：退回修改，要求补充比价材料和供应商资质证明后重新提交。',
    //v2 字段
    action_label: '建议退回',
    confidence: 0.85,
    risk_points: ['供应商未在合格名录中', '缺少竞争性比价材料'],
    suggestions: ['补充供应商资质证明', '提供至少3家供应商报价'],
    ai_summary: '该采购申请整体合规性尚可，但存在两个关键问题需要修正。',
    model_used: 'Qwen2.5-72B',
    interaction_mode: 'two_phase',
    phase1_duration_ms: 2200,
    phase2_duration_ms: 1650,
  }

  const mockBatchAuditResult = {
    batch_id: 'BATCH-20250610-001',
    total: 3,
    completed: 2,
    failed: 0,
    status: 'processing' as const,
    progress_percent: 66,
    results: [
      { process_id: 'WF-2025-001', status: 'completed', recommendation: 'return', action_label: '建议退回', score: 72 },
      { process_id: 'WF-2025-002', status: 'completed', recommendation: 'approve', action_label: '建议通过', score: 88 },
      { process_id: 'WF-2025-003', status: 'in_progress', recommendation: null, action_label: null, score: null },
    ],
  }

  //一些待办事项流程的预先计算的审核结果（模拟已审核的项目）
  const mockTodoAuditResults: Record<string, AuditResult> = {
    'WF-2025-001': { ...mockAuditResult },
    'WF-2025-002': {
      trace_id: 'TR-20250610-B2D4', process_id: 'WF-2025-002', recommendation: 'approve', score: 88, duration_ms: 1280,
      details: [
        { rule_id: 'R006', rule_name: '差旅标准校验', passed: true, reasoning: '差旅费用在公司标准范围内', is_locked: true },
        { rule_id: 'R007', rule_name: '发票合规性', passed: true, reasoning: '发票信息完整，日期与行程匹配' },
        { rule_id: 'R008', rule_name: '审批材料完整性', passed: true, reasoning: '出差申请单、行程单、发票齐全' },
      ],
      ai_reasoning: '该差旅费报销申请完全合规，费用在标准范围内，材料齐全，发票合规。建议通过。',
      action_label: '建议通过', confidence: 0.92, risk_points: [],
      suggestions: ['建议后续出差提前提交预算申请'],
      ai_summary: '差旅费报销合规，材料齐全。',
      model_used: 'Qwen2.5-72B', interaction_mode: 'single_pass', phase1_duration_ms: 1280, phase2_duration_ms: 0,
    },
    'WF-2025-003': {
      trace_id: 'TR-20250610-C3E5', process_id: 'WF-2025-003', recommendation: 'review', score: 65, duration_ms: 2100,
      details: [
        { rule_id: 'R001', rule_name: '预算额度校验', passed: true, reasoning: '合同金额在年度IT预算范围内' },
        { rule_id: 'R004', rule_name: '合同条款完整性', passed: false, reasoning: 'SLA条款缺少故障响应时间约定', is_locked: true },
        { rule_id: 'R009', rule_name: '续签合理性评估', passed: true, reasoning: '服务商过去一年服务记录良好' },
        { rule_id: 'R010', rule_name: '比价要求', passed: false, reasoning: '续签合同金额较上年增长15%，建议补充市场比价' },
      ],
      ai_reasoning: '该合同续签整体可行，但存在两个需关注的问题：SLA条款不完整和价格涨幅较大。建议人工复核后决定。',
      action_label: '建议复核', confidence: 0.78,
      risk_points: ['SLA条款缺少故障响应时间', '合同金额较上年增长15%'],
      suggestions: ['补充SLA故障响应时间条款', '要求供应商提供涨价依据', '考虑引入竞争性报价'],
      ai_summary: '合同续签可行但需关注SLA和价格问题。',
      model_used: 'Qwen2.5-72B', interaction_mode: 'two_phase', phase1_duration_ms: 1300, phase2_duration_ms: 800,
    },
    'WF-2025-005': {
      trace_id: 'TR-20250610-D4F6', process_id: 'WF-2025-005', recommendation: 'approve', score: 95, duration_ms: 1050,
      details: [
        { rule_id: 'R001', rule_name: '预算额度校验', passed: true, reasoning: '预算金额在市场部年度预算范围内', is_locked: true },
        { rule_id: 'R011', rule_name: '活动方案完整性', passed: true, reasoning: '推广方案包含目标、渠道、预期ROI等完整信息' },
        { rule_id: 'R012', rule_name: '审批层级校验', passed: true, reasoning: '金额超过20万，已获得部门经理和财务总监签批' },
      ],
      ai_reasoning: '市场推广预算申请完全合规，方案详实，预算合理，审批链完整。建议通过。',
      action_label: '建议通过', confidence: 0.96, risk_points: [],
      suggestions: ['建议活动结束后提交效果评估报告'],
      ai_summary: '市场推广预算申请完全合规。',
      model_used: 'Qwen2.5-72B', interaction_mode: 'single_pass', phase1_duration_ms: 1050, phase2_duration_ms: 0,
    },
  }

  const mockCronTasks: CronTask[] = [{ id: 'CT-BUILTIN-001', cron_expression: '0 9 * * 1-5', task_type: 'batch_audit', is_active: true, last_run_at: '2025-06-10 09:00', next_run_at: '2025-06-11 09:00', created_at: '2025-05-01', success_count: 28, fail_count: 1, is_builtin: true },
  { id: 'CT-002', cron_expression: '0 18 * * 1-5', task_type: 'daily_report', is_active: true, last_run_at: '2025-06-09 18:00', next_run_at: '2025-06-10 18:00', created_at: '2025-05-01', success_count: 30, fail_count: 0, push_email: 'zhangming@example.com' },
  { id: 'CT-003', cron_expression: '0 10 * * 1', task_type: 'weekly_report', is_active: true, last_run_at: '2025-06-09 10:00', next_run_at: '2025-06-16 10:00', created_at: '2025-05-15', success_count: 4, fail_count: 0, push_email: 'zhangming@example.com' },
  { id: 'CT-004', cron_expression: '0 2 * * *', task_type: 'batch_audit', is_active: false, last_run_at: '2025-06-08 02:00', next_run_at: '-', created_at: '2025-04-20', success_count: 15, fail_count: 3 },
  ]

  // ============================================================
  //Cron 任务类型配置 (机场管理 - 定时任务配置)
  // ============================================================
  const mockCronTaskTypeConfigs: CronTaskTypeConfig[] = [
    {
      task_type: 'batch_audit',
      label: '批量审核',
      enabled: true,
      batch_limit: 50,
      push_format: 'html',
      content_template: {
        subject: '',
        header: '',
        body_template: '',
        footer: '',
      },
    },
    {
      task_type: 'daily_report',
      label: '日报推送',
      enabled: true,
      push_format: 'html',
      content_template: {
        subject: '【OA智审】审核日报 - {{date}}',
        header: '今日审核工作概览：',
        body_template: '今日共处理 {{total}} 条审核，通过 {{approved}} 条，退回 {{returned}} 条。通过率 {{pass_rate}}%。\n\n{{detail_list}}\n\n以上数据截至 {{time}}。',
        footer: '详情请登录系统查看。此邮件由系统自动发送，请勿直接回复。',
      },
    },
    {
      task_type: 'weekly_report',
      label: '周报推送',
      enabled: true,
      push_format: 'markdown',
      content_template: {
        subject: '【OA智审】审核周报 - 第{{week}}周（{{date_range}}）',
        header: '本周审核工作总结：',
        body_template: '本周共处理 {{total}} 条审核，较上周{{trend}}。合规率 {{compliance_rate}}%，环比{{compliance_trend}}。\n\n{{statistics}}\n\n{{detail_list}}',
        footer: '报告生成时间：{{time}}。如需详细数据请导出归档记录。',
      },
    },
  ]

  const mockSnapshots: AuditSnapshot[] = [
    { snapshot_id: 'SN-001', process_id: 'WF-2025-098', title: '年度IT设备采购', applicant: '王强', department: 'IT部', recommendation: 'approve', score: 95, created_at: '2025-06-09 16:30', adopted: true },
    { snapshot_id: 'SN-002', process_id: 'WF-2025-097', title: '客户招待费报销', applicant: '李芳', department: '销售部', recommendation: 'return', score: 35, created_at: '2025-06-09 15:20', adopted: true },
    { snapshot_id: 'SN-003', process_id: 'WF-2025-096', title: '新产品研发立项', applicant: '张明', department: '研发部', recommendation: 'approve', score: 88, created_at: '2025-06-09 14:10', adopted: true },
    { snapshot_id: 'SN-004', process_id: 'WF-2025-095', title: '办公用品批量采购', applicant: '刘洋', department: '行政部', recommendation: 'return', score: 62, created_at: '2025-06-09 11:45', adopted: false },
    { snapshot_id: 'SN-005', process_id: 'WF-2025-094', title: '员工培训费用申请', applicant: '赵丽', department: '人力资源部', recommendation: 'approve', score: 91, created_at: '2025-06-08 17:00', adopted: true },
    { snapshot_id: 'SN-006', process_id: 'WF-2025-093', title: '广告投放合同签署', applicant: '陈伟', department: '市场部', recommendation: 'return', score: 58, created_at: '2025-06-08 14:30', adopted: null },
  ]

  const mockTenants: TenantInfo[] = [
    {
      id: 'T-001', name: '示例集团总部', code: 'DEMO_HQ', oa_type: 'weaver_e9',
      oa_db_connection_id: 'OADB-001',
      token_quota: 100000, token_used: 42350, max_concurrency: 20, status: 'active', created_at: '2025-01-15',
      contact_name: '张明', contact_email: 'zhangming@demo-group.com', contact_phone: '138****8888',
      description: '示例集团总部，使用泛微E9 OA系统，主要用于采购、合同、报销等流程审核',
      ai_config: {
        default_provider: '本地部署', default_model: 'Qwen2.5-72B',
        fallback_provider: '云端API', fallback_model: 'qwen-plus',
        max_tokens_per_request: 8192, temperature: 0.3, timeout_seconds: 60, retry_count: 3,
      },
      log_retention_days: 365, data_retention_days: 1095,
      allow_custom_model: true, sso_enabled: true, sso_endpoint: 'https://sso.demo-group.com/oauth2',
      tenant_admin_id: 'tenantadmin',
    },
    {
      id: 'T-002', name: '华东分公司', code: 'EAST_BRANCH', oa_type: 'weaver_e9',
      oa_db_connection_id: 'OADB-002',
      token_quota: 50000, token_used: 18200, max_concurrency: 10, status: 'active', created_at: '2025-02-20',
      contact_name: '李芳', contact_email: 'lifang@demo-east.com', contact_phone: '139****6666',
      description: '华东区域分公司，与总部共享OA基础配置，独立Token配额',
      ai_config: {
        default_provider: '本地部署', default_model: 'Qwen2.5-72B',
        fallback_provider: '', fallback_model: '',
        max_tokens_per_request: 4096, temperature: 0.3, timeout_seconds: 45, retry_count: 2,
      },
      log_retention_days: 180, data_retention_days: 730,
      allow_custom_model: false, sso_enabled: false, sso_endpoint: '',
      tenant_admin_id: 'tenantadmin2',
    },
    {
      id: 'T-003', name: '测试租户', code: 'TEST_TENANT', oa_type: 'weaver_e9',
      oa_db_connection_id: 'OADB-003',
      token_quota: 10000, token_used: 3100, max_concurrency: 5, status: 'inactive', created_at: '2025-03-10',
      contact_name: '系统管理员', contact_email: 'admin@test.com', contact_phone: '130****7777',
      description: '用于系统测试和演示的租户环境',
      ai_config: {
        default_provider: '本地部署', default_model: 'Qwen2.5-32B',
        fallback_provider: '', fallback_model: '',
        max_tokens_per_request: 2048, temperature: 0.5, timeout_seconds: 30, retry_count: 1,
      },
      log_retention_days: 30, data_retention_days: 90,
      allow_custom_model: true, sso_enabled: false, sso_endpoint: '',
    },
  ]

  // ============================================================
  //系统设置模拟数据（系统设置）
  // ============================================================
  const mockOASystemConfigs: OASystemConfig[] = [
    {
      id: 'OA-001', name: '泛微 Ecology E9', type: 'weaver_e9', type_label: '泛微 Ecology E9',
      version: 'v10.x', status: 'connected', description: '泛微协同办公平台 E9 版本，支持 JDBC 直连和 REST API 两种数据获取方式',
      adapter_version: '2.1.0', last_sync: '2026/2/23 12:17:04', sync_interval: 30, enabled: true,
    },
  ]

  const mockOADatabaseConnections: OADatabaseConnection[] = [
    {
      id: 'OADB-001', name: '总部泛微E9数据库', oa_type: 'weaver_e9', oa_type_label: '泛微 Ecology E9',
      jdbc_config: {
        driver: 'mysql', host: '192.168.1.100', port: 3306, database: 'ecology',
        username: 'oa_reader', password: '********', pool_size: 20,
        connection_timeout: 30, test_on_borrow: true,
      },
      status: 'connected', last_sync: '2026/2/23 12:17:04', sync_interval: 30, enabled: true,
      created_at: '2025-01-10', description: '总部泛微E9 OA系统主数据库，用于流程数据同步',
    },
    {
      id: 'OADB-002', name: '华东分公司E9数据库', oa_type: 'weaver_e9', oa_type_label: '泛微 Ecology E9',
      jdbc_config: {
        driver: 'mysql', host: '192.168.2.100', port: 3306, database: 'ecology_east',
        username: 'oa_reader', password: '********', pool_size: 10,
        connection_timeout: 30, test_on_borrow: true,
      },
      status: 'connected', last_sync: '2026/2/23 11:20:10', sync_interval: 60, enabled: true,
      created_at: '2025-02-15', description: '华东分公司泛微E9数据库',
    },
    {
      id: 'OADB-003', name: '测试环境数据库', oa_type: 'weaver_e9', oa_type_label: '泛微 Ecology E9',
      jdbc_config: {
        driver: 'oracle', host: 'localhost', port: 1521, database: 'ecology_test',
        username: 'test_reader', password: '********', pool_size: 5,
        connection_timeout: 15, test_on_borrow: false,
      },
      status: 'disconnected', last_sync: '', sync_interval: 120, enabled: false,
      created_at: '2025-03-05', description: '用于系统测试和演示的OA数据库连接',
    },
  ]

  const mockAIModelConfigs: AIModelConfig[] = [
    {
      id: 'AI-001', provider: 'Xinference', model_name: 'Qwen2.5-72B', display_name: 'Qwen2.5-72B（本地）',
      type: 'local', endpoint: 'http://192.168.1.50:9997/v1', api_key_configured: false,
      max_tokens: 8192, context_window: 131072, cost_per_1k_tokens: 0,
      status: 'online', enabled: true,
      description: '通义千问2.5 72B 参数大模型，通过 Xinference 本地私有部署，数据不出域',
      capabilities: ['text', 'code', 'reasoning', 'analysis'],
    },
    {
      id: 'AI-002', provider: 'Xinference', model_name: 'Qwen2.5-32B', display_name: 'Qwen2.5-32B（本地）',
      type: 'local', endpoint: 'http://192.168.1.50:9997/v1', api_key_configured: false,
      max_tokens: 4096, context_window: 65536, cost_per_1k_tokens: 0,
      status: 'online', enabled: true,
      description: '通义千问2.5 32B 参数大模型，通过 Xinference 部署，适合轻量级审核任务',
      capabilities: ['text', 'code', 'reasoning'],
    },
    {
      id: 'AI-003', provider: '阿里云百炼', model_name: 'qwen-plus', display_name: 'Qwen-Plus（阿里云百炼）',
      type: 'cloud', endpoint: 'https://dashscope.aliyuncs.com/compatible-mode/v1', api_key_configured: true,
      max_tokens: 16384, context_window: 131072, cost_per_1k_tokens: 0.008,
      status: 'online', enabled: true,
      description: '阿里云百炼 Qwen-Plus 大模型，云端部署，性价比高',
      capabilities: ['text', 'code', 'reasoning', 'analysis'],
    },
    {
      id: 'AI-004', provider: '阿里云百炼', model_name: 'qwen-max', display_name: 'Qwen-Max（阿里云百炼）',
      type: 'cloud', endpoint: 'https://dashscope.aliyuncs.com/compatible-mode/v1', api_key_configured: true,
      max_tokens: 8192, context_window: 131072, cost_per_1k_tokens: 0.02,
      status: 'online', enabled: false,
      description: '阿里云百炼 Qwen-Max 旗舰模型，适合复杂合同和法务审核',
      capabilities: ['text', 'code', 'reasoning', 'vision', 'analysis'],
    },
    {
      id: 'AI-005', provider: 'Xinference', model_name: 'DeepSeek-V3', display_name: 'DeepSeek-V3（本地）',
      type: 'local', endpoint: 'http://192.168.1.51:9997/v1', api_key_configured: false,
      max_tokens: 8192, context_window: 65536, cost_per_1k_tokens: 0,
      status: 'maintenance', enabled: false,
      description: 'DeepSeek V3 大模型，通过 Xinference 部署，擅长代码和推理任务',
      capabilities: ['text', 'code', 'reasoning'],
    },
  ]

  const mockSystemGeneralConfig: SystemGeneralConfig = {
    platform_name: 'OA流程智能审核平台',
    platform_version: 'v1.2.0',
    default_language: 'zh-CN',
    session_timeout: 120,
    max_upload_size: 50,
    enable_audit_trail: true,
    enable_data_encryption: true,
    backup_enabled: true,
    backup_cron: '0 2 * * *',
    backup_retention_days: 30,
    notification_email: 'admin@oa-smart-audit.com',
    smtp_host: 'smtp.example.com',
    smtp_port: 465,
    smtp_username: 'noreply@oa-smart-audit.com',
    smtp_ssl: true,
  }



  //从流程审核配置中派生规则以实现向后兼容性
  const mockRules: AuditRule[] = mockProcessAuditConfigs.flatMap(c => c.rules)

  const mockDashboardStats: DashboardStats = {
    todayAudits: 42,
    todayApproved: 28,
    todayReturned: 14,
    pendingCount: 6,
    avgResponseMs: 1850,
    successRate: 99.2,
    weeklyTrend: [
      { date: '06-04', count: 35 },
      { date: '06-05', count: 41 },
      { date: '06-06', count: 38 },
      { date: '06-07', count: 22 },
      { date: '06-08', count: 15 },
      { date: '06-09', count: 44 },
      { date: '06-10', count: 42 },
    ],
  }


  //批准的流程 - 历史、只读
  const mockApprovedProcesses: OAProcess[] = [
    { process_id: 'WF-2025-098', title: '年度IT设备采购', applicant: '王强', department: 'IT部', submit_time: '2025-06-09 16:30', process_type: '采购审批', status: 'approved', current_node: '已完成', amount: 320000, urgency: 'medium' },
    { process_id: 'WF-2025-096', title: '新产品研发立项', applicant: '张明', department: '研发部', submit_time: '2025-06-09 14:10', process_type: '项目审批', status: 'approved', current_node: '已完成', urgency: 'high' },
    { process_id: 'WF-2025-094', title: '员工培训费用申请', applicant: '赵丽', department: '人力资源部', submit_time: '2025-06-08 17:00', process_type: '费用报销', status: 'approved', current_node: '已完成', amount: 45000, urgency: 'low' },
    { process_id: 'WF-2025-090', title: '办公家具批量采购', applicant: '刘洋', department: '行政部', submit_time: '2025-06-07 10:00', process_type: '采购审批', status: 'approved', current_node: '已完成', amount: 98000, urgency: 'low' },
    { process_id: 'WF-2025-088', title: '年度广告投放合同', applicant: '陈伟', department: '市场部', submit_time: '2025-06-06 15:30', process_type: '合同审批', status: 'approved', current_node: '已完成', amount: 750000, urgency: 'high' },
    { process_id: 'WF-2025-085', title: '销售团队季度奖金发放', applicant: '李芳', department: '销售部', submit_time: '2025-06-06 09:00', process_type: '费用报销', status: 'approved', current_node: '已完成', amount: 180000, urgency: 'medium' },
    { process_id: 'WF-2025-082', title: '网络安全设备采购', applicant: '王强', department: 'IT部', submit_time: '2025-06-05 14:20', process_type: '采购审批', status: 'approved', current_node: '已完成', amount: 420000, urgency: 'high' },
    { process_id: 'WF-2025-079', title: '实习生转正审批（3人）', applicant: '赵丽', department: '人力资源部', submit_time: '2025-06-05 11:00', process_type: '人事审批', status: 'approved', current_node: '已完成', urgency: 'medium' },
    { process_id: 'WF-2025-076', title: '会议室音视频系统升级', applicant: '刘洋', department: '行政部', submit_time: '2025-06-04 16:00', process_type: '工程审批', status: 'approved', current_node: '已完成', amount: 135000, urgency: 'low' },
  ]

  //返回的进程 - 历史、只读
  const mockReturnedProcesses: OAProcess[] = [
    { process_id: 'WF-2025-097', title: '客户招待费报销', applicant: '李芳', department: '销售部', submit_time: '2025-06-09 15:20', process_type: '费用报销', status: 'returned', current_node: '已退回', amount: 28000, urgency: 'medium' },
    { process_id: 'WF-2025-091', title: '未经审批的外包合同', applicant: '陈伟', department: '市场部', submit_time: '2025-06-08 10:00', process_type: '合同审批', status: 'returned', current_node: '已退回', amount: 150000, urgency: 'high' },
    { process_id: 'WF-2025-087', title: '超标准差旅费报销', applicant: '张明', department: '研发部', submit_time: '2025-06-07 09:30', process_type: '费用报销', status: 'returned', current_node: '已退回', amount: 42000, urgency: 'low' },
    { process_id: 'WF-2025-083', title: '未备案供应商采购申请', applicant: '刘洋', department: '行政部', submit_time: '2025-06-06 11:00', process_type: '采购审批', status: 'returned', current_node: '已退回', amount: 95000, urgency: 'medium' },
  ]

  //由 process_id 键控的历史审计结果
  const mockHistoricalResults: Record<string, AuditResult> = {
    'WF-2025-098': {
      trace_id: 'TR-20250609-B1C2', process_id: 'WF-2025-098', recommendation: 'approve', score: 95, duration_ms: 1420,
      details: [
        { rule_id: 'R001', rule_name: '预算额度校验', passed: true, reasoning: '采购金额在部门年度预算范围内', is_locked: true },
        { rule_id: 'R002', rule_name: '审批层级校验', passed: true, reasoning: '审批链完整，已获得所有必要签批' },
        { rule_id: 'R003', rule_name: '供应商资质校验', passed: true, reasoning: '供应商在合格名录中，资质有效期内' },
      ],
      ai_reasoning: '该采购申请完全符合公司采购管理制度要求，预算合理、审批链完整、供应商资质齐全。建议通过。',
      action_label: '建议通过', confidence: 0.95, risk_points: [],
      suggestions: ['可考虑在合同中增加售后服务条款'],
      ai_summary: '该采购申请完全符合公司采购管理制度要求。',
      model_used: 'Qwen2.5-72B', interaction_mode: 'two_phase', phase1_duration_ms: 850, phase2_duration_ms: 570,
    },
    'WF-2025-096': {
      trace_id: 'TR-20250609-D3E4', process_id: 'WF-2025-096', recommendation: 'approve', score: 88, duration_ms: 1680,
      details: [
        { rule_id: 'R010', rule_name: '立项必要性评估', passed: true, reasoning: '市场调研数据充分，立项理由成立' },
        { rule_id: 'R011', rule_name: '预算可行性', passed: true, reasoning: '研发预算在年度规划范围内' },
        { rule_id: 'R012', rule_name: '人员配置合理性', passed: false, reasoning: '项目团队缺少测试工程师角色，但不影响立项' },
      ],
      ai_reasoning: '研发立项申请整体合规，市场调研充分，预算合理。建议补充测试人员配置后通过。',
      action_label: '建议通过', confidence: 0.88, risk_points: ['项目团队缺少测试工程师角色'],
      suggestions: ['建议在项目启动前补充测试工程师配置'],
      ai_summary: '研发立项申请整体合规，市场调研充分，预算合理。',
      model_used: 'Qwen2.5-72B', interaction_mode: 'two_phase', phase1_duration_ms: 1020, phase2_duration_ms: 660,
    },
    'WF-2025-094': {
      trace_id: 'TR-20250608-F5G6', process_id: 'WF-2025-094', recommendation: 'approve', score: 91, duration_ms: 1150,
      details: [
        { rule_id: 'R003', rule_name: '费用标准校验', passed: true, reasoning: '培训费用符合公司标准' },
        { rule_id: 'R004', rule_name: '培训计划审核', passed: true, reasoning: '培训内容与岗位需求匹配' },
      ],
      ai_reasoning: '员工培训费用申请合规，培训内容与业务需求高度相关，费用在标准范围内。建议通过。',
      action_label: '建议通过', confidence: 0.92, risk_points: [],
      suggestions: [],
      ai_summary: '员工培训费用申请合规，费用在标准范围内。',
      model_used: 'Qwen2.5-72B', interaction_mode: 'single_pass', phase1_duration_ms: 1150, phase2_duration_ms: 0,
    },
    'WF-2025-097': {
      trace_id: 'TR-20250609-H7I8', process_id: 'WF-2025-097', recommendation: 'return', score: 35, duration_ms: 1320,
      details: [
        { rule_id: 'R003', rule_name: '费用标准校验', passed: false, reasoning: '招待费用超出公司标准上限 200%', is_locked: true },
        { rule_id: 'R006', rule_name: '审批材料完整性', passed: false, reasoning: '缺少客户拜访记录和招待事由说明' },
        { rule_id: 'R007', rule_name: '发票合规性', passed: false, reasoning: '部分发票日期与申报时间不符' },
      ],
      ai_reasoning: '该报销申请存在多项严重违规：费用严重超标、材料不完整、发票存疑。建议退回并要求重新整理材料。',
      action_label: '建议退回', confidence: 0.93, risk_points: ['招待费用超出标准上限200%', '缺少客户拜访记录', '发票日期存疑'],
      suggestions: ['重新整理合规发票', '补充客户拜访记录', '按公司标准重新申报'],
      ai_summary: '该报销申请存在多项严重违规，建议退回。',
      model_used: 'Qwen2.5-72B', interaction_mode: 'two_phase', phase1_duration_ms: 780, phase2_duration_ms: 540,
    },
    'WF-2025-091': {
      trace_id: 'TR-20250608-J9K0', process_id: 'WF-2025-091', recommendation: 'return', score: 22, duration_ms: 1560,
      details: [
        { rule_id: 'R004', rule_name: '合同审批前置条件', passed: false, reasoning: '合同签署前未经过法务审核', is_locked: true },
        { rule_id: 'R008', rule_name: '供应商准入', passed: false, reasoning: '外包供应商未通过准入评审' },
        { rule_id: 'R009', rule_name: '预算审批', passed: false, reasoning: '合同金额未纳入年度预算' },
      ],
      ai_reasoning: '该合同存在严重合规问题：未经法务审核即签署、供应商未准入、预算未审批。建议退回并启动合规调查。',
      action_label: '建议退回', confidence: 0.97, risk_points: ['未经法务审核', '供应商未通过准入评审', '合同金额未纳入预算'],
      suggestions: ['启动合规调查', '补充法务审核流程', '完成供应商准入评审'],
      ai_summary: '该合同存在严重合规问题，建议退回并启动合规调查。',
      model_used: 'Qwen2.5-72B', interaction_mode: 'two_phase', phase1_duration_ms: 950, phase2_duration_ms: 610,
    },
  }

  // ============================================================
  //Archive Review (归档复盘) - 全流程合规性重新审核
  // ============================================================

  //已完成所有审批节点的归档流程
  const mockArchivedProcesses: ArchivedProcess[] = [
    {
      process_id: 'WF-2025-050',
      title: '2025年度服务器集群采购',
      applicant: '王强',
      department: 'IT部',
      process_type: '采购审批',
      amount: 1200000,
      submit_time: '2025-04-15 09:00',
      archive_time: '2025-05-20 17:30',
      status: 'archived',
      flow_nodes: [
        { node_id: 'N1', node_name: '部门经理审批', approver: '李明', action: 'approve', action_time: '2025-04-16 10:00', opinion: '同意，符合年度IT规划' },
        { node_id: 'N2', node_name: '财务总监审批', approver: '张华', action: 'approve', action_time: '2025-04-18 14:30', opinion: '预算充足，同意' },
        { node_id: 'N3', node_name: '总经理审批', approver: '刘总', action: 'approve', action_time: '2025-04-20 09:15', opinion: '批准' },
      ],
      fields: { supplier: 'XX云计算有限公司', contract_no: 'HT-2025-0088', delivery_date: '2025-06-30', payment_terms: '分期付款（30%/40%/30%）' },
    },
    {
      process_id: 'WF-2025-038',
      title: '华东区域市场推广费用报销',
      applicant: '陈伟',
      department: '市场部',
      process_type: '费用报销',
      amount: 85000,
      submit_time: '2025-03-20 11:00',
      archive_time: '2025-04-10 16:00',
      status: 'archived',
      flow_nodes: [
        { node_id: 'N1', node_name: '部门经理审批', approver: '周磊', action: 'approve', action_time: '2025-03-21 09:30', opinion: '费用合理' },
        { node_id: 'N2', node_name: '财务审核', approver: '张华', action: 'return', action_time: '2025-03-22 14:00', opinion: '部分发票不清晰，请补充' },
        { node_id: 'N3', node_name: '财务审核（重审）', approver: '张华', action: 'approve', action_time: '2025-03-25 10:00', opinion: '材料已补齐，通过' },
      ],
      fields: { event_name: '华东春季产品发布会', venue: '上海国际会议中心', attendees: '320人' },
    },
    {
      process_id: 'WF-2025-025',
      title: '外包开发合同签署 - CRM系统二期',
      applicant: '赵丽',
      department: '研发部',
      process_type: '合同审批',
      amount: 680000,
      submit_time: '2025-02-10 14:00',
      archive_time: '2025-03-15 11:00',
      status: 'archived',
      flow_nodes: [
        { node_id: 'N1', node_name: '部门经理审批', approver: '张明', action: 'approve', action_time: '2025-02-11 09:00', opinion: '技术方案可行' },
        { node_id: 'N2', node_name: '法务审核', approver: '孙律', action: 'return', action_time: '2025-02-15 16:00', opinion: '知识产权条款需修改' },
        { node_id: 'N3', node_name: '法务审核（重审）', approver: '孙律', action: 'approve', action_time: '2025-02-20 11:00', opinion: '条款已修正，通过' },
        { node_id: 'N4', node_name: '财务总监审批', approver: '张华', action: 'approve', action_time: '2025-02-22 14:30', opinion: '预算范围内' },
        { node_id: 'N5', node_name: '总经理审批', approver: '刘总', action: 'approve', action_time: '2025-02-25 10:00', opinion: '批准' },
      ],
      fields: { vendor: 'YY软件科技', contract_period: '2025-03-01 至 2025-08-31', deliverables: 'CRM系统二期全部功能模块' },
    },
    {
      process_id: 'WF-2025-012',
      title: '新员工批量入职审批 - 2025春招',
      applicant: '赵丽',
      department: '人力资源部',
      process_type: '人事审批',
      submit_time: '2025-01-20 09:00',
      archive_time: '2025-02-28 17:00',
      status: 'archived',
      flow_nodes: [
        { node_id: 'N1', node_name: 'HR经理审批', approver: '赵丽', action: 'approve', action_time: '2025-01-20 10:00', opinion: '招聘计划内' },
        { node_id: 'N2', node_name: '用人部门确认', approver: '各部门经理', action: 'approve', action_time: '2025-01-25 16:00', opinion: '确认接收' },
        { node_id: 'N3', node_name: 'HR总监审批', approver: '王总监', action: 'approve', action_time: '2025-01-28 09:30', opinion: '同意' },
      ],
      fields: { headcount: '15人', positions: '开发工程师x8, 产品经理x3, 测试工程师x4', onboard_date: '2025-03-01' },
    },
    {
      process_id: 'WF-2025-008',
      title: '办公楼层装修改造工程',
      applicant: '刘洋',
      department: '行政部',
      process_type: '工程审批',
      amount: 450000,
      submit_time: '2025-01-10 10:00',
      archive_time: '2025-02-15 14:00',
      status: 'archived',
      flow_nodes: [
        { node_id: 'N1', node_name: '行政经理审批', approver: '刘洋', action: 'approve', action_time: '2025-01-10 14:00', opinion: '方案合理' },
        { node_id: 'N2', node_name: '财务总监审批', approver: '张华', action: 'approve', action_time: '2025-01-12 10:00', opinion: '预算可控' },
        { node_id: 'N3', node_name: '总经理审批', approver: '刘总', action: 'return', action_time: '2025-01-15 09:00', opinion: '施工时间与业务高峰冲突，请调整' },
        { node_id: 'N4', node_name: '行政经理重新提交', approver: '刘洋', action: 'approve', action_time: '2025-01-18 11:00', opinion: '已调整至春节假期施工' },
        { node_id: 'N5', node_name: '总经理审批（重审）', approver: '刘总', action: 'approve', action_time: '2025-01-20 09:30', opinion: '时间调整合理，批准' },
      ],
      fields: { floor: '3楼、5楼', contractor: 'ZZ装饰工程公司', construction_period: '2025-01-25 至 2025-02-10' },
    },
    {
      process_id: 'WF-2024-095',
      title: '年终奖金发放审批',
      applicant: '赵丽',
      department: '人力资源部',
      process_type: '人事审批',
      amount: 2800000,
      submit_time: '2024-12-15 09:00',
      archive_time: '2025-01-10 17:00',
      status: 'archived',
      flow_nodes: [
        { node_id: 'N1', node_name: 'HR经理审批', approver: '赵丽', action: 'approve', action_time: '2024-12-15 10:00', opinion: '方案符合公司制度' },
        { node_id: 'N2', node_name: '财务总监审批', approver: '张华', action: 'approve', action_time: '2024-12-18 14:00', opinion: '预算充足，同意发放' },
        { node_id: 'N3', node_name: '总经理审批', approver: '刘总', action: 'approve', action_time: '2024-12-20 09:30', opinion: '批准' },
      ],
      fields: { total_headcount: '128人', avg_bonus: '¥21,875', bonus_pool: '¥2,800,000' },
    },
    {
      process_id: 'WF-2024-088',
      title: '客户管理系统（CRM）一期验收',
      applicant: '张明',
      department: '研发部',
      process_type: '项目审批',
      amount: 350000,
      submit_time: '2024-11-20 10:00',
      archive_time: '2024-12-25 16:00',
      status: 'archived',
      flow_nodes: [
        { node_id: 'N1', node_name: '项目经理确认', approver: '张明', action: 'approve', action_time: '2024-11-20 14:00', opinion: '功能验收通过' },
        { node_id: 'N2', node_name: '测试负责人确认', approver: '周磊', action: 'approve', action_time: '2024-11-25 11:00', opinion: '测试用例全部通过' },
        { node_id: 'N3', node_name: '业务方验收', approver: '李芳', action: 'return', action_time: '2024-12-01 15:00', opinion: '报表导出功能需优化' },
        { node_id: 'N4', node_name: '业务方验收（重审）', approver: '李芳', action: 'approve', action_time: '2024-12-15 10:00', opinion: '问题已修复，验收通过' },
      ],
      fields: { project_name: 'CRM一期', vendor: 'YY软件科技', modules: '客户管理、商机跟踪、报表分析' },
    },
    {
      process_id: 'WF-2024-075',
      title: '全员体检服务采购',
      applicant: '刘洋',
      department: '行政部',
      process_type: '采购审批',
      amount: 192000,
      submit_time: '2024-10-10 09:00',
      archive_time: '2024-11-15 14:00',
      status: 'archived',
      flow_nodes: [
        { node_id: 'N1', node_name: '行政经理审批', approver: '刘洋', action: 'approve', action_time: '2024-10-10 11:00', opinion: '年度福利计划内' },
        { node_id: 'N2', node_name: '财务审核', approver: '张华', action: 'approve', action_time: '2024-10-12 10:00', opinion: '费用合理' },
        { node_id: 'N3', node_name: '总经理审批', approver: '刘总', action: 'approve', action_time: '2024-10-15 09:00', opinion: '同意' },
      ],
      fields: { provider: 'XX健康管理中心', headcount: '128人', package: '高管套餐+标准套餐' },
    },
    {
      process_id: 'WF-2024-062',
      title: '双十一营销活动预算',
      applicant: '陈伟',
      department: '市场部',
      process_type: '预算审批',
      amount: 500000,
      submit_time: '2024-09-15 14:00',
      archive_time: '2024-10-20 17:00',
      status: 'archived',
      flow_nodes: [
        { node_id: 'N1', node_name: '市场总监审批', approver: '周磊', action: 'approve', action_time: '2024-09-16 10:00', opinion: '方案可行' },
        { node_id: 'N2', node_name: '财务总监审批', approver: '张华', action: 'return', action_time: '2024-09-18 15:00', opinion: '线下活动预算偏高，建议缩减' },
        { node_id: 'N3', node_name: '财务总监审批（重审）', approver: '张华', action: 'approve', action_time: '2024-09-22 11:00', opinion: '调整后预算合理' },
        { node_id: 'N4', node_name: '总经理审批', approver: '刘总', action: 'approve', action_time: '2024-09-25 09:00', opinion: '批准执行' },
      ],
      fields: { campaign: '双十一全渠道营销', channels: '线上广告、直播、线下展会', roi_target: '1:5' },
    },
    {
      process_id: 'WF-2024-051',
      title: '销售部门扩编申请',
      applicant: '李芳',
      department: '销售部',
      process_type: '人事审批',
      submit_time: '2024-08-20 09:30',
      archive_time: '2024-09-30 16:00',
      status: 'archived',
      flow_nodes: [
        { node_id: 'N1', node_name: '销售总监审批', approver: '李芳', action: 'approve', action_time: '2024-08-20 14:00', opinion: '业务增长需要' },
        { node_id: 'N2', node_name: 'HR总监审批', approver: '王总监', action: 'approve', action_time: '2024-08-22 10:00', opinion: 'HC计划内' },
        { node_id: 'N3', node_name: '总经理审批', approver: '刘总', action: 'approve', action_time: '2024-08-25 09:00', opinion: '同意扩编' },
      ],
      fields: { new_headcount: '6人', positions: '大客户经理x2, 区域销售x4', budget_impact: '年增人力成本约¥720,000' },
    },
  ]

  //全流程合规复审结果
  const mockArchiveAuditResult: ArchiveAuditResult = {
    trace_id: 'ATR-20250610-X1Y2',
    process_id: 'WF-2025-050',
    overall_compliance: 'compliant',
    overall_score: 92,
    duration_ms: 3200,
    flow_audit: {
      is_complete: true,
      missing_nodes: [],
      node_results: [
        { node_id: 'N1', node_name: '部门经理审批', compliant: true, reasoning: '审批权限匹配，审批时效正常（1个工作日内）' },
        { node_id: 'N2', node_name: '财务总监审批', compliant: true, reasoning: '金额超100万需财务总监审批，流程正确' },
        { node_id: 'N3', node_name: '总经理审批', compliant: true, reasoning: '金额超50万需总经理审批，流程正确' },
      ],
    },
    field_audit: [
      { field_name: '供应商资质', passed: true, reasoning: '供应商在合格名录中，资质有效期至2026-12-31' },
      { field_name: '合同编号', passed: true, reasoning: '合同编号格式正确，已在合同管理系统中登记' },
      { field_name: '交付日期', passed: true, reasoning: '交付日期在合同约定范围内' },
      { field_name: '付款条件', passed: false, reasoning: '分期付款比例（30%/40%/30%）与公司标准（30%/30%/40%）不一致，需确认是否有特批' },
    ],
    rule_audit: [
      { rule_id: 'R001', rule_name: '预算额度校验', passed: true, reasoning: '采购金额在年度IT预算范围内' },
      { rule_id: 'R002', rule_name: '供应商比价', passed: true, reasoning: '已提供3家供应商比价报告' },
      { rule_id: 'R004', rule_name: '合同条款完整性', passed: true, reasoning: '合同包含所有必要条款' },
    ],
    ai_summary: '该采购流程整体合规，审批链完整，各节点审批权限匹配。\n\n主要发现：\n1. 审批流程完整，三级审批均在合理时效内完成\n2. 供应商资质有效，比价流程规范\n3. 付款条件与公司标准略有差异（分期比例不同），建议确认是否有特批记录\n\n合规评级：基本合规（92分），建议关注付款条件差异项。',
  }

  // ============================================================
  //仪表板“归档”选项卡的存档流程
  //这些是具有多轮审核链的完整流程（最终结果 = 批准）
  // ============================================================
  const mockArchivedOAProcesses: OAProcess[] = [
    { process_id: 'WF-2025-050', title: '2025年度服务器集群采购', applicant: '王强', department: 'IT部', submit_time: '2025-04-15 09:00', process_type: '采购审批', status: 'archived', current_node: '已归档', amount: 1200000, urgency: 'high' },
    { process_id: 'WF-2025-038', title: '华东区域市场推广费用报销', applicant: '陈伟', department: '市场部', submit_time: '2025-03-20 11:00', process_type: '费用报销', status: 'archived', current_node: '已归档', amount: 85000, urgency: 'medium' },
    { process_id: 'WF-2025-025', title: '外包开发合同签署 - CRM系统二期', applicant: '赵丽', department: '研发部', submit_time: '2025-02-10 14:00', process_type: '合同审批', status: 'archived', current_node: '已归档', amount: 680000, urgency: 'high' },
    { process_id: 'WF-2025-012', title: '新员工批量入职审批 - 2025春招', applicant: '赵丽', department: '人力资源部', submit_time: '2025-01-20 09:00', process_type: '人事审批', status: 'archived', current_node: '已归档', urgency: 'low' },
    { process_id: 'WF-2025-008', title: '办公楼层装修改造工程', applicant: '刘洋', department: '行政部', submit_time: '2025-01-10 10:00', process_type: '工程审批', status: 'archived', current_node: '已归档', amount: 450000, urgency: 'medium' },
  ]

  //存档流程的多轮审核链快照（最后一轮始终批准）
  const mockArchivedAuditChains: Record<string, AuditSnapshot[]> = {
    'WF-2025-050': [
      { snapshot_id: 'SN-A001', process_id: 'WF-2025-050', title: '2025年度服务器集群采购', applicant: '王强', department: 'IT部', recommendation: 'return', score: 68, created_at: '2025-04-16 10:30', adopted: true },
      { snapshot_id: 'SN-A002', process_id: 'WF-2025-050', title: '2025年度服务器集群采购', applicant: '王强', department: 'IT部', recommendation: 'return', score: 82, created_at: '2025-04-25 14:00', adopted: true },
      { snapshot_id: 'SN-A003', process_id: 'WF-2025-050', title: '2025年度服务器集群采购', applicant: '王强', department: 'IT部', recommendation: 'approve', score: 95, created_at: '2025-05-10 09:15', adopted: true },
    ],
    'WF-2025-038': [
      { snapshot_id: 'SN-A004', process_id: 'WF-2025-038', title: '华东区域市场推广费用报销', applicant: '陈伟', department: '市场部', recommendation: 'return', score: 42, created_at: '2025-03-22 15:00', adopted: true },
      { snapshot_id: 'SN-A005', process_id: 'WF-2025-038', title: '华东区域市场推广费用报销', applicant: '陈伟', department: '市场部', recommendation: 'approve', score: 90, created_at: '2025-03-28 11:30', adopted: true },
    ],
    'WF-2025-025': [
      { snapshot_id: 'SN-A006', process_id: 'WF-2025-025', title: '外包开发合同签署 - CRM系统二期', applicant: '赵丽', department: '研发部', recommendation: 'return', score: 55, created_at: '2025-02-12 10:00', adopted: true },
      { snapshot_id: 'SN-A007', process_id: 'WF-2025-025', title: '外包开发合同签署 - CRM系统二期', applicant: '赵丽', department: '研发部', recommendation: 'return', score: 78, created_at: '2025-02-18 16:00', adopted: true },
      { snapshot_id: 'SN-A008', process_id: 'WF-2025-025', title: '外包开发合同签署 - CRM系统二期', applicant: '赵丽', department: '研发部', recommendation: 'approve', score: 92, created_at: '2025-02-24 09:30', adopted: true },
    ],
    'WF-2025-012': [
      { snapshot_id: 'SN-A009', process_id: 'WF-2025-012', title: '新员工批量入职审批 - 2025春招', applicant: '赵丽', department: '人力资源部', recommendation: 'approve', score: 96, created_at: '2025-01-22 11:00', adopted: true },
    ],
    'WF-2025-008': [
      { snapshot_id: 'SN-A010', process_id: 'WF-2025-008', title: '办公楼层装修改造工程', applicant: '刘洋', department: '行政部', recommendation: 'return', score: 38, created_at: '2025-01-12 14:00', adopted: true },
      { snapshot_id: 'SN-A011', process_id: 'WF-2025-008', title: '办公楼层装修改造工程', applicant: '刘洋', department: '行政部', recommendation: 'return', score: 71, created_at: '2025-01-16 10:30', adopted: true },
      { snapshot_id: 'SN-A012', process_id: 'WF-2025-008', title: '办公楼层装修改造工程', applicant: '刘洋', department: '行政部', recommendation: 'approve', score: 89, created_at: '2025-01-19 15:00', adopted: true },
    ],
  }

  //存档流程的历史审核结果（最终批准结果）
  const mockArchivedHistoricalResults: Record<string, AuditResult> = {
    'WF-2025-050': {
      trace_id: 'TR-20250510-M1N2', process_id: 'WF-2025-050', recommendation: 'approve', score: 95, duration_ms: 2100,
      details: [
        { rule_id: 'R001', rule_name: '预算额度校验', passed: true, reasoning: '采购金额在年度IT预算范围内', is_locked: true },
        { rule_id: 'R002', rule_name: '供应商比价', passed: true, reasoning: '已提供3家供应商比价报告' },
        { rule_id: 'R003', rule_name: '合同条款完整性', passed: true, reasoning: '合同包含所有必要条款' },
      ],
      ai_reasoning: '该采购申请经过两轮修改后完全合规，预算合理、审批链完整、供应商资质齐全。建议通过。',
    },
    'WF-2025-038': {
      trace_id: 'TR-20250328-P3Q4', process_id: 'WF-2025-038', recommendation: 'approve', score: 90, duration_ms: 1650,
      details: [
        { rule_id: 'R003', rule_name: '费用标准校验', passed: true, reasoning: '报销金额符合市场推广费用标准' },
        { rule_id: 'R006', rule_name: '审批材料完整性', passed: true, reasoning: '发票、活动方案、参会名单齐全' },
      ],
      ai_reasoning: '费用报销申请材料已补齐，金额合理，符合公司市场推广费用管理制度。建议通过。',
    },
    'WF-2025-025': {
      trace_id: 'TR-20250224-R5S6', process_id: 'WF-2025-025', recommendation: 'approve', score: 92, duration_ms: 1880,
      details: [
        { rule_id: 'R004', rule_name: '合同法务审核', passed: true, reasoning: '知识产权条款已修正，法务已会签', is_locked: true },
        { rule_id: 'R008', rule_name: '供应商准入', passed: true, reasoning: '外包供应商已通过准入评审' },
        { rule_id: 'R009', rule_name: '预算审批', passed: true, reasoning: '合同金额已纳入年度预算' },
      ],
      ai_reasoning: '合同经过法务条款修正后合规，供应商资质齐全，预算在规划范围内。建议通过。',
    },
    'WF-2025-012': {
      trace_id: 'TR-20250122-T7U8', process_id: 'WF-2025-012', recommendation: 'approve', score: 96, duration_ms: 1200,
      details: [
        { rule_id: 'R005', rule_name: 'HC审批校验', passed: true, reasoning: '招聘人数在年度HC计划内' },
        { rule_id: 'R010', rule_name: '用人部门确认', passed: true, reasoning: '各部门经理已确认接收' },
      ],
      ai_reasoning: '批量入职审批完全合规，招聘计划内，各部门已确认。建议通过。',
    },
    'WF-2025-008': {
      trace_id: 'TR-20250119-V9W0', process_id: 'WF-2025-008', recommendation: 'approve', score: 89, duration_ms: 1750,
      details: [
        { rule_id: 'R001', rule_name: '预算额度校验', passed: true, reasoning: '装修费用在行政预算范围内' },
        { rule_id: 'R011', rule_name: '施工时间合理性', passed: true, reasoning: '已调整至春节假期施工，不影响业务' },
        { rule_id: 'R012', rule_name: '承包商资质', passed: true, reasoning: '承包商具备相应施工资质' },
      ],
      ai_reasoning: '装修方案经调整后合规，施工时间不影响业务运营，预算可控。建议通过。',
    },
  }




  const mockOverviewData: OverviewDashboardData = {
    auditSummary: { approved: mockApprovedProcesses.length, returned: mockReturnedProcesses.length, archived: mockArchivedOAProcesses.length, total: mockApprovedProcesses.length + mockReturnedProcesses.length + mockArchivedOAProcesses.length },
    pendingCount: mockProcesses.length,
    weeklyTrend: [
      { date: '06-04', count: 35 }, { date: '06-05', count: 41 },
      { date: '06-06', count: 38 }, { date: '06-07', count: 22 },
      { date: '06-08', count: 15 }, { date: '06-09', count: 44 },
      { date: '06-10', count: 42 },
    ],
    deptDistribution: [
      { department: '研发部', count: 12, color: '#4f46e5' },
      { department: '销售部', count: 9, color: '#06b6d4' },
      { department: '市场部', count: 7, color: '#f59e0b' },
      { department: 'IT部', count: 6, color: '#10b981' },
      { department: '人力资源部', count: 4, color: '#ef4444' },
      { department: '行政部', count: 3, color: '#8b5cf6' },
      { department: '财务部', count: 1, color: '#ec4899' },
    ],
    recentActivity: [
      { id: 'RA-001', action: 'AI 审核完成', target: '办公设备采购申请', user: '张明', time: '09:35', type: 'audit' },
      { id: 'RA-002', action: '手动通过', target: '年度IT设备采购', user: '王强', time: '09:20', type: 'audit' },
      { id: 'RA-003', action: '批量审核执行', target: '12 条流程', user: '系统', time: '09:05', type: 'cron' },
      { id: 'RA-004', action: '规则配置更新', target: '采购审批规则', user: '赵伟', time: '08:50', type: 'config' },
      { id: 'RA-005', action: 'AI 审核完成', target: '差旅费报销', user: '李芳', time: '08:30', type: 'audit' },
      { id: 'RA-006', action: '日报推送成功', target: '全员日报', user: '系统', time: '08:00', type: 'cron' },
      { id: 'RA-007', action: '新用户加入', target: '刘洋（行政部）', user: '赵伟', time: '昨天 17:30', type: 'system' },
      { id: 'RA-008', action: '合规复核', target: '服务器集群采购', user: '张华', time: '昨天 16:00', type: 'audit' },
    ],
    aiPerformance: {
      avgResponseMs: 1850, successRate: 99.2, totalCalls: 1247,
      dailyStats: [
        { date: '06-04', avgMs: 1920, calls: 35 }, { date: '06-05', avgMs: 1780, calls: 41 },
        { date: '06-06', avgMs: 1850, calls: 38 }, { date: '06-07', avgMs: 2100, calls: 22 },
        { date: '06-08', avgMs: 1650, calls: 15 }, { date: '06-09', avgMs: 1900, calls: 44 },
        { date: '06-10', avgMs: 1850, calls: 42 },
      ],
    },
    tenantUsage: { tokenUsed: 284500, tokenQuota: 500000, storageUsedMB: 1240, storageQuotaMB: 5120, activeUsers: 18, totalUsers: 25 },
    userActivity: [
      { username: 'zhangming', displayName: '张明', department: '研发部', auditCount: 156, lastActive: '2025-06-10 09:35' },
      { username: 'wangqiang', displayName: '王强', department: 'IT部', auditCount: 132, lastActive: '2025-06-10 09:20' },
      { username: 'lifang', displayName: '李芳', department: '销售部', auditCount: 98, lastActive: '2025-06-10 08:30' },
      { username: 'zhaoli', displayName: '赵丽', department: '人力资源部', auditCount: 87, lastActive: '2025-06-09 17:00' },
      { username: 'chenwei', displayName: '陈伟', department: '市场部', auditCount: 76, lastActive: '2025-06-09 16:00' },
    ],
    systemHealth: [
      { service: 'Go 业务中台', status: 'healthy', cpu: 23, memory: 45, uptime: '15d 8h' },
      { service: 'Python AI 引擎', status: 'healthy', cpu: 58, memory: 72, uptime: '15d 8h' },
      { service: 'PostgreSQL', status: 'healthy', cpu: 12, memory: 38, uptime: '30d 2h' },
      { service: 'Nuxt 前端', status: 'healthy', cpu: 5, memory: 22, uptime: '15d 8h' },
      { service: 'OA 数据同步', status: 'degraded', cpu: 8, memory: 15, uptime: '3d 12h' },
    ],
    tenantOverview: [
      { tenantId: 'T-001', tenantName: '默认租户', userCount: 25, auditCount: 1247, tokenUsed: 284500, status: 'active' },
      { tenantId: 'T-002', tenantName: '华东分公司', userCount: 18, auditCount: 856, tokenUsed: 195000, status: 'active' },
      { tenantId: 'T-003', tenantName: '华南分公司', userCount: 12, auditCount: 423, tokenUsed: 98000, status: 'active' },
      { tenantId: 'T-004', tenantName: '测试租户', userCount: 3, auditCount: 56, tokenUsed: 12000, status: 'suspended' },
    ],
    apiMetrics: [
      { endpoint: '/api/audit/execute', calls: 1247, avgMs: 1850, successRate: 99.2 },
      { endpoint: '/api/audit/todo', calls: 3560, avgMs: 120, successRate: 99.8 },
      { endpoint: '/api/auth/login', calls: 892, avgMs: 85, successRate: 98.5 },
      { endpoint: '/api/cron/execute', calls: 156, avgMs: 5200, successRate: 97.4 },
      { endpoint: '/api/archive/review', calls: 234, avgMs: 2100, successRate: 99.1 },
    ],
    monitorMetrics: {
      apiSuccessRate: 99.2,
      avgModelResponseMs: 1250,
      p95Latency: 2100,
      totalRequests24h: 1847,
      activeTenants: 3,
      uptime: '99.97%',
    },
    monitorAlerts: [
      { id: 1, level: 'warning', messageZh: '租户"华东分公司" Token 用量已达 70%', messageEn: 'Tenant "East Division" token usage reached 70%', timeZh: '10 分钟前', timeEn: '10 min ago' },
      { id: 2, level: 'info', messageZh: '系统自动完成每日数据备份', messageEn: 'Daily data backup completed', timeZh: '2 小时前', timeEn: '2 hours ago' },
      { id: 3, level: 'info', messageZh: 'AI 模型响应时间恢复正常', messageEn: 'AI model response time recovered', timeZh: '5 小时前', timeEn: '5 hours ago' },
    ],
  }

  /**
   * 全局进程类别→进程名称映射。
   * - 类别（流程类型/类别）：例如"采购类", "费用类" — 相关流程组
   * - processName（流程名称）：例如"采购流程图", "费用报销" — 具体流程定义
   * - 标题（流程标题）：例如“办公设备采购申请”——流程的具体实例
   *
   * 这是构建级联/多级流程选择器的唯一事实来源。*/
  const processCategoryMap = mockProcessAuditConfigs.map(cfg => ({
    category: cfg.process_type_label || cfg.process_type,
    processName: cfg.process_type,
  }))

  //去重并构建级联选项结构：
  //[{ label: '采购类', value: '采购类', kids: [{ label: '采购类', value: '采购类' }] }]
  const buildProcessCascaderOptions = (configs: ProcessAuditConfig[]) => {
    const categoryMap = new Map<string, { label: string; value: string; children: { label: string; value: string }[] }>()
    for (const cfg of configs) {
      const catLabel = cfg.process_type_label || cfg.process_type
      if (!categoryMap.has(catLabel)) {
        categoryMap.set(catLabel, { label: catLabel, value: catLabel, children: [] })
      }
      const cat = categoryMap.get(catLabel)!
      if (!cat.children.some(c => c.value === cfg.process_type)) {
        cat.children.push({ label: cfg.process_type, value: cfg.process_type })
      }
    }
    return Array.from(categoryMap.values())
  }

  const processCascaderOptions = buildProcessCascaderOptions(mockProcessAuditConfigs)

  //还为存档审查配置构建级联选项
  const archiveProcessCascaderOptions = buildProcessCascaderOptions(
    mockArchiveReviewConfigs.map(c => ({
      ...c,
      kb_mode: c.kb_mode,
      user_permissions: { allow_custom_fields: c.user_permissions.allow_custom_fields, allow_custom_rules: c.user_permissions.allow_custom_rules, allow_modify_strictness: c.user_permissions.allow_modify_strictness },
    } as any))
  )

  /** 每个用户的默认仪表板首选项（由用户名键入）*/
  const mockUserDashboardPrefs: Record<string, UserDashboardPrefs> = {
    zhangming: { enabledWidgets: ['audit_summary', 'pending_tasks', 'weekly_trend', 'cron_tasks', 'archive_review', 'recent_activity'] },
    tenantadmin: { enabledWidgets: ['dept_distribution', 'recent_activity', 'ai_performance', 'tenant_usage', 'user_activity'] },
    admin: { enabledWidgets: ['monitor_metrics', 'recent_activity', 'system_health', 'tenant_overview', 'api_metrics', 'monitor_alerts'] },
  }

  return {
    mockProcesses,
    mockApprovedProcesses,
    mockReturnedProcesses,
    mockHistoricalResults,
    mockAuditResult,
    mockCronTasks,
    mockCronTaskTypeConfigs,
    mockSnapshots,
    mockTenants,
    mockRules,
    mockDashboardStats,
    mockOverviewData,
    mockUserDashboardPrefs,
    mockArchivedProcesses,
    mockArchiveAuditResult,
    mockArchivedOAProcesses,
    mockArchivedAuditChains,
    mockArchivedHistoricalResults,
    mockProcessAuditConfigs: [...mockProcessAuditConfigs],
    mockArchiveReviewConfigs: [...mockArchiveReviewConfigs],
    mockAuditLogs: [...mockAuditLogs],
    mockCronLogs: [...mockCronLogs],
    mockArchiveLogs: [...mockArchiveLogs],
    mockUserPersonalConfigs: [...mockUserPersonalConfigs],
    mockOASystemConfigs: [...mockOASystemConfigs],
    mockOADatabaseConnections: [...mockOADatabaseConnections],
    mockAIModelConfigs: [...mockAIModelConfigs],
    mockSystemGeneralConfig: { ...mockSystemGeneralConfig },
    mockBatchAuditResult,
    mockTodoAuditResults,
    processCategoryMap,
    processCascaderOptions,
    archiveProcessCascaderOptions,
  }
}


