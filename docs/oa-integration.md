# OA 系统对接说明

## 概述

OA 智审通过**适配器模式**对接企业 OA 系统，从 OA 数据库中直接读取流程表单数据和审批流信息，为 AI 审核提供数据源。系统采用只读方式连接 OA 数据库，不会对 OA 系统产生任何写入操作。

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

### 工厂模式

`NewOAAdapter` 工厂函数（`go-service/internal/pkg/oa/factory.go`）根据 `oa_type` 和数据库驱动类型创建对应的适配器实例：

```go
func NewOAAdapter(oaType string, conn *model.OADatabaseConnection) (OAAdapter, error)
```

### 数据库连接管理

OA 数据库连接配置存储在 `oa_database_connections` 表中，支持：

- 连接参数加密存储（密码使用 AES-256 加密）
- 连接池管理（可配置最大连接数、超时时间）
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
| `workflow_requestbase` | 流程实例（请求 ID、创建人、状态） |
| `workflow_currentoperator` | 当前审批人（待办列表数据源） |
| `workflow_nodebase` | 审批节点定义 |
| `hrmresource` | 人员信息（姓名、登录 ID） |
| `hrmdepartment` | 部门信息 |
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

4. FetchTodoListPaged(username, filter)
   └─ hrmresource(loginid→id) → workflow_currentoperator
   └─ JOIN requestbase + base + type + bill + node
   └─ 支持 keyword/applicant/department/processType 筛选下推

5. FetchProcessFlow(processID)
   └─ 拉取审批流节点快照（节点名、审批人、操作、意见）
```

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
4. 在 `NewOAAdapter` 的 `switch` 分支中添加创建逻辑
5. 如需新的数据库驱动，在 `go-service/internal/pkg/oa/` 下创建驱动子目录

**接口实现要点**：

- `FetchTodoListPaged` 和 `FetchArchivedListPaged` 必须实现 SQL 级分页（COUNT + LIMIT/OFFSET），避免全量拉取
- `FetchProcessData` 需正确处理主表和明细表的关联关系
- `FetchProcessFlow` 返回的审批流快照需包含 `HistoryText` 和 `GraphText`，供 AI 提示词使用
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
