-- 000035_weaver_api_url.down.sql

ALTER TABLE oa_database_connections
    DROP COLUMN IF EXISTS weaver_api_url;
