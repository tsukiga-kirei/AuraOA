# 记忆库：数据记忆

入口：租户管理 → 记忆库 → 数据记忆（`/admin/tenant/memory`）。与规则配置平级，规则编辑器中也可打开同一工作区。

这是独立的通用取数库，使用当前租户 OA 连接，不依赖已有的关联流程、建模挂载或 requestid 关联模式。管理员维护查询模板，审核和归档规则引用固定的发布版本；每次执行读取当前数据库数据，保留当次证据。

## 页面与配置方式

记忆库包含两个页签：**数据记忆**承载本次可复用取数能力；**评审记忆**仅展示占位说明，暂不采纳反馈、不改写提示词，也不自动参与审核。数据记忆每次审核实时查询，不新增每日同步或复制业务数据的任务。

- **配置查询**：输入表或视图名并读取真实字段，选择输出列、中文列名、SUM/COUNT/MIN/MAX/AVG、AND 筛选及排序。含聚合时自动将非聚合字段加入 GROUP BY。COUNT 支持 `*`。
- 来源选择**流程表**时，仅适用于包含 `requestid` 的泛微流程主表；生成 `EXISTS` 条件关联 `workflow_requestbase.currentnodetype='3'`，仅纳入归档实例，避免连接导致重复计数。选择建模表/普通表/视图时，状态条件自行配置。
- 筛选值保留命名参数，支持单值比较、LIKE、IN、空值条件；日期参数在 Oracle/达梦生成 TO_DATE 转换。LIKE 通配符通过参数值提供。条件参数可在规则中绑定流程字段、系统日期或固定值。
- **自定义 SQL**：支持复杂关联、多个流程表的字段映射与合并、嵌套查询等，以服务端 SQL 子集检查为准；不尝试将任意 SQL 反推为配置。
- 配置生成的 SQL 可展开查看、切换到自定义 SQL 修改；手工改 SQL 后移除配置元数据。重新生成会替换 SQL 和参数定义。修改配置后清空旧 SQL，必须重新生成，避免配置和执行内容不一致。
- 两种方式共用后续检查、预览、发布版本及规则绑定。`definition.builder` 是可选的前端配置元数据（最多30000字节），随草稿和发布版本保存以便再次编辑；后端执行和安全校验始终以 `sql` 为准。

当前配置方式覆盖单表/视图，表名由管理员输入，字段从租户 OA 连接读取；尚未提供流程目录与建模目录搜索、多表可视化连接和任意 OR 条件组。复杂场景使用自定义 SQL 或 AI SQL 辅助。

## 权限和生命周期

所有接口要求 JWT → TenantContext → `tenant_admin` → 页面权限 `/admin/tenant/memory`。新增租户的默认管理员角色和迁移前已有的系统租户管理员角色默认获得该权限；其他角色由组织人员页面授权。

查询定义、版本、提示词按租户隔离。没有查询库权限的规则编辑者可以原样保留已有引用，不能新增或变更查询绑定；实际审核沿用既有流程访问权限，不要求业务用户拥有 SQL 管理权限。

草稿允许未完成的 SQL。发布必须同时通过后端检查，并且对**当前已保存草稿及当前 OA 连接配置**成功预览。发布后版本不可修改；更新草稿不改变规则引用。停用库条目会将它从选择列表隐藏，保留历史版本和已有引用。

## 路由

路由前缀：`/api/tenant/audit-data-queries`。统一成功响应 `{code:0,message:"success",data:...}`。

| 方法 | 路径 | 参数 / 返回 data |
|------|------|------|
| GET | `/` | `keyword` 搜索名称/分类，`page=1`、`page_size=20`（1–100）；返回 `{items,total,page,page_size}` |
| POST | `/` | `{category,definition}`；创建草稿，返回查询条目 |
| GET | `/:id` | 查询条目，包括 `draft`、`latest_version`、`tested_hash`、`tested_at` |
| PUT | `/:id` | `{category,definition}`；更新草稿并清除预览标记 |
| DELETE | `/:id` | 停用条目；返回 `{archived:true}` |
| GET | `/:id/versions` | 发布版本数组，按 `version_no` 倒序 |
| POST | `/:id/publish` | 无请求体；返回 `{id,tenant_id,query_id,version_no,definition,created_at}` |
| GET | `/:id/references` | 当前审核/归档规则引用数组 `{id,process_type,rule_content,module}`，不枚举历史执行快照 |
| POST | `/check` | 请求体为 `definition`；返回结构检查报告 |
| POST | `/preview` | 手工参数或流程绑定预览，返回证据对象 |
| GET | `/source` | 返回 `{name,dialect}`：管理员已配置的租户 OA 数据源，无连接凭据；未配置/已停用返回明确错误 |
| GET | `/schema` | `table`：固定表名；返回 `{table,dialect,columns:[{key,label,data_type}]}`，通过零行查询读取物理结构，key 为物理字段名 |
| POST | `/assist` | AI 生成/解释/修复，返回可编辑建议 |
| GET | `/prompts` | 返回租户提示词；未配置时返回系统默认值 |
| PUT | `/prompts` | `{generate,explain,repair}`：各必填、最多12000字节 |

