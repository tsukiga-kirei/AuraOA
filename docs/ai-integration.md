# AI 系统对接说明

## 概述

OA 智审通过统一的 AI 调用层对接多种大语言模型（LLM），支持本地部署和云端 API 两种模式。AI 系统在审核流程中承担核心推理角色，采用**两阶段审核**架构（推理 → 结构化提取），确保审核结论的准确性和可解释性。

## 架构设计

### AI 调用接口

所有 AI 模型调用器均实现统一的 `AIModelCaller` 接口（定义于 `go-service/internal/pkg/ai/caller.go`）：

```go
type AIModelCaller interface {
    // TestConnection 测试模型连接是否可用
    TestConnection(ctx context.Context) error

    // Chat 发送对话请求，返回模型响应和 Token 消耗
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
}
```

### 调用器实现

当前所有 AI 服务商均通过 `OpenAICompatCaller`（`go-service/internal/pkg/ai/openai_compat.go`）统一调用，该调用器兼容所有支持 OpenAI Chat Completions API 格式的服务：

- **请求格式**：标准 OpenAI `/chat/completions` 接口
- **流式输出**：支持 SSE（Server-Sent Events）流式响应，实时推送推理过程
- **Token 统计**：非流式模式从响应 `usage` 字段获取精确值；流式模式按字符数估算

### 工厂模式

`NewAIModelCaller` 工厂函数（`go-service/internal/pkg/ai/factory.go`）根据 `provider` 创建对应的调用器实例：

| 部署类型 | Provider | 要求 |
|---------|----------|------|
| 本地部署 | `xinference`, `ollama`, `vllm` | 必须配置 `endpoint` |
| 云端 API | `aliyun_bailian`, `deepseek`, `zhipu`, `openai` | 必须配置 `api_key`，`endpoint` 可选（有默认值） |
| 云端 API | `azure_openai` | 必须同时配置 `api_key` 和 `endpoint` |

### 云端 Provider 默认 Endpoint

| Provider | 默认 Endpoint |
|----------|--------------|
| `aliyun_bailian` | `https://dashscope.aliyuncs.com/compatible-mode/v1` |
| `deepseek` | `https://api.deepseek.com/v1` |
| `zhipu` | `https://open.bigmodel.cn/api/paas/v4` |
| `openai` | `https://api.openai.com/v1` |

## 已支持的 AI 服务商

### 本地部署（`deploy_type=local`）

| 服务商 | 编码 | 状态 | 说明 |
|--------|------|------|------|
| Xinference | `xinference` | ✅ 已支持 | 分布式推理框架，支持多种开源模型 |
| Ollama | `ollama` | ✅ 已支持 | 轻量级本地模型运行工具 |
| vLLM | `vllm` | ✅ 已支持 | 高性能推理引擎，支持 PagedAttention |

### 云端 API（`deploy_type=cloud`）

| 服务商 | 编码 | 状态 | 说明 |
|--------|------|------|------|
| 阿里云百炼 | `aliyun_bailian` | ✅ 已支持 | 阿里云大模型服务平台 |
| DeepSeek | `deepseek` | ✅ 已支持 | DeepSeek 系列模型 API |
| 智谱 AI | `zhipu` | ✅ 已支持 | GLM 系列模型 API |
| OpenAI | `openai` | ✅ 已支持 | GPT 系列模型 API |
| Azure OpenAI | `azure_openai` | ✅ 已支持 | 微软 Azure 托管的 OpenAI 服务 |

## 两阶段审核流程

AI 审核采用两阶段架构，将推理和结构化提取分离，提高审核质量：

```
┌─────────────────────────────────────────────────────────────────┐
│                     两阶段 AI 审核流程                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  阶段一：推理（Reasoning）                                       │
│  ├── 输入：系统提示词 + 用户提示词（含流程数据、规则、审批流）      │
│  ├── 温度：租户配置值（默认 0.3）                                 │
│  ├── 输出：自然语言推理分析文本                                   │
│  └── 流式：支持 SSE 实时推送推理过程到前端                        │
│                                                                 │
│  阶段二：提取（Extraction）                                      │
│  ├── 输入：推理结果 + 规则文本                                   │
│  ├── 温度：固定 0.1（确保输出稳定）                               │
│  ├── 输出：结构化 JSON（建议、评分、置信度、逐条规则评估）         │
│  └── 配额：跳过 Token 预扣（与推理阶段共享配额）                  │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 提示词构建

提示词通过模板 + 占位符替换方式构建（`go-service/internal/service/audit_prompt_builder.go`）：

**推理阶段占位符**：

| 占位符 | 数据来源 |
|--------|---------|
| `{{process_type}}` | 流程类型名称 |
| `{{main_table}}` / `{{fields}}` | 主表字段数据（经字段过滤） |
| `{{detail_tables}}` | 明细表数据（经字段过滤） |
| `{{rules}}` | 合并后的审核规则文本 |
| `{{current_node}}` | 当前审批节点名称 |
| `{{flow_history}}` | 审批流历史文本 |
| `{{flow_graph}}` | 审批流路由图文本 |

**提取阶段占位符**：

| 占位符 | 数据来源 |
|--------|---------|
| `{{reasoning_result}}` | 推理阶段输出的分析文本 |
| `{{rules}}` | 合并后的审核规则文本 |

### 审核尺度

系统支持三种审核严格度，影响提示词中的审核标准描述：

| 尺度 | 说明 |
|------|------|
| `strict` | 严格模式 — 对合规性要求最高 |
| `standard` | 标准模式 — 平衡合规性与灵活性 |
| `lenient` | 宽松模式 — 仅关注重大违规 |

审核尺度的优先级：用户个人配置 > 租户配置 > 系统默认值。

## Token 配额管理

### 预扣-结算机制

为防止并发场景下 Token 超额消耗，系统采用**预扣-结算**机制：

```
1. 预扣（Reserve）
   └─ 原子操作：token_used + max_tokens <= token_quota
   └─ 条件更新，只有满足条件的请求才能成功预扣

