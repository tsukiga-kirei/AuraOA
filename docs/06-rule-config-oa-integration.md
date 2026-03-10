# OA 智审平台 — 规则配置与 OA 集成 详细设计文档

> 文档版本：v1.0 | 更新日期：2026-03-09  
> 本文档详细描述"规则配置与 OA 集成"功能的完整技术实现，涵盖 OA 适配器、AI 模型调用、规则管理、用户个性化配置、数据脱敏、Go↔Python 服务间协议等核心模块。

---

## 一、功能概述

"规则配置与 OA 集成"是 OA 智审平台第一阶段的核心功能模块，将前端规则配置页面和个人设置页面从模拟数据驱动切换为真实后端 API 驱动。

### 1.1 核心能力

| 能力 | 说明 |
|------|------|
| OA 适配器体系 | Go 层按 OA 类型封装查询逻辑，首发泛微 Ecology9，支持流程验证、字段拉取 |
| AI 模型调用体系 | Go 层按 provider 封装调用逻辑，统一使用 OpenAI 兼容协议，支持本地部署（xinference/ollama/vllm）和云端 API（阿里百炼/DeepSeek/智谱/OpenAI/Azure OpenAI） |
| 流程审核配置 CRUD | 租户级流程审核配置的完整生命周期管理 |
| 审核规则持久化 | 规则的创建、查询、更新、删除，按租户隔离 |
| 审核尺度预设 | 三级预设（strict / standard / loose），租户可自定义推理和提取指令 |
| 用户个人配置 | 业务用户在租户管理员授权范围内的个性化配置 |
| 规则合并引擎 | 多优先级规则合并：mandatory > custom > toggle > defaults |
| 数据脱敏 | Go 层对敏感数据（身份证、手机号、银行卡、薪资）脱敏后再发送至 AI 层 |
| Go↔Python 协议 | 明确两个服务的职责边界和数据传输格式 |
| Token 配额管理 | 租户级 Token 用量统计、配额检查、异步日志写入 |

### 1.2 设计决策

| 决策 | 选择 | 理由 |
|------|------|------|
| OA 适配器模式 | Go interface + 工厂函数 | 符合现有代码风格，易于扩展新 OA 类型 |
| AI 调用路由 | Go 层统一入口，按 provider 分发，统一 OpenAI 兼容协议 | Go 负责鉴权/脱敏/Token 统计，Python 专注 LLM 调用；所有 provider 共用 OpenAICompatCaller |
| 字段配置存储 | JSONB 列 | 字段结构灵活多变，JSONB 避免频繁 DDL |
| 用户配置隔离 | tenant_id + user_id 联合唯一约束 | 确保跨租户配置互不干扰 |
| Token 日志写入 | 异步 goroutine | 不阻塞主审核流程 |
| 前端数据迁移 | 逐页替换 mock 引用为 composable API 调用 | 渐进式迁移，降低风险 |


---

## 二、系统架构

### 2.1 整体架构

```
┌─────────────────────────────────────────────────────────────────┐
│                     前端 (Nuxt 3 + Ant Design Vue)               │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────────────┐   │
│  │ rules.vue    │  │ settings.vue │  │ user-configs.vue     │   │
│  │ (规则配置)    │  │ (个人设置)    │  │ (用户配置管理)        │   │
│  └──────┬───────┘  └──────┬───────┘  └──────────┬───────────┘   │
│         │                 │                      │               │
│  ┌──────┴───────┐  ┌──────┴───────┐              │               │
│  │ useRulesApi  │  │useSettingsApi│              │               │
│  └──────┬───────┘  └──────┬───────┘              │               │
└─────────┼─────────────────┼──────────────────────┼───────────────┘
          │    HTTP / REST   │                      │
┌─────────┼─────────────────┼──────────────────────┼───────────────┐
│         ▼                 ▼                      ▼               │
│  ┌─────────────────────────────────────────────────────────┐     │
│  │              Gin Router + JWT + TenantContext            │     │
│  └─────────────────────────┬───────────────────────────────┘     │
│                            │                                     │
│  ┌─────────────────────────┼───────────────────────────────┐     │
│  │                   Handler 层 (6 个)                      │     │
│  │  ProcessAuditConfig │ AuditRule │ StrictnessPreset       │     │
│  │  UserPersonalConfig │ UserConfigMgmt │ LLMMessageLog     │     │
│  └─────────────────────────┬───────────────────────────────┘     │
│                            │                                     │
│  ┌─────────────────────────┼───────────────────────────────┐     │
│  │                   Service 层                             │     │
│  │  ConfigService │ RuleService │ PresetService             │     │
│  │  UserConfigService │ AICallerService │ RuleMerge         │     │
│  │  PromptBuilder │ LLMLogService                           │     │
│  └──────┬──────────────────┼──────────────────┬────────────┘     │
│         │                  │                  │                   │
│  ┌──────┴──────┐  ┌───────┴───────┐  ┌───────┴──────────┐       │
│  │ Repository  │  │  OA Adapter   │  │  AI Caller       │       │
│  │ 层 (6 个)   │  │  (适配器层)    │  │  (调用器层)       │       │
│  └──────┬──────┘  └───────┬───────┘  └───────┬──────────┘       │
│         │                 │                  │                   │
│    Go 业务中台             │                  │                   │
└─────────┼─────────────────┼──────────────────┼───────────────────┘
          │                 │                  │
          ▼                 ▼                  ▼
   ┌────────────┐   ┌────────────┐   ┌──────────────────┐
   │ PostgreSQL │   │ OA 数据库   │   │ Python AI 引擎   │
   │ (业务数据)  │   │ (泛微 E9)  │   │ (LLM / RAG)     │
   └────────────┘   └────────────┘   └──────────────────┘
```

### 2.2 请求流程

#### 流程测试连接

```
前端 → POST /api/tenant/rules/configs/test-connection
     → Go Router + JWT + TenantContext 中间件
     → ProcessAuditConfigService.TestConnection()
     → OAAdapterFactory.NewOAAdapter("weaver_e9", conn)
     → Ecology9Adapter.ValidateProcess(processType)
     → OA 数据库: SELECT FROM workflow_base WHERE ...
     → 返回 ProcessInfo{name, mainTable, detailCount}
```

#### AI 审核调用

```
前端 → POST /api/tenant/audit/execute
     → Go Router + JWT + TenantContext 中间件
     → AuditService.Execute()
     → 1. 组装规则 (MergeRules)
     → 2. 数据脱敏 (SanitizeText)
     → 3. 渲染提示词 (PromptBuilder)
     → 4. 调用 AI (AIModelCallerService.Chat / ChatViaPython)
     → 5. 异步写入 tenant_llm_message_logs
     → 6. 更新 tenants.token_used
     → 返回审核结果
```


---

## 三、OA 适配器体系

### 3.1 接口定义

位置：`go-service/internal/pkg/oa/adapter.go`

OAAdapter 接口定义了所有 OA 系统适配器必须实现的四个方法：

