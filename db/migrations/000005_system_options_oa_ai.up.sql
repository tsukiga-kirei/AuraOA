-- 000005_system_options_oa_ai.up.sql
-- Create option tables, oa_database_connections, ai_model_configs

-- ============================================================
-- oa_type_options — OA系统类型选项表
-- ============================================================
CREATE TABLE oa_type_options (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    code       VARCHAR(50) NOT NULL UNIQUE,
    label      VARCHAR(100) NOT NULL,
    sort_order INT         NOT NULL DEFAULT 0,
    enabled    BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- db_driver_options — 数据库驱动选项表
-- ============================================================
CREATE TABLE db_driver_options (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    code         VARCHAR(50) NOT NULL UNIQUE,
    label        VARCHAR(100) NOT NULL,
    default_port INT         NOT NULL DEFAULT 3306,
    sort_order   INT         NOT NULL DEFAULT 0,
    enabled      BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- ai_deploy_type_options — AI部署类型选项表
-- ============================================================
CREATE TABLE ai_deploy_type_options (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    code       VARCHAR(50) NOT NULL UNIQUE,
    label      VARCHAR(100) NOT NULL,
    sort_order INT         NOT NULL DEFAULT 0,
    enabled    BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- ai_provider_options — AI服务商选项表
-- ============================================================
CREATE TABLE ai_provider_options (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(100) NOT NULL UNIQUE,
    label       VARCHAR(100) NOT NULL,
    deploy_type VARCHAR(50)  NOT NULL,
    sort_order  INT          NOT NULL DEFAULT 0,
    enabled     BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- ============================================================
-- oa_database_connections — OA数据库连接表
-- ============================================================
CREATE TABLE oa_database_connections (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name               VARCHAR(200) NOT NULL,
    oa_type            VARCHAR(50)  NOT NULL,
    oa_type_label      VARCHAR(100) DEFAULT '',
    driver             VARCHAR(50)  NOT NULL DEFAULT 'mysql',
    host               VARCHAR(255) NOT NULL DEFAULT '',
    port               INT          NOT NULL DEFAULT 3306,
    database_name      VARCHAR(200) NOT NULL DEFAULT '',
    username           VARCHAR(200) NOT NULL DEFAULT '',
    password           VARCHAR(500) NOT NULL DEFAULT '',
    pool_size          INT          NOT NULL DEFAULT 10,
    connection_timeout INT          NOT NULL DEFAULT 30,
    test_on_borrow     BOOLEAN      NOT NULL DEFAULT TRUE,
    status             VARCHAR(20)  NOT NULL DEFAULT 'disconnected',
    last_sync          TIMESTAMPTZ,
    sync_interval      INT          NOT NULL DEFAULT 30,
    enabled            BOOLEAN      NOT NULL DEFAULT TRUE,
    description        TEXT         DEFAULT '',
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- ============================================================
-- ai_model_configs — AI模型配置表
-- ============================================================
CREATE TABLE ai_model_configs (
    id                  UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    provider            VARCHAR(100)  NOT NULL,
    provider_label      VARCHAR(100)  DEFAULT '',
    model_name          VARCHAR(100)  NOT NULL,
    display_name        VARCHAR(200)  NOT NULL,
    deploy_type         VARCHAR(20)   NOT NULL DEFAULT 'local',
    endpoint            VARCHAR(500)  NOT NULL DEFAULT '',
    api_key             VARCHAR(500)  DEFAULT '',
    api_key_configured  BOOLEAN       NOT NULL DEFAULT FALSE,
    max_tokens          INT           NOT NULL DEFAULT 8192,
    context_window      INT           NOT NULL DEFAULT 131072,
    cost_per_1k_tokens  DECIMAL(10,6) DEFAULT 0,
    status              VARCHAR(20)   NOT NULL DEFAULT 'offline',
    enabled             BOOLEAN       NOT NULL DEFAULT TRUE,
    description         TEXT          DEFAULT '',
    capabilities        JSONB         NOT NULL DEFAULT '[]'::jsonb,
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ   NOT NULL DEFAULT now()
);

-- ============================================================
-- 为租户表增加AI模型直接引用列，替代 ai_config JSONB
-- ============================================================
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS primary_model_id      UUID,
    ADD COLUMN IF NOT EXISTS fallback_model_id     UUID,
    ADD COLUMN IF NOT EXISTS max_tokens_per_request INT NOT NULL DEFAULT 8192,
    ADD COLUMN IF NOT EXISTS temperature           DECIMAL(3,2) NOT NULL DEFAULT 0.30,
    ADD COLUMN IF NOT EXISTS timeout_seconds       INT NOT NULL DEFAULT 60,
    ADD COLUMN IF NOT EXISTS retry_count           INT NOT NULL DEFAULT 3;

-- 外键约束
ALTER TABLE tenants
    ADD CONSTRAINT fk_tenants_oa_db        FOREIGN KEY (oa_db_connection_id) REFERENCES oa_database_connections(id) ON DELETE SET NULL,
    ADD CONSTRAINT fk_tenants_primary_model  FOREIGN KEY (primary_model_id) REFERENCES ai_model_configs(id) ON DELETE SET NULL,
    ADD CONSTRAINT fk_tenants_fallback_model FOREIGN KEY (fallback_model_id) REFERENCES ai_model_configs(id) ON DELETE SET NULL;

-- 移除冗余列
ALTER TABLE tenants DROP COLUMN IF EXISTS oa_type;
ALTER TABLE tenants DROP COLUMN IF EXISTS allow_custom_model;
ALTER TABLE tenants DROP COLUMN IF EXISTS ai_config;
