# OA 系统对接说明

## 概述

AuraOA 通过**适配器模式**对接企业 OA 系统，从 OA 数据库中直接读取流程表单数据和审批流信息，为 AI 审核提供数据源。系统采用只读方式连接 OA 数据库，不会对 OA 系统产生任何写入操作。

## 架构设计

### 适配器接口

所有 OA 适配器均实现统一的 `OAAdapter` 接口（定义于 `go-service/internal/pkg/oa/adapter.go`），核心方法包括：

| 方法 | 说明 |
|------|------|
| `ValidateProcess` | 验证流程类型是否存在于 OA 系统中 |
| `FetchFields` | 拉取指定流程的全部字段定义（主表 + 明细表） |
| `CheckUserPermission` | 检查用户在 OA 中是否具有指定流程的审批权限 |
| `FetchProcessData` | 拉取指定流程实例的业务数据（主表 + 明细表） |
| `FetchTodoList` / `FetchTodoListPaged` | 拉取用户待审批流程列表（支持分页和筛选下推） |
| `FetchArchivedList` / `FetchArchivedListPaged` | 拉取已归档流程列表（支持分页和筛选下推） |
| `FetchProcessFlow` | 拉取流程审批流快照（审批节点、操作人、意见） |
| `FetchAllTodoItems` | 拉取全量待办（供定时任务批处理使用） |
| `IsProcessInTodo` | 判断指定流程是否仍在用户待办中 |

流程级嵌入刷新还定义了可选接口 `RecentProcessScanner`。支持该接口的适配器可以通过
`FetchRecentProcessSummaries` 按流程类型和业务时区起始时间，限量拉取最近创建的流程实例，
供审核、总结配置中的定时检查使用；不支持该接口的 OA 适配器会跳过定时扫描，不影响事件触发和可见嵌入页。

### 适配器与连接池管理

`ConnectionManager`（`go-service/internal/pkg/oa/connection_manager.go`）根据 `oa_type`
和数据库驱动创建适配器，并按 `oa_database_connections.id` 在进程内复用底层
`*sql.DB` 连接池。适配器对象保持轻量，可按调用注入附件识别服务；MySQL、Oracle
和达梦共用同一套连接池生命周期。

连接管理规则：

- 同一个 OA 连接配置在单个 `go-service` 进程内只保留一个共享连接池。
- 并发首次访问会合并建池操作，避免同一配置同时创建多个连接池。
- 每次取用都会比较连接配置指纹；主机、端口、数据库、账号、密码或连接池参数变化时，
  自动关闭旧池并建立新池。
- 系统管理端更新或删除 OA 连接配置后，会立即使对应共享池失效。
- `go-service` 优雅退出时统一关闭全部 OA 共享连接池。
- “测试连接”使用不进入共享缓存的短生命周期连接，测试结束后立即关闭。
- 建池前先设置最大连接数、最大空闲连接数、空闲时间和生命周期，再按
  `connection_timeout` 执行显式 Ping。

### 数据库连接管理

OA 数据库连接配置存储在 `oa_database_connections` 表中，支持：

- 连接参数加密存储（密码使用 AES-256 加密）
- 共享连接池管理（可配置最大连接数、连接超时；固定设置空闲回收和连接生命周期）
- 连接状态检测（`connected` / `disconnected`）
- 保存前/保存后均可测试连通性

每个租户通过 `tenants.oa_db_connection_id` 外键关联一个 OA 数据库连接。

## 已适配 OA 系统

### 泛微 Ecology E9（`weaver_e9`）

**实现文件**：`go-service/internal/pkg/oa/ecology9.go`

**支持的数据库驱动**：

| 驱动 | 说明 | 默认端口 |
|------|------|---------|
| `mysql` | MySQL | 3306 |
| `oracle` | Oracle | 1521 |
| `dm` | 达梦 DM | 5236 |

**核心表映射**：

