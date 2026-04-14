# 技术设计文档：代码注释规范化与日志系统完善

## 概述

本设计文档覆盖两个方向：

1. **代码注释规范化**：删除现有英文注释和机器翻译腔注释，统一替换为地道中文注释，覆盖后端 Go 代码和前端 Vue/TypeScript 代码。

2. **日志系统完善**：在现有 `go.uber.org/zap` 基础上，新建 `internal/pkg/logger/` 包，集成 `lumberjack` 实现文件轮转，支持全局日志和租户隔离日志，并通过定时任务实现超期日志自动清理。

---

## 架构

### 整体架构图

```mermaid
graph TD
    A[main.go] -->|初始化| B[logger.Init]
    B --> C[GlobalLogger\nlogs/app.log]
    B --> D[TenantLogger 缓存\nlogs/tenants/{code}/tenant.log]

    E[HTTP 中间件] -->|请求日志| C
    F[Panic 恢复中间件] -->|错误日志| C
    G[业务 Service] -->|租户操作日志| D
    G -->|系统日志| C

    H[CronScheduler] -->|每日凌晨| I[日志清理任务]
    I -->|读取| J[system_configs\ntenant.default_log_retention_days\nsystem.global_log_retention_days]
    I -->|删除超期文件| C
    I -->|删除超期文件| D

    K[config.yaml\nlog 配置节] -->|启动时加载| B
    J -->|运行时读取| I
```

### 日志文件目录结构

```
logs/
├── app.log                          # 全局日志（当前写入文件）
├── app-2025-01-15T02-00-00.000.log  # 轮转备份（lumberjack 自动命名）
└── tenants/
    ├── tenant_a/
    │   ├── tenant.log               # 租户 A 当前日志
    │   └── tenant-2025-01-15T02-00-00.000.log
    └── tenant_b/
        └── tenant.log
```

---

## 组件与接口

### 1. `internal/pkg/logger/` 包

这是本次改造的核心新增包，封装所有日志初始化和获取逻辑。

#### 文件结构

```
internal/pkg/logger/
├── logger.go       # 包入口：Init、Global、GetTenantLogger
├── config.go       # LogConfig 结构体定义
└── cleanup.go      # 日志清理逻辑（供 cron 任务调用）
```

#### `logger.go` 核心接口

```go
// Init 根据配置初始化全局 logger，必须在 main.go 最早调用。
func Init(cfg LogConfig) error

// Global 返回全局 *zap.Logger 实例（写入 logs/app.log + stdout）。
func Global() *zap.Logger

// GetTenantLogger 返回指定租户的 *zap.Logger 实例。
// 内部维护 sync.Map 缓存，相同 tenantCode 复用同一实例。
// 租户 logger 同时写入租户专属文件和全局文件。
func GetTenantLogger(tenantCode string) *zap.Logger

// Sync 刷新所有 logger 缓冲区，在程序退出前调用。
func Sync()
```

#### `config.go` 配置结构

```go
// LogConfig 日志系统配置，从 config.yaml 的 log 节读取。
type LogConfig struct {
    Level              string // 日志等级：debug/info/warn/error，默认 info
    Dir                string // 日志根目录，默认 logs
    MaxSizeMB          int    // 单文件最大体积（MB），默认 100
    MaxBackups         int    // 最大保留备份数，默认 5
    Compress           bool   // 是否 gzip 压缩备份，默认 true
    GlobalRetentionDays int   // 全局日志保留天数，默认 30（config.yaml 兜底）
}
```

#### `cleanup.go` 清理接口

```go
// CleanupGlobalLogs 清理 logs/ 目录下超过 retentionDays 天的备份文件。
// 仅清理轮转备份（带时间戳的文件），不删除当前写入的 app.log。
func CleanupGlobalLogs(retentionDays int) (deletedCount int, freedBytes int64, err error)

// CleanupTenantLogs 清理 logs/tenants/{tenantCode}/ 目录下超期备份文件。
// retentionMap 为 tenantCode → retentionDays 的映射。
func CleanupTenantLogs(retentionMap map[string]int) (deletedCount int, freedBytes int64, err error)
```

