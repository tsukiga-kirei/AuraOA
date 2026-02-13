# 前端接口文档

> 本文档梳理前端所有需要对接的后端 API 接口，当前前端使用模拟数据（Mock），后续对接时按此文档实现。

## 1. 认证模块

### 1.1 用户登录
- **POST** `/api/auth/login`
- **请求体**:
  ```json
  {
    "username": "string",
    "password": "string",
    "tenant_id": "string"
  }
  ```
- **响应**:
  ```json
  {
    "access_token": "string",
    "refresh_token": "string",
    "expires_in": 3600
  }
  ```

### 1.2 获取用户菜单（RBAC）
- **GET** `/api/auth/menu`
- **Headers**: `Authorization: Bearer {token}`
- **响应**:
  ```json
  {
    "menus": [
      {
        "key": "dashboard",
        "label": "审核工作台",
        "path": "/dashboard",
        "icon": "DashboardOutlined",
        "children": []
      }
    ]
  }
  ```

### 1.3 获取当前用户信息
- **GET** `/api/auth/profile`
- **Headers**: `Authorization: Bearer {token}`
- **响应**:
  ```json
  {
    "username": "string",
    "display_name": "string",
    "tenant_id": "string",
    "role": "business | tenant_admin | system_admin",
    "role_label": "string（角色显示名称，如'普通用户'、'租户管理员'、'系统管理员'）",
    "department": "string",
    "position": "string",
    "email": "string",
    "phone": "string",
    "permissions": ["business", "tenant_admin", "system_admin"]
  }
  ```
- **字段说明**:
  - `role_label`: 角色的中文显示名称，用于前端界面展示
  - `permissions`: 权限组数组，取值为 `business`（前台工作台）、`tenant_admin`（租户管理）、`system_admin`（系统管理）。前端据此控制侧边栏菜单分区可见性及页面访问权限。不同角色可拥有任意权限组合（如租户管理员可仅有 `tenant_admin` 而无 `business`）

---

## 2. 审核工作台模块

### 2.1 获取待办流程列表
- **GET** `/api/audit/todo`
- **Headers**: `Authorization: Bearer {token}`
- **Query**: `?status=pending&page=1&size=20&search=keyword`
- **响应**:
  ```json
  {
    "processes": [
      {
        "process_id": "WF-2025-001",
        "title": "办公设备采购申请",
        "applicant": "张明",
        "department": "研发部",
        "submit_time": "2025-06-10 09:30",
        "process_type": "采购审批",
        "status": "pending",
        "amount": 156000,
        "urgency": "high | medium | low",
        "oa_url": "https://oa.example.com/workflow/view/WF-2025-001"
      }
    ],
    "total": 6,
    "page": 1,
    "size": 20
  }
  ```

### 2.2 获取已通过/已驳回/已归档流程列表
- **GET** `/api/audit/history`
- **Headers**: `Authorization: Bearer {token}`
- **Query**: `?status=approved|rejected|archived&page=1&size=20`
- **响应**: 同 2.1 结构，status 为 `approved`、`rejected` 或 `archived`
- **说明**: 已归档流程同时可通过 4.2 归档接口获取完整审批链与字段详情

### 2.3 执行 AI 审核
- **POST** `/api/audit/execute`
- **Headers**: `Authorization: Bearer {token}`
- **请求体**:
  ```json
  {
    "process_id": "WF-2025-001"
  }
  ```
- **响应**:
  ```json
  {
    "trace_id": "TR-20250610-A3F8",
    "process_id": "WF-2025-001",
    "recommendation": "approve | reject | revise",
    "score": 72,
    "duration_ms": 1850,
    "details": [
      {
        "rule_id": "R001",
        "rule_name": "预算额度校验",
        "passed": true,
        "reasoning": "采购金额未超过预算上限",
        "is_locked": true
      }
    ],
    "ai_reasoning": "该采购申请整体合规性尚可..."
  }
  ```

### 2.4 提交审核反馈
- **POST** `/api/audit/feedback`
- **Headers**: `Authorization: Bearer {token}`
- **请求体**:
  ```json
  {
    "process_id": "WF-2025-001",
    "adopted": true,
    "action_taken": "approve | reject | revise"
  }
  ```

### 2.5 获取 OA 跳转链接
- **GET** `/api/audit/oa-link/{process_id}`
- **Headers**: `Authorization: Bearer {token}`
- **响应**:
  ```json
  {
    "url": "https://oa.example.com/workflow/view/WF-2025-001"
  }
  ```

---

## 3. 定时任务模块

### 3.1 获取任务列表
- **GET** `/api/cron/tasks`
- **Headers**: `Authorization: Bearer {token}`
- **响应**:
  ```json
  {
    "tasks": [
      {
        "id": "CT-001",
        "cron_expression": "0 9 * * 1-5",
        "task_type": "batch_audit | daily_report | weekly_report",
        "is_active": true,
        "last_run_at": "2025-06-10 09:00",
        "next_run_at": "2025-06-11 09:00",
        "created_at": "2025-05-01",
        "success_count": 28,
        "fail_count": 1,
        "is_builtin": false,
        "push_email": "user@example.com"
      }
    ]
  }
  ```
