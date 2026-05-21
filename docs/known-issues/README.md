# 已知缺陷与待完成事项

本目录用于持续维护当前版本的已知问题、影响范围、优先级以及修复进度。

## 已知缺陷

| 编号 | 标题 | 状态 | 说明 |
|------|------|------|------|
| [#002](./002-audit-data-retention-cleanup.md) | 审核数据保留天数未实现自动清理 | 未修复 | 系统/租户「审核数据保留天数」意图为删除库表中的审核/归档业务数据，当前仅保存配置，无定时删库任务 |
| [#003](./003-embed-single-tenant-config.md) | OA 嵌入审核仅支持单租户部署配置 | 已知限制 | `EMBED_TENANT_CODE` 为部署级固定值，一套实例一组 `EMBED_*` 仅服务一个租户；多租户嵌入待扩展 |
| [#004](./004-ecology9-browse-button-raw-db-value.md) | 泛微 E9 浏览按钮仅读取数据库原始值 | 已知限制 | `fieldhtmltype=3` 不做 ID→显示名解析，AI 审核拿到的是表单表列上的原始 ID |

## 缺陷详情索引

| 文档 | 摘要 |
|------|------|
| [002-audit-data-retention-cleanup.md](./002-audit-data-retention-cleanup.md) | `data_retention_days` / `tenant.default_data_retention_days` 未驱动 `audit_logs` 等表自动清理 |
| [003-embed-single-tenant-config.md](./003-embed-single-tenant-config.md) | 嵌入页免登录场景下租户仅能靠环境变量指定，不支持同一实例多租户动态切换 |
| [004-ecology9-browse-button-raw-db-value.md](./004-ecology9-browse-button-raw-db-value.md) | Ecology9 浏览按钮（`fieldhtmltype=3`）直接读 `formtable_*` 列值，不解析为显示名 |
