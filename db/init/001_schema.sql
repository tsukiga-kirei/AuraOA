-- OA智审 数据库初始化脚本
-- PostgreSQL 16 + pgvector

-- 启用扩展
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS vector;

-- ============================================================
-- 租户表
-- ============================================================
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    token_quota INTEGER NOT NULL DEFAULT 10000,
    token_used INTEGER NOT NULL DEFAULT 0,
    max_concurrency INTEGER NOT NULL DEFAULT 10,
    oa_type VARCHAR(50) NOT NULL DEFAULT 'weaver_e9',
    oa_jdbc_config JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- 用户表
-- ============================================================
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    username VARCHAR(100) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL CHECK (role IN ('admin', 'tenant_admin', 'user')),
    oa_user_id VARCHAR(100),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, username)
);


-- ============================================================
-- 审核规则表
-- ============================================================
CREATE TABLE audit_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    process_type VARCHAR(100) NOT NULL,
    rule_content TEXT NOT NULL,
    rule_scope VARCHAR(20) NOT NULL CHECK (rule_scope IN ('mandatory', 'default_on', 'default_off')),
    is_locked BOOLEAN NOT NULL DEFAULT false,
    priority INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_rules_tenant_process ON audit_rules(tenant_id, process_type);

-- ============================================================
-- 用户私有规则表
-- ============================================================
CREATE TABLE user_private_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rule_content TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_private_rules_user ON user_private_rules(user_id);

-- ============================================================
-- 用户偏好表（规则开关）
-- ============================================================
CREATE TABLE user_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    rule_id UUID NOT NULL REFERENCES audit_rules(id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL,
    UNIQUE(user_id, rule_id)
);

-- ============================================================
-- 用户敏感度设置
-- ============================================================
CREATE TABLE user_sensitivity (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    level VARCHAR(20) NOT NULL DEFAULT 'normal' CHECK (level IN ('strict', 'normal', 'relaxed'))
);

-- ============================================================
-- 知识库模式配置
-- ============================================================
CREATE TABLE kb_mode_config (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    process_type VARCHAR(100) NOT NULL,
    kb_mode VARCHAR(20) NOT NULL CHECK (kb_mode IN ('rules_only', 'rag_only', 'hybrid')),
    UNIQUE(tenant_id, process_type)
);

-- ============================================================
-- Cron 任务表
-- ============================================================
CREATE TABLE cron_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    cron_expression VARCHAR(100) NOT NULL,
    task_type VARCHAR(50) NOT NULL CHECK (task_type IN ('batch_audit', 'daily_report', 'weekly_report')),
    is_active BOOLEAN NOT NULL DEFAULT true,
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_cron_tasks_user ON cron_tasks(user_id);

-- ============================================================
-- 日志留存策略
-- ============================================================
CREATE TABLE log_retention_policies (
    tenant_id UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    retention_days INTEGER, -- NULL = 永久保存
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- AI 配置表
-- ============================================================
CREATE TABLE ai_configs (
    tenant_id UUID PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    model_provider VARCHAR(50) NOT NULL DEFAULT 'local',
    model_name VARCHAR(100) NOT NULL DEFAULT 'default',
    prompt_template TEXT,
    context_window_size INTEGER NOT NULL DEFAULT 4096,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ============================================================
-- 脱敏规则表
-- ============================================================
CREATE TABLE masking_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    field_pattern VARCHAR(255) NOT NULL,
    value_pattern VARCHAR(255) NOT NULL,
    replace_with VARCHAR(255) NOT NULL DEFAULT '***',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_masking_rules_tenant ON masking_rules(tenant_id);

-- ============================================================
-- pgvector 向量表（第二阶段 RAG 启用时使用）
-- 当前仅创建表结构，不写入数据
-- ============================================================
-- CREATE TABLE document_chunks (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
--     document_name VARCHAR(255) NOT NULL,
--     chunk_index INTEGER NOT NULL,
--     chunk_text TEXT NOT NULL,
--     embedding vector(1536),
--     created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
-- );
-- CREATE INDEX idx_document_chunks_embedding ON document_chunks
--     USING ivfflat (embedding vector_cosine_ops) WITH (lists = 100);
