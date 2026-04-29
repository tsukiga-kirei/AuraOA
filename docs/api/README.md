# API 接口文档

## 概述

OA 智审后端基于 Go Gin 框架，所有接口统一以 `/api` 为前缀。接口按功能模块分组，通过 JWT 认证和角色中间件控制访问权限。

## 通用约定

### 基础 URL

```
http://<host>:8080/api
```

### 认证方式

除公开接口外，所有接口均需在请求头中携带 JWT Access Token：

```
Authorization: Bearer <access_token>
```

### 统一响应格式

**成功响应**：

```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

**错误响应**：

```json
{
  "code": 40001,
  "message": "错误描述"
}
```

### 角色说明

| 角色 | 编码 | 说明 |
|------|------|------|
| 系统管理员 | `system_admin` | 管理租户、OA 连接、AI 模型、系统配置 |
| 租户管理员 | `tenant_admin` | 管理组织架构、流程配置、审核规则 |
| 业务用户 | `business` | 使用审核工作台、归档复盘、个人设置 |

### 中间件链

| 中间件 | 说明 |
|--------|------|
| `JWT` | 验证 Access Token 有效性，提取用户信息 |
| `TenantContext` | 从 Token 中提取租户 ID，注入请求上下文 |
| `RequireRole(role)` | 校验当前用户是否具有指定角色 |

## 接口分组索引

| 文档 | 路由前缀 | 说明 |
|------|---------|------|
| [认证接口](./auth.md) | `/api/auth` | 登录、登出、Token 刷新、角色切换、个人信息 |
| [系统管理接口](./system-admin.md) | `/api/admin` | 租户管理、OA 连接、AI 模型、系统配置 |
| [组织架构接口](./org.md) | `/api/tenant/org` | 部门、角色、成员管理 |
| [流程审核配置接口](./audit-config.md) | `/api/tenant/rules` | 流程审核配置、审核规则、提示词模板 |
| [审核工作台接口](./audit.md) | `/api/audit` | 审核执行、任务管理、日志、快照 |
| [归档复盘接口](./archive.md) | `/api/archive` | 归档复盘执行、任务管理、日志、快照 |
| [定时任务接口](./cron.md) | `/api/tenant/cron` | 任务类型配置、任务实例管理、执行日志 |
| [用户设置接口](./user-settings.md) | `/api/tenant/settings` | 个人审核配置、归档配置、仪表盘偏好 |
| [缓存管理接口](./cache.md) | `/api/admin/cache` | 缓存统计、清除、开关 |
