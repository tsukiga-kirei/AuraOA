# 租户基础配置与执行配置版本接口

> 权限要求：JWT + TenantContext + `tenant_admin` 角色
>
> 路由前缀：`/api/tenant/execution-config-versions`

## 查询当前租户基础配置版本状态

```http
GET /api/tenant/execution-config-versions/status?module=audit&source_config_id=<配置 UUID>
```

`module` 支持 `audit`、`archive`、`summary`。该接口为纯只读查询，比对当前保存的配置指纹与最新已发布版本的指纹，用于配置页展示版本状态，不会自动递增版本号。

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "status": "current",
    "current_version_no": 3,
    "latest_version_no": 3,
    "has_pending_changes": false
  }
}
```

| `status` | 说明 |
|------|------|
| `current` | 当前配置与 `current_version_no` 版本一致（`has_pending_changes: false`） |
| `updated` | 当前配置在 `current_version_no` 版本上有未发布的修改（`has_pending_changes: true`） |
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

接口将当前已保存的管理员配置固化为不可变的基础配置新版本（`version_no` 递增），后续新发起的审核、归档或总结流程将开始使用该新版本。若内容相较最新版本无变化，则直接返回现有最新版本。

响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "status": "current",
    "current_version_no": 4,
    "latest_version_no": 4,
    "has_pending_changes": false
  }
}
```

---

租户基础版本只记录会影响 AI 执行的数据源字段、规则、尺度、提示词、个人调整权限和启停状态，不包含流程访问名单，也不包含任何用户个人覆盖。

执行版本是另一层版本：它引用租户基础版本，再叠加执行用户当时的个人字段、个人规则、个人尺度和个人版本，形成最终不可变快照。普通重审若流程已经绑定旧执行版本，会继续复用，不会因为管理员或个人配置变化而改写历史流程。