| 泛微 E9 表 | 用途 |
|------------|------|
| `workflow_base` | 流程定义（流程名称、表单 ID、流程类型） |
| `workflow_type` | 流程类型分类（类型名称） |
| `workflow_bill` | 表单定义（主表名） |
| `workflow_billfield` + `htmllabelinfo` | 字段定义（字段名、字段类型、所属表） |
| `workflow_selectitem` | 选择框 / 下拉框选项定义 |
| `workflow_browserurl` | 浏览按钮定义，含内置 / 系统 / 自定义浏览按钮的关联表与显示列 |
| `workflow_requestbase` | 流程实例（请求 ID、创建人、状态） |
| `workflow_currentoperator` | 当前审批人（待办列表数据源） |
| `workflow_nodebase` | 审批节点定义 |
| `hrmresource` | 人员信息（姓名、登录 ID） |
| `hrmdepartment` | 部门信息 |
| `hrmsubcompany` | 分部信息 |
| `formtable_main_*` | 流程主表数据 |
| `formtable_main_*_dt*` | 流程明细表数据 |

**数据库兼容性处理**：

- Oracle/DM 使用大写标识符，MySQL 不区分大小写 — 通过 `tableName()` / `col()` 方法统一处理
- Oracle 使用 `OFFSET ... ROWS FETCH NEXT ... ROWS ONLY` 分页语法，MySQL/DM 使用 `LIMIT ... OFFSET`
- 字段值读取使用 `mapGet()` / `mapGetInt()` 辅助函数，不区分大小写匹配 key

**数据提取流程**：

```
1. ValidateProcess(processType)
   └─ workflow_base → workflow_type → workflow_bill
   └─ 返回流程名称、主表名、流程类型标签

2. FetchFields(processType)
   └─ workflow_base → workflow_billfield + htmllabelinfo
   └─ 返回主表字段 + 明细表字段定义

3. FetchProcessData(processID)
   └─ workflow_requestbase → workflow_base → workflow_bill
   └─ 查询主表 formtable_main_* 数据
   └─ 查询明细表 formtable_main_*_dt1, dt2, ... 数据
   └─ 增补字段中文名、浏览按钮显示值、选择框显示值

4. FetchTodoListPaged(username, filter)
   └─ hrmresource(loginid→id) → workflow_currentoperator
   └─ JOIN requestbase + base + type + bill + node
   └─ 支持 keyword/applicant/department/processType 筛选下推

5. FetchProcessFlow(processID)
   └─ 拉取审批流节点快照（节点名、审批人、操作、意见）

6. FetchRecentProcessSummaries(processType, since, limit)
   └─ workflow_requestbase → workflow_base → hrmresource → hrmdepartment
   └─ 按流程名称和 createdate 下界筛选，按创建时间倒序限量返回
   └─ 仅用于安排指纹检查；是否调用 AI 仍由审核/总结上下文判断
```

**字段值翻译规则（供 AI prompt 使用）**：

AuraOA 从 OA 物理表读取到的值通常是数据库存储值，例如人员 ID、流程 requestid、下拉框枚举值。进入 AI prompt 前，Ecology9 适配器会尽量把这些值翻译成业务可读文本；解析失败时保留原始值，不阻断审核。

| 字段类型 | 泛微标识 | 数据源 | AuraOA 行为 |
|----------|----------|--------|-------------|
| 字段中文名 | `workflow_billfield.fieldlabel` | `htmllabelinfo.indexid`，`languageid=7` | prompt 中使用中文字段名，避免暴露数据库列名 |
| 浏览按钮 | `fieldhtmltype=3` | `workflow_browserurl` 或内置兜底映射 | 将 ID 增补为显示名，最终 prompt 只展示显示文本 |
| 选择框 / 下拉框 | `fieldhtmltype=5` | `workflow_selectitem` | 用 `fieldid + selectvalue` 找 `selectname`，再解析泛微多语言串 |
| 附件 | `fieldhtmltype=6` | OA 附件接口 | 取附件正文并拼入 `{{attachments}}` |

浏览按钮通用解析优先级：

