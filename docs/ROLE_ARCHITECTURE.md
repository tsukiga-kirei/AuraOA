# 角色与权限架构设计文档

> OA 智能审核平台 — 多租户多角色权限体系

---

## 1. 核心概念

### 1.1 三级角色体系

| 角色类型 | 英文标识 | 作用域 | 说明 |
|----------|----------|--------|------|
| 业务用户 | `business` | 租户级 | 使用审核工作台、定时任务、归档复盘等业务功能 |
| 租户管理员 | `tenant_admin` | 租户级 | 管理本租户的规则配置、组织人员、数据信息、用户偏好 |
| 系统管理员 | `system_admin` | 全局 | 管理所有租户、系统设置、全局监控 |

### 1.2 关键设计原则

1. **角色绑定租户**：`business` 和 `tenant_admin` 角色必须关联具体租户（tenant_id），不存在脱离租户的业务角色
2. **一人多角色**：同一用户可以在不同租户下拥有不同角色（多对多关系）
3. **角色独立切换**：切换角色后，只展示该角色对应的菜单和数据，不显示其他角色的内容
4. **系统管理员全局性**：系统管理员是全局角色，不绑定具体租户

### 1.3 与旧版本的区别

| 维度 | 旧版本 | 新版本 |
|------|--------|--------|
| 权限模型 | 扁平数组 `['business', 'tenant_admin']` | 角色分配列表 `UserRoleAssignment[]` |
| 租户关联 | 用户只属于一个租户 | 用户可在多个租户下拥有角色 |
| 角色切换 | 切换权限组（会丢失信息） | 切换具体角色分配（保留完整信息） |
| 菜单显示 | 混合展示所有权限的菜单 | 仅展示当前角色的菜单 |

---

## 2. 数据模型

### 2.1 角色分配 (UserRoleAssignment)

每个用户拥有一个"角色分配列表"，每条记录表示用户在某个上下文中的具体角色。

```typescript
interface UserRoleAssignment {
  /** 唯一标识 */
  id: string
  /** 角色类型 */
  role: 'business' | 'tenant_admin' | 'system_admin'
  /** 租户ID（system_admin 为 null）*/
  tenant_id: string | null
  /** 租户名称（用于展示）*/
  tenant_name: string | null
  /** 展示标签，如 "示例集团总部 · 业务用户" */
  label: string
}
```

### 2.2 用户模型 (MockUser)

```typescript
interface MockUser {
  username: string
  password: string
  display_name: string
  /** 用户拥有的所有角色分配 */
  roles: UserRoleAssignment[]
}
```

### 2.3 活跃角色状态

前端运行时需要维护以下状态：

```typescript
/** 用户拥有的全部角色分配列表（登录后不变） */
allRoles: UserRoleAssignment[]

/** 当前活跃的角色分配 */
activeRole: UserRoleAssignment

/** 基于当前活跃角色生成的权限组（用于菜单过滤） */
userPermissions: PermissionGroup[]  // 只有一个元素
```

---

## 3. 角色场景举例

### 3.1 场景矩阵

| 用户 | 角色分配 | 说明 |
|------|---------|------|
| 张明 | 示例集团总部 · 业务用户 | 普通员工，只有一个角色 |
| 李芳 | 示例集团总部 · 业务用户<br>华东分公司 · 业务用户 | 在两个租户都有业务权限 |
| 赵伟 | 示例集团总部 · 租户管理员<br>示例集团总部 · 业务用户 | 租户管理员兼业务用户 |
| 王刚 | 华东分公司 · 租户管理员<br>示例集团总部 · 业务用户 | 华东分公司管理员 + 总部业务用户 |
| 陈刚 | 系统管理员<br>示例集团总部 · 租户管理员<br>示例集团总部 · 业务用户 | 超级管理员，全部权限 |
| 周敏 | 系统管理员<br>华东分公司 · 租户管理员 | 系统管理员 + 华东分公司管理 |
| 吴强 | 系统管理员 | 纯系统管理员 |

### 3.2 切换行为

