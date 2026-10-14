# 评论反馈记忆库

权限：JWT → TenantContext → `tenant_admin` → `/admin/tenant/memory` 页面权限。
前缀：`/api/tenant/feedback-memory`。成功响应沿用 `code: 0`、`data`。所有操作仅在当前租户内执行。

## 工作流与生效范围

页面沿用记忆库工作区布局：左侧切换审核/归档并搜索、选择流程，右侧通过反馈素材、生成记忆、版本记录切换内容；生成规则与提示词通过只读抽屉查看。

1. 选择业务模块 `audit` / `archive` 和已配置的 `process_type`（流程类型标识，不是实例 ID）。
2. 分页选择该流程类型的审核评论导入，或逐条添加手动条件。评论保存原文、作者、流程实例、审核批次、更新时间与满意度快照；原文只读，可调整适用条件与启停。原评论之后被修改或删除，不改写此快照。
3. AI 以上一**最新版本**（草稿或已发布版，含人工调整）为基线，只将启用且尚未植入的素材作为新输入。`base_version_id` 指向实际基线；停用不会清空此基线。上一版条目放入 `previous_items`，依据原文标记 `inherited=true`，只供核实旧记忆，禁止重新提炼、恢复人工移除的条目。相似建议按适用范围合并为一条，`source_refs` 保留多个稳定的原始素材引用；旧 `previous:*` 引用会递归展开，原始评论快照仍可追溯。
4. 每次生成保存独立版本号、生成时点、完整来源、逐条 `source_refs`、待确认事项、实际生成输入和系统提示词。生成规则只读，不能通过接口编辑。来源引用必须在本次输入中存在，空内容、虚构引用和超长输出会拒绝，已完成的 AI 调用仍保留用量和原始响应。
5. 管理员可以整体编辑草稿的条件、正文、条目数量和来源引用，实时预览最终记忆正文。草稿保存使用 `edit_revision` 防止并发覆盖；编辑已发布版会另存为新版草稿，历史正文和 `generated_items` 原始AI输出不变。发布校验素材 `revision`、版本是否最新及用户审阅的 `edit_revision`；不匹配返回409。停用只设置 `enabled=false`，保留发布指针，可直接恢复最后发布版。
6. 成功生成且被条目实际引用的素材自动标记 `implanted_version_id`，不再出现参与下次生成或编辑素材按钮，由记忆条目统一维护；没有被引用的素材仍待整合。已植入素材仅可删除。删除保留撤销标记和历史快照，用户评论回到可导入列表，手动素材从列表移除。下一次输入的 `deleted_sources` 告知AI撤销仅由被删素材支持的内容；这些引用也被服务端排除，不允许AI正向引用。重新导入评论使用新的素材ID，避免恢复旧的已撤销身份。修改或删除素材不直接改写当前发布版，要在整合、检查和发布后生效，立即停止注入可以停用记忆。
7. 导入评论列表始终在服务端分页与总数统计前排除已导入评论（“全部建议”筛选也不例外），删除素材后才重新出现。其余建议支持逐条弃用、恢复，默认隐藏已弃用建议，弃用状态在租户、业务模块与流程间独立。不改写原评论，不撤销已导入素材；已导入素材的撤销使用素材删除入口。

审核与归档按 `(tenant_id, module, process_type)` 独立维护。归档可选择同流程类型的审核评论作为素材，两侧生成和发布互不影响。当前系统没有归档评论表，本功能不虚构归档评论入口，也不混入智能体对话反馈。

## 接口

所有工作区、评论、版本读接口使用 query `module`（必填，`audit|archive`）和 `process_type`（必填，最长200字）。写入接口把这两个字段放 JSON，删除素材用 query。

| 方法 | 路径 | 请求 / 响应 |
|---|---|---|
| GET | `/configs?module=audit` | 当前租户配置过的流程类型数组：`process_type, process_type_label` |
| GET | `` | `{memory, entries, published, baseline, system_prompt, user_prompt_template}`；未创建时 `memory/published/baseline=null, entries=[]`，读取不创建记忆库；停用后published仍返回最后发布版本 |
| GET | `/comments` | 当前类型审核评论的服务端分页；支持 `keyword`、`feedback=like|dislike|comments`、`deprecation=active|deprecated|all`（默认active）、`page`、`page_size` |
| PUT | `/comments/:id` | `{module, process_type, deprecated: boolean}`；弃用或恢复建议，后端校验评论租户与流程 |
| POST | `/import` | `{module, process_type, comment_ids: UUID[]}`，1–100条；后端校验租户和流程后读取原文，同一记忆库不重复导入 |
| POST | `/entries` | `{module, process_type, condition, content, enabled}`，新增手动素材 |
| PUT | `/entries/:id` | 同上；评论原文不接受覆盖，未植入时允许修改适用条件与启停；已植入返回409 |
| DELETE | `/entries/:id` | 移除素材，保留删除标记与已生成历史，评论重新可导入 |
| POST | `/generate` | `{module, process_type}`；返回完整 `FeedbackMemoryVersion`，不自动发布 |
| POST | `/preview` | `{module, process_type}`；返回 `{system_prompt, user_prompt, base_version_id}`，与生成共用构建、脱敏逻辑，不消耗Token；基于读取时状态，之后素材变化会改变输入 |
| GET | `/versions` | 版本元数据服务端分页，按版本号倒序；大字段通过详情读取 |
| GET | `/versions/:id` | 同租户、同模块和流程下的完整版本；原生成输入、原AI条目与发布后正文不可变 |
| PUT | `/versions/:id` | `{module, process_type, items: [{condition, content, source_refs}], edit_revision}`；保存最新草稿，或从最新发布版另存新草稿，返回完整版本；有旧基线时可显式确认发布空草稿以撤销最后一条记忆，首版空结果不能发布 |
| POST | `/versions/:id/publish` | `{module, process_type, edit_revision}`；发布已审阅的最新草稿，更新发布时间；旧客户端省略edit_revision按0校验 |
| POST | `/disable` | `{module, process_type}`；关闭运行时注入，保留发布指针、素材修订号与生成基线 |
| POST | `/enable` | `{module, process_type}`；恢复最后发布版本，无需重新生成；未曾发布返回404 |

