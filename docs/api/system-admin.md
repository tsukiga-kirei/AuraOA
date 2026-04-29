# 系统管理接口

> 权限要求：JWT + TenantContext + `system_admin` 角色

## 租户管理

### 获取租户列表

```
GET /api/admin/tenants
```

---

### 创建租户

```
POST /api/admin/tenants
```

创建新租户并初始化租户管理员账号。

---

### 更新租户

```
PUT /api/admin/tenants/:id
```

---

### 删除租户

```
DELETE /api/admin/tenants/:id
```

需要在请求体中提供系统管理员密码进行二次确认。

---

### 获取租户统计

```
GET /api/admin/tenants/:id/stats
```

返回指定租户的成员数、审核数、归档数等统计数据。

---

### 获取租户成员列表

```
GET /api/admin/tenants/:id/members
```

---

## 系统选项（下拉框数据源）

### 获取 OA 系统类型列表

```
GET /api/admin/system/options/oa-types
```

返回支持的 OA 系统类型（泛微 E9、致远 A8+ 等）。

---

### 获取数据库驱动列表

```
GET /api/admin/system/options/db-drivers
```

返回支持的数据库驱动类型（MySQL、Oracle、PostgreSQL、SQL Server、达梦）。

---

### 获取 AI 部署类型列表

```
GET /api/admin/system/options/ai-deploy-types
```

返回 AI 部署类型（本地部署 / 云端 API）。

---

### 获取 AI 服务商列表

```
GET /api/admin/system/options/ai-providers
```

返回支持的 AI 服务商列表（Xinference、Ollama、vLLM、阿里云百炼、DeepSeek 等）。

---

## OA 数据库连接管理

### 获取 OA 连接列表

```
GET /api/admin/system/oa-connections
```

---

### 创建 OA 连接

```
POST /api/admin/system/oa-connections
```

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `name` | string | ✅ | 连接名称 |
| `oa_type` | string | ✅ | OA 系统类型编码 |
| `oa_type_label` | string | — | OA 类型显示名称 |
| `driver` | string | ✅ | 数据库驱动（mysql/oracle/dm） |
| `host` | string | ✅ | 数据库主机地址 |
| `port` | int | — | 数据库端口（默认 3306） |
| `database_name` | string | ✅ | 数据库名称 |
| `username` | string | ✅ | 数据库用户名 |
| `password` | string | ✅ | 数据库密码（加密存储） |
| `pool_size` | int | — | 连接池大小（默认 10） |
| `connection_timeout` | int | — | 连接超时秒数（默认 30） |
| `enabled` | boolean | — | 是否启用 |
| `description` | string | — | 描述 |

---

### 使用参数测试 OA 连接

```
POST /api/admin/system/oa-connections/test
```

使用临时参数测试连通性（保存前预检），请求体与创建接口相同。

---

### 更新 OA 连接

```
PUT /api/admin/system/oa-connections/:id
```

---

### 删除 OA 连接

```
DELETE /api/admin/system/oa-connections/:id
```

---

### 测试已保存的 OA 连接

```
POST /api/admin/system/oa-connections/:id/test
```

测试已保存连接的连通性，结果会更新连接状态字段。

---

## AI 模型配置管理

### 获取 AI 模型列表

```
GET /api/admin/system/ai-models
```

返回所有 AI 模型配置（API Key 不回显）。

---

### 创建 AI 模型

```
POST /api/admin/system/ai-models
```

**请求体**：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `provider` | string | ✅ | 服务商编码 |
| `provider_label` | string | — | 服务商显示名称 |
| `model_name` | string | ✅ | 模型标识名（如 `qwen2.5-72b`） |
| `display_name` | string | ✅ | 模型显示名称 |
| `deploy_type` | string | — | 部署类型（默认 `local`） |
| `endpoint` | string | — | API 端点 URL |
| `api_key` | string | — | API 密钥（加密存储） |
| `max_tokens` | int | — | 最大输出 Token（默认 8192） |
| `context_window` | int | — | 上下文窗口大小（默认 131072） |
| `cost_per_1k_tokens` | decimal | — | 每千 Token 费用 |
| `enabled` | boolean | — | 是否启用 |
| `description` | string | — | 描述 |
| `capabilities` | array | — | 能力标签列表 |

---

### 使用参数测试 AI 模型连接

```
POST /api/admin/system/ai-models/test
```

---

### 更新 AI 模型

```
PUT /api/admin/system/ai-models/:id
```

---

### 删除 AI 模型

```
DELETE /api/admin/system/ai-models/:id
```

---

### 测试已保存的 AI 模型连接

```
POST /api/admin/system/ai-models/:id/test
```

---

## 系统配置（KV）

### 获取系统配置

```
GET /api/admin/system/configs
```

返回所有系统全局配置项（键值对）。

---

### 更新系统配置

```
PUT /api/admin/system/configs
```

批量更新系统配置项。

---

## 统计与监控

### Token 消耗统计（全平台）

```
GET /api/admin/stats/token-usage
```

查询所有租户的 Token 消耗统计。

---

### 平台概览

```
GET /api/admin/dashboard-overview
```

返回平台级仪表盘数据（租户数、用户数、审核数等）。

---

### 系统监控

```
GET /api/admin/system-monitor
```

返回系统运行状态（CPU、内存、磁盘、数据库连接等）。
