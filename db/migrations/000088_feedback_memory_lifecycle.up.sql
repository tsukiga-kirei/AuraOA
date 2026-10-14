-- 停用只关闭运行时注入，保留发布版与增量基线。
ALTER TABLE feedback_memories ADD COLUMN enabled boolean NOT NULL DEFAULT true;
UPDATE feedback_memories SET enabled = (published_version_id IS NOT NULL);
-- 兼容旧停用逻辑已清空指针的数据，恢复最后发布版但不自动启用。
UPDATE feedback_memories m SET published_version_id = (
 SELECT v.id FROM feedback_memory_versions v WHERE v.memory_id=m.id AND v.tenant_id=m.tenant_id
 AND v.status='published' ORDER BY v.version_no DESC LIMIT 1
) WHERE m.published_version_id IS NULL;
ALTER TABLE feedback_memory_entries ADD COLUMN implanted_version_id uuid;
ALTER TABLE feedback_memory_entries ADD COLUMN deleted_at timestamptz;
ALTER TABLE feedback_memory_entries DROP CONSTRAINT feedback_memory_entries_memory_id_source_id_key;
CREATE UNIQUE INDEX feedback_memory_entries_live_source ON feedback_memory_entries(memory_id, source_id) WHERE deleted_at IS NULL;
ALTER TABLE feedback_memory_versions ADD COLUMN generated_items jsonb NOT NULL DEFAULT '[]';
ALTER TABLE feedback_memory_versions ADD COLUMN edit_revision integer NOT NULL DEFAULT 0;
ALTER TABLE feedback_memory_versions ADD COLUMN edited_by varchar(200) NOT NULL DEFAULT '';
ALTER TABLE feedback_memory_versions ADD COLUMN edited_at timestamptz;
UPDATE feedback_memory_versions SET generated_items=items;
-- 仅标记真正被生成条目引用的旧素材，空结果和未引用素材继续待整合。
UPDATE feedback_memory_entries e SET implanted_version_id=(
 SELECT v.id FROM feedback_memory_versions v WHERE v.memory_id=e.memory_id AND v.tenant_id=e.tenant_id
 AND EXISTS (SELECT 1 FROM jsonb_array_elements(v.items) i WHERE i->'source_refs' ? e.id::text)
 ORDER BY v.version_no DESC LIMIT 1
);
CREATE TABLE feedback_memory_comment_exclusions (
 tenant_id uuid NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
 module varchar(20) NOT NULL CHECK(module IN ('audit','archive')),
 process_type varchar(200) NOT NULL, comment_id uuid NOT NULL,
 PRIMARY KEY(tenant_id,module,process_type,comment_id)
);
COMMENT ON COLUMN feedback_memories.enabled IS '是否向后续执行注入已发布记忆，停用仍保留发布版与生成基线';
COMMENT ON COLUMN feedback_memories.published_version_id IS '最后确认发布的版本ID，是否生效由enabled决定';
COMMENT ON COLUMN feedback_memories.revision IS '素材修订号，素材内容、启停或删除时递增';
COMMENT ON COLUMN feedback_memory_entries.implanted_version_id IS '首次被可靠记忆条目引用的版本ID，已植入素材不重复输入AI';
COMMENT ON COLUMN feedback_memory_entries.deleted_at IS '素材删除时间，保留删除标记以便下次生成撤销相关记忆';
COMMENT ON INDEX feedback_memory_entries_live_source IS '同一记忆库内的未删除评论素材唯一，删除后允许重新导入';
COMMENT ON COLUMN feedback_memory_versions.generated_items IS 'AI原始生成条目快照，人工编辑不覆盖';
COMMENT ON COLUMN feedback_memory_versions.edit_revision IS '草稿编辑修订号，防止并发覆盖';
COMMENT ON COLUMN feedback_memory_versions.edited_by IS '最后编辑草稿的管理员账号';
COMMENT ON COLUMN feedback_memory_versions.edited_at IS '最后人工编辑时间';
COMMENT ON COLUMN feedback_memory_versions.base_version_id IS '本次整合的上一版本ID，可以为草稿或已发布版，首次生成为空';
COMMENT ON TABLE feedback_memory_versions IS '反馈记忆版本，保留AI原始输出与来源，草稿可编辑，发布后不可变';
COMMENT ON TABLE feedback_memory_comment_exclusions IS '各业务模块与流程的评论建议弃用记录，不修改用户原始评论';
COMMENT ON COLUMN feedback_memory_comment_exclusions.tenant_id IS '所属租户ID';
COMMENT ON COLUMN feedback_memory_comment_exclusions.module IS '业务模块audit或archive';
COMMENT ON COLUMN feedback_memory_comment_exclusions.process_type IS 'OA流程类型标识';
COMMENT ON COLUMN feedback_memory_comment_exclusions.comment_id IS '弃用的原始审核评论ID';
COMMENT ON INDEX feedback_memory_comment_exclusions_pkey IS '按租户、模块、流程与评论保存独立弃用状态';