分页统一返回 `{items: [], total, page, page_size}`。`page` 默认1，小于1回落1；`page_size` 默认20，仅允许1–100，越界回落20。筛选先于分页；前端修改每页条数重置为第一页。

素材上限每流程200条。`content` 1–2000字，`condition` 最多1000字；生成输入最多160000字节；输出最多80条，最终记忆提示词最多16000字。所有 UUID 在入口解析，外租户或外流程记录按不存在返回。

## 数据结构

- `memory`：`id, revision, last_version, published_version_id, enabled, module, process_type, created_at, updated_at`。
- `entries[]`：`id, memory_id, source_type(manual|comment), source_id, condition, content, enabled, implanted_version_id, source_snapshot, created_by, created_at, updated_at`，仅包含未删除素材。
- 评论列表：`id, audit_log_id, process_id, title, username, content, feedback, updated_at, imported, deprecated`。
- 版本：`id, memory_id, version_no, base_version_id, input_revision, status(draft|published), sources, items, generated_items, edit_revision, edited_by, edited_at, warnings, system_prompt, user_prompt, prompt_text, created_by, created_at, published_at`。
- `sources[]`：`ref, source_type(manual|comment|previous), condition, content, snapshot`。评论/手动源的 snapshot 是当时的完整素材，previous 源是上一版条目。
- `items[]`：`condition, content, source_refs: string[]`。`source_refs` 与本次 `sources[].ref` 对应。
- `sources[].inherited=true` 表示旧记忆依据，仅用于核实、追溯；`deleted_sources` 在生成输入中单列，不能作为条目引用。
- `status=published` 表示曾发布；**当前生效**要求 `memory.enabled=true` 且指向 `memory.published_version_id`。

页面主操作“审核记忆预览”展示实际执行共用注入函数返回的 `runtime_user_prompt`（记忆段含版本与生成时间）和 `runtime_system_prompt`（随记忆追加的规则优先约束），并区分当前生效版与待发布版。完整审核请求还包含当次流程事实、正式规则和模板。“预览整合输入”仅位于生成过程追溯中，用于查看记忆整理AI的资料。整理系统提示词、完整整合输入和来源快照都不作为运行时记忆正文重复注入。发布、停用和素材删除使用居中确认弹窗，避免锚定浮层随按钮位置漂移；历史版本统一在版本记录中浏览。

## 提示词接入

审核和归档在推理分析阶段自动追加当前启用的已发布反馈记忆，无需用户编辑提示词或插入变量，适用于新旧执行配置版本。旧模板残留的 `{{feedback_memory}}` 会先移除，再在用户请求末尾统一追加一次记忆，避免重复注入；未启用时只清理旧变量，不追加记忆正文。

两阶段指“先生成审核分析，再将分析提取为规定的JSON结构”。记忆只在前一步参与分析，后一步接收已有分析结果和规则，不再次提供记忆正文，也不得重新审核、引入新依据或丢失前一步已采纳的意见。分析提示词包含记忆版本号、版本ID、生成时点和逐条条件，正式规则、权限、事实和输出格式优先。

反馈记忆是**运行时参考资料**，与流程已绑定的执行配置版本独立：发布会影响同类型流程下一次实际执行，不改变历史结论，也不会自动重审。已有流程沿用旧执行配置时仍使用当前启用的发布记忆；实际用到的记忆版本保存在推理调用提示词中。发布与执行并发时，单次分析使用读取时的发布版，结构化阶段沿用这次分析结果。

## AI 记录与用量

生成通过 `AIModelCallerService.ChatWithFallback`，租户主/备用模型、配额、重试和统一脱敏策略保持一致。

- `request_type=feedback_memory`，`call_type=structured`。
- `business_log_id` 为本次生成版本UUID，`process_id` 不伪造 OA 流程ID。
- AI 已返回但生成结构校验失败时，调用仍保留在 AI 调用记录中，不产生可发布版本。
- Token 用量、仪表盘总体调用次数/耗时和模型消耗由统一 LLM 日志全场景聚合，包含反馈记忆、SQL辅助和规则梳理；不额外增加审核完成数。

迁移 `000085_feedback_memory` 提供基础表；`000088_feedback_memory_lifecycle` 提供启停、植入状态、删除标记、草稿编辑和评论弃用。兼容修复旧停用逻辑丢失的最后发布指针（保持停用），并回填旧版本已引用素材的植入状态。新表、字段和索引有中文注释和回滚脚本。

业务错误码：`40006` 素材超过200条，`40007` 草稿没有可发布记忆，`40904` 素材修订号或发布基线变化。前端提供对应中英文提示。
