# 数据库设计文档

> 文档版本：v1.0 | 创建日期：2026-03-19
> 系统共包含 **25 张数据表**，按照迁移脚本顺序（000001 ~ 000011）分组说明。

---

## 一、数据库总览

| 分组 | 表数量 | 表名 |
|------|--------|------|
| 基础扩展 | 1 | `pg_extensions`（pgvector 等） |
| 租户与用户 | 4 | `tenants`, `users`, `user_role_assignments`, `login_history` |
| 组织架构 | 4 | `departments`, `org_roles`, `org_members`, `org_member_roles` |
| 系统配置 | 1 | `system_configs` |
| 选项与集成 | 6 | `oa_type_options`, `db_driver_options`, `ai_deploy_type_options`, `ai_provider_options`, `oa_database_connections`, `ai_model_configs` |
| 审核配置与规则 | 5 | `process_audit_configs`, `audit_rules`, `system_prompt_templates`, `process_archive_configs`, `archive_rules` |
| 定时任务 | 3 | `cron_tasks`, `cron_task_type_presets`, `cron_task_type_configs` |
| 日志 | 3 | `audit_logs`, `cron_logs`, `archive_logs` |
| 用户个性化 | 2 | `user_personal_configs`, `user_dashboard_prefs` |
| Token 追踪 | 1 | `tenant_llm_message_logs` |

---

## 二、租户与用户（Migration 002）

### 2.1 tenants — 租户表

> 系统的多租户核心表，每个租户代表一个独立的企业/组织。

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID PK | 主键 |
| `name` | VARCHAR(255) | 租户名称 |
| `code` | VARCHAR(100) UNIQUE | 租户唯一编码（用于 URL 路由标识） |
| `description` | TEXT | 租户描述 |
| `status` | VARCHAR(20) | 状态：`active`/`inactive`/`suspended` |
| `oa_db_connection_id` | UUID FK | 关联的 OA 数据库连接ID |
| `admin_user_id` | UUID FK | 租户管理员用户ID（Migration 006 新增） |
| `token_quota` | INT | Token 总配额 |
| `token_used` | INT | 已消耗 Token |
| `max_concurrency` | INT | 最大并发 AI 审核请求数 |
| `primary_model_id` | UUID FK | 主用 AI 模型ID（Migration 005 新增） |
| `fallback_model_id` | UUID FK | 备用 AI 模型ID（Migration 005 新增） |
| `max_tokens_per_request` | INT | 单次审核最大 Token 限制 |
| `temperature` | DECIMAL(3,2) | AI 温度参数（默认 0.30） |
| `timeout_seconds` | INT | AI 请求超时（默认 60s） |
| `retry_count` | INT | AI 请求重试次数（默认 3） |
| `sso_enabled` | BOOLEAN | 是否启用 SSO |
| `sso_endpoint` | VARCHAR(500) | SSO 接口地址 |
| `log_retention_days` | INT | 操作日志保留天数 |
| `data_retention_days` | INT | 审核数据保留天数 |
| `contact_name/email/phone` | - | 联系人信息 |
| `created_at` / `updated_at` | TIMESTAMPTZ | 时间戳 |

### 2.2 users — 用户表

> 全平台用户账号表，通过 `user_role_assignments` 关联租户。

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID PK | 主键 |
| `username` | VARCHAR(100) UNIQUE | 登录用户名（全局唯一） |
| `password_hash` | VARCHAR(255) | bcrypt 密码哈希 |
| `display_name` | VARCHAR(100) | 显示名称 |
| `email` | VARCHAR(255) | 邮箱 |
| `phone` | VARCHAR(50) | 手机号 |
| `avatar_url` | VARCHAR(500) | 头像 URL |
| `status` | VARCHAR(20) | 状态：`active`/`inactive`/`locked` |
| `password_changed_at` | TIMESTAMPTZ | 最后修改密码时间 |
| `login_fail_count` | INT | 连续登录失败次数 |
| `locked_until` | TIMESTAMPTZ | 锁定截止时间（NULL=未锁定） |
| `locale` | VARCHAR(10) | 用户语言偏好（默认 `zh-CN`） |

### 2.3 user_role_assignments — 用户角色分配表