| 方法 | 说明 | 返回值 |
|------|------|--------|
| `ValidateProcess(ctx, processType)` | 验证流程类型是否存在于 OA 系统中 | `*ProcessInfo` |
| `FetchFields(ctx, processType)` | 拉取指定流程的全部字段定义（主表 + 明细表） | `*ProcessFields` |
| `CheckUserPermission(ctx, userID, processType)` | 检查用户在 OA 中是否具有指定流程的审批权限 | `bool` |
| `FetchProcessData(ctx, processID)` | 拉取指定流程实例的业务数据 | `*ProcessData` |

### 3.2 数据结构

```
ProcessInfo
├── ProcessType   string   // 流程类型标识
├── ProcessName   string   // 流程名称
├── MainTable     string   // 主表名
└── DetailCount   int      // 明细表数量

ProcessFields
├── MainFields    []FieldDef        // 主表字段列表
└── DetailTables  []DetailTableDef  // 明细表列表
    ├── TableName   string
    ├── TableLabel  string
    └── Fields      []FieldDef
        ├── FieldKey   string
        ├── FieldName  string
        └── FieldType  string

ProcessData
├── ProcessID   string                    // 流程实例 ID
├── MainData    map[string]interface{}    // 主表数据
└── DetailData  []map[string]interface{}  // 明细表数据
```

### 3.3 泛微 Ecology9 适配器

位置：`go-service/internal/pkg/oa/ecology9.go`

Ecology9Adapter 通过 GORM 连接泛微 E9 的 MySQL 数据库，封装 E9 特有的表结构查询：

| E9 表名 | 用途 |
|---------|------|
| `workflow_base` | 流程基础信息表 |
| `workflow_billfield` | 流程字段定义表 |
| `workflow_detail_table` | 明细表定义 |

连接管理：通过 `OADatabaseConnection` 模型获取连接参数，使用 `gorm.io/driver/mysql` 建立独立连接池。

### 3.4 工厂函数

位置：`go-service/internal/pkg/oa/factory.go`

```go
func NewOAAdapter(oaType string, conn *model.OADatabaseConnection) (OAAdapter, error)
```

工厂函数在创建适配器前会执行两级校验：
1. 校验 `oaType` 是否在 `supportedDrivers` 注册表中
2. 校验 `conn.Driver` 是否在该 OA 类型的支持驱动列表中

当前支持的 OA 类型及数据库驱动：

| oa_type | 适配器 | 支持的数据库驱动 | 说明 |
|---------|--------|-----------------|------|
| `weaver_e9` | `Ecology9Adapter` | `mysql`, `oracle` | 泛微 Ecology9 |

扩展新 OA 类型需要：(1) 在 `supportedDrivers` 中注册 OA 类型及其支持的驱动列表，(2) 实现 `OAAdapter` 接口，(3) 在工厂函数 switch 中注册 case。


---

## 四、AI 模型调用体系

### 4.1 接口定义

位置：`go-service/internal/pkg/ai/caller.go`

AIModelCaller 接口定义了所有 AI 模型调用器必须实现的两个方法：

| 方法 | 说明 | 返回值 |
|------|------|--------|
| `TestConnection(ctx)` | 测试模型连接是否可用 | `error` |
| `Chat(ctx, req)` | 发送对话请求，返回模型响应和 Token 消耗 | `*ChatResponse` |

### 4.2 请求/响应结构

```
ChatRequest
├── SystemPrompt   string             // 系统提示词
├── UserPrompt     string             // 用户提示词
├── ModelConfig    *model.AIModelConfig // 模型配置（不序列化）
├── Temperature    float64            // 温度参数
└── MaxTokens      int                // 最大 Token 数

ChatResponse
├── Content        string             // 模型返回内容
├── TokenUsage     TokenUsage         // Token 消耗统计
│   ├── InputTokens   int
│   ├── OutputTokens  int
│   └── TotalTokens   int
├── ModelID        string             // 模型标识
└── DurationMs     int64              // 调用耗时（毫秒）
```

### 4.3 调用器实现

#### OpenAICompatCaller（统一调用器）

位置：`go-service/internal/pkg/ai/openai_compat.go`

所有 provider 统一使用 OpenAI Chat Completions API 兼容协议，通过 `OpenAICompatCaller` 实现。

**本地部署 (deploy_type=local)**

| Provider | 必填配置 | 说明 |
|----------|---------|------|
| `xinference` | endpoint | 本地 Xinference 部署 |
| `ollama` | endpoint | 本地 Ollama 部署 |
| `vllm` | endpoint | 本地 vLLM 部署 |

**云端 API (deploy_type=cloud)**

| Provider | 必填配置 | 默认 Endpoint | 说明 |
|----------|---------|---------------|------|
| `aliyun_bailian` | api_key | `https://dashscope.aliyuncs.com/compatible-mode/v1` | 阿里云百炼 |
| `deepseek` | api_key | `https://api.deepseek.com/v1` | DeepSeek |
| `zhipu` | api_key | `https://open.bigmodel.cn/api/paas/v4` | 智谱 AI |
| `openai` | api_key | `https://api.openai.com/v1` | OpenAI |
| `azure_openai` | api_key, endpoint | 无默认值 | Azure OpenAI（endpoint 格式特殊） |

> 云端 provider 如果 `ai_model_configs.endpoint` 有值则优先使用配置值，否则使用上表中的默认 Endpoint。

### 4.4 工厂函数

位置：`go-service/internal/pkg/ai/factory.go`

```go
func NewAIModelCaller(cfg *model.AIModelConfig) (AIModelCaller, error)
```

按 `cfg.Provider` 路由，所有 provider 均返回 `OpenAICompatCaller` 实例：

| Provider | deploy_type | 校验规则 |
|----------|-------------|---------|
| `xinference` / `ollama` / `vllm` | local | 必须配置 endpoint |
| `aliyun_bailian` / `deepseek` / `zhipu` / `openai` | cloud | 必须配置 api_key；endpoint 可选（有默认值） |
| `azure_openai` | cloud | 必须配置 api_key 和 endpoint |

### 4.5 AIModelCallerService

位置：`go-service/internal/service/ai_caller_service.go`

AIModelCallerService 是 AI 调用的业务编排层，封装了完整的调用流程：

```
调用流程：
1. 检查租户 Token 配额 (token_used < token_quota)
2. 通过工厂函数创建 AIModelCaller 实例
3. 执行 AI 调用 (Chat / ChatViaPython)
4. 累加租户 Token 用量 (GORM Expr: token_used + total_tokens)
5. 异步写入 tenant_llm_message_logs (goroutine)
```

提供两种调用模式：

| 方法 | 说明 | 适用场景 |
|------|------|---------|
| `Chat()` | Go 层直接调用 AI 模型 | 第一阶段：仅规则库模式 |
| `ChatViaPython()` | 通过 HTTP 调用 Python AI 服务 | 第二阶段：RAG / 混合模式 |


---

## 五、数据库设计

