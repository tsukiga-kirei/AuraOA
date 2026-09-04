# 如何新增智能体 / 工具 / MCP / Skill

改智能体相关能力时，**先改本目录文档，再改代码**，并走两级分配：系统管理员配额默认值、租户再分配是否要回填。

## 新增系统工具

1. 确认数据来自 [`OAAdapter`](../../go-service/internal/pkg/oa/adapter.go) 或现有 Service，不直连 OA 库。
2. 若读单流程：必须经过 `CheckProcessVisibility`。
3. 在 [system-tools.md](./system-tools.md) 增加 `tool_code`、适配器方法、`ui_kind`、是否 `oa_required`。
4. 在 [ui-visualization.md](./ui-visualization.md) 增加卡片与 payload。
5. 代码注册 `ToolSpec` + 执行器 + 前端组件。
6. 决定默认进入哪些种子智能体；是否加入「新建租户」系统配额默认列表（[allocation.md](./allocation.md)）。
7. 已有租户：**不自动**扩大配额（避免未评估的能力被打开）。需要时由系统管理员在租户 Tab 勾选。
8. 更新 [`docs/api/agents.md`](../api/agents.md) 工具枚举；OpenAPI 若暴露目录接口则同步。
9. i18n：工具名、卡片标签。

`risk=write` 必须单独立项，并先扩展适配器写接口，不得先做前端按钮。

## 新增内置智能体

1. 确定 `agent_code`、提示词、默认绑定工具 ⊆ 拟分配给租户的默认配额。
2. 更新 [README.md](./README.md) 种子表与 [requirements.md](./requirements.md) 场景。
3. 种子数据迁移；系统管理员默认配额是否包含该智能体（新建租户）。
4. 已有租户同样不自动分配。

## 新增 MCP 模板或传输

1. 更新 [mcp.md](./mcp.md)（传输、超时、命名）。
2. 系统管理员：是否允许租户使用该模板。
3. 工具键仍为 `mcp:{server}:{tool}`；通用卡片不够时再加 `ui_kind`。

## 新增内置 Skill

1. 按 [skills.md](./skills.md) 写 `SKILL.md`。
2. 加入平台目录与默认配额策略（默认不自动给已有租户）。
3. 若含脚本：写明一期是否执行；授权键 `skill:{code}:{script}`。

## 新 OA 类型

走 [oa-adapter.md](./oa-adapter.md) 第 7 节，并更新 [`docs/oa-integration.md`](../oa-integration.md)。工具 code 保持不变。

## 文档与代码冲突

以代码为准，同一 PR 回改本目录与 `docs/api/chat.md`、`docs/api/agents.md`、`auraoa.openapi.yaml`。
