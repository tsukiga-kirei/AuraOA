# 需求文档：仪表盘角色差异化优化

## 简介

对 OA 智审平台的仪表盘（Overview）模块进行全面优化，针对三种角色（业务用户 business、租户管理员 tenant_admin、系统管理员 system_admin）提供差异化的数据展示与组件配置。核心变更包括：将数据源从日志表（audit_logs / archive_logs / cron_logs）切换为快照表（audit_process_snapshots / archive_process_snapshots）统计有效数据；引入图表 UI 库实现可视化；移除各角色不需要的组件；增强保留组件的数据维度；全面支持国际化。

## 术语表

- **Dashboard（仪表盘）**：系统首页概览页面，根据用户角色展示不同的统计组件
- **Widget（组件/小部件）**：仪表盘中的独立数据展示卡片，如审核概览、审核趋势等
- **Snapshot_Table（快照表）**：audit_process_snapshots 和 archive_process_snapshots，仅包含解析成功的有效结论记录
- **Valid_Data（有效数据）**：快照表中 valid_log_ids / valid_archive_log_ids 字段所引用的日志条目
- **Business_User（业务用户）**：角色为 business 的普通用户，仅查看个人在当前租户内的数据
- **Tenant_Admin（租户管理员）**：角色为 tenant_admin 的管理员，查看当前租户全部数据
- **System_Admin（系统管理员）**：角色为 system_admin 的平台管理员，查看全平台数据
- **Widget_Registry（组件注册表）**：前端 overviewWidgets.ts 中定义的组件元数据列表，包含角色权限映射
- **Chart_Library（图表库）**：用于渲染柱状图、饼图等可视化图表的前端开源库（如 ECharts 或 Chart.js）
- **Weekly_Overview（本周概览）**：替代原"今日审核概览"的新组件，统计本周（周一至当前）的快照数据
- **Stacked_Bar_Chart（堆叠柱状图）**：按日期分组、按状态堆叠的柱状图，用于审核趋势展示
- **LLM_Call_Type（LLM 调用类型）**：大模型调用的分类，包括推理调用（reasoning）和结构化调用（structured）
- **Page_Permission（页面访问权限）**：基于 RBAC 的页面级权限，控制用户可见的仪表盘组件

## 需求

### 需求 1：数据源切换 — 从日志表迁移到快照表

**用户故事：** 作为平台运营者，我希望仪表盘统计数据基于快照表的有效数据，以确保展示的数字准确反映实际有效的审核和归档复盘结论。

#### 验收标准

1. WHEN Dashboard 构建审核相关统计数据时，THE Dashboard_Service SHALL 从 audit_process_snapshots 表查询有效数据，以 valid_log_ids 数组长度作为有效审核条数
2. WHEN Dashboard 构建归档复盘相关统计数据时，THE Dashboard_Service SHALL 从 archive_process_snapshots 表查询有效数据，以 valid_archive_log_ids 数组长度作为有效归档复盘条数
3. WHEN Dashboard 构建定时任务执行次数时，THE Dashboard_Service SHALL 从 cron_logs 表查询本周执行记录的条数
4. THE Dashboard_Service SHALL 保留从 cron_logs 表查询定时任务数据的逻辑，因为当前不存在 cron_process_snapshots 表

### 需求 2：业务用户 — 权限与数据范围控制

**用户故事：** 作为业务用户，我希望仪表盘仅展示我个人在当前租户内的数据，并根据我的页面访问权限控制可见组件，以确保数据隔离和权限合规。

#### 验收标准

1. WHILE Business_User 访问 Dashboard 时，THE Dashboard_Service SHALL 仅返回该用户个人的快照数据（通过 user_id 过滤关联的审核日志）
2. WHILE Business_User 访问 Dashboard 时，THE Widget_Registry SHALL 根据该用户的 page_permissions 过滤可见组件，仅展示用户有权访问的页面对应的组件
3. THE Widget_Registry SHALL 从 Business_User 的可用组件列表中移除 archive_review 组件

### 需求 3：业务用户 — 本周概览组件

**用户故事：** 作为业务用户，我希望看到本周的审核概览而非今日概览，以便更好地了解本周的工作进展。

#### 验收标准

1. WHEN Business_User 查看 Weekly_Overview 组件时，THE Dashboard_Service SHALL 返回本周（周一 00:00 至当前时刻）的统计数据
2. THE Weekly_Overview 组件 SHALL 展示以下分项：审核工作台快照本周条数、归档复盘快照本周条数、定时任务本周执行次数
3. THE Weekly_Overview 组件 SHALL 在分项之前展示三项之和的总数
4. THE Weekly_Overview 组件 SHALL 替代原有的"今日审核概览"（audit_summary）组件

### 需求 4：业务用户 — 审核趋势组件优化

**用户故事：** 作为业务用户，我希望审核趋势按周排列并区分三种功能的状态，以便直观了解每天各类任务的完成情况。

#### 验收标准

