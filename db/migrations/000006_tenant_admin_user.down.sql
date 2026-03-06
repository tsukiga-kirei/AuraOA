-- 000006_tenant_admin_user.down.sql
DROP INDEX IF EXISTS idx_tenants_admin_user_id;
ALTER TABLE tenants DROP COLUMN IF EXISTS admin_user_id;