### 5.1 迁移文件清单

基于现有 000001–000006 迁移文件，新增以下 5 个迁移：

| 迁移编号 | 文件名 | 新增表 |
|---------|--------|--------|
| 000007 | `audit_configs_rules_presets` | process_audit_configs, audit_rules, strictness_presets |
| 000008 | `cron_tasks` | cron_tasks, cron_task_type_configs |
| 000009 | `audit_cron_archive_logs` | audit_logs, cron_logs, archive_logs |
| 000010 | `user_personal_configs` | user_personal_configs, user_dashboard_prefs |
| 000011 | `tenant_llm_message_logs` | tenant_llm_message_logs |

### 5.2 表结构详解

#### process_audit_configs — 流程审核配置表

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | UUID | PK, DEFAULT gen_random_uuid() | 主键 |
| tenant_id | UUID | NOT NULL, FK → tenants(id) CASCADE | 租户 ID |
| process_type | VARCHAR(200) | NOT NULL | 流程类型标识 |
| process_type_label | VARCHAR(200) | DEFAULT '' | 流程类型显示名 |
| main_table_name | VARCHAR(200) | DEFAULT '' | OA 主表名 |
| main_fields | JSONB | NOT NULL, DEFAULT '[]' | 主表字段定义 |
| detail_tables | JSONB | NOT NULL, DEFAULT '[]' | 明细表定义 |
| field_mode | VARCHAR(20) | NOT NULL, DEFAULT 'all' | 字段模式 (all/selected) |
| kb_mode | VARCHAR(20) | NOT NULL, DEFAULT 'rules_only' | 知识库模式 |
| ai_config | JSONB | NOT NULL, DEFAULT '{}' | AI 配置 |
| user_permissions | JSONB | NOT NULL, DEFAULT '{}' | 用户权限控制 |
| status | VARCHAR(20) | NOT NULL, DEFAULT 'active' | 状态 |
| created_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() | 创建时间 |
| updated_at | TIMESTAMPTZ | NOT NULL, DEFAULT now() | 更新时间 |

唯一约束：`UNIQUE(tenant_id, process_type)`

#### audit_rules — 审核规则表

| 列名 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | UUID | PK | 主键 |
| tenant_id | UUID | NOT NULL, FK → tenants(id) CASCADE | 租户 ID |
| config_id | UUID | FK → process_audit_configs(id) CASCADE | 关联配置 |
| process_type | VARCHAR(200) | NOT NULL | 流程类型 |
| rule_content | TEXT | NOT NULL | 规则内容 |
| rule_scope | VARCHAR(20) | NOT NULL, DEFAULT 'default_on' | 规则范围 |
| priority | INT | NOT NULL, DEFAULT 0 | 优先级 |
| enabled | BOOLEAN | NOT NULL, DEFAULT TRUE | 是否启用 |
| source | VARCHAR(20) | NOT NULL, DEFAULT 'manual' | 来源 (manual/file_import) |
| related_flow | BOOLEAN | NOT NULL, DEFAULT FALSE | 是否关联流程 |

rule_scope 取值：

| 值 | 说明 | 用户可控 |
|----|------|---------|
| `mandatory` | 强制规则，始终生效 | 不可关闭 |
| `default_on` | 默认开启，用户可关闭 | 可 toggle |
| `default_off` | 默认关闭，用户可开启 | 可 toggle |

#### strictness_presets — 审核尺度预设表

| 列名 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| tenant_id | UUID | 租户 ID |
| strictness | VARCHAR(20) | 尺度级别 (strict/standard/loose) |
| reasoning_instruction | TEXT | 推理指令 |
| extraction_instruction | TEXT | 提取指令 |

唯一约束：`UNIQUE(tenant_id, strictness)`  
每个租户自动初始化 strict、standard、loose 三条记录。

#### cron_tasks — 定时任务表

| 列名 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| tenant_id | UUID | 租户 ID |
| task_type | VARCHAR(50) | 任务类型 |
| task_label | VARCHAR(200) | 任务显示名 |
| cron_expression | VARCHAR(100) | Cron 表达式 |
| is_active | BOOLEAN | 是否激活 |
| is_builtin | BOOLEAN | 是否内置 |
| push_email | VARCHAR(255) | 推送邮箱 |
| last_run_at | TIMESTAMPTZ | 上次执行时间 |
| next_run_at | TIMESTAMPTZ | 下次执行时间 |
| success_count | INT | 成功次数 |
| fail_count | INT | 失败次数 |

#### audit_logs — 审核日志表

| 列名 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| tenant_id | UUID | 租户 ID |
| user_id | UUID | 操作用户 |
| process_id | VARCHAR(100) | 流程实例 ID |
| title | VARCHAR(500) | 流程标题 |
| process_type | VARCHAR(200) | 流程类型 |
| recommendation | VARCHAR(20) | 审核建议 |
| score | INT | 审核评分 |
| audit_result | JSONB | 审核结果详情 |
| duration_ms | INT | 审核耗时 |

#### user_personal_configs — 用户个人配置表

| 列名 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| tenant_id | UUID | 租户 ID |
| user_id | UUID | 用户 ID |
| audit_details | JSONB | 审核个性化配置 |
| cron_details | JSONB | 定时任务个性化配置 |
| archive_details | JSONB | 归档个性化配置 |

唯一约束：`UNIQUE(tenant_id, user_id)`

#### tenant_llm_message_logs — 租户大模型消息记录表

| 列名 | 类型 | 说明 |
|------|------|------|
| id | UUID | 主键 |
| tenant_id | UUID | 租户 ID |
| user_id | UUID | 调用用户 |
| model_config_id | UUID | AI 模型配置 ID |
| request_type | VARCHAR(50) | 请求类型 (audit) |
| input_tokens | INT | 输入 Token 数 |
| output_tokens | INT | 输出 Token 数 |
| total_tokens | INT | 总 Token 数 |
| duration_ms | INT | 调用耗时 |
| created_at | TIMESTAMPTZ | 创建时间 |


### 5.3 JSONB 字段结构说明

#### process_audit_configs.ai_config

```json
{
    "audit_strictness": "standard",
    "system_prompt": "你是一个专业的审核助手...",
    "user_prompt_template": "请审核以下{{process_type}}流程...\n字段数据：{{fields}}\n审核规则：{{rules}}",
    "reasoning_instruction": "请逐条检查规则...",
    "extraction_instruction": "请提取关键信息..."
}
```

#### process_audit_configs.user_permissions

```json
{
    "allow_custom_fields": true,
    "allow_custom_rules": true,
    "allow_modify_strictness": false
}
```

三个权限标志控制业务用户可修改的配置范围：

| 标志 | 说明 | 为 false 时 |
|------|------|------------|
| `allow_custom_fields` | 允许用户自定义字段覆盖 | 拒绝修改 field_overrides |
| `allow_custom_rules` | 允许用户添加私有规则 | 拒绝修改 custom_rules |
| `allow_modify_strictness` | 允许用户修改审核尺度 | 拒绝修改 strictness_override |

