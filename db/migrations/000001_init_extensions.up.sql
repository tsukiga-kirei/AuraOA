-- 000001_init_extensions.up.sql
-- 启用 PostgreSQL 扩展：UUID 生成与加密函数

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- 数据库注释（中文）
COMMENT ON EXTENSION "uuid-ossp" IS 'UUID 生成函数扩展';
COMMENT ON EXTENSION "pgcrypto" IS '加密函数扩展';
