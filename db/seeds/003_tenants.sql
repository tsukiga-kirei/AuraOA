-- 003_tenants.sql
-- Seed data: demo tenants
-- Run after 002_users.sql (admin_user_id FK requires users to exist first)

INSERT INTO tenants (id, name, code, description, status,
    oa_db_connection_id, token_quota, token_used, max_concurrency,
    primary_model_id, fallback_model_id,
    max_tokens_per_request, temperature, timeout_seconds, retry_count,
    log_retention_days, data_retention_days,
    sso_enabled, sso_endpoint,
    contact_name, contact_email, contact_phone, admin_user_id)
VALUES
    (
        'a0000000-0000-0000-0000-000000000001',
        '演示总部',
        'DEMO_HQ',
        '演示用总部租户，用于开发和测试',
        'active',
        'b0000000-0000-0000-0000-000000000001',
        50000, 0, 20,
        'c0000000-0000-0000-0000-000000000001',
        'c0000000-0000-0000-0000-000000000003',
        8192, 0.3, 60, 3,
        365, 1095,
        FALSE, '',
        '张三', 'zhangsan@example.com', '13800000001',
        'b0000000-0000-0000-0000-000000000005'
    ),
    (
        'a0000000-0000-0000-0000-000000000002',
        '演示分公司',
        'DEMO_BR1',
        '演示用分公司租户，华东区域分公司',
        'active',
        'b0000000-0000-0000-0000-000000000002',
        10000, 0, 10,
        'c0000000-0000-0000-0000-000000000002',
        'c0000000-0000-0000-0000-000000000006',
        4096, 0.5, 45, 2,
        180, 730,
        FALSE, '',
        '分公司管理员', 'br1_admin@example.com', '13900000006',
        'b0000000-0000-0000-0000-000000000006'
    );
