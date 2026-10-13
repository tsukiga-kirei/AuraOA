-- 反馈素材与生成版本独立保存；发布指针和素材修订号防止并发覆盖。
CREATE TABLE feedback_memories (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
 module varchar(20) NOT NULL CHECK (module IN ('audit','archive')), process_type varchar(200) NOT NULL,
 revision integer NOT NULL DEFAULT 0, last_version integer NOT NULL DEFAULT 0,
 published_version_id uuid, created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
 UNIQUE (tenant_id, module, process_type)
);
CREATE TABLE feedback_memory_entries (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
 memory_id uuid NOT NULL REFERENCES feedback_memories(id) ON DELETE CASCADE,
 source_type varchar(20) NOT NULL CHECK (source_type IN ('manual','comment')), source_id uuid,
 content text NOT NULL, condition text NOT NULL DEFAULT '', enabled boolean NOT NULL DEFAULT true,
 source_snapshot jsonb NOT NULL DEFAULT '{}', created_by varchar(200) NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL, updated_at timestamptz NOT NULL,
 UNIQUE (memory_id, source_id)
);
CREATE INDEX feedback_memory_entries_tenant ON feedback_memory_entries(tenant_id, memory_id);
CREATE TABLE feedback_memory_versions (
 id uuid PRIMARY KEY, tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
 memory_id uuid NOT NULL REFERENCES feedback_memories(id) ON DELETE CASCADE, version_no integer NOT NULL,
 base_version_id uuid, input_revision integer NOT NULL, status varchar(20) NOT NULL DEFAULT 'draft' CHECK (status IN ('draft','published')),
 sources jsonb NOT NULL DEFAULT '[]', items jsonb NOT NULL DEFAULT '[]', warnings jsonb NOT NULL DEFAULT '[]',
 system_prompt text NOT NULL, user_prompt text NOT NULL, prompt_text text NOT NULL,
 created_by varchar(200) NOT NULL DEFAULT '', created_at timestamptz NOT NULL, published_at timestamptz,
 UNIQUE(memory_id, version_no)
);
CREATE INDEX feedback_memory_versions_tenant ON feedback_memory_versions(tenant_id, memory_id, version_no DESC);

COMMENT ON TABLE feedback_memories IS '流程反馈记忆库，按租户、业务模块与流程类型隔离';
COMMENT ON COLUMN feedback_memories.id IS '记忆库主键';
COMMENT ON COLUMN feedback_memories.tenant_id IS '所属租户ID';
COMMENT ON COLUMN feedback_memories.module IS '业务模块：audit 审核、archive 归档复盘';
COMMENT ON COLUMN feedback_memories.process_type IS 'OA 流程类型标识，不是流程实例ID';
COMMENT ON COLUMN feedback_memories.revision IS '素材修订号，修改素材或停用发布版本时递增，用于拒绝过期草稿';
COMMENT ON COLUMN feedback_memories.last_version IS '已分配的最大生成版本号';
COMMENT ON COLUMN feedback_memories.published_version_id IS '当前生效版本ID，空表示尚未发布或已停用';
COMMENT ON COLUMN feedback_memories.created_at IS '记忆库创建时间';
COMMENT ON COLUMN feedback_memories.updated_at IS '素材或发布状态最后变更时间';
COMMENT ON TABLE feedback_memory_entries IS '反馈记忆逐条素材，含评论原文快照与手动补充条件';
COMMENT ON COLUMN feedback_memory_entries.id IS '素材主键';
COMMENT ON COLUMN feedback_memory_entries.tenant_id IS '所属租户ID';
COMMENT ON COLUMN feedback_memory_entries.memory_id IS '所属反馈记忆库ID';
COMMENT ON COLUMN feedback_memory_entries.source_type IS '素材来源：manual 手动补充、comment 用户评论';
COMMENT ON COLUMN feedback_memory_entries.source_id IS '原审核评论ID，手动素材为空；不设外键以保留已删除评论的历史快照';
COMMENT ON COLUMN feedback_memory_entries.content IS '反馈原文或手动补充内容，评论导入后原文只读';
COMMENT ON COLUMN feedback_memory_entries.condition IS '管理员补充的适用条件，可为空';
COMMENT ON COLUMN feedback_memory_entries.enabled IS '是否纳入下一次 AI 生成';
COMMENT ON COLUMN feedback_memory_entries.source_snapshot IS '导入时的原评论快照，含流程实例、审核批次、作者、满意度与时间';
COMMENT ON COLUMN feedback_memory_entries.created_by IS '导入或创建素材的管理员账号';
COMMENT ON COLUMN feedback_memory_entries.created_at IS '素材创建时间';
COMMENT ON COLUMN feedback_memory_entries.updated_at IS '素材最后修改时间';
COMMENT ON TABLE feedback_memory_versions IS '反馈记忆不可变生成快照，保存来源、提示词与发布时点';
COMMENT ON COLUMN feedback_memory_versions.id IS '生成版本主键，同时作为反馈记忆 AI 调用业务关联ID';
COMMENT ON COLUMN feedback_memory_versions.tenant_id IS '所属租户ID';
COMMENT ON COLUMN feedback_memory_versions.memory_id IS '所属反馈记忆库ID';
COMMENT ON COLUMN feedback_memory_versions.version_no IS '记忆库内递增版本号';
COMMENT ON COLUMN feedback_memory_versions.base_version_id IS '生成时整合的上一已发布版本ID，首次生成或停用后为空';
COMMENT ON COLUMN feedback_memory_versions.input_revision IS '生成输入对应的素材修订号，发布时校验是否过期';
COMMENT ON COLUMN feedback_memory_versions.status IS '版本状态：draft 待确认草稿、published 曾发布；当前生效以记忆库发布指针为准';
COMMENT ON COLUMN feedback_memory_versions.sources IS '生成时输入来源快照，保留来源引用标识及原文';
COMMENT ON COLUMN feedback_memory_versions.items IS 'AI 整合后的逐条记忆，含适用条件、内容和来源引用';
COMMENT ON COLUMN feedback_memory_versions.warnings IS 'AI 返回的冲突、证据不足或待确认事项';
COMMENT ON COLUMN feedback_memory_versions.system_prompt IS '本次生成实际使用的系统提示词，只读快照';
COMMENT ON COLUMN feedback_memory_versions.user_prompt IS '本次生成实际使用的用户提示词，只读快照';
COMMENT ON COLUMN feedback_memory_versions.prompt_text IS '根据结构化记忆条目生成的运行时提示词正文';
COMMENT ON COLUMN feedback_memory_versions.created_by IS '发起生成的管理员账号';
COMMENT ON COLUMN feedback_memory_versions.created_at IS 'AI 生成完成并保存的时点';
COMMENT ON COLUMN feedback_memory_versions.published_at IS '首次发布时点，草稿为空';
COMMENT ON INDEX feedback_memory_entries_tenant IS '按租户和记忆库加载反馈素材';
COMMENT ON INDEX feedback_memory_versions_tenant IS '按租户和记忆库倒序分页读取生成版本';