#### user_personal_configs.audit_details

```json
[
    {
        "process_type": "采购审批",
        "custom_rules": [
            {"id": "uuid", "content": "金额超过5万需附合同", "enabled": true}
        ],
        "field_overrides": ["合同编号", "供应商名称"],
        "strictness_override": "strict",
        "rule_toggle_overrides": [
            {"rule_id": "uuid", "enabled": false}
        ]
    }
]
```

### 5.4 ER 关系图

```
tenants ──1:N──> process_audit_configs ──1:N──> audit_rules
   │
   ├──1:3──> strictness_presets (strict / standard / loose)
   │
   ├──1:N──> user_personal_configs <──N:1── users
   │
   ├──1:N──> tenant_llm_message_logs <──N:1── users
   │                                  <──N:1── ai_model_configs
   │
   ├──1:N──> cron_tasks ──1:N──> cron_logs
   │
   ├──1:N──> audit_logs <──N:1── users
   │
   └──1:N──> archive_logs <──N:1── users
```


---

## 六、API 接口设计

所有接口均需 JWT 认证，租户级接口额外需要 TenantContext 中间件注入 tenant_id。

### 6.1 流程审核配置 API（tenant_admin）

路由前缀：`/api/tenant/rules`

| 方法 | 路径 | 说明 | Handler |
|------|------|------|---------|
| GET | `/configs` | 查询当前租户的流程审核配置列表 | configHandler.List |
| POST | `/configs` | 创建流程审核配置 | configHandler.Create |
| GET | `/configs/:id` | 查询单个配置详情 | configHandler.GetByID |
| PUT | `/configs/:id` | 更新流程审核配置 | configHandler.Update |
| DELETE | `/configs/:id` | 删除流程审核配置 | configHandler.Delete |
| POST | `/configs/test-connection` | 测试 OA 流程连接 | configHandler.TestConnection |
| POST | `/configs/:id/fetch-fields` | 拉取 OA 字段定义 | configHandler.FetchFields |

创建请求体 (CreateProcessAuditConfigRequest)：

```json
{
    "process_type": "采购审批",
    "process_type_label": "采购审批流程",
    "main_table_name": "formtable_main_1",
    "main_fields": [...],
    "detail_tables": [...],
    "field_mode": "all",
    "kb_mode": "rules_only",
    "ai_config": {...},
    "user_permissions": {...},
    "status": "active"
}
```

### 6.2 审核规则 API（tenant_admin）

路由前缀：`/api/tenant/rules`

| 方法 | 路径 | 说明 | Handler |
|------|------|------|---------|
| GET | `/audit-rules` | 查询审核规则列表（支持按 process_type 筛选） | ruleHandler.List |
| POST | `/audit-rules` | 创建审核规则 | ruleHandler.Create |
| PUT | `/audit-rules/:id` | 更新审核规则 | ruleHandler.Update |
| DELETE | `/audit-rules/:id` | 删除审核规则（manual=硬删除, file_import=禁用） | ruleHandler.Delete |

删除行为：
- `source = "manual"`：硬删除，从数据库中完全移除
- `source = "file_import"`：软删除，保留记录但标记 `enabled = false`

### 6.3 审核尺度预设 API（tenant_admin）

路由前缀：`/api/tenant/rules`

| 方法 | 路径 | 说明 | Handler |
|------|------|------|---------|
| GET | `/strictness-presets` | 查询当前租户的三级预设 | presetHandler.List |
| PUT | `/strictness-presets/:strictness` | 更新指定尺度的预设 | presetHandler.Update |

### 6.4 用户个人配置 API（业务用户）

路由前缀：`/api/tenant/settings`

| 方法 | 路径 | 说明 | Handler |
|------|------|------|---------|
| GET | `/processes` | 获取当前用户可见的流程列表（双重校验） | userConfigHandler.GetProcessList |
| GET | `/processes/:processType` | 获取单个流程的用户配置详情 | userConfigHandler.GetByProcessType |
| PUT | `/processes/:processType` | 更新用户对某流程的个性化配置 | userConfigHandler.UpdateByProcessType |
| GET | `/dashboard-prefs` | 获取用户仪表板偏好 | userConfigHandler.GetDashboardPrefs |
| PUT | `/dashboard-prefs` | 更新用户仪表板偏好 | userConfigHandler.UpdateDashboardPrefs |

双重校验逻辑：返回的流程列表中每个流程必须同时满足：
1. 在 `process_audit_configs` 表中存在对应租户的配置记录
2. 用户在 OA 系统中具有该流程的审批权限

### 6.5 用户配置管理 API（tenant_admin）

路由前缀：`/api/tenant/user-configs`

| 方法 | 路径 | 说明 | Handler |
|------|------|------|---------|
| GET | `/` | 查询当前租户所有用户的个人配置摘要 | userConfigMgmtHandler.ListUserConfigs |
| GET | `/:userId` | 查询指定用户的完整个人配置 | userConfigMgmtHandler.GetUserConfig |

### 6.6 Token 消耗统计 API

| 方法 | 路径 | 角色 | 说明 |
|------|------|------|------|
| GET | `/api/tenant/stats/token-usage` | tenant_admin | 查询当前租户的 Token 消耗统计 |
| GET | `/api/admin/stats/token-usage` | system_admin | 查询所有租户的 Token 消耗统计 |

查询参数 (TokenUsageQuery)：

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| start_time | string | 是 | 开始时间 |
| end_time | string | 是 | 结束时间 |
| model_config_id | string | 否 | 按模型配置筛选 |


---

## 七、数据脱敏

### 7.1 脱敏规则

位置：`go-service/internal/pkg/sanitize/sanitize.go`

Go 层在将数据发送到 AI 服务前，对敏感信息执行脱敏处理：

| 函数 | 字段类型 | 脱敏规则 | 示例 |
|------|---------|---------|------|
| `MaskIDCard` | 身份证号 | 保留前3后4，中间用 `*` | `110101199001011234` → `110***********1234` |
| `MaskPhone` | 手机号 | 保留前3后4，中间用 `*` | `13812345678` → `138****5678` |
| `MaskBankCard` | 银行卡号 | 仅保留后4位 | `6222021234567890` → `************7890` |
| `MaskSalary` | 薪资金额 | 替换为区间描述 | `15000.00` → `[10000-20000]` |

### 7.2 薪资区间映射

| 金额范围 | 脱敏结果 |
|---------|---------|
| < 0 | `[金额异常]` |
| 0 – 3000 | `[0-3000]` |
| 3000 – 5000 | `[3000-5000]` |
| 5000 – 8000 | `[5000-8000]` |
| 8000 – 10000 | `[8000-10000]` |
| 10000 – 20000 | `[10000-20000]` |
| 20000 – 50000 | `[20000-50000]` |
| ≥ 50000 | `[50000以上]` |

### 7.3 批量脱敏