> 用户与系统角色的关联表。一个用户可以在多个租户中有不同角色。

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID PK | 主键 |
| `user_id` | UUID FK → users | 用户ID |
| `role` | VARCHAR(30) | 角色：`business`/`tenant_admin`/`system_admin` |
| `tenant_id` | UUID FK → tenants | 租户ID（`system_admin` 时为 NULL） |
| `label` | VARCHAR(200) | 角色显示标签 |
| `is_default` | BOOLEAN | 是否为默认角色/租户 |

### 2.4 login_history — 登录历史表

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID PK | 主键 |
| `user_id` | UUID FK | 用户ID |
| `tenant_id` | UUID FK | 登录租户（NULL=系统管理员登录） |
| `ip` | VARCHAR(45) | 客户端IP（支持 IPv6） |
| `user_agent` | VARCHAR(500) | 客户端标识 |
| `login_at` | TIMESTAMPTZ | 登录时间 |

---

## 三、组织架构（Migration 003）

### 3.1 departments — 部门表

> 树形结构，通过 `parent_id` 自引用实现多级部门。

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID PK | 主键 |
| `tenant_id` | UUID FK | 所属租户ID |
| `name` | VARCHAR(200) | 部门名称 |
| `parent_id` | UUID FK → departments | 父级部门（NULL=顶级） |
| `manager` | VARCHAR(100) | 部门负责人 |
| `sort_order` | INT | 排序权重 |

### 3.2 org_roles — 组织角色表

> 租户内的自定义角色，用于页面权限控制。

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID PK | 主键 |
| `tenant_id` | UUID FK | 所属租户 |
| `name` | VARCHAR(100) | 角色名称 |
| `description` | TEXT | 描述 |
| `page_permissions` | JSONB | 页面权限列表 |
| `is_system` | BOOLEAN | 是否系统内置（不可删除） |

### 3.3 org_members — 组织成员表

> 用户与部门的归属关系。同一租户内用户唯一。

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID PK | 主键 |
| `tenant_id` | UUID FK | 所属租户 |
| `user_id` | UUID FK | 用户ID |
| `department_id` | UUID FK | 所属部门 |
| `position` | VARCHAR(100) | 职位 |
| `status` | VARCHAR(20) | 状态：active/inactive |

**唯一约束**：`(tenant_id, user_id)` — 同一租户用户只能属于一个部门。

### 3.4 org_member_roles — 成员角色关联表

> 多对多关联表，一个成员可分配多个组织角色。

| 字段 | 类型 | 说明 |
|------|------|------|
| `org_member_id` | UUID FK | 组织成员ID |
| `org_role_id` | UUID FK | 组织角色ID |

---

## 四、系统配置（Migration 004）

### 4.1 system_configs — 系统全局配置表

> KV 配置表，存储系统级配置项。值统一为字符串，业务层负责类型转换。

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID PK | 主键 |
| `key` | VARCHAR(200) UNIQUE | 配置键名（格式：`模块.配置项`） |
| `value` | TEXT | 配置值 |
| `remark` | VARCHAR(500) | 说明 |

**预置配置项分组**：

| 前缀 | 配置项 | 默认值 |
|------|--------|-------|
| `system.` | `name`、`version`、`default_language`、`max_upload_size_mb` | OA智审、1.0.0、zh-CN、50 |
| `system.` | `enable_audit_trail`、`enable_data_encryption` | true、false |
| `system.` | `backup_enabled`、`backup_cron`、`backup_retention_days` | false、0 2 * * *、30 |
| `system.` | `notification_email`、`smtp_host`、`smtp_port`、`smtp_username`、`smtp_ssl` | (空)、(空)、465、(空)、true |
| `auth.` | `login_fail_lock_threshold`、`account_lock_minutes` | 5、15 |
| `auth.` | `access_token_ttl_hours`、`refresh_token_ttl_days` | 2、7 |
| `tenant.` | `default_token_quota`、`default_max_concurrency` | 10000、10 |
| `tenant.` | `default_log_retention_days`、`default_data_retention_days` | 365、1095 |

---

## 五、选项表与集成配置（Migration 005）

