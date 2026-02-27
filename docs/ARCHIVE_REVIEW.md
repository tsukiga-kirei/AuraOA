# 归档复盘功能文档

> OA 智审平台 — 归档复盘工作台（`/archive`）前端功能规范与数据结构说明

---

## 1. 功能概述

归档复盘是面向业务用户的事后合规复核模块，针对已完成审批并归档的 OA 流程，支持 AI 全流程合规重审、审批链校验、字段校验、规则校验，并提供批量复核与报告导出能力。

核心特点：
- 基于 RBAC 的访问控制：用户只能看到自己有权限的归档流程类型
- 两阶段 AI 复核：阶段一深度推理分析，阶段二结构化结论提取
- 审批链完整性校验：检测缺失节点、异常节点
- 不可篡改：所有复核记录只读，支持审计导出

---

## 2. 页面路由

| 路由 | 说明 |
|------|------|
| `/archive` | 归档复盘工作台主页面 |

---

## 3. 数据结构

### 3.1 归档流程（ArchivedProcess）

```typescript
interface ArchivedProcess {
  process_id: string          // 流程唯一编号，如 PROC-2024-001
  title: string               // 流程标题
  process_type: string        // 流程类型，如 采购审批、费用报销
  applicant: string           // 申请人姓名
  department: string          // 申请部门
  submit_time: string         // 提交时间
  archive_time: string        // 归档时间
  original_result: string     // 原始审批结果
  flow_nodes: FlowNode[]      // 审批链节点列表
}
```

### 3.2 审批链节点（FlowNode）

```typescript
interface FlowNode {
  node_id: string
  node_name: string           // 节点名称，如 部门经理审批
  approver: string            // 审批人
  action: string              // 操作：approve / return / forward
  timestamp: string           // 操作时间
  comment?: string            // 审批意见
}
```

### 3.3 归档复核结果（ArchiveAuditResult）

```typescript
interface ArchiveAuditResult {
  process_id: string
  overall_compliance: 'compliant' | 'partially_compliant' | 'non_compliant'
  overall_score: number       // 0-100
  duration_ms: number         // AI 耗时（毫秒）
  flow_chain_result: {
    is_complete: boolean
    missing_nodes: string[]
    node_results: FlowNodeAuditResult[]
  }
  field_results: FieldAuditResult[]
  rule_checks: ChecklistResult[]
  risk_points: string[]
  suggestions: string[]
  ai_reasoning: string        // 阶段一推理过程原文
  recommendation: 'approve' | 'return'
}
```

### 3.4 归档复盘配置（ArchiveReviewConfig）

由租户管理员在规则配置后台维护，每个流程类型对应一份配置：

```typescript
interface ArchiveReviewConfig {
  id: string
  process_type: string
  main_table_name?: string            // OA 主表名
  main_fields?: ProcessField[]        // 主表字段定义
  detail_tables?: DetailTable[]       // 明细表定义
  fields: ProcessField[]              // 参与复核的字段（兼容旧结构）
  field_mode: 'all' | 'selected'      // 字段模式
  rules: AuditRule[]                  // 复核规则列表
  flow_rules: FlowRuleConfig[]        // 审批流合规规则
  kb_mode: 'rules_only' | 'rag_only' | 'hybrid'
  ai_config: ProcessAIConfig          // AI 配置（尺度、提示词）
  user_permissions: {
    allow_custom_fields: boolean
    allow_custom_rules: boolean
    allow_custom_flow_rules: boolean
    allow_modify_strictness: boolean
  }
  allowed_roles: string[]             // 允许访问的角色 ID 列表
  allowed_members: string[]           // 允许访问的成员 ID 列表
  allowed_departments?: string[]      // 允许访问的部门 ID 列表
}
```

---

## 4. 访问控制机制

归档复盘采用三维度访问控制，用户满足任一条件即可访问对应流程类型：

```
allowed_roles    → 用户的 role_id 在列表中
allowed_members  → 用户的 member id 在列表中
allowed_departments → 用户所属部门 ID 在列表中（预留，前端暂通过角色/成员判断）
```

前端实现（`archive.vue`）：

```typescript
const accessibleConfigs = computed<ArchiveReviewConfig[]>(() => {
  const member = mockOrgMembers.find(m => m.username === currentUser.value?.username)
  if (!member) return []
  return mockArchiveReviewConfigs.filter(cfg =>
    cfg.allowed_roles.includes(member.role_id) ||
    cfg.allowed_members.includes(member.id)
  )
})
```

流程列表只展示 `accessibleProcessTypes` 中包含的流程类型，其余流程对当前用户不可见。

---

## 5. AI 复核流程

### 两阶段模式

| 阶段 | 说明 |
|------|------|
| 阶段一：深度分析 | AI 对审批链、字段、规则逐一推理，输出自然语言分析过程 |
| 阶段二：结构化提取 | 基于阶段一结果，按 JSON Schema 提取结构化复核结论 |

### Prompt 变量

推理 Prompt 可用变量：

