-- 000032_auth_default_password_config.up.sql
-- 新增默认密码配置项，支持管理员自定义新成员默认密码

INSERT INTO system_configs (key, value, remark) VALUES
    ('auth.default_password', 'Audit@2026', '新建成员默认密码（建议定期更换或使用随机密码策略）');
