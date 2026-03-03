-- 000001_init_extensions.down.sql
-- Drop extensions (use with caution in shared databases)

DROP EXTENSION IF EXISTS "pgcrypto";
DROP EXTENSION IF EXISTS "uuid-ossp";
