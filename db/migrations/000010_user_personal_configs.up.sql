-- 000010_user_personal_configs.up.sql
-- 创建用户个人审核配置表、用户仪表板偏好表

-- ============================================================
-- user_personal_configs — 用户个人审核配置表
-- ============================================================
CREATE TABLE user_personal_configs (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),              -- 主键UUID
    tenant_id       UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE, -- 所属租户ID
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,   -- 关联用户ID
    audit_details   JSONB       NOT NULL DEFAULT '[]'::jsonb,                      -- 用户在各流程的审核字段/规则个人偏好配置
    cron_details    JSONB       NOT NULL DEFAULT '[]'::jsonb,                      -- 用户定时任务相关个人偏好配置
    archive_details JSONB       NOT NULL DEFAULT '[]'::jsonb,                      -- 用户归档复盘相关个人偏好配置
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),                             -- 创建时间
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),                             -- 最后更新时间
    UNIQUE(tenant_id, user_id)
);

CREATE INDEX idx_upc_tenant_id ON user_personal_configs(tenant_id);
CREATE INDEX idx_upc_user_id   ON user_personal_configs(user_id);

-- ============================================================
-- user_dashboard_prefs — 用户仪表板组件偏好表
-- ============================================================
CREATE TABLE user_dashboard_prefs (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),              -- 主键UUID
    tenant_id       UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE, -- 所属租户ID
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,   -- 关联用户ID
    enabled_widgets JSONB       NOT NULL DEFAULT '[]'::jsonb,                      -- 已启用的仪表板组件ID列表
    widget_sizes    JSONB       NOT NULL DEFAULT '{}'::jsonb,                      -- 各组件尺寸配置（key=组件ID，value=尺寸规格）
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),                             -- 创建时间
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),                             -- 最后更新时间
    UNIQUE(tenant_id, user_id)
);

CREATE INDEX idx_udp_tenant_user ON user_dashboard_prefs(tenant_id, user_id);