2. 调用 AI 模型

3a. 调用成功 → 结算（Settle）
    └─ token_used = token_used - reserved + actual
    └─ 实际消耗 < 预扣时退还差额

3b. 调用失败 → 回滚（Release）
    └─ token_used = GREATEST(token_used - reserved, 0)
    └─ 使用 GREATEST 防止负值
```

### 配额层级

| 层级 | 字段 | 说明 |
|------|------|------|
| 租户 | `token_quota` | 租户总 Token 配额 |
| 租户 | `token_used` | 租户已消耗 Token 数 |
| 租户 | `max_tokens_per_request` | 单次请求最大输出 Token 限制 |

### 调用日志

每次 AI 调用完成后，异步写入 `tenant_llm_message_logs` 表，记录：

- 租户 ID、用户 ID、模型配置 ID
- 请求类型（`audit` / `archive`）
- 输入/输出/总 Token 消耗
- 调用耗时（毫秒）

日志写入采用异步 goroutine + 指数退避重试（最多 3 次），不阻塞主流程。

## Python AI 服务（可选）

系统预留了 Python AI 服务的对接通道（`ChatViaPython` 方法），适用于需要 Python 侧特殊处理的场景：

- RAG 检索增强生成
- 复杂上下文注入
- 自定义模型推理管线

**调用方式**：通过 HTTP 转发至 Python AI 服务（默认地址 `http://ai-service:8000`）

**请求路径**：`POST /api/v1/chat/completions`

**数据脱敏**：调用前自动对用户提示词中的敏感信息进行脱敏处理

> 注意：Python AI 服务为可选组件，当前核心审核流程直接通过 Go 服务调用 AI 模型，不依赖 Python 服务。

## 未完成的适配内容

### 功能层面

| 功能 | 状态 | 说明 |
|------|------|------|
| 备用模型自动切换 | ❌ 未实现 | 租户已有 `fallback_model_id` 字段，但主模型不可用时尚未自动切换到备用模型 |
| AI 调用重试机制 | ❌ 未实现 | 租户已有 `retry_count` 字段，但 AI 调用失败后尚未按配置自动重试 |
| 流式 Token 精确统计 | ❌ 未实现 | SSE 流式响应不返回 `usage` 字段，当前按字符数粗略估算（÷4），与实际消耗有偏差 |
| Python AI 服务集成 | 🔧 部分实现 | Go 侧调用通道已就绪，但 Python 服务本身尚未开发 |
| RAG 知识库模式 | ❌ 未实现 | 配置中已有 `kb_mode` 字段，但知识库检索增强功能尚未开发 |
| 模型能力匹配 | ❌ 未实现 | `ai_model_configs.capabilities` 字段已定义，但未根据能力标签自动匹配审核任务 |
| 多模态审核 | ❌ 未实现 | 当前仅支持文本审核，不支持图片/附件内容的 AI 分析 |
| 审核结果缓存 | 🔧 部分实现 | 快照表已支持历史审核结果存储，但未实现相同数据的审核结果复用 |

### 服务商层面

| 服务商 | 状态 | 说明 |
|--------|------|------|
| 百度文心一言 | ❌ 未注册 | 需添加 provider 选项和默认 endpoint |
| 腾讯混元 | ❌ 未注册 | 需添加 provider 选项和默认 endpoint |
| Anthropic Claude | ❌ 未注册 | API 格式与 OpenAI 不完全兼容，可能需要独立调用器 |
| Google Gemini | ❌ 未注册 | API 格式与 OpenAI 不兼容，需要独立调用器 |

> 注意：任何兼容 OpenAI Chat Completions API 格式的模型服务，均可通过现有 `OpenAICompatCaller` 直接接入，只需在数据库选项表中注册 provider 并配置 endpoint 即可。

## 配置说明

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `AI_SERVICE_URL` | `http://ai-service:8000` | Python AI 服务地址（可选） |
| `LLM_PROVIDER` | `local` | 默认 LLM 提供商（Python 服务使用） |
| `LLM_MODEL_NAME` | `default` | 默认模型名称（Python 服务使用） |
| `LLM_API_KEY` | 空 | 默认 API Key（Python 服务使用） |

### 管理后台配置路径

1. 系统管理员登录 → 系统设置 → AI 模型配置
2. 新建模型：选择部署类型、服务商、填写 Endpoint 和 API Key
3. 测试连接：验证模型可用性
4. 关联租户：在租户管理中为租户分配主用/备用模型
5. 配置参数：设置温度、最大 Token、超时时间等

### 租户级 AI 参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `primary_model_id` | — | 主用 AI 模型 |
| `fallback_model_id` | — | 备用 AI 模型（未实现自动切换） |
| `max_tokens_per_request` | 8192 | 单次请求最大输出 Token |
| `temperature` | 0.30 | AI 生成温度（0~1） |
| `timeout_seconds` | 60 | AI 请求超时时间（秒） |
| `retry_count` | 3 | AI 请求失败重试次数（未实现） |
| `token_quota` | — | 租户 Token 总配额 |
