-- 000002_tenants_users.down.sql
-- Drop tables in reverse dependency order

DROP TABLE IF EXISTS login_history;
DROP TABLE IF EXISTS user_role_assignments;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;
