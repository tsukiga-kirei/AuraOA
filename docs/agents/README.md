# 智能体与 OA 对话

本目录是 AuraOA **智能体运行时**的需求与设计契约：对话入口、系统工具、MCP、Skills，以及 **系统管理员分配 → 租户管理员再分配**。

> **状态（2026-09-05）**：独立聊天、两级分配、多智能体装配、系统工具、指令 Skill 和 Streamable HTTP MCP 已实现。具体范围以 docs/api/chat.md、docs/api/agents.md 为准；专用嵌入聊天、脚本型技能及本地 stdio MCP 尚未提供。

对话**不替换**审核工作台、归档复盘、流程总结，而是通过系统工具调用它们，并只读访问 OA（经 [`OAAdapter`](../../go-service/internal/pkg/oa/adapter.go)）。真正批准/退回仍在 OA 中操作。

## 先读哪份

| 文档 | 用途 |
|------|------|
| [需求说明书](./requirements.md) | 背景、范围、角色场景、验收标准 |
| [两级分配](./allocation.md) | 系统管理员分给租户，租户管理员再分给角色/智能体 |
| [架构](./architecture.md) | 会话、编排循环、有效工具集、SSE |
| [OA 适配器](./oa-adapter.md) | 系统工具如何走适配器；新 OA 类型如何接入 |
| [系统工具](./system-tools.md) | 每个内置工具的 code、权限键、适配器方法、`ui_kind` |
| [MCP](./mcp.md) | 外部 MCP 登记、发现、调用与授权键 |
| [Skills](./skills.md) | `SKILL.md` 包格式与加载 |
| [权限](./permissions.md) | 页面 / 租户配额 / 智能体 / 工具四层收敛 |
| [工具可视化](./ui-visualization.md) | 每种工具的前端卡片与 SSE payload |
| [如何新增](./authoring.md) | 加智能体、工具、MCP、Skill 的检查清单 |

已实现接口（与 OpenAPI 同步）：

- [`docs/api/chat.md`](../api/chat.md) — 业务用户对话
- [`docs/api/agents.md`](../api/agents.md) — 系统分配与租户智能体管理

相关既有文档：[`ai-integration.md`](../ai-integration.md)、[`oa-integration.md`](../oa-integration.md)、[`oa-configurations/`](../oa-configurations/README.md)。

## 核心原则

1. **维护单位是智能体**，不是「一个写死的聊天机器人」。后续 OA 相关智能体按同一目录扩展。
2. **能力分三类**：系统工具（代码注册）、MCP（外部协议）、Skills（指令包，可选升为工具）。
3. **分配分两级**：系统管理员决定租户能用哪些能力；租户管理员在配额内再绑智能体、再授给组织角色。下级不能突破上级配额。
4. **OA 读写边界**：系统工具只调用适配器；当前适配器只读。写 OA 不在本期范围，也不准绕过适配器直连业务库。
5. **每种系统工具独立可视化**；MCP/Skills 与系统工具统一使用紧凑执行行，点击展开详情。

## 内置智能体（种子）

| `agent_code` | 名称 | 默认绑定 |
|--------------|------|----------|
| `oa_query` | OA 查询 | 待办、流程详情、审批流、已有审核/总结只读 |
| `oa_assist` | OA 辅助办理 | 上述 + 起草意见、触发审核/总结、OA 跳转 |

系统管理员可不把某个种子分给某租户；租户管理员可在分到的范围内停用或改绑（改绑不得越出配额）。
