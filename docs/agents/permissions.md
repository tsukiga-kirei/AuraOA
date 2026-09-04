# 权限模型

对话授权是多层求交，**缺一不可**。分配操作见 [allocation.md](./allocation.md)。

## 1. 层次

```
系统管理员租户配额
    ∩ 租户 chat_enabled
    ∩ 智能体已启用且绑定
    ∩ 组织角色 page_permissions 含 /chat
    ∩ 组织角色智能体授权
    ∩ 组织角色工具授权
    ∩ OA 流程可见性（读实例时）
    ∩ 当前用户身份（待办只查自己）
```

伪代码：

```text
effective_agents = tenant_alloc.agents ∩ role_grants.agents ∩ enabled_agents
effective_tools  = tenant_alloc.tools ∩ agent.bindings ∩ role_grants.tools ∩ enabled
```

MCP 工具键、Skill 脚本工具键与系统 `tool_code` 走同一 `effective_tools`。

## 2. 页面权限

- `/chat`：业务用户进入独立对话页；加入 `org.vue` 的 `ALL_PAGES_CONFIG`、`auth_service` 的 `pathLabels`、侧栏新区。
- `/admin/tenant/agents`：租户管理员智能体管理（租户角色 `page_permissions`）。
- 系统管理员使用 `/admin/system/tenants` 上的配额 Tab，不使用 `/chat`。

未授予 `/chat` 的用户，即使误调 API 也返回无权限。

## 3. 数据与流程

| 对象 | 规则 |
|------|------|
| 会话 | `tenant_id` + `user_id`；禁止读他人会话 |
| 待办列表 | 适配器按当前 `username` |
| 单流程 | `CheckProcessVisibility` |
| 审核/总结触发 | 有效工具含 `run_*` 且流程可见 |
| LLM 日志 | 租户数据管理可见元数据；payload 脱敏同现网 |

嵌入：embed token 只证明租户，不证明可读任意流程。

## 4. 角色叠加

用户多个组织角色时，智能体授权、工具授权、页面权限均取**并集**，再与租户配额求交。

租户管理员若要自己试用对话：必须具备业务角色及 `/chat`（与现网「租户管理员切业务身份」一致），不能用 `tenant_admin` 身份直打业务编排接口。

## 5. 缩小配额的生效

以服务端每轮计算为准，不依赖前端缓存的工具列表。进行中的 SSE 可结束本轮；下一轮按新有效集。
