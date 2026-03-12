-- 000002_tenants_users.down.sql
-- 回滚：按依赖逆序删除表

DROP TABLE IF EXISTS login_history;
DROP TABLE IF EXISTS user_role_assignments;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;
