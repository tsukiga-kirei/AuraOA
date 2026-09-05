# 智能体运行时架构

实现时的编排与数据流。分配见 [allocation.md](./allocation.md)，权限收敛见 [permissions.md](./permissions.md)。

## 1. 与现有 AI 层的关系

- 所有消耗 Token 的调用必须经 `AIModelCallerService.Chat` / `ChatWithFallback`。
- `RequestType` 使用 `chat`；`CallType`：对话主轮可用 `reasoning`，工具结果后的总结轮也可 `reasoning`。不要把工具 HTTP 算进 LLM 日志。
- 现有审核/归档/总结仍用 `system_prompt` + `user_prompt` 两段。调用层扩展可选 `Messages`、`Tools` 后，旧路径必须保持兼容。
- 本地模型若不支持 OpenAI `tool_calls`，使用同一套服务端工具 + JSON 工具协议兜底（见实现）。

对话主链路 **POST SSE**（对齐投资助手的 `fetch` 读 event-stream），不要先做审核那种 Job 再 GET EventSource。`run_audit` / `run_summary` 长任务仍走现有 Job。

## 2. 会话

- `chat_sessions`：`tenant_id`、`user_id`、`agent_id`、`title`、`source`（`standalone` | `embed`）、可选 `process_id`、时间（`apptime`）。
- `chat_messages`：`role`（user / assistant / tool）、正文、工具调用摘要、`llm_log_id`。
- 路由：`/chat/new`、`/chat/:session_id`；嵌入绑定当前流程时写入 `process_id` 并注入系统侧上下文。
- 标题：首轮结束后可由模型或规则生成短标题，允许用户重命名。
- 保留：租户 `chat_retention_days` + Cron 硬删过期会话（级联消息）。

## 3. 一轮对话编排

```
用户发消息（选定 agent_id）
  → 校验 chat_enabled、页面权限、智能体授权
  → 计算 effective_tools（配额 ∩ 智能体绑定 ∩ 角色工具）
  → 加载 Skills 正文（仅已绑定且在配额内）
  → 组装 messages + tools 调模型
  → 若 tool_calls：鉴权 → 执行（系统工具 / MCP / Skill 脚本）→ 推 tool_start/tool_result
  → 回填 tool 结果再调模型，直到结束或达到步数上限（建议 6～8）
  → 推 delta Markdown、done
```

模型不得编造流程字段；无工具结果时只能说明缺少数据或引导用户授权。

## 4. SSE 事件（拟定）

| `event` | 含义 |
|---------|------|
| `session` | `session_id`、标题 |
| `agent` | 当前 `agent_code` |
| `status` | 文案状态：正在查询待办 等 |
| `tool_start` | `tool_code`、`ui_kind`、调用参数摘要（脱敏） |
| `tool_result` | 结构化 payload，供卡片渲染 |
| `reasoning` | 思考增量（可选） |
| `delta` | 助手 Markdown 增量 |
| `card` | 与 tool_result 可合并；保留以便审核 Job 进度 |
| `done` | 正常结束 |
| `error` | 失败 |
| `interrupted` | 客户端取消或断开，已保存草稿 |

## 5. 身份两条链

| 入口 | 鉴权 | 用户 |
|------|------|------|
| `/api/chat/*` | JWT + TenantContext | `claims.Username` |
| `/api/embed/chat/*` | EmbedAccess + OA 用户映射 | `ResolveUsernameByOAUserID` |

编排服务共用，禁止嵌入链路用 JWT 用户顶替 OA 当前操作者。

## 6. 前端结构

- 全局侧栏：仪表盘 → 工作台 → **AI 对话**（仅 `/chat`）。
- 对话页：左历史（今天 / 近 7 天 / 更早）+ 画布 + 底栏（智能体选择 + 输入）。
- 参考交互：仓库外 `global-investment-copilot` 的历史列表与流式 Markdown；技术栈仍为 Nuxt 3 + Ant Design Vue。

## 当前实现补充（2026-09-05）

以 [chat API](../api/chat.md) 和 [agents API](../api/agents.md) 为实际契约；上文未实现的设计不代表现有能力。当前路径为 `/chat?session=<id>`，会话放入全局侧栏的智能体目录中。消息表保存 assistant 的 tool_calls 执行摘要；模型上下文使用 Messages/Tools，要求模型支持工具调用，尚未提供自定义 JSON 兜底。Skill 通过可调用指南读取工具装配。MCP 目前为 Streamable HTTP。保留期由每小时后台清理执行；暂无专用嵌入聊天路由。
