**AuraOA：2026 年 9 月 1 日以来提交审查**

审查日期：2026-09-05。审查版本：`c794812`，比较基线：`9dc6ecd`（9 月 1 日首个提交 `86b9964` 的父提交）。共 13 个提交、184 个变更文件，新增 15,836 行、删除 555 行。开始时工作区无未提交修改。本次只审查并记录，没有修改业务代码或生产数据。

**结论：目前不建议整体通过验收。** 新功能方向符合“AI 辅助 OA 审核、总结、可追溯”的产品定位，文档解析和深度思考已有较完整实现；但聊天、智能体管理和配置版本存在阻断主流程、跨租户影响及配置衔接缺陷。不能认定功能完整或没有 bug。视觉方面，聊天页桌面浅色空态较清晰，但深色主题和手机布局未达标。

证据分为实际构建/测试、隔离复现、静态调用链三类。下列静态问题是根据明确的请求字段、数据库约束、调用路径判断，未冒充生产端到端复现。

**范围与模块判断**

| 提交 | 主题 | 本次判断 |
|---|---|---|
| `86b9964` | 深度思考模型 | 审核/归档两侧均接入开关、落库与展示；AI 适配器测试通过。真实模型兼容性仍待联调 |
| `c61f995`、`47637c5`、`10bb01d`、`6a6b67d` | 发布、激活、编辑配置版本 | 存在历史快照结构不一致、未发布内容进入执行、缓存未失效及唯一约束冲突 |
| `d35c97f` | 文档内容解析 | Java 21 下 16 项测试全部通过；真实 OA 附件与 MinerU 联调未执行 |
| `3ccb459` | 流程总结工作台 | 列表、执行、任务归属检查、历史和个人展示偏好已接入；存在并发查看时结果错配 |
| `3a940e8` | OA 人员反查、个人/标准视角 | 个人执行会覆盖共用流程配置绑定，影响后续标准执行；缓存返回路径遗漏个人视角 |
| `8351e36` | OA 跳转 | 审核/归档工作台已接入；切换租户时前端缓存串用已隔离复现 |
| `a583d94`、`127f888` | 规则卡片与主题样式 | 改动本身较小；未获得登录后的规则页实测证据，不认定视觉验收通过 |
| `e0123be`、`c794812` | 智能体需求、聊天与管理 | 存在多项 P1，前后端契约、权限边界和执行闭环均需补齐 |

**P1：应在发布前解决**

**01. 租户可修改所有租户共用的内置智能体工具绑定。**

位置：[agent_allocation_service.go:289](E:/Work/GJCW/AuraOA/go-service/internal/service/agent_allocation_service.go:289)、[agent_repo.go:78](E:/Work/GJCW/AuraOA/go-service/internal/repository/agent_repo.go:78)。引入：`c794812`。证据：静态调用链与迁移结构。

租户管理员更新平台种子智能体时，`tenant_id=NULL` 通过归属检查；随后 `ReplaceAgentToolBindings` 仅按 `agent_id` 删除原绑定，再写入当前租户的新绑定。所有租户读取同一个种子和同一组 `ToolBindings`，因此租户 A 修改或清空工具会影响租户 B。普通字段更新反而受 `tenant_id = A` 限制而不生效，形成“部分无效、工具全局生效”的错误行为。

复现路径：两个租户都使用 `oa_query`，A 向种子 ID 的更新接口提交 `tool_codes: []`，再检查 B 的有效工具。修复应采用租户副本或明确的租户覆盖表，修改、读取和删除绑定都限定归属。

**02. 清空角色授权反而获得全部租户能力。**

位置：[agent_permission_service.go:83](E:/Work/GJCW/AuraOA/go-service/internal/service/agent_permission_service.go:83)。引入：`c794812`。证据：静态分支。

`grantedAgentCodes` 或 `grantedToolCodes` 为空时，代码复制全部租户配额作为角色授权；没有组织角色的用户也落入同一路径。管理员撤销最后一项授权后权限会扩大，违反两级分配取交集的设计。应将“未迁移默认配置”和“显式空授权”区分，运行时空授权应拒绝能力；默认授权在迁移/角色初始化时落库。

