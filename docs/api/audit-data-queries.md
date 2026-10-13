# 记忆库：数据记忆

入口：租户管理 → 记忆库 → 数据记忆（`/admin/tenant/memory`）。菜单位于规则配置下方，使用书本图标；规则编辑器通过专用选择器引用已发布条目，不嵌入维护工作区。

这是独立的通用取数库，使用当前租户 OA 连接，不依赖已有的关联流程、建模挂载或 requestid 关联模式。管理员维护查询模板，审核和归档规则引用固定的发布版本；每次执行读取当前数据库数据，保留当次证据。

## 页面与配置方式

记忆库包含两个页签：**数据记忆**承载本次可复用取数能力；**评审记忆**仅展示占位说明，暂不采纳反馈、不改写提示词，也不自动参与审核。数据记忆每次审核实时查询，不新增每日同步或复制业务数据的任务。

- **配置查询**：输入表或视图名并读取真实字段，选择输出列、中文列名、SUM/COUNT/MIN/MAX/AVG、AND 筛选及排序。含聚合时自动将非聚合字段加入 GROUP BY。COUNT 支持 `*`。
- 来源选择**流程表**时，单独设置“仅查询已归档流程”开关，新配置默认关闭；开启后要求业务表包含 `requestid`，用它关联 `workflow_requestbase`，并以 `currentnodetype='3'` 判断归档状态。`requestid` 存在不代表已归档。旧配置未指定 `builder.archived_only` 时沿用原有仅归档口径。状态条件完整写入 SQL，配置与自定义 SQL 都可见，执行端不隐式追加。建模表/普通表/视图不自动加状态条件。
- 筛选条件按“字段 → 比较关系 → 固定值/流程字段/系统值”配置；固定值直接填写，流程字段允许暂不选择实际字段，只填写可选中文用途，引用规则时再下拉绑定；系统值下拉选择。英文参数名自动生成并隐藏于配置模式，SQL 模式从占位符自动识别变量，不再手工新增英文参数名。值仍通过驱动参数绑定，支持 LIKE、IN、空值条件；日期参数在 Oracle/达梦生成 TO_DATE 转换。LIKE 通配符通过参数值提供。
- **自定义 SQL**：支持复杂关联、多个流程表的字段映射与合并、嵌套查询等，以服务端 SQL 子集检查为准；不尝试将任意 SQL 反推为配置。编辑 SQL 时自动识别 `:name` 值占位符，忽略字符串、带引号标识符、注释及双冒号转换；变量区只配置中文用途、类型和取值来源，没有占位符时不显示。相同变量保留已有设置；SQL 删除变量后从当前参数数组移除，仍保留服务端参数一致性检查。
- 执行 SQL 预览默认展开。`definition.mode` 区分 `config` / `sql`，两种编辑方式互斥：切到自定义 SQL 保留已生成 SQL 并清除 builder，切回配置会清空 SQL 和变量设置；有编辑内容时先确认。修改配置后清空旧 SQL 和参数，必须重新生成。旧条目根据 builder 与旧 sql_mode 标记推断模式，不反向解析 SQL。
- 两种方式共用后续检查、预览、发布版本及规则绑定。`definition.builder` 是可选的前端配置元数据（最多30000字节），随草稿和发布版本保存以便再次编辑；后端执行和安全校验始终以 `sql` 为准。

进入页面默认打开第一条数据记忆；只有目录为空或点击新建时进入新建表单。左侧条目目录沿用规则配置的分隔行、左侧选中标记和加号入口，静态说明使用中性辅助文字；不展示分页，前端按每批100条读取全部匹配结果并在目录内滚动。复制位于左侧每条目录项旁，确认后创建待保存的独立副本；保存草稿位于编辑区右上角。输出字段、筛选条件、结果排序均可独立折叠，默认展开。编辑区按基本信息、SQL 配置、AI 辅助 SQL 生成、预览与管理分区；预览与管理内集中展示检查报告、测试与预览、发布版本、引用与管理四个内部页签，操作位于对应位置。读取字段成功、校验结果及错误、截断提醒统一使用顶部消息提示；版本与查询时间按应用时区显示中文日期时间，不展示时区后缀。

当前配置方式覆盖单表/视图，表名由管理员输入，字段从租户 OA 连接读取；尚未提供流程目录与建模目录搜索、多表可视化连接和任意 OR 条件组。复杂场景使用自定义 SQL 或 AI SQL 辅助。

读取字段后，OA 中文标签默认填入空白业务说明、已有输出列的空白中文别名及筛选用途；字段名匹配不区分大小写，手工填写的内容优先保留。主表与各明细表的标签按字段所属物理表区分，避免同名字段互相覆盖；字典查询失败时保留物理字段并记录警告。

## 权限和生命周期

