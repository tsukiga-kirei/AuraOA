# 租户基础配置与执行配置版本接口

> 权限要求：JWT + TenantContext + `tenant_admin` 角色
>
> 路由前缀：`/api/tenant/execution-config-versions`

## 查询当前租户基础配置版本

```http
GET /api/tenant/execution-config-versions/status?module=audit&source_config_id=<配置 UUID>
```

`module` 支持 `audit`、`archive`、`summary`。接口将当前已保存的管理员配置固化为不可变的“租户基础配置版本”，用于审核工作台、归档复盘和流程总结配置页展示当前版本。该动作只保存配置版本，不绑定流程，也不会触发 AI 执行。

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
| `current` | 当前已保存配置对应 `current_version_no` 租户基础版本 |
| `updated` | 兼容旧客户端的状态值；新实现会先固化基础版本再返回 `current` |
| `unversioned` | 兼容旧客户端的状态值 |

租户基础版本只记录会影响 AI 执行的数据源字段、规则、尺度、提示词、个人调整权限和启停状态，不包含流程访问名单，也不包含任何用户个人覆盖。

执行版本是另一层版本：它引用租户基础版本，再叠加执行用户当时的个人字段、个人规则、个人尺度和个人版本，形成最终不可变快照。普通重审若流程已经绑定旧执行版本，会继续复用，不会因为管理员或个人配置变化而改写历史流程。

历史执行版本的 `base_config_version_id` 为空时保持“历史未记录”，系统不会猜测或批量回写；新流程首次执行时会自然建立完整关联。