**03. 关闭聊天或撤销智能体后，旧会话仍可调用模型。**

位置：[agent_permission_service.go:138](E:/Work/GJCW/AuraOA/go-service/internal/service/agent_permission_service.go:138)、[agent_runtime_service.go:100](E:/Work/GJCW/AuraOA/go-service/internal/service/agent_runtime_service.go:100)。引入：`c794812`。证据：静态执行链。

发送旧会话时只核验会话所有者，权限函数只返回工具交集，不要求当前智能体属于 `perms.Agents`。关闭 `chat_enabled` 得到空工具集后，仍继续保存消息并调用 AI；停用智能体或移除其配额亦未拒绝旧会话。应在每轮执行前校验聊天总开关及当前智能体的有效授权。

**04. 聊天发送在 HTTP 请求发出前就报错。**

位置：[useChatStream.ts:15](E:/Work/GJCW/AuraOA/frontend/composables/useChatStream.ts:15)、[useChatStream.ts:60](E:/Work/GJCW/AuraOA/frontend/composables/useChatStream.ts:60)、[useAuth.ts:738](E:/Work/GJCW/AuraOA/frontend/composables/useAuth.ts:738)。引入：`c794812`。证据：执行真实 composable 的隔离复现。

聊天解构 `accessToken`、`effectiveTenantCode`，但 `useAuth()` 实际导出 `token`、`effectiveActiveRoleForApi` 等字段。读取 `accessToken.value` 报 `Cannot read properties of undefined (reading 'value')`。隔离运行确认 `fetch_calls=0`。此外相对路径裸 `fetch` 未衔接现有 `apiBase` 与 token 刷新机制；应统一封装认证流式请求。

**05. 智能体标识与历史消息 DTO 不一致，无法正常新建及回看会话。**

位置：[chat.vue:74](E:/Work/GJCW/AuraOA/frontend/pages/chat.vue:74)、[ChatHeader.vue:22](E:/Work/GJCW/AuraOA/frontend/components/Chat/ChatHeader.vue:22)、[ChatThread.vue:53](E:/Work/GJCW/AuraOA/frontend/components/Chat/ChatThread.vue:53)、[chat_dto.go:10](E:/Work/GJCW/AuraOA/go-service/internal/dto/chat_dto.go:10)。引入：`c794812`。证据：前后端契约逐项对照。

后端智能体返回 `agent_code`，前端选择及默认新建读取 `code`，导致发送缺失的必填 `agent_code`。历史消息返回 `role/tool_calls/token_usage`，前端原样赋值后按 `sender_type/tool_executions/token_cost` 渲染；因此重载历史后用户和助手消息均不进入对应模板分支。应统一类型与 wire DTO，避免只修某一处模板。

**06. SSE 事件和载荷不一致，即使认证修好仍没有正常答案。**

位置：[useChatStream.ts:117](E:/Work/GJCW/AuraOA/frontend/composables/useChatStream.ts:117)、[agent_runtime_service.go:240](E:/Work/GJCW/AuraOA/go-service/internal/service/agent_runtime_service.go:240)。引入：`c794812`。证据：按后端实际事件构造流，执行前端原函数。

后端发送 `delta/content`、`reasoning/content`、`tool_start`、`tool_result/payload`、`done/token_usage`；前端监听 `token/delta`、`answer`、`tool_call`，读取 `result/error/token_cost`。注入真实格式的事件后，正文、思考过程仍为空，工具结果未显示。应明确一套 SSE 契约，以 `tool_call_id` 关联工具，补充前后端契约验证。

**07. “触发审核”和“触发总结”并未真正创建任务。**

位置：[executor.go:488](E:/Work/GJCW/AuraOA/go-service/internal/pkg/agenttools/executor.go:488)、[executor.go:515](E:/Work/GJCW/AuraOA/go-service/internal/pkg/agenttools/executor.go:515)。引入：`c794812`。证据：两个执行函数完整调用链。

