# [#003] OA 嵌入审核仅支持单租户部署配置

## 状态

**已解决**（当前版本已改为租户级嵌入密钥，单实例可服务多租户）

## 历史问题

早期版本通过环境变量 `EMBED_TENANT_CODE` + `EMBED_ACCESS_TOKEN` 做部署级固定租户绑定，一套 AuraOA 实例只能服务一个租户。

## 当前方案

| 维度 | 当前实现 |
|------|----------|
| 租户识别 | `X-Embed-Token` → `tenants.embed_token_hash` |
| 密钥管理 | 系统管理 → 租户管理 → OA 嵌入 |
| 多租户 | 同一 AuraOA 实例可为每个租户生成独立密钥 |
| 流程规则 | `process_id → process_type → 租户流程配置` |

## 相关文档

- [OA 嵌入 AI 审核侧边栏](../oa-configurations/02-embed-audit-sidebar.md)
- [嵌入审核 API](../api/embed.md)
