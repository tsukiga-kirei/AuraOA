# 租户基础配置与执行配置版本接口

> 权限要求：JWT + TenantContext + `tenant_admin` 角色
>
> 路由前缀：`/api/tenant/execution-config-versions`

## 查询当前租户基础配置版本状态

```http
GET /api/tenant/execution-config-versions/status?module=audit&source_config_id=<配置 UUID>
```

```http
GET /api/tenant/execution-config-versions/status?module=audit&source_config_id=<配置 UUID>
```

`module` 支持 `audit`、`archive`、`summary`。该接口为纯只读查询，比对当前保存的配置指纹与当前激活版本（`is_active = true`）的指纹，用于配置页展示版本状态，不会自动递增版本号。

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "status": "current",
    "active_version_no": 3,
    "current_version_no": 3,
    "latest_version_no": 3,
    "has_pending_changes": false
  }
}
```

| `status` | 说明 |
|------|------|
| `current` | 当前配置与当前激活版本一致（`has_pending_changes: false`） |
| `updated` | 当前配置在当前激活版本上有未发布的修改（`has_pending_changes: true`） |
| `unversioned` | 尚未生成过版本 |

## 发布新租户基础配置版本

```http
POST /api/tenant/execution-config-versions/publish
```

请求 Body（JSON）：

```json
{
  "module": "audit",
  "source_config_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6"
}
```

接口将当前已保存的管理员配置固化为不可变的基础配置新版本（`version_no` 递增），并自动将新版本激活为当前可用版本（`is_active = true`）。后续新发起的审核、归档或总结流程将开始使用该新版本。若内容相较最新版本无变化，则直接激活现有最新版本。

## 查询配置历史发布版本列表

```http
GET /api/tenant/execution-config-versions/history?module=audit&source_config_id=<配置 UUID>
```

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": [
    {
      "id": "a1b2c3d4-...",
      "version_no": 3,
      "fingerprint": "...",
      "config_snapshot": { ... },
      "is_active": true,
      "created_at": "2026-09-01T17:00:00Z",
      "updated_at": "2026-09-01T17:00:00Z"
    },
    {
      "id": "e5f6a7b8-...",
      "version_no": 2,
      "fingerprint": "...",
      "config_snapshot": { ... },
      "is_active": false,
      "created_at": "2026-08-01T10:00:00Z",
      "updated_at": "2026-08-01T10:00:00Z"
    }
  ]
}
```

## 切换当前可用版本（Active Version）

```http
POST /api/tenant/execution-config-versions/activate
```

请求 Body（JSON）：

```json
{
  "module": "audit",
  "source_config_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "version_no": 2
}
```

接口将指定版本（例如 v2）切换为当前可用版本（`is_active = true`），并将其快照内容同步到主表生效视图中。后续新流程、OA 嵌入页以及个人配置将立即对齐使用该版本。

## 修改并重新保存指定版本快照

```http
POST /api/tenant/execution-config-versions/save-version
```

请求 Body（JSON）：

```json
{
  "module": "audit",
  "source_config_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "version_no": 2,
  "snapshot": { ... }
}
```

支持管理员对历史版本（例如 v2）的内容进行直接编辑并保存更新。

---

租户基础版本只记录会影响 AI 执行的数据源字段、规则、尺度、提示词、个人调整权限和启停状态，不包含流程访问名单，也不包含任何用户个人覆盖。

执行版本是另一层版本：它引用租户基础版本，再叠加执行用户当时的个人字段、个人规则、个人尺度和个人版本，形成最终不可变快照。老流程若已绑定旧执行版本，点击重新审核会继续复用原版本，新发起的流程与 OA 嵌入页面则始终使用当前激活的可用版本。



