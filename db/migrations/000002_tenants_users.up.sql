-- 000002_tenants_users.up.sql
-- 创建租户表、用户表、用户角色分配表、登录历史表

-- ============================================================
-- tenants — 租户表
-- ============================================================
CREATE TABLE tenants (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(), -- 主键UUID
    name                VARCHAR(255) NOT NULL,                              -- 租户名称
    code                VARCHAR(100) NOT NULL,                              -- 租户唯一编码，用于URL路由标识
    description         TEXT,                                              -- 租户描述
    status              VARCHAR(20)  NOT NULL DEFAULT 'active',            -- 状态：active/inactive/suspended
    oa_type             VARCHAR(50)  NOT NULL DEFAULT 'weaver_e9',         -- OA系统类型（v5版本迁移中移除）
    oa_db_connection_id UUID,                                              -- 关联的OA数据库连接ID（外键在v5迁移中添加）
    token_quota         INT          NOT NULL DEFAULT 10000,               -- Token总配额（租户级别）
    token_used          INT          NOT NULL DEFAULT 0,                   -- 已消耗Token数量
    max_concurrency     INT          NOT NULL DEFAULT 10,                  -- 最大并发AI审核请求数
    ai_config           JSONB        NOT NULL DEFAULT '{}',                -- AI配置（v5版本迁移中移除，改为独立字段）
    sso_enabled         BOOLEAN      NOT NULL DEFAULT FALSE,               -- 是否启用单点登录（SSO）
    sso_endpoint        VARCHAR(500),                                      -- SSO接口地址
    log_retention_days  INT          NOT NULL DEFAULT 365,                 -- 操作日志保留天数
    data_retention_days INT          NOT NULL DEFAULT 1095,                -- 审核数据保留天数（约3年）
    allow_custom_model  BOOLEAN      NOT NULL DEFAULT FALSE,               -- 是否允许自定义AI模型（v5版本迁移中移除）
    contact_name        VARCHAR(100),                                      -- 联系人姓名
    contact_email       VARCHAR(255),                                      -- 联系人邮箱
    contact_phone       VARCHAR(50),                                       -- 联系人电话
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),               -- 创建时间
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now()                -- 最后更新时间
);

CREATE UNIQUE INDEX idx_tenants_code ON tenants (code);

-- ============================================================
-- users — 平台用户账号表
-- ============================================================
CREATE TABLE users (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(), -- 主键UUID
    username            VARCHAR(100) NOT NULL,                              -- 登录用户名（全局唯一）
    password_hash       VARCHAR(255) NOT NULL,                              -- bcrypt加密后的密码哈希
    display_name        VARCHAR(100) NOT NULL,                              -- 用户显示名称
    email               VARCHAR(255),                                       -- 邮箱地址
    phone               VARCHAR(50),                                        -- 手机号码
    avatar_url          VARCHAR(500),                                       -- 头像图片URL
    status              VARCHAR(20)  NOT NULL DEFAULT 'active',             -- 账号状态：active/inactive/locked
    password_changed_at TIMESTAMPTZ  NOT NULL DEFAULT now(),                -- 最后修改密码时间
    login_fail_count    INT          NOT NULL DEFAULT 0,                    -- 连续登录失败次数
    locked_until        TIMESTAMPTZ,                                        -- 账号锁定截止时间（NULL表示未锁定）
    locale              VARCHAR(10)  NOT NULL DEFAULT 'zh-CN',              -- 用户界面语言偏好
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT now(),                -- 创建时间
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT now()                 -- 最后更新时间
);

CREATE UNIQUE INDEX idx_users_username ON users (username);

-- ============================================================
-- user_role_assignments — 用户系统级角色分配表
-- ============================================================
CREATE TABLE user_role_assignments (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),               -- 主键UUID
    user_id    UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,    -- 关联用户ID
    role       VARCHAR(30) NOT NULL,                                            -- 系统角色：business/tenant_admin/system_admin
    tenant_id  UUID        REFERENCES tenants (id) ON DELETE CASCADE,          -- 关联租户ID（system_admin角色时为NULL）
    label      VARCHAR(200),                                                    -- 角色显示标签（前端展示用）
    is_default BOOLEAN     NOT NULL DEFAULT FALSE,                              -- 是否为用户的默认角色/默认租户
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()                               -- 创建时间
);

CREATE INDEX idx_user_role_assignments_user_id   ON user_role_assignments (user_id);
CREATE INDEX idx_user_role_assignments_tenant_id ON user_role_assignments (tenant_id);

-- ============================================================
-- login_history — 用户登录历史表
-- ============================================================
CREATE TABLE login_history (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),              -- 主键UUID
    user_id    UUID         NOT NULL REFERENCES users (id) ON DELETE CASCADE,  -- 关联用户ID
    tenant_id  UUID         REFERENCES tenants (id) ON DELETE SET NULL,        -- 登录所属租户（NULL表示系统管理员登录）
    ip         VARCHAR(45),                                                     -- 客户端IP地址（支持IPv6）
    user_agent VARCHAR(500),                                                    -- 浏览器/客户端标识
    login_at   TIMESTAMPTZ  NOT NULL DEFAULT now()                              -- 登录时间
);

CREATE INDEX idx_login_history_user_id   ON login_history (user_id);
CREATE INDEX idx_login_history_tenant_id ON login_history (tenant_id);
