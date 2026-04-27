# 人员组织与配置系统分析报告

## 1. 概述

本文档分析 OA 智审系统的人员组织结构、系统配置、租户配置等模块的设计与实现。

---

## 2. 数据模型关系

### 2.1 核心实体关系图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                          实体关系图                                      │
├─────────────────────────────────────────────────────────────────────────┤
│                                                                         │
│  ┌─────────────┐                                                        │
│  │   Tenant    │◄─────────────────────────────────────────┐            │
│  │   (租户)    │                                          │            │
│  └──────┬──────┘                                          │            │
│         │                                                  │            │
│         │ 1:N                                              │            │
│         ▼                                                  │            │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐ │            │
│  │ Department  │◄────│  OrgMember  │────►│    User     │ │            │
│  │   (部门)    │ N:1 │ (组织成员)  │ N:1 │   (用户)    │ │            │
│  └─────────────┘     └──────┬──────┘     └──────┬──────┘ │            │
│                             │                    │        │            │
│                             │ M:N                │        │            │
│                             ▼                    │        │            │
│                      ┌─────────────┐            │        │            │
│                      │  OrgRole    │            │        │            │
│                      │ (组织角色)  │            │        │            │
│                      └─────────────┘            │        │            │
│                                                  │        │            │
│                                                  │ 1:N    │            │
│                                                  ▼        │            │
│                                          ┌─────────────┐ │            │
│                                          │ UserRole    │ │            │
│                                          │ Assignment  │─┘            │
│                                          │ (系统角色)  │              │
│                                          └─────────────┘              │
│                                                                         │
│  ┌─────────────┐     ┌─────────────┐     ┌─────────────┐              │
│  │ ProcessAudit│────►│ AuditRule   │     │UserPersonal │              │
│  │   Config    │ 1:N │ (审核规则)  │     │   Config    │              │
│  │ (流程配置)  │     └─────────────┘     │ (个人配置)  │              │
│  └─────────────┘                         └─────────────┘              │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
```

### 2.2 角色体系

系统采用双层角色体系：

| 层级 | 角色类型 | 存储位置 | 说明 |
|-----|---------|---------|------|
| 系统级 | system_admin / tenant_admin / business | user_role_assignments | 控制系统功能访问 |
| 组织级 | 自定义角色 | org_roles + org_member_roles | 控制页面权限 |

**系统角色推断逻辑** (`org_service.go`):

```go
func (s *OrgService) syncUserSystemRoles(userID uuid.UUID, tenantID uuid.UUID, displayName string, roles []model.OrgRole) error {
    // 根据 page_permissions 推断需要的系统角色
    // 包含前台页面（非 /admin/ 前缀）→ business
    // 包含后台页面（/admin/ 前缀）→ tenant_admin
    
    for _, role := range roles {
        var paths []string
        json.Unmarshal(role.PagePermissions, &paths)
        for _, p := range paths {
            if strings.HasPrefix(p, "/admin/") {
                needTenantAdmin = true
            } else {
                needBusiness = true
            }
        }
    }
    
    // 至少给予 business 权限
    if !needBusiness && !needTenantAdmin {
        needBusiness = true
    }
}
```

**✅ 设计合理**: 系统角色自动从组织角色的页面权限推断，减少手动配置。

---

## 3. 配置层级

### 3.1 配置优先级

```
系统配置 (system_configs)
    │
    ▼
租户配置 (tenants + process_audit_configs)
    │
    ▼
