# AI 模型调用 — 接口与 Provider 参考

> 对应代码：`go-service/internal/pkg/ai/`

---

## 架构概览

所有 AI 模型调用统一走 `OpenAICompatCaller`，通过 OpenAI Chat Completions 兼容协议。
`factory.go` 根据 `ai_model_configs.provider` 字段分发，而非 `deploy_type`。

```
ai_model_configs.provider
  → factory.NewAIModelCaller()
  → OpenAICompatCaller (统一 HTTP 调用)
```

---

## 支持的 Provider

### 本地部署 (deploy_type = "local")

| provider | 默认 endpoint | 需要 api_key | 备注 |
|---|---|---|---|
| `xinference` | 需手动配置 | 可选 | Xorbits Inference |
| `ollama` | 需手动配置 | 可选 | 通常 `http://host:11434/v1` |
| `vllm` | 需手动配置 | 可选 | vLLM OpenAI 兼容服务 |

### 云端 API (deploy_type = "cloud")

| provider | 默认 endpoint | 需要 api_key | 备注 |
|---|---|---|---|
| `aliyun_bailian` | `https://dashscope.aliyuncs.com/compatible-mode/v1` | 必须 | 阿里云百炼 DashScope |
| `deepseek` | `https://api.deepseek.com/v1` | 必须 | DeepSeek API |
| `zhipu` | `https://open.bigmodel.cn/api/paas/v4` | 必须 | 智谱 AI |
| `openai` | `https://api.openai.com/v1` | 必须 | OpenAI 官方 |
| `azure_openai` | 需手动配置 | 必须 | Azure 部署的 OpenAI |

> 如果 `ai_model_configs.endpoint` 有值，优先使用配置值，覆盖默认 endpoint。

---

## API 调用格式

### TestConnection — 连接测试

```
GET {endpoint}/models
Authorization: Bearer {api_key}   (如果有)
```

返回 200 表示连接正常，401 表示 API Key 无效。

### Chat — 对话请求

```
POST {endpoint}/chat/completions
Content-Type: application/json
Authorization: Bearer {api_key}   (如果有)

{
  "model": "{model_name}",
  "messages": [
    {"role": "system", "content": "..."},
    {"role": "user", "content": "..."}
  ],
  "temperature": 0.3,
  "max_tokens": 8192
}
```

### 响应格式

```json
{
  "choices": [
    {"message": {"role": "assistant", "content": "..."}}
  ],
  "usage": {
    "prompt_tokens": 100,
    "completion_tokens": 200,
    "total_tokens": 300
  }
}
```

---

## Go → Python 代理调用 (ChatViaPython)

当需要 Python AI 服务处理（RAG、LangChain 等）时，Go 通过 HTTP 转发：

```
POST {AI_SERVICE_URL}/api/v1/chat/completions

{
  "system_prompt": "...",
  "user_prompt": "...(已脱敏)",
  "model_config": {
    "model_id":    "uuid",
    "provider":    "aliyun_bailian",
    "deploy_type": "cloud",
    "model_name":  "qwen-plus",
    "endpoint":    "https://...",
    "api_key":     "sk-...",
    "max_tokens":  8192,
    "temperature": 0.3
  },
  "audit_context": { ... }
}
```

---

## Token 配额管理

采用预扣-结算模式，防止并发超额：

1. 调用前：原子预扣 `max_tokens`（`WHERE token_used + ? <= token_quota`）
2. 调用后：结算差额（`token_used = token_used - reserved + actual`）
3. 调用失败：回滚预扣（`token_used = token_used - reserved`）

每次调用的 token 消耗记录写入 `tenant_llm_message_logs`，包含 `tenant_id` + `model_config_id`，支持按租户+模型维度统计。
