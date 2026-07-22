# 流程审核配置接口

> 权限要求：JWT + TenantContext + `tenant_admin` 角色
>
> 路由前缀：`/api/tenant/rules`

## 流程审核配置

### 获取配置列表

```
GET /api/tenant/rules/configs
```

返回当前租户的所有流程审核配置。

---

### 创建配置

```
POST /api/tenant/rules/configs
```

为指定流程类型创建审核配置（字段选择、AI 参数、权限控制等）。

---

### 获取配置详情

```
GET /api/tenant/rules/configs/:id
```

---

### 更新配置

```
PUT /api/tenant/rules/configs/:id
```

---

### 删除配置

```
DELETE /api/tenant/rules/configs/:id
```

---

### 测试 OA 连接

```
POST /api/tenant/rules/configs/test-connection
```

在配置流程时测试 OA 数据库连通性。

---

### 拉取流程字段

```
POST /api/tenant/rules/configs/:id/fetch-fields
```

从 OA 系统拉取指定流程的全部字段定义（主表 + 明细表），用于配置字段选择。

---

## 审核规则

### 获取规则列表

```
GET /api/tenant/rules/audit-rules
```

返回当前租户的所有审核规则。

---

### 创建规则

```
POST /api/tenant/rules/audit-rules
```

---

### 更新规则

```
PUT /api/tenant/rules/audit-rules/:id
```

---

### 删除规则

```
DELETE /api/tenant/rules/audit-rules/:id
```

---

### 查询文件识别导入能力

```
GET /api/tenant/rules/audit-rules/import-capability
```

仅当系统管理员开启 `attachment.recognition_enabled` 且配置 MinerU 地址时，`data.enabled` 才为 `true`。响应同时返回 `max_file_size_mb`、`supported_types` 与不可用原因 `reason`。

---

### 识别文件并生成规则草稿

```
POST /api/tenant/rules/audit-rules/import-preview
Content-Type: multipart/form-data
```

| 表单字段 | 类型 | 说明 |
|------|------|------|
| `config_id` | uuid | 必填，当前流程审核配置 ID |
| `file` | file | 必填，类型与大小受系统附件识别配置限制 |

服务端先调用 MinerU 识别文件，再通过 `AIModelCallerService` 生成结构化规则草稿。此接口**不写入规则库**。每条草稿包含 `rule_content`、`rule_scope`、`related_flow`、`context_recommended`、`confidence`、`reasoning`。

AI 对规则级别、审批流依赖与外部数据依赖的判断均为建议值。判断外部数据依赖时会同时参考当前流程已配置的主表与明细字段；`context_recommended=true` 只表示规则可能需要当前表单和审批历史之外的数据，不会直接设置 `context_enabled`，也不会虚构 `context_mounts`。只有管理员在规则编辑器中配置至少一个启用的挂载后，规则才会显示并启用“外部关联”。

---

### 粘贴文本并生成规则草稿

```
POST /api/tenant/rules/audit-rules/import-text-preview
```

请求体：

```json
{
  "config_id": "配置 UUID",
  "text": "需要拆分为规则的制度或条款文本"
}
```

粘贴导入不经过 MinerU，因此不依赖附件识别开关；它仍使用租户主用/备用 AI 模型，并进入 Token 配额和 AI 调用日志。最多分析 120000 个字符，返回结构与文件识别预览一致。

---

### 确认批量导入规则

```
POST /api/tenant/rules/audit-rules/import-confirm
```

请求体：

```json
{
  "config_id": "配置 UUID",
  "source": "file_import",
  "rules": [
    {
      "rule_content": "合同金额不得超过已批准预算",
      "rule_scope": "mandatory",
      "related_flow": false,
      "context_enabled": false,
      "context_recommended": true,
      "confidence": 0.92,
      "reasoning": "原文使用不得，且需要查询预算数据"
    }
  ]
}
```

每次最多 100 条。`source` 可为 `file_import` 或 `paste_import`；文件导入会重新校验 MinerU 开关，粘贴导入不受该开关限制。`mandatory`、`default_on` 默认启用，`default_off` 默认关闭。与当前配置下既有规则内容完全重复的项会被跳过；若全部重复则返回冲突错误。

---

## 提示词模板

### 获取提示词模板列表

```
GET /api/tenant/rules/prompt-templates
```

返回系统预置的提示词模板（只读），用于配置流程审核时选择提示词模板。

---

## 外部关联数据

### 测试关联数据配置

```
POST /api/tenant/rules/context/test
```

**请求体**：

| 字段 | 类型 | 说明 |
|------|------|------|
| `process_id` | string | OA 流程 ID |
| `context_mounts` | object | 外部关联数据挂载配置 |

**响应**：`data.context_text` 为注入 AI 的文本预览。

建模表挂载会在 `context_text` 中显示“建模表：中文名（英文表名）”；`mode=rows` 时，`return_fields` 必须填写英文物理列名，服务端会基于泛微字段元数据将返回结果的列名转换为中文显示名。`max_rows=-1` 表示返回全部匹配行，正整数表示行数上限。
