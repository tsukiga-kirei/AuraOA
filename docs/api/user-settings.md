# 用户设置接口

> 权限要求：JWT + TenantContext（无角色限制，所有已登录租户用户均可访问）
>
> 路由前缀：`/api/tenant/settings`

## 审核工作台个人配置

### 获取可用流程列表

```
GET /api/tenant/settings/processes
```

返回当前用户可见的已配置流程类型列表（用于筛选下拉框）。

---

### 获取流程个人配置

```
GET /api/tenant/settings/processes/:processType
```

返回指定流程类型的用户个人配置（字段覆盖、规则开关、AI 尺度偏好）。

---

### 更新流程个人配置

```
PUT /api/tenant/settings/processes/:processType
```

**请求体**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `config_id` | string | 流程审核配置 ID |
| `field_config` | object | 字段配置（`field_mode`、`field_overrides`） |
| `rule_config` | object | 规则配置（`custom_rules`、`rule_toggle_overrides`） |
| `ai_config` | object | AI 配置（`strictness_override`） |

---

### 获取完整流程配置（合并视图）

```
GET /api/tenant/settings/processes/:processType/full
```

返回租户配置 + 用户覆盖合并后的完整配置，包括：

- 主表字段（含选中状态和锁定状态）
- 明细表字段
- 租户规则（含用户开关覆盖后的有效状态）
- 用户自定义规则
- 用户权限（是否允许自定义字段/规则/尺度）
- 有效审核严格度

---

## 定时任务个人偏好

### 获取定时任务偏好

```
GET /api/tenant/settings/cron-prefs
```

返回用户的定时任务个人偏好（如默认推送邮箱）。

---

### 更新定时任务偏好

```
PUT /api/tenant/settings/cron-prefs
```

**请求体**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `default_email` | string | 默认推送邮箱 |

---

## 归档复盘个人配置

### 获取可用归档配置列表

```
GET /api/tenant/settings/archive-configs
```

返回当前用户可见的已配置归档流程类型列表。

---

### 获取完整归档配置（合并视图）

```
GET /api/tenant/settings/archive-configs/:processType/full
```

返回租户归档配置 + 用户覆盖合并后的完整配置。

---

### 更新归档个人配置

```
PUT /api/tenant/settings/archive-configs/:processType
```

---

## 仪表盘偏好

### 获取仪表盘偏好

```
GET /api/tenant/settings/dashboard-prefs
```

返回用户的仪表盘 Widget 启用状态和尺寸配置。

---

### 更新仪表盘偏好

```
PUT /api/tenant/settings/dashboard-prefs
```

**请求体**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `enabled_widgets` | JSON | 启用的 Widget 列表 |
| `widget_sizes` | JSON | Widget 尺寸配置 |

---

## 仪表盘聚合数据

### 获取仪表盘概览

```
GET /api/tenant/settings/dashboard-overview
```

返回当前用户的仪表盘聚合数据。

---

## 用户配置管理（JWT + TenantContext + `tenant_admin`）

> 路由前缀：`/api/tenant/user-configs`

租户管理员可集中查看和管理所有用户的个人配置。

### 获取用户配置列表

```
GET /api/tenant/user-configs
```

---

### 导出用户配置

```
GET /api/tenant/user-configs/export
```

---

### 获取指定用户配置

```
GET /api/tenant/user-configs/:userId
```

---

## Token 消耗统计（JWT + TenantContext + `tenant_admin`）

### 查询租户 Token 消耗

```
GET /api/tenant/stats/token-usage
```

查询当前租户的 Token 消耗统计数据。