- **字段说明**:
  - `is_builtin`（可选）: 是否为系统内置任务，内置任务不可删除
  - `push_email`（可选）: 任务推送邮箱地址，为空时使用用户默认邮箱

### 3.2 创建任务
- **POST** `/api/cron/tasks`
- **请求体**:
  ```json
  {
    "cron_expression": "0 9 * * 1-5",
    "task_type": "batch_audit",
    "push_email": "user@example.com"
  }
  ```
- **字段说明**: `push_email` 为可选字段

### 3.3 更新任务
- **PUT** `/api/cron/tasks/{task_id}`
- **Headers**: `Authorization: Bearer {token}`
- **请求体**:
  ```json
  {
    "cron_expression": "0 9 * * 1-5",
    "task_type": "batch_audit",
    "push_email": "user@example.com"
  }
  ```
- **说明**: 所有字段均为可选，仅提交需要修改的字段。内置任务（`is_builtin: true`）的 `task_type` 不可修改。

### 3.4 删除任务
- **DELETE** `/api/cron/tasks/{task_id}`
- **说明**: 内置任务（`is_builtin: true`）不可删除，返回 403。

### 3.5 切换任务状态
- **PATCH** `/api/cron/tasks/{task_id}/toggle`

### 3.6 立即执行任务
- **POST** `/api/cron/tasks/{task_id}/execute`

### 3.7 复制任务
- **POST** `/api/cron/tasks/{task_id}/copy`
- **Headers**: `Authorization: Bearer {token}`
- **说明**: 基于已有任务创建副本，副本默认为暂停状态（`is_active: false`），`is_builtin` 强制为 `false`，统计计数归零。
- **响应**: 同 3.1 中单个任务结构

### 3.8 获取定时任务类型配置列表（租户管理）
- **GET** `/api/tenant/cron-task-configs`
- **Headers**: `Authorization: Bearer {token}`
- **说明**: 返回当前租户下各定时任务类型的配置，包括推送格式、内容模板、AI 配置和用户权限控制。仅租户管理员可访问。
- **响应**:
  ```json
  {
    "configs": [
      {
        "task_type": "batch_audit | daily_report | weekly_report",
        "label": "批量审核",
        "enabled": true,
        "push_format": "html | markdown | plain",
        "content_template": {
          "subject": "【OA智审】批量审核结果通知 - {{date}}",
          "header": "以下是今日批量审核的结果汇总：",
          "body_template": "共审核 {{total}} 条流程...",
          "footer": "如有疑问请联系管理员。",
          "include_ai_summary": true,
          "include_statistics": true,
          "include_detail_list": true
        },
        "ai_config": {
          "model_name": "Qwen2.5-72B",
          "ai_provider": "本地部署",
          "system_prompt": "string"
        },
        "user_permissions": {
          "allow_modify_email": true,
          "allow_modify_schedule": true,
          "allow_modify_prompt": false,
          "allow_modify_template": false
        }
      }
    ]
  }
  ```
- **字段说明**:
  - `content_template`: 推送内容模板配置，包含邮件主题、头部、正文模板、底部及内容模块开关
  - `allow_modify_template`: 是否允许用户自定义推送内容模板

### 3.9 更新定时任务类型配置（租户管理）
- **PUT** `/api/tenant/cron-task-configs/{task_type}`
- **Headers**: `Authorization: Bearer {token}`
- **请求体**: 同 3.8 响应中单个配置对象结构

---

## 4. 归档复盘模块

### 4.1 检索审核快照
- **GET** `/api/archive/snapshots`
- **Query**: `?department=IT部&recommendation=approve&date_from=2025-06-01&date_to=2025-06-10&page=1&size=20`
- **响应**:
  ```json
  {
    "snapshots": [
      {
        "snapshot_id": "SN-001",
        "process_id": "WF-2025-098",
        "title": "年度IT设备采购",
        "applicant": "王强",
        "department": "IT部",
        "recommendation": "approve",
        "score": 95,
        "created_at": "2025-06-09 16:30",
        "adopted": true,
        "oa_url": "https://oa.example.com/workflow/view/WF-2025-098"
      }
    ],
    "total": 100,
    "page": 1,
    "size": 20
  }
  ```

### 4.2 获取归档流程列表
- **GET** `/api/archive/processes`
- **Headers**: `Authorization: Bearer {token}`
- **Query**: `?process_type=采购审批&department=IT部&date_from=2025-01-01&date_to=2025-06-30&page=1&size=20`
- **响应**:
  ```json
  {
    "processes": [
      {
        "process_id": "WF-2025-050",
        "title": "2025年度服务器集群采购",
        "applicant": "王强",
        "department": "IT部",
        "process_type": "采购审批",
        "amount": 1200000,
        "submit_time": "2025-04-15 09:00",
        "archive_time": "2025-05-20 17:30",
        "status": "archived",
        "flow_nodes": [
          {
            "node_id": "N1",
            "node_name": "部门经理审批",
            "approver": "李明",
            "action": "approve | reject | revise",
            "action_time": "2025-04-16 10:00",
            "opinion": "同意，符合年度IT规划"
          }
        ],
        "fields": {
          "supplier": "XX云计算有限公司",
          "contract_no": "HT-2025-0088"
        }
      }
    ],
    "total": 50,
    "page": 1,
    "size": 20
  }
  ```

