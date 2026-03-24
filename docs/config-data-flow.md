# 个人配置与租户设置：数据流转全景

> 文档版本：v1.0 | 创建日期：2026-03-23  
> 描述租户流程配置、用户个人配置在数据库存储、后端处理、前端传输三个层面的完整数据结构和流转逻辑。

---

## 一、数据库存储层

### 1.1 `process_audit_configs`（租户流程审核配置）

> 迁移文件：`db/migrations/000007_audit_configs_rules_presets.up.sql`

| 列名 | 类型 | 说明 |
|------|------|------|
| `id` | UUID PK | 配置 ID |
| `tenant_id` | UUID FK→tenants | 所属租户 |
| `process_type` | VARCHAR(200) | 流程类型标识，如 `purchase_approval` |
| `process_type_label` | VARCHAR(200) | 流程类型显示名称 |
| `main_table_name` | VARCHAR(200) | OA 主表名称，如 `formtable_main_1` |
| `main_fields` | JSONB | 主表字段配置列表 |
| `detail_tables` | JSONB | 明细子表配置列表 |
| `field_mode` | VARCHAR(20) | `all`（全字段）或 `selected`（仅配置字段） |
| `kb_mode` | VARCHAR(20) | `rules_only` 或 `hybrid` |
| `ai_config` | JSONB | AI 配置（提示词、审核尺度） |
| `user_permissions` | JSONB | 用户权限控制 |
| `access_control` | JSONB | 访问控制（角色/成员/部门） |
| `status` | VARCHAR(20) | `active` / `inactive` |

**JSONB 字段内部结构：**

**`main_fields`**（主表字段数组）：

```json
[
  { "field_key": "cgje",  "field_name": "采购金额", "field_type": "number", "selected": true  },
  { "field_key": "cgms",  "field_name": "采购描述", "field_type": "text",   "selected": false }
]
```

**`detail_tables`**（明细表数组，每张表含独立字段列表）：

```json
[
  {
    "table_name": "formtable_main_1_dt1",
    "table_label": "采购明细",
    "fields": [
      { "field_key": "wplx", "field_name": "物品类型", "field_type": "text",   "selected": true },
      { "field_key": "wpsl", "field_name": "物品数量", "field_type": "number", "selected": true }
    ]
  }
]
```

**`ai_config`**：

```json
{
  "audit_strictness": "standard",
  "system_reasoning_prompt": "你是一位专业的审计助手...",
  "system_extraction_prompt": "请严格按以下 JSON Schema 输出...",
  "user_reasoning_prompt": "请对以下 OA 流程进行审核...",
  "user_extraction_prompt": "请从推理结果中提取结构化数据..."
}
```

**`user_permissions`**：

```json
{
  "allow_custom_fields": true,
  "allow_custom_rules": true,
  "allow_modify_strictness": true
}
```

### 1.2 `audit_rules`（租户审核规则）

> 迁移文件：`db/migrations/000007_audit_configs_rules_presets.up.sql`

| 列名 | 类型 | 说明 |
|------|------|------|
| `id` | UUID PK | 规则 ID |
| `tenant_id` | UUID FK→tenants | 所属租户 |
| `config_id` | UUID FK→process_audit_configs | 所属配置（NULL 为通用规则） |
| `process_type` | VARCHAR(200) | 适用流程类型 |
| `rule_content` | TEXT | 规则内容（自然语言，直接送入 AI 提示词） |
| `rule_scope` | VARCHAR(20) | `mandatory`（强制）/ `default_on`（默认启用）/ `default_off`（默认禁用） |
| `enabled` | BOOLEAN | 租户管理员设置的启用状态 |
| `source` | VARCHAR(20) | `manual`（手动）/ `file_import`（文件导入） |
| `related_flow` | BOOLEAN | 是否关联审批流 |

### 1.3 `user_personal_configs`（用户个人配置）

> 迁移文件：`db/migrations/000010_user_personal_configs.up.sql`

| 列名 | 类型 | 说明 |
|------|------|------|
| `id` | UUID PK | 配置 ID |
| `tenant_id` | UUID FK→tenants | 所属租户 |
| `user_id` | UUID FK→users | 关联用户 |
| `audit_details` | JSONB | 审核工作台各流程的个人偏好 |
| `cron_details` | JSONB | 定时任务偏好 |
| `archive_details` | JSONB | 归档复盘各流程的个人偏好 |

**唯一约束**：`UNIQUE(tenant_id, user_id)`

**`audit_details` 内部结构**（`[]AuditDetailItem`）：

