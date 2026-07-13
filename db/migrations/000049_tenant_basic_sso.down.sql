ALTER TABLE tenants
    DROP COLUMN IF EXISTS sso_basic_allowed_domains,
    DROP COLUMN IF EXISTS sso_basic_allowed_ips,
    DROP COLUMN IF EXISTS sso_basic_password,
    DROP COLUMN IF EXISTS sso_basic_enabled;