所有接口要求 JWT → TenantContext → `tenant_admin` → 页面权限 `/admin/tenant/memory`。新增租户的默认管理员角色和迁移前已有的系统租户管理员角色默认获得该权限；其他角色由组织人员页面授权。

查询定义、版本、提示词按租户隔离。没有查询库权限的规则编辑者可以原样保留已有引用，不能新增或变更查询绑定；实际审核沿用既有流程访问权限，不要求业务用户拥有 SQL 管理权限。

草稿允许未完成的 SQL。发布必须同时通过后端检查，并且对**当前已保存草稿及当前 OA 连接配置**成功预览。发布后版本不可修改；更新草稿不改变规则引用。删除前必须取消全部当前审核/归档规则引用，包含关闭的规则及关闭的引用；服务端在事务中再次检查。删除从当前记忆库移除条目，历史不可变版本保留供既有执行快照追溯；已删除条目不可读取/修改草稿、发布或重新加入当前规则。

“引用与管理”仅按流程及审核/归档模块合并展示当前引用，可取消该流程在对应模块全部当前规则中的引用，不改动其他条目及历史配置快照；需同时具有记忆库与规则配置页面权限。取消后刷新对应模块的配置缓存；与规则编辑一致，使用已发布配置的流程需另行发布修改后生效。红色删除按钮位于保存草稿旁，需二次确认。

## 路由

路由前缀：`/api/tenant/audit-data-queries`。统一成功响应 `{code:0,message:"success",data:...}`。

| 方法 | 路径 | 参数 / 返回 data |
|------|------|------|
| GET | `/` | `keyword` 搜索名称/分类，`page=1`、`page_size=20`（1–100）；返回 `{items,total,page,page_size}` |
| POST | `/` | `{category,definition}`；创建草稿，返回查询条目 |
| GET | `/:id` | 查询条目，包括 `draft`、`latest_version`、`tested_hash`、`tested_at` |
| PUT | `/:id` | `{category,definition}`；更新草稿并清除预览标记 |
| DELETE | `/:id` | 无当前规则引用时删除条目；仍有引用返回参数错误；返回 `{deleted:true}` |
| GET | `/:id/versions` | 发布版本数组，按 `version_no` 倒序 |
| POST | `/:id/publish` | 无请求体；返回 `{id,tenant_id,query_id,version_no,definition,created_at}` |
| GET | `/:id/references` | 当前审核/归档规则引用数组 `{id,process_type,rule_content,module}`，`module` 为 `audit_rules` / `archive_rules`；不枚举历史执行快照 |
| DELETE | `/:id/references` | `{module,process_type}`：取消对应模块及流程全部当前规则的引用；需额外规则配置页面权限；返回 `{removed_count}`（受影响规则数） |
| POST | `/check` | 请求体为 `definition`；返回结构检查报告 |
| POST | `/preview` | 手工参数或流程绑定预览，返回证据对象 |
| GET | `/source` | 返回 `{name,dialect,oa_type,oa_name,timezone,business_date}`：管理员已配置的租户 OA 数据源与应用业务日期，无连接凭据；未配置/已停用返回明确错误 |
| GET | `/schema` | `table`：固定表名；返回 `{table,dialect,columns:[{key,label,data_type}]}`，通过零行查询读取物理结构，key 为物理字段名；label 优先取流程/建模表所属主表或明细表的中文标签，兼容 MySQL、Oracle、达梦列名大小写与明细表序号；无标签时回退物理字段名 |
| POST | `/assist` | AI 生成/解释/修复，返回可编辑建议 |
| GET | `/prompts` | 返回租户提示词；未配置时返回系统默认值 |
| PUT | `/prompts` | `{generate,explain,repair}`：各必填、最多12000字节 |

请求体上限256 KiB。分页越界回落默认值；空列表返回 `items:[]`。

## 查询定义

```json
{
  "mode": "sql",
  "ai_context": {"oa_module":"workflow","notes":"按业务单据去重，金额单位为元。","field_notes":{}},
  "name": "申请人年度已完成报销",
  "description": "仅统计已完成报销，人民币元；排除当前流程。按业务单据去重后汇总。",
  "dialect": "mysql",
  "tables": ["formtable_main_123", "workflow_requestbase"],
  "sql": "SELECT COALESCE(SUM(e.amount), 0) AS `已报销金额` FROM formtable_main_123 e JOIN workflow_requestbase r ON r.requestid=e.requestid WHERE e.employee=:employee_id AND e.requestid<>:current_request_id AND r.currentnodetype=3 AND e.expense_date>=:year_start",
  "parameters": [
    {"name":"employee_id","label":"申请人员ID","type":"string"},
    {"name":"current_request_id","label":"当前流程ID","type":"string"},
    {"name":"year_start","label":"年度开始日期","type":"date"}
  ],
  "max_rows": 0
}
```

