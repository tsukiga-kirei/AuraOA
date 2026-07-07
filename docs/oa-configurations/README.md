# OA 系统侧配置说明

本目录用于汇总每个被 AuraOA 接入的 OA 系统、为了配合 AuraOA 某个能力而需要在 OA 侧做的配置或开发改造。

> 当 AuraOA 的某个能力依赖 OA 系统提供数据 / 接口（不仅是只读数据库）时，对应能力的需求文档会在 `docs/known-issues/` 描述「为什么这么做」，而**具体的 OA 侧落地步骤**写在本目录。

## 当前已沉淀的能力

| 编号 | 能力 | 说明 | 文档 |
|------|------|------|------|
| 01   | 附件识别 | 把流程附件（PDF / 图片 / Office 文档等）解析为文本送入 AI | [01-attachment-recognition.md](./01-attachment-recognition.md) |
| 02   | 嵌入 AI 审核侧边栏 | 在 E9 审批页 iframe 嵌入 AuraOA 审核结果，按 requestid 自动/手动审核 | [02-embed-audit-sidebar.md](./02-embed-audit-sidebar.md) |
| 03   | 嵌入流程总结侧边栏 | 在 E9 审批页 iframe 嵌入 AuraOA 总结结果，按多总结块配置摘要流程字段和附件 | [03-embed-process-summary.md](./03-embed-process-summary.md) |

## OA 类型矩阵

| OA 类型 | 标识 | 是否需要原生 API 密钥 | 已支持的能力 |
|---------|------|----------------------|--------------|
| 泛微 Ecology9 | `weaver_e9` | ✅ 当前使用 `api_url` / `appid` / `loginid` | 全部 |

> 当前**仅泛微 E9 需要密钥**。其他 OA 类型如果未来接入，仅在确实需要调用其原生 OpenAPI 时才需要新增密钥字段。

## 配置位置速查

| 配置类别 | 存放位置 | 维护页面 |
|----------|----------|----------|
| OA 数据库连接（host / port / 用户名密码） | `oa_database_connections` 表 | 系统设置 → OA 数据库 |
| 泛微 E9 原生 API 密钥（仅 weaver_e9） | `oa_database_connections` 表（每条记录独立） | 同上，OA 类型选「泛微 E9」时显示 |
| 附件识别 / MinerU 参数 | `system_configs` 表（attachment.* key） | 系统设置 → 附件识别 |
| 流程审核配置（含 embed 开关） | `process_audit_configs` 表 | 租户管理 → 规则配置 → 权限 |
| OA 嵌入 AI 审核 | `process_audit_configs.embed_enabled` / `embed_config` | 租户管理 → 规则配置 → 权限 → OA 嵌入审核 |
| 流程总结配置（含 embed 开关） | `process_summary_configs` 表 | 租户管理 → 规则配置 → 流程总结 |
| OA 嵌入流程总结 | `process_summary_configs.embed_enabled` / `embed_config` | 租户管理 → 规则配置 → 流程总结 → OA 嵌入总结 |

## 维护守则

1. 新增能力时，先在 `docs/known-issues/` 描述需求；落地后**必须**在本目录写一份对应的 OA 配置文档。
2. 若一个能力涉及多种 OA 系统，在同一文档里按 OA 类型分小节写实现示例（数据库表、字段、SQL、Java 示例、PHP 示例等）。
3. 所有外部参考链接（厂商文档、网盘示例、内部 wiki 等）都直接列在文档内，不要外链 PR / Slack。
