-- 移除租户 SSO 配置列（产品已下线该能力）
ALTER TABLE tenants DROP COLUMN IF EXISTS sso_enabled;
ALTER TABLE tenants DROP COLUMN IF EXISTS sso_endpoint;
