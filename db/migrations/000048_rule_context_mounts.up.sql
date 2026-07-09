-- 000048_rule_context_mounts.up.sql
-- 为审核规则和归档规则增加轻量级外部关联数据配置。

ALTER TABLE audit_rules
    ADD COLUMN IF NOT EXISTS context_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS context_mounts JSONB NOT NULL DEFAULT '[]'::jsonb;

ALTER TABLE archive_rules
    ADD COLUMN IF NOT EXISTS context_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS context_mounts JSONB NOT NULL DEFAULT '[]'::jsonb;

COMMENT ON COLUMN audit_rules.context_enabled IS '审核规则是否启用外部关联数据（流程/建模表）';
COMMENT ON COLUMN audit_rules.context_mounts IS '审核规则外部关联数据查询配置（流程/建模表）';
COMMENT ON COLUMN archive_rules.context_enabled IS '归档规则是否启用外部关联数据（流程/建模表）';
COMMENT ON COLUMN archive_rules.context_mounts IS '归档规则外部关联数据查询配置（流程/建模表）';