### 5.1 oa_type_options — OA 系统类型选项表

| 字段 | 说明 | 预置数据 |
|------|------|---------|
| `code` | 选项编码 | weaver_e9, weaver_ebridge, zhiyuan_a8, landray_ekp, custom |
| `label` | 显示名称 | 泛微 Ecology E9, 泛微 E-Bridge, 致远 A8+, 蓝凌 EKP, 自定义 OA |
| `enabled` | 是否启用 | 全部 true |

### 5.2 db_driver_options — 数据库驱动类型选项表

| `code` | `label` | `default_port` |
|--------|---------|----------------|
| mysql | MySQL | 3306 |
| oracle | Oracle | 1521 |
| postgresql | PostgreSQL | 5432 |
| sqlserver | SQL Server | 1433 |
| dm | 达梦 DM | 5236 |

### 5.3 ai_deploy_type_options — AI 部署类型选项表

| `code` | `label` |
|--------|---------|
| local | 本地部署 |
| cloud | 云端API |

### 5.4 ai_provider_options — AI 服务商选项表

| `code` | `label` | `deploy_type` |
|--------|---------|--------------|
| xinference | Xinference | local |
| ollama | Ollama | local |
| vllm | vLLM | local |
| aliyun_bailian | 阿里云百炼 | cloud |
| deepseek | DeepSeek | cloud |
| zhipu | 智谱 AI | cloud |
| openai | OpenAI | cloud |
| azure_openai | Azure OpenAI | cloud |

### 5.5 oa_database_connections — OA 数据库连接表

