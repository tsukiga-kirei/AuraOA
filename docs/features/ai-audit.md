# AI 智能审核功能说明

> 文档版本：v1.0 | 创建日期：2026-03-19
> 从代码层面介绍 OA 智审平台的 AI 审核引擎架构、调用流程与模型适配能力。

---

## 一、AI 审核整体架构

OA 智审采用 **两阶段审核** 架构：先由 LLM 进行自由文本推理分析，再由 LLM 将推理结果结构化为标准 JSON 格式。

```
┌── 阶段一：推理（Reasoning）────────────────────────┐
│                                                      │
│  System Prompt（系统提示词，含审核尺度指令）            │
│  + User Prompt（用户提示词，含主表数据、明细表、规则）   │
│       ↓                                              │
│  LLM 自由文本输出（分析过程与推理结论）                 │
│                                                      │
└──────────────────────────────────────────────────────┘
                         ↓
┌── 阶段二：提取（Extraction）───────────────────────┐
│                                                      │
│  System Prompt（结构化提取指令，含 JSON Schema）       │
│  + User Prompt（推理结果 + 审核规则）                  │
│       ↓                                              │
│  LLM 输出标准 JSON（含建议/评分/规则结果/风险点）       │
│                                                      │
└──────────────────────────────────────────────────────┘
```

---

## 二、AI 模型调用层

### 2.1 调用器接口

**文件位置**：`go-service/internal/pkg/ai/caller.go`

```go
type AIModelCaller interface {
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
}
```

### 2.2 请求/响应结构

| 结构 | 字段 | 说明 |
|------|------|------|
| `ChatRequest` | `SystemPrompt` | 系统角色提示词 |
| | `UserPrompt` | 用户角色提示词 |
| | `MaxTokens` | 最大输出 Token 数 |
| | `Temperature` | 生成温度（0~1） |
| `ChatResponse` | `Content` | LLM 输出内容 |
| | `TokenUsage` | Token 消耗统计（输入/输出/总计） |
| | `ModelID` | 使用的模型标识 |
| | `DurationMs` | 调用耗时（毫秒） |

### 2.3 OpenAI 兼容调用器

**文件位置**：`go-service/internal/pkg/ai/openai_compat.go`

所有已支持的 AI 服务商均通过 **OpenAI 兼容协议** 调用（`/v1/chat/completions`），包括本地部署和云端 API。

---

## 三、AI 模型工厂

**文件位置**：`go-service/internal/pkg/ai/factory.go`

```go
func NewAIModelCaller(cfg *model.AIModelConfig) (AIModelCaller, error)
```

### 支持的服务商

| 部署类型 | Provider 编码 | 服务商名称 | 调用协议 |
|---------|--------------|-----------|---------|
| 本地部署 | `xinference` | Xinference | OpenAI 兼容 |
| 本地部署 | `ollama` | Ollama | OpenAI 兼容 |
| 本地部署 | `vllm` | vLLM | OpenAI 兼容 |
| 云端 API | `aliyun_bailian` | 阿里云百炼 | OpenAI 兼容 |
| 云端 API | `deepseek` | DeepSeek | OpenAI 兼容 |
| 云端 API | `zhipu` | 智谱 AI | OpenAI 兼容 |
| 云端 API | `openai` | OpenAI | 原生 OpenAI |
| 云端 API | `azure_openai` | Azure OpenAI | OpenAI 兼容 |

### 默认 API 端点

| Provider | 默认 Endpoint |
|----------|---------------|
| `aliyun_bailian` | `https://dashscope.aliyuncs.com/compatible-mode/v1` |
| `deepseek` | `https://api.deepseek.com/v1` |
| `zhipu` | `https://open.bigmodel.cn/api/paas/v4` |
| `openai` | `https://api.openai.com/v1` |

---

## 四、AI 模型调用服务

**文件位置**：`go-service/internal/service/ai_caller_service.go`

`AIModelCallerService` 是 AI 调用的核心编排层，封装了：

### 4.1 两种调用模式

| 方法 | 说明 | 适用场景 |
|------|------|---------|
| `Chat()` | Go 直接调用 AI 模型 | 本地部署 / 简单场景 |
| `ChatViaPython()` | 通过 Python AI 服务中转调用 | 需要 Python 生态能力（RAG/向量检索等） |

### 4.2 Token 配额管理

采用 **预扣→结算** 模式确保高并发下不超配额：

```
调用前 → reserveTokenQuota()   预扣 max_tokens 额度
调用中 → 实际 AI 调用
调用后 → settleTokenUsage()    用实际消耗替换预扣额度
调用失败 → releaseTokenQuota()  回滚预扣
```

**原子操作实现**（防止并发超额）：
```sql
UPDATE tenants
SET token_used = token_used + ?
WHERE id = ? AND token_used + ? <= token_quota
```

