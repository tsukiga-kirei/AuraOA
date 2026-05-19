# [#002] 审核数据保留天数未实现自动清理

## 状态

**未修复**（配置已存在，定时清理逻辑未实现）

## 问题描述

系统设置中的 **「默认审核数据保留天数」**（`tenant.default_data_retention_days`）以及租户上的 **`data_retention_days`** 字段，产品设计意图是：超过保留期限后 **自动删除数据库中的审核/归档相关业务数据**（例如 `audit_logs`、`archive_logs`、`audit_process_snapshots`、`archive_process_snapshots` 等，具体范围待实现时确认）。

当前实现仅：

- 在 **新建租户** 时，若未指定 `data_retention_days`，从系统配置写入默认值（默认 1095 天）；
- 在 **租户管理** 中可查看、修改该字段。

**未实现**按该天数执行的定时清理任务，修改配置 **不会** 导致历史审核/归档数据被删除。

## 影响范围

| 位置 | 说明 |
|------|------|
| 管理端 → 系统设置 | 「默认审核数据保留天数」易被理解为已生效的自动删库策略 |
| 管理端 → 租户管理 | 租户级「审核数据保留天数」同样无自动清理 |
| 数据管理页 | `audit_logs` / `archive_logs` 等数据会随业务持续增长 |
| 磁盘 | 与 Zap 文件日志（`app.log` / `tenant.log`）无关；本项针对 **PostgreSQL 业务表** |

## 与相关配置的区别

| 配置 | 键名 | 当前行为 |
|------|------|----------|
| 全局系统日志保留天数 | `system.global_log_retention_days` | ✅ 已实现：清理 `logs/app.log` **轮转备份** |
| 默认/租户日志保留天数 | `tenant.default_log_retention_days` / `tenants.log_retention_days` | ✅ 已实现：清理 `logs/tenants/{code}/` **轮转备份** |
| 默认/租户审核数据保留天数 | `tenant.default_data_retention_days` / `tenants.data_retention_days` | ❌ **未实现**库表数据自动删除 |

## 预期行为（待实现）

1. 定时任务（建议与现有日志清理任务协调，避免高峰重叠）按租户 `data_retention_days` 清理过期记录。
2. 明确清理表范围、是否级联快照/LLM 日志、是否保留统计聚合等规则。
3. `data_retention_days = 0` 的语义需在实现前定义（禁止删除 / 立即清理 / 不限制等），并与前端文案一致。
4. 系统默认仅影响 **新建租户**；若需批量同步已有租户，应单独提供管理操作或迁移说明。

## 参考代码

- 默认值读取：`go-service/internal/service/tenant_service.go`（创建租户）
- 租户字段：`tenants.data_retention_days`（`db/migrations/000002_tenants_users.up.sql`）
- 系统配置种子：`tenant.default_data_retention_days`（`db/migrations/000004_system_configs.up.sql`）
- 已实现对比：`go-service/internal/service/log_cleanup_service.go`（仅文件日志备份）

## 优先级建议

**中** — 配置已暴露给用户，长期运行可能导致库表膨胀；实现前应在界面或文档中标明「尚未自动清理」（本条目即作跟踪）。
