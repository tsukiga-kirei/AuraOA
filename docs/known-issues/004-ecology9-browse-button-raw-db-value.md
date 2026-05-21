# [#004] 泛微 E9 浏览按钮仅读取数据库原始值

## 状态

**已知限制**（当前版本按设计接受；浏览值解析待后续扩展）

## 问题描述

泛微 Ecology E9 中，`workflow_billfield.fieldhtmltype = 3` 表示**浏览按钮**字段。AuraOA 在拉取流程数据时，对该类字段**不做 ID→显示名解析**，而是与文本框等字段一样，直接从表单物理表读取列上的**数据库原始值**。

当前实现要点：

1. **字段定义**（`FetchFields`）：`fieldhtmltype = 3` 映射为通用类型 `select`，仅用于前端展示与字段勾选，不参与取值逻辑。
2. **流程数据**（`FetchProcessData`）：按 `workflow_billfield.fieldname` 对应列，从 `formtable_main_*` 主表及 `formtable_main_*_dt*` 明细表 `SELECT` 整行数据；浏览按钮取值 = 该列在库中的原始字符串（通常为浏览对象主键 ID，如人员 ID、部门 ID）。
3. **无关联查询**：不会根据 `workflow_billfield.type` / `fielddbtype` 判断浏览类型，也不会 JOIN `hrmresource`、`hrmdepartment` 或自定义浏览表；不会读取 E9 表单上可能存在的 `字段名span` 等显示列。
4. **AI 审核**：提示词中的 `{{main_table}}` / `{{detail_tables}}` 注入的 JSON 即为上述原始值；模型看到的是 ID 而非页面上的人名、部门名等显示文本。

作为对比：**附件字段**（`fieldhtmltype = 6`）会识别 docId 并调用泛微 `weaver_api_url` 拉取文件内容；浏览按钮**没有**类似的二次解析链路。

## 影响范围

| 场景 | 是否受影响 |
|------|------------|
| 人员、部门、客户等浏览按钮字段参与 AI 审核 | ⚠️ 模型可能只看到数字 ID，无法理解业务含义 |
| 规则中要求核对「申请人姓名」「所属部门名称」等显示信息 | ⚠️ 若字段为浏览按钮且仅存 ID，审核结论可能偏差 |
| 纯文本、多行文本、选择框（`fieldhtmltype` 1/2/5） | ✅ 不受影响 |
| 附件识别（`fieldhtmltype = 6`） | ✅ 有独立解析，与本条无关 |
| 待办/归档列表、审批流快照 | ✅ 人员姓名等来自 `hrmresource` 等表，非浏览按钮字段解析 |

## 泛微 E9 侧常见存储方式（参考）

| 存储内容 | AuraOA 当前行为 |
|----------|-----------------|
| 浏览对象主键（如 `123`） | ✅ 原样读出 |
| 页面显示名（可能在 `xxxspan` 列或运行时计算） | ❌ 不读取、不解析 |

具体列名与是否另有显示列，以各客户 E9 表单物理表结构为准；AuraOA 不依赖泛微前端渲染逻辑。

## 临时方案（不改代码）

1. **字段选择**：在规则/工作台字段配置中，优先勾选已存明文或可读文本的字段；避免仅依赖浏览按钮 ID 字段做关键判断。
2. **规则表述**：在审核规则中说明「若某字段为数字 ID，可结合同表其他文本字段或流程摘要判断」，降低模型对 ID 的误读。
3. **OA 表单设计**：若业务强依赖显示名参与审核，可在表单中增加冗余文本字段（由 E9 公式/触发器写入显示名），并在 AuraOA 中勾选该文本字段。

## 后续扩展方向（未实现）

| 方案 | 说明 |
|------|------|
| 按 `workflow_billfield.type` 解析 | 人员 → `hrmresource`，部门 → `hrmdepartment`，自定义浏览 → 对应业务表 |
| 读取 `fieldname + span` 显示列 | 若物理表存在 E9 标准 span 列，一并输出显示名 |
| 泛微 REST / 自定义桥接 | 与附件接口类似，调用 OA 侧显示值转换 API |

## 参考代码

- 字段类型映射：`go-service/internal/pkg/oa/ecology9.go` — `mapFieldType`（`case "3"`）
- 字段定义拉取：`go-service/internal/pkg/oa/ecology9.go` — `FetchFields`
- 流程数据拉取：`go-service/internal/pkg/oa/ecology9.go` — `FetchProcessData`
- 提示词注入：`go-service/internal/service/audit_prompt_builder.go` — `formatMainData` / `formatGroupedDetailData`

## 相关文档

- [OA 系统对接说明](../oa-integration.md)
- [附件识别（对比：fieldhtmltype=6 有解析）](../oa-configurations/01-attachment-recognition.md)
