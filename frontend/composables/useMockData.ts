/**
 * Mock data for development - simulates API responses
 * All mock/virtual data lives here. Business code only references this file.
 */

// ============================================================
// Mock user accounts for login
// ============================================================
export type UserRole = 'business' | 'tenant_admin' | 'system_admin'

export interface MockUser {
  username: string
  password: string
  tenant_id: string
  role: UserRole
  display_name: string
}

export const MOCK_USERS: MockUser[] = [
  { username: 'zhangming', password: '123456', tenant_id: 'default', role: 'business', display_name: '张明' },
  { username: 'user', password: '123456', tenant_id: 'default', role: 'business', display_name: '测试用户' },
  { username: 'tenantadmin', password: '123456', tenant_id: 'default', role: 'tenant_admin', display_name: '租户管理员' },
  { username: 'admin', password: '123456', tenant_id: 'default', role: 'system_admin', display_name: '系统管理员' },
  { username: 'lifang', password: '123456', tenant_id: 'T-002', role: 'business', display_name: '李芳' },
]

// ============================================================
// Mock menus by role (RBAC)
// ============================================================
export interface MockMenuItem {
  key: string
  label: string
  icon?: string
  path: string
  children?: MockMenuItem[]
}

/**
 * Permission matrix: which roles can access which pages.
 * Used by middleware and layouts to control visibility.
 */
export const PAGE_PERMISSIONS: Record<string, UserRole[]> = {
  '/dashboard': ['business', 'tenant_admin', 'system_admin'],
  '/cron': ['business', 'tenant_admin', 'system_admin'],
  '/archive': ['business', 'tenant_admin', 'system_admin'],
  '/settings': ['business', 'tenant_admin', 'system_admin'],
  '/admin/tenant': ['tenant_admin', 'system_admin'],
  '/admin/tenant/org': ['tenant_admin', 'system_admin'],
  '/admin/tenant/ai': ['tenant_admin', 'system_admin'],
  '/admin/tenant/kb': ['tenant_admin', 'system_admin'],
  '/admin/system': ['system_admin'],
  '/admin/monitor': ['system_admin'],
  '/admin/system/settings': ['system_admin'],
}

export function hasPagePermission(path: string, role: UserRole): boolean {
  const allowed = PAGE_PERMISSIONS[path]
  if (!allowed) return true // pages not in the map are accessible by all
  return allowed.includes(role)
}

export function getMockMenusByRole(role: UserRole): MockMenuItem[] {
  const base: MockMenuItem[] = [
    { key: 'dashboard', label: '审核工作台', path: '/dashboard' },
    { key: 'cron', label: '定时任务', path: '/cron' },
    { key: 'archive', label: '归档复盘', path: '/archive' },
  ]
  const tenant: MockMenuItem[] = [
    { key: 'tenant', label: '规则配置', path: '/admin/tenant' },
  ]
  const sys: MockMenuItem[] = [
    { key: 'system', label: '系统管理', path: '/admin/system' },
    { key: 'monitor', label: '全局监控', path: '/admin/monitor' },
  ]
  if (role === 'system_admin') return [...base, ...tenant, ...sys]
  if (role === 'tenant_admin') return [...base, ...tenant]
  return base
}

// ============================================================
// Business mock data
// ============================================================
export interface OAProcess {
  process_id: string
  title: string
  applicant: string
  department: string
  submit_time: string
  process_type: string
  status: string
  amount?: number
  urgency: 'high' | 'medium' | 'low'
}

export interface ChecklistResult {
  rule_id: string
  rule_name: string
  passed: boolean
  reasoning: string
  is_locked?: boolean
}