### 4.3 执行归档合规复审
- **POST** `/api/archive/re-audit`
- **Headers**: `Authorization: Bearer {token}`
- **请求体**:
  ```json
  {
    "process_id": "WF-2025-050"
  }
  ```
- **响应**:
  ```json
  {
    "trace_id": "ATR-20250610-X1Y2",
    "process_id": "WF-2025-050",
    "overall_compliance": "compliant | non_compliant | partially_compliant",
    "overall_score": 92,
    "duration_ms": 3200,
    "flow_audit": {
      "is_complete": true,
      "missing_nodes": [],
      "node_results": [
        {
          "node_id": "N1",
          "node_name": "部门经理审批",
          "compliant": true,
          "reasoning": "审批权限匹配，审批时效正常"
        }
      ]
    },
    "field_audit": [
      {
        "field_name": "供应商资质",
        "passed": true,
        "reasoning": "供应商在合格名录中"
      }
    ],
    "rule_audit": [
      {
        "rule_id": "R001",
        "rule_name": "预算额度校验",
        "passed": true,
        "reasoning": "采购金额在年度IT预算范围内"
      }
    ],
    "ai_summary": "该采购流程整体合规，审批链完整..."
  }
  ```

### 4.4 导出审核记录
- **GET** `/api/archive/export`
- **Query**: `?format=json|csv|excel&department=IT部`
- **响应**: 文件下载流

---

## 5. 个人设置模块

### 5.1 获取个人信息
- **GET** `/api/user/profile`
- **响应**: 同 1.3

### 5.2 更新个人信息
- **PUT** `/api/user/profile`
- **请求体**:
  ```json
  {
    "nickname": "string",
    "email": "string",
    "phone": "string"
  }
  ```

### 5.3 获取用户审核工作台配置
- **GET** `/api/user/audit-config`
- **说明**: 返回用户可见的流程审核配置列表（复用租户 `ProcessAuditConfig` 结构），以及用户的个性化覆盖（自定义规则、字段覆盖）。用户可操作范围受各流程 `user_permissions` 控制。
- **响应**:
  ```json
  {
    "process_configs": [
      {
        "id": "PAC-001",
        "process_type": "采购审批",
        "flow_path": "部门经理 → 财务总监 → 总经理",
        "fields": [
          {
            "field_key": "amount",
            "field_name": "采购金额",
            "field_type": "number",
            "selected": true
          }
        ],
        "field_mode": "all | selected",
        "rules": [
          {
            "id": "R001",
            "process_type": "采购审批",
            "rule_content": "单笔采购金额不得超过部门季度预算上限",
            "rule_scope": "mandatory | default_on | default_off",
            "priority": 100,
            "enabled": true,
            "source": "manual | file_import"
          }
        ],
        "kb_mode": "rules_only | rag_only | hybrid",
        "ai_config": {
          "ai_provider": "string",
          "model_name": "string",
          "audit_strictness": "strict | standard | loose",
          "system_prompt": "string",
          "context_window": 8192,
          "temperature": 0.3
        },
        "user_permissions": {
          "allow_custom_fields": false,
          "allow_custom_rules": true,
          "allow_modify_strictness": true
        }
      }
    ],
    "user_custom_rules": {
      "PAC-001": [
        { "id": "UCR-001", "content": "供应商必须在合格名录中", "enabled": true }
      ]
    },
    "user_field_overrides": {
      "PAC-004": ["salary_range"]
    }
  }
  ```

### 5.4 更新用户审核工作台配置
- **PUT** `/api/user/audit-config/{process_config_id}`
- **说明**: 保存用户对某个流程的个性化配置。可提交的字段受该流程 `user_permissions` 控制：仅当对应权限开启时，相关字段才会被后端接受。
- **请求体**:
  ```json
  {
    "audit_strictness": "strict | standard | loose",
    "custom_rules": [
      { "id": "UCR-001", "content": "string", "enabled": true }
    ],
    "field_overrides": ["salary_range"],
    "rule_toggle_overrides": [
      { "rule_id": "R003", "enabled": false }
    ]
  }
  ```

