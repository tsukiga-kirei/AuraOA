# OA 系统适配功能说明

> 文档版本：v1.0 | 创建日期：2026-03-19
> 从代码层面介绍 OA 智审平台与企业 OA 系统的连接与数据适配能力。

---

## 一、整体架构

OA 智审通过 **OA 适配器（OAAdapter）** 接口与企业 OA 系统交互。适配器通过直连 OA 系统数据库的方式获取流程表单与字段信息，不依赖 OA 系统的 API 接口。

```
                         ┌─────────────────┐
                         │   Go Service    │
                         │                 │
  ┌──────────┐    SQL    │  ┌───────────┐  │     ┌──────────────────┐
  │  OA 数据库 │◄────────│  │ OAAdapter │  │     │ process_audit_   │
  │ (MySQL/   │          │  │ (泛微E9)  │  │────►│ configs          │
  │  Oracle/  │          │  └───────────┘  │     │ (PostgreSQL)     │
  │  达梦)    │          │                 │     └──────────────────┘
  └──────────┘          └─────────────────┘
```

---

## 二、OA 适配器接口定义

**文件位置**：`go-service/internal/pkg/oa/adapter.go`

```go
type OAAdapter interface {
    // 验证流程类型是否存在于 OA 系统中
    ValidateProcess(ctx context.Context, processType string) (*ProcessInfo, error)

    // 拉取指定流程的全部字段定义（主表 + 明细表）
    FetchFields(ctx context.Context, processType string) (*ProcessFields, error)

    // 检查用户在 OA 中是否具有指定流程的审批权限
    CheckUserPermission(ctx context.Context, userID string, processType string) (bool, error)

    // 拉取指定流程实例的业务数据（用于审核执行）
    FetchProcessData(ctx context.Context, processID string) (*ProcessData, error)
}
```

### 核心数据结构

| 结构体 | 说明 |
|--------|------|
| `ProcessInfo` | 流程基本信息（类型、名称、主表名、明细表数量），支持校验表名/类型标签是否与 OA 一致 |
| `ProcessFields` | 流程字段集合，包含主表字段列表 `main_fields` 和明细子表列表 `detail_tables` |
| `FieldDef` | 单个字段定义：`field_key`（内部键）、`field_name`（显示名）、`field_type`（字段类型） |
| `DetailTableDef` | 明细表定义，含表名 `table_name`、标签 `table_label`、字段列表 `fields` |
| `ProcessData` | 流程实例业务数据，含主表数据 `main_data` 和明细表数据 `detail_data` |

---

## 三、当前已适配的 OA 系统

### 3.1 泛微 Ecology E9

**文件位置**：`go-service/internal/pkg/oa/ecology9.go`

| 项目 | 说明 |
|------|------|
| 支持的数据库驱动 | MySQL、Oracle、达梦 DM |
| 流程类型识别 | 通过 `workflow_base` 表查询流程名称和主表名 |
| 字段提取 | 从 `workflow_fieldinfo` 和 `htmllabelinfo` 表提取字段元数据 |
| 明细表提取 | 从 `workflow_detail_table` 和 `htmllabel_detail` 表提取明细子表 |
| 数据拉取 | 动态拼接 SQL 查询 `formtable_main_N` 和 `formtable_main_N_dtN` |

**E9 特有逻辑**：
- 流程表名格式为 `formtable_main_{N}`，其中 N 对应流程 ID
- 明细子表格式为 `formtable_main_{N}_dt{M}`
- 字段名格式为 `field{N}`，显示名通过 `htmllabelinfo` 关联

### 3.2 驱动层适配

**Oracle 适配**：`go-service/internal/pkg/oa/oracle/oracle.go`
- 使用 `sijms/go-ora/v2` 驱动
- 通过 `gorm-oracle` 适配 GORM

**达梦 DM 适配**：`go-service/internal/pkg/oa/dm/dm.go`
- 使用 `dm-driver-gorm` 驱动
- 需要 `dm_security` 编译标签

---

## 四、OA 数据库连接管理

### 4.1 数据模型

**表名**：`oa_database_connections`

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | VARCHAR(200) | 连接名称 |
| `oa_type` | VARCHAR(50) | OA 系统类型编码（关联 `oa_type_options`） |
| `driver` | VARCHAR(50) | 数据库驱动（mysql/oracle/postgresql/sqlserver/dm） |
| `host` / `port` | - | 数据库地址 |
| `database_name` | VARCHAR(200) | 数据库名 |
| `username` / `password` | - | 认证信息（密码 AES-256 加密存储） |
| `pool_size` | INT | 连接池大小（默认 10） |
| `connection_timeout` | INT | 连接超时（秒，默认 30） |
| `test_on_borrow` | BOOLEAN | 取出连接时是否测试 |
| `status` | VARCHAR(20) | 状态：connected/disconnected/error |
| `sync_interval` | INT | 同步间隔（分钟，默认 30） |

### 4.2 连接管理 API（系统管理员）

