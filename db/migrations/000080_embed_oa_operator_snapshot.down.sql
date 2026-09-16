-- 回滚 OA 嵌入操作人快照字段。

ALTER TABLE process_summary_logs
    DROP COLUMN IF EXISTS oa_operator_dept,
    DROP COLUMN IF EXISTS oa_operator_name,
    DROP COLUMN IF EXISTS oa_operator_id;

ALTER TABLE audit_logs
    DROP COLUMN IF EXISTS oa_operator_dept,
    DROP COLUMN IF EXISTS oa_operator_name,
    DROP COLUMN IF EXISTS oa_operator_id;