`SanitizeText(text string) string` 函数对文本中的敏感信息进行批量脱敏，依次处理：
1. 身份证号（18位或15位数字）
2. 手机号（1开头的11位数字）
3. 银行卡号（16-19位数字）

正则表达式均为预编译，避免运行时重复编译开销。

### 7.4 脱敏时机

在 `AIModelCallerService.ChatViaPython()` 方法中，对 `req.UserPrompt` 执行 `sanitize.SanitizeText()` 后再构建请求体发送到 Python AI 服务。确保敏感数据不会离开 Go 层。

---

## 八、规则合并逻辑

### 8.1 合并引擎

位置：`go-service/internal/service/rule_merge.go`

`MergeRules()` 函数将租户规则和用户个性化配置合并为最终生效的规则列表。

### 8.2 优先级排序

```
优先级从高到低：
1. mandatory  (强制规则) — 始终生效，忽略用户 toggle
2. custom     (用户私有规则) — 用户自定义添加的规则
3. default_on (默认开启) — 用户可通过 toggle 关闭
4. default_off(默认关闭) — 用户可通过 toggle 开启
```

### 8.3 合并流程

```
输入：
  - tenantRules []AuditRule        // 租户级审核规则
  - userDetail  *AuditDetailItem   // 用户个性化配置（可为 nil）

处理步骤：
  1. 构建用户 toggle 覆盖映射 (rule_id → enabled)
  2. 遍历租户规则：
     - 跳过 enabled=false 的规则
     - mandatory: 始终 enabled=true，忽略 toggle
     - default_on: 默认 enabled=true，检查 toggle 覆盖
     - default_off: 默认 enabled=false，检查 toggle 覆盖
  3. 添加用户私有规则 (scope="custom", source="user")
  4. 按优先级排序

输出：
  - []MergedRule  // 合并后的最终生效规则列表
```

### 8.4 MergedRule 结构

| 字段 | 类型 | 说明 |
|------|------|------|
| RuleID | string | 规则 ID |
| Content | string | 规则内容 |
| Scope | string | 范围 (mandatory/default_on/default_off/custom) |
| Enabled | bool | 是否生效 |
| Source | string | 来源 (tenant/user) |


---

## 九、Go ↔ Python 服务间协议

### 9.1 职责边界

| 职责 | Go 业务中台 | Python AI 引擎 |
|------|------------|----------------|
| 用户鉴权 | ✅ JWT + RBAC | — |
| 租户隔离 | ✅ TenantContext 中间件 | — |
| 数据脱敏 | ✅ sanitize 包 | — |
| Token 配额检查 | ✅ 调用前校验 | — |
| Token 用量统计 | ✅ 累加 + 异步日志 | — |
| 规则合并 | ✅ MergeRules | — |
| 提示词组装 | ✅ PromptBuilder | — |
| LLM 直接调用 | ✅ 第一阶段 (Rules_Only) | — |
| RAG 检索 | — | ✅ 第二阶段 |
| OCR 解析 | — | ✅ 第二阶段 |
| LLM 链编排 | — | ✅ LangChain |

### 9.2 通信协议

传输方式：HTTP POST，JSON 格式  
服务地址：环境变量 `AI_SERVICE_URL`，默认 `http://ai-service:8000`  
端点：`/api/v1/chat/completions`

### 9.3 请求格式 (Go → Python)

```json
{
    "system_prompt": "你是一个专业的审核助手，请根据以下规则对流程数据进行审核...",
    "user_prompt": "请审核以下采购申请...(已脱敏数据)",
    "model_config": {
        "model_id": "uuid",
        "provider": "xinference",
        "model_name": "qwen2.5-72b",
        "endpoint": "http://xinference:9997",
        "max_tokens": 4096,
        "temperature": 0.3
    },
    "audit_context": {
        "tenant_id": "uuid",
        "process_type": "采购审批",
        "rules": ["规则1: ...", "规则2: ..."],
        "process_data": {
            "process_id": "12345",
            "main_data": {"申请人": "张三", "金额": "[10000-20000]"},
            "detail_data": [...]
        }
    }
}
```

关键说明：
- `user_prompt` 中的敏感数据已在 Go 层完成脱敏
- `model_config` 包含 Python 侧调用 LLM 所需的全部配置
- `audit_context` 提供审核上下文，Python 侧可用于 RAG 检索

### 9.4 响应格式 (Python → Go)

```json
{
    "content": "审核结果 JSON 字符串",
    "token_usage": {
        "input_tokens": 1200,
        "output_tokens": 800,
        "total_tokens": 2000
    },
    "model_id": "qwen2.5-72b",
    "duration_ms": 3500
}
```

### 9.5 错误处理

| 场景 | Go 侧处理 |
|------|-----------|
| Python 服务不可达 | 返回 `ErrAICallFailed`，前端展示连接失败提示 |
| Python 返回非 200 | 读取响应体，返回 `ErrAICallFailed` 附带错误详情 |
| 响应 JSON 解析失败 | 返回 `ErrAICallFailed`，提示响应解析失败 |
| Token 配额不足 | 调用前拦截，返回 `ErrTokenQuotaExceeded` |


---

## 十、前端集成

### 10.1 新增 Composable

#### useRulesApi.ts

位置：`frontend/composables/useRulesApi.ts`

封装规则配置相关 API 调用，替代 `useMockData` 中的模拟数据。

| 函数 | 说明 |
|------|------|
| `fetchConfigs()` | 获取流程审核配置列表 |
| `createConfig(data)` | 创建流程审核配置 |
| `updateConfig(id, data)` | 更新流程审核配置 |
| `deleteConfig(id)` | 删除流程审核配置 |
| `testConnection(processType)` | 测试 OA 流程连接 |
| `fetchFields(configId)` | 拉取 OA 字段定义 |
| `fetchAuditRules(params)` | 获取审核规则列表 |
| `createAuditRule(data)` | 创建审核规则 |
| `updateAuditRule(id, data)` | 更新审核规则 |
| `deleteAuditRule(id)` | 删除审核规则 |
| `fetchStrictnessPresets()` | 获取审核尺度预设 |
| `updateStrictnessPreset(strictness, data)` | 更新审核尺度预设 |

#### useSettingsApi.ts

位置：`frontend/composables/useSettingsApi.ts`

封装个人设置相关 API 调用，替代 `useMockData` 中的模拟数据。

| 函数 | 说明 |
|------|------|
| `fetchProcessList()` | 获取用户可见的流程列表 |
| `fetchProcessConfig(processType)` | 获取单个流程的用户配置 |
| `updateProcessConfig(processType, data)` | 更新用户流程配置 |
| `fetchDashboardPrefs()` | 获取仪表板偏好 |
| `updateDashboardPrefs(data)` | 更新仪表板偏好 |
| `fetchAllUserConfigs()` | 管理端：获取所有用户配置摘要 |
| `fetchUserConfig(userId)` | 管理端：获取指定用户配置 |
| `fetchTokenUsage(params)` | 查询 Token 消耗统计 |

### 10.2 页面变更

