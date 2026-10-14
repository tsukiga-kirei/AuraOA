-- 租户管理员统一控制用户端审核评论可见范围，默认保留全部可见。
ALTER TABLE tenants ADD COLUMN audit_comments_own_only BOOLEAN NOT NULL DEFAULT FALSE;
COMMENT ON COLUMN tenants.audit_comments_own_only IS '用户端仅可查看本人审核评论；管理员反馈查询仍可查看租户全部评论';
