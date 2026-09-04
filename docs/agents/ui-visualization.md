# 工具可视化

每种**系统工具**对应独立前端卡片，禁止全部挤进同一个 JSON 折叠块。MCP / Skill 脚本第一期用通用卡，允许后续按 `ui_kind` 加专用组件。

过程展示对齐投资助手的 activity：思考、工具进行中、工具结果。个人设置可关闭「过程」，但 **结果卡仍显示**（待办列表、意见草稿、OA 链接）。

## 1. SSE 与渲染

`tool_start` / `tool_result` 必须带：

```json
{
  "tool_code": "list_my_todos",
  "ui_kind": "todo_list",
  "status": "running | success | error",
  "payload": {}
}
```

前端：`toolRenderers[ui_kind] || GenericMcpCard`。未知 kind 回退通用卡，不白屏。

文案走 i18n，卡片内业务字段（流程标题、节点）来自 OA/工具 payload，不是前端写死。

## 2. ui_kind 对照

| ui_kind | 工具 | 卡片内容 |
|---------|------|----------|
| `todo_list` | `list_my_todos` | 分页列表：标题、申请人、节点、时间；行内「去 OA」「查看详情」 |
| `process_detail` | `get_process` | 主表关键字段、明细表摘要、附件名；展开勿一次渲染超大 JSON |
| `approval_flow` | `get_approval_flow` | 时间线：节点、操作人、动作、意见 |
| `audit_result` | `get_latest_audit` | 建议、评分、时间；链到工作台 |
| `summary_result` | `get_latest_summary` | 块标题 + 短摘要 |
| `audit_job` | `run_audit` | 已提交、job 状态、打开工作台/嵌入进度 |
| `summary_job` | `run_summary` | 同上 |
| `opinion_draft` | `draft_comment` | 意见全文、复制、去 OA |
| `oa_link` | `resolve_oa_url` | 主按钮打开新窗口（个人设置可改） |
| `mcp_generic` | MCP | 标题、摘要字段、原始 JSON 折叠 |
| `skill_script` | Skill 脚本 | 退出码/文本输出折叠 |
| `reasoning` | 模型思考 | 可折叠，默认收起或进行中展开 |

## 3. payload 约定（系统工具）

字段 `snake_case`，与后端 DTO 一致。

`todo_list`：`items[]`（`process_id`、`title`、`applicant`、`department`、`current_node`、`submit_time`、`oa_url` 可选）、`total`、`page`、`page_size`。

`process_detail`：`process_id`、`title`、`main_fields[]`（`label`、`value`）、`detail_tables[]`、`attachments[]`（文件名，无二进制）。

`opinion_draft`：`process_id`、`intent`、`comment`、`oa_url`。

`audit_job`：`job_id`、`process_id`、`status`。

失败时 `status=error`，`payload.message` 为可展示错误（已 i18n 或键）。

## 4. 实现提示

- 不要复用工作台 `AiMarkdownStream` 的固定 max-height 卡片作为主画布；对话区单独组件，Markdown 用 GFM。
- 工具进行中显示骨架或 activity 行，完成后替换为对应卡片。
- 列表超过一页时卡片内提示「可让我继续查下一页」，由模型再调 `list_my_todos`。
