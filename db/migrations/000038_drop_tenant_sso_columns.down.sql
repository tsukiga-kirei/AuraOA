-- 恢复租户 SSO 列（与 000002 初始定义一致）
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS sso_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE tenants ADD COLUMN IF NOT EXISTS sso_endpoint VARCHAR(500);

COMMENT ON COLUMN tenants.sso_enabled IS '是否启用单点登录（SSO）';
COMMENT ON COLUMN tenants.sso_endpoint IS 'SSO接口地址';
