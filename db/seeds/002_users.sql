-- 002_users.sql
-- Seed data: users and user_role_assignments
-- Run after 001_tenants.sql

-- Fixed UUIDs for referential integrity:
-- Users:
--   b0000000-0000-0000-0000-000000000001  admin (system_admin)
--   b0000000-0000-0000-0000-000000000002  tenant_admin_user (tenant_admin for DEMO_HQ)
--   b0000000-0000-0000-0000-000000000003  reviewer01 (business user)
--   b0000000-0000-0000-0000-000000000004  reviewer02 (business user)
--   b0000000-0000-0000-0000-000000000005  supervisor01 (business user)
--
-- Password placeholder: bcrypt hash of "123456"
--   $2a$12$RI08DxemoYuiefF0PjWXkeOV9MlLHSeLVxI32rjGjKkQETh6UuT/e

-- ============================================================
-- users
-- ============================================================
INSERT INTO users (id, username, password_hash, display_name, email, phone, status)
VALUES
    (
        'b0000000-0000-0000-0000-000000000001',
        'admin',
        '$2a$12$RI08DxemoYuiefF0PjWXkeOV9MlLHSeLVxI32rjGjKkQETh6UuT/e',
        '系统管理员',
        'admin@example.com',
        '13900000001',
        'active'
    ),
    (
        'b0000000-0000-0000-0000-000000000002',
        'tenant_admin_user',
        '$2a$12$RI08DxemoYuiefF0PjWXkeOV9MlLHSeLVxI32rjGjKkQETh6UuT/e',
        '租户管理员',
        'tenant_admin@example.com',
        '13900000002',
        'active'
    ),
    (
        'b0000000-0000-0000-0000-000000000003',
        'reviewer01',
        '$2a$12$RI08DxemoYuiefF0PjWXkeOV9MlLHSeLVxI32rjGjKkQETh6UuT/e',
        '审核员张三',
        'reviewer01@example.com',
        '13900000003',
        'active'
    ),
    (
        'b0000000-0000-0000-0000-000000000004',
        'reviewer02',
        '$2a$12$RI08DxemoYuiefF0PjWXkeOV9MlLHSeLVxI32rjGjKkQETh6UuT/e',
        '审核员李四',
        'reviewer02@example.com',
        '13900000004',
        'active'
    ),
    (
        'b0000000-0000-0000-0000-000000000005',
        'supervisor01',
        '$2a$12$RI08DxemoYuiefF0PjWXkeOV9MlLHSeLVxI32rjGjKkQETh6UuT/e',
        '审核主管王五',
        'supervisor01@example.com',
        '13900000005',
        'active'
    );

-- ============================================================
-- user_role_assignments
-- ============================================================
-- admin: system_admin (no tenant) + tenant_admin for DEMO_HQ + business for DEMO_HQ
INSERT INTO user_role_assignments (id, user_id, role, tenant_id, label, is_default)
VALUES
    (
        'f0000000-0000-0000-0000-000000000001',
        'b0000000-0000-0000-0000-000000000001',
        'system_admin',
        NULL,
        '系统管理员',
        TRUE
    ),
    (
        'f0000000-0000-0000-0000-000000000002',
        'b0000000-0000-0000-0000-000000000001',
        'tenant_admin',
        'a0000000-0000-0000-0000-000000000001',
        '演示总部 - 租户管理员',
        FALSE
    ),
    (
        'f0000000-0000-0000-0000-000000000008',
        'b0000000-0000-0000-0000-000000000001',
        'business',
        'a0000000-0000-0000-0000-000000000001',
        '演示总部 - 业务用户',
        FALSE
    );

-- tenant_admin_user: tenant_admin for DEMO_HQ
INSERT INTO user_role_assignments (id, user_id, role, tenant_id, label, is_default)
VALUES
    (
        'f0000000-0000-0000-0000-000000000003',
        'b0000000-0000-0000-0000-000000000002',
        'tenant_admin',
        'a0000000-0000-0000-0000-000000000001',
        '演示总部 - 租户管理员',
        TRUE
    ),
    (
        'f0000000-0000-0000-0000-000000000004',
        'b0000000-0000-0000-0000-000000000002',
        'business',
        'a0000000-0000-0000-0000-000000000001',
        '演示总部 - 业务用户',
        FALSE
    );

-- reviewer01: business for DEMO_HQ
INSERT INTO user_role_assignments (id, user_id, role, tenant_id, label, is_default)
VALUES
    (
        'f0000000-0000-0000-0000-000000000005',
        'b0000000-0000-0000-0000-000000000003',
        'business',
        'a0000000-0000-0000-0000-000000000001',
        '演示总部 - 业务用户',
        TRUE
    );

-- reviewer02: business for DEMO_HQ
INSERT INTO user_role_assignments (id, user_id, role, tenant_id, label, is_default)
VALUES
    (
        'f0000000-0000-0000-0000-000000000006',
        'b0000000-0000-0000-0000-000000000004',
        'business',
        'a0000000-0000-0000-0000-000000000001',
        '演示总部 - 业务用户',
        TRUE
    );

-- supervisor01: business for DEMO_HQ
INSERT INTO user_role_assignments (id, user_id, role, tenant_id, label, is_default)
VALUES
    (
        'f0000000-0000-0000-0000-000000000007',
        'b0000000-0000-0000-0000-000000000005',
        'business',
        'a0000000-0000-0000-0000-000000000001',
        '演示总部 - 业务用户',
        TRUE
    );
