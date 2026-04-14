# 需求文档

## 简介

本功能涵盖两大方向的优化：

1. **代码注释规范化**：对前端（Nuxt.js/Vue 3）和后端（Go）代码进行全面梳理，在重点位置补充中文注释，删除所有英文注释及机器翻译注释，确保注释准确、地道、统一。

2. **日志系统完善**：后端目前日志输出极少，需要建立完整的结构化日志体系，包括：按租户隔离的独立日志文件、全局日志文件、日志轮转与备份机制、基于 `system_configs` 表的日志保留天数配置（`tenant.default_log_retention_days`）、合理的日志等级划分（DEBUG/INFO/WARN/ERROR），以及系统管理员可在前端配置日志保留策略。

---

## 词汇表

- **日志系统（Logger）**：负责记录运行时事件的模块，基于 `go.uber.org/zap` 实现。
- **全局日志（GlobalLogger）**：记录与租户无关的系统级事件（启动、迁移、中间件等）的日志实例。
- **租户日志（TenantLogger）**：为每个租户单独生成的日志实例，日志写入该租户专属文件。
- **日志轮转（LogRotation）**：按文件大小或时间自动切割日志文件，防止单文件过大，基于 `lumberjack` 实现。
- **日志保留天数（LogRetentionDays）**：日志文件在磁盘上保留的最大天数，超期自动清理。
- **系统配置（SystemConfig）**：存储在 `system_configs` 表中的全局键值配置，键 `tenant.default_log_retention_days` 控制新建租户的默认日志保留天数。
- **中文注释（ChineseComment）**：代码中使用中文书写的注释，用于说明逻辑意图，禁止使用英文或机器翻译注释。
- **日志等级（LogLevel）**：日志的严重程度分级，包括 DEBUG、INFO、WARN、ERROR。
- **系统管理员（SystemAdmin）**：拥有最高权限的管理员角色，可配置全局系统参数。

---

## 需求

### 需求 1：后端代码全面补充中文注释

**用户故事：** 作为开发者，我希望后端 Go 代码的重点位置都有中文注释，以便快速理解业务逻辑，降低维护成本。

#### 验收标准

1. THE 开发团队 SHALL 对 `go-service/internal/` 目录下所有 `.go` 文件的包声明、结构体、公开函数、关键业务逻辑分支进行中文注释覆盖。
2. THE 开发团队 SHALL 删除所有英文注释（包括机器翻译得来的中文注释，如"//Logger 返回一个中间件，使用提供的记录每个请求"此类翻译腔注释）。
3. WHEN 函数或方法包含复杂业务逻辑时，THE 开发团队 SHALL 在关键步骤前添加行内中文注释说明意图。
4. THE 开发团队 SHALL 确保注释语言自然、准确，不得使用直译英文的表达方式。
5. THE 开发团队 SHALL 对 `cmd/server/main.go` 中的初始化步骤、`internal/config/config.go` 中的配置字段、`internal/middleware/` 中的中间件函数均添加中文注释。

---

### 需求 2：前端代码全面补充中文注释

**用户故事：** 作为前端开发者，我希望 Vue/TypeScript 代码的重点位置都有中文注释，以便理解组件逻辑和 API 调用意图。

#### 验收标准

1. THE 开发团队 SHALL 对 `frontend/composables/`、`frontend/pages/`、`frontend/components/` 目录下所有 `.vue` 和 `.ts` 文件的关键函数、响应式变量声明、API 调用处添加中文注释。
2. THE 开发团队 SHALL 删除所有英文注释及翻译腔中文注释。
3. WHEN 组件包含复杂的状态管理或业务逻辑时，THE 开发团队 SHALL 在对应代码块前添加中文说明注释。
4. THE 开发团队 SHALL 确保 `frontend/locales/` 国际化文件中的键名含义通过注释或命名自解释，无需额外注释。
5. THE 开发团队 SHALL 对 `frontend/middleware/auth.ts` 和 `frontend/composables/useAuth.ts` 中的鉴权逻辑添加详细中文注释。

---

### 需求 3：建立结构化全局日志系统

**用户故事：** 作为运维人员，我希望系统有完整的全局日志输出，以便排查系统级问题（启动失败、数据库连接异常、中间件错误等）。

#### 验收标准

1. THE Logger SHALL 将全局日志同时输出到控制台（stdout）和磁盘文件 `logs/app.log`。
2. WHEN 系统启动时，THE Logger SHALL 记录 INFO 级别日志，包含服务端口、数据库连接状态、Redis 连接状态、迁移执行结果。
3. WHEN 数据库或 Redis 连接失败时，THE Logger SHALL 记录 ERROR 级别日志并包含错误详情。
4. WHEN HTTP 请求完成时，THE Logger SHALL 记录 INFO 级别日志，包含请求方法、路径、状态码、耗时、客户端 IP。
5. WHEN 请求处理发生 panic 时，THE Logger SHALL 记录 ERROR 级别日志并包含堆栈信息。
6. THE Logger SHALL 支持通过 `config.yaml` 中的 `log.level` 字段配置全局日志等级（debug/info/warn/error），默认为 `info`。
7. THE Logger SHALL 支持通过 `config.yaml` 中的 `log.global_retention_days` 字段配置全局日志文件保留天数，默认为 `30`。

---

### 需求 4：建立租户隔离的独立日志文件

**用户故事：** 作为系统管理员，我希望每个租户的操作日志写入独立文件，以便按租户审计和排查问题，同时避免日志混杂。

