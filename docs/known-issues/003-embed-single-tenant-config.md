# [#003] OA 嵌入审核仅支持单租户部署配置

## 状态

**已知限制**（当前版本按设计接受；多租户嵌入待后续扩展）

## 问题描述

OA 嵌入页（`/embed/audit`）通过环境变量 `EMBED_TENANT_CODE` + `EMBED_ACCESS_TOKEN` 鉴权，**不经过用户 JWT 登录**。由于无法在请求中识别「当前 OA 用户属于哪个 AuraOA 租户」，系统采用 **部署级固定租户**：

- Nuxt 服务端代理从 `frontend/.env` 读取固定的 `EMBED_TENANT_CODE`，转发至 Go 的 `X-Tenant-Code`
- Go `EmbedAccess` 中间件据此解析 `tenant_id`，后续查询规则、快照、审核日志均限定在该租户

因此：**一套 AuraOA 部署 + 一组 `EMBED_*` 环境变量，当前只能服务一个租户**（如 `tenants.code = T-20260330-0001` 的「总部」）。

## 影响范围

| 场景 | 是否受影响 |
|------|------------|
| 单公司 / 单 OA ↔ 单 AuraOA 租户 | ✅ 不受影响，按文档配置即可 |
| 每租户独立部署 AuraOA（不同域名或不同 `.env`） | ✅ 每套各自配置 `EMBED_TENANT_CODE` |
| **同一 AuraOA 实例**，多套 OA 或多个租户都要嵌侧边栏 | ❌ 仅最后一个配置的租户生效，其他租户数据会查错或无权 |
| 工作台、管理端等需 JWT 的页面 | ✅ 不受影响，仍按登录用户租户隔离 |

## 与「免登录」的关系

| 维度 | 工作台 | OA 嵌入（当前） |
|------|--------|-----------------|
| 用户身份 | JWT → `user_id` | 不识别具体用户 |
| 租户识别 | JWT → `tenant_id` | `EMBED_TENANT_CODE`（环境变量） |
| 授权方式 | 用户权限 | `EMBED_ACCESS_TOKEN`（嵌入专用密钥） |

嵌入页「免登录」指的是 OA 用户无需 AuraOA 账号；**租户仍须在服务端配置中指定**，并非「无租户」。

## 临时方案（不改代码）

1. **一租户一部署**：每个 AuraOA 租户单独环境，`EMBED_TENANT_CODE` 填对应 `tenants.code`（如 `T-20260330-0001`），OA iframe `src` 指向该环境地址。
2. **接受单租户**：若业务上仅一个租户使用嵌入，在根目录 `.env` 与 `frontend/.env` 中配置该租户 code 即可。

## 后续扩展方向（未实现）

| 方案 | 说明 |
|------|------|
| OA `postMessage` 传 `tenant_code` | 泛微脚本与 `requestid` 一并推送；Nuxt 按请求动态设置 `X-Tenant-Code` |
| 按 `requestid` 反查租户 | 后端在 OA 只读库/业务表解析流程归属租户；需保证 requestid 全局唯一 |
| 每租户独立 `embed_access_token` | 库表或管理端配置多组令牌，token 映射租户，取消全局单 `EMBED_TENANT_CODE` |

## 相关文档

- [OA 嵌入 AI 审核侧边栏](../oa-configurations/02-embed-audit-sidebar.md)
- [嵌入审核 API](../api/embed.md)