### 2. `config.go` 扩展

在 `Config` 结构体中新增 `Log LogConfig` 字段，对应 `config.yaml` 的 `log` 节。

### 3. `config.yaml` 扩展

新增 `log` 配置节：

```yaml
log:
  level: "info"           # 日志等级：debug/info/warn/error
  dir: "logs"             # 日志根目录
  max_size_mb: 100        # 单文件最大体积（MB）
  max_backups: 5          # 最大保留备份数
  compress: true          # 是否 gzip 压缩备份
  global_retention_days: 30  # 全局日志保留天数（兜底值，优先读 system_configs）
```

### 4. 日志清理 Cron 任务

在 `internal/service/` 中新增 `log_cleanup_service.go`，实现日志清理逻辑，由 `CronScheduler` 在每日凌晨调度。

```go
// LogCleanupService 日志清理服务，依赖 system_configs 读取保留天数。
type LogCleanupService struct {
    systemConfigRepo *repository.SystemConfigRepo
    tenantRepo       *repository.TenantRepo
}

// RunCleanup 执行一次完整的日志清理：
// 1. 从 system_configs 读取 system.global_log_retention_days
// 2. 清理全局日志备份
// 3. 遍历所有租户，读取各自 log_retention_days
// 4. 清理各租户日志备份
// 5. 将清理结果写入全局日志
func (s *LogCleanupService) RunCleanup(ctx context.Context) error
```

### 5. 数据库迁移

新增迁移文件 `db/migrations/000030_global_log_retention_config.up.sql`，向 `system_configs` 表插入 `system.global_log_retention_days` 配置项。

### 6. 代码注释改造范围

**原则**：覆盖 `go-service/internal/` 下全部 ~100 个 `.go` 文件，以及 `frontend/` 下全部 ~42 个 `.vue`/`.ts` 文件。每个文件均需：删除英文注释和翻译腔注释，在包声明、结构体、公开函数、关键业务逻辑分支补充地道中文注释。

#### 后端（Go）—— 全量覆盖

**`cmd/`**

| 文件 | 改造内容 |
|------|---------|
| `cmd/server/main.go` | 删除英文步骤注释，补充各初始化阶段中文说明 |

**`internal/config/`**

| 文件 | 改造内容 |
|------|---------|
| `internal/config/config.go` | 删除翻译腔注释，为每个配置结构体字段补充中文说明 |

**`internal/middleware/`**（5 个文件）

| 文件 | 改造内容 |
|------|---------|
| `middleware/auth.go` | 补充 JWT 校验流程中文注释 |
| `middleware/cors.go` | 补充跨域配置中文注释 |
| `middleware/logger.go` | 删除翻译腔注释，改为中文函数说明 |
| `middleware/recovery.go` | 删除翻译腔注释，改为中文 panic 恢复说明 |
| `middleware/role.go` | 补充角色权限校验中文注释 |
| `middleware/tenant.go` | 补充租户上下文注入中文注释 |

**`internal/model/`**（29 个文件）

全部模型文件均需：删除英文注释，为结构体及关键字段补充中文说明，包括：
`ai_deploy_type_option.go`、`ai_model_config.go`、`ai_provider_option.go`、`archive_process_snapshot.go`、`archive_rule.go`、`audit_log.go`、`audit_process_snapshot.go`、`audit_rule.go`、`cron_task_type_preset.go`、`cron_task.go`、`db_driver_option.go`、`department.go`、`job_status.go`、`login_history.go`、`oa_database_connection.go`、`oa_type_option.go`、`org_member.go`、`org_role.go`、`process_archive_config.go`、`process_audit_config.go`、`system_config.go`、`system_prompt_template.go`、`tenant_llm_message_log.go`、`tenant.go`、`user_dashboard_pref.go`、`user_notification.go`、`user_personal_config.go`、`user_role_assignment.go`、`user.go`

