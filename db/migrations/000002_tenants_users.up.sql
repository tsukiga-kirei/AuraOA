-- 000002_tenants_users.up.sql
-- Create tenants, users, user_role_assignments, and login_history tables

-- ============================================================
-- tenants
-- ============================================================
CREATE TABLE tenants (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(255) NOT NULL,
    code                VARCHAR(100) NOT NULL,
    description         TEXT,
    status              VARCHAR(20)  NOT NULL DEFAULT 'active',
    oa_type             VARCHAR(50)  NOT NULL DEFAULT 'weaver_e9',
    oa_db_connection_id UUID,
    token_quota         INT          NOT NULL DEFAULT 10000,
    token_used          INT          NOT NULL DEFAULT 0,
    max_concurrency     INT          NOT NULL DEFAULT 10,
    ai_config           JSONB        NOT NULL DEFAULT '{}',
    sso_enabled         BOOLEAN      NOT NULL DEFAULT FALSE,
    sso_endpoint        VARCHAR(500),
    log_retention_days  INT          NOT NULL DEFAULT 365,
    data_retention_days INT          NOT NULL DEFAULT 1095,
    allow_custom_model  BOOLEAN      NOT NULL DEFAULT FALSE,
    contact_name        VARCHAR(100),
    contact_email       VARCHAR(255),
    contact_phone       VARCHAR(50),
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_tenants_code ON tenants (code);

-- ============================================================
-- users
-- ============================================================
CREATE TABLE users (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    username            VARCHAR(100) NOT NULL,
    password_hash       VARCHAR(255) NOT NULL,
    display_name        VARCHAR(100) NOT NULL,
    email               VARCHAR(255),
    phone               VARCHAR(50),
    avatar_url          VARCHAR(500),
    status              VARCHAR(20)  NOT NULL DEFAULT 'active',
    password_changed_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    login_fail_count    INT          NOT NULL DEFAULT 0,
    locked_until        TIMESTAMPTZ,
    locale              VARCHAR(10)  NOT NULL DEFAULT 'zh-CN',
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_users_username ON users (username);

-- ============================================================
-- user_role_assignments
-- ============================================================
CREATE TABLE user_role_assignments (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role       VARCHAR(30) NOT NULL,
    tenant_id  UUID        REFERENCES tenants (id) ON DELETE CASCADE,
    label      VARCHAR(200),
    is_default BOOLEAN     NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_role_assignments_user_id   ON user_role_assignments (user_id);
CREATE INDEX idx_user_role_assignments_tenant_id ON user_role_assignments (tenant_id);

-- ============================================================
-- login_history
-- ============================================================
CREATE TABLE login_history (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID         NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    tenant_id  UUID         REFERENCES tenants (id) ON DELETE SET NULL,
    ip         VARCHAR(45),
    user_agent VARCHAR(500),
    login_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_login_history_user_id   ON login_history (user_id);
CREATE INDEX idx_login_history_tenant_id ON login_history (tenant_id);