```json
[
  {
    "config_id": "uuid-of-process-config",
    "process_type": "purchase_approval",
    "field_config": {
      "field_mode": "selected",
      "field_overrides": ["main:cgms", "formtable_main_1_dt1:wpbz"]
    },
    "rule_config": {
      "rule_toggle_overrides": [
        { "rule_id": "uuid-of-tenant-rule", "enabled": false }
      ],
      "custom_rules": [
        { "id": "user-rule-1", "content": "金额超过5万需注意", "enabled": true, "related_flow": false }
      ]
    },
    "ai_config": {
      "strictness_override": "strict"
    }
  }
]
```

| 子字段 | 说明 |
|--------|------|
| `field_config.field_overrides` | 用户额外选中的字段，格式为 `table:field_key`（旧数据可能无前缀，默认归为 `main`） |
| `rule_config.rule_toggle_overrides` | 用户对租户通用规则的开关覆盖，以 `rule_id` 关联 |
| `rule_config.custom_rules` | 用户自定义的私有规则，不依赖租户实体 |
| `ai_config.strictness_override` | 用户覆盖的审核尺度（空字符串表示使用租户默认） |

---

## 二、后端处理层

### 2.1 Go Model 结构对应关系

```
数据库列                    Go Model 类型                    内嵌结构
───────────────────────────────────────────────────────────────────────
process_audit_configs
  ├─ main_fields           datatypes.JSON                  → []fieldItem (解析时局部定义)
  ├─ detail_tables         datatypes.JSON                  → []detailTableItem (解析时局部定义)
  ├─ ai_config             datatypes.JSON                  → model.AIConfigData
  ├─ user_permissions      datatypes.JSON                  → model.UserPermissionsData
  └─ access_control        datatypes.JSON                  → (无独立 struct)

user_personal_configs
  ├─ audit_details         datatypes.JSON                  → []model.AuditDetailItem
  │   ├─ field_config                                      → model.FieldConfig
  │   ├─ rule_config                                       → model.RuleConfig
  │   │   ├─ custom_rules                                  → []model.CustomRule
  │   │   └─ rule_toggle_overrides                         → []model.RuleToggleOverride
  │   └─ ai_config                                         → model.UserAIConfig
  ├─ cron_details          datatypes.JSON                  → []model.CronDetailItem
  └─ archive_details       datatypes.JSON                  → []model.ArchiveDetailItem
```

### 2.2 审核执行时的合并逻辑（`audit_execute_service.go`）

审核执行调用 `resolveUserConfig()` 进行合并，核心原则：**以租户配置为权威来源**。

#### 字段合并流程（`resolveFieldSet`）

```
┌───────────────────────┐
│ 租户 field_mode=all?  │──是──→ 返回 nil（全字段，不过滤）
└──────────┬────────────┘
           │ 否
           ▼
┌───────────────────────────────────┐
│ 解析租户 main_fields/detailTables │
│ 构建 tenantFieldIndex (有效字段) │
└──────────┬────────────────────────┘
           ▼
┌───────────────────────────────────────┐
│ AllowCustomFields = true?             │
│   是 → 解析用户 field_overrides       │
│        过滤：仅保留 tenantFieldIndex  │
│        中存在的 key                   │
│   否 → userAddedMap 为空（忽略覆盖）  │
└──────────┬────────────────────────────┘
           ▼
┌───────────────────────────────────────┐
│ 遍历租户字段列表：                    │
│   selected=true → 纳入               │
│   userAddedMap 命中 → 纳入            │
│   其余 → 排除                         │
│ 输出 SelectedFieldSet                 │
└───────────────────────────────────────┘
```

**关键保障**：

- 用户 `field_overrides` 中引用了租户已删除的字段 → `tenantFieldIndex` 中不存在 → 自动过滤
- 租户关闭 `allow_custom_fields` → 用户额外字段完全不生效
- 遍历基准始终是租户字段列表，而非用户配置

#### 规则合并流程（`resolveRulesText`）

```
┌─────────────────────────────────────┐
│ 构建 tenantRuleIDs（租户规则ID集）  │
│ 用户 toggleMap 中不在此集合的 → 丢弃 │
└──────────┬──────────────────────────┘
           ▼
┌─────────────────────────────────────────┐
│ 遍历租户规则列表：                       │
│                                          │
│   scope=mandatory → 强制写入（不看       │
│                      Enabled/用户覆盖）  │
│                                          │
│   scope=default_on/default_off：         │
│     1. 取租户 Enabled 作为默认值         │
│     2. 用户有覆盖 → 使用用户值           │
│     3. enabled=false → 跳过              │
│     4. enabled=true → 写入               │
└──────────┬──────────────────────────────┘
           ▼
┌─────────────────────────────────────────┐
│ AllowCustomRules = true?                 │
│   是 → 追加用户 custom_rules（启用的）   │
│   否 → 不追加                            │
└──────────────────────────────────────────┘
```

**关键保障**：

