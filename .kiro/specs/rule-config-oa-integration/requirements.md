# 需求文档：规则配置与 OA 集成

## 简介

本功能将租户管理的规则配置页面和个人设置的审核工作台配置页面从前端模拟数据替换为真实后端 API 驱动。核心目标包括：建立按 OA 类型和 AI 模型类型区分的后端类体系（首发泛微 Ecology9）、实现流程测试连接、OA 字段拉取、审核规则持久化、AI 提示词配置、用户权限关联，以及个人设置与租户的多租户关联。同时统计租户 Token 用量，记录大模型消息日志。

## 术语表

- **Go_Service**：Go 语言业务中台服务，负责规则管理、OA 数据对接、租户管理等业务逻辑
- **AI_Service**：Python 智能引擎服务，负责 LLM 调用、RAG 检索等 AI 功能
- **OA_Adapter**：Go_Service 中按 OA 类型区分的适配器类，封装不同 OA 系统的查询逻辑（首发实现 Ecology9Adapter）
- **AI_Model_Caller**：Go_Service 中按 AI 模型部署类型区分的调用类，封装不同模型提供商的调用逻辑（首发实现 XinferenceCaller 和 AliyunBailianCaller）
- **Process_Audit_Config**：流程审核配置，租户级别的审核流程定义，包含字段配置、审核规则、AI 配置和用户权限
- **User_Personal_Config**：用户个性化配置，用户在个人设置中对审核工作台、定时任务、归档复盘的自定义偏好
- **Tenant_LLM_Message_Log**：租户大模型消息记录表，记录每次 AI 调用的 Token 消耗和调用详情
- **Strictness_Preset**：审核尺度预设，租户级别的严格/标准/宽松三级提示词预设配置
- **Ecology9**：泛微 E9 OA 系统，本功能首发适配的 OA 类型

## 需求

### 需求 1：OA 适配器类体系

**用户故事：** 作为开发者，我希望后端按 OA 类型建立独立的适配器类，以便针对不同 OA 系统封装各自的查询语法和数据映射逻辑，方便后续扩展。

#### 验收标准

1. THE Go_Service SHALL 定义 OA_Adapter 接口，包含流程验证、字段拉取、流程数据查询等方法签名
2. THE Go_Service SHALL 实现 Ecology9Adapter 作为首个 OA_Adapter 实现，封装泛微 E9 的 SQL 查询语法
3. WHEN 接收到包含 tenant_id 的 API 请求时，THE Go_Service SHALL 根据该租户关联的 OA 数据库连接的 oa_type 字段自动选择对应的 OA_Adapter 实现
4. IF 租户关联的 oa_type 没有对应的 OA_Adapter 实现，THEN THE Go_Service SHALL 返回错误码和描述信息，说明该 OA 类型暂不支持

### 需求 2：AI 模型调用类体系

**用户故事：** 作为开发者，我希望后端按 AI 模型部署类型建立独立的调用类，以便区分本地 Xinference 和云端阿里百炼等不同模型提供商的调用方式，并关联租户统计 Token 用量。

#### 验收标准

1. THE Go_Service SHALL 定义 AI_Model_Caller 接口，包含模型连接测试、消息发送、Token 统计等方法签名
2. THE Go_Service SHALL 实现 XinferenceCaller 用于本地部署模型的调用
3. THE Go_Service SHALL 实现 AliyunBailianCaller 用于阿里云百炼云端模型的调用
4. WHEN AI_Model_Caller 完成一次模型调用时，THE Go_Service SHALL 将本次调用消耗的 Token 数量累加到对应租户的 token_used 字段
5. WHEN AI_Model_Caller 完成一次模型调用时，THE Go_Service SHALL 在 Tenant_LLM_Message_Log 表中写入一条记录，包含租户 ID、模型 ID、输入 Token 数、输出 Token 数、调用时间和调用来源
6. THE Go_Service SHALL 根据 ai_model_configs 表的 deploy_type 字段自动选择对应的 AI_Model_Caller 实现

### 需求 3：流程测试连接

**用户故事：** 作为租户管理员，我希望在规则配置的基本信息中测试流程连接，以便验证该流程在 OA 系统中存在且配置正确。

#### 验收标准

