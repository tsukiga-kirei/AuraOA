UPDATE org_roles SET page_permissions = page_permissions - '/admin/tenant/experience';
DROP INDEX IF EXISTS idx_chat_messages_feedback;
DROP TABLE IF EXISTS audit_comments;