**`internal/dto/`**（14 个文件）

全部 DTO 文件均需：删除英文注释，为请求/响应结构体补充中文说明，包括：
`archive_config_dto.go`、`archive_review_dto.go`、`archive_rule_dto.go`、`audit_list_dto.go`、`auth_dto.go`、`cron_dto.go`、`dashboard_overview_dto.go`、`org_dto.go`、`rules_dto.go`、`settings_dto.go`、`system_dto.go`、`tenant_dto.go`、`user_config_admin_dto.go`、`user_notification_dto.go`

**`internal/repository/`**（22 个文件）

全部 Repository 文件均需：删除英文注释，为每个公开方法补充中文说明（查询条件、返回值含义），包括：
`ai_model_repo.go`、`archive_config_repo.go`、`archive_log_repo.go`、`archive_process_snapshot_repo.go`、`audit_log_repo.go`、`audit_process_snapshot_repo.go`、`audit_rule_repo.go`、`base_repo.go`、`cron_log_repo.go`、`cron_repo.go`、`llm_message_log_repo.go`、`oa_connection_repo.go`、`option_repo.go`、`org_repo.go`、`process_audit_config_repo.go`、`system_config_repo.go`、`system_prompt_template_repo.go`、`tenant_repo.go`、`user_dashboard_pref_repo.go`、`user_notification_repo.go`、`user_personal_config_repo.go`、`user_repo.go`

**`internal/handler/`**（18 个文件）

全部 Handler 文件均需：删除英文注释，为每个 HTTP 处理函数补充中文说明（接口用途、参数来源、返回格式），包括：
`archive_config_handler.go`、`archive_review_handler.go`、`archive_rule_handler.go`、`audit_review_handler.go`、`audit_rule_handler.go`、`auth_handler.go`、`cron_config_handler.go`、`cron_task_handler.go`、`dashboard_overview_handler.go`、`health_handler.go`、`llm_message_log_handler.go`、`org_handler.go`、`process_audit_config_handler.go`、`system_handler.go`、`tenant_handler.go`、`user_config_management_handler.go`、`user_notification_handler.go`、`user_personal_config_handler.go`

**`internal/service/`**（32 个文件）

全部 Service 文件均需：删除英文注释，为核心业务方法补充中文说明（业务意图、关键分支逻辑），包括：
`ai_caller_service.go`、`ai_model_service.go`、`ai_utils.go`、`archive_config_service.go`、`archive_prompt_builder.go`、`archive_result_parser.go`、`archive_review_service.go`、`archive_rule_service.go`、`archive_stream_worker.go`、`audit_prompt_builder.go`、`audit_result_parser.go`、`audit_review_service.go`、`audit_rule_service.go`、`audit_stream_worker.go`、`auth_service.go`、`cron_config_service.go`、`cron_scheduler.go`、`cron_task_service.go`、`dashboard_overview_service.go`、`llm_message_log_service.go`、`mail_service.go`、`oa_connection_service.go`、`option_service.go`、`org_service.go`、`process_audit_config_service.go`、`report_calculator_service.go`、`rule_merge.go`、`system_config_service.go`、`tenant_service.go`、`user_notification_service.go`、`user_personal_config_service.go`、`log_cleanup_service.go`（新建）

**`internal/pkg/`**（各子包）

| 子包 | 改造内容 |
|------|---------|
| `pkg/ai/` | 补充 AI 调用封装中文注释 |
| `pkg/crypto/` | 补充 AES 加解密中文注释 |
| `pkg/errcode/` | 补充错误码定义中文注释 |
| `pkg/hash/` | 补充哈希工具中文注释 |
| `pkg/jwt/` | 补充 JWT 签发/校验中文注释 |
| `pkg/mail/` | 补充邮件发送中文注释 |
| `pkg/oa/` | 补充 OA 数据源连接中文注释 |
| `pkg/response/` | 补充统一响应格式中文注释 |
| `pkg/sanitize/` | 补充输入清洗中文注释 |
| `pkg/logger/`（新建） | 全部使用中文注释 |