### 5.5 获取用户定时任务配置
- **GET** `/api/user/cron-config`
- **Headers**: `Authorization: Bearer {token}`
- **说明**: 返回用户可见的定时任务类型配置列表（复用租户 `CronTaskTypeConfig` 结构），以及用户的个性化设置（默认推送邮箱等）。用户可操作范围受各任务类型 `user_permissions` 控制。提示词内容仅在 `allow_modify_prompt` 为 `true` 时返回。内容模板仅在 `allow_modify_template` 为 `true` 时可编辑。
- **响应**:
  ```json
  {
    "default_push_email": "user@example.com",
    "task_type_configs": [
      {
        "task_type": "batch_audit",
        "label": "批量审核",
        "enabled": true,
        "push_format": "html",
        "content_template": {
          "subject": "【OA智审】批量审核结果通知 - {{date}}",
          "header": "以下是今日批量审核的结果汇总：",
          "body_template": "共审核 {{total}} 条流程...",
          "footer": "如有疑问请联系管理员。",
          "include_ai_summary": true,
          "include_statistics": true,
          "include_detail_list": true
        },
        "ai_config": {
          "model_name": "Qwen2.5-72B",
          "ai_provider": "本地部署",
          "system_prompt": "string（仅 allow_modify_prompt 为 true 时返回）"
        },
        "user_permissions": {
          "allow_modify_email": true,
          "allow_modify_schedule": true,
          "allow_modify_prompt": false,
          "allow_modify_template": false
        }
      }
    ],
    "user_email_overrides": {
      "daily_report": "custom-email@example.com"
    },
    "user_template_overrides": {
      "weekly_report": {
        "subject": "自定义主题",
        "header": "自定义头部",
        "body_template": "自定义正文",
        "footer": "自定义底部",
        "include_ai_summary": true,
        "include_statistics": true,
        "include_detail_list": false
      }
    }
  }
  ```
- **字段说明**:
  - `content_template`: 推送内容模板（始终返回，用于展示当前配置）
  - `user_template_overrides`: 用户自定义的模板覆盖（仅 `allow_modify_template` 为 `true` 的任务类型可提交）

### 5.6 更新用户定时任务配置
- **PUT** `/api/user/cron-config`
- **Headers**: `Authorization: Bearer {token}`
- **说明**: 保存用户的定时任务个性化配置。可提交的字段受各任务类型 `user_permissions` 控制。`template_overrides` 仅在 `allow_modify_template` 为 `true` 时被后端接受。
- **请求体**:
  ```json
  {
    "default_push_email": "user@example.com",
    "email_overrides": {
      "daily_report": "custom-email@example.com"
    },
    "template_overrides": {
      "weekly_report": {
        "subject": "自定义主题",
        "header": "自定义头部",
        "body_template": "自定义正文",
        "footer": "自定义底部",
        "include_ai_summary": true,
        "include_statistics": true,
        "include_detail_list": false
      }
    }
  }
  ```

### 5.7 获取用户归档复盘配置
- **GET** `/api/user/archive-review-config`
- **Headers**: `Authorization: Bearer {token}`
- **说明**: 返回用户可见的归档复盘配置列表（复用租户 `ArchiveReviewConfig` 结构），以及用户的个性化覆盖（自定义规则、自定义审批流规则、字段覆盖、复核尺度）。用户可操作范围受各流程 `user_permissions` 控制。
- **响应**:
  ```json
  {
    "archive_configs": [
      {
        "id": "ARC-001",
        "process_type": "采购审批",
        "flow_path": "部门经理 → 财务总监 → 总经理",
        "fields": [
          {
            "field_key": "amount",
            "field_name": "采购金额",
            "field_type": "number",
            "selected": true
          }
        ],
        "field_mode": "all | selected",
        "rules": [
          {
            "id": "AR001",
            "rule_content": "采购金额超过50万需总经理审批记录",
            "rule_scope": "mandatory | default_on | default_off",
            "priority": 100,
            "enabled": true
          }
        ],
        "flow_rules": [
          {
            "id": "FR001",
            "rule_name": "审批链完整性",
            "rule_content": "审批流程必须经过所有必要节点，不得跳过",
            "rule_scope": "mandatory | default_on | default_off",
            "priority": 100,
            "enabled": true
          }
        ],
        "kb_mode": "rules_only | rag_only | hybrid",
        "ai_config": {
          "ai_provider": "string",
          "model_name": "string",
          "audit_strictness": "strict | standard | loose",
          "system_prompt": "string",
          "context_window": 8192,
          "temperature": 0.3
        },
        "user_permissions": {
          "allow_custom_fields": true,
          "allow_custom_rules": true,
          "allow_modify_strictness": true,
          "allow_custom_flow_rules": false
        }
      }
    ],
    "user_custom_rules": {
      "ARC-001": [
        { "id": "UCAR-001", "content": "归档前需确认所有附件完整", "enabled": true }
      ]
    },
    "user_custom_flow_rules": {
      "ARC-001": [
        { "id": "UCFR-001", "content": "审批流程中需包含法务审核节点", "enabled": true }
      ]
    },
    "user_field_overrides": {
      "ARC-002": ["contract_no"]
    },
    "user_strictness_overrides": {
      "ARC-001": "strict"
    }
  }
  ```
- **字段说明**:
  - `user_custom_rules`: 用户自定义的归档审核规则（仅 `allow_custom_rules` 为 `true` 时可提交）
  - `user_custom_flow_rules`: 用户自定义的审批流规则（仅 `allow_custom_flow_rules` 为 `true` 时可提交）
  - `user_field_overrides`: 用户自定义的字段覆盖（仅 `allow_custom_fields` 为 `true` 时可提交）
  - `user_strictness_overrides`: 用户自定义的复核尺度覆盖（仅 `allow_modify_strictness` 为 `true` 时可提交）