| 页面 | 文件路径 | 变更内容 |
|------|---------|---------|
| 规则配置 | `frontend/pages/admin/tenant/rules.vue` | 替换 mockProcessAuditConfigs 为 useRulesApi；接入测试连接、字段拉取 API；AI 配置区域标签修改 |
| 个人设置 | `frontend/pages/settings.vue` | 替换模拟数据为 useSettingsApi；实现权限锁定 UI；接入双重校验流程列表 |
| 用户配置管理 | `frontend/pages/admin/tenant/user-configs.vue` | 替换 mockUserPersonalConfigs 为 useSettingsApi 管理端 API |

### 10.3 数据迁移策略

采用渐进式迁移方案：
1. 新增 composable 封装真实 API 调用
2. 逐页替换 `useMockData` 引用为对应 composable
3. 保留 `useMockData` 供未迁移页面继续使用
4. 所有页面迁移完成后移除 `useMockData`


---

## 十一、多租户数据隔离策略

### 11.1 隔离机制

本功能所有新增表均包含 `tenant_id` 列，通过四层机制确保租户数据隔离：

```
┌──────────────────────────────────────────────────────┐
│  第 1 层：中间件层                                     │
│  TenantContext() 从 JWT 提取 tenant_id 注入 gin.Context │
├──────────────────────────────────────────────────────┤
│  第 2 层：Repository 层                               │
│  BaseRepo.WithTenant(c) 自动添加 WHERE tenant_id = ?  │
├──────────────────────────────────────────────────────┤
│  第 3 层：数据库层                                     │
│  外键约束 REFERENCES tenants(id) ON DELETE CASCADE     │
├──────────────────────────────────────────────────────┤
│  第 4 层：唯一约束层                                   │
│  UNIQUE(tenant_id, process_type) 等确保租户内唯一性     │
└──────────────────────────────────────────────────────┘
```

### 11.2 隔离覆盖范围

| 表名 | 隔离方式 | 唯一约束 |
|------|---------|---------|
| process_audit_configs | tenant_id + WithTenant | UNIQUE(tenant_id, process_type) |
| audit_rules | tenant_id + WithTenant | — |
| strictness_presets | tenant_id + WithTenant | UNIQUE(tenant_id, strictness) |
| user_personal_configs | tenant_id + user_id + WithTenant | UNIQUE(tenant_id, user_id) |
| user_dashboard_prefs | tenant_id + user_id + WithTenant | UNIQUE(tenant_id, user_id) |
| tenant_llm_message_logs | tenant_id + WithTenant | — |
| cron_tasks | tenant_id + WithTenant | — |
| audit_logs | tenant_id + WithTenant | — |
| cron_logs | tenant_id + WithTenant | — |
| archive_logs | tenant_id + WithTenant | — |

### 11.3 用户跨租户切换

用户可能属于多个租户。切换租户时：
- JWT 中的 tenant_id 更新为目标租户
- 所有 API 请求自动使用新的 tenant_id
- 用户个人配置按 `(tenant_id, user_id)` 隔离，不同租户下的配置互不干扰

---

## 十二、错误码体系

### 12.1 新增错误码

沿用现有 `errcode` 包的错误码风格，新增以下 13 个错误码：

| 错误码 | 常量名 | HTTP 状态码 | 说明 |
|--------|--------|------------|------|
| 40401 | `ErrProcessNotFound` | 404 | 流程在 OA 系统中不存在 |
| 40402 | `ErrConfigNotFound` | 404 | 流程审核配置不存在 |
| 40403 | `ErrRuleNotFound` | 404 | 审核规则不存在 |
| 40901 | `ErrDuplicateProcessType` | 409 | 同一租户下流程类型重复 |
| 40301 | `ErrPermissionDenied` | 403 | 用户权限不足 |
| 40302 | `ErrMandatoryRuleLocked` | 403 | 强制规则不可修改 |
| 50201 | `ErrOAConnectionFailed` | 502 | OA 数据库连接失败 |
| 50202 | `ErrOAQueryFailed` | 502 | OA 数据库查询失败 |
| 50203 | `ErrOATypeUnsupported` | 502 | 不支持的 OA 类型 |
| 50301 | `ErrAIConnectionFailed` | 503 | AI 模型连接失败 |
| 50302 | `ErrAICallFailed` | 503 | AI 模型调用失败 |
| 50303 | `ErrAIProviderUnsupported` | 503 | 不支持的 AI provider |
| 50304 | `ErrTokenQuotaExceeded` | 503 | 租户 Token 配额已用尽 |

### 12.2 错误处理策略

| 场景 | 策略 |
|------|------|
| OA 连接错误 | 返回具体错误信息（连接超时、认证失败），前端展示友好提示 |
| AI 调用错误 | 记录错误日志，返回降级结果，不中断审核流程 |
| Token 配额超限 | 调用前检查剩余配额，不足时拒绝调用 |
| 并发冲突 | 使用 `updated_at` 乐观锁，冲突时返回 409 |
| 数据脱敏失败 | 记录告警日志，拒绝发送未脱敏数据到 AI 服务 |


---

## 十三、代码层级与文件清单

### 13.1 Go 后端代码结构

```
go-service/internal/
├── model/                              # 数据模型
│   ├── process_audit_config.go         # 流程审核配置模型
│   ├── audit_rule.go                   # 审核规则模型
│   ├── strictness_preset.go            # 审核尺度预设模型
│   ├── user_personal_config.go         # 用户个人配置模型
│   ├── user_dashboard_pref.go          # 用户仪表板偏好模型
│   ├── tenant_llm_message_log.go       # 大模型消息记录模型
│   ├── cron_task.go                    # 定时任务模型
│   └── audit_log.go                    # 审核日志模型
│
├── repository/                         # 数据访问层
│   ├── process_audit_config_repo.go    # 流程审核配置 Repository
│   ├── audit_rule_repo.go             # 审核规则 Repository
│   ├── strictness_preset_repo.go      # 审核尺度预设 Reposi

---

## 九、Go ↔ Python 服务间协议

### 9.1 职责边界

| 职责 | Go 业务中台 | Python AI 引擎 |
|------|------------|----------------|
| 用户鉴权 | ✅ JWT + RBAC | — |
| 租户隔离 | ✅ TenantContext 中间件 | — |
| 数据脱敏 | ✅ sanitize 包 | — |
| Token 配额检查 | ✅ 调用前检查 | — |
| Token 用量统计 | ✅ 异步写入日志 | — |
| 规则合并 | ✅ MergeRules | — |
| 提示词组装 | ✅ PromptBuilder | — |
| LLM 直接调用 | ✅ 第一阶段 (rules_only) | — |
| RAG 检索 | — | ✅ 第二阶段 |
| OCR 解析 | — | ✅ 第二阶段 |
| LangChain 编排 | — | ✅ 第二阶段 |

### 9.2 通信方式

Go → Python 通过 HTTP POST 调用，服务地址由环境变量 `AI_SERVICE_URL` 配置，默认 `http://ai-service:8000`。