| 激活角色 | 看到的菜单 | 数据范围 |
|----------|-----------|----------|
| 示例集团总部 · 业务用户 | 仪表盘、审核工作台、定时任务、归档复盘、个人设置 | 示例集团总部的数据 |
| 华东分公司 · 业务用户 | 同上 | 华东分公司的数据 |
| 示例集团总部 · 租户管理员 | 仪表盘、规则配置、组织人员、数据信息、用户偏好 | 示例集团总部的管理数据 |
| 系统管理员 | 仪表盘、全局监控、租户管理、系统设置 | 全局数据 |

> **注意**：切换到租户管理员时，**不展示**业务用户的菜单（工作台、定时任务等），即使该用户同时拥有业务角色。用户需要切回业务角色才能看到业务菜单。

---

## 4. 权限与菜单映射

### 4.1 菜单 → 所需权限组

```
/overview          → 所有角色可见（通用仪表盘）
/dashboard         → business
/cron              → business
/archive           → business
/settings          → 所有角色可见（个人设置）
/admin/tenant/rules → tenant_admin
/admin/tenant/org  → tenant_admin
/admin/tenant/data → tenant_admin
/admin/tenant/user-configs → tenant_admin
/admin/system/tenants → system_admin
/admin/system/settings → system_admin
```

### 4.2 菜单生成规则

切换角色时，根据该角色的 `role` 类型生成对应菜单：

| 角色类型 | 展示的菜单分组 |
|----------|---------------|
| `business` | 仪表盘 + 业务菜单（工作台、定时任务、归档复盘）+ 个人设置 |
| `tenant_admin` | 仪表盘 + 租户管理菜单（规则配置、组织人员、数据信息、用户偏好）|
| `system_admin` | 仪表盘 + 系统管理菜单（全局监控、租户管理、系统设置）|

---

## 5. 前端状态管理

### 5.1 登录流程

```
1. 用户输入凭据 → 后端返回用户信息 + 角色分配列表
2. 存储 allRoles（全部角色）到 state 和 localStorage
3. 自动激活第一个角色分配 → 设为 activeRole
4. 根据 activeRole.role 生成对应菜单
5. 跳转到该角色的默认页面
```

### 5.2 角色切换流程

```
1. 用户在 Header 点击角色切换器
2. 展示 allRoles 中所有角色，按类型分组显示
3. 用户选择一个角色 → 设为 activeRole
4. 重新生成菜单（仅该角色对应的菜单）
5. 跳转到新角色的默认页面
6. 更新 localStorage
```

### 5.3 角色切换器 UI 设计

下拉面板通过 **hover** 触发，分组顺序为业务用户优先：

```
┌─────────────────────────────────────┐
│ 切换角色                             │
│─────────────────────────────────────│
│ 📊 业务操作                          │
│   ├ 示例集团总部 · 业务用户       ✓  │
│   ├ 华东分公司 · 业务用户            │
│─────────────────────────────────────│
│ ⚙️ 租户管理                         │
│   ├ 示例集团总部 · 租户管理员        │
│   ├ 华东分公司 · 租户管理员          │
│─────────────────────────────────────│
│ 🛡️ 系统管理                         │
│   ├ 系统管理员                       │
└─────────────────────────────────────┘
```

---

## 6. 后端 API 规范

### 6.1 登录接口返回

```json
POST /api/auth/login
Response: {
  "access_token": "...",
  "refresh_token": "...",
  "user": {
    "username": "admin",
    "display_name": "陈刚",
    "roles": [
      {
        "id": "role-1",
        "role": "system_admin",
        "tenant_id": null,
        "tenant_name": null,
        "label": "系统管理员"
      },
      {
        "id": "role-2",
        "role": "tenant_admin",
        "tenant_id": "T-001",
        "tenant_name": "示例集团总部",
        "label": "示例集团总部 · 租户管理员"
      },
      {
        "id": "role-3",
        "role": "business",
        "tenant_id": "T-001",
        "tenant_name": "示例集团总部",
        "label": "示例集团总部 · 业务用户"
      }
    ]
  }
}
```

### 6.2 角色切换接口