检查流程可见性后，仅生成随机 UUID 并返回 `submitted`、“已在后台排队处理”，没有调用现有审核/总结 Service，没有写业务日志，也没有入队。用户和模型会被告知不存在的任务已提交。应接入 `AuditExecuteService` / `ProcessSummaryService` 的真实执行与权限逻辑，返回真实任务 ID 和状态。

**08. 租户智能体管理页面加载必然触发权限错误。**

位置：[agents.vue:74](E:/Work/GJCW/AuraOA/frontend/pages/admin/tenant/agents.vue:74)、[router.go:110](E:/Work/GJCW/AuraOA/go-service/internal/router/router.go:110)。引入：`c794812`。证据：路由角色限制及 Promise 调用链。

租户管理员页面在 `Promise.all` 中请求 `/api/admin/agent-catalog`，该路由要求 `system_admin`。这一请求返回 403 后，其他已成功的三个列表也不会赋值。即便允许访问，前端读 `system_tools`，后端实际返回 `tool_catalog`。应提供适合租户配额的只读目录，使用真实字段，并处理局部加载失败。

**09. 智能体、Skill 的保存表单与接口不匹配。**

位置：[agents.vue:121](E:/Work/GJCW/AuraOA/frontend/pages/admin/tenant/agents.vue:121)、[agents.vue:215](E:/Work/GJCW/AuraOA/frontend/pages/admin/tenant/agents.vue:215)、[agent_dto.go:74](E:/Work/GJCW/AuraOA/go-service/internal/dto/agent_dto.go:74)。引入：`c794812`。证据：请求体与校验规则对照。

智能体提交 `code/system_prompt_override/tools`，后端要求 `agent_code/system_prompt/tool_codes`；编辑也始终 POST 集合接口，未调用 PUT `/:id`。Skill 提交 `prompt_template`，后端要求 `content`，会触发必填校验。MCP 的 `is_enabled` 与后端 `enabled` 亦不一致。应分别补齐创建/编辑行为和精确 DTO，而非仅让 TypeScript 本地自洽。

**10. 保存历史配置会破坏快照结构，后续激活失败或丢失配置。**

位置：[rules.vue:333](E:/Work/GJCW/AuraOA/frontend/pages/admin/tenant/rules.vue:333)、[execution_config_source.go:279](E:/Work/GJCW/AuraOA/go-service/internal/service/execution_config_source.go:279)。引入：`6a6b67d`。证据：请求、反序列化和数据库列约束对照。

前端历史快照保存 `ai_strictness/ai_prompt_template/allow_user_*/enabled`；后端发布和恢复使用 `main_table_name/field_mode/kb_mode/ai_config/user_permissions/status`。前端载入历史时也未恢复嵌套 `ai_config`。保存后的快照缺少主表名、AI 配置、权限和状态。激活时这些字段成为空值；审核/归档的非空 JSON 列可能直接导致恢复失败，总结的状态等字段也可能被置空。

尤其 `SaveVersion` 已先写入损坏快照，然后忽略 `applySnapshotToSource` 错误；`ActivateVersion` 也先提交 active 切换，再恢复主表。可能出现提示成功或 active 已切换、实际内容未生效。应使用后端定义的完整、经过校验的快照 DTO，在事务内完成必要写入，传播恢复错误。

**11. 未发布的配置也参与执行，但标记为旧的已激活版本。**

位置：[audit_review_service.go:202](E:/Work/GJCW/AuraOA/go-service/internal/service/audit_review_service.go:202)、[process_summary_service.go:482](E:/Work/GJCW/AuraOA/go-service/internal/service/process_summary_service.go:482)。引入：`c61f995`。证据：版本获取前后的数据来源。

新流程或“按最新配置执行”先从当前主表合并字段、规则、AI 配置，再调用 `GetOrCreateLatestBaseVersion` 取已激活版本 ID/编号，却未使用该版本快照构建实际执行配置。管理员保存修改但未发布时，执行会使用未发布内容并关联旧版本；审核、归档、总结均需检查。应从激活版本构建执行输入，或清晰分离草稿存储与生效存储。