### 4.3 异步日志写入

AI 调用完成后，Token 消耗记录异步写入 `tenant_llm_message_logs` 表：
- 使用 goroutine 异步执行，不阻塞主请求
- 失败时指数退避重试（最多 3 次：1s → 2s → 4s）
- 重试耗尽后写入标准日志（运维可通过日志采集发现）

### 4.4 数据脱敏

`ChatViaPython()` 调用前，对用户提示词进行数据脱敏处理：

**文件位置**：`go-service/internal/pkg/sanitize/sanitize.go`

---

## 五、提示词模板系统

### 5.1 系统提示词模板

**表名**：`system_prompt_templates`

系统初始化时预置了 **24 条** 提示词模板（审核 12 条 + 归档 12 条）：

```
审核/归档 × 推理/提取 × 严格/标准/宽松 × 系统/用户 = 2 × 2 × 3 × 2 = 24 条
```

| 维度 | 取值 | 说明 |
|------|------|------|
| **类型** | `system` / `user` | 系统提示词（角色设定） / 用户提示词（任务指令） |
| **阶段** | `reasoning` / `extraction` | 推理阶段（自由分析） / 提取阶段（结构化输出） |
| **尺度** | `strict` / `standard` / `loose` | 严格（零容忍） / 标准（平衡） / 宽松（聚焦重大违规） |

### 5.2 提示词键名格式

| 场景 | 键名格式 | 示例 |
|------|---------|------|
| 审核 | `audit_{type}_{phase}_{strictness}` | `audit_system_reasoning_strict` |
| 归档 | `archive_{type}_{phase}_{strictness}` | `archive_user_extraction_standard` |

### 5.3 用户提示词模板变量

用户提示词中使用 `{{variable}}` 格式的模板变量，在实际调用时替换：

| 变量 | 说明 |
|------|------|
| `{{main_table}}` | 主表数据 |
| `{{detail_tables}}` | 明细表数据 |
| `{{rules}}` | 审核/复核规则列表 |
| `{{flow_history}}` | 审批流历史 |
| `{{flow_graph}}` | 流程图节点 |
| `{{current_node}}` | 当前审批/归档节点 |
| `{{reasoning_result}}` | 推理阶段输出（仅提取阶段使用） |
| `{{process_type}}` | 流程类型 |

### 5.4 提示词构建

**文件位置**：`go-service/internal/service/prompt_builder.go`

```go
func BuildPrompt(aiConfig *AIConfigData, processType, fields, rules string) *ChatRequest
```

当前 `BuildPrompt` 仅处理推理阶段的提示词组装，将模板变量替换为实际数据。

---

## 六、AI 审核结果结构

### 6.1 审核工作台结果

```json
{
  "recommendation": "approve | return | review",
  "overall_score": 0-100,
  "rule_results": [
    {
      "rule_content": "规则内容",
      "passed": true/false,
      "reason": "判断理由"
    }
  ],
  "risk_points": ["风险点描述"],
  "suggestions": ["改进建议"],
  "confidence": 0-100
}
```

| 字段 | 说明 |
|------|------|
| `recommendation` | 综合建议：approve=通过、return=退回、review=人工复核 |
| `overall_score` | 综合评分（0-100），评分阈值因审核尺度而异 |
| `rule_results` | 逐条规则校验结果 |
| `confidence` | AI 结论置信度 |

### 6.2 归档复盘结果

```json
{
  "overall_compliance": "compliant | non_compliant | partially_compliant",
  "overall_score": 0-100,
  "rule_results": [...],
  "risk_points": [...],
  "suggestions": [...],
  "confidence": 0-100
}
```

两者结构相似，但归档使用 `overall_compliance` 替代 `recommendation`。

### 6.3 评分阈值（按尺度）

| 尺度 | 审核通过阈值 | 审核退回阈值 | 归档合规阈值 | 归档不合规阈值 |
|------|-------------|-------------|-------------|--------------|
| 严格 | ≥80 通过 | <60 退回 | ≥80 合规 | <60 不合规 |
| 标准 | ≥70 通过 | <50 退回 | ≥70 合规 | <50 不合规 |
| 宽松 | ≥50 通过 | <30 退回 | ≥50 合规 | <30 不合规 |

---

## 七、AI 模型配置管理

### 7.1 数据模型

**表名**：`ai_model_configs`

| 字段 | 说明 |
|------|------|
| `provider` | 服务商编码 |
| `model_name` | 模型标识名（API 调用时使用） |
| `display_name` | 显示名称（前端展示） |
| `deploy_type` | 部署类型：local/cloud |
| `endpoint` | API 端点 URL |
| `api_key` | API 密钥（JSON 序列化时隐藏） |
| `api_key_configured` | API 密钥是否已配置 |
| `max_tokens` | 单次最大输出 Token |
| `context_window` | 上下文窗口大小 |
| `cost_per_1k_tokens` | 千 Token 费用 |
| `capabilities` | 能力列表（JSONB） |

