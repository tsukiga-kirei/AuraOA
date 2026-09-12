-- 嵌入审核的评价与留言按审核批次保存，避免重新审核后意见错配。
CREATE TABLE audit_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    audit_log_id UUID NOT NULL REFERENCES audit_logs(id) ON DELETE CASCADE,
    oa_user_id VARCHAR(128) NOT NULL,
    username VARCHAR(255) NOT NULL,
    content TEXT NOT NULL CHECK (char_length(btrim(content)) BETWEEN 1 AND 2000),
    feedback VARCHAR(16) CHECK (feedback IN ('like', 'dislike')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
COMMENT ON TABLE audit_comments IS '嵌入审核体验评论：按审核批次逐条保存用户评论及可选满意度，同一用户可提交多条';
COMMENT ON COLUMN audit_comments.id IS '评论唯一标识';
COMMENT ON COLUMN audit_comments.tenant_id IS '所属租户标识，用于租户数据隔离';
COMMENT ON COLUMN audit_comments.audit_log_id IS '关联审核日志标识，对应本次审核批次';
COMMENT ON COLUMN audit_comments.oa_user_id IS '评论作者的 OA 用户标识，用于校验本人编辑和删除权限';
COMMENT ON COLUMN audit_comments.username IS '提交评论时的作者姓名快照';
COMMENT ON COLUMN audit_comments.content IS '评论正文，去除首尾空格后长度为 1 至 2000 个字符';
COMMENT ON COLUMN audit_comments.feedback IS '本条评论附带的 AI 审核满意度：like 满意，dislike 不满意，NULL 未评价';
COMMENT ON COLUMN audit_comments.created_at IS '评论创建时间';
COMMENT ON COLUMN audit_comments.updated_at IS '评论最后修改时间';

CREATE INDEX idx_audit_comments_thread ON audit_comments (tenant_id, audit_log_id, created_at, id);
CREATE INDEX idx_chat_messages_feedback ON chat_messages (tenant_id, feedback_at DESC) WHERE feedback IS NOT NULL;
COMMENT ON INDEX idx_audit_comments_thread IS '按租户和审核批次查询评论，并按创建时间及标识稳定分页';
COMMENT ON INDEX idx_chat_messages_feedback IS '按租户和评价时间查询已评价的智能体消息';
UPDATE org_roles SET page_permissions = page_permissions || '["/admin/tenant/experience"]'::jsonb
WHERE page_permissions @> '["/admin/tenant/org"]'::jsonb
AND NOT page_permissions @> '["/admin/tenant/experience"]'::jsonb;
