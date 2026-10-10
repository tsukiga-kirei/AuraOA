-- 新增记忆库的数据记忆能力：独立于关联流程和建模挂载，支持租户复用、版本及证据快照。
CREATE TABLE audit_data_queries (
    id uuid PRIMARY KEY,
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name varchar(200) NOT NULL,
    category varchar(100) NOT NULL DEFAULT '',
    draft jsonb NOT NULL DEFAULT '{}',
    tested_hash varchar(100) NOT NULL DEFAULT '',
    tested_at timestamptz,
    latest_version integer NOT NULL DEFAULT 0,
    archived boolean NOT NULL DEFAULT false,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- 支持查询库按租户分页及最近修改排序。
CREATE INDEX idx_audit_data_queries_tenant ON audit_data_queries(tenant_id, updated_at DESC);

-- 发布快照永久保留，组合唯一约束防止并发产生重复版本号。
CREATE TABLE audit_data_query_versions (
    id uuid PRIMARY KEY,
    tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    query_id uuid NOT NULL REFERENCES audit_data_queries(id) ON DELETE CASCADE,
    version_no integer NOT NULL,
    definition jsonb NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, query_id, version_no)
);

-- 未配置时服务端使用默认提示词；配置后按租户覆盖。
CREATE TABLE audit_data_query_prompts (
    tenant_id uuid PRIMARY KEY REFERENCES tenants(id) ON DELETE CASCADE,
    generate text NOT NULL,
    explain text NOT NULL,
    repair text NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- 审核与归档对称扩展，旧规则及日志以空数组兼容。
ALTER TABLE audit_rules ADD COLUMN data_queries jsonb NOT NULL DEFAULT '[]';
ALTER TABLE archive_rules ADD COLUMN data_queries jsonb NOT NULL DEFAULT '[]';
ALTER TABLE audit_logs ADD COLUMN query_evidence jsonb NOT NULL DEFAULT '[]';
ALTER TABLE archive_logs ADD COLUMN query_evidence jsonb NOT NULL DEFAULT '[]';
-- 默认系统租户管理员角色获得入口；其他角色仍需显式授权。
UPDATE org_roles
SET page_permissions = page_permissions || '["/admin/tenant/memory"]'::jsonb
WHERE is_system = true
    AND page_permissions @> '["/admin/tenant/rules"]'::jsonb
    AND NOT page_permissions @> '["/admin/tenant/memory"]'::jsonb;

-- 查询库只保存租户内定义；发布版本不可修改，归档仅隐藏入口，保留历史引用。
COMMENT ON TABLE audit_data_queries IS '租户级审核数据查询库，保存可编辑草稿及最新发布版本号';
COMMENT ON COLUMN audit_data_queries.id IS '查询定义唯一标识';
COMMENT ON COLUMN audit_data_queries.tenant_id IS '所属租户，所有访问必须按租户隔离';
COMMENT ON COLUMN audit_data_queries.name IS '查询中文名称';
COMMENT ON COLUMN audit_data_queries.category IS '查询分类，用于搜索和复用';
COMMENT ON COLUMN audit_data_queries.draft IS '查询草稿：可视化配置、SQL、方言、表范围、参数、中文用途说明和行数上限';
COMMENT ON COLUMN audit_data_queries.latest_version IS '最新发布版本号，零表示尚未发布';
COMMENT ON COLUMN audit_data_queries.tested_hash IS '成功预览的草稿及数据源指纹，修改草稿后清空';
COMMENT ON COLUMN audit_data_queries.tested_at IS '最近一次成功预览时间';
COMMENT ON COLUMN audit_data_queries.archived IS '是否归档，归档后从查询库选择列表隐藏，历史版本继续可执行';
COMMENT ON COLUMN audit_data_queries.created_at IS '创建时间';
COMMENT ON COLUMN audit_data_queries.updated_at IS '最后修改时间';
COMMENT ON INDEX idx_audit_data_queries_tenant IS '按租户及修改时间查询分页列表';
COMMENT ON TABLE audit_data_query_versions IS '审核数据查询不可变发布版本';
COMMENT ON COLUMN audit_data_query_versions.id IS '版本唯一标识';
COMMENT ON COLUMN audit_data_query_versions.tenant_id IS '所属租户';
COMMENT ON COLUMN audit_data_query_versions.query_id IS '所属查询定义';
COMMENT ON COLUMN audit_data_query_versions.version_no IS '查询内递增版本号，通过行锁分配';
COMMENT ON COLUMN audit_data_query_versions.definition IS '发布时的完整查询定义快照';
COMMENT ON COLUMN audit_data_query_versions.created_at IS '发布时间';
COMMENT ON TABLE audit_data_query_prompts IS '租户可编辑的 SQL 生成、解释和修复提示词';
COMMENT ON COLUMN audit_data_query_prompts.tenant_id IS '所属租户，每租户一组提示词';
COMMENT ON COLUMN audit_data_query_prompts.generate IS 'AI 生成 SQL 的业务提示词，不覆盖系统安全约束';
COMMENT ON COLUMN audit_data_query_prompts.explain IS 'AI 解释 SQL 的业务提示词';
COMMENT ON COLUMN audit_data_query_prompts.repair IS 'AI 修复 SQL 的业务提示词';
COMMENT ON COLUMN audit_data_query_prompts.updated_at IS '提示词最后修改时间';
COMMENT ON COLUMN audit_rules.data_queries IS '审核规则引用的查询版本与流程字段参数绑定';
COMMENT ON COLUMN archive_rules.data_queries IS '归档复盘规则引用的查询版本与流程字段参数绑定';
COMMENT ON COLUMN audit_logs.query_evidence IS '当次审核取数证据：规则、版本、参数、结果及完整性状态';
COMMENT ON COLUMN archive_logs.query_evidence IS '当次归档复盘取数证据：规则、版本、参数、结果及完整性状态';
