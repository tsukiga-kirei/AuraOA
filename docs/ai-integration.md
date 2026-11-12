# AI 系统对接说明

## 概述

AuraOA 通过统一的 AI 调用层对接多种大语言模型（LLM），支持本地部署和云端 API 两种模式。AI 系统在审核流程中承担核心推理角色，采用**两阶段审核**架构（推理 → 结构化提取），确保审核结论的准确性和可解释性。

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
- **Token 统计**：非流式模式从响应 `usage` 字段获取精确值；流式模式优先通过 `stream_options.include_usage=true` 获取最终 usage chunk，未返回时保持空值（不做估算）

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

## 思考模型（Reasoning / Thinking Models）与深度思考支持

系统全链路支持具备原生思考链（CoT）能力的深度思考模型（如 DeepSeek-R1、Qwen-QwQ 等）：

1. **模型标注**：在系统管理 AI 模型配置中提供 `supports_thinking` 开关，标识模型是否支持原生思考过程。
2. **规则配置**：在流程审核配置、归档复盘配置及流程总结各块配置的提示词部分提供 `enable_thinking` 开关。开启后，在向模型发起请求时自动注入 `chat_template_kwargs: {"enable_thinking": true, "thinking": true}`（vLLM / Jinja 模板标准参数）。
3. **思考过程解析与落库**：
   - AI 调用器兼容解析 vLLM 新版 `reasoning` 字段以及 DeepSeek / 阿里百炼 / 历史 vLLM `reasoning_content` 字段。
   - 流式请求时支持独立分发思考增量（`StreamReasoningChunkFunc`）与回答增量（`StreamChunkFunc`）。
   - 审核与归档把模型增量累积后，以完整快照发送 SSE 的 `message`（推理）和 `thinking`（思考）事件；重连覆盖、重试清空，完成后以持久化结果为准，避免重复拼接。
   - 深度思考内容自动落库至 `tenant_llm_message_payloads.reasoning_content`、`audit_logs.deep_thinking`、`archive_logs.deep_thinking` 以及总结块结果，并在前端数据管理、审核工作台、归档复盘和 OA 嵌入页完整展示。

## 两阶段审核流程

> 从触发开关、配置解析到最终结论的完整路径见 [审核链路与决策路径](./audit-review-path.md)。

AI 审核采用两阶段架构，将推理和结构化提取分离，提高审核质量：

