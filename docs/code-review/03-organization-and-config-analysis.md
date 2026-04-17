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
| tenant.default_token_quota | 10000 | 新租户默认 Token 配额 |
| tenant.default_max_concurrency | 10 | 新租户默认最大并发数 |

**⚠️ 问题**: 如前文所述，auth.* 配置未被 JWT 生成代码使用。

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
    "allow_custom_fields": true,      // 允许用户自定义字段
    "allow_custom_rules": true,       // 允许用户自定义规则
    "allow_modify_strictness": true   // 允许用户修改审核尺度
}
```

**access_control 结构**:
```json
{
    "allowed_roles": ["role-uuid-1"],       // 允许的组织角色
    "allowed_members": ["member-uuid-1"],   // 允许的成员
    "allowed_departments": ["dept-uuid-1"]  // 允许的部门
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

## 4. 发现的问题

### 🟡 问题 1: 租户管理员保护逻辑分散

**严重程度**: 低

**问题描述**:
租户管理员的保护逻辑在多处重复实现：

```go
// org_service.go - UpdateMember
if req.Status == "disabled" {
    var tenant model.Tenant
    if err := s.db.Where("admin_user_id = ? AND id = ?", member.UserID, member.TenantID).First(&tenant).Error; err == nil {
        return nil, newServiceError(errcode.ErrParamValidation, "该成员是租户管理员，不允许禁用。")
    }
}

// org_service.go - DeleteMember
var tenant model.Tenant
if err := s.db.Where("admin_user_id = ? AND id = ?", member.UserID, member.TenantID).First(&tenant).Error; err == nil {
    return newServiceError(errcode.ErrParamValidation, "该成员是租户管理员，不允许删除。")
}
```

**建议**: 抽取为独立方法 `isTenantAdmin(userID, tenantID)`。

---

### 🟡 问题 2: 组织成员创建时的默认密码

**严重程度**: 中

**问题描述**:
创建组织成员时，如果未提供密码，使用硬编码的默认密码：

```go
// org_service.go - CreateMember
password := req.Password
if password == "" {
    password = "123456"  // ⚠️ 硬编码默认密码
}
```

**风险**:
- 安全风险：默认密码过于简单
- 用户可能忘记修改密码

**建议**:
1. 从系统配置读取默认密码策略
2. 强制用户首次登录时修改密码
3. 或生成随机密码并通过邮件发送

---

### 🟢 问题 3: 成员信息同步

**严重程度**: 低（已处理）

**代码位置**: `org_service.go - UpdateMember`

```go
// 同步更新 users 表字段
userUpdates := map[string]interface{}{}
if req.DisplayName != "" { userUpdates["display_name"] = req.DisplayName }
if req.Email != "" { userUpdates["email"] = req.Email }
if req.Phone != "" { userUpdates["phone"] = req.Phone }
if req.Status != "" { userUpdates["status"] = req.Status }

// 反向同步：如果该成员是租户管理员，同步更新租户表的联系人信息
var adminTenant model.Tenant
if err := s.db.Where("admin_user_id = ? AND id = ?", member.UserID, member.TenantID).First(&adminTenant).Error; err == nil {
    tenantUpdates := map[string]interface{}{}
    if req.DisplayName != "" { tenantUpdates["contact_name"] = req.DisplayName }
    if req.Email != "" { tenantUpdates["contact_email"] = req.Email }
    if req.Phone != "" { tenantUpdates["contact_phone"] = req.Phone }
}
```

**✅ 已正确处理**: 成员信息更新时会同步到 users 表和租户联系人信息。

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

### ⚠️ 待改进

1. 默认密码应可配置
2. 租户管理员保护逻辑应抽取
3. 建议添加首次登录强制改密机制

---

## 8. 建议优化项

| 优先级 | 优化项 | 说明 |
|-------|-------|------|
| P1 | 默认密码可配置化 | 从系统配置读取或生成随机密码 |
| P1 | 首次登录强制改密 | 增加 password_must_change 标志 |
| P2 | 抽取租户管理员检查 | 减少代码重复 |
| P3 | 添加操作审计日志 | 记录组织结构变更 |