用户个人配置 (user_personal_configs)
```

### 3.2 系统配置

**表**: `system_configs`

| 配置键 | 默认值 | 说明 |
|-------|-------|------|
| auth.login_fail_lock_threshold | 5 | 登录失败锁定阈值 |
| auth.account_lock_minutes | 15 | 账户锁定时长 |
| auth.access_token_ttl_hours | 2 | Access Token 有效期 |
| auth.refresh_token_ttl_days | 7 | Refresh Token 有效期 |
| auth.default_password | Audit@2026 | 新建成员默认密码 |
| tenant.default_token_quota | 10000 | 新租户默认 Token 配额 |
| tenant.default_max_concurrency | 10 | 新租户默认最大并发数 |

**✅ 已修复**: auth.* 配置已被 JWT 生成代码和成员创建逻辑正确使用。

### 3.3 租户配置

**表**: `tenants`

```sql
CREATE TABLE tenants (
    id                  UUID PRIMARY KEY,
    name                VARCHAR(255) NOT NULL,
    code                VARCHAR(100) NOT NULL UNIQUE,
    status              VARCHAR(20) DEFAULT 'active',
    oa_db_connection_id UUID,                          -- 关联 OA 连接
    token_quota         INT DEFAULT 10000,             -- Token 配额
    token_used          INT DEFAULT 0,                 -- 已用 Token
    max_concurrency     INT DEFAULT 10,                -- 最大并发
    admin_user_id       UUID,                          -- 租户管理员
    -- ...
);
```

### 3.4 流程审核配置

**表**: `process_audit_configs`

```sql
CREATE TABLE process_audit_configs (
    id                 UUID PRIMARY KEY,
    tenant_id          UUID NOT NULL,
    process_type       VARCHAR(200) NOT NULL,
    process_type_label VARCHAR(200),
    main_table_name    VARCHAR(200),
    main_fields        JSONB DEFAULT '[]',      -- 主表字段配置
    detail_tables      JSONB DEFAULT '[]',      -- 明细表配置
    field_mode         VARCHAR(20) DEFAULT 'all',
    kb_mode            VARCHAR(20) DEFAULT 'rules_only',
    ai_config          JSONB DEFAULT '{}',      -- AI 配置
    user_permissions   JSONB DEFAULT '{}',      -- 用户权限控制
    access_control     JSONB DEFAULT '{}',      -- 访问控制
    status             VARCHAR(20) DEFAULT 'active',
    UNIQUE(tenant_id, process_type)
);
```

**user_permissions 结构**:
```json
{
    "allow_custom_fields": true,
    "allow_custom_rules": true,
    "allow_modify_strictness": true
}
```

**access_control 结构**:
```json
{
    "allowed_roles": ["role-uuid-1"],
    "allowed_members": ["member-uuid-1"],
    "allowed_departments": ["dept-uuid-1"]
}
```

### 3.5 用户个人配置

**表**: `user_personal_configs`

```sql
CREATE TABLE user_personal_configs (
    id              UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    audit_details   JSONB DEFAULT '[]',    -- 审核工作台个人配置
    cron_details    JSONB DEFAULT '{}',    -- 定时任务个人配置
    archive_details JSONB DEFAULT '[]',    -- 归档复盘个人配置
    UNIQUE(tenant_id, user_id)
);
```

**audit_details 结构**:
```json
[
    {
        "config_id": "uuid",
        "process_type": "采购审批",
        "field_config": {
            "field_mode": "selected",
            "field_overrides": ["main:field1", "detail1:field2"]
        },
        "rule_config": {
            "custom_rules": [
                {"id": "custom-1", "content": "...", "enabled": true}
            ],
            "rule_toggle_overrides": [
                {"rule_id": "tenant-rule-1", "enabled": false}
            ]
        },
        "ai_config": {
            "strictness_override": "strict"
        }
    }
]
```

---

## 4. 发现的问题与修复

### ✅ 问题 1: 租户管理员保护逻辑分散（已修复）

**严重程度**: 低

**原问题描述**:
租户管理员的保护逻辑在 `UpdateMember` 和 `DeleteMember` 中各自内联查询 `tenants` 表，代码重复。

**修复内容**:
- 抽取为独立辅助方法 `isTenantAdmin(userID, tenantID) bool`
- `UpdateMember` 和 `DeleteMember` 统一调用该方法
- 反向同步联系人信息也复用该方法判断

```go
// isTenantAdmin 检查指定用户是否为指定租户的管理员。
func (s *OrgService) isTenantAdmin(userID uuid.UUID, tenantID uuid.UUID) bool {
    var tenant model.Tenant
    err := s.db.Where("admin_user_id = ? AND id = ?", userID, tenantID).First(&tenant).Error
    return err == nil
}
```

**修改文件**: `go-service/internal/service/org_service.go`

**状态**: ✅ 已修复

---

### ✅ 问题 2: 组织成员创建时的默认密码（已修复）

**严重程度**: 中

**原问题描述**:
创建组织成员时使用硬编码的默认密码 `"123456"`，安全风险高。

**修复内容**:
- 优先从 `system_configs` 表读取 `auth.default_password` 配置
- 降级使用更安全的默认密码 `Audit@2026`
- 新增数据库迁移 `000032_auth_default_password_config` 添加配置项
- 管理员可在系统设置中随时修改默认密码

```go
password := req.Password
if password == "" {
    if defaultPwd, err := s.systemConfigRepo.FindByKey("auth.default_password"); err == nil && defaultPwd != "" {
        password = defaultPwd
    } else {
        password = "Audit@2026"
    }
}
```

**修改文件**:
- `go-service/internal/service/org_service.go`
- `db/migrations/000032_auth_default_password_config.up.sql`
- `db/migrations/000032_auth_default_password_config.down.sql`

**状态**: ✅ 已修复

---

### ✅ 问题 3: 后端错误消息国际化（已修复）

**严重程度**: 中

**原问题描述**:
`org_service.go` 中所有 `newServiceError` 的 message 参数为硬编码中文字符串（如 "数据库错误"、"参数校验失败"），不支持多语言环境。

**修复内容**:
- 后端错误消息统一改为英文（作为 fallback 和日志标识）
- 前端 `useAuth.ts` 的 `ERROR_CODE_I18N_MAP` 新增 `40001`（参数校验）、`40900`（资源冲突）、`50001`（数据库错误）映射
- 前端 `zh-CN.ts` / `en-US.ts` 新增对应翻译键
- 前端根据错误码自动匹配当前语言的翻译文案，后端 message 仅作降级显示

**修改文件**:
- `go-service/internal/service/org_service.go`
- `go-service/internal/handler/org_handler.go`
- `frontend/composables/useAuth.ts`
- `frontend/locales/zh-CN.ts`
- `frontend/locales/en-US.ts`

**状态**: ✅ 已修复

---

### ✅ 问题 4: 日志消息国际化统一（已修复）

**严重程度**: 低

**原问题描述**:
`org_service.go` 中的 `pkglogger.Global().Info(...)` 日志消息为中文（如 "成员创建成功"），在英文环境下不利于日志检索和监控告警匹配。

**修复内容**:
- 所有结构化日志消息统一改为英文（如 `"member created"`、`"department deleted"`）
- 日志字段名保持不变（`memberID`、`tenantID` 等）

**状态**: ✅ 已修复

---

### 🟢 问题 5: 成员信息同步

**严重程度**: 低（已正确处理）

**代码位置**: `org_service.go - UpdateMember`

成员信息更新时会同步到 users 表和租户联系人信息，且反向同步逻辑已改用 `isTenantAdmin()` 方法判断。

**✅ 设计合理**: 无需额外修改。

---

## 5. 菜单权限系统

### 5.1 菜单生成逻辑

**文件**: `auth_service.go - GetMenu`

```go
func (s *AuthService) GetMenu(activeRole jwtpkg.ActiveRoleClaim, userID string, tenantID string) (*dto.MenuResponse, error) {
    switch activeRole.Role {
    case "system_admin":
        // 硬编码菜单
        return &dto.MenuResponse{
            Menus: []dto.MenuItem{
                {Key: "tenant-management", Label: "租户管理", Path: "/admin/system/tenants"},
                {Key: "system-settings", Label: "系统设置", Path: "/admin/system/settings"},
            },
        }, nil

    case "tenant_admin", "business":
        // 从 org_roles.page_permissions 动态读取
        return s.getMenuFromOrgRoles(userID, tenantID, activeRole.Role)
    }
}
```

### 5.2 页面权限过滤

```go
// 按系统角色过滤：
// tenant_admin 只看后台管理页面
// business 只看前台业务页面
if activeSystemRole == "tenant_admin" {
    for _, m := range menus {
        if strings.HasPrefix(m.Path, "/admin/tenant/") || m.Path == "/overview" || m.Path == "/settings" {
            filtered = append(filtered, m)
        }
    }
} else if activeSystemRole == "business" {
    for _, m := range menus {
        if !strings.HasPrefix(m.Path, "/admin/") {
            filtered = append(filtered, m)
        }
    }
}
```

**✅ 设计合理**: 菜单权限基于组织角色的 page_permissions，并按系统角色过滤。

---

## 6. 数据关联性分析

### 6.1 级联删除关系

| 父表 | 子表 | 删除行为 |
|-----|------|---------|
| tenants | departments | CASCADE |
| tenants | org_roles | CASCADE |
| tenants | org_members | CASCADE |
| tenants | process_audit_configs | CASCADE |
| tenants | user_personal_configs | CASCADE |
| users | org_members | CASCADE |
| users | user_role_assignments | CASCADE |
| departments | org_members | RESTRICT (需先迁移成员) |
| process_audit_configs | audit_rules | CASCADE |

### 6.2 数据一致性保障

1. **租户删除**: 级联删除所有关联数据
2. **用户删除**: 级联删除组织成员和角色分配
3. **部门删除**: 需先迁移成员（RESTRICT）
4. **流程配置删除**: 级联删除关联规则

**✅ 设计合理**: 外键约束确保数据一致性。

---

## 7. 代码质量评估

### ✅ 优点

1. **完善的外键约束**: 数据库层面保证数据一致性
2. **双层角色体系**: 系统角色 + 组织角色，灵活且清晰
3. **自动角色推断**: 从页面权限自动推断系统角色
4. **信息同步机制**: 成员信息变更自动同步到相关表
5. **租户管理员保护**: 统一 `isTenantAdmin()` 方法，逻辑集中
6. **默认密码可配置**: 从 `system_configs` 读取，管理员可随时修改
7. **错误消息国际化**: 后端返回错误码，前端按语言映射翻译
8. **字段合并逻辑独立**: `MergeFields` 函数抽取至 `field_merge.go`，可独立测试

### ⚠️ 待改进

1. 首次登录强制改密机制（`password_must_change` 标志）
2. 添加操作审计日志（记录组织结构变更）

---

## 8. 建议优化项

| 优先级 | 优化项 | 说明 | 状态 |
|-------|-------|------|------|
| P1 | 默认密码可配置化 | 从系统配置读取 `auth.default_password` | ✅ 已完成 |
| P1 | 首次登录强制改密 | 增加 `password_must_change` 标志 | 📋 待修复 |
| P2 | 抽取租户管理员检查 | `isTenantAdmin()` 辅助方法 | ✅ 已完成 |
| P2 | 后端错误消息国际化 | 错误码 + 前端 i18n 映射 | ✅ 已完成 |
| P2 | 日志消息统一英文 | 便于日志检索和监控 | ✅ 已完成 |
| P2 | 字段合并逻辑重构 | 抽取为 `field_merge.go` | ✅ 已完成 |
| P3 | 添加操作审计日志 | 记录组织结构变更 | 📋 待修复 |