**`internal/router/`**

| 文件 | 改造内容 |
|------|---------|
| `router/router.go` | 补充路由分组、中间件挂载中文注释 |

**`internal/dbmigrate/`**

| 文件 | 改造内容 |
|------|---------|
| `dbmigrate/dbmigrate.go` | 补充迁移执行逻辑中文注释 |

#### 前端（Vue/TypeScript）—— 全量覆盖

**`frontend/middleware/`**（1 个文件）

| 文件 | 改造内容 |
|------|---------|
| `middleware/auth.ts` | 补充路由守卫、token 校验流程中文注释 |

**`frontend/composables/`**（18 个文件）

全部 composable 文件均需：删除英文注释，为关键函数、响应式变量、API 调用补充中文说明，包括：
`useAdminDataApi.ts`、`useAdminUserConfigApi.ts`、`useArchiveConfigApi.ts`、`useArchiveReviewApi.ts`、`useAuditApi.ts`、`useAuditConfigApi.ts`、`useAuth.ts`、`useCronApi.ts`、`useDashboardOverviewApi.ts`、`useI18n.ts`、`useLayoutPrefs.ts`、`useNotifications.ts`、`useOrgApi.ts`、`usePagination.ts`、`useSettingsApi.ts`、`useSidebarMenu.ts`、`useSystemApi.ts`、`useTheme.ts`

**`frontend/components/`**（9 个文件）

全部组件文件均需：删除英文注释，为 props、emit、关键计算逻辑补充中文说明，包括：
`AppHeader.vue`、`AppSidebar.vue`、`AuditPanel.vue`、`CronHistory.vue`、`RuleEditor.vue`、`RuleList.vue`、`SnapshotDetail.vue`、`charts/DeptDistributionChart.vue`、`charts/StackedBarChart.vue`

**`frontend/pages/`**（全部页面文件）

所有页面文件均需：删除英文注释，为页面初始化逻辑、数据加载、表单提交补充中文说明，包括 `admin/system/`、`admin/tenant/` 等目录下所有 `.vue` 文件。

**`frontend/constants/`**

| 文件 | 改造内容 |
|------|---------|
| `constants/overviewWidgets.ts` | 补充仪表盘组件配置中文注释 |

---

## 数据模型

### `LogConfig`（Go 结构体）

```go
type LogConfig struct {
    Level               string `mapstructure:"level"`
    Dir                 string `mapstructure:"dir"`
    MaxSizeMB           int    `mapstructure:"max_size_mb"`
    MaxBackups          int    `mapstructure:"max_backups"`
    Compress            bool   `mapstructure:"compress"`
    GlobalRetentionDays int    `mapstructure:"global_retention_days"`
}
```

### `system_configs` 新增记录

| key | value | remark |
|-----|-------|--------|
| `system.global_log_retention_days` | `30` | 全局系统日志文件保留天数 |

### `Tenant` 模型（已有字段，无需新增）

`LogRetentionDays int`（默认 365）已存在于 `model/tenant.go`，直接复用。

---

## 正确性属性

*属性（Property）是在系统所有合法执行中都应成立的特征或行为——本质上是对系统应做什么的形式化陈述。属性是人类可读规范与机器可验证正确性保证之间的桥梁。*

### 属性 1：租户日志文件路径唯一性

*对于任意* 两个不同 `tenantCode` 的租户，`GetTenantLogger` 返回的 logger 写入的文件路径必须不同，不存在路径冲突。

**验证：需求 4.1、4.2**

### 属性 2：日志等级过滤正确性

*对于任意* 配置的日志等级 L，所有严重程度低于 L 的日志条目均不应出现在输出文件中；所有严重程度大于等于 L 的日志条目均应出现。

**验证：需求 8.5**

### 属性 3：日志清理保留天数边界

*对于任意* 保留天数 N 和日志备份文件集合，清理后剩余的文件修改时间距今均不超过 N 天；修改时间距今超过 N 天的文件均已被删除。

