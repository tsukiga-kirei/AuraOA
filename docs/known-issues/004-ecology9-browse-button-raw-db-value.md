# [#004] 泛微 E9 表单字段显示值解析覆盖范围

## 状态

**已部分修复 / 持续扩展**。已支持常见浏览按钮、相关流程、自定义 `browser.xx`、系统集成浏览按钮、选择框 / 下拉框显示值解析；复杂自定义 SQL 或特殊客户二开字段仍可能保留原始值。

## 问题描述

泛微 Ecology E9 中，表单物理表 `formtable_main_*` / `formtable_main_*_dt*` 经常只保存数据库值，例如人员 ID、部门 ID、流程 requestid、选择框枚举值。AuraOA 在拉取流程数据后，会按 `workflow_billfield`、`workflow_browserurl`、`workflow_selectitem` 等元数据尽量把这些值翻译成业务可读文本，再进入 AI prompt。

当前实现要点：

1. **字段定义**（`FetchFields`）：`fieldhtmltype = 3/5` 都映射为通用类型 `select`，用于前端展示与字段勾选。
2. **流程数据**（`FetchProcessData`）：先读取主表 / 明细表原始值，再按字段选择集只解析会发送给 AI 的字段。
3. **字段中文名**：通过 `workflow_billfield.fieldlabel → htmllabelinfo.indexid`，优先使用 `languageid=7` 的中文标签。
4. **浏览按钮通用解析**：优先按 `workflow_billfield.type = workflow_browserurl.id` 查询定义。若 `TABLENAME`、`COLUMNAME`、`KEYCOLUMNAME` 都不为空，则通过 `KEYCOLUMNAME` 查值并展示 `COLUMNAME`。
5. **内置浏览按钮兜底**：若 `workflow_browserurl` 元数据不完整，再使用人员、部门、分部、相关流程等少量兜底映射；`TYPE` 映射以当前客户环境为准，例如本环境中 `TYPE=2` 是日期，不是部门。
6. **自定义 / 集成浏览框**：对 `TYPE=161/162/226/256/257` 或 `FIELDDBTYPE=browser.xx` 的字段，会查询 `workflow_browserurl`，优先使用通用元数据；`161/162` 这类建模浏览框会继续按 `FIELDDBTYPE=browser.xxx → mode_browser.SHOWNAME=xxx` 查询 `SQLTEXT` / `SEARCHBYID`，解析出关联表和显示字段。
7. **选择框 / 下拉框**：对 `fieldhtmltype = 5` 的字段，通过 `workflow_billfield.id = workflow_selectitem.fieldid` 和 `selectvalue` 匹配选项；`selectname` 若为泛微多语言串，优先取语言 `7`。
8. **AI 审核**：prompt 中会尽量只展示中文字段名和业务显示值，例如 `"报销人": "张三"`、`"酒店级别": "四星级"`，不暴露 `value/display` 结构。

作为对比：**附件字段**（`fieldhtmltype = 6`）会识别 docId 并调用泛微 `weaver_api_url` 拉取文件内容；浏览按钮解析则直接基于 OA 数据库中的字段定义与关联表。

## 影响范围

| 场景 | 是否受影响 |
|------|------------|
| 人员、部门、分部、相关流程浏览按钮字段参与 AI 审核 | ✅ 会增补显示名 |
| 自定义 / 集成浏览框字段参与 AI 审核 | ✅ `workflow_browserurl` 通用元数据或 `mode_browser` 建模配置可解析时会增补显示名 |
| 选择框 / 下拉框字段参与 AI 审核 | ✅ 通过 `workflow_selectitem` 解析枚举显示名 |
| 规则中要求核对「申请人姓名」「所属部门名称」等显示信息 | ✅ 已覆盖常见浏览按钮；特殊自定义 SQL 仍需校验 |
| 纯文本、多行文本（`fieldhtmltype` 1/2） | ✅ 不受影响 |
| 附件识别（`fieldhtmltype = 6`） | ✅ 有独立解析，与本条无关 |
| 待办/归档列表、审批流快照 | ✅ 人员姓名等来自 `hrmresource` 等表，非浏览按钮字段解析 |

## 泛微 E9 侧常见存储方式（参考）

| 存储内容 | AuraOA 当前行为 |
|----------|-----------------|
| 浏览对象主键（如 `123`） | ✅ 尝试查关联表显示名 |
| 多选浏览对象主键（如 `1,3,5`） | ✅ 逗号拆分后批量查询显示名 |
| `workflow_browserurl` 存在 `TABLENAME/COLUMNAME/KEYCOLUMNAME` | ✅ 通过 `KEYCOLUMNAME` 查询并展示 `COLUMNAME` |
| 选择框枚举值（如 `0` / `1`） | ✅ 通过 `workflow_selectitem.fieldid + selectvalue` 查显示名 |
| `selectname` 多语言串 | ✅ 优先取语言 `7`，再回退到 `9` / `8` |
| 页面运行时计算或复杂 SQL 拼接的显示名 | ⚠️ 可能无法解析，保留原始值 |

具体列名与是否另有显示列，以各客户 E9 表单物理表结构为准；AuraOA 不依赖泛微前端渲染逻辑。

## 仍需注意

1. 自定义浏览框只解析常见的 `select id, display_col from uf_xxx ...` 形态；若显示字段来自函数、拼接表达式、复杂视图逻辑，可能无法自动识别。
2. 解析失败不会阻断审核流程，字段会保持 OA 原始值。
3. 当前不读取物理表中可能存在的 `fieldname + span` 显示列。
4. 不同 E9 版本或客户二开环境的 `workflow_billfield.type` 含义可能存在差异，新增内置映射前必须以现场配置核验。

## 后续扩展方向

| 方案 | 说明 |
|------|------|
| 读取 `fieldname + span` 显示列 | 若物理表存在 E9 标准 span 列，一并输出显示名 |
| 泛微 REST / 自定义桥接 | 与附件接口类似，调用 OA 侧显示值转换 API |
| 扩展更多内置 TYPE | 按客户实际配置补充岗位、角色、资产、客户等内置浏览按钮 |

## 参考代码

- 字段类型映射：`go-service/internal/pkg/oa/ecology9.go` — `mapFieldType`（`case "3"` / `case "5"`）
- 字段定义拉取：`go-service/internal/pkg/oa/ecology9.go` — `FetchFields`
- 流程数据拉取：`go-service/internal/pkg/oa/ecology9.go` — `FetchProcessData`
- 浏览按钮显示值解析：`go-service/internal/pkg/oa/ecology9.go` — `ResolveBrowseDisplayValues`
- 提示词注入：`go-service/internal/service/ai_utils.go` — `formatMainData` / `formatGroupedDetailData`

## 相关文档

- [OA 系统对接说明](../oa-integration.md)
- [附件识别（对比：fieldhtmltype=6 有解析）](../oa-configurations/01-attachment-recognition.md)
