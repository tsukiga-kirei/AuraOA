-- 000032_auth_default_password_config.down.sql

DELETE FROM system_configs WHERE key = 'auth.default_password';
