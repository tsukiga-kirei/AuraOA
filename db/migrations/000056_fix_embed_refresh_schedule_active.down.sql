-- 仅恢复旧的数据库默认值，不恢复已经纠正的错误启用状态。
ALTER TABLE embed_refresh_schedules
    ALTER COLUMN is_active SET DEFAULT TRUE;