```json
POST /api/auth/switch-role
Request: {
  "role_id": "role-2"
}
Response: {
  "active_role": { ... },
  "menus": [ ... ],
  "default_page": "/overview"
}
```

### 6.3 数据接口的租户上下文

所有业务和租户管理接口都需要在请求头中带上当前活跃的租户ID：

```
X-Tenant-Id: T-001
```

后端根据此 header 过滤数据。系统管理员角色不需要此 header（或值为 `null`）。

---

## 7. 数据库设计建议

### 7.1 用户角色关联表

```sql
CREATE TABLE user_role_assignments (
  id          VARCHAR(36) PRIMARY KEY,
  user_id     VARCHAR(36) NOT NULL,
  role        ENUM('business', 'tenant_admin', 'system_admin') NOT NULL,
  tenant_id   VARCHAR(36) NULL,        -- NULL for system_admin
  created_at  TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  created_by  VARCHAR(36),
  
  FOREIGN KEY (user_id) REFERENCES users(id),
  FOREIGN KEY (tenant_id) REFERENCES tenants(id),
  
  -- 同一用户在同一租户下不能有相同角色
  UNIQUE KEY uk_user_role_tenant (user_id, role, tenant_id)
);
```

### 7.2 查询用户的全部角色

```sql
SELECT 
  ura.id,
  ura.role,
  ura.tenant_id,
  t.name AS tenant_name,
  CASE 
    WHEN ura.role = 'system_admin' THEN '系统管理员'
    WHEN ura.role = 'tenant_admin' THEN CONCAT(t.name, ' · 租户管理员')
    WHEN ura.role = 'business' THEN CONCAT(t.name, ' · 业务用户')
  END AS label
FROM user_role_assignments ura
LEFT JOIN tenants t ON ura.tenant_id = t.id
WHERE ura.user_id = ?
ORDER BY 
  FIELD(ura.role, 'system_admin', 'tenant_admin', 'business'),
  t.name;
```

---

## 8. 边界情况与约束

### 8.1 约束规则

1. `system_admin` 的 `tenant_id` 必须为 `null`
2. `business` 和 `tenant_admin` 的 `tenant_id` 不能为 `null`
3. 同一用户在同一租户下不能同时创建两个 `business` 角色（通过唯一约束保证）
4. 删除租户时，需级联删除/停用该租户下所有角色分配
5. 用户至少有一个角色分配，否则无法登录

### 8.2 默认角色选择（登录后）

登录后自动激活角色的优先级：
1. 如果用户有 `system_admin` 角色 → 激活系统管理员
2. 否则如果有 `tenant_admin` 角色 → 激活第一个租户管理员
3. 否则 → 激活第一个业务用户

### 8.3 角色切换器显示条件

- 仅当 `allRoles.length > 1` 时才显示角色切换按钮
- 当只有一个角色时，不显示切换按钮

### 8.4 跨租户数据隔离

- 切换到"示例集团总部 · 业务用户"后，所有业务接口只返回示例集团总部的数据
- 切换到"华东分公司 · 业务用户"后，只返回华东分公司的数据
- 系统管理员可以看到所有租户的数据

---

## 9. Mock 用户数据

### 9.1 测试账号清单

| 账号 | 密码 | 角色数量 | 角色列表 | 测试场景 |
|------|------|---------|---------|---------|
| admin | 123456 | 3 | 系统管理员 + 总部租户管理员 + 总部业务用户 | 全角色用户 |
| sysadmin2 | 123456 | 2 | 系统管理员 + 华东租户管理员 | 系统管理 + 分公司管理 |
| sysadmin3 | 123456 | 1 | 系统管理员 | 纯系统管理 |
| tenantadmin | 123456 | 2 | 总部租户管理员 + 总部业务用户 | 租户管理员兼业务 |
| wanggang | 123456 | 2 | 华东租户管理员 + 总部业务用户 | 跨租户角色 |
| zhangming | 123456 | 1 | 总部业务用户 | 单角色用户 |
| lifang | 123456 | 2 | 总部业务用户 + 华东业务用户 | 多租户业务 |
| user | 123456 | 1 | 总部业务用户 | 单角色测试 |
