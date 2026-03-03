-- 000004_system_configs.up.sql
-- Create system_configs table for global key-value configuration

CREATE TABLE system_configs (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    key        VARCHAR(200) NOT NULL,
    value      TEXT         NOT NULL DEFAULT '',
    remark     VARCHAR(500),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_system_configs_key ON system_configs (key);
