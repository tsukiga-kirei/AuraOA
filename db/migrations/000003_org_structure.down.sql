-- 000003_org_structure.down.sql
-- Drop tables in reverse dependency order

DROP TABLE IF EXISTS org_member_roles;
DROP TABLE IF EXISTS org_members;
DROP TABLE IF EXISTS org_roles;
DROP TABLE IF EXISTS departments;