1. WHEN Business_User 查看审核趋势组件时，THE Dashboard_Service SHALL 返回本周每天的快照数据，按审核工作台、定时任务、归档复盘三个功能分组
2. THE 审核趋势组件 SHALL 使用 Stacked_Bar_Chart 展示每天的数据，每天区分出三种功能各自的数量
3. THE 前端 SHALL 引入 Chart_Library（ECharts 或 Chart.js）来渲染 Stacked_Bar_Chart

### 需求 5：业务用户 — 定时任务组件优化

**用户故事：** 作为业务用户，我希望定时任务列表包含说明信息，以便快速理解每个任务的用途。

#### 验收标准

1. THE 定时任务组件 SHALL 在每个任务行中展示说明列，显示任务的具体用途描述（如"批量审核"、"审核日报推送"）
2. WHEN 定时任务没有自定义说明时，THE 定时任务组件 SHALL 显示任务类型的国际化翻译作为默认说明

### 需求 6：业务用户 — 最近动态组件优化

**用户故事：** 作为业务用户，我希望最近动态展示有效的快照数据并标注具体效果，以便快速了解每条动态的关键结果。

#### 验收标准

1. THE 最近动态组件 SHALL 最多展示 10 条有效数据记录
2. THE 最近动态组件 SHALL 按快照表数据截取前 10 条有效记录
3. WHEN 动态类型为审核工作台时，THE 最近动态组件 SHALL 标注该条记录的建议结论（recommendation）和评分（score）
4. WHEN 动态类型为归档复盘时，THE 最近动态组件 SHALL 标注该条记录的合规性结论（compliance）和合规评分（compliance_score）
5. WHEN 动态类型为定时任务时，THE 最近动态组件 SHALL 标注该条记录的执行状态和任务说明

### 需求 7：业务用户 — 待办任务组件优化

**用户故事：** 作为业务用户，我希望待办任务区分归档复盘和审核工作台两类待办，以便分别跟踪不同类型的待处理工作。

#### 验收标准

1. THE 待办任务组件 SHALL 分别展示归档复盘待办数量和审核工作台待办数量
2. THE Dashboard_Service SHALL 统计近 90 天内的待办任务数据
3. THE Dashboard_Service SHALL 根据用户的流程权限过滤待办任务，仅展示用户有权处理的待办

### 需求 8：租户管理员 — 数据范围与组件配置

**用户故事：** 作为租户管理员，我希望仪表盘展示当前租户的整体统计数据，并移除不需要的组件，以便高效掌握租户运营状况。

#### 验收标准

1. WHILE Tenant_Admin 访问 Dashboard 时，THE Dashboard_Service SHALL 返回当前租户全部用户的快照聚合数据
2. THE Widget_Registry SHALL 从 Tenant_Admin 的可用组件列表中移除以下组件：pending_tasks（待办任务）、cron_tasks（定时任务）、archive_review（归档复盘）、ai_performance（AI 模型表现）、tenant_usage（租户资源用量）
3. THE Tenant_Admin 的 Weekly_Overview 组件 SHALL 展示整个租户的本周概览数据（统计方式同业务用户，但范围为租户级）
4. THE Tenant_Admin 的审核趋势组件 SHALL 展示整个租户的趋势数据

### 需求 9：租户管理员 — 部门分布组件优化

**用户故事：** 作为租户管理员，我希望部门分布统计基于快照数据并区分三个功能模块，以便准确了解各部门的使用情况。

#### 验收标准

1. THE 部门分布组件 SHALL 基于快照表数据进行统计，而非日志表
2. THE 部门分布组件 SHALL 区分审核工作台、定时任务、归档复盘三个功能的各自数量
3. THE 部门分布组件 SHALL 使用 Chart_Library 渲染可视化图表

### 需求 10：租户管理员 — 最近动态与用户活跃排名

**用户故事：** 作为租户管理员，我希望最近动态和用户活跃排名基于快照有效数据，以确保统计的准确性。

#### 验收标准

1. THE Tenant_Admin 的最近动态组件 SHALL 截取前 10 条有效快照数据
2. THE 用户活跃排名组件 SHALL 基于快照数据统计每个用户的有效数据量
3. THE 用户活跃排名组件 SHALL 仅保留有效数据的用户排名

### 需求 11：系统管理员 — 组件配置与移除

**用户故事：** 作为系统管理员，我希望仪表盘聚焦于平台级运营指标，移除不相关的组件。

#### 验收标准

1. THE Widget_Registry SHALL 从 System_Admin 的可用组件列表中移除以下组件：audit_summary（审核概览）、weekly_trend（审核趋势）、archive_review（归档复盘）、recent_activity（最近动态）
2. THE System_Admin 的 Dashboard SHALL 保留以下组件：platform_tenant_stats（租户规模）、ai_performance（AI 模型表现）、tenant_usage（租户资源用量）、platform_tenant_ranking（租户审核排名）

### 需求 12：系统管理员 — AI 模型表现组件增强

**用户故事：** 作为系统管理员，我希望 AI 模型表现组件展示具体的模型信息和详细的调用分类统计，以便精确评估各模型的实际表现。

#### 验收标准