| 变量 | 说明 |
|------|------|
| `{{main_table}}` | 主表字段数据 |
| `{{detail_tables}}` | 明细表数据 |
| `{{rules}}` | 复核规则列表 |
| `{{flow_history}}` | 审批历史记录 |
| `{{flow_graph}}` | 审批流结构图 |
| `{{current_node}}` | 当前审批节点 |

提取 Prompt 可用变量：

| 变量 | 说明 |
|------|------|
| `{{rules}}` | 原始规则列表 |
| `{{flow_graph}}` | 审批流结构图 |

### 审核尺度

| 尺度 | 说明 |
|------|------|
| `strict` | 任何疑点均建议退回，零容忍 |
| `standard` | 明确违规建议退回，存疑项说明理由 |
| `loose` | 仅明显违规建议退回，轻微问题仅提示 |

---

## 6. 前端功能清单

### 6.1 流程列表区

- 搜索：流程名称、申请人、流程编号模糊搜索
- 筛选：部门、流程类型（多选）、合规状态
- 全选 / 批量复核
- 已复核计数展示
- OA 跳转按钮（link 样式，带 ExportOutlined 图标）

### 6.2 复核详情区

- 流程基本信息（申请人、部门、提交时间、归档时间）
- 审批链可视化（节点列表，含缺失节点标记）
- 字段校验结果
- 规则校验结果（逐条 pass/fail）
- AI 推理过程（阶段一原文）
- 综合评分 + 合规状态 + 耗时
- 风险点列表
- 改进建议列表
- 复核建议（approve / return）

### 6.3 统计卡片

页面顶部展示四项统计：

| 指标 | 说明 |
|------|------|
| 归档总数 | 当前筛选列表总数 |
| 合规 | overall_compliance = compliant 的数量 |
| 部分合规 | overall_compliance = partially_compliant 的数量 |
| 不合规 | overall_compliance = non_compliant 的数量 |

### 6.4 导出功能

支持三种格式导出合规复核报告（需先选中流程）：
- JSON
- CSV
- Excel

---

## 7. 租户管理员配置（rules.vue — 归档复盘 Tab）

路由：`/admin/tenant/rules`，顶部 Tab 切换至「归档复盘」。

### 7.1 配置项

每个归档流程配置包含以下 Tab：

| Tab | 说明 |
|-----|------|
| 字段配置 | 选择参与复核的主表/明细表字段，支持全部字段/选择字段两种模式 |
| 复核规则 | 手工添加或文件导入规则，支持强制/默认开启/默认关闭三级 |
| AI 配置 | 审核尺度选择、两阶段 Prompt 模板编辑、变量插入 |
| 用户权限 | 控制用户是否可自定义字段/规则/尺度 |
| 访问控制 | 配置允许访问的角色、人员、部门 |

### 7.2 访问控制配置

三个维度均支持搜索过滤：

```
角色维度  → archiveRoleSearch   → filteredArchiveRoles
人员维度  → archiveMemberSearch → filteredArchiveMembers
部门维度  → archiveDeptSearch   → filteredArchiveDepts
```

部门通过 `toggleArchiveDept(deptId)` 切换选中状态，数据来源于 `mockDepartments`。

---

## 8. 个人设置（settings.vue — 归档复盘 Tab）

路由：`/settings`，Tab 切换至「归档复盘」。

用户可在权限允许范围内个性化配置：

| 配置项 | 权限控制字段 |
|--------|-------------|
| 复核字段开关 | `allow_custom_fields` |
| 个人自定义复核规则 | `allow_custom_rules` |
| 个人自定义审批流规则 | `allow_custom_flow_rules` |
| 复核尺度调整 | `allow_modify_strictness` |

---

## 9. Mock 数据说明

当前前端使用 `useMockData.ts` 中的模拟数据，后续对接后端时按以下映射替换：

| Mock 数据 | 对应后端接口 |
|-----------|-------------|
| `mockArchiveReviewConfigs` | `GET /api/tenant/archive-configs` |
| `mockArchivedProcesses` | `GET /api/archive/processes` |
| `mockArchiveAuditResult` | `GET /api/archive/audit-result/:processId` |
| AI 复核触发 | `POST /api/archive/audit` |
| 批量复核 | `POST /api/archive/batch-audit` |
| 导出报告 | `GET /api/archive/export?format=json|csv|excel` |

---

## 10. i18n Key 索引

归档复盘相关 i18n key 分布在以下命名空间：

| 命名空间 | 说明 |
|----------|------|
| `archive.*` | 归档复盘页面文案 |
| `archive.selectAll` | 全选 |
| `archive.selected` | 已选 N 项 |
| `admin.ruleConfig.archive*` | 租户管理员归档配置文案 |
| `admin.ruleConfig.archiveAllowedDepts` | 允许访问的部门 |
| `admin.ruleConfig.archiveAccessSearch` | 访问控制搜索框占位符 |
| `settings.archive.*` | 个人设置归档复盘 Tab 文案 |
