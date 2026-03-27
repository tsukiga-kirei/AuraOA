-- 回滚会丢弃 tenant_admin 维度的布局，仅保留 business；请先确认可接受数据损失。
DROP INDEX IF EXISTS user_dashboard_prefs_tenant_user_scope_unique;

DELETE FROM user_dashboard_prefs WHERE pref_scope = 'tenant_admin';

UPDATE user_dashboard_prefs SET pref_scope = 'shared' WHERE tenant_id IS NOT NULL;

ALTER TABLE user_dashboard_prefs
    ALTER COLUMN pref_scope SET DEFAULT 'shared';

-- 若 platform 行存在 pref_scope=platform，保持不变；租户行合并后应仅一行 per (tenant_id,user_id)
CREATE UNIQUE INDEX user_dashboard_prefs_tenant_user_unique
    ON user_dashboard_prefs (tenant_id, user_id)
    WHERE tenant_id IS NOT NULL;

ALTER TABLE user_dashboard_prefs DROP COLUMN IF EXISTS pref_scope;
