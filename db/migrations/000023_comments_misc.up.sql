-- 000023_comments_misc.up.sql
-- 补全历史迁移中遗漏的表/列 COMMENT（user_dashboard_prefs.pref_scope、user_notifications 全列、tenant_id 可空语义）

COMMENT ON COLUMN user_dashboard_prefs.tenant_id IS '所属租户ID；NULL 表示系统管理员平台级仪表盘布局';
COMMENT ON COLUMN user_dashboard_prefs.pref_scope IS '布局作用域：platform=系统管理员平台；business=租户内业务用户；tenant_admin=租户内管理员；shared 为迁移过程旧值（应已清空）';

COMMENT ON COLUMN user_notifications.id IS '主键UUID';
COMMENT ON COLUMN user_notifications.user_id IS '接收通知的用户ID';
COMMENT ON COLUMN user_notifications.role_assignment_id IS '角色分配ID（user_role_assignments.id），与 JWT active_role.id 对齐';
COMMENT ON COLUMN user_notifications.title IS '通知标题';
COMMENT ON COLUMN user_notifications.body IS '通知正文（可空）';
COMMENT ON COLUMN user_notifications.link_path IS '点击跳转的前端路由路径（可空）';
COMMENT ON COLUMN user_notifications.read_at IS '已读时间；NULL 表示未读';
COMMENT ON COLUMN user_notifications.metadata IS '扩展元数据（JSON）';
COMMENT ON COLUMN user_notifications.created_at IS '创建时间';
