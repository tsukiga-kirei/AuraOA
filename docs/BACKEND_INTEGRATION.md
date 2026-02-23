# 后端集成需求文档

> OA 智能审核平台 — 前端与后端 API 对接规范

---

## 1. 租户角色与权限体系

### 1.1 角色定义

| 角色 | 标识 | 权限组 | 说明 |
|------|------|--------|------|
| 系统管理员 | `system_admin` | `['user', 'tenant', 'system']` | 可管理所有租户、系统设置、全局监控 |
| 租户管理员 | `tenant_admin` | `['user', 'tenant']` | 可管理本租户的规则配置、组织人员、数据信息 |
| 业务用户 | `business` | `['user']` | 可使用审核工作台、定时任务、归档复盘、个人设置 |

### 1.2 权限验证规则

- 每个用户归属一个租户（通过 `tenant_id` 关联）
- 用户的权限组决定其可访问的功能模块
- 系统管理员拥有所有权限，可跨租户操作
- 租户管理员在创建租户时指定（`tenant_admin_id` 字段）
- 租户管理员拥有本租户的组织管理权限（增删成员、角色分配）
- 用户可拥有多个角色/权限，支持**角色切换**（前端 Header 已实现切换按钮）

### 1.3 角色切换机制

**前端实现**：Header 组件中的 SwapOutlined 按钮，根据用户权限组动态生成可切换角色列表。

**后端需求**：
```
POST /api/auth/switch-role
Request: { "target_role": "tenant_admin" }
Response: {
  "permissions": ["user", "tenant"],
  "menus": [...],
  "default_page": "/admin/tenant"
}
```

---

## 2. OA 系统管理

### 2.1 OA 系统类型

当前版本仅支持 **泛微 Ecology E9**（`weaver_e9`），后续版本可扩展其他 OA 系统。

**数据模型**：

```typescript
interface OASystemConfig {
  id: string
  name: string               // e.g. "泛微 Ecology E9"
  type: string                // "weaver_e9" (当前唯一值)
  type_label: string
  version: string
  status: 'connected' | 'disconnected' | 'testing'
  description: string
  adapter_version: string
  last_sync: string           // 格式: "2026/2/23 12:17:04"
  sync_interval: number       // 分钟
  enabled: boolean
}
```

### 2.2 API 接口

```
GET    /api/system/oa-systems           获取 OA 系统列表
PUT    /api/system/oa-systems/:id       更新 OA 系统配置
POST   /api/system/oa-systems/:id/test  测试 OA 系统连接
```

---

## 3. AI 模型管理

### 3.1 服务商定义

| 服务商 | 标识 | 类型 | 说明 |
|--------|------|------|------|
| Xinference | `Xinference` | 本地部署 (`local`) | 本地 GPU 推理引擎 |
| 阿里云百炼 | `阿里云百炼` | 云端 API (`cloud`) | 阿里云大模型平台 |

### 3.2 数据模型

```typescript
interface AIModelConfig {
  id: string
  provider: string            // "Xinference" 或 "阿里云百炼"
  model_name: string          // e.g. "Qwen2.5-72B"
  display_name: string
  type: 'local' | 'cloud'
  endpoint: string            // API 端点地址
  api_key_configured: boolean
  max_tokens: number
  context_window: number
  cost_per_1k_tokens: number
  status: 'online' | 'offline' | 'maintenance'
  enabled: boolean
  description: string
  capabilities: string[]      // ['text', 'code', 'reasoning', 'vision', 'analysis']
}
```

### 3.3 API 接口

```
GET    /api/system/ai-models            获取模型列表
POST   /api/system/ai-models            新增模型
PUT    /api/system/ai-models/:id        更新模型配置
DELETE /api/system/ai-models/:id        删除模型
POST   /api/system/ai-models/:id/test   测试模型连接
PATCH  /api/system/ai-models/:id/toggle 启用/禁用模型
```

### 3.4 测试连接

**数据库连接测试**：
```
POST /api/system/oa-databases/:id/test-connection
Request: { jdbc_config: { driver, host, port, database, username, password } }
Response: { success: boolean, latency_ms: number, error?: string }
```

**AI 模型连接测试**：
```
POST /api/system/ai-models/:id/test-connection
Request: { endpoint: string, api_key?: string }
Response: { success: boolean, latency_ms: number, model_info?: { name, version }, error?: string }
```

---

## 4. 平台配置

### 4.1 通用配置

```typescript
interface SystemGeneralConfig {
  platform_name: string
  platform_version: string
  session_timeout: number      // 分钟
  max_upload_size: number      // MB
  enable_audit_trail: boolean
  enable_data_encryption: boolean
  backup_enabled: boolean
  backup_cron: string
  backup_retention_days: number
  notification_email: string
  smtp_host: string
  smtp_port: number
  smtp_username: string
  smtp_ssl: boolean
}
```

> 注意：`default_language` 字段已从前端配置中移除，语言由用户个人偏好控制。

### 4.2 API 接口

```
GET  /api/system/config        获取平台配置
PUT  /api/system/config        保存平台配置
```

---

## 5. 租户管理

### 5.1 数据模型

