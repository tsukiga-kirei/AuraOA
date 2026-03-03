-- 000001_init_extensions.up.sql
-- Enable required PostgreSQL extensions for UUID generation and cryptographic functions

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
