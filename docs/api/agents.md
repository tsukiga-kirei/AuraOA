# 智能体、工具与两级分配接口

普通成功响应统一为 `{code:0,message:"success",data:...}`。配置目录规模较小，列表一次性返回数组，无服务端分页。

## 系统管理员

| 方法 | 路由 | 说明 |
|---|---|---|
| GET | /api/admin/agent-catalog | tool_catalog、agent_catalog、skill_catalog |
| GET / PUT | /api/admin/tenants/:id/chat-allocation | 查看/更新租户配额 |

配额字段：chat_enabled、chat_retention_days（1–3650）、chat_primary_model_id、chat_fallback_model_id、allow_custom_skills、allow_tenant_mcp、max_mcp_servers（0–100）。
系统管理员只控制总开关与 MCP/Skill 是否开放；智能体、系统工具的具体装配在租户侧完成。模型 ID 未传则不变，显式 null 恢复继承。

另：`GET /api/tenant/agent-stats` 返回智能体会话、消息、工具/MCP/Skill 调用与 Token 汇总，供数据管理页使用。

## 租户管理员

| 方法 | 路由 | 说明 |
|---|---|---|
| GET | /api/tenant/agent-catalog | 配额内可装配工具和技能目录 |
| GET / POST | /api/tenant/agents | 查询 / 创建多个智能体 |
| PUT / DELETE | /api/tenant/agents/:id | 更新 / 删除租户智能体 |
| GET / POST | /api/tenant/mcp-servers | 查询 / 注册 MCP 服务 |
| PUT / DELETE | /api/tenant/mcp-servers/:id | 更新 / 删除本租户 MCP |
| POST | /api/tenant/mcp-servers/:id/test | 初始化连接并发现全部分页工具，返回 tools |
| GET / POST | /api/tenant/skills | 查询 / 创建 Skill |
| PUT / DELETE | /api/tenant/skills/:id | 更新 / 删除本租户 Skill |
| GET | /api/tenant/agent-sessions | 查询租户下所有用户的智能体会话列表（分页）；支持 keyword、agent_code、username、start_date、end_date 筛选 |
| GET | /api/tenant/agent-sessions/:id/messages | 查询指定会话的完整消息记录（供抽屉审计回放） |

智能体创建字段：agent_code、name、description、system_prompt、enabled、tool_codes、quick_questions（快捷提问配置数组：`[{icon, title, prompt, description}]`）、access_control（访问控制对象：`{allow_all: boolean, allowed_roles: string[], allowed_members: string[], allowed_departments: string[]}`，默认 `allow_all: true` 全员可用）。
更新支持 name、description、system_prompt、enabled、tool_codes、quick_questions、access_control，标识码不可变。
返回字段额外包含 `skill_codes`、`mcp_codes`（从 `tool_codes` 解析提取，方便前端快速装配与卡片展示）、`access_control`。
修改平台智能体时创建租户专属覆盖，平台模板和其他租户保持独立；历史会话按 agent_code 解析本租户当前定义。定义和绑定同事务保存。

MCP 字段：server_code、name、description、transport_type（http）、endpoint_url、headers（JSON 字符串对象，加密存储）、enabled、agent_codes。返回 cached_tools、last_synced_at、agent_codes，不返回密钥。`agent_codes` 表示挂载到哪些智能体；测试连接发现工具后会按原挂载智能体重写绑定。编辑时 headers 留空保留，传 "{}" 清空。地址或请求头改变后清空工具缓存，需要再次测试连接。创建检查租户配额并持行锁防止并发超额；标识码创建后不可变。

Skill 字段：skill_code、name、description、content（最多 64 KB）、enabled、agent_codes；返回 is_system、agent_codes。内置 Skill 只读。当前 Skill 是可读取的指令指南，不执行本地脚本或安装软件。

## 装配和运行权限

既有能力已封装为九个可独立绑定的系统工具：list_my_todos、get_process、get_approval_flow、get_latest_audit、get_latest_summary、draft_comment、run_audit、run_summary、resolve_oa_url。
其中 run_audit 和 run_summary 经过 OA 可见性检查并提交既有工作台任务，返回真实任务信息；不自动批准 OA 流程。

绑定标识统一放在 tool_codes：
- 系统工具：例如 get_process。
- Skill：skill:expense_review，模型调用后读取完整指南。
- MCP：mcp:server_code:tool_name，测试发现后可装配。

有效工具 = 租户 MCP/Skill 开关 ∩ 智能体绑定 ∩ 启用状态。
有效智能体 = 组织角色授权 ∩ 智能体人员访问控制 (`access_control`) ∩ 启用状态。组织角色只再分配智能体（agent_codes）；不选智能体表示可用全部已启用智能体。用户同时需要 /chat 页面权限。
租户管理员默认拥有 `/admin/tenant/agents`。Token 配额小于 0 表示不限制。

## MCP 实现范围

支持 Streamable HTTP：initialize、initialized 通知、会话 ID、JSON/SSE 响应、tools/list 分页和 tools/call；操作结束释放会话。协议支持 2025-03-26 / 2025-06-18。不支持 stdio 本地进程和旧式独立 SSE 端点。
参考 [官方传输规范](https://modelcontextprotocol.io/specification/2025-06-18/basic/transports)、[生命周期](https://modelcontextprotocol.io/specification/2025-06-18/basic/lifecycle)。
