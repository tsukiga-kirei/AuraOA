-- 005_org_roles.sql
-- Seed data: system org roles with page_permissions (path string array format)
-- Run after 002_tenants.sql
-- Role names, descriptions, and page_permissions match TenantService.CreateTenant defaults

-- Fixed UUIDs for referential integrity:
-- OrgRoles (DEMO_HQ tenant a0000000-0000-0000-0000-000000000001):
--   d0000000-0000-0000-0000-000000000001  业务用户
--   d0000000-0000-0000-0000-000000000002  审计管理员
--   d0000000-0000-0000-0000-000000000003  租户管理员
--
-- OrgRoles (DEMO_BR1 tenant a0000000-0000-0000-0000-000000000002):
--   d0000000-0000-0000-0000-000000000004  业务用户
--   d0000000-0000-0000-0000-000000000005  审计管理员
--   d0000000-0000-0000-0000-000000000006  租户管理员

-- ============================================================
-- DEMO_HQ 角色
-- ============================================================
INSERT INTO org_roles (id, tenant_id, name, description, page_permissions, is_system)
VALUES
    (
        'd0000000-0000-0000-0000-000000000001',
        'a0000000-0000-0000-0000-000000000001',
        '业务用户',
        '普通业务人员，可使用审核工作台等前台功能。仪表盘为所有角色默认拥有。',
        '["/overview", "/dashboard", "/settings"]',
        TRUE
    ),
    (
        'd0000000-0000-0000-0000-000000000002',
        'a0000000-0000-0000-0000-000000000001',
        '审计管理员',
        '在业务用户基础上，额外拥有归档复盘权限，可进行合规复核。',
        '["/overview", "/dashboard", "/cron", "/archive", "/settings"]',
        TRUE
    ),
    (
        'd0000000-0000-0000-0000-000000000003',
        'a0000000-0000-0000-0000-000000000001',
        '租户管理员',
        '可进入后台管理，配置规则、组织人员、数据信息、用户偏好。',
        '["/overview", "/dashboard", "/cron", "/archive", "/settings", "/admin/tenant/rules", "/admin/tenant/org", "/admin/tenant/data", "/admin/tenant/user-configs"]',
        TRUE
    );

-- ============================================================
-- DEMO_BR1 角色（与 CreateTenant 事务生成的默认角色一致）
-- ============================================================
INSERT INTO org_roles (id, tenant_id, name, description, page_permissions, is_system)
VALUES
    (
        'd0000000-0000-0000-0000-000000000004',
        'a0000000-0000-0000-0000-000000000002',
        '业务用户',
        '普通业务人员，可使用审核工作台等前台功能。仪表盘为所有角色默认拥有。',
        '["/overview", "/dashboard", "/settings"]',
        TRUE
    ),
    (
        'd0000000-0000-0000-0000-000000000005',
        'a0000000-0000-0000-0000-000000000002',
        '审计管理员',
        '在业务用户基础上，额外拥有归档复盘权限，可进行合规复核。',
        '["/overview", "/dashboard", "/cron", "/archive", "/settings"]',
        TRUE
    ),
    (
        'd0000000-0000-0000-0000-000000000006',
        'a0000000-0000-0000-0000-000000000002',
        '租户管理员',
        '可进入后台管理，配置规则、组织人员、数据信息、用户偏好。',
        '["/overview", "/dashboard", "/cron", "/archive", "/settings", "/admin/tenant/rules", "/admin/tenant/org", "/admin/tenant/data", "/admin/tenant/user-configs"]',
        TRUE
    );