**12. 激活版本未清理审核配置缓存，无法立即生效。**

位置：[execution_config_source.go:235](E:/Work/GJCW/AuraOA/go-service/internal/service/execution_config_source.go:235)、[audit_review_service.go](E:/Work/GJCW/AuraOA/go-service/internal/service/audit_review_service.go) 的 `getProcessConfigCached`。引入：`47637c5`，历史覆盖保存同样受影响。证据：与既有配置 Service 的 invalidator 调用对照。

恢复版本直接调用 Repo 更新主表和规则，没有调用现有 `InvalidationManager`。之前执行过该流程、缓存仍有效时，接下来新流程可继续拿到旧配置，默认 TTL 为 10 分钟。应在成功激活/覆盖当前版本后统一清理对应配置缓存，并检查相关调度刷新。

**13. 个人嵌入审核覆盖标准视角共用的流程配置绑定。**

位置：[embed_audit_service.go:477](E:/Work/GJCW/AuraOA/go-service/internal/service/embed_audit_service.go:477)、[execution_config_version_repo.go:453](E:/Work/GJCW/AuraOA/go-service/internal/repository/execution_config_version_repo.go:453)。引入：`3a940e8`。证据：新个人视角分支与既有绑定唯一键对照。

个人视角强制 `useLatest=true`，最终覆盖 `(tenant_id,module,process_id)` 唯一绑定；该绑定没有用户或视角维度。之后标准嵌入自动审核、其他用户默认重审会复用该人的个性化执行快照。这使新增双视角无法与旧标准审核稳定共存。应隔离个人/标准及必要的用户维度，保留旧标准绑定兼容。

**P2：明确缺陷与完整性问题**

**14. 发布历史相同内容触发唯一约束冲突。**

位置：[execution_config_version_repo.go:255](E:/Work/GJCW/AuraOA/go-service/internal/repository/execution_config_version_repo.go:255)、[000061 迁移](E:/Work/GJCW/AuraOA/db/migrations/000061_execution_config_source_fingerprint.up.sql)。引入：`c61f995`。证据：SQL 唯一约束与发布算法。

依次发布内容 A、B，再把当前配置改回 A 并发布：仅检查最新 B 的指纹，随后尝试创建 A 的新版本；但数据库对同一配置的 fingerprint 有唯一约束，必然失败。应恢复按完整历史指纹复用/激活，或通过迁移明确改变版本语义。历史编辑成另一版本相同内容也需覆盖。

**15. SSE 状态更新绕过 Vue 响应式对象，且事件名跨网络分块丢失。**

位置：[useChatStream.ts:39](E:/Work/GJCW/AuraOA/frontend/composables/useChatStream.ts:39)、[useChatStream.ts:85](E:/Work/GJCW/AuraOA/frontend/composables/useChatStream.ts:85)。引入：`c794812`。证据：Vue 深监听隔离复现及循环作用域。

消息 push 进响应式数组后，回调继续修改原始普通对象 `assistantMsg`。隔离验证中，HTTP 返回之后的这些修改触发响应式更新次数为 0；修正事件名后仍可能只在其他状态变化时显示。另 `currentEvent` 在每次 reader.read 循环内重置，`event:` 与 `data:` 分属两个网络块时事件名丢失。应通过响应式消息代理更新，以完整 SSE 帧解析并保留跨块状态。

**16. OA 跳转配置跨租户缓存串用。**

位置：[useOAJump.ts:14](E:/Work/GJCW/AuraOA/frontend/composables/useOAJump.ts:14)。引入：`8351e36`。证据：执行真实 composable 的隔离复现。

模块级 `configState`、`fetchPromise` 不带租户键，也未监听角色切换。A 加载配置后切到 B，再打开审核/归档，仍复用 A 的地址。测试输出当前租户 B、生成 URL `https://A.example.test/123`、配置请求次数 1。应按有效租户缓存，并在登出/切租户时清理且隔离在途请求。

