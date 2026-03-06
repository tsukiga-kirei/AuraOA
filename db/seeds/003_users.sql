-- 003_users.sql
-- Seed data: users and user_role_assignments
-- Run after 002_tenants.sql

-- Fixed UUIDs for referential integrity:
-- Users:
--   b0000000-0000-0000-0000-000000000001  admin (system_admin, 跨租户)
--   b0000000-0000-0000-0000-000000000002  tenant_admin_user (tenant_admin for DEMO_HQ)
--   b0000000-0000-0000-0000-000000000003  reviewer01 (business user, 跨租户：DEMO_HQ + DEMO_BR1)
--   b0000000-0000-0000-0000-000000000004  reviewer02 (business user for DEMO_HQ)
--   b0000000-0000-0000-0000-000000000005  supervisor01 (business user for DEMO_HQ)
--   b0000000-0000-0000-0000-000000000006  br1_admin (tenant_admin for DEMO_BR1)
--   b0000000-0000-0000-0000-000000000007  br1_reviewer (business user for DEMO_BR1)
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
    ),
    -- DEMO_BR1 专属用户
    (
        'b0000000-0000-0000-0000-000000000006',
        'br1_admin',
        '$2a$12$RI08DxemoYuiefF0PjWXkeOV9MlLHSeLVxI32rjGjKkQETh6UuT/e',
        '分公司管理员',
        'br1_admin@example.com',
        '13900000006',
        'active'
    ),
    (
        'b0000000-0000-0000-0000-000000000007',
        'br1_reviewer',
        '$2a$12$RI08DxemoYuiefF0PjWXkeOV9MlLHSeLVxI32rjGjKkQETh6UuT/e',
        '分公司审核员赵六',
        'br1_reviewer@example.com',
        '13900000007',
        'active'
    );

-- ============================================================
-- user_role_assignments
-- ============================================================

-- ---- admin (system_admin + 两个租户的 tenant_admin + business) ----
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
    -- admin -> DEMO_HQ tenant_admin
    (
        'f0000000-0000-0000-0000-000000000002',
        'b0000000-0000-0000-0000-000000000001',
        'tenant_admin',
        'a0000000-0000-0000-0000-000000000001',
        '租户管理员 - 系统管理员',
        FALSE
    ),
    -- admin -> DEMO_HQ business
    (
        'f0000000-0000-0000-0000-000000000008',
        'b0000000-0000-0000-0000-000000000001',
        'business',
        'a0000000-0000-0000-0000-000000000001',
        '业务用户 - 系统管理员',
        FALSE
    ),
    -- admin -> DEMO_BR1 tenant_admin（跨租户）
    (
        'f0000000-0000-0000-0000-000000000009',
        'b0000000-0000-0000-0000-000000000001',
        'tenant_admin',
        'a0000000-0000-0000-0000-000000000002',
        '租户管理员 - 系统管理员（分公司）',
        FALSE
    ),
    -- admin -> DEMO_BR1 business（跨租户）
    (
        'f0000000-0000-0000-0000-000000000010',
        'b0000000-0000-0000-0000-000000000001',
        'business',
        'a0000000-0000-0000-0000-000000000002',
        '业务用户 - 系统管理员（分公司）',
        FALSE
    );

-- ---- tenant_admin_user -> DEMO_HQ tenant_admin + business ----
INSERT INTO user_role_assignments (id, user_id, role, tenant_id, label, is_default)
VALUES
    (
        'f0000000-0000-0000-0000-000000000003',
        'b0000000-0000-0000-0000-000000000002',
        'tenant_admin',
        'a0000000-0000-0000-0000-000000000001',
        '租户管理员 - 租户管理员',
        TRUE
    ),
    (
        'f0000000-0000-0000-0000-000000000004',
        'b0000000-0000-0000-0000-000000000002',
        'business',
        'a0000000-0000-0000-0000-000000000001',
        '业务用户 - 租户管理员',
        FALSE
    );

-- ---- reviewer01 -> DEMO_HQ business + DEMO_BR1 business（跨租户重复人员）----
INSERT INTO user_role_assignments (id, user_id, role, tenant_id, label, is_default)
VALUES
    (
        'f0000000-0000-0000-0000-000000000005',
        'b0000000-0000-0000-0000-000000000003',
        'business',
        'a0000000-0000-0000-0000-000000000001',
        '业务用户 - 审核员张三',
        TRUE
    ),
    -- reviewer01 -> DEMO_BR1 business（跨租户）
    (
        'f0000000-0000-0000-0000-000000000011',
        'b0000000-0000-0000-0000-000000000003',
        'business',
        'a0000000-0000-0000-0000-000000000002',
        '业务用户 - 审核员张三（分公司）',
        FALSE
    );

-- ---- reviewer02 -> DEMO_HQ business ----
INSERT INTO user_role_assignments (id, user_id, role, tenant_id, label, is_default)
VALUES
    (
        'f0000000-0000-0000-0000-000000000006',
        'b0000000-0000-0000-0000-000000000004',
        'business',
        'a0000000-0000-0000-0000-000000000001',
        '业务用户 - 审核员李四',
        TRUE
    );

-- ---- supervisor01 -> DEMO_HQ business ----
INSERT INTO user_role_assignments (id, user_id, role, tenant_id, label, is_default)
VALUES
    (
        'f0000000-0000-0000-0000-000000000007',
        'b0000000-0000-0000-0000-000000000005',
        'business',
        'a0000000-0000-0000-0000-000000000001',
        '业务用户 - 审核主管王五',
        TRUE
    );

-- ---- br1_admin -> DEMO_BR1 tenant_admin ----
INSERT INTO user_role_assignments (id, user_id, role, tenant_id, label, is_default)
VALUES
    (
        'f0000000-0000-0000-0000-000000000012',
        'b0000000-0000-0000-0000-000000000006',
        'tenant_admin',
        'a0000000-0000-0000-0000-000000000002',
        '租户管理员 - 分公司管理员',
        TRUE
    ),
    (
        'f0000000-0000-0000-0000-000000000013',
        'b0000000-0000-0000-0000-000000000006',
        'business',
        'a0000000-0000-0000-0000-000000000002',
        '业务用户 - 分公司管理员',
        FALSE
    );

-- ---- br1_reviewer -> DEMO_BR1 business ----
INSERT INTO user_role_assignments (id, user_id, role, tenant_id, label, is_default)
VALUES
    (
        'f0000000-0000-0000-0000-000000000014',
        'b0000000-0000-0000-0000-000000000007',
        'business',
        'a0000000-0000-0000-0000-000000000002',
        '业务用户 - 分公司审核员赵六',
        TRUE
    );
