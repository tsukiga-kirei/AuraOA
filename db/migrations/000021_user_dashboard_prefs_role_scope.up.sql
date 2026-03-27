-- 仪表盘布局按「当前激活角色」分条存储，避免同一用户以 business / tenant_admin 保存时互相覆盖。

DROP INDEX IF EXISTS user_dashboard_prefs_tenant_user_unique;

ALTER TABLE user_dashboard_prefs
    ADD COLUMN IF NOT EXISTS pref_scope VARCHAR(32) NOT NULL DEFAULT 'shared';

UPDATE user_dashboard_prefs
SET pref_scope = 'platform'
WHERE tenant_id IS NULL;

-- 原「每租户用户一行」的数据复制为 business 与 tenant_admin 各一条（初始布局相同）
INSERT INTO user_dashboard_prefs (id, tenant_id, user_id, pref_scope, enabled_widgets, widget_sizes, created_at, updated_at)
SELECT gen_random_uuid(),
       tenant_id,
       user_id,
       'business',
       enabled_widgets,
       widget_sizes,
       created_at,
       updated_at
FROM user_dashboard_prefs
WHERE tenant_id IS NOT NULL
  AND pref_scope = 'shared';

INSERT INTO user_dashboard_prefs (id, tenant_id, user_id, pref_scope, enabled_widgets, widget_sizes, created_at, updated_at)
SELECT gen_random_uuid(),
       tenant_id,
       user_id,
       'tenant_admin',
       enabled_widgets,
       widget_sizes,
       created_at,
       updated_at
FROM user_dashboard_prefs
WHERE tenant_id IS NOT NULL
  AND pref_scope = 'shared';

DELETE FROM user_dashboard_prefs
WHERE tenant_id IS NOT NULL
  AND pref_scope = 'shared';

ALTER TABLE user_dashboard_prefs
    ALTER COLUMN pref_scope DROP DEFAULT;

CREATE UNIQUE INDEX user_dashboard_prefs_tenant_user_scope_unique
    ON user_dashboard_prefs (tenant_id, user_id, pref_scope)
    WHERE tenant_id IS NOT NULL;