```
┌─────────────────────────────────────────────────────────────────┐
│                     两阶段 AI 审核流程                           │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  阶段一：推理（Reasoning）                                       │
│  ├── 输入：系统提示词 + 用户提示词（含流程数据、规则、审批流）      │
│  ├── 思考模式：模型支持且配置开启时激活深度思考                   │
│  ├── 温度：租户配置值（默认 0.3）                                 │
│  ├── 输出：自然语言推理分析文本（含深度思考过程）                 │
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
| `{{attachments}}` | 被最终字段配置选中的主表附件解析文本；未选中的附件不会下载或识别 |
| `{{rules}}` | 合并后的审核规则文本 |
| `{{current_node}}` | 当前审批节点名称（提示词未引用审批流且无关联规则时使用默认文本） |
| `{{flow_history}}` | 审批流历史文本（提示词引用审批流/当前节点，或存在 `related_flow` 的启用规则时才拉取） |
| `{{flow_graph}}` | 审批流路由图文本（同上） |

审核与归档在字段选择范围内应用空值传输策略：`ai_config.skip_empty_fields=true` 统一省略
`null`、空字符串和纯空白；关闭后仅省略 `skip_empty_field_keys` 中指定字段的空值。
引用格式为 `main:字段标识` 或 `明细表名:字段标识`；`0` 和 `false` 保留，明细逐行处理且保留行位置。
仅生成提示词副本，OA 数据、业务快照及业务数据变化指纹不受过滤影响。策略由现有执行配置快照携带，
旧快照缺省不启用过滤；保存、发布、激活及旧流程版本绑定规则保持不变。

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
| `loose` | 宽松模式 — 对有证据确认无实质影响的一般瑕疵减少扣分，仍检查关键错误与疑点 |

审核尺度的优先级：管理员允许个人调整时，用户个人配置 > 租户配置 > 系统默认值。个人尺度会切换对应尺度的完整推理与结构化提示词，并随个人版本写入最终执行快照；仅修改尺度字段而继续使用原提示词不视为生效。每份默认提示词只写所选尺度对应的具体核查、证据和扣分要求，不在正文中并列介绍其他尺度。

### 问题定性与评分

审核与归档采用新版系统模板的推理、提取提示词遵循以下要求：

- 规则检查与主表、明细、附件及审批上下文的独立核验共同决定结论。规则全部通过不能抵消独立发现的实质问题；规则不适用只豁免该规则，不得因此虚构或豁免独立问题。
- 存在未通过项或尚未排除的实际风险时不能通过/合规，也不能给80分及以上。确认且需要整改的问题判为退回/不合规（0–59分）；事实冲突或证据不足、需要核实的问题判为复核/部分合规（60–79分）；没有未通过项且无未解决风险时，才能通过/合规（80–100分）。归档节点或字段未通过也参与判断。先确定问题与结论，再在对应区间给分；纯优化建议不作为风险或未通过项。
- 金额先核实单位、币种和业务口径。不得默认附件正确，也不得仅以“非恶意、未造成损失、可补正”为由认定无害笔误；确认同口径一致时排除问题，确认关键字段错误时补正，无法确认时复核。
- 扣分取决于问题的业务影响、证据与整改必要性，不按规则总数、通过率或不适用项数量分摊。同一事实重复发现只扣一次，纯优化建议不扣分；严格、标准、宽松只调整一般瑕疵容忍度。
- 提取阶段仅根据已有分析与规则工作，不重新审核原始业务或补造事实；若分析已明确关键问题未解决，却给高分通过，应依据已有分析统一修正扣分、分数与结论，不能删除问题来保留高分。

这些是发送给模型的提示词约束，不是程序对自然语言风险的语义校验，不保证模型一定判断正确。现有 JSON 字段、解析与分数校准逻辑保持不变。

迁移 `000091_review_problem_driven_scoring` 仅更新十二条系统模板；`000093_review_prompt_scale_requirements` 按判断、风险、扣分与结论的一致性重写十二条系统模板和四条复核模板，各尺度分别写自己的具体要求，不并列介绍尺度。原有十二条推理/提取用户模板保持原文。这些迁移不在请求发送时追加新的评分要求，不修改已有流程配置、不可变执行快照或历史结果。新建配置可使用新版默认模板；已有配置需采用新版提示词并按现有机制保存、发布。流程可继续沿用绑定的旧版本重新执行，也可在新版配置发布后明确选择最新版本。实际发送内容仍记录于 `tenant_llm_message_payloads`；升级系统模板本身不会自动切换流程的执行版本或重新审核。完整默认模板及运行时附加内容见[提示词审核稿](./reviews/2026-10-26-audit-archive-prompts-for-review.md)。

真实模型回放场景与验收口径见 [问题定性评分验证](./reviews/review-problem-driven-scoring.md)。

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
- 请求类型（`audit` / `archive` / `summary`）
- 调用阶段（`reasoning` / `structured`）
- 输入/输出/总 Token 消耗
- 调用耗时（毫秒）

同时写入 `tenant_llm_message_payloads` 表，按 `tenant_llm_message_logs.id` 一对一保存本次调用实际发送给模型的 `system_prompt`、`user_prompt`，以及模型返回的 `response_content`。大文本内容与统计表拆分，避免仪表盘和用量报表查询扫描提示词正文。

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
| 流式 Token 精确统计 | 🔧 部分实现 | 已发送 `stream_options.include_usage=true` 并解析流末尾 usage chunk；对不返回 usage 的兼容服务保持空值（不估算） |
| Python AI 服务集成 | 🔧 部分实现 | Go 侧调用通道已就绪，但 Python 服务本身尚未开发 |
| RAG 知识库模式 | ❌ 未实现 | 配置中已有 `kb_mode` 字段，但知识库检索增强功能尚未开发 |
| 模型能力匹配 | ❌ 未实现 | `ai_model_configs.capabilities` 字段已定义，但未根据能力标签自动匹配审核任务 |
| 原生多模态审核 | ❌ 未实现 | 附件可先经内置解析、代码文档解析或 MinerU 转成文本后参与审核；尚未把图片原图直接发送给多模态模型 |
| 审核结果缓存 | 🔧 部分实现 | 快照表已支持历史审核结果存储，但未实现相同数据的审核结果复用 |

### 服务商层面

| 服务商 | 状态 | 说明 |
|--------|------|------|
| 百度文心一言 | ❌ 未注册 | 需添加 provider 选项和默认 endpoint |
| 腾讯混元 | ❌ 未注册 | 需添加 provider 选项和默认 endpoint |
| Anthropic Claude | ❌ 未注册 | API 格式与 OpenAI 不完全兼容，可能需要独立调用器 |
| Google Gemini | ❌ 未注册 | API 格式与 OpenAI 不兼容，需要独立调用器 |

> 注意：当前实现采用统一 OpenAI Chat Completions 协议解析（含流式 chunk）。若某厂商返回格式与该协议不兼容或有私有扩展字段语义差异，需要新增独立调用器或增加厂商特化解析逻辑。

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

对话场景（尚未落地）使用同一调用层，`request_type=chat`，并走智能体有效工具集。需求见 [`docs/agents/README.md`](./agents/README.md)；配额由系统管理员授予租户，与「关联租户主备模型」同一层级。

## 审核数据 SQL 辅助

查询库新增 `sql_assist` 场景，`call_type=structured`，经统一 `AIModelCallerService.ChatWithFallback` 使用租户主/备用模型、配额及完整调用日志。每次辅助使用唯一业务标识，数据管理页单独筛选和计数；仪表盘总体 Token/费用包含此场景，不增加审核业务次数。

租户可编辑生成、解释、修复三种业务提示词，固定系统约束保留。AI 仅返回待采用的 SQL 草稿和说明；不执行、不发布。查询取数本身不调用 LLM。审核和归档在推理阶段将查询证据按规则组织，附中文列名 JSON、查询版本、用途和完整性提示，失败或截断信息交由 AI 判断业务影响，不触发程序强制降分或改判。参见 [查询库 API](./api/audit-data-queries.md)。

## 规则适用前提

审核工作台与归档复盘规则的 `applicability_condition` 为可选自然语言前提。为空时按原逻辑校验规则；满足时检查正文；明确不满足时直接通过（`passed=true`），原因以前提不满足的说明开头，不扣分、不增加风险。缺少判断前提所需的信息时，按既有尺度处理，不能擅自视为前提不满足。

两阶段请求均保留正文与前提，并在系统提示词末尾追加统一说明，兼容已有租户自定义提示词。结果字段与通过/不通过两种状态不变。关联查询失败或截断信息随已有请求传给 AI，由 AI 判断影响；程序不再按查询状态改写结论或规则结果，也不追加人工复核风险。

AI 文件/粘贴导入将原文明示的适用范围或触发条件提取为独立前提，正文保留实际检查要求、阈值、例外与否决条件。草稿及确认导入均保留该字段，去重同时比较正文与前提。


## 评论反馈记忆与辅助操作记录

反馈记忆支持按审核/归档流程类型收集用户评论、补充手动条件，以最新草稿或发布版（含人工调整）进行 AI 增量整合。已植入素材不重复作为新输入，旧记忆保留稳定原始来源；删除素材通过撤销标记传递给AI。生成结果可整体编辑、预览后发布，原始AI输出与生成输入留存；停用保留发布版并可恢复。发布且启用后自动追加到分析请求，适用于所有新旧执行配置，无需人工插入变量。结构化阶段只提取已有分析，不重复传入记忆正文或重新审核。正式规则优先；记忆发布不自动重审、不改写历史结论。版本和执行配置独立，详见 [反馈记忆 API](./api/feedback-memory.md)。

反馈记忆 `feedback_memory`、规则梳理 `rule_assist` 和数据记忆 `sql_assist` 均经 `ChatWithFallback`。规则梳理旧实现已有 Token 与 payload 日志，但错误归类为 audit/archive；现在新调用独立分类，查询层兼容识别旧 `rule-import:` 记录。辅助操作用独立业务ID归集，不冒充 OA 流程。全场景 Token 汇总与仪表盘性能统计自然包含新场景，不计入审核业务完成量。

## 可选第三阶段 AI 复核

迁移 `000095_unparsed_attachment_prompt_policy` 为审核、归档标准与宽松尺度的推理及结构化
系统模板增加附件解析失败政策：已获取但正文无法解析的附件暂按无问题处理，不仅因此扣分、
增加风险或要求复核；附件未提供、获取失败以及其他材料已证实的问题仍按原规则判断。
该行为仅由提示词指导，不增加程序强制评分或改判，严格尺度保持原样。已有配置需采用新版
模板并保存、发布；已绑定执行版本和历史结果不自动覆盖。

配置 `ai_config.recheck` 随执行配置版本保存，缺省关闭。默认复核系统提示词按当前审核尺度区分具体要求，用户可继续编辑；切换尺度只调整未自定义的默认要求，不覆盖自定义提示词或已有版本。000094迁移更新通用复核默认内容并新增六条按尺度区分的系统模板，不改已有配置或历史快照。默认用户输入为 `{{rules}}` 与 `{{initial_result}}`，原材料、分析和原系统要求仅在引用可选变量时传入。

复核重点是规则条件、通过状态、理由、风险、分数与结论的内部一致性，不因缺少原始材料重新认定已通过规则。无需纠正时仅返回 `{"reuse_original":true}`；有明确修改时返回false标记、说明和完整结果。后端按共享固定结构严格解码，拒绝重复键、前后缀、未知字段、错误类型和无效数值，并检查覆盖及分数关系；随后按最终业务内容比较，忽略纯排版和条目顺序变化。通用工作台仅有变化时显示最终调整卡片；生产嵌入页在分析之后保留复核页签，区分沿用、调整和未完成，不重复展示结构化全文。

格式约定由后端生成，经模板列表接口的只读 `output_instructions` 返回，同时原样追加到实际系统请求。界面把这段实际原文和不发送给AI的页面说明分开展示。JSON Schema包含“只沿用”和“完整修改”两种互斥分支。模型调用、配额和日志仍通过 `AIModelCallerService.ChatWithFallback`，业务类别保持audit/archive，阶段为recheck。调用或输出无效时拒绝复核结果，采用已验证初审，保留复核失败状态及原因；取消仍禁止发布新结果。审核嵌入页可创建独立复核任务，复用当次规则与输入，不重新取 OA/附件或执行初审与提取。

分析、结构化和复核均支持现有SSE链路，后两者发送完整阶段快照并提供轮询及重连缓存。下一阶段开始输出时上一块默认收起，用户手动展开或收起的选择持续保留；嵌入生成中的面板标题与完成态一样吸顶。流式JSON不被提前当成正式结果。分析Markdown在每次更新时渲染，兼容模型紧贴文字的标题，同时保留代码围栏和HTML消毒。只有初审分析阶段按配置开启独立深度思考；结构化与复核不设置 `EnableThinking`，也不接收单独的深度思考流。

完整默认模板、各尺度要求和两套实际输出约定见[提示词审核稿](./reviews/2026-10-26-audit-archive-prompts-for-review.md)。更多契约见[审核配置](./api/audit-config.md#可选-ai-复核第三阶段)与[执行留痕](./api/audit.md#ai-复核执行及留痕)。
