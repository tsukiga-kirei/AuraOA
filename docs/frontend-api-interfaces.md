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
    "department": "string",
    "position": "string",
    "email": "string",
    "phone": "string",
    "permissions": ["dashboard", "cron", "archive"]
  }
  ```

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
        "fail_count": 1
      }
    ]
  }
  ```

### 3.2 创建任务
- **POST** `/api/cron/tasks`
- **请求体**:
  ```json
  {
    "cron_expression": "0 9 * * 1-5",
    "task_type": "batch_audit"
  }
  ```

### 3.3 删除任务
- **DELETE** `/api/cron/tasks/{task_id}`

### 3.4 切换任务状态
- **PATCH** `/api/cron/tasks/{task_id}/toggle`

### 3.5 立即执行任务
- **POST** `/api/cron/tasks/{task_id}/execute`

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

---

## 7. 系统管理模块（系统管理员）

### 7.1 获取租户列表
- **GET** `/api/system/tenants`
- **响应**:
  ```json
  {
    "tenants": [
      {
        "id": "T-001",
        "name": "示例集团总部",
        "oa_type": "weaver_e9",
        "token_quota": 100000,
        "token_used": 42350,
        "max_concurrency": 20,
        "status": "active | inactive",
        "created_at": "2025-01-15"
      }
    ]
  }
  ```

### 7.2 创建租户
- **POST** `/api/system/tenants`

### 7.3 切换租户状态
- **PATCH** `/api/system/tenants/{tenant_id}/toggle`

### 7.4 获取全局监控指标
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

### 7.5 获取 OA 集成状态
- **GET** `/api/system/oa-status`
- **响应**:
  ```json
  {
    "connected": true,
    "oa_type": "weaver_e9",
    "connection_method": "JDBC",
    "sync_interval_seconds": 30,
    "last_sync_at": "2025-06-10 15:30:22",
    "version": "E9 v10.x"
  }
  ```

---

## 8. 仪表盘统计

### 8.1 获取工作台统计
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
