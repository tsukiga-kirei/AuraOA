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

export function getMockMenusByRole(role: UserRole): MockMenuItem[] {
  const base: MockMenuItem[] = [
    { key: 'dashboard', label: '审核工作台', path: '/dashboard' },
    { key: 'cron', label: '定时任务', path: '/cron' },
    { key: 'archive', label: '归档复盘', path: '/archive' },
  ]
  const tenant: MockMenuItem[] = [
    { key: 'tenant', label: '租户配置', path: '/admin/tenant' },
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

  const mockRules: AuditRule[] = [
    { id: 'R001', process_type: '采购审批', rule_content: '单笔采购金额不得超过部门季度预算上限', rule_scope: 'mandatory', priority: 100, enabled: true },
    { id: 'R002', process_type: '采购审批', rule_content: '超过10万元需提供至少3家供应商比价', rule_scope: 'mandatory', priority: 95, enabled: true },
    { id: 'R003', process_type: '费用报销', rule_content: '单次报销金额超过5000元需附发票原件', rule_scope: 'default_on', priority: 80, enabled: true },
    { id: 'R004', process_type: '合同审批', rule_content: '合同金额超过50万需法务部会签', rule_scope: 'mandatory', priority: 100, enabled: true },
    { id: 'R005', process_type: '人事审批', rule_content: '新增HC需部门负责人和HR总监双签', rule_scope: 'default_on', priority: 75, enabled: true },
    { id: 'R006', process_type: '费用报销', rule_content: '差旅住宿标准不超过城市限额', rule_scope: 'default_off', priority: 60, enabled: false },
  ]

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

  return {
    mockProcesses,
    mockAuditResult,
    mockCronTasks,
    mockSnapshots,
    mockTenants,
    mockRules,
    mockDashboardStats,
  }
}