1. 先按 `workflow_billfield.type = workflow_browserurl.id` 查浏览按钮定义。
2. 若 `workflow_browserurl.TABLENAME`、`COLUMNAME`、`KEYCOLUMNAME` 都不为空，则按通用方式查询显示值：`WHERE KEYCOLUMNAME IN (...)`，展示 `COLUMNAME`。
3. `COLUMNAME` 是最终展示内容，可以是普通列名（如 `lastname`、`departmentname`、`requestname`），也可能是表达式（如成本中心编码 + 名称）。
4. 多选字段通常在表单物理列中用英文逗号拼接多个值，AuraOA 会拆分后批量查询，再用 `, ` 拼接显示名。
5. 若通用元数据不完整，再走少量内置兜底映射；自定义 `browser.xxx` 会继续按建模浏览框规则解析。

常用内置兜底映射：

| TYPE | 名称 | 关联表 | ID 字段 | 显示字段 | 备注 |
|------|------|--------|---------|----------|------|
| `1` | 人力资源 | `hrmresource` | `id` | `lastname` | 单选 |
| `17` | 多人力资源 | `hrmresource` | `id` | `lastname` | 多选，逗号拆分 |
| `4` | 部门 | `hrmdepartment` | `id` | `departmentname` | 单选 |
| `57` | 多部门 | `hrmdepartment` | `id` | `departmentname` | 多选 |
| `164` | 分部 | `hrmsubcompany` | `id` | `subcompanyname` | 单选 |
| `194` | 多分部 | `hrmsubcompany` | `id` | `subcompanyname` | 多选 |
| `16` | 流程 | `workflow_requestbase` | `requestid` | `requestname` | 单选 |
| `152` | 多流程 | `workflow_requestbase` | `requestid` | `requestname` | 多选 |
| `171` | 归档流程 | `workflow_requestbase` | `requestid` | `requestname` | 单选 |
| `165` / `166` | 分权人力资源 | `hrmresource` | `id` | `lastname` | 单选 / 多选 |
| `167` / `168` | 分权部门 | `hrmdepartment` | `id` | `departmentname` | 单选 / 多选 |
| `169` / `170` | 分权分部 | `hrmsubcompany` | `id` | `subcompanyname` | 单选 / 多选 |

> 不同 E9 版本或客户环境的 `TYPE` 含义可能不同。以客户环境中的浏览按钮配置为准，日期、时间、年份等系统字段本身就是可读值，不应按 ID 查询关联表。

自定义 / 集成浏览按钮处理：

| TYPE / FIELDDBTYPE | 处理方式 |
|--------------------|----------|
| `TYPE=161` | 自定义单选，通常 `workflow_browserurl` 只是通用入口；优先用 `FIELDDBTYPE=browser.xxx` 到 `mode_browser.SHOWNAME=xxx` 找建模浏览框 |
| `TYPE=162` | 自定义多选，同上，表单值按逗号拆分多个 ID |
| `TYPE=226` | 系统集成单选浏览按钮，按 `workflow_browserurl` 动态解析 |
| `TYPE=256` | 自定义树形单选，按 `workflow_browserurl` 动态解析 |
| `TYPE=257` | 自定义树形多选，按 `workflow_browserurl` 动态解析并按多选处理 |
| `FIELDDBTYPE=browser.xxx` | 即使 `TYPE` 不在上述列表，也会按自定义浏览框尝试解析 |

建模浏览框 `mode_browser` 处理：

`FIELDDBTYPE=browser.keshangfenlei` 会去掉前缀得到 `keshangfenlei`，再查询 `mode_browser.SHOWNAME='keshangfenlei'`。优先解析 `SQLTEXT`，例如 `select id,dabm,dabm from uf_keshangfenlei`，取第一列作为存储值列、第二列作为显示列；若只有 `SEARCHBYID` / `SQLTEXT1`，则从 `where id=?` 中识别存储值列，从 `select` 第一列识别显示列。

选择框 / 下拉框处理：

