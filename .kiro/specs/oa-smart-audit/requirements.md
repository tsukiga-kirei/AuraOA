# 需求文档：OA智审（流程智能审核平台）

## 简介

OA智审是一个面向 OA 审批流程的 SaaS / 多租户智能审核平台。通过"流程信息配置 + AI 大模型审核"的核心机制，实现对 OA 审批流程的个性化、自动化辅助审核。系统分为前台业务端（面向用户/审批者）和后台管理端（面向租户管理员及系统管理员），采用 Go + Python 混合架构，前端使用 Nuxt 3 + TypeScript + Ant Design Vue 构建现代化界面。

## 术语表

- **OA智审平台**：本系统的整体名称，即流程智能审核平台
- **租户（Tenant）**：使用本平台的组织单位，每个租户拥有独立的配置空间和数据隔离
- **审核工作台（Audit_Workbench）**：前台核心模块，用户日常审核操作的主界面
- **规则引擎（Rule_Engine）**：负责加载、合并、执行审核规则的后端组件
- **知识库（Knowledge_Base）**：包含规则库（结构化 Checklist）和制度库（RAG 文档）的配置集合
- **AI审核服务（AI_Audit_Service）**：Python 智能引擎，负责 LLM 交互、RAG 检索和 OCR 解析
- **Go业务服务（Go_Business_Service）**：Go 业务中台，负责鉴权、菜单、OA 轮询、任务调度、持久化
- **规则作用域（Rule_Scope）**：规则的可见范围，包括强制执行、默认开启、默认关闭三级
- **知识库模式（KB_Mode）**：知识库策略枚举，包含 Rules_Only、RAG_Only、Hybrid 三种模式
- **脱敏处理（Data_Masking）**：对敏感字段（薪资、身份证号等）进行正则替换的安全处理
- **审核快照（Audit_Snapshot）**：一次完整审核的全量记录，包含输入、配置、推理过程、结果和用户反馈
- **OA适配器（OA_Adapter）**：负责对接不同 OA 系统（首发泛微 E9）的数据库连接和表结构映射组件
- **Cron任务（Cron_Task）**：用户配置的定时批量审核任务
- **私有规则（Private_Rule）**：用户自定义的仅对自己可见的审核关注点
- **RBAC**：基于角色的访问控制模型，用于动态菜单和权限管理

## 需求

### 需求 1：用户认证与动态权限菜单

**用户故事：** 作为平台用户，我希望通过安全的认证机制登录系统，并根据我的角色看到对应的功能菜单，以便我只访问被授权的模块。

#### 验收标准

1. WHEN 用户提交有效的登录凭证, THE Go_Business_Service SHALL 验证凭证并返回包含角色信息的 JWT Token
2. WHEN 用户携带有效 Token 请求菜单, THE Go_Business_Service SHALL 根据 RBAC 模型返回该角色可访问的动态菜单列表
3. WHEN 用户的 Token 过期或无效, THE Go_Business_Service SHALL 返回 401 状态码并拒绝请求
4. WHEN 用户尝试访问未授权的资源, THE Go_Business_Service SHALL 返回 403 状态码
5. WHEN 前端接收到动态菜单数据, THE OA智审平台 SHALL 仅渲染用户有权访问的路由和菜单项

### 需求 2：智能待办审核工作台

**用户故事：** 作为审批者，我希望在工作台上看到待审核的 OA 流程，并获得 AI 辅助审核建议，以便我高效、准确地完成审批工作。

#### 验收标准

1. WHEN 用户打开审核工作台, THE Go_Business_Service SHALL 从 OA 数据库轮询并返回该用户的待办流程列表
2. WHEN 用户选择一条待办流程, THE 审核工作台 SHALL 在侧边栏展示该流程加载的所有审核规则清单
3. WHEN 审核规则执行完毕, THE 审核工作台 SHALL 直观展示每条规则的通过或不通过状态
4. WHEN 用户点击某条规则的推理结果, THE 审核工作台 SHALL 展开显示 AI 的推理依据，包括引用的制度原文或具体计算过程
5. WHEN AI审核完成, THE AI_Audit_Service SHALL 输出审核建议（通过、驳回或修改建议）
6. WHEN 用户确认审核建议, THE 审核工作台 SHALL 提供一键跳转至 OA 系统进行处理的功能

### 需求 3：用户个性化审核偏好

**用户故事：** 作为审批者，我希望能够自定义审核规则的开关和敏感度，并添加私有关注点，以便审核结果更贴合我的实际需求。

#### 验收标准

