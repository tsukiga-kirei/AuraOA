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

`access_control.allow_all=true` 时允许当前租户内所有成员访问，不需要逐个选择。
关闭“所有人”后，`allowed_roles`、`allowed_members`、`allowed_departments` 按“任一命中即允许”解释；
三项全部为空或 JSON 无法解析时默认拒绝前台访问，`tenant_admin` 不会自动绕过此业务访问控制。

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

`ai_config.enable_thinking` 控制是否开启深度思考模式（需模型本身支持思考）。
`ai_config.system_extraction_prompt` 与 `ai_config.user_extraction_prompt` 均为后端锁定字段。
保存时服务端会根据 `ai_config.audit_strictness` 使用对应系统模板覆盖客户端传值，
避免固定 JSON Schema、输出指令和变量结构被误改；
推理阶段的系统提示词和用户提示词仍按现有配置维护。

`embed_config` 控制 OA 嵌入审核的自动刷新策略：

| 字段 | 默认值 | 说明 |
|------|--------|------|
| `auto_audit_on_open` | `true` | 没有历史审核结果时自动审核 |
| `auto_audit_on_data_change` | `true` | 审核实际使用的主表、明细或附件版本变化后自动审核 |
| `auto_audit_on_return_resubmit` | `true` | 发生退回或退回后的重新提交时自动审核 |
| `auto_audit_on_flow_change` | `false` | 普通批准、批注、转发、抄送或节点推进后自动审核 |
| `scheduled_refresh_enabled` | `false` | 是否定时拉取该流程类型的近期实例并检查变化 |
| `scheduled_refresh_lookback_days` | `3` | 拉取近几天创建的流程，范围 1–30 |
| `scheduled_refresh_interval_minutes` | `5` | 检查频率，支持 5、10、15、30、60 分钟 |

保存配置后，系统会立即同步对应的 `embed_refresh_schedules` 持久化调度记录；关闭定时检查
会停用并移除内存 Cron，不需要等待配置轮询。定时任务执行前还会重新核验审核配置的状态、
`embed_enabled` 和 `scheduled_refresh_enabled`，源配置未明确开启时不会访问 OA 数据库。

普通审批推进默认不重新调用 AI，以减少等待时间和 Token 消耗。附件版本通过
`DocImageFile` 的最新 `versionid` / `imagefileid` 与附件字段 `docId` 共同判断。
流程级定时检查只发现候选流程，已有结果且指纹未变化时不会创建审核或 LLM 日志。

规则、审核尺度、提示词和字段配置属于执行配置，不属于 OA 业务数据。保存这些配置只会形成后续
新流程可用的新内容版本，不会触发已绑定老流程自动审核。旧流程默认继续使用首次执行时锁定的
最终生效配置；如需升级，必须由用户在审核页面明确选择“使用最新配置重新执行”。

执行审核时，附件识别遵循最终生效字段范围：`field_mode=all` 才识别全部主表附件；
选择字段模式仅下载、解析被选中的附件字段，未选附件不会调用 MinerU，也不会进入模型提示词。

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

### 批量删除规则

```
POST /api/tenant/rules/audit-rules/batch-delete
```

请求体：

```json
{
  "config_id": "配置 UUID",
  "rule_ids": ["规则 UUID 1", "规则 UUID 2"]
}
```

仅删除当前租户、当前配置下匹配的规则，一次支持 1–5000 条；响应 `deleted_count` 为实际删除数量。

---

### 查询文件识别导入能力

```
GET /api/tenant/rules/audit-rules/import-capability
```

仅当系统管理员开启 `attachment.recognition_enabled`，且当前至少有一种文件类型具备可用解析路径时，`data.enabled` 才为 `true`。响应同时返回 `max_file_size_mb`、按解析器开关过滤后的 `supported_types` 与不可用原因 `reason`。

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

服务端先按文件类型路由到内置解析、MinerU 或兼容格式解析服务，再通过 `AIModelCallerService` 生成结构化规则草稿。此接口**不写入规则库**。每条草稿包含 `rule_content`、`rule_scope`、`related_flow`、`context_recommended`、`confidence`、`reasoning`。

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
