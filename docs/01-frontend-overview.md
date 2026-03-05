# OA 智审平台 — 前端功能说明文档

> 文档版本：v1.0 | 更新日期：2026-03-02  
> 本文档基于前端代码（Nuxt 3 + Ant Design Vue）的完整分析，梳理系统功能模块、模拟数据结构及其勾稽关系。

---

## 一、项目概览

**OA 智审（OA Smart Audit）** 是一个面向企业 OA 流程的 **AI 辅助智能审核平台**。其核心能力是：

1. **对接企业 OA 系统**（如泛微 E9、致远 A8 等），获取待审批流程数据
2. **利用大模型 AI**（Qwen、DeepSeek 等）自动执行合规性审核
3. **提供多角色、多租户的管理后台**，支持规则配置、组织管理、数据分析

### 1.1 技术栈

| 层级 | 技术 |
|------|------|
| 前端框架 | Nuxt 3 (Vue 3 + TypeScript) |
| UI 组件库 | Ant Design Vue |
| 状态管理 | Nuxt `useState` + `localStorage` 持久化 |
| 国际化 | 自定义 `useI18n` composable (zh-CN / en-US) |
| 主题 | CSS Variables + `data-theme` 属性切换（明/暗） |
| 图标 | @ant-design/icons-vue |
| 构建 | Vite (内置于 Nuxt 3) |

### 1.2 整体架构

```
frontend/
├── app.vue                 # 根组件，提供 Ant Design ConfigProvider
├── nuxt.config.ts          # Nuxt 配置（API Base、Mock Mode 等）
├── assets/css/             # CSS 变量定义 + 全局样式
├── components/             # 可复用组件（7 个）
├── composables/            # 核心业务逻辑（8 个 composable）
├── layouts/                # 布局模板（default.vue）
├── locales/                # 国际化文件（zh-CN、en-US）
├── middleware/             # 路由守卫（auth.ts）
└── pages/                  # 页面路由
    ├── login.vue           # 登录页
    ├── index.vue           # 首页重定向
    ├── overview.vue        # 仪表盘（所有角色首页）
    ├── dashboard.vue       # 审核工作台（业务用户）
    ├── cron.vue            # 定时任务（业务用户）
    ├── archive.vue         # 归档复盘（业务用户）
    ├── settings.vue        # 个人设置（所有角色）
    └── admin/
        ├── tenant/         # 租户管理后台
        │   ├── rules.vue   # 规则配置
        │   ├── org.vue     # 组织人员
        │   ├── data.vue    # 数据信息
        │   └── user-configs.vue  # 用户偏好
        └── system/         # 系统管理后台
            ├── tenants.vue # 租户管理
            └── settings.vue # 系统设置
```

---

## 二、角色与权限体系

### 2.1 三级角色模型

系统采用 **三级角色模型**，角色类型定义为：

| 角色类型 | 英文标识 | 说明 |
|----------|----------|------|
| 业务用户 | `business` | 使用审核工作台、定时任务、归档复盘等前台功能 |
| 租户管理员 | `tenant_admin` | 管理租户范围内的规则、组织人员、数据、用户偏好 |
| 系统管理员 | `system_admin` | 管理全局租户、系统设置 |

### 2.2 用户角色分配 (UserRoleAssignment)

一个用户可以拥有 **多个角色分配**，每个分配绑定到特定租户：

```typescript
interface UserRoleAssignment {
  id: string           // 分配唯一ID
  role: UserRole       // 'business' | 'tenant_admin' | 'system_admin'
  tenant_id: string | null    // 租户ID（system_admin 为 null）
  tenant_name: string | null  // 租户名称
  label: string        // 显示标签，如 "示例集团总部 · 业务用户"
}
```

### 2.3 模拟用户数据

系统预置了 **8 个测试用户**，覆盖各种角色组合场景：

