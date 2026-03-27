DROP INDEX IF EXISTS user_dashboard_prefs_platform_user_unique;
DROP INDEX IF EXISTS user_dashboard_prefs_tenant_user_unique;

DELETE FROM user_dashboard_prefs WHERE tenant_id IS NULL;

ALTER TABLE user_dashboard_prefs
    ALTER COLUMN tenant_id SET NOT NULL;

ALTER TABLE user_dashboard_prefs
    ADD CONSTRAINT user_dashboard_prefs_tenant_id_user_id_key UNIQUE (tenant_id, user_id);