### 9.3 请求格式 (Go → Python)

端点：`POST /api/v1/chat/completions`

```json
{
    "system_prompt": "你是一个专业的审核助手...",
    "user_prompt": "请审核以下采购申请...（已脱敏）",
    "model_config": {
        "model_id": "uuid",
        "provider": "xinference",
        "model_name": "qwen2.5-72b",
        "endpoint": "http://xinference:9997",
        "max_tokens": 4096,
        "temperature": 0.3
    },
    "audit_context": {
        "tenant_id": "uuid",
        "process_type": "采购审批",
        "rules": ["规则1", "规则2"],
        "process_data": { "字段1": "值1" }
    }
}
```

关键点：
- `user_prompt` 已经过 `SanitizeText()` 脱敏处理
- `model_config` 包含完整的模型连接信息，Python 端据此选择调用方式
- `audit_context` 提供审核上下文，Python 端可用于 RAG 检索（第二阶段）

### 9.4 响应格式 (Python → Go)

```json
{
    "content": "审核结果 JSON 字符串",
    "token_usage": {
        "input_tokens": 1200,
        "output_tokens": 800,
        "total_tokens": 2000
    },
    "model_id": "qwen2.5-72b",
    "duration_ms": 3500
}
```

### 9.5 错误处理

- HTTP 状态码非 200 时，Go 层读取响应体并包装为 `ErrAICallFailed` 错误
- 网络超时或连接失败时，返回 `ErrAICallFailed` 并附带具体错误信息
- Python 端响应解析失败时，返回 "Python AI服务响应解析失败"


---

## 十、前端集成

### 10.1 新增 Composable

#### useRulesApi.ts

位置：`frontend/composables/useRulesApi.ts`

封装规则配置相关 API 调用，替代 `useMockData` 中的模拟数据。

| 函数 | 说明 |
|------|------|
| `fetchConfigs()` | 获取流程审核配置列表 |
| `createConfig(data)` | 创建流程审核配置 |
| `updateConfig(id, data)` | 更新流程审核配置 |
| `deleteConfig(id)` | 删除流程审核配置 |
| `testConnection(processType)` | 测试 OA 流程连接 |
| `fetchFields(configId)` | 拉取 OA 字段定义 |
| `fetchAuditRules(params)` | 获取审核规则列表 |
| `createAuditRule(data)` | 创建审核规则 |
| `updateAuditRule(id, data)` | 更新审核规则 |
| `deleteAuditRule(id)` | 删除审核规则 |
| `fetchStrictnessPresets()` | 获取审核尺度预设 |
| `updateStrictnessPreset(strictness, data)` | 更新审核尺度预设 |

#### useSettingsApi.ts

位置：`frontend/composables/useSettingsApi.ts`

封装个人设置相关 API 调用，替代 `useMockData` 中的模拟数据。

| 函数 | 说明 |
|------|------|
| `fetchProcessList()` | 获取用户可见的流程列表 |
| `fetchProcessConfig(processType)` | 获取单个流程的用户配置 |
| `updateProcessConfig(processType, data)` | 更新用户流程配置 |
| `fetchDashboardPrefs()` | 获取仪表板偏好 |
| `updateDashboardPrefs(data)` | 更新仪表板偏好 |
| `fetchUserConfigs()` | (管理端) 获取所有用户配置摘要 |
| `fetchUserConfig(userId)` | (管理端) 获取指定用户配置 |
| `fetchTokenUsage(params)` | 获取 Token 消耗统计 |

### 10.2 页面变更

| 页面 | 文件路径 | 变更内容 |
|------|---------|---------|
| 规则配置 | `frontend/pages/admin/tenant/rules.vue` | 替换 mockProcessAuditConfigs 为 useRulesApi；接入测试连接、字段拉取 API；AI 配置区域标签修改 |
| 个人设置 | `frontend/pages/settings.vue` | 替换模拟数据为 useSettingsApi；实现权限锁定 UI；接入双重校验流程列表 |
| 用户配置管理 | `frontend/pages/admin/tenant/user-configs.vue` | 替换 mockUserPersonalConfigs 为 useSettingsApi 管理端 API |

### 10.3 迁移策略

采用渐进式迁移方案：
1. 新增 composable 文件，封装所有 API 调用
2. 逐页替换 `useMockData` 引用为对应 composable
3. 保留 `useMockData` 作为降级方案，API 不可用时可快速回退


---

## 十一、多租户数据隔离策略

### 11.1 隔离机制

本功能所有新增表均包含 `tenant_id` 列，通过四层机制确保租户数据隔离：

```
┌─────────────────────────────────────────────────────────┐
│  第1层：中间件层                                         │
│  TenantContext() 从 JWT 提取 tenant_id 注入 gin.Context  │
├─────────────────────────────────────────────────────────┤
│  第2层：Repository 层                                    │
│  BaseRepo.WithTenant(c) 自动添加 WHERE tenant_id = ?    │
├─────────────────────────────────────────────────────────┤
│  第3层：数据库层                                         │
│  外键约束 REFERENCES tenants(id) ON DELETE CASCADE       │
├─────────────────────────────────────────────────────────┤
│  第4层：唯一约束层                                       │
│  UNIQUE(tenant_id, process_type) 等确保租户内唯一性       │
└─────────────────────────────────────────────────────────┘
```

### 11.2 各表隔离约束

| 表名 | 唯一约束 | 外键级联 |
|------|---------|---------|
| process_audit_configs | `UNIQUE(tenant_id, process_type)` | `ON DELETE CASCADE` |
| audit_rules | — | `tenant_id → tenants(id) CASCADE` |
| strictness_presets | `UNIQUE(tenant_id, strictness)` | `ON DELETE CASCADE` |
| user_personal_configs | `UNIQUE(tenant_id, user_id)` | `ON DELETE CASCADE` |
| user_dashboard_prefs | `UNIQUE(tenant_id, user_id)` | `ON DELETE CASCADE` |
| tenant_llm_message_logs | — | `tenant_id → tenants(id) CASCADE` |
| cron_tasks | — | `tenant_id → tenants(id) CASCADE` |
| audit_logs | — | `tenant_id → tenants(id) CASCADE` |
| archive_logs | — | `tenant_id → tenants(id) CASCADE` |

### 11.3 用户跨租户配置

用户个人配置通过 `UNIQUE(tenant_id, user_id)` 约束实现跨租户隔离。同一用户在不同租户下拥有独立的配置记录，切换租户时加载对应租户的配置。

---

## 十二、错误码体系

### 12.1 新增错误码

沿用现有 `errcode` 包的错误码风格，新增以下错误码：

