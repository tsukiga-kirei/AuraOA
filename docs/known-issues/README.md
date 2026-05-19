# 已知缺陷与待完成事项

本目录用于持续维护当前版本的已知问题、影响范围、优先级以及修复进度。

## 已知缺陷

| 编号 | 标题 | 状态 | 说明 |
|------|------|------|------|
| [#002](./002-audit-data-retention-cleanup.md) | 审核数据保留天数未实现自动清理 | 未修复 | 系统/租户「审核数据保留天数」意图为删除库表中的审核/归档业务数据，当前仅保存配置，无定时删库任务 |

## 缺陷详情索引

| 文档 | 摘要 |
|------|------|
| [002-audit-data-retention-cleanup.md](./002-audit-data-retention-cleanup.md) | `data_retention_days` / `tenant.default_data_retention_days` 未驱动 `audit_logs` 等表自动清理 |