`workflow_selectitem.FIELDID` 对应 `workflow_billfield.ID`，同一个 `FIELDID` 会有多行，每行是一项选项；真正定位选项的是 `FIELDID + SELECTVALUE`。`SELECTNAME` 在部分 E9 环境中是多语言串，例如 `~\`~7 新客站~\`~8 New passenger station~\`~9 新客站~\`~`，AuraOA 优先取语言 `7`，其次 `9`、`8`，普通文本如 `EMS` 则原样使用。

## 未完成的 OA 适配

以下 OA 系统已在数据库选项表中注册，但尚未实现适配器代码：

| OA 类型 | 编码 | 状态 | 说明 |
|---------|------|------|------|
| 泛微 E-Bridge | `weaver_ebridge` | ❌ 未实现 | 泛微轻量级 OA，表结构与 E9 不同 |
| 致远 A8+ | `zhiyuan_a8` | ❌ 未实现 | 致远协同 OA，需适配其流程引擎表结构 |
| 蓝凌 EKP | `landray_ekp` | ❌ 未实现 | 蓝凌知识管理平台，需适配 EKP 流程表 |
| 自定义 OA | `custom` | ❌ 未实现 | 通用适配器，需用户自行配置表映射关系 |

### 新增 OA 适配器开发指南

1. 在 `go-service/internal/pkg/oa/` 下创建新适配器文件（如 `zhiyuan_a8.go`）
2. 实现 `OAAdapter` 接口的所有方法
3. 在 `factory.go` 的 `supportedDrivers` 中注册支持的数据库驱动
4. 在 `newOAAdapterWithDB` 的分支中添加轻量适配器创建逻辑
5. 如需新的数据库驱动，在 `go-service/internal/pkg/oa/` 下创建驱动子目录
6. 将底层连接创建接入 `ConnectionManager`，禁止在业务 Service 中直接新建连接池
7. 如需支持流程级定时检查，实现可选接口 `RecentProcessScanner`

**接口实现要点**：

- `FetchTodoListPaged` 和 `FetchArchivedListPaged` 必须实现 SQL 级分页（COUNT + LIMIT/OFFSET），避免全量拉取
- `FetchProcessData` 需正确处理主表和明细表的关联关系
- `FetchProcessFlow` 返回的审批流快照需包含 `HistoryText` 和 `GraphText`，供 AI 提示词使用
- `FetchRecentProcessSummaries` 必须按流程类型和时间下界在 SQL 中过滤，并强制限制返回条数
- 所有数据库查询应使用参数化查询，防止 SQL 注入
- 建议使用 `context.Context` 传递超时控制

## 数据流向

```
┌──────────────┐     只读连接      ┌──────────────┐
│   OA 数据库   │ ◄──────────────── │  OA 适配器    │
│  (MySQL/      │                   │ (Ecology9     │
│   Oracle/DM)  │                   │  Adapter)     │
└──────────────┘                   └──────┬───────┘
                                          │
                                          ▼
                                   ┌──────────────┐
                                   │  审核服务      │
                                   │ (AuditExecute │
                                   │  Service)     │
                                   └──────┬───────┘
                                          │
                              ┌───────────┼───────────┐
                              ▼           ▼           ▼
                        字段提取     数据提取     审批流提取
                        FetchFields  FetchData   FetchFlow
                              │           │           │
                              ▼           ▼           ▼
                        ┌─────────────────────────────────┐
                        │       提示词构建 & AI 审核        │
                        └─────────────────────────────────┘
```

## 配置说明

### 环境变量

OA 数据库连接通过系统管理后台配置，不依赖环境变量。连接参数存储在 PostgreSQL 的 `oa_database_connections` 表中。

### 管理后台配置路径

1. 系统管理员登录 → 系统设置 → OA 数据库连接
2. 新建连接：选择 OA 类型、数据库驱动、填写连接参数
3. 测试连接：验证数据库连通性
4. 关联租户：在租户管理中将 OA 连接分配给租户

### 嵌入 AI 审核（iframe）

若需在泛微 E9 审批页侧边展示 AI 审核结论，见 [OA 嵌入 AI 审核侧边栏](./oa-configurations/02-embed-audit-sidebar.md)（AuraOA 开关 + E9 iframe / postMessage 配置）。
