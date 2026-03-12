-- 000006_tenant_admin_user.up.sql
-- 为 tenants 表新增 admin_user_id 字段，记录租户管理员用户

ALTER TABLE tenants ADD COLUMN admin_user_id UUID REFERENCES users(id);

-- 数据回填：为每个租户找到最早的 tenant_admin 角色分配用户
UPDATE tenants t
SET admin_user_id = (
    SELECT ura.user_id
    FROM user_role_assignments ura
    WHERE ura.tenant_id = t.id AND ura.role = 'tenant_admin'
    ORDER BY ura.created_at ASC
    LIMIT 1
);

CREATE INDEX idx_tenants_admin_user_id ON tenants (admin_user_id);

-- ============================================================
-- 数据库注释（中文）
-- ============================================================
COMMENT ON COLUMN tenants.admin_user_id IS '租户管理员用户ID';
