-- 保留所有版本与日志；回滚时仅保留原标准流程绑定，个人绑定不能覆盖它。
DELETE FROM process_execution_config_bindings WHERE scope <> '';
ALTER TABLE process_execution_config_bindings DROP CONSTRAINT uq_process_execution_config_bindings_process;
ALTER TABLE process_execution_config_bindings DROP COLUMN scope;
ALTER TABLE process_execution_config_bindings ADD CONSTRAINT uq_process_execution_config_bindings_process UNIQUE (tenant_id, module, process_id);
-- 显式角色授权保留，避免回滚删除管理员后来调整的权限。