**17. 总结完成回调会覆盖另一个流程的详情。**

位置：[summary.vue:139](E:/Work/GJCW/AuraOA/frontend/pages/summary.vue:139)。引入：`3ccb459`。证据：异步回调与页面共享状态。

启动 A 的总结后关闭抽屉，打开 B 的结果；A 的进度和完成回调无条件写 `currentResult`，导致 B 标题下显示 A 的总结。`resumeJob` 已检查 `selected.process_id`，`runSummary` 却遗漏。应在进度和完成分支都核对当前选中流程，或按流程独立保存结果。

**18. 标准结果缓存返回时遗漏个人视角。**

位置：[embed_audit_service.go:205](E:/Work/GJCW/AuraOA/go-service/internal/service/embed_audit_service.go:205)、[embed_audit_service.go:257](E:/Work/GJCW/AuraOA/go-service/internal/service/embed_audit_service.go:257)。引入：`3a940e8`。证据：提前返回分支。

已有标准结果且 `prefer_cached=true` 时提前返回，个人用户识别和 `personal_view/default_perspective` 组装尚未执行；标准任务运行中的提前返回也存在此问题。于是缓存命中和实时查询返回的双视角能力不一致。应在公共返回阶段组装个人视角，或在分支前完成相关读取。

**19. 深色主题与窄屏聊天布局不合格。**

位置：[chat.vue:175](E:/Work/GJCW/AuraOA/frontend/pages/chat.vue:175)、[ChatSidebar.vue:133](E:/Work/GJCW/AuraOA/frontend/components/Chat/ChatSidebar.vue:133)，以及 Chat 子组件样式。引入：`c794812`。证据：本轮真实生产构建截图。

桌面深色模式下全局导航已变暗，但聊天画布、顶部、会话栏和输入框仍大面积浅色；搜索框受到全局主题样式影响而变暗，造成同一区域主题混用。390×844 下聊天侧栏维持 260px，又没有折叠/抽屉响应式处理，正文和输入区被截出屏幕。应沿用已有 CSS 变量，并在手机宽度切换会话列表呈现方式。

**视觉检查步骤与证据边界**

1. 打开实际生产构建的 `/chat` 空态：能渲染，标题、快捷问题、输入区层次清楚；当前没有认证业务数据。普通审核/总结页需要登录，本次未将聊天空态当成全部页面验收。
2. 桌面 1440×900 切换深色主题：不通过，聊天主体未随全局切换。截图已保存并重新读取核对。
3. 手机 390×844 检查布局：不通过，正文与输入区裁切。截图在布局稳定后保存并重新核对。检查后恢复浅色主题和默认视口。

![初始聊天空态](E:/Work/GJCW/AuraOA/docs/reviews/2026-09-05-assets/01-chat-empty.png)

![桌面深色模式：全局导航与聊天主体主题不一致](E:/Work/GJCW/AuraOA/docs/reviews/2026-09-05-assets/02-chat-dark-desktop.png)

![手机宽度：会话栏挤占正文和输入区](E:/Work/GJCW/AuraOA/docs/reviews/2026-09-05-assets/03-chat-mobile.png)

没有从截图推断完整无障碍合规；键盘主流程、读屏、真实长消息、工具结果卡片、登录后的总结/规则页仍需补验。

**尚未完成的产品衔接**

以下有明确的代码/文档缺口，但没有与以上主问题重复计算为编号缺陷：