1. WHEN 租户管理员点击"测试连接"按钮并提供流程类型名称时，THE Go_Service SHALL 通过对应的 OA_Adapter 查询 OA 数据库，验证该流程是否存在
2. WHEN 流程在 OA 数据库中存在且配置正确时，THE Go_Service SHALL 返回成功状态和流程基本信息（流程名称、主表名、明细表数量）
3. IF 流程在 OA 数据库中不存在，THEN THE Go_Service SHALL 返回失败状态和具体错误描述
4. IF OA 数据库连接不可用，THEN THE Go_Service SHALL 返回连接失败的错误信息
5. THE 规则配置页面 SHALL 在基本信息区域展示测试连接按钮和测试结果状态

### 需求 4：OA 字段拉取

**用户故事：** 作为租户管理员，我希望在字段配置时能从对应的 OA 系统拉取真实字段列表，以便准确配置需要审核的字段。

#### 验收标准

1. WHEN 租户管理员请求拉取字段时，THE Go_Service SHALL 通过对应的 OA_Adapter 从 OA 数据库查询该流程的全部字段定义
2. THE Go_Service SHALL 返回按主表和明细表分组的字段列表，每个字段包含字段标识、字段名称和字段类型
3. WHEN 选择"全部字段"传输模式时，THE 规则配置页面 SHALL 显示主表字段数、明细表数量和明细字段总数的统计提示
4. THE Go_Service SHALL 将拉取到的字段信息持久化到 process_audit_configs 表的 main_fields 和 detail_tables 字段中
5. IF OA 字段拉取失败，THEN THE Go_Service SHALL 返回错误信息，THE 规则配置页面 SHALL 展示错误提示并保留已有字段配置

### 需求 5：审核规则持久化

**用户故事：** 作为租户管理员，我希望审核规则能持久化到数据库中，以便规则配置在系统重启后不丢失，并支持按租户隔离。

#### 验收标准

1. THE Go_Service SHALL 提供审核规则的 CRUD API，所有操作自动关联当前租户 ID
2. WHEN 创建审核规则时，THE Go_Service SHALL 将规则存储到 audit_rules 表，包含规则内容、规则范围（mandatory/default_on/default_off）、优先级、来源和是否关联审批流
3. THE Go_Service SHALL 提供按流程类型查询审核规则列表的 API，支持按规则范围和启用状态筛选
4. WHEN 删除审核规则时，THE Go_Service SHALL 执行软删除或硬删除（根据规则来源决定），手动创建的规则可硬删除
5. THE Go_Service SHALL 确保所有审核规则操作通过租户中间件进行租户隔离，禁止跨租户访问

### 需求 6：流程审核配置 CRUD

**用户故事：** 作为租户管理员，我希望能创建、查看、编辑和删除流程审核配置，以便管理不同审批流程的审核策略。

#### 验收标准

1. THE Go_Service SHALL 提供流程审核配置的完整 CRUD API，包含基本信息、字段配置、知识库模式、AI 配置和用户权限
2. WHEN 创建流程审核配置时，THE Go_Service SHALL 验证同一租户下流程类型名称的唯一性
3. THE Go_Service SHALL 将 AI 配置（审核尺度、推理提示词、提取提示词）存储在 process_audit_configs 表的 ai_config JSONB 字段中
4. THE Go_Service SHALL 将用户权限配置（允许自定义字段、允许自定义规则、允许修改尺度）存储在 process_audit_configs 表的 user_permissions JSONB 字段中
5. THE 规则配置页面 SHALL 替换所有 mockProcessAuditConfigs 引用为真实 API 调用

### 需求 7：AI 提示词配置优化

**用户故事：** 作为租户管理员，我希望"编辑预设提示词"功能改为配置预设的系统提示词，变量部分的提示词作为首次提问的用户提示词，以便更精确地控制 AI 审核行为。

#### 验收标准

1. THE 规则配置页面 SHALL 将 AI 配置区域的"编辑预设提示词"标签修改为"预设系统提示词"
2. THE 规则配置页面 SHALL 将推理阶段提示词区域标注为"系统提示词"，将变量模板区域标注为"用户提示词（首次提问）"
3. THE Go_Service SHALL 在 process_audit_configs 的 ai_config 中区分存储 system_prompt 和 user_prompt_template 两个字段
4. WHEN Go_Service 调用 AI_Service 时，THE Go_Service SHALL 将 system_prompt 作为系统角色消息、user_prompt_template 渲染后作为用户角色消息发送

