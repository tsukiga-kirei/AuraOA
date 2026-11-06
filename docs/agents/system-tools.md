# 系统工具目录

系统工具由代码注册，`tool_code` 跨版本稳定。租户能否使用由 **系统管理员配额** 决定，用户能否调用由 **智能体使用范围与工具绑定** 决定。执行必须走 [OA 适配器](./oa-adapter.md) 或现有 AuraOA Service，禁止直连 OA 库。

每个工具必须同时登记：JSON Schema、适配器/服务依赖、`ui_kind`、权限键（默认等于 `tool_code`）、是否 `oa_required`。

前端卡片见 [ui-visualization.md](./ui-visualization.md)。

## 1. 一期工具清单

### 查询（默认绑定 `oa_query`、`oa_assist`）

| tool_code | 作用 | 适配器 / 服务 | ui_kind | oa_required |
|-----------|------|---------------|---------|-------------|
| `list_my_todos` | 当前用户待办分页 | `FetchTodoListPaged` | `todo_list` | 是 |
| `list_my_requests` | 当前用户发起的流程列表分页 | `FetchMyRequestsPaged` | `my_request_list` | 是 |
| `get_process` | 流程主表/明细/附件摘要 | `FetchProcessData` + 可见性 | `process_detail` | 是 |
| `get_approval_flow` | 审批轨迹 | `FetchProcessFlow` + 可见性 | `approval_flow` | 是 |
| `get_latest_audit` | AuraOA 最近一次审核结论 | `audit_logs` + 可见性 | `audit_result` | 是 |
| `get_latest_summary` | AuraOA 最近一次总结 | `process_summary_*` + 可见性 | `summary_result` | 是 |

`list_my_todos` 参数应对齐工作台：`keyword`、`applicant`、`department`、`process_types`、日期范围、`page` / `page_size`。未传条数时使用该智能体配置的默认值（默认 20），并截断到配置的单页上限（默认 50）。`list_my_requests` 同样读取 `page_size` / `max_page_size`。`get_latest_audit` 与 `get_latest_summary` 的批量上限读取 `max_batch`（默认 50）。`execute_safe_query` 读取 `max_rows`（默认 100，0 为不限制）和 `max_queries`（默认 5）；多条 SQL 放在一次调用的 `queries` 数组里。这些数值保存在 `agent_tool_bindings.config`，在智能体抽屉里按工具配置。单轮工具步数保存在 `agent_definitions.max_loop_steps`（默认 8）。

`get_process` / `get_approval_flow` 在 `CheckProcessVisibility` 为假时返回无权，不返回表单。

### 辅助办（默认绑定 `oa_assist`）

| tool_code | 作用 | 依赖 | ui_kind | oa_required |
|-----------|------|------|---------|-------------|
| `draft_comment` | 按意图起草批准/退回意见（纯文本） | 可见流程 + 可选审核结论 | `opinion_draft` | 是 |
| `run_audit` | 触发现有审核 Job | `AuditExecuteService` | `audit_job` | 是 |
| `run_summary` | 触发现有总结 Job | 总结 execute | `summary_job` | 是 |
| `resolve_oa_url` | 生成 OA 打开链接 | `oa.BuildProcessURL`（与工作台同一兜底） | `oa_link` | 否（需连接上的 Web URL） |

`run_*` 立即返回 `job_id` 与状态，由前端再订现有 stream；对话 HTTP 不阻塞到审核结束。

### 填单（默认绑定 `oa_creator`）

| tool_code | 作用 | 依赖 | ui_kind | oa_required |
|-----------|------|------|---------|-------------|
| `list_creatable_workflows` | 查询当前用户可创建的流程 | 官方 `getCreateWorkflowList`，人员 ID 取自 `hrmresource` | `creatable_workflow_list` | 是 |
| `get_workflow_schema` | 有创建权限后读取空白表单字段与下拉选项 | 可创建列表通过后，再 `FetchFieldsByWorkflowID` | `workflow_schema` | 是 |
| `create_oa_draft` | 创建流程草稿 | 官方 `doCreateRequest`；`userid` 使用当前用户的 `hrmresource.id` | `create_workflow_draft` | 是 |

`GetAllToolSpecs` 必须覆盖 `BuiltinTools` 的每一个键。能力装配页和对话运行时只读取这份列表，未登记的工具不会出现，模型也调用不到。

## 2. 明确不进一期目录

| 候选 | 原因 |
|------|------|
| `list_archived` | 第一期未纳入对话工具范围；后续开放时须复用当前用户可见性边界 |
| `approve_process` / `reject_process` | 适配器无写接口，产品本期禁止 |
| `query_model_table` | 建模查询需严格白名单，避免对话变成任意 SQL |
| `search_org` | 无独立 Org 适配器 API |

## 3. 注册表示例（实现约定）

```text
ToolSpec {
  code            string   // list_my_todos
  display_key     string   // i18n
  description     string   // 给模型看的英文/中文说明
  ui_kind         string
  oa_required     bool
  adapter_need    []string // 方法名或可选接口名
  risk            string   // read | assist  （write 预留且一期不用）
}
```

`risk=assist` 的工具默认不要分给「仅查询」配额模板。

## 4. 错误

统一业务错误（前端可展示 i18n 键）：

- 未配置 OA 连接
- 当前 OA 类型未实现适配器
- 可选能力不支持
- 流程不可见
- 工具未在有效集中（配额或角色）
- 参数非法（含超大 page_size）

## 个人流程边界

读取最新审核/总结与触发审核/总结均强制通过 OAAdapter.CheckProcessVisibility 校验本人待办、发起或历史审批参与关系，校验失败即拒绝。所有角色规则一致。
结果优先本人个人记录，无个人结果时允许使用该流程通用嵌入结果；不能将其他人的系统内个人记录作为通用兜底。嵌入自动任务与 Agent 共用适配器，但 Agent 必须传当前用户身份并先校验。
