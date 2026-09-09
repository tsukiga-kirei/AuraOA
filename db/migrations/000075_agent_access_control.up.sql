-- 000075_agent_access_control.up.sql
-- 为智能体定义表增加人员访问控制字段，默认全员可用

ALTER TABLE agent_definitions
    ADD COLUMN IF NOT EXISTS access_control JSONB NOT NULL DEFAULT '{"allow_all": true, "allowed_roles": [], "allowed_members": [], "allowed_departments": []}'::jsonb;

COMMENT ON COLUMN agent_definitions.access_control IS '访问控制配置（allow_all: 所有人可用；allowed_roles: 允许角色; allowed_members: 允许成员; allowed_departments: 允许部门）';

-- 确保已有的智能体数据具备默认的全员访问权限
UPDATE agent_definitions
SET access_control = '{"allow_all": true, "allowed_roles": [], "allowed_members": [], "allowed_departments": []}'::jsonb
WHERE access_control IS NULL OR access_control = '{}'::jsonb;
