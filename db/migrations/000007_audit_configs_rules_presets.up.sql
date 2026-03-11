-- 000007_audit_configs_rules_presets.up.sql
-- 创建流程审核配置表、审核规则表、审核尺度预设表

-- ============================================================
-- process_audit_configs — 流程审核配置表
-- ============================================================
CREATE TABLE process_audit_configs (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    process_type      VARCHAR(200) NOT NULL,
    process_type_label VARCHAR(200) DEFAULT '',
    main_table_name   VARCHAR(200) DEFAULT '',
    main_fields       JSONB        NOT NULL DEFAULT '[]'::jsonb,
    detail_tables     JSONB        NOT NULL DEFAULT '[]'::jsonb,
    field_mode        VARCHAR(20)  NOT NULL DEFAULT 'all',
    kb_mode           VARCHAR(20)  NOT NULL DEFAULT 'rules_only',
    ai_config         JSONB        NOT NULL DEFAULT '{}'::jsonb,
    user_permissions  JSONB        NOT NULL DEFAULT '{}'::jsonb,
    status            VARCHAR(20)  NOT NULL DEFAULT 'active',
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, process_type)
);

CREATE INDEX idx_pac_tenant_id ON process_audit_configs(tenant_id);

-- ============================================================
-- audit_rules — 审核规则表
-- ============================================================
CREATE TABLE audit_rules (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    config_id    UUID         REFERENCES process_audit_configs(id) ON DELETE CASCADE,
    process_type VARCHAR(200) NOT NULL,
    rule_content TEXT         NOT NULL,
    rule_scope   VARCHAR(20)  NOT NULL DEFAULT 'default_on',
    enabled      BOOLEAN      NOT NULL DEFAULT TRUE,
    source       VARCHAR(20)  NOT NULL DEFAULT 'manual',
    related_flow BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_ar_tenant_id ON audit_rules(tenant_id);
CREATE INDEX idx_ar_config_id ON audit_rules(config_id);
CREATE INDEX idx_ar_process_type ON audit_rules(tenant_id, process_type);

-- ============================================================
-- strictness_presets — 审核尺度预设表
-- ============================================================
CREATE TABLE strictness_presets (
    id                     UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    strictness             VARCHAR(20)  NOT NULL,
    reasoning_instruction  TEXT         NOT NULL DEFAULT '',
    extraction_instruction TEXT         NOT NULL DEFAULT '',
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE(tenant_id, strictness)
);

CREATE INDEX idx_sp_tenant_id ON strictness_presets(tenant_id);