### 5.8 更新用户归档复盘配置
- **PUT** `/api/user/archive-review-config/{config_id}`
- **Headers**: `Authorization: Bearer {token}`
- **说明**: 保存用户对某个流程的归档复盘个性化配置。可提交的字段受该流程 `user_permissions` 控制。
- **请求体**:
  ```json
  {
    "audit_strictness": "strict | standard | loose",
    "custom_rules": [
      { "id": "UCAR-001", "content": "string", "enabled": true }
    ],
    "custom_flow_rules": [
      { "id": "UCFR-001", "content": "string", "enabled": true }
    ],
    "field_overrides": ["contract_no"],
    "rule_toggle_overrides": [
      { "rule_id": "AR003", "enabled": false }
    ]
  }
  ```

---

## 6. 租户管理模块（租户管理员）

> 租户管理页面（`/admin/tenant`）已重构为**以流程为维度**的配置模式。页面左侧为流程列表导航，右侧按流程展示四个配置 Tab：字段配置、审核规则、AI 配置、用户权限。主要使用 6.5/6.6 流程审核配置接口，旧的扁平规则接口（6.1-6.4）仅作为兼容保留。

### 6.1 获取审核规则列表（兼容接口）
- **GET** `/api/tenant/rules`
- **说明**: 返回所有流程的规则扁平列表。前端已改为按流程维度管理规则（见 6.5），此接口仅作兼容保留。
- **响应**:
  ```json
  {
    "rules": [
      {
        "id": "R001",
        "process_type": "采购审批",
        "rule_content": "单笔采购金额不得超过部门季度预算上限",
        "rule_scope": "mandatory | default_on | default_off",
        "priority": 100,
        "enabled": true,
        "source": "manual | file_import"
      }
    ]
  }
  ```

### 6.2 创建/更新规则（兼容接口）
- **POST** `/api/tenant/rules`
- **PUT** `/api/tenant/rules/{rule_id}`
- **说明**: 前端已改为通过 6.6 更新流程级配置来管理规则，此接口仅作兼容保留。

### 6.3 删除规则（兼容接口）
- **DELETE** `/api/tenant/rules/{rule_id}`
- **说明**: 同上，前端已改为流程级操作。

### 6.4 获取/设置知识库模式（兼容接口）
- **GET** `/api/tenant/kb-mode`
- **PUT** `/api/tenant/kb-mode`
- **请求体**: `{ "mode": "rules_only | rag_only | hybrid" }`
- **说明**: 知识库模式已纳入流程审核配置（6.5 中的 `kb_mode` 字段），此接口仅作兼容保留。

### 6.5 获取流程审核配置列表
- **GET** `/api/tenant/process-configs`
- **Headers**: `Authorization: Bearer {token}`
- **说明**: 返回当前租户下所有流程的审核配置，用于页面左侧流程导航列表。
- **响应**:
  ```json
  {
    "configs": [
      {
        "id": "PAC-001",
        "process_type": "采购审批",
        "flow_path": "部门经理 → 财务总监 → 总经理"
      }
    ]
  }
  ```

### 6.6 获取单个流程审核配置
- **GET** `/api/tenant/process-config/{process_type}`
- **Headers**: `Authorization: Bearer {token}`
- **响应**:
  ```json
  {
    "id": "PC-001",
    "process_type": "采购审批",
    "flow_path": "部门经理 → 财务总监 → 总经理",
    "fields": [
      {
        "field_key": "amount",
        "field_name": "采购金额",
        "field_type": "number",
        "selected": true
      }
    ],
    "field_mode": "all | selected",
    "rules": [
      {
        "id": "R001",
        "process_type": "采购审批",
        "rule_content": "单笔采购金额不得超过部门季度预算上限",
        "rule_scope": "mandatory | default_on | default_off",
        "priority": 100,
        "enabled": true,
        "source": "manual | file_import"
      }
    ],
    "kb_mode": "rules_only | rag_only | hybrid",
    "ai_config": {
      "ai_provider": "openai",
      "model_name": "gpt-4",
      "audit_strictness": "strict | standard | loose",
      "system_prompt": "string",
      "context_window": 8192,
      "temperature": 0.3
    },
    "user_permissions": {
      "allow_custom_fields": true,
      "allow_custom_rules": true,
      "allow_modify_strictness": false
    }
  }
  ```

### 6.7 更新流程审核配置
- **PUT** `/api/tenant/process-config/{process_type}`
- **Headers**: `Authorization: Bearer {token}`
- **请求体**: 同 6.6 响应结构

### 6.8 新增流程审核配置
- **POST** `/api/tenant/process-configs`
- **Headers**: `Authorization: Bearer {token}`
- **说明**: 创建一个新的流程审核配置。`process_type` 和 `flow_path` 为必填，其余字段使用默认值（field_mode: selected, kb_mode: rules_only, 默认 AI 配置, 用户权限全部关闭）。
- **请求体**:
  ```json
  {
    "process_type": "string (必填)",
    "flow_path": "string (选填，默认'待配置')"
  }
  ```
