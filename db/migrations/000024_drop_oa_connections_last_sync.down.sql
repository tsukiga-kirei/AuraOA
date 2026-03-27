ALTER TABLE oa_database_connections
    ADD COLUMN IF NOT EXISTS last_sync TIMESTAMPTZ;

COMMENT ON COLUMN oa_database_connections.last_sync IS '最后一次成功同步时间';