### 7.2 管理 API（系统管理员）

| API | 说明 |
|-----|------|
| `GET /api/admin/system/ai-models` | 列出所有模型 |
| `POST /api/admin/system/ai-models` | 创建模型配置 |
| `PUT /api/admin/system/ai-models/:id` | 更新模型配置 |
| `DELETE /api/admin/system/ai-models/:id` | 删除模型 |
| `POST /api/admin/system/ai-models/test` | 测试模型连接（参数形式） |
| `POST /api/admin/system/ai-models/:id/test` | 测试已保存的模型 |

### 7.3 租户与模型的关系

每个租户配置一个**主用模型**和一个**备用模型**：

| 租户字段 | 说明 |
|---------|------|
| `primary_model_id` | 主用 AI 模型（外键 → `ai_model_configs.id`） |
| `fallback_model_id` | 备用 AI 模型（主模型不可用时切换） |
| `max_tokens_per_request` | 单次审核最大输出 Token 限制 |
| `temperature` | AI 生成温度参数（默认 0.30） |
| `timeout_seconds` | 请求超时（默认 60 秒） |
| `retry_count` | 失败重试次数（默认 3 次） |
| `token_quota` | Token 总配额 |
| `token_used` | 已消耗 Token |

---

## 八、审核规则系统

### 8.1 规则层级

```
系统提示词模板（全局）
  └── 租户通用规则（audit_rules / archive_rules）
        └── 用户个人规则覆盖（user_personal_configs.rule_toggle_overrides / custom_rules）
```

### 8.2 规则作用域

| 作用域 | 标识 | 说明 |
|--------|------|------|
| 强制 | `mandatory` | 所有用户必须执行，不可关闭 |
| 默认启用 | `default_on` | 默认启用，用户可个人关闭 |
| 默认禁用 | `default_off` | 默认禁用，用户可个人启用 |

### 8.3 规则合并逻辑

**文件位置**：`go-service/internal/service/rule_merge.go`

最终送入 AI 的规则列表 = 租户通用规则（merged with 用户个人覆盖） + 用户自定义规则。

---

## 九、Python AI 服务（规划中）

`docker-compose.yml` 中定义了 `ai-service` 容器，用于承载 Python AI 服务：

```yaml
ai-service:
  build: ./ai-service
  ports: ["8000:8000"]
  environment:
    - LLM_PROVIDER=local
    - LLM_MODEL_NAME=default
```

Go 后端通过 `ChatViaPython()` 方法调用：
```
POST http://ai-service:8000/api/v1/chat/completions
```

**当前状态**：`ai-service/` 目录下暂未有实际代码，该服务尚未实现。Go 后端的 `Chat()` 方法可直连模型，无需 Python 中转。

---

## 十、Token 消耗追踪

**表名**：`tenant_llm_message_logs`

| 字段 | 说明 |
|------|------|
| `tenant_id` | 所属租户 |
| `user_id` | 发起用户（NULL = 系统自动） |
| `model_config_id` | 使用的模型 |
| `request_type` | 请求类型：audit/archive/other |
| `input_tokens` | 输入 Token |
| `output_tokens` | 输出 Token |
| `total_tokens` | 总 Token |
| `duration_ms` | 调用耗时 |

### Token 统计 API

| API | 权限 | 说明 |
|-----|------|------|
| `GET /api/tenant/stats/token-usage` | 租户管理员 | 查询本租户 Token 消耗 |
| `GET /api/admin/stats/token-usage` | 系统管理员 | 查询所有租户 Token 消耗 |

---

## 十一、已知问题与注意事项

1. **Python AI 服务未实现**：`ai-service` 容器尚无代码，当前仅支持 Go 直连模型。需要 RAG/向量检索等复杂功能时需实现此服务。

2. **备用模型切换未实现**：`fallback_model_id` 字段已定义，但运行时自动切换逻辑尚未编码。

3. **提示词模板固定**：系统提示词模板目前为只读（租户管理员通过 `GET /api/tenant/rules/prompt-templates` 查看），租户不可自定义系统级提示词。

4. **提取阶段输出解析**：提取阶段要求 LLM 输出纯 JSON，但实际运行中 LLM 可能输出额外文字或格式不正确的 JSON，目前缺少健壮的 JSON 解析与容错逻辑。

5. **审核执行完整闭环尚未打通**：当前后端已具备 AI 调用、Token 管理、日志记录的能力，但从「拉取 OA 数据 → 构建提示词 → 调用 AI → 解析结果 → 写入审核日志」的完整链路尚未在 Handler 层串联起来。前端审核工作台（`overview.vue`）使用的是 Mock 数据。