| 用户名 | 姓名 | 密码 | 角色分配 |
|--------|------|------|----------|
| `admin` | 陈刚 | 123456 | 系统管理员 + 总部租户管理员 + 总部业务用户 |
| `sysadmin2` | 周敏 | 123456 | 系统管理员 + 华东分公司管理员 |
| `sysadmin3` | 吴强 | 123456 | 纯系统管理员 |
| `tenantadmin` | 赵伟 | 123456 | 总部租户管理员 + 总部业务用户 |
| `wanggang` | 王刚 | 123456 | 华东管理员 + 总部业务用户（跨租户） |
| `zhangming` | 张明 | 123456 | 总部业务用户 |
| `lifang` | 李芳 | 123456 | 总部业务用户 + 华东业务用户（多租户） |
| `user` | 测试用户 | 123456 | 总部业务用户 |

### 2.4 页面权限矩阵

```typescript
const PAGE_PERMISSIONS = {
  '/overview':                ['business', 'tenant_admin', 'system_admin'],
  '/dashboard':               ['business'],
  '/cron':                    ['business'],
  '/archive':                 ['business'],
  '/settings':                ['business', 'tenant_admin', 'system_admin'],
  '/admin/tenant/rules':      ['tenant_admin'],
  '/admin/tenant/org':        ['tenant_admin'],
  '/admin/tenant/data':       ['tenant_admin'],
  '/admin/tenant/user-configs': ['tenant_admin'],
  '/admin/system/tenants':    ['system_admin'],
  '/admin/system/settings':   ['system_admin'],
}
```

### 2.5 业务角色（租户内部）

在租户管理员视角下，组织内的业务角色有更细粒度的页面权限控制：

| 角色ID | 名称 | 页面权限 | 系统角色 |
|--------|------|----------|----------|
| ROLE-001 | 业务用户 | 仪表盘、审核工作台、个人设置 | ✅ |
| ROLE-002 | 审计管理员 | 业务用户权限 + 定时任务、归档复盘 | ✅ |
| ROLE-003 | 租户管理员 | 全部前台 + 全部后台管理 | ✅ |

---

## 三、功能模块详解

### 3.1 登录页 (`login.vue`)

**三入口登录**：用户在登录页选择身份入口（业务用户/租户管理员/系统管理员），系统根据选择优先激活匹配的角色。

- **Portal 选择器**：水平 Pill 标签切换入口类型
- **租户选择**：非系统管理员需选择租户
- **快速填充**：Mock 模式下显示可用测试账号
- **登录流程**：用户名 + 密码 → 匹配用户 → 设置 activeRole → 生成菜单 → 跳转 `/overview`

### 3.2 仪表盘 (`overview.vue`)

**可自定义的角色感知仪表盘**，根据当前角色显示不同的 Widget：

| Widget ID | 名称 | 可见角色 | 尺寸 |
|-----------|------|----------|------|
| `audit_summary` | 审核概览 | business | lg |
| `pending_tasks` | 待办任务 | business | sm |
| `weekly_trend` | 审核趋势 | business | md |
| `cron_tasks` | 定时任务 | business | md |
| `archive_review` | 归档复盘 | business | md |
| `dept_distribution` | 部门分布 | tenant_admin | md |
| `recent_activity` | 最近动态 | 所有角色 | md |
| `ai_performance` | AI 模型表现 | tenant_admin | md |
| `tenant_usage` | 租户资源用量 | tenant_admin | md |
| `user_activity` | 用户活跃排行 | tenant_admin | md |
| `system_health` | 系统健康 | system_admin | lg |
| `tenant_overview` | 租户总览 | system_admin | md |
| `api_metrics` | API 调用指标 | system_admin | md |
| `monitor_metrics` | 运行指标 | system_admin | lg |
| `monitor_alerts` | 最近告警 | system_admin | md |

### 3.3 审核工作台 (`dashboard.vue`)

**核心业务功能页面**，包含三个 Tab：

1. **待办流程 (Todo)**：显示待审核的 OA 流程列表
   - 支持按流程类型筛选
   - 每个流程可执行 AI 单条审核或批量审核
   - 审核结果显示：建议操作、得分、规则校验详情、风险点、改进建议
   - 支持"采纳"或"不采纳"AI 建议

2. **已通过 (Approved)**：历史已通过的流程记录

3. **已退回 (Returned)**：历史已退回的流程记录

