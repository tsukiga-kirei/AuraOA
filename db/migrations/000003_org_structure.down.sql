-- 000003_org_structure.down.sql
-- 回滚：按依赖逆序删除表

DROP TABLE IF EXISTS org_member_roles;
DROP TABLE IF EXISTS org_members;
DROP TABLE IF EXISTS org_roles;
DROP TABLE IF EXISTS departments;
