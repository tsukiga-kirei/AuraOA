-- 000022_user_notifications.up.sql
-- 用户通知：按「角色分配」隔离，与 JWT active_role.id 一致；切换角色/租户后列表与未读数随之变化。
--
-- 写入示例（将 :assignment_id 换为 user_role_assignments.id，:user_id 换为 users.id）：
-- INSERT INTO user_notifications (user_id, role_assignment_id, category, title, body, link_path)
-- VALUES (:user_id, :assignment_id, 'tenant_admin', '标题', '正文', '/overview');

CREATE TABLE user_notifications (
    id                   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              UUID         NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role_assignment_id   UUID         NOT NULL REFERENCES user_role_assignments (id) ON DELETE CASCADE,
    category             VARCHAR(64)  NOT NULL DEFAULT 'general',
    title                TEXT         NOT NULL,
    body                 TEXT,
    link_path            TEXT,
    read_at              TIMESTAMPTZ,
    metadata             JSONB        NOT NULL DEFAULT '{}',
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_user_notifications_assignment_created
    ON user_notifications (role_assignment_id, created_at DESC);

CREATE INDEX idx_user_notifications_assignment_unread
    ON user_notifications (role_assignment_id)
    WHERE read_at IS NULL;

COMMENT ON TABLE user_notifications IS '用户通知；scope 为 role_assignment_id，与登录 JWT active_role.id 对齐';
COMMENT ON COLUMN user_notifications.category IS '业务分类：system_admin | tenant_admin | business | audit | archive | cron | general 等';