`max_rows` 可选，省略或0表示不限制返回行数；正整数表示指定上限，不再限制为100以内。前端留空或输入0时提交0，已有版本中的正数上限继续生效。两种编辑方式共用“执行 SQL 预览”：指定上限时显示与服务端相同的外层 `LIMIT max_rows+1`（MySQL）或 `ROWNUM <= max_rows+1`（Oracle/达梦），并说明额外一行仅用于截断判定；不限制时直接显示原始 SQL。该包装仅展示，不能写回模板，避免重复限制干扰截断判断。

示例表名、字段和完成状态须按实际 OA 环境确认，不能直接当作项目内置预算口径。预算累计建议用聚合查询；同一事项分散报销可按人员、项目、合同、时间段等组合条件返回历史明细，由规则指导 AI 判断，不能仅凭同一人员即认定拆单。

支持 MySQL、Oracle、达梦；工作区从 `/source` 自动读取管理员已配置的租户连接类型，只读展示，无需用户选择。参数 `dialect` 保留用于已发布版本记录及服务端一致性检查，必须与实际连接一致。参数类型：`string`、`number`、`date`、`string_list`。列表使用 `IN (:name)`，按元素绑定；日期输入为 `YYYY-MM-DD`，Oracle/达梦的 DATE 比较应使用 `TO_DATE(:date,'YYYY-MM-DD')`。流程字段绑定使用原始数据库值，避免人员 ID 被浏览字段显示名称替换。

### 检查和执行范围

- 仅单条 SELECT / 顶层只读 WITH、常见无副作用内置函数；禁止多语句、注释、写入、锁定、自定义函数、限定函数调用、跨库表名及表函数。不支持的复杂结构需改写为此查询子集。
- 每个最终输出列必须显式 `AS` 中文业务别名，且名称唯一；禁止最终通配列。MySQL 推荐反引号，Oracle/达梦用双引号。统计表达式如 `COUNT(*) AS 笔数` 可用。
- 所有物理表必须在 `tables` 中声明；SQL 使用的参数必须定义且没有多余定义。参数标签须包含中文。表和参数各最多20项、输出列最多40项、SQL最多30000字节。
- 检查返回 `{valid,issues:[{code,message,position}],parameters,columns}`；`position` 从1开始，供编辑器定位。检查不证明字段存在、数据库语法或统计口径正确，需继续预览。
- 值始终使用驱动绑定；执行在数据库只读事务中，10秒超时，结束回滚，无法开启只读事务时拒绝执行。Oracle 使用首条 `SET TRANSACTION READ ONLY`，不会降级为可写执行。
- 不设固定行数上限；配置了正数 `max_rows` 时读取额外一行判定截断，未配置时不注入外层行数限制。单项行数据仍限制64000字节，达到体积限制也会标记截断。每次审核/归档最多执行20项查询，查询阶段总超时30秒；总模型数据约128000字节，超限项作为失败证据处理。
- 查询库允许管理员读取租户 OA 连接权限覆盖的数据。表范围用于限制每个模板的取数范围，不构成 OA 数据库本身的行级权限策略。

## 预览与规则引用

手工预览：

```json
{
  "query_id": "查询库UUID",
  "definition": {"name":"...","description":"...","sql":"...","dialect":"mysql","tables":[],"parameters":[],"max_rows":20},
  "parameters": {"employee_id":"42","current_request_id":"1001","year_start":"2026-01-01"}
}
```

规则 `data_queries` 是数组；与 `context_mounts` 独立：

```json
[{
  "query_id": "查询库UUID",
  "version_no": 1,
  "enabled": true,
  "bindings": {
    "employee_id": {"source":"field","field":"main:employee"},
    "current_request_id": {"source":"system","value":"process_id"},
    "year_start": {"source":"system","value":"year_start","days_offset":0}
  }
}]
```

规则里的“从记忆库选择”打开独立选择弹窗，只列出当前租户未删除且已发布的条目，可搜索并选择具体发布版本；显示说明、取值来源及只读 SQL，没有创建、保存、发布或 AI 修改入口。确认引用后显示当前规则的字段下拉绑定；已有引用的版本升级由用户明确选择。

参数可携带 `default_binding`，例如 `{"name":"filter_1","label":"费用类型","type":"string","default_binding":{"source":"constant","value":"招待费"}}`。固定值和系统值随发布版本保存；规则的显式 `bindings` 优先，未覆盖时由服务端沿用默认取值。记忆库检查允许 `source=field` 未选择实际字段，规则保存及实际执行仍必须完成字段绑定。手工预览中省略某项时采用固定/系统默认值；显式测试值覆盖默认值且不重复应用日期偏移；流程字段及当前流程编号需填写测试值。默认来源及常数类型由后端校验，日期遵循应用时区。

