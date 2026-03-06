-- 002_tenants.sql
-- Seed data: demo tenants
-- Run after migrations are applied (including 000005)

-- Fixed UUIDs for referential integrity across seed files
-- Tenant: DEMO_HQ  -> a0000000-0000-0000-0000-000000000001
-- Tenant: DEMO_BR1 -> a0000000-0000-0000-0000-000000000002

INSERT INTO tenants (id, name, code, description, status,
    oa_db_connection_id, token_quota, token_used, max_concurrency,
    primary_model_id, fallback_model_id,
    max_tokens_per_request, temperature, timeout_seconds, retry_count,
    log_retention_days, data_retention_days,
    sso_enabled, sso_endpoint,
    contact_name, contact_email, contact_phone)
VALUES
    (
        'a0000000-0000-0000-0000-000000000001',
        '演示总部',
        'DEMO_HQ',
        '演示用总部租户，用于开发和测试',
        'active',
        'b0000000-0000-0000-0000-000000000001',  -- 总部泛微E9数据库
        50000, 0, 20,
        'c0000000-0000-0000-0000-000000000001',  -- Qwen2.5-72B（本地）
        'c0000000-0000-0000-0000-000000000003',  -- Qwen-Plus（阿里云百炼）
        8192, 0.3, 60, 3,
        365, 1095,
        FALSE, '',
        '张三', 'zhangsan@example.com', '13800000001'
    ),
    (
        'a0000000-0000-0000-0000-000000000002',
        '演示分公司',
        'DEMO_BR1',
        '演示用分公司租户',
        'active',
        'b0000000-0000-0000-0000-000000000002',  -- 华东分公司E9数据库
        10000, 0, 10,
        'c0000000-0000-0000-0000-000000000002',  -- Qwen2.5-32B（本地）
        NULL,                                     -- 无备用模型
        4096, 0.5, 45, 2,
        180, 730,
        FALSE, '',
        '李四', 'lisi@example.com', '13800000002'
    );
