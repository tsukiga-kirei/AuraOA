-- 000075_agent_access_control.down.sql
ALTER TABLE agent_definitions DROP COLUMN IF EXISTS access_control;