1. THE AI 模型表现组件 SHALL 展示每个 AI 模型的名称、版本等具体信息
2. THE AI 模型表现组件 SHALL 按模型分别展示成功率，成功率定义为一次完整调用（从请求到获得有效响应）的成功比例
3. THE AI 模型表现组件 SHALL 区分推理调用（reasoning）和结构化调用（structured）两种 LLM_Call_Type，分别统计调用次数、成功率和平均响应时间
4. THE Dashboard_Service SHALL 从 llm_message_logs 表按 model 字段和调用类型分组聚合统计数据

### 需求 13：系统管理员 — 租户资源用量组件优化

**用户故事：** 作为系统管理员，我希望租户资源用量按租户分别展示，而非显示全平台汇总，以便了解各租户的资源消耗情况。

#### 验收标准

1. THE 租户资源用量组件 SHALL 按租户分别展示每个租户的 Token 使用量和配额
2. THE 租户资源用量组件 SHALL 移除全平台汇总的展示方式

### 需求 14：系统管理员 — 租户规模组件增强

**用户故事：** 作为系统管理员，我希望租户规模组件展示每个租户下的人员数量，并明确活跃租户的判断标准。

#### 验收标准

1. THE 租户规模组件 SHALL 新增展示每个租户下的注册人员数量
2. THE Dashboard_Service SHALL 定义活跃租户的判断标准：近 30 天内有至少一条审核或归档复盘快照记录的租户视为活跃租户
3. THE 租户规模组件 SHALL 展示活跃租户的判断依据说明文案

### 需求 15：系统管理员 — 租户审核排名组件增强

**用户故事：** 作为系统管理员，我希望租户审核排名展示详细的按租户分开的使用统计，包括失败记录，以便全面评估各租户的使用情况。

#### 验收标准

1. THE 租户审核排名组件 SHALL 按租户分别展示详细的使用统计数据
2. THE 租户审核排名组件 SHALL 统计每个租户的审核快照数、归档复盘快照数、定时任务执行数
3. THE 租户审核排名组件 SHALL 统计每个租户的失败记录数（审核失败、归档复盘失败）
4. THE Dashboard_Service SHALL 从快照表和日志表联合查询，将失败记录纳入统计

### 需求 16：图表库集成

**用户故事：** 作为前端开发者，我希望引入一个成熟的图表 UI 库，以便高效实现仪表盘中的各类可视化图表。

#### 验收标准

1. THE 前端项目 SHALL 集成一个开源图表库（ECharts 或 Chart.js）用于渲染仪表盘图表
2. THE 图表库 SHALL 支持堆叠柱状图（Stacked_Bar_Chart）、饼图、水平条形图等常用图表类型
3. THE 图表组件 SHALL 支持响应式布局，适配不同的 Widget 尺寸（sm / md / lg）

### 需求 17：国际化支持

**用户故事：** 作为国际化用户，我希望仪表盘所有新增和修改的文案都支持中英文切换，以便在不同语言环境下使用。

#### 验收标准

1. THE 前端 SHALL 为所有新增的组件标题、标签、提示文案添加 zh-CN 和 en-US 两种语言的翻译
2. THE 前端 SHALL 为所有修改的组件文案更新对应的国际化翻译
3. WHEN 用户切换语言时，THE Dashboard SHALL 立即以新语言重新渲染所有组件文案

### 需求 18：后端 API 响应结构调整

**用户故事：** 作为前端开发者，我希望后端 API 返回的数据结构能够支持新的组件需求，以便前端正确渲染优化后的仪表盘。

#### 验收标准

1. THE Dashboard_Service 的 BuildOverview 方法 SHALL 返回包含本周概览（按审核/归档/定时任务分项）的数据结构
2. THE Dashboard_Service 的 BuildOverview 方法 SHALL 返回包含按日按功能分组的趋势数据，支持 Stacked_Bar_Chart 渲染
3. THE Dashboard_Service 的 BuildOverview 方法 SHALL 返回最近动态中每条记录的详细效果标注（评分、合规性等）
4. THE Dashboard_Service 的 BuildPlatformOverview 方法 SHALL 返回按模型和调用类型分组的 AI 性能数据
5. THE Dashboard_Service 的 BuildPlatformOverview 方法 SHALL 返回按租户分列的资源用量数据
6. THE Dashboard_Service 的 BuildPlatformOverview 方法 SHALL 返回包含人员数量的租户规模数据
7. THE Dashboard_Service 的 BuildPlatformOverview 方法 SHALL 返回包含失败记录的租户审核排名数据

### 需求 19：未来优化方向建议

**用户故事：** 作为产品负责人，我希望了解各角色仪表盘的后续优化方向，以便规划产品路线图。

#### 验收标准

1. THE 需求文档 SHALL 提供以下未来优化方向建议：
   - 业务用户：个人效率趋势图（周/月对比）、审核质量评分趋势、个人目标与完成度追踪、常用流程类型快捷入口
   - 租户管理员：租户级 SLA 达成率监控、部门间审核效率对比、异常流程预警通知、审核规则命中率统计
   - 系统管理员：系统运行状态监控（CPU/内存/磁盘）、API 接口响应时间与错误率、数据库连接池与慢查询监控、各租户 LLM Token 消耗趋势图、平台级安全事件告警