**验证：需求 6.3、6.4**

### 属性 4：配置值合法性校验

*对于任意* 日志保留天数配置值 V，当 V 不是正整数或 V 超出范围 [1, 3650] 时，系统应拒绝保存并返回错误；当 V 在合法范围内时，系统应成功保存。

**验证：需求 7.4**

### 属性 5：租户 logger 缓存幂等性

*对于任意* `tenantCode`，多次调用 `GetTenantLogger(tenantCode)` 返回的是同一个 `*zap.Logger` 实例（指针相等），不会重复创建。

**验证：需求 4.5**

---

## 错误处理

### logger 初始化失败

- `Init` 失败时返回 `error`，`main.go` 调用 `log.Fatalf` 终止进程，避免无日志运行。
- 若 `logs/` 目录无写权限，`Init` 应返回明确错误信息，包含目录路径。

### 租户 logger 创建失败

- `GetTenantLogger` 内部若目录创建失败，降级返回全局 logger，并在全局日志中记录 WARN，不 panic。
- 降级行为确保业务流程不因日志问题中断。

### 日志清理任务失败

- 单个文件删除失败（如权限问题）时，记录 WARN 并继续处理其他文件，不中断整个清理任务。
- 清理任务完成后，无论成功与否，均向全局日志写入一条 INFO 汇总（删除数量、释放空间、失败数量）。

### 配置值缺失兜底

- `system_configs` 中 `system.global_log_retention_days` 不存在时，使用 `config.yaml` 中 `log.global_retention_days` 的值（默认 30）。
- `config.yaml` 中 `log` 节缺失时，所有字段使用硬编码默认值，并在启动时记录 WARN。

### 配置值合法性

- 保存日志保留天数时，后端 handler 校验值为正整数且在 [1, 3650] 范围内，不合法时返回 HTTP 400 和中文错误提示。

---

## 测试策略

### 单元测试

针对以下纯函数/逻辑编写示例测试：

- `LogConfig` 默认值填充逻辑（缺失字段是否正确使用默认值）
- `CleanupGlobalLogs` / `CleanupTenantLogs`：给定模拟文件系统，验证超期文件被删除、未超期文件保留
- 配置值合法性校验函数：边界值 1、3650、0、-1、3651 的处理结果
- `GetTenantLogger` 缓存命中：同一 tenantCode 两次调用返回相同指针

### 属性测试

使用 [`pgregory.net/rapid`](https://github.com/pgregory/rapid) 库（Go 生态主流 PBT 库）实现以下属性测试，每个属性最少运行 100 次：

**属性测试 1：日志等级过滤正确性**
```
// Feature: code-comments-and-logging, Property 2: 日志等级过滤正确性
// 生成随机日志等级配置和随机日志条目，验证过滤行为符合预期
```

**属性测试 2：日志清理保留天数边界**
```
// Feature: code-comments-and-logging, Property 3: 日志清理保留天数边界
// 生成随机文件集合（随机修改时间）和随机保留天数，验证清理结果
```

**属性测试 3：配置值合法性校验**
```
// Feature: code-comments-and-logging, Property 4: 配置值合法性校验
// 生成随机整数，验证 [1,3650] 内通过，范围外拒绝
```

**属性测试 4：租户 logger 缓存幂等性**
```
// Feature: code-comments-and-logging, Property 5: 租户 logger 缓存幂等性
// 生成随机 tenantCode 序列，验证相同 code 返回相同指针
```

### 集成测试

- 启动时日志文件自动创建（目录不存在时）
- lumberjack 轮转触发（写入超过 maxSize 后验证备份文件存在）
- 日志清理任务读取 `system_configs` 并正确执行清理

### 注释质量验收

代码注释改造属于人工审查范畴，不适合自动化测试。验收标准：
- 无英文注释（`grep -r "// [A-Z]" internal/` 结果为空）
- 无翻译腔特征词（"返回一个"、"保存应用程序"等）
- 每个公开函数/结构体均有中文注释
