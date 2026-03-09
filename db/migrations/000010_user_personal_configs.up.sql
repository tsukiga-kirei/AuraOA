-- 000010_user_personal_configs.up.sql
-- 创建用户个人配置表、用户仪表板偏好表

-- ============================================================
-- user_personal_configs — 用户个人配置表
-- ============================================================
CREATE TABLE user_personal_configs (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    audit_details   JSONB       NOT NULL DEFAULT '[]'::jsonb,
    cron_details    JSONB       NOT NULL DEFAULT '[]'::jsonb,
    archive_details JSONB       NOT NULL DEFAULT '[]'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, user_id)
);

CREATE INDEX idx_upc_tenant_id ON user_personal_configs(tenant_id);
CREATE INDEX idx_upc_user_id ON user_personal_configs(user_id);

-- ============================================================
-- user_dashboard_prefs — 用户仪表板偏好表
-- ============================================================
CREATE TABLE user_dashboard_prefs (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    enabled_widgets JSONB       NOT NULL DEFAULT '[]'::jsonb,
    widget_sizes    JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, user_id)
);

CREATE INDEX idx_udp_tenant_user ON user_dashboard_prefs(tenant_id, user_id);