#### 验收标准

1. THE Logger SHALL 为每个租户生成独立的日志文件，路径格式为 `logs/tenants/{tenant_code}/tenant.log`。
2. WHEN 租户相关的业务操作（审核、归档、AI 调用、定时任务等）发生时，THE Logger SHALL 将日志同时写入全局日志和该租户的专属日志文件。
3. THE Logger SHALL 在每条租户日志中携带 `tenant_id` 和 `tenant_code` 字段，便于检索。
4. WHEN 租户日志文件不存在时，THE Logger SHALL 自动创建对应目录和文件。
5. THE Logger SHALL 支持通过依赖注入方式获取租户专属 Logger 实例（`GetTenantLogger(tenantCode string) *zap.Logger`）。

---

### 需求 5：日志轮转与备份机制

**用户故事：** 作为运维人员，我希望日志文件能自动轮转和备份，防止单个日志文件无限增长占满磁盘。

#### 验收标准

1. THE Logger SHALL 使用 `lumberjack` 库对所有日志文件（全局和租户）实现基于文件大小的自动轮转，单文件最大 `100MB`。
2. THE Logger SHALL 保留最近 `5` 个轮转备份文件（可通过配置调整）。
3. THE Logger SHALL 支持对备份文件进行 gzip 压缩，减少磁盘占用。
4. WHEN 日志文件超过最大大小时，THE Logger SHALL 自动将当前文件重命名为带时间戳的备份文件，并创建新的日志文件继续写入。
5. THE Logger SHALL 支持通过 `config.yaml` 中的 `log.max_size_mb`、`log.max_backups`、`log.compress` 字段配置轮转参数。

---

### 需求 6：基于系统配置的日志保留天数控制

**用户故事：** 作为系统管理员，我希望能在系统配置中设置租户日志的默认保留天数，超期日志自动清理，避免磁盘空间浪费。

#### 验收标准

1. THE Logger SHALL 读取 `system_configs` 表中 `tenant.default_log_retention_days` 的值作为新建租户的默认日志保留天数。
2. WHEN 租户的 `log_retention_days` 字段有值时，THE Logger SHALL 优先使用租户自身的配置，而非全局默认值。
3. THE Logger SHALL 通过定时任务（每日凌晨执行）扫描 `logs/tenants/` 目录，删除超过对应租户保留天数的日志备份文件。
4. THE Logger SHALL 通过定时任务扫描 `logs/` 目录，删除超过 `log.global_retention_days` 天的全局日志备份文件。
5. WHEN 日志清理任务执行时，THE Logger SHALL 记录清理结果（删除文件数量、释放空间）到全局日志。

---

### 需求 7：系统管理员可配置日志保留策略

**用户故事：** 作为系统管理员，我希望能在系统设置页面配置日志保留天数，无需修改配置文件或重启服务。

#### 验收标准

1. THE SystemAdmin SHALL 能在系统设置页面的"数据保留策略"分组中查看和修改 `tenant.default_log_retention_days` 配置项。
2. WHEN 系统管理员保存配置时，THE SystemConfig SHALL 将新值持久化到 `system_configs` 表，并在下次日志清理任务执行时生效。
3. THE SystemAdmin SHALL 能在系统设置页面新增 `system.global_log_retention_days` 配置项，用于控制全局日志文件的保留天数。
4. WHEN 配置值不合法（非正整数、超出范围 1~3650）时，THE SystemConfig SHALL 拒绝保存并返回明确的错误提示。
5. THE SystemAdmin SHALL 能在租户管理页面单独修改每个租户的 `log_retention_days` 字段，覆盖全局默认值。

---

### 需求 8：合理的日志等级划分

**用户故事：** 作为开发者和运维人员，我希望日志等级划分合理，生产环境只输出必要信息，调试时可开启详细日志。

#### 验收标准

1. THE Logger SHALL 对以下场景使用 DEBUG 级别：SQL 查询详情、AI 请求/响应原始内容、中间件详细参数。
2. THE Logger SHALL 对以下场景使用 INFO 级别：服务启动/停止、HTTP 请求完成、定时任务执行开始/结束、租户操作（创建/更新/删除）、AI 调用成功。
3. THE Logger SHALL 对以下场景使用 WARN 级别：重试操作、配置项缺失使用默认值、非关键性错误（如邮件发送失败但不影响主流程）。
4. THE Logger SHALL 对以下场景使用 ERROR 级别：数据库操作失败、AI 调用最终失败（重试耗尽）、认证/鉴权失败、panic 恢复。
5. WHEN 生产环境日志等级设置为 `info` 时，THE Logger SHALL 不输出 DEBUG 级别日志，避免敏感信息泄露。

---

### 需求 9：新增全局日志保留配置项

**用户故事：** 作为系统管理员，我希望系统配置表中有专门的全局日志保留天数配置项，与租户日志保留配置分开管理。

#### 验收标准

1. THE SystemConfig SHALL 在 `system_configs` 表中新增键 `system.global_log_retention_days`，默认值为 `30`，备注为"全局系统日志文件保留天数"。
2. THE SystemConfig SHALL 通过数据库迁移文件（新建 migration）添加该配置项，保证可回滚。
3. THE Logger SHALL 在启动时读取该配置项，若不存在则使用 `config.yaml` 中的 `log.global_retention_days` 作为兜底默认值。
4. WHEN 该配置项被更新时，THE Logger SHALL 在下次清理任务执行时使用新值，无需重启服务。