| API | 说明 |
|-----|------|
| `GET /api/admin/system/oa-connections` | 列出所有 OA 连接 |
| `POST /api/admin/system/oa-connections` | 创建新连接 |
| `PUT /api/admin/system/oa-connections/:id` | 更新连接配置 |
| `DELETE /api/admin/system/oa-connections/:id` | 删除连接 |
| `POST /api/admin/system/oa-connections/test` | 测试连接参数（不保存） |
| `POST /api/admin/system/oa-connections/:id/test` | 测试已保存的连接 |

### 4.3 连接与租户的关系

租户通过 `tenants.oa_db_connection_id` 关联一个 OA 数据库连接。删除连接时，关联租户的字段会被置为 NULL（`ON DELETE SET NULL`）。

---

## 五、流程配置与字段拉取

### 5.1 审核工作台（Process Audit Config）

**表名**：`process_audit_configs`

租户管理员配置需要审核的流程类型：
1. 输入流程类型标识（如"采购审批"）
2. 系统通过 OA 适配器验证流程存在性，返回主表名
3. 点击「拉取字段」，适配器从 OA 数据库拉取全部字段定义
4. 管理员选择需要审核的字段（`field_mode`：`all` 全部 / `selected` 部分）
5. 字段元数据保存到 `main_fields`（JSONB）和 `detail_tables`（JSONB）

**相关 API**：
| API | 说明 |
|-----|------|
| `POST /api/tenant/rules/configs/test-connection` | 测试流程连接 |
| `POST /api/tenant/rules/configs/:id/fetch-fields` | 拉取流程字段 |

### 5.2 归档复盘（Process Archive Config）

**表名**：`process_archive_configs`

结构与审核配置类似，额外增加：
- `access_control`（JSONB）：访问控制列表（允许的角色/成员/部门）
- 独立的归档规则表 `archive_rules`

**相关 API**：
| API | 说明 |
|-----|------|
| `POST /api/tenant/archive/configs/test-connection` | 测试连接 |
| `POST /api/tenant/archive/configs/:id/fetch-fields` | 拉取字段 |

---

## 六、当前支持的 OA 类型选项

由 `oa_type_options` 表管理（种子数据初始化）：

| 编码 | 名称 | 适配器状态 |
|------|------|-----------|
| `weaver_e9` | 泛微 Ecology E9 | ✅ 已实现 |
| `weaver_ebridge` | 泛微 E-Bridge | ❌ 未实现 |
| `zhiyuan_a8` | 致远 A8+ | ❌ 未实现 |
| `landray_ekp` | 蓝凌 EKP | ❌ 未实现 |
| `custom` | 自定义 OA | ❌ 未实现 |

### 扩展新 OA 类型的步骤

1. 在 `go-service/internal/pkg/oa/` 下新建适配器文件，实现 `OAAdapter` 接口
2. 在 `factory.go` 的 `supportedDrivers` 中注册支持的驱动
3. 在 `NewOAAdapter` 的 `switch` 中添加新 case
4. 在 `oa_type_options` 表中添加选项记录

---

## 七、当前支持的数据库驱动

由 `db_driver_options` 表管理：

| 编码 | 名称 | 默认端口 | Go 驱动 |
|------|------|---------|---------|
| `mysql` | MySQL | 3306 | `go-sql-driver/mysql` |
| `oracle` | Oracle | 1521 | `sijms/go-ora/v2` + `gorm-oracle` |
| `postgresql` | PostgreSQL | 5432 | `jackc/pgx` |
| `sqlserver` | SQL Server | 1433 | `gorm.io/driver/sqlserver`（未引入） |
| `dm` | 达梦 DM | 5236 | `dm-driver-gorm`（需 `dm_security` 标签） |

---

## 八、密码安全

OA 数据库连接的密码字段 (`oa_database_connections.password`) 采用 AES-256 对称加密存储：

- **加密工具**：`go-service/internal/pkg/crypto/aes.go`
- **密钥配置**：通过 `config.yaml` 的 `encryption.key` 或环境变量 `ENCRYPTION_KEY` 配置
- **密钥要求**：必须为 32 字节（AES-256）
- **加密时机**：创建/更新 OA 连接时在 Service 层加密后写入
- **解密时机**：连接测试、字段拉取等需要实际连接 OA 数据库时解密

---

## 九、已知问题与注意事项

1. **达梦驱动编译问题**：`dm-driver-gorm` 在 macOS 上编译需要 `dm_security` 标签，可能导致 `go build` 失败。建议在 Linux 环境下编译或使用 Docker 构建。

2. **E-Bridge / 致远 / 蓝凌适配器未实现**：目前仅泛微 E9 有完整适配器，其他 OA 系统需要根据其数据库结构单独实现。

3. **字段拉取为一次性操作**：拉取后的字段元数据保存在 PostgreSQL 的 JSONB 中，如果 OA 系统修改了表单字段，需要手动重新拉取。

4. **连接池管理**：每次测试连接或拉取字段时会创建新的数据库连接，没有全局连接池管理。对于高频的审核执行场景，需要考虑引入连接池复用。
