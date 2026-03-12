-- 000004_system_configs.up.sql
-- 创建系统全局KV配置表，并初始化默认配置项

-- ============================================================
-- system_configs — 系统全局键值配置表
-- ============================================================
CREATE TABLE system_configs (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(), -- 主键UUID
    key        VARCHAR(200) NOT NULL,                              -- 配置键名（全局唯一，格式：模块.配置项）
    value      TEXT         NOT NULL DEFAULT '',                   -- 配置值（统一存为字符串，业务层负责类型转换）
    remark     VARCHAR(500),                                       -- 配置说明/备注
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),                -- 创建时间
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()                 -- 最后更新时间
);

CREATE UNIQUE INDEX idx_system_configs_key ON system_configs (key);

-- ============================================================
-- 初始化系统默认配置
-- UUID 约定：00000000-0000-0000-0000-0000000000xx
-- ============================================================
INSERT INTO system_configs (key, value, remark) VALUES
    ('system.name',                        'OA智审',    '系统名称'),
    ('system.version',                     '1.0.0',     '系统版本号'),
    ('auth.login_fail_lock_threshold',     '5',         '连续登录失败达到此次数后锁定账户'),
    ('auth.account_lock_minutes',          '15',        '账户锁定时长（分钟）'),
    ('auth.access_token_ttl_hours',        '2',         'Access Token 有效期（小时）'),
    ('auth.refresh_token_ttl_days',        '7',         'Refresh Token 有效期（天）'),
    ('tenant.default_token_quota',         '10000',     '新建租户默认 Token 配额'),
    ('tenant.default_max_concurrency',     '10',        '新建租户默认最大 AI 并发审核数'),
    ('system.default_language',            'zh-CN',     '系统默认界面语言'),
    ('system.max_upload_size_mb',          '50',        '上传文件最大体积限制（MB）'),
    ('system.enable_audit_trail',          'true',      '是否启用操作审计日志'),
    ('system.enable_data_encryption',      'false',     '是否启用静态数据加密'),
    ('system.backup_enabled',              'false',     '是否启用自动数据库备份'),
    ('system.backup_cron',                 '0 2 * * *', '自动备份 Cron 表达式（默认每天凌晨2点）'),
    ('system.backup_retention_days',       '30',        '备份文件保留天数'),
    ('system.notification_email',          '',          '系统通知发件邮箱地址'),
    ('system.smtp_host',                   '',          'SMTP 服务器地址'),
    ('system.smtp_port',                   '465',       'SMTP 服务器端口'),
    ('system.smtp_username',               '',          'SMTP 认证用户名'),
    ('system.smtp_ssl',                    'true',      '是否启用 SMTP SSL/TLS 加密'),
    ('tenant.default_log_retention_days',  '365',       '新建租户默认操作日志保留天数'),
    ('tenant.default_data_retention_days', '1095',      '新建租户默认审核数据保留天数（约3年）');