详见 [OA 适配功能说明](../features/oa-integration.md#四oa-数据库连接管理)。

### 5.6 ai_model_configs — AI 模型配置表

详见 [AI 智能审核功能说明](../features/ai-audit.md#七ai-模型配置管理)。

---

## 六、审核配置与规则（Migration 007）

### 6.1 process_audit_configs — 流程审核配置表

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID PK | 主键 |
| `tenant_id` | UUID FK | 所属租户 |
| `process_type` | VARCHAR(200) | 流程类型标识 |
| `process_type_label` | VARCHAR(200) | 流程类型显示名称 |
| `main_table_name` | VARCHAR(200) | OA 主表名 |
| `main_fields` | JSONB | 主表字段配置 |
| `detail_tables` | JSONB | 明细子表配置 |
| `field_mode` | VARCHAR(20) | 字段模式：`all`/`selected` |
| `kb_mode` | VARCHAR(20) | 知识库模式：`rules_only`/`hybrid` |
| `ai_config` | JSONB | AI 审核配置（尺度/提示词/模型覆盖） |
| `user_permissions` | JSONB | 用户权限开关 |
| `status` | VARCHAR(20) | 状态：`active`/`inactive` |

**唯一约束**：`(tenant_id, process_type)`

### 6.2 audit_rules — 审核规则表

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID PK | 主键 |
| `tenant_id` | UUID FK | 所属租户 |
| `config_id` | UUID FK → process_audit_configs | 关联配置（NULL=通用规则） |
| `process_type` | VARCHAR(200) | 适用流程 |
| `rule_content` | TEXT | 规则内容（自然语言） |
| `rule_scope` | VARCHAR(20) | 作用域：`mandatory`/`default_on`/`default_off` |
| `enabled` | BOOLEAN | 是否启用 |
| `source` | VARCHAR(20) | 来源：`manual`/`file_import` |
| `related_flow` | BOOLEAN | 是否关联审批流 |

### 6.3 system_prompt_templates — 系统提示词模板表

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID PK | 主键 |
| `prompt_key` | VARCHAR(100) UNIQUE | 提示词唯一键 |
| `prompt_type` | VARCHAR(20) | 类型：`system`/`user` |
| `phase` | VARCHAR(20) | 阶段：`reasoning`/`extraction` |
| `strictness` | VARCHAR(20) | 尺度：`strict`/`standard`/`loose` |
| `content` | TEXT | 提示词内容 |
| `description` | VARCHAR(500) | 说明 |

**预置数据**：24 条（审核 12 + 归档 12）

### 6.4 process_archive_configs — 归档复盘配置表

结构与 `process_audit_configs` 基本一致，额外增加：

| 字段 | 类型 | 说明 |
|------|------|------|
| `access_control` | JSONB | 访问控制：`{allowed_roles, allowed_members, allowed_departments}` |

### 6.5 archive_rules — 归档规则表

结构与 `audit_rules` 完全一致，独立存储归档复盘的规则。

---

## 七、定时任务（Migration 008）

### 7.1 cron_tasks — 定时任务实例表

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID PK | 主键 |
| `tenant_id` | UUID FK | 所属租户 |
| `task_type` | VARCHAR(50) | 任务类型编码 |
| `task_label` | VARCHAR(200) | 任务显示名称 |
| `cron_expression` | VARCHAR(100) | Cron 表达式 |
| `is_active` | BOOLEAN | 是否启用 |
| `is_builtin` | BOOLEAN | 是否系统内置 |
| `push_email` | VARCHAR(255) | 推送邮箱 |
| `last_run_at` / `next_run_at` | TIMESTAMPTZ | 上次/下次执行时间 |
| `success_count` / `fail_count` | INT | 成功/失败次数 |

### 7.2 cron_task_type_presets — 任务类型系统预设表

> 全局预设（不绑定租户），共 6 条。

| `task_type` | `module` | 说明 |
|------------|---------|------|
| `audit_batch` | audit | 审核-批量处理 |
| `audit_daily` | audit | 审核-日报推送 |
| `audit_weekly` | audit | 审核-周报推送 |
| `archive_batch` | archive | 归档-批量处理 |
| `archive_daily` | archive | 归档-日报推送 |
| `archive_weekly` | archive | 归档-周报推送 |

每条预设包含：中英文名称/描述、默认 Cron 表达式、推送格式、内容模板（JSONB）。

### 7.3 cron_task_type_configs — 任务类型租户配置表

> 租户启用某任务类型后的自定义覆盖配置。

| 字段 | 类型 | 说明 |
|------|------|------|
| `tenant_id` | UUID FK | 所属租户 |
| `task_type` | VARCHAR(50) FK → presets | 任务类型 |
| `batch_limit` | INT | 单次批处理上限 |
| `push_format` | VARCHAR(20) | 推送格式 |
| `content_template` | JSONB | 自定义模板 |

**唯一约束**：`(tenant_id, task_type)`

---

## 八、日志表（Migration 009）

### 8.1 audit_logs — AI 审核执行日志

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID PK | 主键 |
| `tenant_id` | UUID FK | 所属租户 |
| `user_id` | UUID FK | 发起用户 |
| `process_id` | VARCHAR(100) | OA 流程单号 |
| `title` | VARCHAR(500) | 流程标题 |
| `process_type` | VARCHAR(200) | 流程类型 |
| `recommendation` | VARCHAR(20) | 审核建议：approve/return/review |
| `score` | INT | 综合评分 |
| `audit_result` | JSONB | 完整审核结果 JSON |
| `duration_ms` | INT | 审核耗时 |

### 8.2 cron_logs — 定时任务执行日志

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID PK | 主键 |
| `tenant_id` | UUID FK | 所属租户 |
| `task_id` | UUID FK → cron_tasks | 关联任务 |
| `task_type` | VARCHAR(50) | 任务类型 |
| `status` | VARCHAR(20) | 状态：running/success/failed |
| `message` | TEXT | 结果消息 |
| `started_at` / `finished_at` | TIMESTAMPTZ | 执行时间 |

### 8.3 archive_logs — 归档复盘日志

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID PK | 主键 |
| `tenant_id` | UUID FK | 所属租户 |
| `user_id` | UUID FK | 发起用户 |
| `process_id` | VARCHAR(100) | OA 流程单号 |
| `title` | VARCHAR(500) | 流程标题 |
| `process_type` | VARCHAR(200) | 流程类型 |
| `compliance` | VARCHAR(30) | 合规结论：compliant/non_compliant/partial |
| `compliance_score` | INT | 合规评分 |
| `archive_result` | JSONB | 归档复盘结果 JSON |

---

## 九、用户个性化配置（Migration 010）

### 9.1 user_personal_configs — 用户个人配置表

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID PK | 主键 |
| `tenant_id` | UUID FK | 所属租户 |
| `user_id` | UUID FK | 用户ID |
| `audit_details` | JSONB | 审核工作台各流程的个人偏好 |
| `cron_details` | JSONB | 定时任务偏好 |
| `archive_details` | JSONB | 归档复盘个人偏好 |

**唯一约束**：`(tenant_id, user_id)`

**JSONB 结构详见**：[个人配置脏数据清理](../todo/personal-config-dirty-data-cleanup.md)

### 9.2 user_dashboard_prefs — 用户仪表板偏好表

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID PK | 主键 |
| `tenant_id` | UUID FK | 所属租户 |
| `user_id` | UUID FK | 用户ID |
| `enabled_widgets` | JSONB | 已启用的仪表板组件ID列表 |
| `widget_sizes` | JSONB | 各组件尺寸配置 |

---

## 十、Token 追踪（Migration 011）

### 10.1 tenant_llm_message_logs — 租户大模型调用记录表

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | UUID PK | 主键 |
| `tenant_id` | UUID FK | 所属租户 |
| `user_id` | UUID FK | 发起用户（NULL=系统触发） |
| `model_config_id` | UUID FK → ai_model_configs | 使用的模型 |
| `request_type` | VARCHAR(50) | 请求类型：audit/archive/other |
| `input_tokens` | INT | 输入 Token |
| `output_tokens` | INT | 输出 Token |
| `total_tokens` | INT | 总 Token |
| `duration_ms` | INT | 调用耗时 |
| `call_type` | VARCHAR(20) | LLM 调用类型：`reasoning`（推理调用）/ `structured`（结构化调用），默认 `reasoning`（Migration 029 新增） |

**索引**：`idx_tllm_call_type (tenant_id, call_type)`（Migration 029 新增）

---

## 十一、表关系图

```
users ──┬── user_role_assignments ──── tenants
        │                                │
        ├── login_history                 ├── departments ─── org_members ─── org_member_roles ─── org_roles
        │                                │
        ├── user_personal_configs ────────┤
        │                                │
        ├── user_dashboard_prefs ─────────┤
        │                                │
        ├── audit_logs ───────────────────┤
        │                                │
        ├── archive_logs ─────────────────┤
        │                                │
        └── tenant_llm_message_logs ──────┤
                                          │
                                          ├── oa_database_connections
                                          │
                                          ├── ai_model_configs
                                          │
                                          ├── process_audit_configs ─── audit_rules
                                          │
                                          ├── process_archive_configs ── archive_rules
                                          │
                                          ├── cron_tasks ─── cron_logs
                                          │
                                          └── cron_task_type_configs ─── cron_task_type_presets (全局)

独立表: system_configs, system_prompt_templates
选项表: oa_type_options, db_driver_options, ai_deploy_type_options, ai_provider_options
```

---

## 十二、Migration 脚本列表

| 编号 | 文件名 | 说明 |
|------|--------|------|
| 000001 | `init_extensions` | 初始化 PostgreSQL 扩展（pgvector 等） |
| 000002 | `tenants_users` | 创建租户、用户、角色分配、登录历史表 |
| 000003 | `org_structure` | 创建部门、组织角色、组织成员、成员角色表 |
| 000004 | `system_configs` | 创建系统配置 KV 表并初始化默认值 |
| 000005 | `system_options_oa_ai` | 创建选项表、OA 连接表、AI 模型表，扩展租户表 |
| 000006 | `tenant_admin_user` | 租户表新增 admin_user_id 字段 |
| 000007 | `audit_configs_rules_presets` | 创建审核/归档配置、规则、提示词模板表 |
| 000008 | `cron_tasks` | 创建定时任务实例、预设、租户配置表 |
| 000009 | `audit_cron_archive_logs` | 创建审核/定时/归档日志表 |
| 000010 | `user_personal_configs` | 创建用户个人配置、仪表板偏好表 |
| 000011 | `tenant_llm_message_logs` | 创建 LLM 调用记录表 |
| 000029 | `llm_call_type` | tenant_llm_message_logs 新增 call_type 列及复合索引 |

每个迁移脚本均包含对应的 `.down.sql` 回滚脚本。
