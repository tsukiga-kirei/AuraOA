# 系统管理员分配与租户管理员再分配

AuraOA 现有模式：系统管理员把 **OA 连接、AI 模型、Token 配额** 授给租户，租户管理员再在租户内配置规则与组织权限。智能体能力必须走同一模式，禁止租户直接使用平台未授予的工具或 MCP。

```
系统管理员大开关（对话 / 允许 MCP / 允许 Skill / Token 配额）
        │
        ▼
租户管理员以智能体为单位装配
        ├─► 智能体绑定系统工具、MCP、Skill
        └─► 组织角色只授权「可用哪些智能体」
                │
                ▼
         业务用户有效集
```

运行时有效工具见 [permissions.md](./permissions.md)。本文只定义**谁把什么分给谁**。

## 1. 平台目录（系统管理员维护，不分给用户）

| 对象 | 所有者 | 说明 |
|------|--------|------|
| 系统工具 | 代码注册 | `tool_code` 稳定，随版本发布；管理端只读展示 |
| 内置智能体 | 代码/种子数据 | `oa_query`、`oa_assist`；系统管理员可改「是否允许分配给租户」 |
| 内置 Skills | 仓库或种子 | 平台级 `SKILL.md` |
| MCP 模板 | 系统配置 | 可选：预置 URL/协议说明；密钥仍建议租户自填或系统代管后加密 |

系统管理员**不**在业务对话里使用这些能力。

## 2. 第一级：系统管理员 → 租户

配置位置（实现时）：**系统管理 → 租户管理**，与 AI 模型、OA 连接同一租户详情，建议独立 Tab「智能体与工具」。

租户级字段：

| 字段 | 含义 |
|------|------|
| `chat_enabled` | 总开关。关闭则该租户无对话入口、无管理端智能体配置 |
| `chat_retention_days` | 会话保留天数；默认来自 `tenant.default_chat_retention_days` |
| `chat_primary_model_id` / `chat_fallback_model_id` | 可选；空则用租户已有 `primary_model_id` / `fallback_model_id` |
| Skills 开关 | 是否允许租户编写自定义 Skill |
| MCP 开关 | 是否允许租户自建 MCP 连接及最多个数 |
| `token_quota` | Token 配额；**小于 0 表示不限制** |

约束：

- 关闭 `chat_enabled` 后运行时全部拒绝，已有租户配置只读保留。
- 系统管理员不再勾选具体智能体、工具或 Skill；装配在租户智能体上完成。
- Embed：沿用租户 `embed_enabled` + embed token；另发 `/embed/chat` 地址。未打开对话开关则嵌入对话同样不可用。

系统默认（新建租户）：

- `chat_enabled=true`
- 允许自定义 Skill / 自建 MCP：默认否（由系统管理员按客户打开）

`tenant.default_chat_retention_days` 放在系统设置「通用」，与现有 `tenant.default_log_retention_days` 并列。不要复用尚未驱动清理的 `data_retention_days`（见 [known-issues/002](../known-issues/002-audit-data-retention-cleanup.md)）。

## 3. 第二级：租户管理员 → 智能体与角色

配置位置（实现时）：

- **租户管理 → 智能体**：在配额内 CRUD 智能体、绑定工具/MCP/Skills
- **租户管理 → 组织架构 → 角色**：除 `page_permissions` 外，只授权智能体（不选=全部已启用）
- MCP / Skills 在智能体管理页挂载到对应智能体

再分配规则：

1. 角色未授予 `/chat` 则不能进入对话页。
2. 用户有效智能体 = 其角色授权并集 ∩ 已启用；角色未勾选智能体时视为全部已启用。
3. 用户有效工具 = 当前智能体绑定 ∩ 租户 MCP/Skill 大开关 ∩ 启用状态。

典型再分配：

| 角色意图 | `/chat` | 智能体 |
|----------|---------|--------|
| 仅查询 | 有 | `oa_query` |
| 辅助办理 | 有 | `oa_query` + `oa_assist` |
| 全部智能体 | 有 | 不选 |
| 不可对话 | 无 | — |

## 4. 数据关系（拟定表）

实现时用迁移创建，名称可微调，语义固定：

| 表 | 层级 | 用途 |
|----|------|------|
| （代码）系统工具注册表 | 平台 | `tool_code`、schema、`ui_kind`、适配器能力 |
| `agent_definitions` | 平台种子 + 租户自定义 | 智能体主数据；租户行带 `tenant_id` |
| `tenant_chat_settings` 或 `tenants` 扩展列 | 租户 | `chat_enabled`、保留天数、对话模型 |
| `tenant_agent_allocations` | 系统→租户 | 租户可用 `agent_code` |
| `tenant_tool_allocations` | 系统→租户 | 租户可用 `tool_code` |
| `tenant_skill_allocations` | 系统→租户 | 租户可用 Skill |
| `tenant_mcp_allocations` | 系统→租户 | 是否允许自建 MCP、模板 ID |
| `agent_tool_bindings` 等 | 租户智能体 | 绑定 ⊆ 配额 |
| `org_role_agent_grants` | 租户→角色 | 再分配智能体 |
| `org_role_tool_grants` | 租户→角色 | 再分配工具 |
| `mcp_servers` | 系统模板或租户连接 | 加密凭据 |
| `agent_skills` | 平台或租户 | Skill 包 |
| `chat_sessions` / `chat_messages` | 用户 | 会话；含 `agent_id` |

列表接口分页遵循项目 `page` / `page_size` 规范。

## 5. 与现有分配对照

| 现有能力 | 系统管理员 | 租户管理员 |
|----------|------------|------------|
| OA 连接 | 创建连接并挂到 `tenants.oa_db_connection_id` | 不能改连接串；用该连接做规则/工作台 |
| AI 模型 | 维护目录；租户选主备模型 | 流程上调温度等（有限度） |
| Token | `token_quota` | 不能突破配额 |
| 页面 | — | `org_roles.page_permissions` |
| **智能体** | 对话/MCP/Skill 大开关 | 组装智能体 + 角色再分配智能体 |
| **工具 / MCP / Skill** | 是否允许 MCP、Skill | 挂到智能体，不再单独授给角色 |

OA 连接仍是系统管理员分配：未绑定 OA 的租户，即使有工具配额，OA 类系统工具执行失败并提示未配置 OA。适配器细节见 [oa-adapter.md](./oa-adapter.md)。

## 6. 管理端验收

- 系统管理员只保存大开关，不再勾选具体智能体/工具/Skill。
- 租户管理员在智能体、MCP、Skill 上完成挂载；角色只选择可见智能体。
- 关闭 MCP/Skill 或停用智能体后，业务用户下一轮对话立即按新有效集裁剪工具。
