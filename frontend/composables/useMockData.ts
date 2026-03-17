/**
 * 用于开发的模拟数据 - 模拟 API 响应
 * 所有模拟/虚拟数据都存放在这里。业务代码只引用该文件。*/
import type { PermissionGroup } from "~/types/auth";


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
  current_node: string
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


export interface DetailTable {
  table_name: string
  table_label: string
  fields: ProcessField[]
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

export const mockProcessAuditConfigs: ProcessAuditConfig[] = []

// ============================================================
//存档审核配置 (归档复盘配置 - 机场管理)
// ============================================================
export const mockArchiveReviewConfigs: ArchiveReviewConfig[] = []

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
  const mockCronTaskTypeConfigs: CronTaskTypeConfig[] = []

  const mockSnapshots: AuditSnapshot[] = [
    { snapshot_id: 'SN-001', process_id: 'WF-2025-098', title: '年度IT设备采购', applicant: '王强', department: 'IT部', recommendation: 'approve', score: 95, created_at: '2025-06-09 16:30', adopted: true },
    { snapshot_id: 'SN-002', process_id: 'WF-2025-097', title: '客户招待费报销', applicant: '李芳', department: '销售部', recommendation: 'return', score: 35, created_at: '2025-06-09 15:20', adopted: true },
    { snapshot_id: 'SN-003', process_id: 'WF-2025-096', title: '新产品研发立项', applicant: '张明', department: '研发部', recommendation: 'approve', score: 88, created_at: '2025-06-09 14:10', adopted: true },
    { snapshot_id: 'SN-004', process_id: 'WF-2025-095', title: '办公用品批量采购', applicant: '刘洋', department: '行政部', recommendation: 'return', score: 62, created_at: '2025-06-09 11:45', adopted: false },
    { snapshot_id: 'SN-005', process_id: 'WF-2025-094', title: '员工培训费用申请', applicant: '赵丽', department: '人力资源部', recommendation: 'approve', score: 91, created_at: '2025-06-08 17:00', adopted: true },
    { snapshot_id: 'SN-006', process_id: 'WF-2025-093', title: '广告投放合同签署', applicant: '陈伟', department: '市场部', recommendation: 'return', score: 58, created_at: '2025-06-08 14:30', adopted: null },
  ]


  // ============================================================
  //系统设置模拟数据（系统设置）
  // ============================================================


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
    { process_id: 'WF-2025-098', title: '年度IT设备采购', applicant: '王强', department: 'IT部', submit_time: '2025-06-09 16:30', process_type: '采购审批', status: 'approved', current_node: '已完成' },
    { process_id: 'WF-2025-096', title: '新产品研发立项', applicant: '张明', department: '研发部', submit_time: '2025-06-09 14:10', process_type: '项目审批', status: 'approved', current_node: '已完成' },
    { process_id: 'WF-2025-094', title: '员工培训费用申请', applicant: '赵丽', department: '人力资源部', submit_time: '2025-06-08 17:00', process_type: '费用报销', status: 'approved', current_node: '已完成' },
    { process_id: 'WF-2025-090', title: '办公家具批量采购', applicant: '刘洋', department: '行政部', submit_time: '2025-06-07 10:00', process_type: '采购审批', status: 'approved', current_node: '已完成' },
    { process_id: 'WF-2025-088', title: '年度广告投放合同', applicant: '陈伟', department: '市场部', submit_time: '2025-06-06 15:30', process_type: '合同审批', status: 'approved', current_node: '已完成' },
    { process_id: 'WF-2025-085', title: '销售团队季度奖金发放', applicant: '李芳', department: '销售部', submit_time: '2025-06-06 09:00', process_type: '费用报销', status: 'approved', current_node: '已完成' },
    { process_id: 'WF-2025-082', title: '网络安全设备采购', applicant: '王强', department: 'IT部', submit_time: '2025-06-05 14:20', process_type: '采购审批', status: 'approved', current_node: '已完成'},
    { process_id: 'WF-2025-079', title: '实习生转正审批（3人）', applicant: '赵丽', department: '人力资源部', submit_time: '2025-06-05 11:00', process_type: '人事审批', status: 'approved', current_node: '已完成' },
    { process_id: 'WF-2025-076', title: '会议室音视频系统升级', applicant: '刘洋', department: '行政部', submit_time: '2025-06-04 16:00', process_type: '工程审批', status: 'approved', current_node: '已完成'},
  ]

  //返回的进程 - 历史、只读
  const mockReturnedProcesses: OAProcess[] = [
    { process_id: 'WF-2025-097', title: '客户招待费报销', applicant: '李芳', department: '销售部', submit_time: '2025-06-09 15:20', process_type: '费用报销', status: 'returned', current_node: '已退回'},
    { process_id: 'WF-2025-091', title: '未经审批的外包合同', applicant: '陈伟', department: '市场部', submit_time: '2025-06-08 10:00', process_type: '合同审批', status: 'returned', current_node: '已退回' },
    { process_id: 'WF-2025-087', title: '超标准差旅费报销', applicant: '张明', department: '研发部', submit_time: '2025-06-07 09:30', process_type: '费用报销', status: 'returned', current_node: '已退回'},
    { process_id: 'WF-2025-083', title: '未备案供应商采购申请', applicant: '刘洋', department: '行政部', submit_time: '2025-06-06 11:00', process_type: '采购审批', status: 'returned', current_node: '已退回'},
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
    { process_id: 'WF-2025-050', title: '2025年度服务器集群采购', applicant: '王强', department: 'IT部', submit_time: '2025-04-15 09:00', process_type: '采购审批', status: 'archived', current_node: '已归档'},
    { process_id: 'WF-2025-038', title: '华东区域市场推广费用报销', applicant: '陈伟', department: '市场部', submit_time: '2025-03-20 11:00', process_type: '费用报销', status: 'archived', current_node: '已归档' },
    { process_id: 'WF-2025-025', title: '外包开发合同签署 - CRM系统二期', applicant: '赵丽', department: '研发部', submit_time: '2025-02-10 14:00', process_type: '合同审批', status: 'archived', current_node: '已归档' },
    { process_id: 'WF-2025-012', title: '新员工批量入职审批 - 2025春招', applicant: '赵丽', department: '人力资源部', submit_time: '2025-01-20 09:00', process_type: '人事审批', status: 'archived', current_node: '已归档'},
    { process_id: 'WF-2025-008', title: '办公楼层装修改造工程', applicant: '刘洋', department: '行政部', submit_time: '2025-01-10 10:00', process_type: '工程审批', status: 'archived', current_node: '已归档' },
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
    mockBatchAuditResult,
    mockTodoAuditResults,
    processCascaderOptions,
    archiveProcessCascaderOptions,
  }
}