export interface AuditResult {
  trace_id: string
  process_id: string
  recommendation: 'approve' | 'reject' | 'revise'
  score: number
  details: ChecklistResult[]
  ai_reasoning: string
  duration_ms: number
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

export interface TenantInfo {
  id: string
  name: string
  oa_type: string
  token_quota: number
  token_used: number
  max_concurrency: number
  status: 'active' | 'inactive'
  created_at: string
}

export interface AuditRule {
  id: string
  process_type: string
  rule_content: string
  rule_scope: 'mandatory' | 'default_on' | 'default_off'
  priority: number
  enabled: boolean
}

export interface FlowNode {
  node_id: string
  node_name: string
  approver: string
  action: 'approve' | 'reject' | 'revise'
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
  todayRejected: number
  todayRevised: number
  pendingCount: number
  avgResponseMs: number
  successRate: number
  weeklyTrend: { date: string; count: number }[]
}

// ============================================================
// Archive Review types (归档复盘 - 全流程合规复核)
// ============================================================
// FlowNode, ArchivedProcess, FlowNodeAuditResult, ArchiveAuditResult
// are defined above in the Business mock data section.

export interface FieldAuditResult {
  field_name: string
  passed: boolean
  reasoning: string
}

// ============================================================
// Process-centric audit config types (审核工作台配置)
// ============================================================
export interface ProcessField {
  field_key: string
  field_name: string
  field_type: 'text' | 'number' | 'date' | 'select' | 'textarea' | 'file'
  selected: boolean
}

export interface ProcessAIConfig {
  ai_provider: string
  model_name: string
  audit_strictness: 'strict' | 'standard' | 'loose'
  system_prompt: string
  context_window: number
  temperature: number
}

export interface UserPermissions {
  allow_custom_fields: boolean
  allow_custom_rules: boolean
  allow_modify_strictness: boolean
}

export interface ProcessAuditConfig {
  id: string
  process_type: string
  flow_path: string
  fields: ProcessField[]
  field_mode: 'all' | 'selected'
  rules: (AuditRule & { source: 'manual' | 'file_import' })[]
  kb_mode: 'rules_only' | 'rag_only' | 'hybrid'
  ai_config: ProcessAIConfig
  user_permissions: UserPermissions
}

export const mockProcessAuditConfigs: ProcessAuditConfig[] = [
  {
    id: 'PAC-001',
    process_type: '采购审批',
    flow_path: '部门经理 → 财务总监 → 总经理',
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
      { id: 'R001', process_type: '采购审批', rule_content: '单笔采购金额不得超过部门季度预算上限', rule_scope: 'mandatory', priority: 100, enabled: true, source: 'manual' },
      { id: 'R002', process_type: '采购审批', rule_content: '超过10万元需提供至少3家供应商比价', rule_scope: 'mandatory', priority: 95, enabled: true, source: 'manual' },
      { id: 'R013', process_type: '采购审批', rule_content: '供应商须在合格供应商名录中', rule_scope: 'default_on', priority: 85, enabled: true, source: 'file_import' },
      { id: 'R014', process_type: '采购审批', rule_content: '合同条款须包含付款条件、交付时间、售后条款', rule_scope: 'default_on', priority: 80, enabled: true, source: 'manual' },
    ],
    kb_mode: 'rules_only',
    ai_config: {
      ai_provider: '本地部署',
      model_name: 'Qwen2.5-72B',
      audit_strictness: 'standard',
      system_prompt: '你是一个专业的采购审核助手。请根据以下规则对采购申请进行合规性审核，逐条检查并给出判断理由。对于不合规项，请明确指出问题并给出修改建议。',
      context_window: 8192,
      temperature: 0.3,
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
    flow_path: '部门经理 → 财务审核',
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
      { id: 'R003', process_type: '费用报销', rule_content: '单次报销金额超过5000元需附发票原件', rule_scope: 'default_on', priority: 80, enabled: true, source: 'manual' },
      { id: 'R006', process_type: '费用报销', rule_content: '差旅住宿标准不超过城市限额', rule_scope: 'default_off', priority: 60, enabled: false, source: 'manual' },
      { id: 'R015', process_type: '费用报销', rule_content: '发票日期须在报销申请日期前90天内', rule_scope: 'mandatory', priority: 90, enabled: true, source: 'file_import' },
    ],
    kb_mode: 'rules_only',
    ai_config: {
      ai_provider: '本地部署',
      model_name: 'Qwen2.5-72B',
      audit_strictness: 'standard',
      system_prompt: '你是一个专业的费用报销审核助手。请根据以下规则对报销申请进行合规性审核，重点关注金额合理性、发票合规性和审批材料完整性。',
      context_window: 4096,
      temperature: 0.2,
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
    flow_path: '部门经理 → 法务审核 → 财务总监 → 总经理',
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
      { id: 'R004', process_type: '合同审批', rule_content: '合同金额超过50万需法务部会签', rule_scope: 'mandatory', priority: 100, enabled: true, source: 'manual' },
      { id: 'R016', process_type: '合同审批', rule_content: '合同须包含知识产权归属条款', rule_scope: 'default_on', priority: 85, enabled: true, source: 'manual' },
      { id: 'R017', process_type: '合同审批', rule_content: '合作方须通过准入评审', rule_scope: 'mandatory', priority: 95, enabled: true, source: 'file_import' },
    ],
    kb_mode: 'rules_only',
    ai_config: {
      ai_provider: '云端API',
      model_name: 'GPT-4o',
      audit_strictness: 'strict',
      system_prompt: '你是一个专业的合同审核助手。请根据以下规则对合同进行全面审核，重点关注法律条款完整性、金额合理性和合作方资质。对于高风险条款请特别标注。',
      context_window: 16384,
      temperature: 0.1,
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
    flow_path: 'HR经理 → 用人部门 → HR总监',
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
      { id: 'R005', process_type: '人事审批', rule_content: '新增HC需部门负责人和HR总监双签', rule_scope: 'default_on', priority: 75, enabled: true, source: 'manual' },
      { id: 'R018', process_type: '人事审批', rule_content: '招聘人数须在年度HC计划范围内', rule_scope: 'mandatory', priority: 90, enabled: true, source: 'manual' },
    ],
    kb_mode: 'rules_only',
    ai_config: {
      ai_provider: '本地部署',
      model_name: 'Qwen2.5-32B',
      audit_strictness: 'loose',
      system_prompt: '你是一个专业的人事审批审核助手。请根据以下规则对人事申请进行审核，关注HC计划匹配度、审批链完整性和岗位合理性。',
      context_window: 4096,
      temperature: 0.3,
    },
    user_permissions: {
      allow_custom_fields: true,
      allow_custom_rules: true,
      allow_modify_strictness: true,
    },
  },
]

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
      amount: 320000,
      urgency: 'low',
    },
  ]

  const mockAuditResult: AuditResult = {
    trace_id: 'TR-20250610-A3F8',
    process_id: 'WF-2025-001',
    recommendation: 'revise',
    score: 72,
    duration_ms: 1850,
    details: [
      { rule_id: 'R001', rule_name: '预算额度校验', passed: true, reasoning: '采购金额 ¥156,000 未超过部门季度预算上限 ¥200,000', is_locked: true },
      { rule_id: 'R002', rule_name: '审批层级校验', passed: true, reasoning: '金额在 10-20 万区间，需部门经理 + 财务总监双签，审批链完整' },
      { rule_id: 'R003', rule_name: '供应商资质校验', passed: false, reasoning: '供应商"XX科技"未在合格供应商名录中，建议补充资质证明或更换供应商', is_locked: true },
      { rule_id: 'R004', rule_name: '采购比价要求', passed: false, reasoning: '单笔采购超过 ¥100,000 需提供至少 3 家供应商报价，当前仅提供 1 家' },
      { rule_id: 'R005', rule_name: '合同条款完整性', passed: true, reasoning: '合同包含付款条件、交付时间、售后条款等必要条款' },
    ],
    ai_reasoning: '该采购申请整体合规性尚可，但存在两个关键问题需要修正：\n\n1. 供应商资质问题：所选供应商未在企业合格供应商名录中登记，存在合规风险。建议申请人补充供应商资质材料或从已认证供应商中选择。\n\n2. 比价流程缺失：根据公司采购管理制度第 12 条，单笔采购金额超过 10 万元需进行竞争性比价（至少 3 家），当前申请仅提供了单一报价。\n\n建议：退回修改，要求补充比价材料和供应商资质证明后重新提交。',
  }

  const mockCronTasks: CronTask[] = [
    { id: 'CT-001', cron_expression: '0 9 * * 1-5', task_type: 'batch_audit', is_active: true, last_run_at: '2025-06-10 09:00', next_run_at: '2025-06-11 09:00', created_at: '2025-05-01', success_count: 28, fail_count: 1 },
    { id: 'CT-002', cron_expression: '0 18 * * 1-5', task_type: 'daily_report', is_active: true, last_run_at: '2025-06-09 18:00', next_run_at: '2025-06-10 18:00', created_at: '2025-05-01', success_count: 30, fail_count: 0 },
    { id: 'CT-003', cron_expression: '0 10 * * 1', task_type: 'weekly_report', is_active: true, last_run_at: '2025-06-09 10:00', next_run_at: '2025-06-16 10:00', created_at: '2025-05-15', success_count: 4, fail_count: 0 },
    { id: 'CT-004', cron_expression: '0 2 * * *', task_type: 'batch_audit', is_active: false, last_run_at: '2025-06-08 02:00', next_run_at: '-', created_at: '2025-04-20', success_count: 15, fail_count: 3 },
  ]

  const mockSnapshots: AuditSnapshot[] = [
    { snapshot_id: 'SN-001', process_id: 'WF-2025-098', title: '年度IT设备采购', applicant: '王强', department: 'IT部', recommendation: 'approve', score: 95, created_at: '2025-06-09 16:30', adopted: true },
    { snapshot_id: 'SN-002', process_id: 'WF-2025-097', title: '客户招待费报销', applicant: '李芳', department: '销售部', recommendation: 'reject', score: 35, created_at: '2025-06-09 15:20', adopted: true },
    { snapshot_id: 'SN-003', process_id: 'WF-2025-096', title: '新产品研发立项', applicant: '张明', department: '研发部', recommendation: 'approve', score: 88, created_at: '2025-06-09 14:10', adopted: true },
    { snapshot_id: 'SN-004', process_id: 'WF-2025-095', title: '办公用品批量采购', applicant: '刘洋', department: '行政部', recommendation: 'revise', score: 62, created_at: '2025-06-09 11:45', adopted: false },
    { snapshot_id: 'SN-005', process_id: 'WF-2025-094', title: '员工培训费用申请', applicant: '赵丽', department: '人力资源部', recommendation: 'approve', score: 91, created_at: '2025-06-08 17:00', adopted: true },
    { snapshot_id: 'SN-006', process_id: 'WF-2025-093', title: '广告投放合同签署', applicant: '陈伟', department: '市场部', recommendation: 'revise', score: 58, created_at: '2025-06-08 14:30', adopted: null },
  ]

  const mockTenants: TenantInfo[] = [
    { id: 'T-001', name: '示例集团总部', oa_type: 'weaver_e9', token_quota: 100000, token_used: 42350, max_concurrency: 20, status: 'active', created_at: '2025-01-15' },
    { id: 'T-002', name: '华东分公司', oa_type: 'weaver_e9', token_quota: 50000, token_used: 18200, max_concurrency: 10, status: 'active', created_at: '2025-02-20' },
    { id: 'T-003', name: '测试租户', oa_type: 'weaver_e9', token_quota: 10000, token_used: 3100, max_concurrency: 5, status: 'inactive', created_at: '2025-03-10' },
  ]

  // Derive rules from process audit configs for backward compatibility
  const mockRules: AuditRule[] = mockProcessAuditConfigs.flatMap(c => c.rules)

  const mockDashboardStats: DashboardStats = {
    todayAudits: 42,
    todayApproved: 28,
    todayRejected: 6,
    todayRevised: 8,
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

  // Approved processes - historical, read-only
  const mockApprovedProcesses: OAProcess[] = [
    { process_id: 'WF-2025-098', title: '年度IT设备采购', applicant: '王强', department: 'IT部', submit_time: '2025-06-09 16:30', process_type: '采购审批', status: 'approved', amount: 320000, urgency: 'medium' },
    { process_id: 'WF-2025-096', title: '新产品研发立项', applicant: '张明', department: '研发部', submit_time: '2025-06-09 14:10', process_type: '项目审批', status: 'approved', urgency: 'high' },
    { process_id: 'WF-2025-094', title: '员工培训费用申请', applicant: '赵丽', department: '人力资源部', submit_time: '2025-06-08 17:00', process_type: '费用报销', status: 'approved', amount: 45000, urgency: 'low' },
  ]

  // Rejected processes - historical, read-only
  const mockRejectedProcesses: OAProcess[] = [
    { process_id: 'WF-2025-097', title: '客户招待费报销', applicant: '李芳', department: '销售部', submit_time: '2025-06-09 15:20', process_type: '费用报销', status: 'rejected', amount: 28000, urgency: 'medium' },
    { process_id: 'WF-2025-091', title: '未经审批的外包合同', applicant: '陈伟', department: '市场部', submit_time: '2025-06-08 10:00', process_type: '合同审批', status: 'rejected', amount: 150000, urgency: 'high' },
  ]

  // Historical audit results keyed by process_id
  const mockHistoricalResults: Record<string, AuditResult> = {
    'WF-2025-098': {
      trace_id: 'TR-20250609-B1C2', process_id: 'WF-2025-098', recommendation: 'approve', score: 95, duration_ms: 1420,
      details: [
        { rule_id: 'R001', rule_name: '预算额度校验', passed: true, reasoning: '采购金额在部门年度预算范围内', is_locked: true },
        { rule_id: 'R002', rule_name: '审批层级校验', passed: true, reasoning: '审批链完整，已获得所有必要签批' },
        { rule_id: 'R003', rule_name: '供应商资质校验', passed: true, reasoning: '供应商在合格名录中，资质有效期内' },
      ],
      ai_reasoning: '该采购申请完全符合公司采购管理制度要求，预算合理、审批链完整、供应商资质齐全。建议通过。',
    },
    'WF-2025-096': {
      trace_id: 'TR-20250609-D3E4', process_id: 'WF-2025-096', recommendation: 'approve', score: 88, duration_ms: 1680,
      details: [
        { rule_id: 'R010', rule_name: '立项必要性评估', passed: true, reasoning: '市场调研数据充分，立项理由成立' },
        { rule_id: 'R011', rule_name: '预算可行性', passed: true, reasoning: '研发预算在年度规划范围内' },
        { rule_id: 'R012', rule_name: '人员配置合理性', passed: false, reasoning: '项目团队缺少测试工程师角色，但不影响立项' },
      ],
      ai_reasoning: '研发立项申请整体合规，市场调研充分，预算合理。建议补充测试人员配置后通过。',
    },
    'WF-2025-094': {
      trace_id: 'TR-20250608-F5G6', process_id: 'WF-2025-094', recommendation: 'approve', score: 91, duration_ms: 1150,
      details: [
        { rule_id: 'R003', rule_name: '费用标准校验', passed: true, reasoning: '培训费用符合公司标准' },
        { rule_id: 'R004', rule_name: '培训计划审核', passed: true, reasoning: '培训内容与岗位需求匹配' },
      ],
      ai_reasoning: '员工培训费用申请合规，培训内容与业务需求高度相关，费用在标准范围内。建议通过。',
    },
    'WF-2025-097': {
      trace_id: 'TR-20250609-H7I8', process_id: 'WF-2025-097', recommendation: 'reject', score: 35, duration_ms: 1320,
      details: [
        { rule_id: 'R003', rule_name: '费用标准校验', passed: false, reasoning: '招待费用超出公司标准上限 200%', is_locked: true },
        { rule_id: 'R006', rule_name: '审批材料完整性', passed: false, reasoning: '缺少客户拜访记录和招待事由说明' },
        { rule_id: 'R007', rule_name: '发票合规性', passed: false, reasoning: '部分发票日期与申报时间不符' },
      ],
      ai_reasoning: '该报销申请存在多项严重违规：费用严重超标、材料不完整、发票存疑。建议驳回并要求重新整理材料。',
    },
    'WF-2025-091': {
      trace_id: 'TR-20250608-J9K0', process_id: 'WF-2025-091', recommendation: 'reject', score: 22, duration_ms: 1560,
      details: [
        { rule_id: 'R004', rule_name: '合同审批前置条件', passed: false, reasoning: '合同签署前未经过法务审核', is_locked: true },
        { rule_id: 'R008', rule_name: '供应商准入', passed: false, reasoning: '外包供应商未通过准入评审' },
        { rule_id: 'R009', rule_name: '预算审批', passed: false, reasoning: '合同金额未纳入年度预算' },
      ],
      ai_reasoning: '该合同存在严重合规问题：未经法务审核即签署、供应商未准入、预算未审批。建议驳回并启动合规调查。',
    },
  }

  // ============================================================
  // Archive Review (归档复盘) - Full process compliance re-audit
  // ============================================================

  // Archived processes that have completed all approval nodes
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
        { node_id: 'N2', node_name: '财务审核', approver: '张华', action: 'revise', action_time: '2025-03-22 14:00', opinion: '部分发票不清晰，请补充' },
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
        { node_id: 'N2', node_name: '法务审核', approver: '孙律', action: 'revise', action_time: '2025-02-15 16:00', opinion: '知识产权条款需修改' },
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
        { node_id: 'N3', node_name: '总经理审批', approver: '刘总', action: 'reject', action_time: '2025-01-15 09:00', opinion: '施工时间与业务高峰冲突，请调整' },
        { node_id: 'N4', node_name: '行政经理重新提交', approver: '刘洋', action: 'approve', action_time: '2025-01-18 11:00', opinion: '已调整至春节假期施工' },
        { node_id: 'N5', node_name: '总经理审批（重审）', approver: '刘总', action: 'approve', action_time: '2025-01-20 09:30', opinion: '时间调整合理，批准' },
      ],
      fields: { floor: '3楼、5楼', contractor: 'ZZ装饰工程公司', construction_period: '2025-01-25 至 2025-02-10' },
    },
  ]

  // Full-process compliance re-audit result
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
  // Archived processes for dashboard "归档" tab
  // These are completed processes (final result = approved) with multi-round audit chains
  // ============================================================
  const mockArchivedOAProcesses: OAProcess[] = [
    { process_id: 'WF-2025-050', title: '2025年度服务器集群采购', applicant: '王强', department: 'IT部', submit_time: '2025-04-15 09:00', process_type: '采购审批', status: 'archived', amount: 1200000, urgency: 'high' },
    { process_id: 'WF-2025-038', title: '华东区域市场推广费用报销', applicant: '陈伟', department: '市场部', submit_time: '2025-03-20 11:00', process_type: '费用报销', status: 'archived', amount: 85000, urgency: 'medium' },
    { process_id: 'WF-2025-025', title: '外包开发合同签署 - CRM系统二期', applicant: '赵丽', department: '研发部', submit_time: '2025-02-10 14:00', process_type: '合同审批', status: 'archived', amount: 680000, urgency: 'high' },
    { process_id: 'WF-2025-012', title: '新员工批量入职审批 - 2025春招', applicant: '赵丽', department: '人力资源部', submit_time: '2025-01-20 09:00', process_type: '人事审批', status: 'archived', urgency: 'low' },
    { process_id: 'WF-2025-008', title: '办公楼层装修改造工程', applicant: '刘洋', department: '行政部', submit_time: '2025-01-10 10:00', process_type: '工程审批', status: 'archived', amount: 450000, urgency: 'medium' },
  ]

  // Multi-round audit chain snapshots for archived processes (final round always approve)
  const mockArchivedAuditChains: Record<string, AuditSnapshot[]> = {
    'WF-2025-050': [
      { snapshot_id: 'SN-A001', process_id: 'WF-2025-050', title: '2025年度服务器集群采购', applicant: '王强', department: 'IT部', recommendation: 'revise', score: 68, created_at: '2025-04-16 10:30', adopted: true },
      { snapshot_id: 'SN-A002', process_id: 'WF-2025-050', title: '2025年度服务器集群采购', applicant: '王强', department: 'IT部', recommendation: 'revise', score: 82, created_at: '2025-04-25 14:00', adopted: true },
      { snapshot_id: 'SN-A003', process_id: 'WF-2025-050', title: '2025年度服务器集群采购', applicant: '王强', department: 'IT部', recommendation: 'approve', score: 95, created_at: '2025-05-10 09:15', adopted: true },
    ],
    'WF-2025-038': [
      { snapshot_id: 'SN-A004', process_id: 'WF-2025-038', title: '华东区域市场推广费用报销', applicant: '陈伟', department: '市场部', recommendation: 'reject', score: 42, created_at: '2025-03-22 15:00', adopted: true },
      { snapshot_id: 'SN-A005', process_id: 'WF-2025-038', title: '华东区域市场推广费用报销', applicant: '陈伟', department: '市场部', recommendation: 'approve', score: 90, created_at: '2025-03-28 11:30', adopted: true },
    ],
    'WF-2025-025': [
      { snapshot_id: 'SN-A006', process_id: 'WF-2025-025', title: '外包开发合同签署 - CRM系统二期', applicant: '赵丽', department: '研发部', recommendation: 'revise', score: 55, created_at: '2025-02-12 10:00', adopted: true },
      { snapshot_id: 'SN-A007', process_id: 'WF-2025-025', title: '外包开发合同签署 - CRM系统二期', applicant: '赵丽', department: '研发部', recommendation: 'revise', score: 78, created_at: '2025-02-18 16:00', adopted: true },
      { snapshot_id: 'SN-A008', process_id: 'WF-2025-025', title: '外包开发合同签署 - CRM系统二期', applicant: '赵丽', department: '研发部', recommendation: 'approve', score: 92, created_at: '2025-02-24 09:30', adopted: true },
    ],
    'WF-2025-012': [
      { snapshot_id: 'SN-A009', process_id: 'WF-2025-012', title: '新员工批量入职审批 - 2025春招', applicant: '赵丽', department: '人力资源部', recommendation: 'approve', score: 96, created_at: '2025-01-22 11:00', adopted: true },
    ],
    'WF-2025-008': [
      { snapshot_id: 'SN-A010', process_id: 'WF-2025-008', title: '办公楼层装修改造工程', applicant: '刘洋', department: '行政部', recommendation: 'reject', score: 38, created_at: '2025-01-12 14:00', adopted: true },
      { snapshot_id: 'SN-A011', process_id: 'WF-2025-008', title: '办公楼层装修改造工程', applicant: '刘洋', department: '行政部', recommendation: 'revise', score: 71, created_at: '2025-01-16 10:30', adopted: true },
      { snapshot_id: 'SN-A012', process_id: 'WF-2025-008', title: '办公楼层装修改造工程', applicant: '刘洋', department: '行政部', recommendation: 'approve', score: 89, created_at: '2025-01-19 15:00', adopted: true },
    ],
  }

  // Historical audit results for archived processes (final approved result)
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

  return {
    mockProcesses,
    mockApprovedProcesses,
    mockRejectedProcesses,
    mockHistoricalResults,
    mockAuditResult,
    mockCronTasks,
    mockSnapshots,
    mockTenants,
    mockRules,
    mockDashboardStats,
    mockArchivedProcesses,
    mockArchiveAuditResult,
    mockArchivedOAProcesses,
    mockArchivedAuditChains,
    mockArchivedHistoricalResults,
    mockProcessAuditConfigs: [...mockProcessAuditConfigs],
  }
}