```typescript
interface TenantInfo {
  id: string
  name: string
  code: string
  oa_type: string                      // "weaver_e9"
  oa_db_connection_id: string          // 关联系统级 OA 数据库连接
  token_quota: number
  token_used: number
  max_concurrency: number
  status: 'active' | 'inactive'
  created_at: string
  contact_name: string
  contact_email: string
  contact_phone: string                // 新增：联系电话
  description: string
  tenant_admin_id?: string             // 新增：租户管理员用户ID
  ai_config: {
    default_provider: string           // "Xinference" 或 "阿里云百炼"
    default_model: string
    fallback_provider: string
    fallback_model: string
    max_tokens_per_request: number
    temperature: number
    timeout_seconds: number
    retry_count: number
  }
  log_retention_days: number
  data_retention_days: number
  allow_custom_model: boolean
  sso_enabled: boolean
  sso_endpoint: string
}
```

### 5.2 AI 模型级联关系

前端已实现 **Provider → Model 级联过滤**：

- 选择 `Xinference` 服务商 → 仅显示 `type: 'local'` 的模型
- 选择 `阿里云百炼` 服务商 → 仅显示 `type: 'cloud'` 的模型
- 切换服务商时自动清空已选模型

后端在创建/更新租户时需验证此级联关系。

### 5.3 创建租户参数

创建租户时前端提交的字段：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | ✅ | 租户名称 |
| code | string | ✅ | 租户编码 |
| oa_db_connection_id | string | ❌ | OA 数据库连接ID |
| token_quota | number | ✅ | Token 配额 |
| max_concurrency | number | ✅ | 最大并发数 |
| contact_name | string | ❌ | 联系人 |
| contact_email | string | ❌ | 联系邮箱 |
| contact_phone | string | ❌ | 联系电话 |
| description | string | ❌ | 描述 |
| ai_provider | string | ❌ | AI 服务商 |
| ai_model | string | ❌ | AI 默认模型 |

### 5.4 API 接口

```
GET    /api/system/tenants              获取租户列表
POST   /api/system/tenants              创建租户
GET    /api/system/tenants/:id          获取租户详情
PUT    /api/system/tenants/:id          更新租户配置
PATCH  /api/system/tenants/:id/status   启用/停用租户
DELETE /api/system/tenants/:id          删除租户
```

### 5.5 OA 数据库连接（系统级）

> OA 数据库连接已从租户基本信息中移除，改在独立的 "OA 数据库" Tab 中管理。
> 数据库连接在系统设置中统一创建，租户通过 `oa_db_connection_id` 引用。

```
GET    /api/system/oa-databases                获取所有 OA 数据库连接
POST   /api/system/oa-databases                新增连接
PUT    /api/system/oa-databases/:id            更新连接
DELETE /api/system/oa-databases/:id            删除连接
POST   /api/system/oa-databases/:id/test       测试连接
POST   /api/system/oa-databases/:id/sync       触发同步
```

---

## 6. 登录与认证

### 6.1 登录流程

1. 用户选择**入口**（业务用户 / 租户管理员 / 系统管理员）
2. 用户从**下拉列表**中选择租户（系统管理员无需选择）
3. 输入用户名和密码
4. 后端验证凭据并返回 Token 和权限信息

### 6.2 租户选择

前端已将租户选择改为下拉框，数据来源为租户管理列表：
- 选项格式：`租户名称（租户编码）`
- 默认选项：`默认租户`
- 系统管理员登录时不显示租户选择

### 6.3 API 接口

```
POST   /api/auth/login
Request: {
  username: string,
  password: string,
  tenant_id: string    // 租户编码或 "default"
}
Response: {
  access_token: string,
  refresh_token: string,
  expires_in: number,
  user: {
    username: string,
    display_name: string,
    tenant_id: string,
    role_label: string,
    permissions: string[]
  }
}

GET    /api/auth/tenants         获取可登录的租户列表（公开接口）
Response: [{
  id: string,
  name: string,
  code: string,
  status: 'active' | 'inactive'
}]

POST   /api/auth/switch-role     角色切换
Request: { target_role: string }
Response: {
  permissions: string[],
  menus: MenuItem[],
  default_page: string
}
```

---

## 7. 配置联动关系

### 7.1 OA 系统 → OA 数据库连接

- OA 数据库连接的 `oa_type` 字段必须匹配已启用的 OA 系统类型
- 当前仅支持 `weaver_e9`

### 7.2 OA 数据库连接 → 租户

- 租户的 `oa_db_connection_id` 引用系统级 OA 数据库连接
- 删除数据库连接前需检查是否有租户引用

### 7.3 AI 模型 → 租户 AI 配置

- 租户的 `ai_config.default_provider` 决定可选模型范围
- `Xinference` → 仅 `type: 'local'` 模型
- `阿里云百炼` → 仅 `type: 'cloud'` 模型
- 禁用模型前需检查是否有租户正在使用

### 7.4 租户 → 租户管理员

- `tenant_admin_id` 关联系统用户
- 该用户被赋予该租户的管理权限
- 租户管理员可管理本租户的组织人员、规则配置等

---

## 8. 日期格式规范

所有时间字段统一使用以下格式：

```
YYYY/M/D H:mm:ss
```

示例：`2026/2/23 12:17:04`

---

## 9. 数据库类型支持

当前版本仅支持以下数据库驱动：

| 驱动 | 标识 | 默认端口 |
|------|------|----------|
| MySQL | `mysql` | 3306 |
| Oracle | `oracle` | 1521 |

> 已移除 PostgreSQL 和 SQL Server 支持。
