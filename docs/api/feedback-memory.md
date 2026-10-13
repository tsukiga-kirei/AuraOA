# 评论反馈记忆库

权限：JWT → TenantContext → `tenant_admin` → `/admin/tenant/memory` 页面权限。
前缀：`/api/tenant/feedback-memory`。成功响应沿用 `code: 0`、`data`。所有操作仅在当前租户内执行。

## 工作流与生效范围

页面沿用记忆库工作区布局：左侧切换审核/归档并搜索、选择流程，右侧通过反馈素材、生成记忆、版本记录切换内容；生成规则与提示词通过只读抽屉查看。

1. 选择业务模块 `audit` / `archive` 和已配置的 `process_type`（流程类型标识，不是实例 ID）。
2. 分页选择该流程类型的审核评论导入，或逐条添加手动条件。评论保存原文、作者、流程实例、审核批次、更新时间与满意度快照；原文只读，可调整适用条件与启停。原评论之后被修改或删除，不改写此快照。
3. AI 使用当前启用素材与上一**已发布版**的逐条记忆生成新草稿。上一版每条记忆作为 `previous:<版本UUID>:<条目序号>` 来源，沿 `base_version_id` 可递归回看原始依据。未发布草稿不会作为增量基线。
4. 每次生成保存独立版本号、生成时点、完整来源、逐条 `source_refs`、待确认事项、实际生成输入和系统提示词。生成规则只读，不能通过接口编辑。来源引用必须在本次输入中存在，空内容、虚构引用和超长输出会拒绝，已完成的 AI 调用仍保留用量和原始响应。
5. 管理员检查后显式发布。发布校验素材 `revision` 和上一已发布版本是否仍匹配；不匹配返回冲突，必须重新生成。没有可靠条目的草稿不能发布。可停用当前发布版，历史版本永久保留。
6. 修改、启停或删除素材**不直接改变已发布记忆**。要纠正旧记忆，应增加明确的纠正素材，重新生成并发布；需要立即停止使用则停用记忆。重新生成始终综合上一已发布版，不能把关闭某条素材误解为撤销历史记忆。

审核与归档按 `(tenant_id, module, process_type)` 独立维护。归档可选择同流程类型的审核评论作为素材，两侧生成和发布互不影响。当前系统没有归档评论表，本功能不虚构归档评论入口，也不混入智能体对话反馈。

## 接口

所有工作区、评论、版本读接口使用 query `module`（必填，`audit|archive`）和 `process_type`（必填，最长200字）。写入接口把这两个字段放 JSON，删除素材用 query。

| 方法 | 路径 | 请求 / 响应 |
|---|---|---|
| GET | `/configs?module=audit` | 当前租户配置过的流程类型数组：`process_type, process_type_label` |
| GET | `` | `{memory, entries, published, system_prompt, user_prompt_template}`；未创建时 `memory/published=null, entries=[]`，读取不创建记忆库 |
| GET | `/comments` | 当前类型审核评论的服务端分页；支持 `keyword`、`feedback=like|dislike|comments`、`page`、`page_size` |
| POST | `/import` | `{module, process_type, comment_ids: UUID[]}`，1–100条；后端校验租户和流程后读取原文，同一记忆库不重复导入 |
| POST | `/entries` | `{module, process_type, condition, content, enabled}`，新增手动素材 |
| PUT | `/entries/:id` | 同上；评论原文不接受覆盖，允许修改适用条件与启停 |
| DELETE | `/entries/:id` | 移除素材，保留已生成历史 |
| POST | `/generate` | `{module, process_type}`；返回完整 `FeedbackMemoryVersion`，不自动发布 |
| GET | `/versions` | 版本元数据服务端分页，按版本号倒序；大字段通过详情读取 |
| GET | `/versions/:id` | 同租户、同模块和流程下的完整不可变版本 |
| POST | `/versions/:id/publish` | `{module, process_type}`；切换当前生效版本，更新发布时间 |
| POST | `/disable` | `{module, process_type}`；清空当前生效指针，递增修订号使旧草稿失效 |

分页统一返回 `{items: [], total, page, page_size}`。`page` 默认1，小于1回落1；`page_size` 默认20，仅允许1–100，越界回落20。筛选先于分页；前端修改每页条数重置为第一页。

素材上限每流程200条。`content` 1–2000字，`condition` 最多1000字；生成输入最多160000字节；输出最多80条，最终记忆提示词最多16000字。所有 UUID 在入口解析，外租户或外流程记录按不存在返回。

## 数据结构

- `memory`：`id, revision, last_version, published_version_id, module, process_type, created_at, updated_at`。
- `entries[]`：`id, memory_id, source_type(manual|comment), source_id, condition, content, enabled, source_snapshot, created_by, created_at, updated_at`。
- 评论列表：`id, audit_log_id, process_id, title, username, content, feedback, updated_at, imported`。
- 版本：`id, memory_id, version_no, base_version_id, input_revision, status(draft|published), sources, items, warnings, system_prompt, user_prompt, prompt_text, created_by, created_at, published_at`。
- `sources[]`：`ref, source_type(manual|comment|previous), condition, content, snapshot`。评论/手动源的 snapshot 是当时的完整素材，previous 源是上一版条目。
- `items[]`：`condition, content, source_refs: string[]`。`source_refs` 与本次 `sources[].ref` 对应。
- `status=published` 表示曾发布；**当前生效**以 `memory.published_version_id` 为准。

## 提示词接入

审核和归档的推理、结构化提取阶段都支持 `{{feedback_memory}}`，系统和用户提示词均可使用。旧模板没有变量时自动追加到用户提示词。未启用时变量替换为“当前流程未启用反馈记忆”，不额外添加记忆内容。

每次执行在推理前读取当前发布记忆，两个阶段共享同一份内容；提示词包含记忆版本号、版本ID、生成时点和逐条条件。正式规则、权限、事实和输出格式优先，反馈不能覆盖强制规则。

反馈记忆是**运行时参考资料**，与流程已绑定的执行配置版本独立：发布会影响同类型流程下一次实际执行，不改变历史结论，也不会自动重审。已有流程沿用旧执行配置时仍使用当前发布记忆；实际用到的记忆版本保存在该次 AI 调用提示词中。发布与执行并发时，单次执行使用读取时的发布版，不在两个阶段之间换版。

## AI 记录与用量

生成通过 `AIModelCallerService.ChatWithFallback`，租户主/备用模型、配额、重试和统一脱敏策略保持一致。

- `request_type=feedback_memory`，`call_type=structured`。
- `business_log_id` 为本次生成版本UUID，`process_id` 不伪造 OA 流程ID。
- AI 已返回但生成结构校验失败时，调用仍保留在 AI 调用记录中，不产生可发布版本。
- Token 用量、仪表盘总体调用次数/耗时和模型消耗由统一 LLM 日志全场景聚合，包含反馈记忆、SQL辅助和规则梳理；不额外增加审核完成数。

新增迁移 `000085_feedback_memory` 提供三张表及全部字段、索引的中文数据库注释，并提供回滚脚本。

业务错误码：`40006` 素材超过200条，`40007` 草稿没有可发布记忆，`40904` 素材修订号或发布基线变化。前端提供对应中英文提示。
