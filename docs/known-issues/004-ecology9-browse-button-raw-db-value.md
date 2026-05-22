# [#004] 泛微 E9 浏览按钮仅读取数据库原始值

## 状态

**已实现基础解析**（人员、部门、分部、多人力资源、自定义 `browser.xx` 的常见 SQL 配置）

## 问题描述

泛微 Ecology E9 中，`workflow_billfield.fieldhtmltype = 3` 表示**浏览按钮**字段。AuraOA 在拉取流程数据时，会按 `workflow_billfield.type` / `fielddbtype` 尝试把浏览按钮字段的原始 ID 增补为显示值。

当前实现要点：

1. **字段定义**（`FetchFields`）：`fieldhtmltype = 3` 仍映射为通用类型 `select`，用于前端展示与字段勾选。
2. **流程数据**（`FetchProcessData`）：先从 `formtable_main_*` 主表及 `formtable_main_*_dt*` 明细表读取原始值，再对浏览按钮字段做显示值解析。
3. **内置浏览按钮**：已支持单人力资源（`TYPE=1`）、部门（`TYPE=2`）、分部（`TYPE=3`）、多人力资源（`TYPE=17`）。
4. **自定义浏览框**：对 `TYPE=161/162` 或 `FIELDDBTYPE=browser.xx` 的字段，会查询 `workflow_browserurl`，从 `browserurl` 中常见的 `SQL=select id,name from uf_xxx ...` 解析出关联表和显示字段。
5. **AI 审核**：提示词中的 `{{main_table}}` / `{{detail_tables}}` 会看到结构化值，例如 `{"value":"12","display":"张三"}`；多选字段会包含 `items`。

作为对比：**附件字段**（`fieldhtmltype = 6`）会识别 docId 并调用泛微 `weaver_api_url` 拉取文件内容；浏览按钮解析则直接基于 OA 数据库中的字段定义与关联表。

## 影响范围

| 场景 | 是否受影响 |
|------|------------|
| 人员、部门、分部、多人力资源浏览按钮字段参与 AI 审核 | ✅ 会增补显示名 |
| 自定义浏览框字段参与 AI 审核 | ✅ 常见 `workflow_browserurl.browserurl` SQL 可解析时会增补显示名 |
| 规则中要求核对「申请人姓名」「所属部门名称」等显示信息 | ✅ 已覆盖常见浏览按钮；特殊自定义 SQL 仍需校验 |
| 纯文本、多行文本、选择框（`fieldhtmltype` 1/2/5） | ✅ 不受影响 |
| 附件识别（`fieldhtmltype = 6`） | ✅ 有独立解析，与本条无关 |
| 待办/归档列表、审批流快照 | ✅ 人员姓名等来自 `hrmresource` 等表，非浏览按钮字段解析 |

## 泛微 E9 侧常见存储方式（参考）

| 存储内容 | AuraOA 当前行为 |
|----------|-----------------|
| 浏览对象主键（如 `123`） | ✅ 保留为 `value` |
| 可通过关联表查到的显示名 | ✅ 增补为 `display` |
| 页面运行时计算或复杂 SQL 拼接的显示名 | ⚠️ 可能无法解析，保留原始值 |

具体列名与是否另有显示列，以各客户 E9 表单物理表结构为准；AuraOA 不依赖泛微前端渲染逻辑。

## 仍需注意

1. 自定义浏览框只解析常见的 `select id, display_col from uf_xxx ...` 形态；若显示字段来自函数、拼接表达式、复杂视图逻辑，可能无法自动识别。
2. 解析失败不会阻断审核流程，字段会保持 OA 原始值。
3. 当前不读取物理表中可能存在的 `fieldname + span` 显示列。

## 后续扩展方向

| 方案 | 说明 |
|------|------|
| 读取 `fieldname + span` 显示列 | 若物理表存在 E9 标准 span 列，一并输出显示名 |
| 泛微 REST / 自定义桥接 | 与附件接口类似，调用 OA 侧显示值转换 API |
| 扩展更多内置 TYPE | 按客户实际 `workflow_browserurl` 数据补充岗位、角色、流程等内置浏览按钮 |

## 参考代码

- 字段类型映射：`go-service/internal/pkg/oa/ecology9.go` — `mapFieldType`（`case "3"`）
- 字段定义拉取：`go-service/internal/pkg/oa/ecology9.go` — `FetchFields`
- 流程数据拉取：`go-service/internal/pkg/oa/ecology9.go` — `FetchProcessData`
- 提示词注入：`go-service/internal/service/audit_prompt_builder.go` — `formatMainData` / `formatGroupedDetailData`

## 相关文档

- [OA 系统对接说明](../oa-integration.md)
- [附件识别（对比：fieldhtmltype=6 有解析）](../oa-configurations/01-attachment-recognition.md)