### 需求 8：审核尺度预设管理

**用户故事：** 作为租户管理员，我希望管理严格/标准/宽松三级审核尺度的预设提示词，以便统一控制不同审核严格程度下的 AI 行为。

#### 验收标准

1. THE Go_Service SHALL 提供审核尺度预设的查询和更新 API，按租户隔离
2. THE Go_Service SHALL 在 strictness_presets 表中为每个租户维护 strict、standard、loose 三条预设记录
3. WHEN 租户管理员修改审核尺度预设时，THE Go_Service SHALL 更新对应记录的 reasoning_instruction 和 extraction_instruction 字段
4. THE 规则配置页面 SHALL 替换 fetchStrictnessPresets 和 saveStrictnessPresets 的模拟实现为真实 API 调用

### 需求 9：用户权限与个人设置关联

**用户故事：** 作为租户管理员，我希望规则配置中的用户权限设置能正确关联到个人设置页面，以便业务用户在个人设置中只能操作被授权的配置项。

#### 验收标准

1. WHEN 租户管理员在规则配置中设置 allow_custom_fields 为 false 时，THE 个人设置页面 SHALL 禁用该流程的字段自定义功能，显示锁定状态
2. WHEN 租户管理员在规则配置中设置 allow_custom_rules 为 false 时，THE 个人设置页面 SHALL 禁用该流程的自定义规则添加功能
3. WHEN 租户管理员在规则配置中设置 allow_modify_strictness 为 false 时，THE 个人设置页面 SHALL 禁用该流程的审核尺度修改功能
4. THE 个人设置页面 SHALL 从后端 API 实时获取当前租户的流程审核配置及其用户权限设置，替换模拟数据

### 需求 10：个人设置流程双重校验

**用户故事：** 作为业务用户，我希望个人设置中的流程列表经过双重校验，以便确保我只能看到自己在 OA 中有审批权限且租户管理员已配置的流程。

#### 验收标准

1. WHEN 业务用户访问个人设置的审核工作台配置时，THE Go_Service SHALL 验证该用户在 OA 系统中是否具有该流程的审批权限
2. THE Go_Service SHALL 验证该流程是否已在当前租户的规则配置中完成配置
3. THE Go_Service SHALL 仅返回同时满足 OA 审批权限和租户配置两个条件的流程列表
4. IF 用户在 OA 中无某流程的审批权限，THEN THE Go_Service SHALL 从返回的流程列表中排除该流程
5. IF 某流程未在租户规则配置中配置，THEN THE Go_Service SHALL 从返回的流程列表中排除该流程

### 需求 11：个人设置审核字段配置

**用户故事：** 作为业务用户，我希望在个人设置中配置审核字段的传输模式，以便在租户管理员允许的范围内自定义审核字段。

#### 验收标准

1. WHILE 流程的字段配置未被租户管理员锁定时，THE 个人设置页面 SHALL 允许业务用户切换传输模式（全部字段/部分字段）
2. WHILE 流程的字段配置被租户管理员锁定（allow_custom_fields 为 false）时，THE 个人设置页面 SHALL 以只读模式展示字段配置，禁止修改
3. WHEN 业务用户修改字段配置时，THE Go_Service SHALL 将修改保存到 user_personal_configs 表的 audit_details JSONB 字段中
4. THE Go_Service SHALL 在审核执行时合并租户级字段配置和用户级字段覆盖，用户覆盖优先

### 需求 12：个人设置审核规则配置

**用户故事：** 作为业务用户，我希望在个人设置中管理审核规则的开关和私有规则，以便个性化审核行为。

#### 验收标准

1. THE 个人设置页面 SHALL 展示租户级审核规则列表，mandatory 规则显示为锁定状态不可关闭
2. WHILE allow_custom_rules 为 true 时，THE 个人设置页面 SHALL 允许业务用户添加、编辑和删除私有审核规则
3. THE 个人设置页面 SHALL 允许业务用户切换 default_on 和 default_off 规则的启用状态
4. WHEN 业务用户修改规则配置时，THE Go_Service SHALL 将规则开关覆盖和私有规则保存到 user_personal_configs 表
5. THE Go_Service SHALL 在审核执行时按优先级合并规则：租户强制规则 > 用户私有规则 > 租户默认规则