1. WHEN 管理员将某条规则设定为"可选", THE 审核工作台 SHALL 为用户提供该规则的开启/关闭 Toggle 开关
2. WHEN 用户添加一条私有规则（自定义 Prompt）, THE 规则引擎 SHALL 将该私有规则仅追加到该用户的审核请求中
3. WHEN 用户调整 AI 敏感度设置, THE AI_Audit_Service SHALL 在基准范围内按照用户选择的力度（如严格模式）执行审核
4. WHILE 规则合并执行中, THE 规则引擎 SHALL 按照"租户强制规则 > 用户私有规则 > 租户默认规则"的优先级顺序合并规则列表
5. WHEN 用户修改个性化偏好, THE Go_Business_Service SHALL 持久化该偏好配置并在下次审核时自动加载

### 需求 4：定时任务中心

**用户故事：** 作为审批者，我希望配置定时批量审核任务，并接收日报/周报摘要推送，以便我在固定时间集中处理待办事项。

#### 验收标准

1. WHEN 用户创建一个 Cron 任务（如"每天下午4点"）, THE Go_Business_Service SHALL 注册该定时任务并按配置的时间触发执行
2. WHEN 定时任务触发, THE Go_Business_Service SHALL 批量获取该用户的待办流程并调用 AI_Audit_Service 进行审核
3. WHEN 日报/周报推送时间到达, THE Go_Business_Service SHALL 汇总指定时间段内的待办流程及风险点，通过邮件或 IM 推送摘要
4. WHEN 用户查看历史推送记录, THE OA智审平台 SHALL 展示过往的推送记录及当时的审核快照
5. IF 定时任务执行失败, THEN THE Go_Business_Service SHALL 记录失败原因并在下一个调度周期重试

### 需求 5：归档流程复盘

**用户故事：** 作为审批者或合规人员，我希望对已完结的流程进行事后 AI 复核和全量历史检索，以便进行合规性评估和审计追溯。

#### 验收标准

1. WHEN 用户选择一条已归档流程进行复盘, THE AI_Audit_Service SHALL 对该流程执行事后合规性 AI 复核并输出合规率分析
2. WHEN 用户执行历史检索, THE OA智审平台 SHALL 支持按时间、部门、流程类型等条件检索所有 AI 审核记录
3. THE 审核快照 SHALL 包含完整的审核上下文：OA 表单输入、生效规则配置、AI 推理过程（思维链）、审核建议结果、用户采纳情况
4. WHEN 检索结果返回, THE OA智审平台 SHALL 展示每条记录的 Prompt 内容、AI 回复原文和用户采纳状态

### 需求 6：租户管理员 — 知识库与规则配置

**用户故事：** 作为租户管理员，我希望灵活配置知识库模式和审核规则分级，以便为组织定制最合适的审核策略。

#### 验收标准

1. WHEN 租户管理员为某个流程配置知识库模式, THE Go_Business_Service SHALL 保存该流程的 KB_Mode 配置（Rules_Only、RAG_Only 或 Hybrid）
2. WHEN KB_Mode 为 Rules_Only, THE AI_Audit_Service SHALL 仅使用结构化 Checklist 进行审核
3. WHEN KB_Mode 为 RAG_Only, THE AI_Audit_Service SHALL 仅基于挂载的 PDF/Word 制度文档通过 RAG 检索进行审核
4. WHEN KB_Mode 为 Hybrid, THE AI_Audit_Service SHALL 同时启用规则库和制度库进行双重校验
5. WHEN 租户管理员设置规则属性, THE Go_Business_Service SHALL 将规则标记为强制执行（Mandatory）、默认开启或默认关闭
6. WHEN 规则被标记为强制执行, THE 审核工作台 SHALL 锁定展示该规则，用户不可修改
7. WHEN 租户管理员配置日志留存策略, THE Go_Business_Service SHALL 按照配置的保留时长（如永久保存或保存3年）管理 AI 推理日志和用户偏好记录

### 需求 7：系统管理员 — 平台级管理

**用户故事：** 作为系统管理员，我希望管理租户资源、OA 集成和全局监控，以便保障平台的稳定运行和资源合理分配。

#### 验收标准

1. WHEN 系统管理员创建新租户, THE Go_Business_Service SHALL 初始化租户的独立配置空间和数据隔离环境
2. WHEN 系统管理员设置 Token 配额, THE Go_Business_Service SHALL 限制该租户的 AI 调用次数，超出配额时拒绝新的审核请求
3. WHEN 系统管理员配置 OA 集成, THE OA_Adapter SHALL 建立 JDBC 连接并完成数据库表结构映射（首发适配泛微 E9）
4. THE OA智审平台 SHALL 在全局监控面板展示系统健康度、模型响应时间和 API 调用成功率
5. WHEN 系统管理员调整并发数控制, THE Go_Business_Service SHALL 按配置限制同时处理的审核请求数量

### 需求 8：OA 数据集成与适配

**用户故事：** 作为系统管理员，我希望平台能够灵活对接不同版本的 OA 系统，以便支持多种 OA 环境下的流程数据获取。

#### 验收标准