- **响应**:
  ```json
  {
    "id": "PAC-xxx",
    "process_type": "新流程名称",
    "flow_path": "待配置",
    "fields": [],
    "field_mode": "selected",
    "rules": [],
    "kb_mode": "rules_only",
    "ai_config": {
      "ai_provider": "本地部署",
      "model_name": "Qwen2.5-72B",
      "audit_strictness": "standard",
      "system_prompt": "",
      "context_window": 8192,
      "temperature": 0.3
    },
    "user_permissions": {
      "allow_custom_fields": false,
      "allow_custom_rules": false,
      "allow_modify_strictness": false
    }
  }
  ```

### 6.9 获取归档复盘配置列表
- **GET** `/api/tenant/archive-review-configs`
- **Headers**: `Authorization: Bearer {token}`
- **说明**: 返回当前租户下所有流程的归档复盘配置，用于租户管理页面"归档复盘"页签。包含字段配置、审核规则、审批流规则、AI 配置和用户权限五个维度。
- **响应**:
  ```json
  {
    "configs": [
      {
        "id": "ARC-001",
        "process_type": "采购审批",
        "flow_path": "部门经理 → 财务总监 → 总经理",
        "fields": [
          {
            "field_key": "amount",
            "field_name": "采购金额",
            "field_type": "number",
            "selected": true
          }
        ],
        "field_mode": "all | selected",
        "rules": [
          {
            "id": "AR001",
            "process_type": "采购审批",
            "rule_content": "采购金额超过50万需总经理审批记录",
            "rule_scope": "mandatory | default_on | default_off",
            "priority": 100,
            "enabled": true,
            "source": "manual | file_import"
          }
        ],
        "flow_rules": [
          {
            "id": "FR001",
            "rule_name": "审批链完整性",
            "rule_content": "审批流程必须经过所有必要节点，不得跳过",
            "rule_scope": "mandatory | default_on | default_off",
            "priority": 100,
            "enabled": true
          }
        ],
        "kb_mode": "rules_only | rag_only | hybrid",
        "ai_config": {
          "ai_provider": "string",
          "model_name": "string",
          "audit_strictness": "strict | standard | loose",
          "system_prompt": "string",
          "context_window": 8192,
          "temperature": 0.3
        },
        "user_permissions": {
          "allow_custom_fields": true,
          "allow_custom_rules": true,
          "allow_modify_strictness": true,
          "allow_custom_flow_rules": false
        }
      }
    ]
  }
  ```
- **字段说明**:
  - `flow_rules`: 审批流规则配置，用于校验整个审批流程是否符合要求（如审批链完整性、节点顺序、审批时效等）
  - `allow_custom_flow_rules`: 是否允许用户自定义审批流规则

### 6.10 获取单个归档复盘配置
- **GET** `/api/tenant/archive-review-config/{config_id}`
- **Headers**: `Authorization: Bearer {token}`
- **响应**: 同 6.9 响应中单个配置对象结构

### 6.11 更新归档复盘配置
- **PUT** `/api/tenant/archive-review-config/{config_id}`
- **Headers**: `Authorization: Bearer {token}`
- **请求体**: 同 6.9 响应中单个配置对象结构
- **说明**: 更新指定流程的归档复盘配置，包括字段、审核规则、审批流规则、AI 配置和用户权限。

---

## 7. 组织人员模块（租户管理员）

### 7.1 获取部门列表
- **GET** `/api/tenant/departments`
- **Headers**: `Authorization: Bearer {token}`
- **响应**:
  ```json
  {
    "departments": [
      {
        "id": "D-001",
        "name": "研发部",
        "parent_id": null,
        "manager": "张明",
        "member_count": 12
      }
    ]
  }
  ```

### 7.2 创建/更新/删除部门
- **POST** `/api/tenant/departments`
- **PUT** `/api/tenant/departments/{dept_id}`
- **DELETE** `/api/tenant/departments/{dept_id}`
- **说明**: 删除部门前需确保部门下无成员

### 7.3 获取角色列表
- **GET** `/api/tenant/roles`
- **Headers**: `Authorization: Bearer {token}`
- **响应**:
  ```json
  {
    "roles": [
      {
        "id": "ROLE-001",
        "name": "业务用户",
        "description": "普通业务人员",
        "page_permissions": ["/dashboard", "/cron", "/settings"],
        "is_system": true
      }
    ]
  }
  ```

### 7.4 创建/更新/删除角色
- **POST** `/api/tenant/roles`
- **PUT** `/api/tenant/roles/{role_id}`
- **DELETE** `/api/tenant/roles/{role_id}`
- **说明**: 系统角色（`is_system: true`）不可删除；删除角色前需确保无成员使用该角色