| 错误码 | 常量名 | HTTP 状态码 | 说明 |
|--------|--------|------------|------|
| 40401 | `ErrProcessNotFound` | 404 | 流程在 OA 系统中不存在 |
| 40402 | `ErrConfigNotFound` | 404 | 流程审核配置不存在 |
| 40403 | `ErrRuleNotFound` | 404 | 审核规则不存在 |
| 40901 | `ErrDuplicateProcessType` | 409 | 同一租户下流程类型重复 |
| 40301 | `ErrPermissionDenied` | 403 | 用户权限不足 |
| 40302 | `ErrMandatoryRuleLocked` | 403 | 强制规则不可修改 |
| 50201 | `ErrOAConnectionFailed` | 502 | OA 数据库连接失败 |
| 50202 | `ErrOAQueryFailed` | 502 | OA 数据库查询失败 |
| 50203 | `ErrOATypeUnsupported` | 502 | 不支持的 OA 类型 |
| 50301 | `ErrAIConnectionFailed` | 503 | AI 模型连接失败 |
| 50302 | `ErrAICallFailed` | 503 | AI 模型调用失败 |
| 50303 | `ErrAIProviderUnsupported` | 503 | 不支持的 AI provider |
| 50304 | `ErrTokenQuotaExceeded` | 503 | 租户 Token 配额已用尽 |

### 12.2 错误处理策略

| 场景 | 处理方式 |
|------|---------|
| OA 连接错误 | 返回具体错误信息（连接超时、认证失败、数据库不存在），前端展示友好提示 |
| AI 调用错误 | 记录错误日志，返回降级结果（如仅规则检查结果），不中断审核流程 |
| Token 配额超限 | 调用前检查剩余配额，不足时拒绝调用并返回 50304 错误 |
| 并发冲突 | 使用 `updated_at` 乐观锁，冲突时返回 409 状态码 |
| 数据脱敏失败 | 记录告警日志，拒绝发送未脱敏数据到 AI 服务 |

---

## 十三、代码文件清单

### 13.1 数据库迁移

| 文件 | 说明 |
|------|------|
| `db/migrations/000007_audit_configs_rules_presets.up.sql` | 创建 process_audit_configs, audit_rules, strictness_presets |
| `db/migrations/000007_audit_configs_rules_presets.down.sql` | 回滚 |
| `db/migrations/000008_cron_tasks.up.sql` | 创建 cron_tasks, cron_task_type_configs |
| `db/migrations/000008_cron_tasks.down.sql` | 回滚 |
| `db/migrations/000009_audit_cron_archive_logs.up.sql` | 创建 audit_logs, cron_logs, archive_logs |
| `db/migrations/000009_audit_cron_archive_logs.down.sql` | 回滚 |
| `db/migrations/000010_user_personal_configs.up.sql` | 创建 user_personal_configs, user_dashboard_prefs |
| `db/migrations/000010_user_personal_configs.down.sql` | 回滚 |
| `db/migrations/000011_tenant_llm_message_logs.up.sql` | 创建 tenant_llm_message_logs |
| `db/migrations/000011_tenant_llm_message_logs.down.sql` | 回滚 |

### 13.2 Go 后端

| 层级 | 文件 | 说明 |
|------|------|------|
| Model | `internal/model/process_audit_config.go` | 流程审核配置模型 |
| Model | `internal/model/audit_rule.go` | 审核规则模型 |
| Model | `internal/model/strictness_preset.go` | 审核尺度预设模型 |
| Model | `internal/model/user_personal_config.go` | 用户个人配置模型 |
| Model | `internal/model/user_dashboard_pref.go` | 用户仪表板偏好模型 |
| Model | `internal/model/tenant_llm_message_log.go` | LLM 消息日志模型 |
| Model | `internal/model/cron_task.go` | 定时任务模型 |
| Model | `internal/model/audit_log.go` | 审核日志模型 |
| Errcode | `internal/pkg/errcode/errcode.go` | 13 个新增错误码 |
| Repository | `internal/repository/process_audit_config_repo.go` | 流程审核配置仓储 |
| Repository | `internal/repository/audit_rule_repo.go` | 审核规则仓储 |
| Repository | `internal/repository/strictness_preset_repo.go` | 审核尺度预设仓储 |
| Repository | `internal/repository/user_personal_config_repo.go` | 用户个人配置仓储 |
| Repository | `internal/repository/llm_message_log_repo.go` | LLM 消息日志仓储 |
| Repository | `internal/repository/user_dashboard_pref_repo.go` | 用户仪表板偏好仓储 |
| OA Adapter | `internal/pkg/oa/adapter.go` | OA 适配器接口 |
| OA Adapter | `internal/pkg/oa/ecology9.go` | 泛微 E9 适配器实现 |
| OA Adapter | `internal/pkg/oa/factory.go` | OA 适配器工厂 |
| AI Caller | `internal/pkg/ai/caller.go` | AI 模型调用接口 |
| AI Caller | `internal/pkg/ai/openai_compat.go` | OpenAI 兼容协议统一调用器 |
| AI Caller | `internal/pkg/ai/factory.go` | AI 调用器工厂（按 provider 路由） |
| Sanitize | `internal/pkg/sanitize/sanitize.go` | 数据脱敏工具 |
| Service | `internal/service/process_audit_config_service.go` | 流程审核配置服务 |
| Service | `internal/service/audit_rule_service.go` | 审核规则服务 |
| Service | `internal/service/strictness_preset_service.go` | 审核尺度预设服务 |
| Service | `internal/service/user_personal_config_service.go` | 用户个人配置服务 |
| Service | `internal/service/rule_merge.go` | 规则合并引擎 |
| Service | `internal/service/ai_caller_service.go` | AI 调用服务 |
| Service | `internal/service/llm_message_log_service.go` | LLM 日志服务 |
| Service | `internal/service/prompt_builder.go` | 提示词组装器 |
| Handler | `internal/handler/process_audit_config_handler.go` | 流程审核配置处理器 |
| Handler | `internal/handler/audit_rule_handler.go` | 审核规则处理器 |
| Handler | `internal/handler/strictness_preset_handler.go` | 审核尺度预设处理器 |
| Handler | `internal/handler/user_personal_config_handler.go` | 用户个人配置处理器 |
| Handler | `internal/handler/user_config_management_handler.go` | 用户配置管理处理器 |
| Handler | `internal/handler/llm_message_log_handler.go` | LLM 日志处理器 |
| DTO | `internal/dto/rules_dto.go` | 规则相关 DTO |
| DTO | `internal/dto/settings_dto.go` | 设置相关 DTO |
| Router | `internal/router/router.go` | 路由注册（更新） |

### 13.3 前端

| 文件 | 说明 |
|------|------|
| `frontend/types/user-config.ts` | 用户个人配置相关类型定义 |
| `frontend/composables/useRulesApi.ts` | 规则配置 API composable |
| `frontend/composables/useSettingsApi.ts` | 个人设置 API composable（类型从 `types/user-config.ts` 导入并 re-export） |
| `frontend/pages/admin/tenant/rules.vue` | 规则配置页面（更新） |
| `frontend/pages/settings.vue` | 个人设置页面（更新） |
| `frontend/pages/admin/tenant/user-configs.vue` | 用户配置管理页面（更新） |
