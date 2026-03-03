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
    ),
    (
        '00000000-0000-0000-0000-000000000009',
        'system.default_language',
        'zh-CN',
        '系统默认语言'
    ),
    (
        '00000000-0000-0000-0000-000000000010',
        'system.max_upload_size_mb',
        '50',
        '最大上传文件大小（MB）'
    ),
    (
        '00000000-0000-0000-0000-000000000011',
        'system.enable_audit_trail',
        'true',
        '是否启用审计日志'
    ),
    (
        '00000000-0000-0000-0000-000000000012',
        'system.enable_data_encryption',
        'false',
        '是否启用数据加密'
    ),
    (
        '00000000-0000-0000-0000-000000000013',
        'system.backup_enabled',
        'false',
        '是否启用自动备份'
    ),
    (
        '00000000-0000-0000-0000-000000000014',
        'system.backup_cron',
        '0 2 * * *',
        '备份 Cron 表达式（默认每天凌晨 2 点）'
    ),
    (
        '00000000-0000-0000-0000-000000000015',
        'system.backup_retention_days',
        '30',
        '备份保留天数'
    ),
    (
        '00000000-0000-0000-0000-000000000016',
        'system.notification_email',
        '',
        '系统通知邮箱'
    ),
    (
        '00000000-0000-0000-0000-000000000017',
        'system.smtp_host',
        '',
        'SMTP 服务器地址'
    ),
    (
        '00000000-0000-0000-0000-000000000018',
        'system.smtp_port',
        '465',
        'SMTP 端口'
    ),
    (
        '00000000-0000-0000-0000-000000000019',
        'system.smtp_username',
        '',
        'SMTP 用户名'
    ),
    (
        '00000000-0000-0000-0000-000000000020',
        'system.smtp_ssl',
        'true',
        '是否启用 SMTP SSL/TLS'
    );