- 用户 `rule_toggle_overrides` 引用了已删除的规则 → `tenantRuleIDs` 中不存在 → 构建 toggleMap 时跳过
- `mandatory` 规则始终强制启用，不受 `Enabled` 字段或用户覆盖影响
- 租户关闭 `allow_custom_rules` → 用户自定义规则完全不追加
- 遍历基准始终是 `tenantRules`（从 `audit_rules` 表查出的当前有效规则）

### 2.3 前端配置读取时的合并逻辑（`user_personal_config_service.go`）

`GetFullAuditProcessConfig()` 返回前端用于展示的完整配置：

| 步骤 | 逻辑 | 说明 |
|------|------|------|
| 1 | 解析 `UserPermissions` | 确定三个 `Allow*` 开关 |
| 2 | 过滤 `rule_toggle_overrides` | 移除 `rule_id` 不在租户规则列表中的项 |
| 3 | 构建 `toggleMap` | 仅含有效规则的用户覆盖 |
| 4 | 解析字段列表 | `AllowCustomFields=true` 时合并用户 `field_overrides`；否则忽略 |
| 5 | 字段 DTO 构建 | `Locked=true`（租户选中或 all 模式）+ `Selected`（Locked 或用户额外选中） |
| 6 | 规则 DTO 构建 | `mandatory` 强制 `enabled=true`；其余应用用户覆盖 |
| 7 | 返回 `FullAuditProcessConfigResponse` | 包含合并后的字段、规则、权限、用户自定义规则 |

### 2.4 管理端用户配置查看（`user_config_management_handler.go`）

管理端在展示某用户的个人配置时，会做以下裁剪：

| 函数 | 作用 |
|------|------|
| `applyFieldOverridesPerm` | `AllowCustomFields=false` → 清空字段覆盖列表（仅影响展示） |
| `applyCustomRulesPerm` | `AllowCustomRules=false` → 清空自定义规则列表（仅影响展示） |
| `applyStrictnessPerm` | `AllowModifyStrictness=false` → 清空审核尺度覆盖（仅影响展示） |
| `filterToggleOverrides` | 1. 过滤已删除规则（`rule_id` 不在 ruleMap）<br>2. 过滤未实际修改的项（`enabled` 与管理员默认值相同） |
| `toAdminProcessDetail` | 字段覆盖标记：`abandoned`（租户已删除该字段）/ `user_added`（租户未选中但用户额外增加） |

**注意**：以上均为"读时过滤"，不写回数据库。持久化清理方案见 `docs/todo/personal-config-dirty-data-cleanup.md`。

---

## 三、前端传输层

### 3.1 API 接口

**获取完整配置**（前端个人设置页面）：

```
GET /api/user-config/audit/full/{processType}
→ FullAuditProcessConfigResponse
```

**审核执行**（审核工作台）：

```
POST /api/audit/execute
Body: { "process_id": "...", "process_type": "...", "title": "..." }
→ AuditExecuteResponse
```

### 3.2 前端接收的 DTO 结构

**`FullAuditProcessConfigResponse`**（个人设置页用）：

```typescript
interface FullAuditProcessConfig {
  process_type: string
  process_type_label: string
  config_id: string
  field_mode: string         // "all" | "selected"
  kb_mode: string
  audit_strictness: string   // 合并后的有效尺度
  user_permissions: {
    allow_custom_fields: boolean
    allow_custom_rules: boolean
    allow_modify_strictness: boolean
  }
  main_fields: TenantField[]     // 含 selected/locked 状态
  detail_tables: DetailTable[]
  tenant_rules: TenantRule[]     // 含合并后的 enabled 状态
  custom_rules: CustomRule[]
}

interface TenantField {
  field_key: string
  field_name: string
  field_type: string
  selected: boolean   // 有效选中状态（租户选中 || 用户额外选中）
  locked: boolean     // 租户预设字段，用户不可取消
}

interface TenantRule {
  id: string
  rule_content: string
  rule_scope: string   // "mandatory" | "default_on" | "default_off"
  related_flow: boolean
  enabled: boolean     // 合并后的有效启用状态
}
```

**前端行为约束**：

- `locked=true` 的字段：UI 上显示为选中且不可取消的复选框
- `allow_custom_fields=false`：隐藏"额外选择字段"的交互入口
- `scope=mandatory` 的规则：UI 上显示为强制启用且不可切换的开关
- `allow_custom_rules=false`：隐藏"添加自定义规则"的入口

### 3.3 用户保存配置时的请求体

```
PUT /api/user-config/audit/{processType}
```

```json
{
  "field_config": {
    "field_mode": "selected",
    "field_overrides": ["main:cgms", "formtable_main_1_dt1:wpbz"]
  },
  "rule_config": {
    "rule_toggle_overrides": [
      { "rule_id": "uuid", "enabled": false }
    ],
    "custom_rules": [
      { "id": "custom-1", "content": "...", "enabled": true, "related_flow": false }
    ]
  },
  "ai_config": {
    "strictness_override": "strict"
  }
}
```

