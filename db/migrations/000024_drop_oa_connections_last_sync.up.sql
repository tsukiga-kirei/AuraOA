-- 移除未使用的 last_sync 列（应用侧从未持久化写入）
ALTER TABLE oa_database_connections DROP COLUMN IF EXISTS last_sync;
