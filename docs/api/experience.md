# 用户体验优化

租户管理员菜单：`/admin/tenant/experience`，包含「审核」「智能体」两个页签。
新建租户默认授予管理员；迁移为已有组织管理角色补齐菜单，可在组织角色页面配置。

接口前缀 `/api/tenant/experience`，认证链为 JWT → TenantContext → RequireRole(tenant_admin) → RequirePagePermission(/admin/tenant/experience)。
所有查询限定当前租户，不接受客户端传入的 tenant_id。

| 方法 | 路由 | 说明 |
|---|---|---|
| GET | /api/tenant/experience/audit | 有评论的审核批次 |
| GET | /api/tenant/experience/audit/:id | 原审核结果和分页评论 |
| GET | /api/tenant/experience/agents | 被点赞或点踩的助手消息 |

列表查询参数：`keyword`（标题、评论或用户；审核还匹配 process_id，智能体还匹配回复内容）、`feedback`（空 / like / dislike / comments）、`page`（默认 1）、`page_size`（默认 20，1–100）。
筛选先于分页，按最近互动时间及 ID 倒序。无效页码回落 1，无效 page_size 回落 20。
统一成功响应：`{code:0,message:"success",data:{items:[],total:0,page:1,page_size:20}}`。

审核列表字段：`id`（audit_log_id）、`process_id`、`title`、`process_type`、`like_count`、`dislike_count`、`comment_count`、`updated_at`（最近互动时间）。同一流程的不同审核批次独立展示。评论及满意度按条保存，同一人可多次评论；like_count/dislike_count 表示满意/不满意的评论数，不表示人数。
审核详情 data：`{audit_result, interactions}`。`audit_result` 为嵌入审核结果结构；`interactions` 见 [嵌入接口](./embed.md)。详情的 page/page_size 只用于评论分页。

智能体列表字段：`id`（message_id）、`session_id`、`title`、`agent_name`、`username`（优先显示姓名）、`content`、`feedback`、`feedback_comment`、`updated_at`（评价时间）。
完整聊天记录复用 `GET /api/tenant/agent-sessions/:id/messages`，抽屉高亮对应被评价的消息。评价接口仍为 `POST /api/chat/messages/:id/feedback`，仅允许本人会话的助手消息。

本期仅嵌入审核开放评论及可选满意度，归档、总结、审核工作台暂不增加入口。评论为纯文本，不触发 AI 调用，无新增 Token 消耗。
标准嵌入审核的评论对有嵌入访问权的人可见；个人审核继承原任务的本人权限；管理员可查看本租户反馈。
数据生命周期沿用原记录：删除审核日志级联删除对应评论及可选满意度；聊天删除或保留期清理后，相关反馈及上下文同步删除。


评论新增 `updated_at` 与 `can_manage` 字段。嵌入列表仅对当前已识别 OA 作者返回 `can_manage=true`；该字段只控制界面入口，后端写入仍独立校验租户、审核批次和 OA 作者。

- `POST /api/embed/audits/:id/comments/:comment_id/update`：完整提交 `{content, feedback}`，正文 1–2000 字；feedback 可为 like/dislike/null。只更新本人评论，保留创建时间，更新 updated_at。
- `POST /api/embed/audits/:id/comments/:comment_id/delete`：删除本人评论，返回 `{updated:true}`。不存在或无权操作均返回权限错误，不能修改或删除他人、其他审核批次或其他租户的评论。
- 编辑或删除后满意度计数及后台列表随实际记录重新统计；删除最后一条评论后，该审核不再出现在有互动列表。

数据库迁移文件为 `000078_experience_feedback.up.sql` / `.down.sql`，接续当前 77 号迁移。