保存后写入 `user_personal_configs.audit_details` 的对应流程条目中。

---

## 四、数据一致性保障机制

### 4.1 当前已实现（读时过滤）

| 层面 | 位置 | 机制 |
|------|------|------|
| 审核执行 | `resolveFieldSet` | 遍历租户字段列表，用户 override 中不存在于租户列表的 key 自动忽略 |
| 审核执行 | `resolveRulesText` | 遍历租户规则列表，用户 toggle 中不存在于租户规则的 ID 自动忽略 |
| 审核执行 | `resolveFieldSet` | 检查 `AllowCustomFields` 权限，关闭时忽略全部用户字段覆盖 |
| 审核执行 | `resolveRulesText` | 检查 `AllowCustomRules` 权限，关闭时忽略全部用户自定义规则 |
| 审核执行 | `resolveRulesText` | `mandatory` 规则始终强制启用，不受 Enabled 字段或用户覆盖影响 |
| 前端配置读取 | `GetFullAuditProcessConfig` | 同样遍历租户字段/规则，合并用户覆盖时过滤无效项 |
| 管理端查看 | `toAdminProcessDetail` | 将脏字段标记为 `abandoned`，将无效规则覆盖过滤不展示 |

### 4.2 尚未实现（持久化清理）

以下脏数据目前**只在读时过滤**，数据库 JSON 中仍保留：

| 脏数据类型 | 产生场景 | 影响 |
|-----------|---------|------|
| 过期的 `field_overrides` | 租户重新同步字段后删除了某字段 | 数据库 JSON 冗余，无运行时影响 |
| 过期的 `rule_toggle_overrides` | 租户删除了某条规则 | 数据库 JSON 冗余，无运行时影响 |
| 旧格式的 `field_overrides` | 格式从 `key` 升级为 `table:key` | `parseFieldOverride` 兼容处理，默认归为 `main` |

清理方案参见 `docs/todo/personal-config-dirty-data-cleanup.md`，建议方向为"写时清理"（用户保存配置时自动剔除无效项）。

---

## 五、数据流转总览

```
租户管理员                         用户                           AI 审核执行
    │                               │                                │
    │  配置字段/规则/权限            │  个人偏好设置                   │
    ▼                               ▼                                │
┌──────────────────┐    ┌─────────────────────────┐                  │
│ process_audit    │    │ user_personal_configs    │                  │
│ _configs         │    │                          │                  │
│ ┌──────────────┐ │    │ ┌─────────────────────┐  │                  │
│ │ main_fields  │ │    │ │ audit_details[]     │  │                  │
│ │ detail_tables│ │    │ │  ├ field_overrides   │  │                  │
│ │ field_mode   │ │    │ │  ├ rule_toggles      │  │                  │
│ │ ai_config    │ │    │ │  ├ custom_rules      │  │                  │
│ │ user_perms   │ │    │ │  └ strictness        │  │                  │
│ └──────────────┘ │    │ └─────────────────────┘  │                  │
└────────┬─────────┘    └────────────┬──────────────┘                  │
         │                           │                                │
         │  ┌────────────────┐       │                                │
         └──┤ audit_rules    ├───────┘                                │
            │ (租户规则表)    │                                        │
            └───────┬────────┘                                        │
                    │                                                 │
                    ▼                                                 │
        ┌───────────────────────────────────────┐                     │
        │         resolveUserConfig()           │◄────────────────────┘
        │                                       │
        │  1. 解析 UserPermissions              │
        │  2. resolveFieldSet()                 │
        │     - 以租户字段为基准                │
        │     - AllowCustomFields 权限检查      │
        │     - 用户 override 需在租户列表内    │
        │  3. resolveRulesText()                │
        │     - mandatory 强制启用              │
        │     - 已删除规则 toggle 自动忽略      │
        │     - AllowCustomRules 权限检查       │
        │                                       │
        │  输出: SelectedFieldSet + rulesText    │
        └───────────────────┬───────────────────┘
                            │
                            ▼
                ┌───────────────────────┐
                │ BuildReasoningPrompt  │
                │ BuildExtractionPrompt │
                │ → AI 两阶段调用       │
                └───────────────────────┘
```

---

## 六、变更记录

| 日期 | 版本 | 变更内容 |
|------|------|---------|
| 2026-03-23 | v1.0 | 初始文档，记录数据库存储、后端合并逻辑、前端传输的完整数据流 |
| 2026-03-23 | v1.0 | 修复 `resolveFieldSet` / `resolveRulesText` 权限检查缺失和 mandatory 逻辑 BUG |