### 7.5 获取组织成员列表
- **GET** `/api/tenant/members`
- **Headers**: `Authorization: Bearer {token}`
- **Query**: `?department_id=D-001&role_id=ROLE-001&search=keyword&page=1&size=20`
- **响应**:
  ```json
  {
    "members": [
      {
        "id": "M-001",
        "name": "张明",
        "username": "zhangming",
        "department_id": "D-001",
        "department_name": "研发部",
        "role_id": "ROLE-001",
        "role_name": "业务用户",
        "email": "zhangming@example.com",
        "phone": "138****8888",
        "position": "高级工程师",
        "status": "active | disabled",
        "created_at": "2024-03-15"
      }
    ],
    "total": 12,
    "page": 1,
    "size": 20
  }
  ```

### 7.6 创建/更新/删除/切换成员状态
- **POST** `/api/tenant/members`
- **PUT** `/api/tenant/members/{member_id}`
- **DELETE** `/api/tenant/members/{member_id}`
- **PATCH** `/api/tenant/members/{member_id}/toggle`

---

## 8. 数据信息模块（租户管理员）

### 8.1 获取审核操作日志
- **GET** `/api/tenant/data/audit-logs`
- **Headers**: `Authorization: Bearer {token}`
- **Query**: `?action=ai_audit&search=keyword&page=1&size=20`
- **响应**:
  ```json
  {
    "logs": [
      {
        "id": "AL-001",
        "process_id": "WF-2025-001",
        "title": "办公设备采购申请",
        "operator": "张明",
        "action": "ai_audit | manual_approve | manual_reject | feedback",
        "action_label": "AI 审核",
        "result": "建议修改（72分）",
        "created_at": "2025-06-10 09:35"
      }
    ],
    "total": 100,
    "page": 1,
    "size": 20
  }
  ```

### 8.2 获取定时任务执行日志
- **GET** `/api/tenant/data/cron-logs`
- **Headers**: `Authorization: Bearer {token}`
- **Query**: `?status=success&search=keyword&page=1&size=20`
- **响应**:
  ```json
  {
    "logs": [
      {
        "id": "CL-001",
        "task_id": "CT-BUILTIN-001",
        "task_type": "batch_audit",
        "task_label": "批量审核",
        "status": "success | failed | running",
        "recipients": "zhangming@example.com",
        "started_at": "2025-06-10 09:00",
        "finished_at": "2025-06-10 09:05",
        "message": "成功审核 12 条流程"
      }
    ],
    "total": 50,
    "page": 1,
    "size": 20
  }
  ```

### 8.3 获取归档操作日志
- **GET** `/api/tenant/data/archive-logs`
- **Headers**: `Authorization: Bearer {token}`
- **Query**: `?action=re_audit&search=keyword&page=1&size=20`
- **响应**:
  ```json
  {
    "logs": [
      {
        "id": "ARL-001",
        "process_id": "WF-2025-050",
        "title": "2025年度服务器集群采购",
        "operator": "张华",
        "action": "re_audit | export | view",
        "action_label": "合规复核",
        "compliance": "合规（92分）",
        "created_at": "2025-06-10 10:30"
      }
    ],
    "total": 30,
    "page": 1,
    "size": 20
  }
  ```

### 8.4 删除日志记录
- **DELETE** `/api/tenant/data/audit-logs/{log_id}`
- **DELETE** `/api/tenant/data/cron-logs/{log_id}`
- **DELETE** `/api/tenant/data/archive-logs/{log_id}`

### 8.5 导出日志
- **GET** `/api/tenant/data/export`
- **Query**: `?type=audit|cron|archive&format=json|csv|excel`
- **响应**: 文件下载流

---

## 9. 系统管理模块（系统管理员）

> 系统管理模块包含三大功能：**租户管理**（含 JDBC 连接、AI 模型配置、配额策略）、**全局监控**、**系统设置**（OA 系统管理、AI 模型管理、平台配置）。

### 9.1 获取租户列表
- **GET** `/api/system/tenants`
- **响应**:
  ```json
  {
    "tenants": [
      {
        "id": "T-001",
        "name": "示例集团总部",
        "code": "DEMO_HQ",
        "oa_type": "weaver_e9",
        "token_quota": 100000,
        "token_used": 42350,
        "max_concurrency": 20,
        "status": "active | inactive",
        "created_at": "2025-01-15",
        "contact_name": "张明",
        "contact_email": "zhangming@demo-group.com",
        "contact_phone": "138****8888",
        "description": "示例集团总部",
        "jdbc_config": {
          "driver": "mysql | postgresql | oracle | sqlserver",
          "host": "192.168.1.100",
          "port": 3306,
          "database": "ecology",
          "username": "oa_reader",
          "password": "********（前端展示脱敏）",
          "pool_size": 20,
          "connection_timeout": 30,
          "test_on_borrow": true
        },
        "ai_config": {
          "default_provider": "本地部署",
          "default_model": "Qwen2.5-72B",
          "fallback_provider": "云端API",
          "fallback_model": "GPT-4o",
          "max_tokens_per_request": 8192,
          "temperature": 0.3,
          "timeout_seconds": 60,
          "retry_count": 3
        },
        "log_retention_days": 365,
        "data_retention_days": 1095,
        "allow_custom_model": true,
        "sso_enabled": true,
        "sso_endpoint": "https://sso.demo-group.com/oauth2"
      }
    ]
  }
  ```

