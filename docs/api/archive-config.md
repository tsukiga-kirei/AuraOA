# 归档复盘配置接口

> 权限要求：JWT + TenantContext + `tenant_admin` 角色
>
> 路由前缀：`/api/tenant/archive`

结构与 [流程审核配置](./audit-config.md) 对称。

## 归档数据源配置

### 获取配置列表

```
GET /api/tenant/archive/configs
```

---

### 创建配置

```
POST /api/tenant/archive/configs
```

`access_control.allow_all=true` 时允许当前租户内所有成员访问，不需要逐个选择。
关闭“所有人”后，`allowed_roles`、`allowed_members`、`allowed_departments` 按“任一命中即允许”解释；
三项全部为空或 JSON 无法解析时默认拒绝前台访问，`tenant_admin` 不会自动绕过此业务访问控制。

---

### 获取配置详情

```
GET /api/tenant/archive/configs/:id
```

---

### 更新配置

```
PUT /api/tenant/archive/configs/:id
```

`ai_config.system_extraction_prompt` 为后端锁定字段。保存时服务端会根据
`ai_config.audit_strictness` 使用归档系统模板覆盖客户端传值，固定 JSON Schema 不允许租户侧改写。

执行归档复盘时，附件识别遵循最终生效字段范围：`field_mode=all` 才识别全部主表附件；
选择字段模式仅下载、解析被选中的附件字段，未选附件不会调用 MinerU，也不会进入模型提示词。

---

### 删除配置

```
DELETE /api/tenant/archive/configs/:id
```

---

### 测试 OA 连接

```
POST /api/tenant/archive/configs/test-connection
```

---

### 拉取流程字段

```
POST /api/tenant/archive/configs/:id/fetch-fields
```

---

## 归档规则

### 获取规则列表

```
GET /api/tenant/archive/rules
```

**查询参数**：

| 参数 | 类型 | 说明 |
|------|------|------|
| `config_id` | uuid | 必填，归档配置 ID |
| `rule_scope` | string | 可选 |
| `enabled` | boolean | 可选 |

---

### 创建规则

```
POST /api/tenant/archive/rules
```

---

### 更新规则

```
PUT /api/tenant/archive/rules/:id
```

---

### 删除规则

```
DELETE /api/tenant/archive/rules/:id
```

---

### 批量删除规则

```
POST /api/tenant/archive/rules/batch-delete
```

请求体包含 `config_id` 与 `rule_ids`（1–5000 个规则 UUID）。仅删除当前租户、当前配置下匹配的规则，响应 `deleted_count` 为实际删除数量。文件导入、粘贴导入和手工规则均执行硬删除。

---

### 文件识别导入

归档规则与审核规则保持对称，提供以下接口：

```
GET  /api/tenant/archive/rules/import-capability
POST /api/tenant/archive/rules/import-preview
POST /api/tenant/archive/rules/import-text-preview
POST /api/tenant/archive/rules/import-confirm
```

- `import-capability`：返回附件解析是否已由系统管理员启用，以及当前实际可用的大小和类型限制。
- `import-preview`：使用 `multipart/form-data` 上传 `config_id` 与 `file`，先按文件类型路由到内置解析、MinerU 或兼容格式解析服务，再由统一 AI 调用入口返回可编辑草稿，不写库。
- `import-text-preview`：提交 `config_id` 与粘贴的 `text`，不经过 MinerU，直接返回 AI 草稿。
- `import-confirm`：提交 `config_id`、`source` 与 1–100 条确认后的草稿；`source` 支持 `file_import`、`paste_import`。

草稿字段、AI 建议值语义和外部关联数据注意事项与 [审核规则文件导入](./audit-config.md#识别文件并生成规则草稿) 相同。

---

## 外部关联数据

### 测试关联数据配置

```
POST /api/tenant/archive/context/test
```

请求体与审核侧相同：`process_id`、`context_mounts`。返回注入 AI 的 `context_text`。

建模表挂载会在 `context_text` 中显示“建模表：中文名（英文表名）”；`mode=rows` 时，`return_fields` 必须填写英文物理列名，服务端会基于泛微字段元数据将返回结果的列名转换为中文显示名。`max_rows=-1` 表示返回全部匹配行，正整数表示行数上限。

---

## 提示词模板

### 获取提示词模板列表

```
GET /api/tenant/archive/prompt-templates
```

返回系统预置的归档提示词模板（只读）。
