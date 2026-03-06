-- 000006_tenant_admin_user.up.sql
-- Add admin_user_id to tenants table to track the tenant admin user

ALTER TABLE tenants ADD COLUMN admin_user_id UUID REFERENCES users(id);

-- Backfill: for each tenant, find the user who has tenant_admin role assignment
UPDATE tenants t
SET admin_user_id = (
    SELECT ura.user_id
    FROM user_role_assignments ura
    WHERE ura.tenant_id = t.id AND ura.role = 'tenant_admin'
    ORDER BY ura.created_at ASC
    LIMIT 1
);

CREATE INDEX idx_tenants_admin_user_id ON tenants (admin_user_id);
