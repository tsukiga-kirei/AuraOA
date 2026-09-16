-- 记录 OA 嵌入触发时的实际当前人员快照，避免将内部租户管理员归属账号展示为操作人。

ALTER TABLE audit_logs
    ADD COLUMN IF NOT EXISTS oa_operator_id VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS oa_operator_name VARCHAR(200) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS oa_operator_dept VARCHAR(200) NOT NULL DEFAULT '';

ALTER TABLE process_summary_logs
    ADD COLUMN IF NOT EXISTS oa_operator_id VARCHAR(128) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS oa_operator_name VARCHAR(200) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS oa_operator_dept VARCHAR(200) NOT NULL DEFAULT '';

COMMENT ON COLUMN audit_logs.oa_operator_id IS 'OA 嵌入触发时传入的当前人员标识；非嵌入或未知时为空';
COMMENT ON COLUMN audit_logs.oa_operator_name IS '任务创建时解析并固化的 OA 当前人员姓名快照';
COMMENT ON COLUMN audit_logs.oa_operator_dept IS '任务创建时解析并固化的 OA 当前人员部门快照';
COMMENT ON COLUMN process_summary_logs.oa_operator_id IS 'OA 嵌入触发时传入的当前人员标识；非嵌入或未知时为空';
COMMENT ON COLUMN process_summary_logs.oa_operator_name IS '任务创建时解析并固化的 OA 当前人员姓名快照';
COMMENT ON COLUMN process_summary_logs.oa_operator_dept IS '任务创建时解析并固化的 OA 当前人员部门快照';
