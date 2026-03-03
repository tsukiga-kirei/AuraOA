-- 001_tenants.sql
-- Seed data: demo tenants
-- Run after migrations are applied

-- Fixed UUIDs for referential integrity across seed files
-- Tenant: DEMO_HQ  -> a0000000-0000-0000-0000-000000000001
-- Tenant: DEMO_BR1 -> a0000000-0000-0000-0000-000000000002

INSERT INTO tenants (id, name, code, description, status, oa_type, token_quota, token_used, max_concurrency, ai_config, contact_name, contact_email, contact_phone)
VALUES
    (
        'a0000000-0000-0000-0000-000000000001',
        '演示总部',
        'DEMO_HQ',
        '演示用总部租户，用于开发和测试',
        'active',
        'weaver_e9',
        50000,
        0,
        20,
        '{"model": "gpt-4o", "temperature": 0.3, "max_tokens": 4096, "prompt_template": "default"}',
        '张三',
        'zhangsan@example.com',
        '13800000001'
    ),
    (
        'a0000000-0000-0000-0000-000000000002',
        '演示分公司',
        'DEMO_BR1',
        '演示用分公司租户',
        'active',
        'weaver_e9',
        10000,
        0,
        10,
        '{"model": "gpt-4o-mini", "temperature": 0.5, "max_tokens": 2048, "prompt_template": "default"}',
        '李四',
        'lisi@example.com',
        '13800000002'
    );
