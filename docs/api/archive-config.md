# 归档复盘配置接口

## 字段空值传输策略

归档与审核使用相同的 `ai_config.skip_empty_fields`（统一开关，默认 `false`）及
`ai_config.skip_empty_field_keys`（单字段引用数组，默认 `[]`），字段引用为 `main:字段标识`
或 `明细表名:字段标识`，大小写不敏感。

统一开启时，当前传输范围内所有 `null`、空字符串或纯空白字段均不传给 AI；统一关闭时
恢复各字段的单独设置。单字段开启时跳过空值，关闭时空值也传；`0`、`false` 及字符串 `"null"` 保留。
全部字段模式也应用该策略；OA 回显包装按 `value` 原始值判断，明细逐行过滤且保留原行数。
仅过滤提示词字段副本，不修改原始 OA 数据、业务快照或外部关联查询。
配置读写、导入导出、发布和执行快照保留这两个属性；沿用现有版本生命周期与旧流程绑定规则，
旧配置未设置时保持传输空值，个人复核尺度调整保留策略。详见[审核配置](./audit-config.md#更新配置)。

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

`ai_config.enable_thinking` 控制是否开启深度思考模式（需模型本身支持思考）。
`ai_config.system_extraction_prompt` 与 `ai_config.user_extraction_prompt` 均为后端锁定字段。
保存时服务端会根据 `ai_config.audit_strictness` 使用对应归档系统模板覆盖客户端传值，
固定 JSON Schema、输出指令和变量结构不允许租户侧改写。

执行归档复盘时，附件识别遵循最终生效字段范围：`field_mode=all` 才识别全部主表附件；
选择字段模式仅下载、解析被选中的附件字段，未选附件不会调用 MinerU，也不会进入模型提示词。

配置保存不会自动重建既有归档复盘结果。每个流程首次复盘时锁定最终生效配置版本，普通再次复盘
继续使用原版本；只有用户明确选择使用最新配置时才升级流程绑定。

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

规则新增可选字段 `applicability_condition`（适用前提，字符串，最多 5000 字符），列表与创建/更新响应均返回该字段。

- 留空或未设置：沿用原有逻辑，直接校验规则正文。
- 满足前提：校验 `rule_content`，按实际结果判定通过或不通过。
- 前提明确不满足：直接通过（`passed=true`），原因说明前提不满足，不扣分、不增加风险；强制执行规则也遵循此逻辑。
- 判断前提的证据缺失不等于前提不满足，按既有审核尺度处理。

创建时省略默认为空；更新时省略保持原值，传 `""` 可清空。结果仍只有通过/不通过，不增加状态。
字段同步保留在配置导入导出、草稿、发布版本和执行快照中；旧配置没有此字段时视为空。

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
- `import-preview`：使用 `multipart/form-data` 上传 `config_id` 与 `file`，先按文件类型和管理员选择路由到内置解析、代码文档解析服务或 MinerU，再由统一 AI 调用入口返回可编辑草稿，不写库。
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

## 数据记忆引用

规则读写和执行配置快照新增 `data_queries` 数组，引用当前租户查询库的不可变发布版本，最多8项。与 `related_flow` / `context_mounts` 独立；支持流程字段、系统变量和固定值参数绑定。详情见 [记忆库 / 数据记忆](./audit-data-queries.md)。

修改引用需要查询库页面权限；无权限者可原样保留已有引用。引用的版本不会随查询草稿变化，变更引用后按既有执行配置发布/激活机制生效。

## 适用前提与 AI 提示词

规则文本按正文与前提成组注入。推理、结构化提取两个阶段均补充统一前提语义，兼容租户已有自定义提示词；系统默认模板的严格、标准、宽松尺度同步更新。
文件识别和粘贴生成草稿从原文提取 `applicability_condition`，无明确前提时返回空字符串，不得臆造；审批流、外部数据依赖同时考虑判断前提和校验正文所需的数据。预览时可编辑或清空前提。


## 反馈记忆提示词变量

审核与归档两阶段提示词支持 `{{feedback_memory}}`。仅使用当前租户、当前业务模块和流程类型的已发布反馈记忆，未放置变量的旧模板自动追加；同一次两阶段共享记忆版本。反馈作为运行时参考资料，与流程执行配置版本独立，不能覆盖正式强制规则。实际记忆版本保留在 AI 调用提示词中，详见 [评论反馈记忆库](./feedback-memory.md)。

### 草稿恢复与版本提示

已发布配置支持[丢弃草稿](./execution-config.md#丢弃当前草稿)，恢复当前发布可用版本，不删除发布历史或执行记录。草稿是否修改按规范化后的实际内容判断；嵌入页是否有版本可用则只比较执行所依据的基础版本号与当前发布可用版本号是否不同，不比较内容或版本大小。