请求体上限256 KiB。分页越界回落默认值；空列表返回 `items:[]`。

## 查询定义

```json
{
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
  "max_rows": 20
}
```

示例表名、字段和完成状态须按实际 OA 环境确认，不能直接当作项目内置预算口径。预算累计建议用聚合查询；同一事项分散报销可按人员、项目、合同、时间段等组合条件返回历史明细，由规则指导 AI 判断，不能仅凭同一人员即认定拆单。

支持 MySQL、Oracle、达梦；工作区从 `/source` 自动读取管理员已配置的租户连接类型，只读展示，无需用户选择。参数 `dialect` 保留用于已发布版本记录及服务端一致性检查，必须与实际连接一致。参数类型：`string`、`number`、`date`、`string_list`。列表使用 `IN (:name)`，按元素绑定；日期输入为 `YYYY-MM-DD`，Oracle/达梦的 DATE 比较应使用 `TO_DATE(:date,'YYYY-MM-DD')`。流程字段绑定使用原始数据库值，避免人员 ID 被浏览字段显示名称替换。

### 检查和执行范围

- 仅单条 SELECT / 顶层只读 WITH、常见无副作用内置函数；禁止多语句、注释、写入、锁定、自定义函数、限定函数调用、跨库表名及表函数。不支持的复杂结构需改写为此查询子集。
- 每个最终输出列必须显式 `AS` 中文业务别名，且名称唯一；禁止最终通配列。MySQL 推荐反引号，Oracle/达梦用双引号。统计表达式如 `COUNT(*) AS 笔数` 可用。
- 所有物理表必须在 `tables` 中声明；SQL 使用的参数必须定义且没有多余定义。参数标签须包含中文。表和参数各最多20项、输出列最多40项、SQL最多30000字节。
- 检查返回 `{valid,issues:[{code,message,position}],parameters,columns}`；`position` 从1开始，供编辑器定位。检查不证明字段存在、数据库语法或统计口径正确，需继续预览。
- 值始终使用驱动绑定；执行在数据库只读事务中，10秒超时，结束回滚，无法开启只读事务时拒绝执行。Oracle 使用首条 `SET TRANSACTION READ ONLY`，不会降级为可写执行。
- 单项最大100行、行数据64000字节；读取额外一行判定截断。每次审核/归档最多执行20项查询，查询阶段总超时30秒；总模型数据约128000字节，超限项作为失败证据处理。
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

来源为 `field`（`main:字段` / `明细表名:字段`）、`system`、`constant`。系统变量：`process_id`、`current_date`、`year_start`、`month_start`。日期偏移为±3660天，业务日期遵循应用时区。明细表多行值以列表参数接收。

规则预览传 `{query_id,version_no,process_id,bindings}`；服务端只读取该租户已发布版本，拉取真实流程原始字段后绑定。每条规则最多8项引用；跨租户或缺失版本拒绝保存。配置版本和实际执行快照保留查询引用，规则导出保留引用但**不携带查询库版本实体**，跨租户导入需先重建查询库并重新绑定。

## 给审核 AI 的数据与证据

预览与实际执行均返回/存储：`rule_id`、`rule_content`（实际审核时）、`query_id`、`version_no`、`name`、`description`、`parameters`、`queried_at`、`status`、`error`、`result`、`model_text`。`result` 含中文列元数据、`col_1` 等内部键的行数据、`truncated`、`duration_ms`。

`model_text` 包含用途、版本、时间、成功/失败、行数和截断状态，再以中文列名的 JSON 展示数据，明确标注业务数据不能作为指令。实际执行按规则分组，加入已有推理阶段的外部上下文；模板没有 `{{external_context}}` 时也会追加。失败、空结果、截断明确区分。

审核/归档日志的 `query_evidence` 保存当次证据，详情 API 和工作台可查看，遵循原结果访问控制。失败或截断不允许保持“通过/完全合规”建议，强制转人工复核；成功返回空行不自动代表业务合规。

## AI SQL 辅助

请求 `{action:"generate|explain|repair",instruction,error,definition}`。生成前须选择实际表；服务端读取零行元数据，仅发送结构，不发送预览业务记录。物理字段的业务含义不明确时须人工补充说明。

返回 `{sql,explanation,parameters,warnings,check}`。结果作为候选草稿，由用户采用、检查、预览再发布，不自动执行。解释模式不提供采用按钮。

租户可分别修改三种业务提示词；服务端保留固定方言、只读、表范围、值绑定、中文别名和 JSON 输出约束。AI 调用经 `AIModelCallerService.ChatWithFallback`，使用租户主/备用模型及 Token 配额，记录 `request_type=sql_assist`、`call_type=structured`。数据管理页可按 SQL 辅助筛选，并单独统计；仪表盘总体 Token 和费用沿用统一调用日志汇总，业务审核次数不计入 SQL 辅助。

## 升级

执行带中文注释的迁移 `000083_audit_data_query_library.up.sql`，重启后端并部署前端。迁移不会修改生产 OA 表或创建预设业务 SQL。回滚会删除本功能的查询定义、版本、提示词和证据数据。