### 需求 13：个人设置审核尺度配置

**用户故事：** 作为业务用户，我希望在个人设置中调整审核尺度，以便根据个人需求选择严格、标准或宽松的审核模式。

#### 验收标准

1. WHILE allow_modify_strictness 为 true 时，THE 个人设置页面 SHALL 允许业务用户选择审核尺度（strict/standard/loose）
2. WHILE allow_modify_strictness 为 false 时，THE 个人设置页面 SHALL 以只读模式展示当前审核尺度，禁止修改
3. WHEN 业务用户修改审核尺度时，THE Go_Service SHALL 将尺度覆盖保存到 user_personal_configs 表的 audit_details 中
4. THE Go_Service SHALL 在审核执行时优先使用用户覆盖的审核尺度，未覆盖时使用租户级默认尺度

### 需求 14：多租户关联与切换

**用户故事：** 作为跨租户用户，我希望个人设置的审核工作台、定时任务和归档复盘配置与当前激活的租户关联，以便在切换租户后看到对应租户的配置。

#### 验收标准

1. THE 个人设置页面 SHALL 根据当前激活角色的 tenant_id 加载对应租户的配置数据
2. WHEN 用户切换角色（切换租户）后访问个人设置时，THE 个人设置页面 SHALL 重新加载新租户的流程配置和用户个性化配置
3. THE Go_Service SHALL 在 user_personal_configs 表中按 tenant_id + user_id 唯一约束存储用户配置，确保不同租户的配置互相隔离
4. THE Go_Service SHALL 确保定时任务和归档复盘的个人配置同样按租户隔离

### 需求 15：租户大模型消息记录

**用户故事：** 作为系统管理员，我希望记录每个租户的大模型调用详情，以便监控 Token 消耗和审计 AI 使用情况。

#### 验收标准

1. THE Go_Service SHALL 创建 tenant_llm_message_logs 表，包含租户 ID、用户 ID、模型配置 ID、请求类型、输入 Token 数、输出 Token 数、总 Token 数、调用耗时和调用时间
2. WHEN AI_Model_Caller 完成模型调用时，THE Go_Service SHALL 异步写入一条消息记录
3. THE Go_Service SHALL 提供按租户查询 Token 消耗统计的 API，支持按时间范围和模型筛选
4. THE Go_Service SHALL 确保消息记录的写入不阻塞主审核流程

### 需求 16：数据库迁移

**用户故事：** 作为开发者，我希望通过数据库迁移脚本创建规则配置相关的数据表，以便支持规则配置功能的持久化存储。

#### 验收标准

1. THE Go_Service SHALL 创建 000007 迁移文件，包含 process_audit_configs、audit_rules、strictness_presets 表
2. THE Go_Service SHALL 创建 000008 迁移文件，包含 cron_tasks、cron_task_type_configs 表
3. THE Go_Service SHALL 创建 000009 迁移文件，包含 audit_logs、cron_logs、archive_logs 表
4. THE Go_Service SHALL 创建 000010 迁移文件，包含 user_personal_configs、user_dashboard_prefs 表
5. THE Go_Service SHALL 创建 tenant_llm_message_logs 表的迁移文件
6. THE Go_Service SHALL 为所有迁移文件提供对应的 down 回滚脚本

### 需求 17：Go 与 Python 服务间 AI 调用协议

**用户故事：** 作为开发者，我希望明确 Go_Service 和 AI_Service 之间的 AI 调用职责分配和数据传输协议，以便两个服务协同完成审核任务。

#### 验收标准

1. THE Go_Service SHALL 负责规则组装、数据脱敏、提示词渲染和 Token 统计
2. THE AI_Service SHALL 负责 LLM 调用、RAG 检索和结果解析
3. WHEN Go_Service 调用 AI_Service 时，THE Go_Service SHALL 通过 HTTP API 发送包含系统提示词、用户提示词、模型配置和审核上下文的请求体
4. WHEN AI_Service 返回结果时，THE AI_Service SHALL 返回包含推理结果、Token 消耗统计和模型标识的响应体
5. THE Go_Service SHALL 在发送数据到 AI_Service 前对敏感字段（薪资、身份证号等）执行脱敏处理
