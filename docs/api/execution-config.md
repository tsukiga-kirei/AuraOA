# 租户基础配置与执行配置版本接口

> 权限要求：JWT + TenantContext + `tenant_admin` 角色
>
> 路由前缀：`/api/tenant/execution-config-versions`

## 查询当前租户基础配置版本状态

```http
GET /api/tenant/execution-config-versions/status?module=audit&source_config_id=<配置 UUID>
```

`module` 支持 `audit`、`archive`、`summary`。该接口为纯只读查询，按相同规则规范化当前保存的配置及当前激活版本（`is_active = true`）后比较实际内容，用于配置页展示版本状态，不会自动递增版本号。

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
| `unversioned` | 当前仅有草稿，流程尚未启用；首次发布后生成 v1 |

## 保存当前配置草稿

```http
POST /api/tenant/execution-config-versions/save-draft
```

请求 Body（JSON）：

```json
{
  "module": "audit",
  "source_config_id": "3fa85f64-5717-4562-b3fc-2c963f66afa6",
  "snapshot": {
    "process_type": "contract",
    "main_table_name": "formtable_main_1",
    "main_fields": [],
    "detail_tables": [],
    "field_mode": "all",
    "kb_mode": "rules_only",
    "ai_config": {},
    "user_permissions": {},
    "status": "active",
    "rules": []
  }
}
```

接口将页面中已确认的配置和规则写回当前工作草稿，但不会创建或更新发布版本。首次发布前，配置始终保持 `unversioned`，不会进入审核、归档复盘、流程总结工作台或 OA 嵌入执行；已有发布版本时，只有内容发生实质修改才返回 `updated`；原样保存返回 `current`。线上仍继续使用当前激活版本。

## 丢弃当前草稿

```http
POST /api/tenant/execution-config-versions/discard-draft
```

请求：`{"module":"audit","source_config_id":"<配置 UUID>"}`。三个模块均支持。

接口在事务内将工作草稿恢复为当前发布可用版本的内容，返回与状态查询相同的响应。不会删除配置、历史发布版本或执行记录，也不会改变流程绑定、访问名单和嵌入开关。尚无发布版本或源配置不存在时返回 `ErrConfigNotFound`（404）。配置恢复失败时事务回滚。事务提交后同步缓存与审核/总结定时检查调度；若调度同步失败，返回明确的“配置已保存，但定时检查调度同步失败”错误，配置恢复仍已提交。

配置页还会清除当前配置尚未保存的页面编辑。草稿差异比较统一处理 JSON 键顺序、缺省字段与空数组；规则创建/更新时间不参与比较。历史快照读取时补齐缺省值，不重写历史内容和指纹。

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

接口将当前已保存的管理员草稿固化为基础配置新版本，并自动将新版本激活为当前可用版本（`is_active = true`）。首次发布生成 v1；后续草稿有变化时发布才递增 `version_no`。后续新发起的审核、归档或总结流程将开始使用该新版本。若内容与已有发布版本相同，则直接复用并激活该版本，优先保留当前可用版本；不会因默认值补齐或 JSON 表达变化生成重复版本。

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

## 基于指定历史版本另存草稿（兼容接口）

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

该兼容接口校验指定版本属于当前租户与配置后，将提交内容另存为工作草稿，不再修改历史发布版本；“基于此版本编辑草稿”会把历史快照载入工作草稿，管理员保存草稿后再发布为新版本。

---

租户基础版本只记录会影响 AI 执行的数据源字段、规则、尺度、提示词、个人调整权限和启停状态，不包含流程访问名单，也不包含任何用户个人覆盖。

执行版本是另一层版本：它引用租户基础版本，再叠加执行用户当时的个人字段、个人规则、个人尺度和个人版本，形成最终不可变快照。老流程若已绑定旧执行版本，点击重新审核会继续复用原版本，新发起的流程使用当前激活的可用版本，已有 OA 嵌入流程仍沿用绑定版本，显式选择当前发布配置重新执行才切换。

在对外接口（如审核/复盘/总结链路与嵌入页面上下文）中，`config_version_no` 统一映射展示为对应的租户基础版本号（即管理端规则配置中展示的当前可用版本号，如 v10），保持业务人员心智模型一致；底层不可变执行快照的精确溯源与审计继续由 `config_version_id` UUID 严格保障。



审核与归档快照的每条规则支持可选 `applicability_condition`。草稿保存、版本发布/激活、历史版本保存与执行配置恢复均保留此前提；省略或为空时沿用旧规则行为，单独修改前提会形成配置内容变更。

嵌入页 `config_upgrade_available` 只比较流程执行版本所依据的租户基础版本号与当前发布可用版本号：不同即提示，不比较内容，也不判断版本号大小。两版本内容相同但编号不同仍提示；切换回较小编号同样提示。相同编号不会因个人配置、规则时间戳或 JSON 表达差异提示。`use_latest_config=true` 使用当前发布可用版本，草稿与未激活的高编号历史版本不参与选择。