来源为 `field`（`main:字段` / `明细表名:字段`）、`system`、`constant`。系统变量：`process_id`、`current_date`、`year_start`、`month_start`。日期偏移为±3660天，业务日期遵循应用时区。明细表多行值以列表参数接收。

规则预览传 `{query_id,version_no,process_id,bindings}`；服务端只读取该租户已发布版本，拉取真实流程原始字段后绑定。每条规则最多8项引用；跨租户或缺失版本拒绝保存。配置版本和实际执行快照保留查询引用，规则导出保留引用但**不携带查询库版本实体**，跨租户导入需先重建查询库并重新绑定。

## 给审核 AI 的数据与证据

预览与实际执行均返回/存储：`rule_id`、`rule_content`（实际审核时）、`query_id`、`version_no`、`name`、`description`、`parameters`、`queried_at`、`status`、`error`、`result`、`model_text`。`result` 含中文列元数据、`col_1` 等内部键的行数据、`truncated`、`duration_ms`。

`model_text` 包含用途、版本、时间、成功/失败、行数和截断状态，再以中文列名的 JSON 展示数据，明确标注业务数据不能作为指令。实际执行按规则分组，加入已有推理阶段的外部上下文；模板没有 `{{external_context}}` 时也会追加。失败、空结果、截断明确区分。

审核/归档日志的 `query_evidence` 保存当次证据，详情 API 和工作台可查看，遵循原结果访问控制。失败或截断不允许保持“通过/完全合规”建议，强制转人工复核；成功返回空行不自动代表业务合规。

## AI 辅助 SQL 生成

独立页签内维护 AI 补充说明、表与字段业务解释及提示词配置。表结构按表分组展示，支持字段搜索、类型查看和复制完整字段名；不再常驻右上角提示词按钮。

请求 `{action:"generate|explain|repair",category,instruction,error,definition,preview?}`。`preview` 仅含 `{status,error,row_count,truncated,columns}` 诊断摘要，不发送预览业务记录或手工测试值。生成前须选择实际表，配置查询尚未生成 SQL 时也可使用已经选择的主表和筛选设置。

`definition.ai_context` 随草稿及版本保存：`oa_module` 为 `general|workflow|model`，`notes` 最多8000字节，`field_notes` 为 `表名.字段名 → 业务说明`，最多200项、每项200字符，总 JSON 最多30000字节。它们补充 SQL 助手上下文，不替代审核规则的业务判断。

服务端重新读取租户 OA 类型、数据库方言、应用时区和当前业务日期，以及所有声明表的真实结构。发送内容积累当前名称、分类、用途、SQL、输出字段及统计方式、筛选与取值来源、排序、行数限制、字段说明、业务补充、检查报告及预览诊断。生成上下文总量最多128000字节，预览摘要最多12000字节。可在页签中查看当前准备的内容，实际物理结构由服务端重新获取。泛微归档状态仅在业务明确要求且存在真实字段时使用，requestid 不作为归档依据；不臆造表或关联。

返回 `{mode:"sql",tables,sql,explanation,parameters,warnings,check}`。AI 仅生成自定义 SQL 候选，不生成或修改可视化 builder，不自动执行。采用候选时进入自定义 SQL，清除 builder；从配置模式采用时先确认。同名同类型变量在 AI 未返回默认取值时沿用现有配置。候选仍须检查、预览再发布，解释模式不提供采用按钮。

租户可在 AI 页签分别修改生成、解释、修复三种业务提示词；服务端保留固定方言、只读、表范围、值绑定、中文别名和 JSON 输出约束。AI 调用经 `AIModelCallerService.ChatWithFallback`，使用租户主/备用模型及 Token 配额，记录 `request_type=sql_assist`、`call_type=structured`。数据管理页可按 SQL 辅助筛选，并单独统计；仪表盘总体 Token 和费用沿用统一调用日志汇总，业务审核次数不计入 SQL 辅助。

## 升级

执行带中文注释的迁移 `000083_audit_data_query_library.up.sql`，重启后端并部署前端。迁移不会修改生产 OA 表或创建预设业务 SQL。回滚会删除本功能的查询定义、版本、提示词和证据数据。

规则查询证据可包含 `applicability_condition`，记录当次规则的适用前提。查询失败不能当作前提不满足；若其他证据已明确证明前提不满足，规则按直接通过处理，查询失败兜底不再改判此规则。


提示词接口新增只读 `system_constraints`，展示 SQL 辅助实际使用的固定系统约束；PUT 时服务端始终覆盖该值，不能修改固定约束。生成、解释、修复三种租户业务提示词仍按现有权限编辑。辅助调用使用每次独立的 `business_log_id` 归集，在 AI 调用记录中展示结构化 SQL、待确认事项与完整提示词，详见 [AI 调用记录](./llm-logs.md)。