- `CleanExpiredSessions` 只有定义，没有定时调度调用；`chat_retention_days` 尚未形成自动清理能力。
- 聊天保存了备用模型字段，但运行时只调用主模型 `Chat`，没有接入 `ChatWithFallback`；同时强制 `EnableThinking=true`，未沿用审核/归档的模型能力与开关判断。
- `MaxMCPServers` 可保存但 `SaveMCPServer` 不校验数量；工具绑定统一写 `ToolType=system`，运行时按 `skill` 加载的分支缺少正常管理链路。
- 已有 `TodoListCard.vue` 未被 `ChatThread` 分派使用，所有工具均走 `GenericToolCard`；工具独立可视化没有完成。
- 聊天会话列表固定取第一页 50 条，没有加载后续页的交互；长会话历史也无分页/上下文窗口控制。
- 系统租户页没有智能体配额配置界面，组织角色页只补了页面权限选项，没有智能体/工具授权编辑控件；目前两级分配主要停留在后端接口。
- 新聊天页和智能体管理页未声明 `middleware: 'auth'`，与既有页面不同。未登录可以打开空壳页面；后端 API 仍有 JWT/角色鉴权，不能据此声称匿名读取了业务数据。
- LLM 日志仍从 `SystemPrompt/UserPrompt` 落请求正文，聊天仅设置 `Messages`，聊天请求/工具上下文未完整进入统一 payload。Token 统计已接入 `chat`，但日志追溯仍有缺口。
- `ChatThread` 对模型内容直接 `marked.parse` 后 `v-html`，没有 HTML 清洗。既有 `AiMarkdownStream` 也有同类路径；应在输出链路统一处理不可信 HTML，尤其接入 OA/MCP 内容后。该项未进行攻击载荷的浏览器实测。
- `docs/agents/README.md` 和 API 索引仍标记代码尚未落地/拟定；实际接口与前端类型、文档需要一起收敛。新增聊天代码还存在硬编码错误/新会话文案，未完全遵循中英文 i18n 规范。

**已执行验证**

| 检查 | 结果 | 可证明的范围 |
|---|---|---|
| `frontend: npm run build` | 通过 | Nuxt 生产构建；不等于前后端契约一致，也未单独完成 Vue 类型检查 |
| `go-service: GOOS=linux GOARCH=amd64 go build ./...` | 通过 | Linux amd64 目标可编译；未运行 Linux 二进制 |
| Windows `go test ./...` | 未整体通过 | `system_monitor_service.go:268–269` 使用 Linux 专属 `syscall.Statfs_t/Statfs`，阻挡 service/handler/router；这是本轮之前的代码 |
| Windows 可执行测试包 | 通过 | middleware、repository、pkg/ai、pkg/oa、pkg/validate；服务层测试未因这些通过而视为通过 |
| Java 21 `document-parser: mvn -B test` | 16 项通过，0 失败/错误/跳过 | 文档解析 API、OFD 和临时目录测试；首次默认 JDK 太旧，改用本机已有 Java 21 后成功 |
| 聊天认证隔离复现 | 确认失败 | 原 composable 读取不存在的导出字段，发出请求次数为 0 |
| 后端格式 SSE 隔离复现 | 确认失败 | 仅补认证桩后，正文/思考仍为空，工具 payload 丢失；响应式更新次数为 0 |
| OA 切租户隔离复现 | 确认失败 | B 租户使用 A 的 OA URL，未重新请求配置 |
| 真实前端截图 | 完成局部检查 | 聊天空态、深色桌面与手机布局；未模拟成已登录完整业务流程 |

未连接真实租户数据库、Redis、OA 或付费模型执行任务；Docker 服务当前未运行。未执行真实数据库升级/回滚与跨租户 API 集成测试。数据库相关结论基于迁移约束与调用链，后续修复应在隔离 PostgreSQL 上验证。

**建议的修复和验收顺序**

1. 先修跨租户种子绑定、空授权放行、关闭聊天后旧会话继续执行，以及个人视角覆盖标准绑定。
2. 修历史快照契约、发布/生效边界、激活事务与缓存；覆盖“保存未发布、A→B→A、覆盖历史、切回旧版本、缓存已预热”场景。
3. 统一聊天/管理端 DTO、认证与 SSE，接入真实审核/总结任务；完成“新建→发送→工具→完成→刷新历史→撤销权限”的完整路径。
4. 修 OA 租户缓存和总结详情竞态，再补角色/配额管理、清理、降级、工具卡片与分页。
5. 最后在真实登录环境验收浅色/深色/暖色、桌面/手机、中英文及长消息，补上 PostgreSQL/Redis/OA/模型端到端回归。