**AI 审核结果结构 (AuditResult)**：
```typescript
interface AuditResult {
  trace_id: string                              // 追踪ID
  process_id: string                            // 流程ID
  recommendation: 'approve' | 'return' | 'review'  // AI建议操作
  score: number                                 // 合规得分(0-100)
  details: ChecklistResult[]                    // 逐条规则校验
  ai_reasoning: string                          // AI推理过程
  action_label: string                          // 建议操作标签
  confidence: number                            // 置信度
  risk_points: string[]                         // 风险点列表
  suggestions: string[]                         // 改进建议列表
  ai_summary: string                            // AI摘要
  model_used: string                            // 使用的模型
  interaction_mode: 'two_phase' | 'single_pass' // 交互模式
  phase1_duration_ms: number                    // 第一阶段耗时
  phase2_duration_ms: number                    // 第二阶段耗时
}
```

### 3.4 定时任务 (`cron.vue`)

管理自动化审核和报告推送任务：

| 任务类型 | 标识 | 说明 |
|----------|------|------|
| 批量审核 | `batch_audit` | 定时批量执行AI审核 |
| 日报推送 | `daily_report` | 每日审核汇总邮件推送 |
| 周报推送 | `weekly_report` | 每周审核汇总邮件推送 |

- 支持 CRON 表达式配置
- 显示执行历史和成功/失败统计
- 内建任务不可删除，用户可自定义任务

### 3.5 归档复盘 (`archive.vue`)

对已完成的 OA 流程进行 **全流程合规复核**：

- **流程审批链审核**：检查审批节点是否完整、权限是否匹配
- **字段数据审核**：校验表单字段值的合规性
- **规则合规审核**：逐条规则校验
- 支持多轮审核链追溯（审核→退回→修改→再审核→通过）

### 3.6 个人设置 (`settings.vue`)

- 密码修改（新旧密码不能相同校验；修改成功后自动退出登录）
- 登录历史查看
- 语言偏好设置
- 仪表盘 Widget 自定义

### 3.7 租户管理后台

#### 3.7.1 规则配置 (`admin/tenant/rules.vue`)

按流程类型管理审核规则和归档复盘规则：

**审核规则配置 (ProcessAuditConfig)**：
- 流程类型分类（采购类、费用类、合同类、人事类、工程类、项目类、预算类）
- 主表/明细表字段选择（来源于 OA 表单）
- 审核规则管理（mandatory/default_on/default_off）
- 知识库模式（rules_only/rag_only/hybrid）
- AI 配置（审核尺度、推理提示词、提取提示词）
- 用户权限配置（是否允许自定义字段/规则/尺度）

**审核尺度预设 (StrictnessPromptPreset)**：
- strict: 最严格标准，宁可误判也不放过
- standard: 常规标准，客观公正
- loose: 宽松标准，只关注重大风险

#### 3.7.2 组织人员 (`admin/tenant/org.vue`)

管理租户内的部门、角色、人员：

- **部门管理**：8 个模拟部门（研发、销售、市场、人力、IT、财务、行政、法务）
- **角色管理**：业务角色的创建、编辑、页面权限分配
- **人员管理**：用户的部门归属、角色分配、状态管理

#### 3.7.3 数据信息 (`admin/tenant/data.vue`)

- **审核日志**：AI 审核的操作记录
- **定时任务日志**：定时任务的执行记录
- **归档复盘日志**：合规复核的记录

#### 3.7.4 用户偏好 (`admin/tenant/user-configs.vue`)

查看和管理租户内用户的个人配置覆盖：

- 用户自定义的审核规则
- 字段选择覆盖
- 审核尺度覆盖
- 自定义定时任务

### 3.8 系统管理后台

#### 3.8.1 租户管理 (`admin/system/tenants.vue`)

管理平台全部租户：

**租户信息 (TenantInfo)**：
- 基本信息（名称、编码、联系人等）
- OA 系统连接（关联系统级 OA 数据库连接）
- Token 配额与用量
- AI 配置（默认模型、备用模型、温度等参数）
- 安全配置（SSO、数据保留天数）

#### 3.8.2 系统设置 (`admin/system/settings.vue`)

**三个 Tab**：

1. **OA 系统管理**：OA 数据库连接管理（JDBC 配置、同步状态）
2. **AI 模型管理**：AI 模型配置（端点、密钥、状态、能力标签）
3. **平台设置**：平台名称、版本、SMTP、备份策略等通用配置

---

## 四、数据模型勾稽关系

