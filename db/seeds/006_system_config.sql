-- 006_system_config.sql
-- Seed data: system key-value configurations
-- Run after migrations are applied

INSERT INTO system_configs (id, key, value, remark)
VALUES
    (
        '00000000-0000-0000-0000-000000000001',
        'system.name',
        'OA智审',
        '系统名称'
    ),
    (
        '00000000-0000-0000-0000-000000000002',
        'system.version',
        '1.0.0',
        '系统版本号'
    ),
    (
        '00000000-0000-0000-0000-000000000003',
        'auth.login_fail_lock_count',
        '5',
        '登录失败锁定阈值'
    ),
    (
        '00000000-0000-0000-0000-000000000004',
        'auth.lock_duration_minutes',
        '15',
        '账户锁定时长（分钟）'
    ),
    (
        '00000000-0000-0000-0000-000000000005',
        'auth.access_token_ttl_hours',
        '2',
        'Access Token 有效期（小时）'
    ),
    (
        '00000000-0000-0000-0000-000000000006',
        'auth.refresh_token_ttl_days',
        '7',
        'Refresh Token 有效期（天）'
    ),
    (
        '00000000-0000-0000-0000-000000000007',
        'tenant.default_token_quota',
        '10000',
        '租户默认 Token 配额'
    ),
    (
        '00000000-0000-0000-0000-000000000008',
        'tenant.default_max_concurrency',
        '10',
        '租户默认最大并发数'
    );
