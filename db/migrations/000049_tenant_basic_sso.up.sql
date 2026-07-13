-- 为租户增加 Basic 单点登录配置。
-- 共享密码只保存 AES-GCM 密文；外部系统登录后仍由 AuraOA 本地角色完成授权。
ALTER TABLE tenants
    ADD COLUMN sso_basic_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN sso_basic_password TEXT NOT NULL DEFAULT '',
    ADD COLUMN sso_basic_allowed_ips TEXT NOT NULL DEFAULT '',
    ADD COLUMN sso_basic_allowed_domains TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN tenants.sso_basic_enabled IS '是否启用租户级 Basic 单点登录';
COMMENT ON COLUMN tenants.sso_basic_password IS 'Basic 单点登录共享密码的 AES-GCM 密文';
COMMENT ON COLUMN tenants.sso_basic_allowed_ips IS '允许换取单点登录地址的来源 IP 或 CIDR，逗号分隔';
COMMENT ON COLUMN tenants.sso_basic_allowed_domains IS '允许换取单点登录地址的 Origin/Referer 域名，逗号分隔';