### 9.2 创建租户
- **POST** `/api/system/tenants`
- **请求体**: 同 9.1 中单个租户结构（不含 `id`、`token_used`、`created_at`）

### 9.3 更新租户配置
- **PUT** `/api/system/tenants/{tenant_id}`
- **请求体**: 同 9.1 中单个租户结构

### 9.4 切换租户状态
- **PATCH** `/api/system/tenants/{tenant_id}/toggle`

### 9.5 测试租户数据库连接
- **POST** `/api/system/tenants/{tenant_id}/test-jdbc`
- **响应**:
  ```json
  {
    "success": true,
    "message": "连接成功",
    "latency_ms": 45
  }
  ```

### 9.6 获取全局监控指标
- **GET** `/api/system/metrics`
- **响应**:
  ```json
  {
    "system_health": "healthy",
    "api_success_rate": 99.2,
    "avg_model_response_ms": 1250,
    "active_tenants": 3,
    "total_audits_today": 42,
    "uptime": "99.97%",
    "p95_latency": 2100,
    "total_requests_24h": 1847,
    "weekly_trend": [
      { "date": "06-04", "count": 35 }
    ],
    "alerts": [
      {
        "id": 1,
        "level": "warning | error | info",
        "message": "租户Token用量已达70%",
        "time": "10分钟前"
      }
    ]
  }
  ```

### 9.7 获取 OA 系统配置列表
- **GET** `/api/system/oa-configs`
- **响应**:
  ```json
  {
    "oa_systems": [
      {
        "id": "OA-001",
        "name": "泛微 E9",
        "type": "weaver_e9 | weaver_ebridge | zhiyuan_a8 | landray_ekp | custom",
        "type_label": "泛微 Ecology E9",
        "version": "v10.x",
        "status": "connected | disconnected | testing",
        "description": "说明文本",
        "adapter_version": "2.1.0",
        "last_sync": "2026-02-12 15:30:22",
        "sync_interval": 30,
        "enabled": true
      }
    ]
  }
  ```

### 9.8 切换 OA 系统启用状态
- **PATCH** `/api/system/oa-configs/{oa_id}/toggle`

### 9.9 测试 OA 系统连接
- **POST** `/api/system/oa-configs/{oa_id}/test`

### 9.10 获取 AI 模型配置列表
- **GET** `/api/system/ai-models`
- **响应**:
  ```json
  {
    "models": [
      {
        "id": "AI-001",
        "provider": "本地部署",
        "model_name": "Qwen2.5-72B",
        "display_name": "Qwen2.5-72B（本地）",
        "type": "local | cloud",
        "endpoint": "http://192.168.1.50:8000/v1",
        "api_key_configured": false,
        "max_tokens": 8192,
        "context_window": 131072,
        "cost_per_1k_tokens": 0,
        "status": "online | offline | maintenance",
        "enabled": true,
        "description": "模型说明",
        "capabilities": ["text", "code", "reasoning", "analysis"]
      }
    ]
  }
  ```

### 9.11 切换 AI 模型启用状态
- **PATCH** `/api/system/ai-models/{model_id}/toggle`

### 9.12 获取系统平台配置
- **GET** `/api/system/general-config`
- **响应**:
  ```json
  {
    "platform_name": "OA流程智能审核平台",
    "platform_version": "v1.2.0",
    "default_language": "zh-CN",
    "session_timeout": 120,
    "max_upload_size": 50,
    "enable_audit_trail": true,
    "enable_data_encryption": true,
    "backup_enabled": true,
    "backup_cron": "0 2 * * *",
    "backup_retention_days": 30,
    "notification_email": "admin@oa-smart-audit.com",
    "smtp_host": "smtp.example.com",
    "smtp_port": 465,
    "smtp_username": "noreply@oa-smart-audit.com",
    "smtp_ssl": true
  }
  ```

### 9.13 更新系统平台配置
- **PUT** `/api/system/general-config`
- **请求体**: 同 9.12 响应结构

---


## 10. 仪表盘统计

### 10.1 获取工作台统计
- **GET** `/api/dashboard/stats`
- **响应**:
  ```json
  {
    "todayAudits": 42,
    "todayApproved": 28,
    "todayRejected": 6,
    "todayRevised": 8,
    "pendingCount": 6,
    "avgResponseMs": 1850,
    "successRate": 99.2,
    "weeklyTrend": [
      { "date": "06-04", "count": 35 }
    ]
  }
  ```

---

## 通用约定

| 项目 | 说明 |
|------|------|
| 认证方式 | Bearer Token（JWT） |
| 基础路径 | `{API_BASE}/api/` |
| 分页参数 | `page`（从1开始）、`size`（默认20） |
| 错误响应 | `{ "code": 400, "message": "错误描述" }` |
| 时间格式 | `YYYY-MM-DD HH:mm:ss` |
| 敏感数据 | Go 层脱敏后传递给 AI 层 |
| 审计要求 | 所有审核记录不可篡改 |