### 4.1 租户 ← → 用户 ← → 角色

```
Tenant (T-001 示例集团总部)
  ├── UserRoleAssignment (admin-r2: admin → tenant_admin @ T-001)
  ├── UserRoleAssignment (admin-r3: admin → business @ T-001)
  ├── UserRoleAssignment (ta1-r1: tenantadmin → tenant_admin @ T-001)
  ├── ...
  └── OrgMember (M-001 张明)
       ├── role_ids: [ROLE-001, ROLE-002]  → 业务角色
       └── department_id: D-001 → 研发部
```

### 4.2 规则配置 ← → 审核结果

```
ProcessAuditConfig (PAC-001 采购审批)
  ├── rules: [R001预算校验, R002比价要求, R013供应商校验, ...]
  ├── ai_config: { strictness, reasoning_prompt, extraction_prompt }
  └── user_permissions: { allow_custom_rules, ... }

                    ↓ AI 审核引擎 ↓

AuditResult (TR-20250610-A3F8)
  ├── recommendation: 'return'
  ├── score: 72
  ├── details: [
  │     { rule_id: R001, passed: true, reasoning: "..." },
  │     { rule_id: R003, passed: false, reasoning: "..." },
  │   ]
  └── risk_points, suggestions, ai_summary
```

### 4.3 流程数据流

```
OA系统 → [OAProcess 待办流程]
              ↓
         审核工作台 Dashboard
              ↓  (AI审核)
         AuditResult → Snapshot
              ↓  (采纳/不采纳)
         ApprovedProcess / ReturnedProcess
              ↓  (归档)
         ArchivedProcess → ArchiveAuditResult (合规复核)
```

### 4.4 系统配置关系

```
SystemGeneralConfig (平台全局配置)
  ├── OASystemConfig[] (OA系统类型定义)
  ├── OADatabaseConnection[] (JDBC连接实例)
  │     ├── OADB-001 → 被 Tenant T-001 引用
  │     └── OADB-002 → 被 Tenant T-002 引用
  └── AIModelConfig[] (AI模型定义)
        ├── AI-001 Qwen2.5-72B (本地) → 被租户AI配置引用
        └── AI-003 qwen-plus (云端) → 被租户AI配置引用
```

---

## 五、流程类型全景

系统支持 **7 种流程类型**，每种关联主表和明细表：

| 流程类型 | 分类标签 | 主表名 | 明细表 | 规则数 |
|----------|----------|--------|--------|--------|
| 采购审批 | 采购类 | formtable_main_001 | 采购明细 | 5 |
| 费用报销 | 费用类 | formtable_main_002 | 发票明细 | 3 |
| 合同审批 | 合同类 | formtable_main_003 | — | 3 |
| 人事审批 | 人事类 | formtable_main_004 | — | 2 |
| 工程审批 | 工程类 | formtable_main_005 | — | 2 |
| 项目审批 | 项目类 | formtable_main_006 | — | 2 |
| 预算审批 | 预算类 | formtable_main_007 | — | 1 |

---

## 六、模拟数据统计汇总

| 数据类型 | 数量 | 说明 |
|----------|------|------|
| 用户 | 8 | 覆盖所有角色组合 |
| 租户 | 3 | 总部、华东分公司、测试 |
| 部门 | 8 | 研发/销售/市场/人力/IT/财务/行政/法务 |
| 组织角色 | 3 | 业务用户/审计管理员/租户管理员 |
| 组织成员 | 12 | 分布在各部门 |
| 待办流程 | 12 | 多种流程类型 |
| 已通过流程 | 9 | 历史记录 |
| 已退回流程 | 4 | 历史记录 |
| 归档流程 | 10 | 含审批链 |
| 审核日志 | 8 | AI审核操作记录 |
| 定时任务日志 | 7 | 执行记录 |
| 归档日志 | 6 | 合规复核记录 |
| 审核规则配置 | 7 | 7种流程类型 |
| 归档复核配置 | 4 | 4种流程类型 |
| OA数据库连接 | 3 | 系统级配置 |
| AI模型 | 5 | 3本地+2云端 |
| 用户偏好配置 | 8 | 用户个性化 |
| 仪表盘Widget | 15 | 可定制卡片 |
