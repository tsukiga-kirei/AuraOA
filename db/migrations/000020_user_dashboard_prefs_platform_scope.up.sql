-- 允许系统管理员在无租户上下文时保存仪表盘布局：tenant_id 可为 NULL，与 user_id 唯一对应。

ALTER TABLE user_dashboard_prefs
    DROP CONSTRAINT IF EXISTS user_dashboard_prefs_tenant_id_user_id_key;

ALTER TABLE user_dashboard_prefs
    ALTER COLUMN tenant_id DROP NOT NULL;

-- 租户内用户：仍按 (tenant_id, user_id) 唯一
CREATE UNIQUE INDEX user_dashboard_prefs_tenant_user_unique
    ON user_dashboard_prefs (tenant_id, user_id)
    WHERE tenant_id IS NOT NULL;

-- 平台（系统管理员）仪表盘：每用户一行，tenant_id IS NULL
CREATE UNIQUE INDEX user_dashboard_prefs_platform_user_unique
    ON user_dashboard_prefs (user_id)
    WHERE tenant_id IS NULL;
