# 对话接口

前缀 `/api/chat`，JWT → TenantContext → business 角色。所有会话查询限制为当前租户、当前用户。创建及每次发送重新检查聊天开关、页面权限、智能体启用状态及角色授权。停用智能体后可查看本人历史，但不能继续调用。

## 会话与智能体

| 方法 | 路由 | 说明 |
|---|---|---|
| GET | /api/chat/agents | 有效智能体数组，字段 id、agent_code、name、description、is_system、tool_codes |
| GET | /api/chat/sessions | 本人的会话列表；keyword 匹配标题或消息正文；page 默认 1、page_size 默认 20，范围 1–100 |
| POST | /api/chat/sessions | 创建：agent_code 必填；title、process_id、source 可选 |
| GET | /api/chat/sessions/:id | 返回 session 与 messages |
| PATCH | /api/chat/sessions/:id | 更新 title、pinned，未传字段保持原值 |
| DELETE | /api/chat/sessions/:id | 删除本人会话及消息 |
| POST | /api/chat/sessions/:id/messages/stream | 请求 {"content":"问题"}；返回 SSE |

普通响应为 `{code:0,message:"success",data:...}`。列表 data 为 `{items:[],total:0,page:1,page_size:20}`。
会话字段：id、agent_id、agent_code、agent_name、title、source、process_id、pinned、created_at、updated_at。
消息字段：id、session_id、role、content、reasoning_content、status、tool_calls、token_usage、created_at。字段为 snake_case。

## 流式事件

前端使用带 Bearer Token 的 fetch，支持一次刷新过期 Token；已开始的消息流不自动重发。

| event | data | 含义 |
|---|---|---|
| status | status、step | 编排进度 |
| delta | content | 追加回答 |
| reasoning | content | 追加模型思考 |
| reset | content、reasoning_content | 重试或降级时恢复已完成轮次，清除本次失败的半截输出 |
| tool_start | tool_call_id、tool_code、ui_kind、status、arguments | 工具开始 |
| tool_result | 上述字段及 payload | 工具结果，以 tool_call_id 更新原记录 |
| session | session_id、title | 首轮自动标题 |
| done | status、token_usage | 完成 |
| error | message | 失败，包括权限撤销 |
| interrupted | message | 中断 |

tool status 为 running / success / error。客户端停止通过 AbortController 取消连接。
SSE 解析支持 UTF-8 分片、CRLF、跨网络包事件名及多行 data。消息和工具记录为响应式对象。

## 界面与限制

全局侧栏在「AI 助理」分区按智能体归组会话，仅智能体旁提供新建；搜索改为分类弹出框（智能体 / 工具 / 对话），对话检索覆盖标题与消息正文。已撤销智能体的历史保留在原分组。
回答为安全过滤后的 GFM Markdown 文档；系统工具、MCP、Skill 默认采用紧凑执行行，详细输出按需展开。
运行时最多 8 轮，末轮不提供工具。所有 AI 调用走统一配额、日志、重试入口；聊天专用主/备用模型为空时继承租户模型。聊天保留期由系统管理员配置，后台每小时清理到期会话。
当前实现独立对话页；专用嵌入对话路由尚未提供。
