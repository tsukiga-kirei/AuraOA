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
| `base_config_version` | integer | 页面加载时的当前租户基础版本；保存时用于阻止覆盖管理员的新配置 |
| `personal_version` | integer | 页面加载时的个人配置版本；保存时用于阻止多页面互相覆盖 |
| `field_config` | object | 字段配置（`field_mode`、`field_overrides`） |
| `rule_config` | object | 规则配置（`custom_rules`、`rule_toggle_overrides`） |
| `ai_config` | object | AI 配置（`strictness_override`） |

保存成功后个人版本递增。若租户基础版本或个人版本已经变化，接口返回 HTTP `409`，前端应刷新合并视图后由用户重新确认保存。

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
- `base_config_version`：当前个人配置原先基于的租户版本
- `current_base_config_version`：管理员当前租户配置版本
- `personal_version`：个人配置版本

每条个人规则还返回 `base_config_version` 和 `added_in_personal_version`，用于说明它基于哪个租户版本、在哪个个人版本首次加入。历史个人规则没有版本元数据时显示为未记录，首次重新保存后补齐。

当管理员同时开启流程访问权限和 `allow_modify_strictness` 时，个人尺度可以调整。执行端会按个人尺度装配完整的推理与结构化提示词；未授权时忽略个人尺度并使用租户配置。

---

### 获取审核流程基线版本差异（Diff）

```
GET /api/tenant/settings/processes/:processType/version-diff?from_version=1&to_version=2
```

对比租户配置在两个版本之间的基准变更（新增规则、删除规则、修改规则、新增/删除字段、审核尺度调整等）。若不提供参数，默认对比用户当前基于版本 `base_config_version` 与租户最新版本 `current_base_config_version`。

响应示例：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "process_type": "leave_request",
    "from_version_no": 1,
    "to_version_no": 2,
    "added_rules": [
      { "id": "TR-102", "rule_content": "请假超过3天须上传病假证明", "rule_scope": "mandatory" }
    ],
    "removed_rules": [],
    "modified_rules": [
      { "id": "TR-101", "rule_content": "请假理由字数不少于15字", "rule_scope": "default_on", "change_desc": "规则内容已更新" }
    ],
    "added_fields": [
      { "table": "main", "field_key": "hospital_cert", "field_name": "医院证明附件" }
    ],
    "removed_fields": [],
    "strictness_from": "normal",
    "strictness_to": "strict",
    "total_changes": 3
  }
}
```

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

请求和版本冲突语义与审核工作台一致；个人复核尺度也只有在管理员允许时才会进入最终执行快照。

---

## 流程总结个人展示偏好

个人设置只控制流程总结工作台中哪些总结块可见，不修改租户总结块、提示词、字段范围、
深度思考开关或实际生成内容，避免个人偏好改变共享流程级总结快照。

### 获取可用流程总结配置

```
GET /api/tenant/settings/summary-configs
```

返回租户内状态为 `active` 的流程总结配置简表。

### 获取完整个人展示偏好

```
GET /api/tenant/settings/summary-configs/:processType/full
```

返回流程配置 ID、流程类型、流程名称和启用的总结块。每个块包含 `id`、`title`、
`visible` 和租户配置的 `enable_thinking`。

### 更新个人展示偏好

```
PUT /api/tenant/settings/summary-configs/:processType
```

**请求体**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `config_id` | string | 流程总结配置 ID，须与路径流程类型匹配 |
| `visible_block_ids` | string[] | 当前用户要展示的启用总结块 ID，至少一项 |

服务端会拒绝已停用、已删除或不属于该配置的总结块 ID。

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

## OA 流程跳转配置

### 获取当前租户 OA 流程跳转配置

```
GET /api/tenant/settings/oa-jump-config
```

返回当前租户关联 OA 连接的跳转配置。若未关联 OA 库或未配置 Web 访问地址，则 `enabled` 为 `false`。

**响应示例**：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "enabled": true,
    "oa_base_url": "http://oa.example.com:8088",
    "process_url_template": "/workflow/request/ViewRequestForwardSPA.jsp?requestid={process_id}",
    "resolved_template": "http://oa.example.com:8088/workflow/request/ViewRequestForwardSPA.jsp?requestid={process_id}"
  }
}
```

---

## 仪表盘聚合数据

### 获取仪表盘概览

```
GET /api/tenant/settings/dashboard-overview
```

返回当前用户的仪表盘聚合数据。本周概览、每日趋势与最近动态包含
`summary_count` / `kind=summary`，普通业务用户只统计本人从流程总结工作台完成的记录；
租户管理员查看租户级汇总。OA 嵌入页自动或手动生成的总结不计入个人工作台指标。

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