1. WHEN OA_Adapter 连接泛微 E9 数据库, THE OA_Adapter SHALL 通过 JDBC 读取流程表单数据并映射为平台统一的数据结构
2. WHEN 需要适配新版本 OA, THE OA_Adapter SHALL 通过加载对应版本的适配脚本完成表结构映射，无需修改核心代码
3. WHEN OA 数据库连接中断, THE OA_Adapter SHALL 记录错误日志并在配置的重试间隔后自动重连
4. THE Go_Business_Service SHALL 支持按目录、具体路径或流程 ID 选择需要 AI 介入审核的流程

### 需求 9：AI 审核引擎与 RAG 管线

**用户故事：** 作为平台用户，我希望 AI 审核引擎能够准确地基于规则和制度文档进行智能审核，以便获得可靠的审核建议。

#### 验收标准

1. WHEN AI_Audit_Service 接收到审核请求, THE AI_Audit_Service SHALL 根据流程的 KB_Mode 配置选择对应的 LangChain 链（Checklist Chain、Retrieval Chain 或并行混合）
2. WHEN 使用 Checklist Chain, THE AI_Audit_Service SHALL 逐条执行结构化规则并返回每条规则的通过/不通过结果
3. WHEN 使用 Retrieval Chain, THE AI_Audit_Service SHALL 从向量库中检索相关制度文档片段，并基于检索结果生成审核意见
4. WHEN 使用并行混合模式, THE AI_Audit_Service SHALL 同时执行 Checklist Chain 和 Retrieval Chain，合并两者结果后输出综合审核建议
5. THE AI_Audit_Service SHALL 在每次审核中记录完整的推理过程（思维链），包括引用的规则或文档片段
6. WHEN AI 配置变更（模型选型、Prompt 模板、上下文窗口大小）, THE AI_Audit_Service SHALL 在下次审核请求时使用更新后的配置

### 需求 10：数据安全与脱敏

**用户故事：** 作为平台运营者，我希望敏感数据在传输到 AI 层之前被脱敏处理，以便保障用户隐私和数据安全。

#### 验收标准

1. WHEN Go_Business_Service 向 AI_Audit_Service 发送审核数据, THE Go_Business_Service SHALL 对敏感字段（薪资、身份证号等）执行正则脱敏处理
2. THE Data_Masking 组件 SHALL 支持可配置的脱敏规则，包括字段匹配模式和替换策略
3. IF 脱敏处理过程中发现无法识别的敏感字段格式, THEN THE Go_Business_Service SHALL 记录警告日志并使用默认的全遮蔽策略
4. THE 审核快照 SHALL 存储脱敏后的数据版本，确保日志中不包含原始敏感信息

### 需求 11：审核记录不可篡改与审计导出

**用户故事：** 作为合规人员，我希望所有审核记录不可篡改且支持导出，以便满足审计合规要求。

#### 验收标准

1. WHEN 一次审核完成, THE Go_Business_Service SHALL 将完整的审核快照（输入、配置、推理过程、结果、用户反馈）写入日志库（ES/MongoDB）
2. THE 审核快照 SHALL 采用追加写入模式，已写入的记录不可修改或删除
3. WHEN 合规人员请求审计导出, THE OA智审平台 SHALL 按指定条件导出审核记录为标准格式文件
4. THE 审核快照 SHALL 包含时间戳和操作者标识，确保每条记录可追溯

### 需求 12：可观测性与监控

**用户故事：** 作为系统管理员，我希望平台提供跨服务链路追踪和关键指标监控，以便及时发现和定位系统问题。

#### 验收标准

1. THE OA智审平台 SHALL 在 Go_Business_Service 和 AI_Audit_Service 之间实现跨服务链路追踪，每个请求携带唯一的 Trace ID
2. THE OA智审平台 SHALL 记录并展示 API 调用成功率指标
3. THE OA智审平台 SHALL 记录并展示模型响应时间指标
4. WHEN 某项指标超出预设阈值, THE OA智审平台 SHALL 触发告警通知

### 需求 13：前端现代化体验

**用户故事：** 作为平台用户，我希望前端界面现代化、美观且响应迅速，以便获得流畅的使用体验。

#### 验收标准

1. THE OA智审平台 SHALL 使用 Nuxt 3 + TypeScript + Ant Design Vue 构建前端应用，支持服务端渲染（SSR）以提升首屏加载速度
2. THE OA智审平台 SHALL 基于 RBAC 模型实现动态路由，根据用户角色自动加载对应的菜单和页面
3. THE 审核工作台 SHALL 提供响应式布局，适配桌面端和平板端的不同屏幕尺寸
4. WHEN 审核数据加载中, THE OA智审平台 SHALL 展示骨架屏或加载动画，避免页面空白
5. THE OA智审平台 SHALL 支持暗色模式和亮色模式切换
