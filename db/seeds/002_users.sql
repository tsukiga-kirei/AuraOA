-- 002_users.sql
-- Seed data: users only (no role assignments)
-- Run after 001_oa_ai_seeds.sql
-- Role assignments moved to 004 to avoid circular FK with tenants

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
